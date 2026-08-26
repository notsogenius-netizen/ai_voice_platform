package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sourabh/ai-voice-platform/tools/quality/config"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func TestLoadMissingUsesDefaults(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "nope.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FailOn != rules.SeverityMajor {
		t.Fatalf("fail_on=%s", cfg.FailOn)
	}
	if cfg.Thresholds[rules.Complexity].Minor != 5 {
		t.Fatalf("complexity minor=%d", cfg.Thresholds[rules.Complexity].Minor)
	}
}

func TestLoadOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".quality.yaml")
	content := `
quality:
  fail_on: minor
rules:
  complexity:
    minor: 3
    major: 6
  parameters:
    minor: 2
    major: 4
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.FailOn != rules.SeverityMinor {
		t.Fatalf("fail_on=%s", cfg.FailOn)
	}
	if cfg.Thresholds[rules.Complexity].Major != 6 {
		t.Fatalf("complexity=%v", cfg.Thresholds[rules.Complexity])
	}
	if cfg.Thresholds[rules.Parameters].Minor != 2 {
		t.Fatalf("parameters=%v", cfg.Thresholds[rules.Parameters])
	}
	// Unspecified rules keep defaults.
	if cfg.Thresholds[rules.FunctionLength].Minor != 20 {
		t.Fatalf("function_length default broken: %v", cfg.Thresholds[rules.FunctionLength])
	}
}

func TestLoadRejectsInvalidFailOn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".quality.yaml")
	if err := os.WriteFile(path, []byte("quality:\n  fail_on: nope\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected error")
	}
}

func TestLoadRejectsInvertedThresholds(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".quality.yaml")
	content := `
rules:
  complexity:
    minor: 10
    major: 5
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(path); err == nil {
		t.Fatal("expected error")
	}
}
