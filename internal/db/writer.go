package db

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Entry struct {
	Content     string
	SourceType  string
	Timestamp   string
	ProjectPath string
	ProjectHash string
	SessionID   string
	MessageID   string
	Model       string
	SourceFile  string
	IsSidechain bool
	ByteOffset  int64
}

type IndexState struct {
	SourceFile string
	ByteOffset int64
	FileSize   int64
	FileMtime  string
	EntryCount int
	FileHash   string
}

func (db *DB) InsertBatch(entries []Entry) error {
	if len(entries) == 0 {
		return nil
	}

	tx, err := db.conn.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`INSERT INTO entries
		(content, source_type, timestamp, project_path, project_hash,
		 session_id, message_id, model, source_file, is_sidechain, byte_offset)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing statement: %w", err)
	}
	defer stmt.Close()

	for _, e := range entries {
		sidechain := 0
		if e.IsSidechain {
			sidechain = 1
		}
		_, err := stmt.Exec(
			e.Content, e.SourceType, e.Timestamp,
			e.ProjectPath, e.ProjectHash,
			e.SessionID, e.MessageID, e.Model,
			e.SourceFile, sidechain, e.ByteOffset,
		)
		if err != nil {
			return fmt.Errorf("inserting entry: %w", err)
		}
	}

	return tx.Commit()
}

func (db *DB) UpsertIndexState(state IndexState) error {
	_, err := db.conn.Exec(`INSERT INTO index_state
		(source_file, byte_offset, file_size, file_mtime, entry_count, last_indexed, file_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_file) DO UPDATE SET
			byte_offset = excluded.byte_offset,
			file_size = excluded.file_size,
			file_mtime = excluded.file_mtime,
			entry_count = excluded.entry_count,
			last_indexed = excluded.last_indexed,
			file_hash = excluded.file_hash`,
		state.SourceFile, state.ByteOffset, state.FileSize,
		state.FileMtime, state.EntryCount,
		time.Now().UTC().Format(time.RFC3339), state.FileHash,
	)
	return err
}

func (db *DB) GetIndexState(sourceFile string) (*IndexState, error) {
	row := db.conn.QueryRow(
		`SELECT source_file, byte_offset, file_size, file_mtime, entry_count, file_hash
		 FROM index_state WHERE source_file = ?`, sourceFile)

	var s IndexState
	err := row.Scan(&s.SourceFile, &s.ByteOffset, &s.FileSize, &s.FileMtime, &s.EntryCount, &s.FileHash)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (db *DB) DeleteEntriesForFile(sourceFile string) error {
	_, err := db.conn.Exec(`DELETE FROM entries WHERE source_file = ?`, sourceFile)
	return err
}

func (db *DB) DeleteChunksForFile(sourceFile string) error {
	_, err := db.conn.Exec(`DELETE FROM chunks WHERE source_file = ?`, sourceFile)
	return err
}

func (db *DB) DeleteIndexState(sourceFile string) error {
	_, err := db.conn.Exec(`DELETE FROM index_state WHERE source_file = ?`, sourceFile)
	return err
}

func (db *DB) UpsertChunk(sourceFile, sessionID, projectHash, chunkDate string, lines []string, firstTS, lastTS string) error {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(strings.Join(lines, "\n")))
	if err != nil {
		return fmt.Errorf("compressing chunk: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("closing gzip writer: %w", err)
	}

	_, err = db.conn.Exec(`INSERT INTO chunks
		(source_file, session_id, project_hash, chunk_date, first_ts, last_ts, entry_count, raw_gz)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_file, chunk_date) DO UPDATE SET
			first_ts = excluded.first_ts,
			last_ts = excluded.last_ts,
			entry_count = excluded.entry_count,
			raw_gz = excluded.raw_gz`,
		sourceFile, sessionID, projectHash, chunkDate,
		firstTS, lastTS, len(lines), buf.Bytes(),
	)
	return err
}
