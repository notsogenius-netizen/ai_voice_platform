package reporter

import (
	"encoding/json"
	"io"
	"time"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

// JSONReport is the machine-readable report document.
type JSONReport struct {
	GeneratedAt       string               `json:"generated_at"`
	Repository        string               `json:"repository,omitempty"`
	Commit            string               `json:"commit,omitempty"`
	Base              string               `json:"base,omitempty"`
	FailOn            rules.Severity       `json:"fail_on"`
	FilesAnalyzed     int                  `json:"files_analyzed"`
	FunctionsAnalyzed int                  `json:"functions_analyzed"`
	NewMajor          int                  `json:"new_major_violations"`
	NewMinor          int                  `json:"new_minor_violations"`
	LegacyMajor       int                  `json:"legacy_major_violations"`
	LegacyMinor       int                  `json:"legacy_minor_violations"`
	Passed            bool                 `json:"passed"`
	Result            string               `json:"result"`
	Violations        []analyzer.Violation `json:"violations"`
	ParseErrors       []string             `json:"parse_errors,omitempty"`
}

// Meta holds optional repository metadata for reports.
type Meta struct {
	Repository string
	Commit     string
	Base       string
}

// WriteJSON encodes the report as indented JSON.
func WriteJSON(w io.Writer, res analyzer.Result, sum Summary, meta Meta) error {
	rep := JSONReport{
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		Repository:        meta.Repository,
		Commit:            meta.Commit,
		Base:              meta.Base,
		FailOn:            sum.FailOn,
		FilesAnalyzed:     sum.FilesAnalyzed,
		FunctionsAnalyzed: sum.FunctionsAnalyzed,
		NewMajor:          sum.NewMajor,
		NewMinor:          sum.NewMinor,
		LegacyMajor:       sum.LegacyMajor,
		LegacyMinor:       sum.LegacyMinor,
		Passed:            sum.Passed,
		Result:            resultString(sum.Passed),
		Violations:        nonNilViolations(res.Violations),
		ParseErrors:       nonNilStrings(res.ParseErrors),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rep)
}

func nonNilViolations(vs []analyzer.Violation) []analyzer.Violation {
	if vs == nil {
		return []analyzer.Violation{}
	}
	return vs
}

func nonNilStrings(ss []string) []string {
	if ss == nil {
		return []string{}
	}
	return ss
}

func resultString(passed bool) string {
	if passed {
		return "PASSED"
	}
	return "FAILED"
}
