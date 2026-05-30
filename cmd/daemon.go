package cmd

import (
	"fmt"
	"log"
	"os"
	"os/signal"

	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/hrishikeshs/sift/internal/index"
)

var daemonInterval string

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run continuous indexing in the foreground",
	RunE: func(cmd *cobra.Command, args []string) error {
		interval, err := time.ParseDuration(daemonInterval)
		if err != nil {
			return fmt.Errorf("invalid interval: %w", err)
		}

		d, err := openDB()
		if err != nil {
			return err
		}
		defer d.Close()

		idx := &index.Indexer{DB: d, Verbose: true}

		fmt.Fprintf(os.Stderr, "sift daemon started (interval: %s)\n", interval)

		// Run immediately on start
		if err := idx.IndexAll(); err != nil {
			log.Printf("Index error: %v", err)
		}

		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

		for {
			select {
			case <-ticker.C:
				if err := idx.IndexAll(); err != nil {
					log.Printf("Index error: %v", err)
				}
			case <-sig:
				fmt.Fprintf(os.Stderr, "\nsift daemon stopped\n")
				return nil
			}
		}
	},
}

func init() {
	daemonCmd.Flags().StringVar(&daemonInterval, "interval", "60m", "indexing interval (e.g. 30m, 1h)")
	rootCmd.AddCommand(daemonCmd)
}
