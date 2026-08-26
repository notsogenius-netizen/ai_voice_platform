// Package reporter formats analysis results for humans and machines.
package reporter

import (
	"fmt"
	"io"
	"strings"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

// Summary is a roll-up used by all output formats.
type Summary struct {
	FilesAnalyzed     int
	FunctionsAnalyzed int
	NewMajor          int
	NewMinor          int
	LegacyMajor       int
	LegacyMinor       int
	Passed            bool
	FailOn            rules.Severity
}

// BuildSummary counts new vs legacy violations and applies the fail-on gate.
func BuildSummary(res analyzer.Result, failOn rules.Severity) Summary {
	s := Summary{
		FilesAnalyzed:     res.FilesAnalyzed,
		FunctionsAnalyzed: res.FunctionsAnalyzed,
		FailOn:            failOn,
	}
	for _, v := range res.Violations {
		tallyViolation(&s, v)
	}
	s.Passed = !gateFailed(s, failOn)
	return s
}

func tallyViolation(s *Summary, v analyzer.Violation) {
	switch {
	case v.IsNew && v.Severity == rules.SeverityMajor:
		s.NewMajor++
	case v.IsNew && v.Severity == rules.SeverityMinor:
		s.NewMinor++
	case !v.IsNew && v.Severity == rules.SeverityMajor:
		s.LegacyMajor++
	case !v.IsNew && v.Severity == rules.SeverityMinor:
		s.LegacyMinor++
	}
}

func severityLabel(sev rules.Severity) string {
	if sev == rules.SeverityMajor {
		return "Major"
	}
	if sev == rules.SeverityMinor {
		return "Minor"
	}
	return string(sev)
}

func gateFailed(s Summary, failOn rules.Severity) bool {
	switch failOn {
	case rules.SeverityMinor:
		return s.NewMajor > 0 || s.NewMinor > 0
	case rules.SeverityMajor:
		return s.NewMajor > 0
	default:
		return false
	}
}

// WriteTerminal writes a human-readable report.
func WriteTerminal(w io.Writer, res analyzer.Result, sum Summary, baseRef string) error {
	var b strings.Builder
	writeTerminalHeader(&b, baseRef)
	writeTerminalViolations(&b, res, baseRef)
	writeTerminalFooter(&b, res, sum, baseRef)
	_, err := io.WriteString(w, b.String())
	return err
}

func writeTerminalHeader(b *strings.Builder, baseRef string) {
	b.WriteString("Code Quality Report\n")
	b.WriteString("────────────────────────────────────────\n\n")
	if baseRef != "" {
		fmt.Fprintf(b, "Base: %s (new violations only affect the gate)\n\n", baseRef)
	}
}

func writeTerminalViolations(b *strings.Builder, res analyzer.Result, baseRef string) {
	shown := 0
	for _, v := range res.Violations {
		if baseRef != "" && !v.IsNew {
			continue
		}
		shown++
		writeOneViolation(b, v, baseRef)
	}
	if shown == 0 {
		b.WriteString("No violations to report.\n\n")
	}
}

func writeOneViolation(b *strings.Builder, v analyzer.Violation, baseRef string) {
	icon, label := "⚠️", "MINOR"
	if v.Severity == rules.SeverityMajor {
		icon, label = "❌", "MAJOR"
	}
	fmt.Fprintf(b, "%s %s  %s:%d\n", icon, label, v.File, v.Line)
	fmt.Fprintf(b, "   Rule: %s\n", rules.ShortName(v.RuleID))
	if v.Function != "" {
		fmt.Fprintf(b, "   Function: %s\n", v.Function)
	}
	fmt.Fprintf(b, "   Actual: %d %s\n", v.Actual, rules.Unit(v.RuleID))
	fmt.Fprintf(b, "   %s threshold: %d\n", severityLabel(v.Severity), v.Threshold)
	if baseRef != "" {
		fmt.Fprintf(b, "   New: %v\n", v.IsNew)
	}
	b.WriteString("\n")
}

func writeTerminalFooter(b *strings.Builder, res analyzer.Result, sum Summary, baseRef string) {
	b.WriteString("────────────────────────────────────────\n\n")
	fmt.Fprintf(b, "Files analyzed:       %d\n", sum.FilesAnalyzed)
	fmt.Fprintf(b, "Functions analyzed:   %d\n", sum.FunctionsAnalyzed)
	fmt.Fprintf(b, "\n")
	fmt.Fprintf(b, "New major violations: %d\n", sum.NewMajor)
	fmt.Fprintf(b, "New minor violations: %d\n", sum.NewMinor)
	if baseRef != "" {
		fmt.Fprintf(b, "Legacy major:         %d\n", sum.LegacyMajor)
		fmt.Fprintf(b, "Legacy minor:         %d\n", sum.LegacyMinor)
	}
	fmt.Fprintf(b, "\n")
	if sum.Passed {
		b.WriteString("Result: PASSED\n")
	} else {
		b.WriteString("Result: FAILED\n")
	}
	if len(res.ParseErrors) == 0 {
		return
	}
	b.WriteString("\nParse errors:\n")
	for _, e := range res.ParseErrors {
		fmt.Fprintf(b, "  - %s\n", e)
	}
}
