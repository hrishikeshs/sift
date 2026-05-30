package cmd

import (
	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/output"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show index statistics",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		stats, err := d.GetStats()
		if err != nil {
			return err
		}

		output.PrintStats(stats)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
