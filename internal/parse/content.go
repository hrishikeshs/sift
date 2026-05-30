package parse

import (
	"encoding/json"

	"github.com/hrishikeshs/sift/internal/db"
)

func ExtractEntries(line []byte, sourceFile, projectPath, projectHash string, offset int64) []db.Entry {
	var raw RawEntry
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil
	}

	switch raw.Type {
	case "user":
		return extractUser(raw, sourceFile, projectPath, projectHash, offset)
	case "assistant":
		return extractAssistant(raw, sourceFile, projectPath, projectHash, offset)
	default:
		return nil
	}
}

func extractUser(raw RawEntry, sourceFile, projectPath, projectHash string, offset int64) []db.Entry {
	var msg Message
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	// User content is either a string or a tool_result array.
	// Only index string content.
	var content string
	if err := json.Unmarshal(msg.Content, &content); err != nil {
		return nil
	}

	if content == "" {
		return nil
	}

	return []db.Entry{{
		Content:     content,
		SourceType:  "user",
		Timestamp:   raw.Timestamp,
		ProjectPath: projectPath,
		ProjectHash: projectHash,
		SessionID:   raw.SessionID,
		MessageID:   msg.ID,
		Model:       msg.Model,
		SourceFile:  sourceFile,
		IsSidechain: raw.IsSidechain,
		ByteOffset:  offset,
	}}
}

func extractAssistant(raw RawEntry, sourceFile, projectPath, projectHash string, offset int64) []db.Entry {
	var msg Message
	if err := json.Unmarshal(raw.Message, &msg); err != nil {
		return nil
	}

	var blocks []ContentBlock
	if err := json.Unmarshal(msg.Content, &blocks); err != nil {
		return nil
	}

	var entries []db.Entry
	for _, block := range blocks {
		var content, sourceType string

		switch block.Type {
		case "thinking":
			content = block.Thinking
			sourceType = "thinking"
		case "text":
			content = block.Text
			sourceType = "text"
		case "tool_use":
			content = block.Name
			sourceType = "tool_use"
		default:
			continue
		}

		if content == "" {
			continue
		}

		entries = append(entries, db.Entry{
			Content:     content,
			SourceType:  sourceType,
			Timestamp:   raw.Timestamp,
			ProjectPath: projectPath,
			ProjectHash: projectHash,
			SessionID:   raw.SessionID,
			MessageID:   msg.ID,
			Model:       msg.Model,
			SourceFile:  sourceFile,
			IsSidechain: raw.IsSidechain,
			ByteOffset:  offset,
		})
	}
	return entries
}
