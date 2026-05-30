package index

import (
	"fmt"
	"os"

	"github.com/hrishikeshs/sift/internal/db"
	"github.com/hrishikeshs/sift/internal/project"
)

type Indexer struct {
	DB      *db.DB
	Verbose bool
}

func (idx *Indexer) IndexAll() error {
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

func (idx *Indexer) IndexProject(projectPath string) error {
	projects, err := project.Discover()
	if err != nil {
		return err
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
			return nil
		}
	}

	return fmt.Errorf("project not found: %s", projectPath)
}
