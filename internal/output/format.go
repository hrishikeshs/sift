package output

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hrishikeshs/sift/internal/db"
)

const (
	bold  = "\033[1m"
	yellow = "\033[33m"
	cyan  = "\033[36m"
	dim   = "\033[2m"
	reset = "\033[0m"
)

func PrintResults(results []db.SearchResult, asJSON bool) {
	deduped := dedup(results)

	if asJSON {
		enc := json.NewEncoder(os.Stdout)
		for _, r := range deduped {
			enc.Encode(map[string]interface{}{
				"content":     r.Content,
				"snippet":     r.Snippet,
				"source_type": r.SourceType,
				"timestamp":   r.Timestamp,
				"project":     projectName(r.ProjectPath, r.ProjectHash),
				"session_id":  r.SessionID,
				"model":       r.Model,
				"source_file": r.SourceFile,
			})
		}
		return
	}

	for i, r := range deduped {
		if i > 0 {
			fmt.Println()
		}
		ts := formatTimestamp(r.Timestamp)
		proj := projectName(r.ProjectPath, r.ProjectHash)

		fmt.Printf("%s[%s]%s %s%s%s | %s %s#%d%s\n", dim, ts, reset, cyan, r.SourceType, reset, proj, dim, r.ID, reset)
		fmt.Printf("  %s\n", highlight(r.Snippet))
		fmt.Printf("  %s%s  →  sift show %d%s\n", dim, shortSource(r.SourceFile, r.SessionID), r.ID, reset)
	}
}

func PrintStats(stats *db.Stats) {
	fmt.Printf("Total entries: %s%d%s\n", bold, stats.TotalEntries, reset)
	fmt.Printf("Archived chunks: %d\n\n", stats.ChunkCount)

	if len(stats.ProjectCounts) > 0 {
		fmt.Println("By project:")
		for proj, count := range stats.ProjectCounts {
			fmt.Printf("  %-40s %d\n", projectName("", proj), count)
		}
		fmt.Println()
	}

	if len(stats.TypeCounts) > 0 {
		fmt.Println("By type:")
		for typ, count := range stats.TypeCounts {
			fmt.Printf("  %-12s %d\n", typ, count)
		}
	}
}

func PrintContext(entries []db.SearchResult, targetID int64) {
	for _, r := range entries {
		ts := formatTimestamp(r.Timestamp)
		isTarget := r.ID == targetID

		if isTarget {
			fmt.Printf("%s%s[%s] %s | %s #%d ◀%s\n", bold, yellow, ts, r.SourceType,
				projectName(r.ProjectPath, r.ProjectHash), r.ID, reset)
		} else {
			fmt.Printf("%s[%s]%s %s%s%s | %s\n", dim, ts, reset, cyan, r.SourceType, reset,
				projectName(r.ProjectPath, r.ProjectHash))
		}

		content := truncateRunes(r.Content, 500)
		fmt.Printf("  %s\n\n", content)
	}
}

func highlight(snippet string) string {
	s := strings.ReplaceAll(snippet, ">>>", bold+yellow)
	s = strings.ReplaceAll(s, "<<<", reset)
	return s
}

func shortSource(path, sessionID string) string {
	path = abbreviatePath(path)

	// Shorten session UUIDs: abc12345-...-6789abcd.jsonl → abc123…cd.jsonl
	if sessionID != "" && len(sessionID) > 12 {
		short := sessionID[:6] + "…" + sessionID[len(sessionID)-2:]
		path = strings.Replace(path, sessionID, short, 1)
	}

	// Shorten subagent paths
	if strings.Contains(path, "/subagents/") {
		path = strings.Replace(path, "/subagents/", "/sub/", 1)
	}

	return path
}

func dedup(results []db.SearchResult) []db.SearchResult {
	seen := make(map[string]bool)
	var out []db.SearchResult
	for _, r := range results {
		key := r.Snippet + "|" + r.Timestamp
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, r)
	}
	return out
}

func formatTimestamp(ts string) string {
	if len(ts) >= 19 {
		return strings.Replace(ts[:19], "T", " ", 1)
	}
	return ts
}

func projectName(path, hash string) string {
	if path != "" && path != hash {
		return filepath.Base(path)
	}
	parts := strings.Split(hash, "-")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return hash
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max]) + "..."
	}
	return s
}

func abbreviatePath(path string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}
