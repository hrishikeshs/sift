package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/index"
)

var (
	indexAll     bool
	indexProject string
	indexVerbose bool
)

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index JSONL session files",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		idx := &index.Indexer{DB: d, Verbose: indexVerbose}

		if indexProject != "" {
			return idx.IndexProject(indexProject)
		}
		if indexAll {
			return idx.IndexAll()
		}

		fmt.Println("Specify --all to index all projects, or --project PATH for a specific project")
		return nil
	},
}

func init() {
	indexCmd.Flags().BoolVar(&indexAll, "all", false, "index all discovered projects")
	indexCmd.Flags().StringVar(&indexProject, "project", "", "index a specific project by path")
	indexCmd.Flags().BoolVar(&indexVerbose, "verbose", false, "print progress details")
	rootCmd.AddCommand(indexCmd)
}
