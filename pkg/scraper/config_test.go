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

func TestRepositoryMappingIsConsistent(t *testing.T) {
	config, err := LoadConfig(filepath.Join("..", "..", "mapping.json"))
	if err != nil {
		t.Fatalf("failed to load repository mapping: %v", err)
	}

	const expectedOrganizationCount = 34
	if len(config.Organizations) != expectedOrganizationCount {
		t.Fatalf("expected %d organizations, got %d", expectedOrganizationCount, len(config.Organizations))
	}

	seenOrganizations := make(map[string]struct{}, len(config.Organizations))
	for _, organization := range config.Organizations {
		name := strings.TrimSpace(organization.Name)
		if name == "" {
			t.Error("mapping contains an organization with an empty name")
			continue
		}
		if _, exists := seenOrganizations[name]; exists {
			t.Errorf("mapping contains duplicate organization %q", name)
		}
		seenOrganizations[name] = struct{}{}

		seenLabels := make(map[string]struct{}, len(organization.Labels))
		for _, rawLabel := range organization.Labels {
			label := strings.TrimSpace(rawLabel)
			if label == "" {
				t.Errorf("organization %q has an empty label", name)
				continue
			}
			if _, exists := seenLabels[label]; exists {
				t.Errorf("organization %q has duplicate label %q", name, label)
			}
			seenLabels[label] = struct{}{}
		}
	}

	for _, organizationName := range []string{"golemcloud", "zio"} {
		organization := findOrganization(config.Organizations, organizationName)
		if organization == nil {
			t.Fatalf("mapping is missing required organization %q", organizationName)
		}
		if !hasLabel(organization.Labels, "\U0001F48E Bounty") {
			t.Errorf("organization %q is missing the Algora bounty label", organizationName)
		}
	}

	for _, readme := range []string{"README.md", "README_CN.md"} {
		content, err := os.ReadFile(filepath.Join("..", "..", readme))
		if err != nil {
			t.Fatalf("failed to read %s: %v", readme, err)
		}
		text := string(content)
		if strings.Contains(text, "15+") || strings.Contains(text, "35+") {
			t.Errorf("%s contains a stale organization count", readme)
		}
		if !strings.Contains(text, "34+") || !strings.Contains(text, "34%2B") {
			t.Errorf("%s does not consistently advertise 34+ organizations", readme)
		}
	}
}

func findOrganization(organizations []Organization, name string) *Organization {
	for index := range organizations {
		if organizations[index].Name == name {
			return &organizations[index]
		}
	}
	return nil
}

func hasLabel(labels []string, wanted string) bool {
	for _, label := range labels {
		if label == wanted {
			return true
		}
	}
	return false
}
