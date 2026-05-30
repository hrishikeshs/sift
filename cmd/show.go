package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/output"
)

var showContext int

var showCmd = &cobra.Command{
	Use:   "show [id]",
	Short: "Show conversation context around a search result",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid entry ID: %w", err)
		}

		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		entry, err := d.GetEntry(id)
		if err != nil {
			return fmt.Errorf("entry not found: %w", err)
		}

		context, err := d.GetContext(entry, showContext, showContext)
		if err != nil {
			return fmt.Errorf("loading context: %w", err)
		}

		output.PrintContext(context, id)
		return nil
	},
}

func init() {
	showCmd.Flags().IntVar(&showContext, "context", 5, "number of entries to show before and after")
	rootCmd.AddCommand(showCmd)
}
