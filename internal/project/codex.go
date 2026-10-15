package project

import (
	"bufio"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var codexSessionIDPattern = regexp.MustCompile(`([[:xdigit:]]{8}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{4}-[[:xdigit:]]{12})\.jsonl$`)

// DiscoverCodex finds active and archived Codex rollout files. CODEX_HOME
// defaults to ~/.codex, matching the Codex CLI.
func DiscoverCodex() ([]JSONLFile, error) {
	codexHome, err := codexHomeDir()
	if err != nil {
		return nil, err
	}

	var files []JSONLFile
	for _, dir := range []string{
		filepath.Join(codexHome, "sessions"),
		filepath.Join(codexHome, "archived_sessions"),
	} {
		discovered, err := discoverCodexDir(dir)
		if err != nil {
			return nil, err
		}
		files = append(files, discovered...)
	}

	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

func codexHomeDir() (string, error) {
	if dir := os.Getenv("CODEX_HOME"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".codex"), nil
}

func discoverCodexDir(root string) ([]JSONLFile, error) {
	var files []JSONLFile
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			return nil
		}

		meta := readCodexSessionMeta(path)
		sessionID := meta.SessionID
		if sessionID == "" {
			sessionID = codexSessionID(path)
		}
		projectHash := "codex:unknown"
		if meta.CWD != "" {
			projectHash = "codex:" + meta.CWD
		}

		files = append(files, JSONLFile{
			Path:        path,
			SessionID:   sessionID,
			ProjectHash: projectHash,
			ProjectPath: meta.CWD,
			Provider:    ProviderCodex,
		})
		return nil
	})
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return files, err
}

type codexSessionMeta struct {
	CWD       string
	SessionID string
}

func readCodexSessionMeta(path string) codexSessionMeta {
	f, err := os.Open(path)
	if err != nil {
		return codexSessionMeta{}
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for lines := 0; lines < 20 && scanner.Scan(); lines++ {
		var record struct {
			Type    string `json:"type"`
			Payload struct {
				CWD       string `json:"cwd"`
				ID        string `json:"id"`
				SessionID string `json:"session_id"`
			} `json:"payload"`
		}
		if json.Unmarshal(scanner.Bytes(), &record) != nil || record.Type != "session_meta" {
			continue
		}
		sessionID := record.Payload.SessionID
		if sessionID == "" {
			sessionID = record.Payload.ID
		}
		return codexSessionMeta{CWD: record.Payload.CWD, SessionID: sessionID}
	}
	return codexSessionMeta{}
}

func codexSessionID(path string) string {
	match := codexSessionIDPattern.FindStringSubmatch(filepath.Base(path))
	if len(match) == 2 {
		return match[1]
	}
	return strings.TrimSuffix(filepath.Base(path), ".jsonl")
}
