package index

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/hrishikeshs/sift/internal/db"
	"github.com/hrishikeshs/sift/internal/project"
)

type Indexer struct {
	DB      *db.DB
	Verbose bool
}

func (idx *Indexer) IndexAll() error {
	return idx.IndexAllProvider("claude")
}

func (idx *Indexer) IndexAllProvider(provider string) error {
	switch provider {
	case "claude":
		return idx.indexAllClaude()
	case "codex":
		return idx.indexAllCodex()
	case "all":
		if err := idx.indexAllClaude(); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return idx.indexAllCodex()
	default:
		return fmt.Errorf("unknown provider %q (want claude, codex, or all)", provider)
	}
}

func (idx *Indexer) indexAllClaude() error {
	projects, err := project.Discover()
	if err != nil {
		return fmt.Errorf("discovering projects: %w", err)
	}

	if idx.Verbose {
		fmt.Fprintf(os.Stderr, "Found %d projects\n", len(projects))
	}

	totalNew := 0
	for _, proj := range projects {
		files := project.FindJSONLFiles(proj)
		for _, f := range files {
			n, err := idx.IndexFile(f)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error indexing %s: %v\n", f.Path, err)
				continue
			}
			totalNew += n
			if idx.Verbose && n > 0 {
				fmt.Fprintf(os.Stderr, "  +%d entries from %s\n", n, f.Path)
			}
		}
	}

	if idx.Verbose {
		fmt.Fprintf(os.Stderr, "Indexed %d new entries\n", totalNew)
	}
	return nil
}

func (idx *Indexer) indexAllCodex() error {
	files, err := project.DiscoverCodex()
	if err != nil {
		return fmt.Errorf("discovering Codex sessions: %w", err)
	}
	if idx.Verbose {
		fmt.Fprintf(os.Stderr, "Found %d Codex sessions\n", len(files))
	}

	totalNew := 0
	for _, f := range files {
		n, err := idx.IndexFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error indexing %s: %v\n", f.Path, err)
			continue
		}
		totalNew += n
		if idx.Verbose && n > 0 {
			fmt.Fprintf(os.Stderr, "  +%d entries from %s\n", n, f.Path)
		}
	}
	if idx.Verbose {
		fmt.Fprintf(os.Stderr, "Indexed %d new entries\n", totalNew)
	}
	return nil
}

func (idx *Indexer) IndexProject(projectPath string) error {
	return idx.IndexProjectProvider(projectPath, "claude")
}

func (idx *Indexer) IndexProjectProvider(projectPath, provider string) error {
	switch provider {
	case "claude":
		found, err := idx.indexClaudeProject(projectPath)
		return projectResult(projectPath, found, err)
	case "codex":
		found, err := idx.indexCodexProject(projectPath)
		return projectResult(projectPath, found, err)
	case "all":
		claudeFound, err := idx.indexClaudeProject(projectPath)
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		codexFound, err := idx.indexCodexProject(projectPath)
		return projectResult(projectPath, claudeFound || codexFound, err)
	default:
		return fmt.Errorf("unknown provider %q (want claude, codex, or all)", provider)
	}
}

func (idx *Indexer) indexClaudeProject(projectPath string) (bool, error) {
	projects, err := project.Discover()
	if err != nil {
		return false, err
	}

	for _, proj := range projects {
		if proj.Path == projectPath || proj.Hash == projectPath {
			files := project.FindJSONLFiles(proj)
			for _, f := range files {
				n, err := idx.IndexFile(f)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error indexing %s: %v\n", f.Path, err)
					continue
				}
				if idx.Verbose && n > 0 {
					fmt.Fprintf(os.Stderr, "  +%d entries from %s\n", n, f.Path)
				}
			}
			return true, nil
		}
	}

	return false, nil
}

func (idx *Indexer) indexCodexProject(projectPath string) (bool, error) {
	files, err := project.DiscoverCodex()
	if err != nil {
		return false, err
	}
	found := false
	for _, f := range files {
		if f.ProjectPath != projectPath && f.ProjectHash != projectPath {
			continue
		}
		found = true
		n, err := idx.IndexFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error indexing %s: %v\n", f.Path, err)
			continue
		}
		if idx.Verbose && n > 0 {
			fmt.Fprintf(os.Stderr, "  +%d entries from %s\n", n, f.Path)
		}
	}
	return found, nil
}

func projectResult(projectPath string, found bool, err error) error {
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("project not found: %s", projectPath)
	}
	return nil
}
