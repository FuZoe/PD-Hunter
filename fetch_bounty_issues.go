package main

import (
	"fmt"
	"os"

	"github.com/FuZoe/PD-Hunter/pkg/exporter"
	"github.com/FuZoe/PD-Hunter/pkg/scraper"
)

const (
	legacyConfigFile = "mapping.json"
	legacyOutputFile = "bounty_issues.json"
)

// This compatibility entry point intentionally delegates to the same packages
// as cmd/hunter. Keeping a second scraper implementation caused the legacy
// executable to retain obsolete rate-limit and partial-result behavior.
func main() {
	if err := runLegacyScan(legacyConfigFile, legacyOutputFile); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runLegacyScan(configFile, outputFile string) error {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		fmt.Println("Note: GITHUB_TOKEN not set. GitHub API limits will be lower")
	}

	config, err := scraper.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("Loaded %d organizations from config\n", len(config.Organizations))
	issues, err := scraper.NewClient(token).ScanAll(config)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	if err := exporter.WriteJSON(issues, outputFile); err != nil {
		return fmt.Errorf("exporting: %w", err)
	}
	return nil
}
