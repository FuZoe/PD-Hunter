package scraper

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// fakeClock lets tests observe waits without sleeping.
type fakeClock struct {
	t     time.Time
	slept []time.Duration
}

func (f *fakeClock) now() time.Time { return f.t }

func (f *fakeClock) sleep(d time.Duration) {
	f.slept = append(f.slept, d)
	f.t = f.t.Add(d)
}

func (f *fakeClock) total() time.Duration {
	var sum time.Duration
	for _, d := range f.slept {
		sum += d
	}
	return sum
}

func newClockClient(serverURL string) (*Client, *fakeClock) {
	clock := &fakeClock{t: time.Unix(1_700_000_000, 0)}
	c := newMockClient(serverURL, "test")
	c.sleep = clock.sleep
	c.now = clock.now
	return c, clock
}

func TestDoRequest_RateLimitWaitsForReset(t *testing.T) {
	var clock *fakeClock
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprint(clock.t.Add(40*time.Second).Unix()))
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "API rate limit exceeded for installation."}`))
			return
		}
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
	defer server.Close()

	client, c := newClockClient(server.URL)
	clock = c

	if _, err := client.DoRequest(server.URL + "/search/issues?q=x"); err != nil {
		t.Fatalf("expected success after waiting for reset, got: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls, got %d", calls)
	}
	// Reset is 40s away, plus a 1s buffer. The old code waited a fixed 5s.
	if got := clock.total(); got != 41*time.Second {
		t.Errorf("expected to wait 41s for the reset, waited %v", got)
	}
}

func TestDoRequest_RetryAfterHeader(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("Retry-After", "7")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, clock := newClockClient(server.URL)
	if _, err := client.DoRequest(server.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := clock.total(); got != 7*time.Second {
		t.Errorf("expected to honour Retry-After (7s), waited %v", got)
	}
}

func TestDoRequest_SecondaryRateLimitBacksOff(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"message": "You have exceeded a secondary rate limit."}`))
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, clock := newClockClient(server.URL)
	if _, err := client.DoRequest(server.URL); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := clock.total(); got != time.Minute {
		t.Errorf("expected a 1 minute backoff, waited %v", got)
	}
}

func TestDoRequest_ResetBeyondCapFails(t *testing.T) {
	var clock *fakeClock
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(clock.t.Add(50*time.Minute).Unix()))
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "API rate limit exceeded"}`))
	}))
	defer server.Close()

	client, c := newClockClient(server.URL)
	clock = c

	_, err := client.DoRequest(server.URL)
	if err == nil {
		t.Fatal("expected an error when the reset is beyond the wait cap")
	}
	if !strings.Contains(err.Error(), "longer than") {
		t.Errorf("expected a wait-cap error, got: %v", err)
	}
	if clock.total() != 0 {
		t.Errorf("expected no sleeping, slept %v", clock.total())
	}
}

func TestDoRequest_PermissionDeniedNotRetried(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-RateLimit-Remaining", "4000")
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"message": "Resource not accessible by integration"}`))
	}))
	defer server.Close()

	client, _ := newClockClient(server.URL)
	if _, err := client.DoRequest(server.URL); err == nil {
		t.Fatal("expected an error")
	}
	if calls != 1 {
		t.Errorf("expected a non-rate-limit 403 to fail without retrying, got %d calls", calls)
	}
}

