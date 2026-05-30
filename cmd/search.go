package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/db"
	"github.com/hrishikeshs/sift/internal/output"
	"github.com/hrishikeshs/sift/internal/parse"
)

var (
	searchSince   string
	searchType    string
	searchProject string
	searchSession string
	searchLimit   int
	searchJSON    bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search indexed sessions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		params := db.SearchParams{
			Query:      args[0],
			SourceType: searchType,
			Project:    searchProject,
			SessionID:  searchSession,
			Limit:      searchLimit,
			JSON:       searchJSON,
		}

		if searchSince != "" {
			t, err := parse.ParseSince(searchSince)
			if err != nil {
				return err
			}
			params.Since = t.Format("2006-01-02T15:04:05Z")
		}

		results, total, err := d.Search(params)
		if err != nil {
			return fmt.Errorf("search failed: %w", err)
		}

		if len(results) == 0 {
			fmt.Println("No results found.")
			return nil
		}

		output.PrintResults(results, searchJSON)

		if !searchJSON && total > len(results) {
			fmt.Printf("\n%sShowing %d of %d results. Use --limit %d to see more.%s\n",
				"\033[2m", len(results), total, min(total, params.Limit*2), "\033[0m")
		}
		return nil
	},
}

func init() {
	searchCmd.Flags().StringVar(&searchSince, "since", "", "filter by time (2w, 3d, yesterday, 2026-04-20)")
	searchCmd.Flags().StringVar(&searchType, "type", "", "filter by source type (thinking, text, user, tool_use)")
	searchCmd.Flags().StringVar(&searchProject, "project", "", "filter by project path or hash")
	searchCmd.Flags().StringVar(&searchSession, "session", "", "filter by session ID")
	searchCmd.Flags().IntVar(&searchLimit, "limit", 20, "max results to return")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "output as JSON lines")
	rootCmd.AddCommand(searchCmd)
}
