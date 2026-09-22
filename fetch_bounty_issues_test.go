package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLegacyScanDoesNotWriteOutputWhenConfigIsInvalid(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "mapping.json")
	outputFile := filepath.Join(tempDir, "bounty_issues.json")
	if err := os.WriteFile(configFile, []byte(`{"organizations":[]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := runLegacyScan(configFile, outputFile)
	if err == nil || !strings.Contains(err.Error(), "config contains no organizations") {
		t.Fatalf("expected invalid config error, got %v", err)
	}
	if _, statErr := os.Stat(outputFile); !os.IsNotExist(statErr) {
		t.Fatalf("legacy entry wrote output after a failed scan: %v", statErr)
	}
}
