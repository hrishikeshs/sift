package index

import (
	"bufio"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/hrishikeshs/sift/internal/db"
	"github.com/hrishikeshs/sift/internal/parse"
	"github.com/hrishikeshs/sift/internal/project"
)

const batchSize = 500

func (idx *Indexer) IndexFile(f project.JSONLFile) (int, error) {
	info, err := os.Stat(f.Path)
	if err != nil {
		return 0, err
	}

	state, err := idx.DB.GetIndexState(f.Path)
	if err != nil {
		return 0, err
	}

	fileSize := info.Size()
	fileMtime := info.ModTime().UTC().Format(time.RFC3339)

	// Check for file rotation
	if state != nil {
		if fileSize < state.ByteOffset {
			idx.DB.DeleteEntriesForFile(f.Path)
			idx.DB.DeleteIndexState(f.Path)
			idx.DB.DeleteChunksForFile(f.Path)
			state = nil
		} else if state.FileHash != "" && !hashMatches(f.Path, state.FileHash) {
			idx.DB.DeleteEntriesForFile(f.Path)
			idx.DB.DeleteIndexState(f.Path)
			idx.DB.DeleteChunksForFile(f.Path)
			state = nil
		}
	}

	// Skip if unchanged
	if state != nil && fileMtime == state.FileMtime && fileSize == state.FileSize {
		return 0, nil
	}

	startOffset := int64(0)
	if state != nil {
		startOffset = state.ByteOffset
	}

	file, err := os.Open(f.Path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	if startOffset > 0 {
		if _, err := file.Seek(startOffset, io.SeekStart); err != nil {
			return 0, err
		}
	}

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 256*1024), 50*1024*1024)

	var batch []db.Entry
	var chunkLines = make(map[string][]string) // date -> raw lines for archival
	totalNew := 0
	currentOffset := startOffset

	for scanner.Scan() {
		line := scanner.Bytes()
		lineStr := string(line)
		lineLen := int64(len(line)) + 1 // +1 for newline

		entries := parse.ExtractEntries(line, f.Path, f.ProjectPath, f.ProjectHash, currentOffset)
		batch = append(batch, entries...)
		totalNew += len(entries)

		// Archive only user + assistant lines (skip tool_result, progress, etc.)
		if archiveWorthy(line) {
			ts := extractTimestamp(line)
			if ts != "" {
				date := ts[:10] // YYYY-MM-DD
				chunkLines[date] = append(chunkLines[date], lineStr)
			}
		}

		if len(batch) >= batchSize {
			if err := idx.DB.InsertBatch(batch); err != nil {
				return totalNew, fmt.Errorf("inserting batch: %w", err)
			}
			batch = batch[:0]
		}

		currentOffset += lineLen
	}

	if err := scanner.Err(); err != nil {
		// Oversized lines cause ErrTooLong — advance past them to avoid stalling
		fmt.Fprintf(os.Stderr, "Warning: skipped oversized line in %s at offset %d: %v\n", f.Path, currentOffset, err)
		currentOffset = fileSize
	}

	// Flush remaining batch
	if len(batch) > 0 {
		if err := idx.DB.InsertBatch(batch); err != nil {
			return totalNew, fmt.Errorf("inserting final batch: %w", err)
		}
	}

	// Archive daily chunks
	for date, lines := range chunkLines {
		firstTS, lastTS := findTimestampRange(lines)
		if err := idx.DB.UpsertChunk(f.Path, f.SessionID, f.ProjectHash, date, lines, firstTS, lastTS); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to archive chunk %s/%s: %v\n", f.Path, date, err)
		}
	}

	// Compute file hash for rotation detection
	hash := computeFileHash(f.Path)

	prevCount := 0
	if state != nil {
		prevCount = state.EntryCount
	}

	if err := idx.DB.UpsertIndexState(db.IndexState{
		SourceFile: f.Path,
		ByteOffset: currentOffset,
		FileSize:   fileSize,
		FileMtime:  fileMtime,
		EntryCount: prevCount + totalNew,
		FileHash:   hash,
	}); err != nil {
		return totalNew, fmt.Errorf("updating index state: %w", err)
	}

	return totalNew, nil
}

func archiveWorthy(line []byte) bool {
	var entry struct {
		Type    string `json:"type"`
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	}
	if json.Unmarshal(line, &entry) != nil {
		return false
	}
	switch entry.Type {
	case "assistant":
		return true
	case "user":
		// Only archive string content (not tool_result arrays)
		var s string
		return json.Unmarshal(entry.Message.Content, &s) == nil
	default:
		return false
	}
}

func extractTimestamp(line []byte) string {
	var entry struct {
		Timestamp string `json:"timestamp"`
	}
	json.Unmarshal(line, &entry)
	return entry.Timestamp
}

func findTimestampRange(lines []string) (string, string) {
	var timestamps []string
	for _, line := range lines {
		ts := extractTimestamp([]byte(line))
		if ts != "" {
			timestamps = append(timestamps, ts)
		}
	}
	if len(timestamps) == 0 {
		return "", ""
	}
	sort.Strings(timestamps)
	return timestamps[0], timestamps[len(timestamps)-1]
}

func computeFileHash(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	if n == 0 {
		return ""
	}

	h := sha256.Sum256(buf[:n])
	return fmt.Sprintf("%x", h)
}

func hashMatches(path, expectedHash string) bool {
	actual := computeFileHash(path)
	return actual == expectedHash
}

