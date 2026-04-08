package main

import (
	"fmt"
	"os"

	"github.com/FuZoe/PD-Hunter/pkg/exporter"
	"github.com/FuZoe/PD-Hunter/pkg/scraper"
	"github.com/spf13/cobra"
)

var (
	configFile string
	outputFile string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "hunter",
		Short: "PD-Hunter — Open Source Bounty Intelligence Platform",
		Long:  "Find high-value open source bounties matched to your skills, powered by AI.",
	}

	scanCmd := &cobra.Command{
		Use:   "scan",
		Short: "Scan GitHub organizations for bounty issues",
		RunE:  runScan,
	}

	scanCmd.Flags().StringVarP(&configFile, "config", "c", "mapping.json", "Path to organization config file")
	scanCmd.Flags().StringVarP(&outputFile, "output", "o", "bounty_issues.json", "Output JSON file path")

	rootCmd.AddCommand(scanCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runScan(cmd *cobra.Command, args []string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Note: GITHUB_TOKEN not set. Using unauthenticated requests (rate limited to 60/hour)")
	}

	config, err := scraper.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("Loaded %d organizations from config\n", len(config.Organizations))

	client := scraper.NewClient(token)
	issues, err := client.ScanAll(config)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	if err := exporter.WriteJSON(issues, outputFile); err != nil {
		return fmt.Errorf("exporting: %w", err)
	}

	return nil
}