func TestDoRequest_RetriesServerErrors(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls < 3 {
			w.WriteHeader(http.StatusGatewayTimeout)
			return
		}
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, _ := newClockClient(server.URL)
	if _, err := client.DoRequest(server.URL); err != nil {
		t.Fatalf("expected success after transient 504s, got: %v", err)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestDoRequest_PacesSearchRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, clock := newClockClient(server.URL)
	client.SearchInterval = 2 * time.Second
	client.RequestInterval = 0

	for i := 0; i < 3; i++ {
		if _, err := client.DoRequest(server.URL + "/search/issues?q=x"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}
	// No wait before the first request, then one interval before each other.
	if got := clock.total(); got != 4*time.Second {
		t.Errorf("expected 4s of pacing for 3 search requests, got %v", got)
	}

	// Non-search requests use their own bucket and are not held back.
	before := clock.total()
	if _, err := client.DoRequest(server.URL + "/repos/o/r"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clock.total() != before {
		t.Error("expected core requests not to wait on the search bucket")
	}
}

func TestDoRequest_ExhaustedBudgetDelaysNextRequest(t *testing.T) {
	var clock *fakeClock
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.Header().Set("X-RateLimit-Reset", fmt.Sprint(clock.t.Add(20*time.Second).Unix()))
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	client, c := newClockClient(server.URL)
	clock = c

	if _, err := client.DoRequest(server.URL + "/search/issues?q=a"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if clock.total() != 0 {
		t.Fatalf("first request should not wait, waited %v", clock.total())
	}
	if _, err := client.DoRequest(server.URL + "/search/issues?q=b"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := clock.total(); got != 21*time.Second {
		t.Errorf("expected the second request to wait 21s for the reset, waited %v", got)
	}
}

func TestSearchIssues_IncompleteResultsRetriedThenFails(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Write([]byte(`{"total_count": 1, "incomplete_results": true, "items": []}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	if _, err := client.SearchBountyIssues("org", "bounty"); err == nil {
		t.Fatal("expected an error for persistently incomplete results")
	}
	if calls != maxAttempts {
		t.Errorf("expected %d attempts, got %d", maxAttempts, calls)
	}
}

func TestSearchIssues_IncompleteResultsRecovers(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Write([]byte(`{"total_count": 0, "incomplete_results": true, "items": []}`))
			return
		}
		w.Write([]byte(`{"total_count": 1, "incomplete_results": false, "items": [{"number": 5, "html_url": "https://github.com/o/r/issues/5", "state": "open"}]}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.SearchBountyIssues("org", "bounty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 1 {
		t.Errorf("expected 1 issue after retry, got %d", len(issues))
	}
}

func TestSearchBountyIssues_TooManyResultsFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"total_count": 1500, "items": [{"number": 1}]}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	if _, err := client.SearchBountyIssues("org", "bounty"); err == nil {
		t.Fatal("expected an error instead of silently truncating at 1000 results")
	}
}

// scanServer serves label/text searches from a query→items map and answers
// PR-count queries with prCount (or 500 when prCount < 0).
func scanServer(t *testing.T, results map[string]string, prCount int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if strings.Contains(q, "is:pr") {
			if prCount < 0 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			fmt.Fprintf(w, `{"total_count": %d, "items": []}`, prCount)
			return
		}
		for needle, items := range results {
			if strings.Contains(q, needle) {
				n := strings.Count(items, `"number"`)
				fmt.Fprintf(w, `{"total_count": %d, "items": [%s]}`, n, items)
				return
			}
		}
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
}

func issueJSON(number int, title, body string, labels ...string) string {
	quoted := make([]string, len(labels))
	for i, l := range labels {
		quoted[i] = fmt.Sprintf(`{"name": %q}`, l)
	}
	return fmt.Sprintf(`{"number": %d, "title": %q, "body": %q, "html_url": "https://github.com/org/repo/issues/%d", "state": "open", "labels": [%s], "user": {"login": "u"}}`,
		number, title, body, number, strings.Join(quoted, ","))
}

func scanNumbers(issues []Issue) map[int]bool {
	got := make(map[int]bool)
	for _, i := range issues {
		got[i.Number] = true
	}
	return got
}

func TestScanAll_ExcludesUnavailableBounties(t *testing.T) {
	server := scanServer(t, map[string]string{
		`label:"💎 Bounty"`: strings.Join([]string{
			issueJSON(1, "open bounty", "", "💎 Bounty"),
			issueJSON(2, "already paid", "", "💎 Bounty", "💰 Rewarded"),
			// Variation selector differs from the config spelling.
			issueJSON(3, "on hold", "", "💎 Bounty", "\u23f8 Hold"),
			issueJSON(4, "core only", "", "💎 Bounty", "👥 Core Team Only"),
		}, ","),
	}, 0)
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.ScanAll(&Config{
		Organizations: []Organization{{Name: "org", Labels: []string{"💎 Bounty"}}},
		ExcludeLabels: []string{"💰 Rewarded", "\u23f8\ufe0f Hold", "👥 Core Team Only"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := scanNumbers(issues)
	if len(issues) != 1 || !got[1] {
		t.Errorf("expected only issue #1, got %v", got)
	}
}

func TestScanAll_ExcludesProjectItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" {
			w.Write([]byte(`{"data": {"organization": {"projectV2": {"items": {"pageInfo": {"hasNextPage": false}, "nodes": [
				{"content": {"number": 7, "title": "paid", "url": "https://github.com/org/repo/issues/7", "state": "OPEN", "labels": {"nodes": [{"name": "bounty"}, {"name": "💰 Rewarded"}]}, "repository": {"nameWithOwner": "org/repo"}}}
			]}}}}}`))
			return
		}
		w.Write([]byte(`{"total_count": 0, "items": []}`))
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.ScanAll(&Config{
		Organizations: []Organization{{Name: "org", Labels: []string{"bounty"}}},
		Projects:      []ProjectBoard{{OrgLogin: "org", ProjectNumber: 1}},
		ExcludeLabels: []string{"💰 Rewarded"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(issues) != 0 {
		t.Errorf("expected the rewarded project item to be excluded, got %d issues", len(issues))
	}
}

func TestScanAll_SkipsStaleLabelHits(t *testing.T) {
	// The search index still returns #2 for the label after it was removed.
	server := scanServer(t, map[string]string{
		`label:"💎 Bounty"`: issueJSON(1, "a", "", "💎 Bounty") + "," + issueJSON(2, "b", "", "enhancement"),
	}, 0)
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.ScanAll(&Config{
		Organizations: []Organization{{Name: "org", Labels: []string{"💎 Bounty"}}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := scanNumbers(issues); len(issues) != 1 || !got[1] {
		t.Errorf("expected only issue #1, got %v", got)
	}
}

func TestScanAll_TextSearches(t *testing.T) {
	var queries []string
	inner := scanServer(t, map[string]string{
		`in:title "bounty"`: strings.Join([]string{
			issueJSON(10, "rock tracks [bounty: 40 XTR]", "", "audio"),
			// Stem match only: must be dropped.
			issueJSON(11, "Add a bounties page", "", "enhancement"),
			issueJSON(12, "excluded [bounty: 5 XTR]", "", "💰 Rewarded"),
		}, ","),
		`in:body "retro bounty"`: issueJSON(20, "Write the wiki page", "This is a Retro\nbounty on merge."),
	}, 0)
	defer inner.Close()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries = append(queries, r.URL.Query().Get("q"))
		inner.Config.Handler.ServeHTTP(w, r)
	}))
	defer server.Close()

	client := newMockClient(server.URL, "test")
	issues, err := client.ScanAll(&Config{
		Organizations: []Organization{{
			Name:   "org",
			Labels: []string{"bounty"},
			TextSearches: []TextSearch{
				{In: "title", Phrase: "bounty"},
				{In: "body", Phrase: "retro bounty"},
			},
		}},
		ExcludeLabels: []string{"💰 Rewarded"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := scanNumbers(issues)
	if len(issues) != 2 || !got[10] || !got[20] {
		t.Errorf("expected issues #10 and #20, got %v", got)
	}
	want := `is:open is:issue org:org in:title "bounty"`
	found := false
	for _, q := range queries {
		if q == want {
			found = true
		}
	}
	if !found {
		t.Errorf("expected query %q, got %v", want, queries)
	}
}

func TestScanAll_PRCountFailureFailsScan(t *testing.T) {
	server := scanServer(t, map[string]string{
		`label:"bounty"`: issueJSON(1, "a", "", "bounty"),
	}, -1)
	defer server.Close()

	client := newMockClient(server.URL, "test")
	_, err := client.ScanAll(&Config{
		Organizations: []Organization{{Name: "org", Labels: []string{"bounty"}}},
	})
	if err == nil {
		t.Fatal("expected the scan to fail when a PR count cannot be fetched")
	}
	if !strings.Contains(err.Error(), "counting open PRs for org/repo#1") {
		t.Errorf("expected error to name the issue, got: %v", err)
	}
}

func TestNormalizeLabel(t *testing.T) {
	tests := map[string]string{
		"⏸️ Hold":          "⏸ hold",
		"⏸ Hold":           "⏸ hold",
		"  💎   Bounty ":    "💎 bounty",
		"bounty_candidate": "bounty_candidate",
	}
	for in, want := range tests {
		if got := normalizeLabel(in); got != want {
			t.Errorf("normalizeLabel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTruncate_KeepsRunesIntact(t *testing.T) {
	if got := truncate("💎💎💎", 2); got != "💎💎" {
		t.Errorf("truncate split a rune: %q", got)
	}
	if got := truncate("short", 50); got != "short" {
		t.Errorf("truncate changed a short string: %q", got)
	}
}
