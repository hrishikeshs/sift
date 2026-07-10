package parse

import (
	"encoding/json"

	"github.com/hrishikeshs/sift/internal/db"
)

type codexRecord struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

type codexResponseItem struct {
	Type    string              `json:"type"`
	ID      string              `json:"id"`
	Role    string              `json:"role"`
	Content []codexContentBlock `json:"content"`
	Summary []codexSummaryBlock `json:"summary"`
	// encrypted_content is deliberately absent. Sift never decodes or indexes it.
}

type codexContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type codexSummaryBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ExtractCodexEntries extracts canonical response items from a Codex rollout.
// Event messages are ignored because they duplicate response_item content.
func ExtractCodexEntries(line []byte, sourceFile, projectPath, projectHash, sessionID string, offset int64) []db.Entry {
	var record codexRecord
	if json.Unmarshal(line, &record) != nil || record.Type != "response_item" {
		return nil
	}

	var item codexResponseItem
	if json.Unmarshal(record.Payload, &item) != nil {
		return nil
	}

	entry := func(content, sourceType string) db.Entry {
		return db.Entry{
			Content:     content,
			SourceType:  sourceType,
			Timestamp:   record.Timestamp,
			ProjectPath: projectPath,
			ProjectHash: projectHash,
			SessionID:   sessionID,
			MessageID:   item.ID,
			SourceFile:  sourceFile,
			ByteOffset:  offset,
		}
	}

	var entries []db.Entry
	switch item.Type {
	case "message":
		var sourceType, blockType string
		switch item.Role {
		case "user":
			sourceType, blockType = "user", "input_text"
		case "assistant":
			sourceType, blockType = "text", "output_text"
		default:
			return nil
		}
		for _, block := range item.Content {
			if block.Type == blockType && block.Text != "" {
				entries = append(entries, entry(block.Text, sourceType))
			}
		}
	case "reasoning":
		for _, summary := range item.Summary {
			if summary.Type == "summary_text" && summary.Text != "" {
				entries = append(entries, entry(summary.Text, "reasoning_summary"))
			}
		}
	}
	return entries
}

func CodexArchiveWorthy(line []byte) bool {
	var record codexRecord
	if json.Unmarshal(line, &record) != nil || record.Type != "response_item" {
		return false
	}
	var item codexResponseItem
	if json.Unmarshal(record.Payload, &item) != nil || item.Type != "message" {
		return false
	}
	return item.Role == "user" || item.Role == "assistant"
}

func CodexTimestamp(line []byte) string {
	var record codexRecord
	if json.Unmarshal(line, &record) != nil {
		return ""
	}
	return record.Timestamp
}
