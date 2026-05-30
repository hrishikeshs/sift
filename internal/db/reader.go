package db

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
)

type SearchParams struct {
	Query      string
	Since      string
	SourceType string
	Project    string
	SessionID  string
	Limit      int
	JSON       bool
}

type SearchResult struct {
	ID          int64
	Content     string
	Snippet     string
	SourceType  string
	Timestamp   string
	ProjectPath string
	ProjectHash string
	SessionID   string
	MessageID   string
	Model       string
	SourceFile  string
}

type Stats struct {
	TotalEntries  int
	ProjectCounts map[string]int
	TypeCounts    map[string]int
	ChunkCount    int
}

func (db *DB) Search(params SearchParams) ([]SearchResult, int, error) {
	if params.Limit <= 0 {
		params.Limit = 20
	}

	var conditions []string
	var args []interface{}

	conditions = append(conditions, "entries_fts MATCH ?")
	args = append(args, autoPhrase(params.Query))

	if params.Since != "" {
		conditions = append(conditions, "e.timestamp >= ?")
		args = append(args, params.Since)
	}
	if params.SourceType != "" {
		conditions = append(conditions, "e.source_type = ?")
		args = append(args, params.SourceType)
	}
	if params.Project != "" {
		conditions = append(conditions, "(e.project_hash = ? OR e.project_path LIKE ?)")
		args = append(args, params.Project, "%"+params.Project+"%")
	}
	if params.SessionID != "" {
		conditions = append(conditions, "e.session_id = ?")
		args = append(args, params.SessionID)
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count total matches
	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM entries_fts
		JOIN entries e ON entries_fts.rowid = e.id
		WHERE %s`, whereClause)
	var total int
	if err := db.conn.QueryRow(countQuery, args...).Scan(&total); err != nil {
		total = 0
	}

	args = append(args, params.Limit)

	query := fmt.Sprintf(`
		SELECT e.id, e.content,
			snippet(entries_fts, 0, '>>>', '<<<', '...', 40) as snippet,
			e.source_type, e.timestamp, e.project_path, e.project_hash,
			e.session_id, e.message_id, e.model, e.source_file
		FROM entries_fts
		JOIN entries e ON entries_fts.rowid = e.id
		WHERE %s
		ORDER BY rank
		LIMIT ?`,
		whereClause)

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("executing search: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var r SearchResult
		var projectPath, projectHash, sessionID, messageID, model sql.NullString
		err := rows.Scan(
			&r.ID, &r.Content, &r.Snippet,
			&r.SourceType, &r.Timestamp,
			&projectPath, &projectHash,
			&sessionID, &messageID, &model, &r.SourceFile,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("scanning row: %w", err)
		}
		r.ProjectPath = projectPath.String
		r.ProjectHash = projectHash.String
		r.SessionID = sessionID.String
		r.MessageID = messageID.String
		r.Model = model.String
		results = append(results, r)
	}
	return results, total, rows.Err()
}

func (db *DB) GetEntry(id int64) (*SearchResult, error) {
	row := db.conn.QueryRow(`
		SELECT id, content, source_type, timestamp, project_path, project_hash,
			session_id, message_id, model, source_file
		FROM entries WHERE id = ?`, id)

	var r SearchResult
	var projectPath, projectHash, sessionID, messageID, model sql.NullString
	err := row.Scan(
		&r.ID, &r.Content, &r.SourceType, &r.Timestamp,
		&projectPath, &projectHash,
		&sessionID, &messageID, &model, &r.SourceFile,
	)
	if err != nil {
		return nil, err
	}
	r.ProjectPath = projectPath.String
	r.ProjectHash = projectHash.String
	r.SessionID = sessionID.String
	r.MessageID = messageID.String
	r.Model = model.String
	r.Snippet = r.Content
	return &r, nil
}

func (db *DB) GetContext(entry *SearchResult, before, after int) ([]SearchResult, error) {
	rows, err := db.conn.Query(`
		SELECT id, content, source_type, timestamp, project_path, project_hash,
			session_id, message_id, model, source_file
		FROM entries
		WHERE source_file = ? AND id < ?
		ORDER BY id DESC LIMIT ?`,
		entry.SourceFile, entry.ID, before)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var preceding []SearchResult
	for rows.Next() {
		var r SearchResult
		var pp, ph, sid, mid, m sql.NullString
		if err := rows.Scan(&r.ID, &r.Content, &r.SourceType, &r.Timestamp, &pp, &ph, &sid, &mid, &m, &r.SourceFile); err != nil {
			continue
		}
		r.ProjectPath = pp.String
		r.ProjectHash = ph.String
		r.SessionID = sid.String
		r.MessageID = mid.String
		r.Model = m.String
		r.Snippet = r.Content
		preceding = append(preceding, r)
	}

	// Reverse preceding to chronological order
	for i, j := 0, len(preceding)-1; i < j; i, j = i+1, j-1 {
		preceding[i], preceding[j] = preceding[j], preceding[i]
	}

	rows2, err := db.conn.Query(`
		SELECT id, content, source_type, timestamp, project_path, project_hash,
			session_id, message_id, model, source_file
		FROM entries
		WHERE source_file = ? AND id > ?
		ORDER BY id ASC LIMIT ?`,
		entry.SourceFile, entry.ID, after)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	var following []SearchResult
	for rows2.Next() {
		var r SearchResult
		var pp, ph, sid, mid, m sql.NullString
		if err := rows2.Scan(&r.ID, &r.Content, &r.SourceType, &r.Timestamp, &pp, &ph, &sid, &mid, &m, &r.SourceFile); err != nil {
			continue
		}
		r.ProjectPath = pp.String
		r.ProjectHash = ph.String
		r.SessionID = sid.String
		r.MessageID = mid.String
		r.Model = m.String
		r.Snippet = r.Content
		following = append(following, r)
	}

	// Combine: preceding + target + following
	result := make([]SearchResult, 0, len(preceding)+1+len(following))
	result = append(result, preceding...)
	result = append(result, *entry)
	result = append(result, following...)
	return result, nil
}

func (db *DB) GetStats() (*Stats, error) {
	stats := &Stats{
		ProjectCounts: make(map[string]int),
		TypeCounts:    make(map[string]int),
	}

	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM entries`).Scan(&stats.TotalEntries); err != nil {
		return nil, err
	}

	rows, err := db.conn.Query(`SELECT COALESCE(project_hash, 'unknown'), COUNT(*) FROM entries GROUP BY project_hash`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			continue
		}
		stats.ProjectCounts[name] = count
	}

	rows2, err := db.conn.Query(`SELECT source_type, COUNT(*) FROM entries GROUP BY source_type`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()
	for rows2.Next() {
		var name string
		var count int
		if err := rows2.Scan(&name, &count); err != nil {
			continue
		}
		stats.TypeCounts[name] = count
	}

	if err := db.conn.QueryRow(`SELECT COUNT(*) FROM chunks`).Scan(&stats.ChunkCount); err != nil {
		return nil, err
	}

	return stats, nil
}

