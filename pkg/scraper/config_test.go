package scraper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig_Valid(t *testing.T) {
	config, err := LoadConfig(filepath.Join("testdata", "valid_mapping.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(config.Organizations) != 2 {
		t.Fatalf("expected 2 organizations, got %d", len(config.Organizations))
	}

	org := config.Organizations[0]
	if org.Name != "testorg" {
		t.Errorf("expected name 'testorg', got '%s'", org.Name)
	}
	if len(org.Labels) != 2 {
		t.Errorf("expected 2 labels, got %d", len(org.Labels))
	}
	if org.Labels[0] != "bounty" {
		t.Errorf("expected first label 'bounty', got '%s'", org.Labels[0])
	}
	if org.Note != "Test organization" {
		t.Errorf("expected note 'Test organization', got '%s'", org.Note)
	}
}

func TestLoadConfig_EmptyOrgs(t *testing.T) {
	_, err := LoadConfig(filepath.Join("testdata", "empty_orgs.json"))
	if err == nil {
		t.Fatal("expected error for empty organizations, got nil")
	}
}

func TestLoadConfig_InvalidJSON(t *testing.T) {
	_, err := LoadConfig(filepath.Join("testdata", "invalid.json"))
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("nonexistent_file.json")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestLoadConfig_SingleOrg(t *testing.T) {
	// Create a temp config with a single org
	tmpDir := t.TempDir()
	content := `{"organizations": [{"name": "solo", "labels": ["bug-bounty"], "note": "Solo org"}]}`
	tmpFile := filepath.Join(tmpDir, "single.json")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	config, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config.Organizations) != 1 {
		t.Fatalf("expected 1 organization, got %d", len(config.Organizations))
	}
	if config.Organizations[0].Name != "solo" {
		t.Errorf("expected name 'solo', got '%s'", config.Organizations[0].Name)
	}
}

// TestLoadConfig_RootMappingJSON validates the repository root mapping.json
// with the same strict loader the scanner uses, so invalid JSON, misspelled
// fields, empty or duplicate labels, and duplicate organizations fail CI.
func TestLoadConfig_RootMappingJSON(t *testing.T) {
	config, err := LoadConfig("../../mapping.json")
	if err != nil {
		t.Fatalf("root mapping.json is invalid: %v", err)
	}
	if len(config.Organizations) == 0 {
		t.Fatal("root mapping.json has zero organizations")
	}
}

func TestLoadConfig_RejectsInvalidConfigs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr string
	}{
		{"misspelled field", `{"organizations": [{"name": "a", "lables": ["bounty"]}]}`, `unknown field "lables"`},
		{"missing labels", `{"organizations": [{"name": "a", "note": "x"}]}`, "labels and text_searches are both empty"},
		{"empty labels", `{"organizations": [{"name": "a", "labels": []}]}`, "labels and text_searches are both empty"},
		{"blank label", `{"organizations": [{"name": "a", "labels": ["bounty", "  "]}]}`, "entry 1 is blank"},
		{"padded label", `{"organizations": [{"name": "a", "labels": [" bounty"]}]}`, "leading or trailing whitespace"},
		{"quoted label", `{"organizations": [{"name": "a", "labels": ["say \"hi\""]}]}`, "double quotes"},
		{"duplicate label", `{"organizations": [{"name": "a", "labels": ["Bounty", "bounty"]}]}`, "duplicate label"},
		{"blank name", `{"organizations": [{"name": " ", "labels": ["bounty"]}]}`, "name is empty"},
		{"invalid name", `{"organizations": [{"name": "a b", "labels": ["bounty"]}]}`, "not a valid GitHub organization login"},
		{"duplicate org", `{"organizations": [{"name": "Org", "labels": ["bounty"]}, {"name": "org", "labels": ["x"]}]}`, "already listed"},
		{"trailing content", `{"organizations": [{"name": "a", "labels": ["bounty"]}]} {"x": 1}`, "unexpected content"},
		{"bad text search field", `{"organizations": [{"name": "a", "labels": ["b"], "text_searches": [{"in": "comments", "phrase": "bounty"}]}]}`, `in must be "title" or "body"`},
		{"empty text search phrase", `{"organizations": [{"name": "a", "labels": ["b"], "text_searches": [{"in": "title", "phrase": ""}]}]}`, "phrase is empty"},
		{"blank exclude label", `{"organizations": [{"name": "a", "labels": ["b"]}], "exclude_labels": [""]}`, "exclude_labels: entry 0 is blank"},
		{"inactive without note", `{"organizations": [{"name": "a", "labels": ["b"]}], "inactive_organizations": [{"name": "c", "labels": ["b"]}]}`, "note must explain"},
		{"inactive duplicates active", `{"organizations": [{"name": "a", "labels": ["b"]}], "inactive_organizations": [{"name": "A", "labels": ["b"], "note": "x"}]}`, "already listed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "mapping.json")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("failed to write temp file: %v", err)
			}
			_, err := LoadConfig(path)
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("expected error containing %q, got: %v", tt.wantErr, err)
			}
		})
	}
}

func TestLoadConfig_AcceptsNewFields(t *testing.T) {
	content := `{
		"organizations": [{"name": "a", "labels": ["bounty"], "note": "n",
			"text_searches": [{"in": "title", "phrase": "bounty"}]}],
		"exclude_labels": ["💰 Rewarded"],
		"inactive_organizations": [{"name": "old", "labels": [], "note": "paused"}]
	}`
	path := filepath.Join(t.TempDir(), "mapping.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(config.Organizations[0].TextSearches) != 1 || len(config.ExcludeLabels) != 1 || len(config.InactiveOrganizations) != 1 {
		t.Errorf("new fields were not parsed: %+v", config)
	}
}

func TestLoadConfig_TextOnlyOrganization(t *testing.T) {
	content := `{"organizations": [{"name": "a", "labels": [], "note": "n",
		"text_searches": [{"in": "title", "phrase": "bounty"}]}]}`
	path := filepath.Join(t.TempDir(), "mapping.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}
	if _, err := LoadConfig(path); err != nil {
		t.Fatalf("an organization with only text_searches should be valid: %v", err)
	}
}
