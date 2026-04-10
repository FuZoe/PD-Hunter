package exporter

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/FuZoe/PD-Hunter/pkg/scraper"
)

func TestWriteJSON(t *testing.T) {
	issues := []scraper.Issue{
		{
			Number:       1,
			Title:        "Test Issue",
			URL:          "https://github.com/test/repo/issues/1",
			State:        "open",
			Labels:       []string{"bounty"},
			CommentCount: 5,
			OpenPRCount:  2,
			Repository:   "test/repo",
			CreatedAt:    "2024-01-01T00:00:00Z",
			UpdatedAt:    "2024-06-01T00:00:00Z",
			Author:       "testuser",
			Body:         "Test body content",
		},
	}

	t.Run("writes valid JSON file", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "test_output.json")

		err := WriteJSON(issues, outFile)
		if err != nil {
			t.Fatalf("WriteJSON returned error: %v", err)
		}

		// Verify file exists
		if _, err := os.Stat(outFile); os.IsNotExist(err) {
			t.Fatal("Output file was not created")
		}

		// Verify valid JSON
		data, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		var result []scraper.Issue
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		if len(result) != 1 {
			t.Fatalf("Expected 1 issue, got %d", len(result))
		}

		if result[0].Number != 1 {
			t.Errorf("Expected issue number 1, got %d", result[0].Number)
		}

		if result[0].Title != "Test Issue" {
			t.Errorf("Expected title 'Test Issue', got '%s'", result[0].Title)
		}
	})

	t.Run("writes empty array for no issues", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "empty.json")

		err := WriteJSON([]scraper.Issue{}, outFile)
		if err != nil {
			t.Fatalf("WriteJSON returned error: %v", err)
		}

		data, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		var result []scraper.Issue
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatalf("Output is not valid JSON: %v", err)
		}

		if len(result) != 0 {
			t.Fatalf("Expected 0 issues, got %d", len(result))
		}
	})

	t.Run("returns error for invalid path", func(t *testing.T) {
		err := WriteJSON(issues, "/nonexistent/dir/file.json")
		if err == nil {
			t.Fatal("Expected error for invalid path, got nil")
		}
	})

	t.Run("preserves all fields", func(t *testing.T) {
		tmpDir := t.TempDir()
		outFile := filepath.Join(tmpDir, "fields.json")

		err := WriteJSON(issues, outFile)
		if err != nil {
			t.Fatalf("WriteJSON returned error: %v", err)
		}

		data, _ := os.ReadFile(outFile)
		var result []scraper.Issue
		json.Unmarshal(data, &result)

		issue := result[0]
		if issue.URL != "https://github.com/test/repo/issues/1" {
			t.Errorf("URL mismatch: %s", issue.URL)
		}
		if issue.State != "open" {
			t.Errorf("State mismatch: %s", issue.State)
		}
		if len(issue.Labels) != 1 || issue.Labels[0] != "bounty" {
			t.Errorf("Labels mismatch: %v", issue.Labels)
		}
		if issue.CommentCount != 5 {
			t.Errorf("CommentCount mismatch: %d", issue.CommentCount)
		}
		if issue.OpenPRCount != 2 {
			t.Errorf("OpenPRCount mismatch: %d", issue.OpenPRCount)
		}
		if issue.Repository != "test/repo" {
			t.Errorf("Repository mismatch: %s", issue.Repository)
		}
		if issue.Author != "testuser" {
			t.Errorf("Author mismatch: %s", issue.Author)
		}
	})
}
