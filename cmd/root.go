package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/db"
)

var dbPath string

var rootCmd = &cobra.Command{
	Use:   "sift",
	Short: "Full-text search for Claude Code sessions",
	Long:  `sift indexes and searches Claude Code JSONL session transcripts.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", db.DefaultPath(), "path to sift database")
}

func openDB() (*db.DB, error) {
	d, err := db.Open(dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	return d, nil
}
