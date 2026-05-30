package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

type Project struct {
	Hash       string
	Path       string
	SessionDir string
}

type JSONLFile struct {
	Path        string
	SessionID   string
	ProjectHash string
	ProjectPath string
}

func Discover() ([]Project, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	projectsDir := filepath.Join(home, ".claude", "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	var projects []Project
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		hash := e.Name()
		sessionDir := filepath.Join(projectsDir, hash)

		path := resolveProjectPath(sessionDir, hash)

		projects = append(projects, Project{
			Hash:       hash,
			Path:       path,
			SessionDir: sessionDir,
		})
	}
	return projects, nil
}

func FindJSONLFiles(proj Project) []JSONLFile {
	var files []JSONLFile

	entries, err := os.ReadDir(proj.SessionDir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() && strings.HasSuffix(name, ".jsonl") {
			sessionID := strings.TrimSuffix(name, ".jsonl")
			files = append(files, JSONLFile{
				Path:        filepath.Join(proj.SessionDir, name),
				SessionID:   sessionID,
				ProjectHash: proj.Hash,
				ProjectPath: proj.Path,
			})
		}

		// Check for subagent files inside session directories
		if e.IsDir() {
			subagentDir := filepath.Join(proj.SessionDir, name, "subagents")
			subs, err := os.ReadDir(subagentDir)
			if err != nil {
				continue
			}
			for _, sub := range subs {
				if strings.HasSuffix(sub.Name(), ".jsonl") {
					files = append(files, JSONLFile{
						Path:        filepath.Join(subagentDir, sub.Name()),
						SessionID:   name,
						ProjectHash: proj.Hash,
						ProjectPath: proj.Path,
					})
				}
			}
		}
	}
	return files
}

// resolveProjectPath tries to recover the original directory path from the hash.
// Falls back to reading the cwd field from the first JSONL entry in the project.
func resolveProjectPath(sessionDir, hash string) string {
	// Try the simple reverse: leading dash = /, interior dashes stay
	candidate := reverseHash(hash)
	if info, err := os.Stat(candidate); err == nil && info.IsDir() {
		return candidate
	}

	// Fall back to reading cwd from first JSONL entry
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return hash
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".jsonl") && !e.IsDir() {
			if cwd := extractCWD(filepath.Join(sessionDir, e.Name())); cwd != "" {
				return cwd
			}
		}
	}

	return hash
}

// reverseHash does a best-effort reversal of the project hash.
// The hash replaces /  ~ _ with -. We assume leading - is always /.
func reverseHash(hash string) string {
	if hash == "" {
		return ""
	}
	// Leading dash becomes /
	if hash[0] == '-' {
		hash = "/" + hash[1:]
	}
	// Replace remaining dashes with / to form a path candidate
	return strings.ReplaceAll(hash, "-", "/")
}

func extractCWD(jsonlPath string) string {
	f, err := os.Open(jsonlPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	buf := make([]byte, 64*1024)
	n, err := f.Read(buf)
	if n == 0 {
		return ""
	}

	// Find lines with cwd field
	for _, line := range strings.Split(string(buf[:n]), "\n") {
		if line == "" {
			continue
		}
		var entry struct {
			CWD  string `json:"cwd"`
			Type string `json:"type"`
		}
		if err := json.Unmarshal([]byte(line), &entry); err == nil && entry.CWD != "" {
			return entry.CWD
		}
	}
	return ""
}
