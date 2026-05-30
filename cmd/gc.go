package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var gcDryRun bool

var gcCmd = &cobra.Command{
	Use:   "gc",
	Short: "Clean up entries from deleted source files",
	RunE: func(cmd *cobra.Command, args []string) error {
		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		orphaned, total, err := d.OrphanedFiles()
		if err != nil {
			return err
		}

		if len(orphaned) == 0 {
			fmt.Printf("No orphaned entries (all %d source files exist)\n", total)
			return nil
		}

		fmt.Printf("Found %d orphaned source files (out of %d total):\n", len(orphaned), total)
		for _, f := range orphaned {
			fmt.Printf("  %s\n", f)
		}

		if gcDryRun {
			fmt.Println("\n(dry run — no changes made)")
			return nil
		}

		deleted, err := d.DeleteOrphaned(orphaned)
		if err != nil {
			return err
		}
		fmt.Printf("\nDeleted %d entries and vacuumed database\n", deleted)
		return nil
	},
}

func init() {
	gcCmd.Flags().BoolVar(&gcDryRun, "dry-run", false, "show what would be deleted without deleting")
	rootCmd.AddCommand(gcCmd)
}
