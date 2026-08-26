package reporter_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/reporter"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func sampleResult() analyzer.Result {
	return analyzer.Result{
		FilesAnalyzed:     2,
		FunctionsAnalyzed: 4,
		Violations: []analyzer.Violation{
			{
				RuleID:    rules.FunctionLength,
				Severity:  rules.SeverityMajor,
				File:      "internal/call/service.go",
				Line:      42,
				Column:    1,
				Function:  "ProcessCall",
				Actual:    34,
				Threshold: 30,
				Message:   "function_length is 34 (major threshold 30)",
				IsNew:     true,
			},
			{
				RuleID:    rules.ReturnsPerFunction,
				Severity:  rules.SeverityMinor,
				File:      "internal/call/service.go",
				Line:      87,
				Column:    1,
				Function:  "ValidateRequest",
				Actual:    6,
				Threshold: 5,
				Message:   "returns_per_function is 6 (minor threshold 5)",
				IsNew:     true,
			},
		},
	}
}

func TestBuildSummaryFailOnMajor(t *testing.T) {
	res := sampleResult()
	sum := reporter.BuildSummary(res, rules.SeverityMajor)
	if sum.Passed {
		t.Fatal("expected fail on new major")
	}
	if sum.NewMajor != 1 || sum.NewMinor != 1 {
		t.Fatalf("counts=%d/%d", sum.NewMajor, sum.NewMinor)
	}
}

func TestBuildSummaryFailOnNever(t *testing.T) {
	res := sampleResult()
	sum := reporter.BuildSummary(res, rules.SeverityPass)
	if !sum.Passed {
		t.Fatal("fail_on never should pass")
	}
}

func TestBuildSummaryLegacyDoesNotFail(t *testing.T) {
	res := analyzer.Result{
		Violations: []analyzer.Violation{{
			RuleID:   rules.Complexity,
			Severity: rules.SeverityMajor,
			IsNew:    false,
			Actual:   12,
		}},
	}
	sum := reporter.BuildSummary(res, rules.SeverityMajor)
	if !sum.Passed {
		t.Fatal("legacy major should not fail gate")
	}
}

func TestWriteTerminal(t *testing.T) {
	var buf bytes.Buffer
	res := sampleResult()
	sum := reporter.BuildSummary(res, rules.SeverityMajor)
	if err := reporter.WriteTerminal(&buf, res, sum, ""); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if !strings.Contains(out, "MAJOR") || !strings.Contains(out, "ProcessCall") {
		t.Fatalf("terminal output missing expected content:\n%s", out)
	}
	if !strings.Contains(out, "Result: FAILED") {
		t.Fatalf("expected FAILED:\n%s", out)
	}
}

func TestWriteJSON(t *testing.T) {
	var buf bytes.Buffer
	res := sampleResult()
	sum := reporter.BuildSummary(res, rules.SeverityMajor)
	if err := reporter.WriteJSON(&buf, res, sum, reporter.Meta{Commit: "abc", Base: "main"}); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["result"] != "FAILED" {
		t.Fatalf("result=%v", doc["result"])
	}
	vs, ok := doc["violations"].([]any)
	if !ok || len(vs) != 2 {
		t.Fatalf("violations=%v", doc["violations"])
	}
}

func TestWriteSARIF(t *testing.T) {
	var buf bytes.Buffer
	res := sampleResult()
	sum := reporter.BuildSummary(res, rules.SeverityMajor)
	if err := reporter.WriteSARIF(&buf, res, sum, reporter.Meta{Base: "main"}); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	if doc["version"] != "2.1.0" {
		t.Fatalf("version=%v", doc["version"])
	}
	runs := doc["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results=%d", len(results))
	}
	first := results[0].(map[string]any)
	if first["ruleId"] != string(rules.FunctionLength) {
		t.Fatalf("ruleId=%v", first["ruleId"])
	}
	if first["level"] != "error" {
		t.Fatalf("level=%v", first["level"])
	}
}