func (db *DB) OrphanedFiles() ([]string, int, error) {
	rows, err := db.conn.Query(`SELECT DISTINCT source_file FROM entries`)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orphaned []string
	total := 0
	for rows.Next() {
		var f string
		if err := rows.Scan(&f); err != nil {
			continue
		}
		total++
		if !fileExists(f) {
			orphaned = append(orphaned, f)
		}
	}

	return orphaned, total, rows.Err()
}

func (db *DB) DeleteOrphaned(files []string) (int, error) {
	tx, err := db.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	total := 0
	for _, f := range files {
		res, err := tx.Exec(`DELETE FROM entries WHERE source_file = ?`, f)
		if err != nil {
			return total, err
		}
		n, _ := res.RowsAffected()
		total += int(n)
		tx.Exec(`DELETE FROM index_state WHERE source_file = ?`, f)
	}

	if err := tx.Commit(); err != nil {
		return total, err
	}
	db.conn.Exec(`VACUUM`)
	return total, nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// autoPhrase wraps multi-word queries in quotes for phrase matching.
// Queries that already use FTS5 operators (AND, OR, NOT, quotes) are left alone.
func autoPhrase(query string) string {
	if strings.ContainsAny(query, `"`) {
		return query
	}
	for _, op := range []string{" AND ", " OR ", " NOT "} {
		if strings.Contains(query, op) {
			return query
		}
	}
	if strings.Contains(query, " ") {
		return `"` + query + `"`
	}
	return query
}
