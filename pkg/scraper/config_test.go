package scraper

import (
	"os"
	"path/filepath"
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
