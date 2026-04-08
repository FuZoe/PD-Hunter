package exporter

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/FuZoe/PD-Hunter/pkg/scraper"
)

// WriteJSON writes issues to a JSON file with indentation.
func WriteJSON(issues []scraper.Issue, outputFile string) error {
	data, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}

	if err := os.WriteFile(outputFile, data, 0644); err != nil {
		return fmt.Errorf("writing file %s: %w", outputFile, err)
	}

	fmt.Printf("Results saved to: %s\n", outputFile)
	return nil
}
