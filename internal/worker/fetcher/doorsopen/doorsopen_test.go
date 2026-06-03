package doorsopen

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func readFixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

func TestParseCompanyPage_1Job(t *testing.T) {
	count, ids, err := parseCompanyPage(readFixture(t, "company_1job.html"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expectedCount: want 1, got %d", count)
	}
	if len(ids) != 1 || ids[0] != "15174" {
		t.Errorf("ids: want [15174], got %v", ids)
	}
}

func TestParseCompanyPage_0Jobs(t *testing.T) {
	count, ids, err := parseCompanyPage(readFixture(t, "company_0jobs.html"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expectedCount: want 0, got %d", count)
	}
	if len(ids) != 0 {
		t.Errorf("ids: want [], got %v", ids)
	}
}

// If Doors Open removes the "Jobs (N)" tab label (template rename, A/B test,
// etc), parseCompanyPage must error out so the worker telemetry surfaces it
// rather than silently reporting zero jobs.
func TestParseCompanyPage_MissingJobCountSentinel(t *testing.T) {
	body := []byte(`<html><body><div id="all-vacancy"></div></body></html>`)
	_, _, err := parseCompanyPage(body)
	if !errors.Is(err, ErrMissingJobCount) {
		t.Fatalf("want ErrMissingJobCount, got %v", err)
	}
}

// Same defense for the tab panel container itself.
func TestParseCompanyPage_MissingTabPanelSentinel(t *testing.T) {
	body := []byte(`<a aria-controls="all-vacancy" data-toggle="tab">Jobs (3)</a>`)
	_, _, err := parseCompanyPage(body)
	if !errors.Is(err, ErrMissingTabPanel) {
		t.Fatalf("want ErrMissingTabPanel, got %v", err)
	}
}
