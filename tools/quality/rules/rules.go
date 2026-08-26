// Package rules defines quality rule identifiers, severities, and threshold evaluation.
package rules

import "fmt"

// Severity is the outcome of evaluating a metric against thresholds.
type Severity string

const (
	SeverityPass  Severity = "pass"
	SeverityMinor Severity = "minor"
	SeverityMajor Severity = "major"
)

// Rank returns a comparable rank (higher is worse).
func (s Severity) Rank() int {
	switch s {
	case SeverityMajor:
		return 2
	case SeverityMinor:
		return 1
	default:
		return 0
	}
}

// RuleID uniquely identifies a quality rule (stable across reports / SARIF).
type RuleID string

const (
	MethodsPerFile     RuleID = "quality/methods-per-file"
	ReturnsPerFunction RuleID = "quality/returns-per-function"
	Complexity         RuleID = "quality/complexity"
	LOCPerFile         RuleID = "quality/loc-per-file"
	FunctionLength     RuleID = "quality/function-length"
	Parameters         RuleID = "quality/parameters"
	NestingDepth       RuleID = "quality/nesting-depth"
)

// AllRuleIDs is the ordered list of built-in rules.
var AllRuleIDs = []RuleID{
	MethodsPerFile,
	ReturnsPerFunction,
	Complexity,
	LOCPerFile,
	FunctionLength,
	Parameters,
	NestingDepth,
}

// Thresholds holds inclusive upper bounds for pass/minor bands.
type Thresholds struct {
	Minor int `yaml:"minor" json:"minor"`
	Major int `yaml:"major" json:"major"`
}

// Classify returns the severity for an actual metric value.
func (t Thresholds) Classify(actual int) Severity {
	if actual <= t.Minor {
		return SeverityPass
	}
	if actual <= t.Major {
		return SeverityMinor
	}
	return SeverityMajor
}

// ThresholdFor returns the threshold that was exceeded for the given severity.
func (t Thresholds) ThresholdFor(sev Severity) int {
	if sev == SeverityMajor {
		return t.Major
	}
	return t.Minor
}

var ruleUnits = map[RuleID]string{
	MethodsPerFile:     "methods",
	ReturnsPerFunction: "returns",
	Complexity:         "complexity",
	LOCPerFile:         "lines",
	FunctionLength:     "lines",
	Parameters:         "parameters",
	NestingDepth:       "depth",
}

var ruleShortNames = map[RuleID]string{
	MethodsPerFile:     "methods_per_file",
	ReturnsPerFunction: "returns_per_function",
	Complexity:         "complexity",
	LOCPerFile:         "loc_per_file",
	FunctionLength:     "function_length",
	Parameters:         "parameters",
	NestingDepth:       "nesting_depth",
}

var ruleDescriptions = map[RuleID]string{
	MethodsPerFile:     "Number of functions and methods declared in a file",
	ReturnsPerFunction: "Number of return statements in a function",
	Complexity:         "Cyclomatic complexity of a function",
	LOCPerFile:         "Source lines of code in a file (excluding blanks, comments, package, and imports)",
	FunctionLength:     "Source lines of code in a function body (excluding blanks and comments)",
	Parameters:         "Number of parameters on a function (grouped names counted individually)",
	NestingDepth:       "Maximum nesting depth of control structures and blocks in a function",
}

// Unit describes what the metric counts (for human-readable messages).
func Unit(id RuleID) string {
	if u, ok := ruleUnits[id]; ok {
		return u
	}
	return "units"
}

// ShortName is a concise rule name for terminal output.
func ShortName(id RuleID) string {
	if n, ok := ruleShortNames[id]; ok {
		return n
	}
	return string(id)
}

// Description is a short human-readable rule description for SARIF.
func Description(id RuleID) string {
	if d, ok := ruleDescriptions[id]; ok {
		return d
	}
	return string(id)
}

// Message builds a concise violation message.
func Message(id RuleID, actual int, sev Severity, thr Thresholds) string {
	return fmt.Sprintf("%s is %d (%s threshold %d)",
		ShortName(id), actual, sev, thr.ThresholdFor(sev))
}
