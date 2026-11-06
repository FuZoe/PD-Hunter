package scraper

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// orgLoginPattern matches GitHub organization/user logins.
var orgLoginPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]*[A-Za-z0-9])?$`)

// LoadConfig reads, strictly parses, and validates a mapping.json file.
// Unknown fields (for example a misspelled "lables") and trailing content
// after the top-level object are rejected.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()

	var config Config
	if err := dec.Decode(&config); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return nil, fmt.Errorf("parsing config JSON: unexpected content after the top-level object")
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// Validate checks the config for mistakes that would make a scan silently
// return wrong or empty data.
func (c *Config) Validate() error {
	var errs []error
	add := func(format string, args ...interface{}) {
		errs = append(errs, fmt.Errorf(format, args...))
	}

	if len(c.Organizations) == 0 {
		add("config contains no organizations")
	}

	seenOrgs := make(map[string]string)
	checkName := func(where, name string) {
		if strings.TrimSpace(name) == "" {
			add("%s: name is empty", where)
			return
		}
		if !orgLoginPattern.MatchString(name) {
			add("%s: %q is not a valid GitHub organization login", where, name)
			return
		}
		key := strings.ToLower(name)
		if prev, ok := seenOrgs[key]; ok {
			add("%s: organization %q is already listed at %s", where, name, prev)
			return
		}
		seenOrgs[key] = where
	}

	for i, org := range c.Organizations {
		where := fmt.Sprintf("organizations[%d]", i)
		checkName(where, org.Name)
		if len(org.Labels) == 0 && len(org.TextSearches) == 0 {
			add("%s (%s): labels and text_searches are both empty", where, org.Name)
		}
		errs = append(errs, validateLabels(fmt.Sprintf("%s (%s) labels", where, org.Name), org.Labels)...)
		for j, ts := range org.TextSearches {
			tsWhere := fmt.Sprintf("%s (%s) text_searches[%d]", where, org.Name, j)
			if ts.In != "title" && ts.In != "body" {
				add("%s: in must be \"title\" or \"body\", got %q", tsWhere, ts.In)
			}
			if strings.TrimSpace(ts.Phrase) == "" {
				add("%s: phrase is empty", tsWhere)
			} else if strings.Contains(ts.Phrase, `"`) {
				add("%s: phrase must not contain double quotes", tsWhere)
			}
		}
	}

	for i, org := range c.InactiveOrganizations {
		where := fmt.Sprintf("inactive_organizations[%d]", i)
		checkName(where, org.Name)
		if strings.TrimSpace(org.Note) == "" {
			add("%s (%s): note must explain why the organization is inactive", where, org.Name)
		}
		errs = append(errs, validateLabels(fmt.Sprintf("%s (%s) labels", where, org.Name), org.Labels)...)
	}

	errs = append(errs, validateLabels("exclude_labels", c.ExcludeLabels)...)

	return errors.Join(errs...)
}

// validateLabels rejects blank, padded, quoted, and duplicate labels.
func validateLabels(where string, labels []string) []error {
	var errs []error
	seen := make(map[string]bool)
	for i, label := range labels {
		switch {
		case strings.TrimSpace(label) == "":
			errs = append(errs, fmt.Errorf("%s: entry %d is blank", where, i))
			continue
		case strings.TrimSpace(label) != label:
			errs = append(errs, fmt.Errorf("%s: entry %d %q has leading or trailing whitespace", where, i, label))
		case strings.Contains(label, `"`):
			errs = append(errs, fmt.Errorf("%s: entry %d %q must not contain double quotes", where, i, label))
		}
		key := normalizeLabel(label)
		if seen[key] {
			errs = append(errs, fmt.Errorf("%s: duplicate label %q", where, label))
		}
		seen[key] = true
	}
	return errs
}
