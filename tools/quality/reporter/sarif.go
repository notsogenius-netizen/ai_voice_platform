package reporter

import (
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

type sarifDocument struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool    sarifTool     `json:"tool"`
	Results []sarifResult `json:"results"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name           string               `json:"name"`
	InformationURI string               `json:"informationUri,omitempty"`
	Version        string               `json:"version"`
	Rules          []sarifReportingRule `json:"rules"`
}

type sarifReportingRule struct {
	ID               string             `json:"id"`
	Name             string             `json:"name,omitempty"`
	ShortDescription sarifText          `json:"shortDescription"`
	FullDescription  sarifText          `json:"fullDescription,omitempty"`
	DefaultConfig    sarifDefaultConfig `json:"defaultConfiguration,omitempty"`
}

type sarifDefaultConfig struct {
	Level string `json:"level"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifResult struct {
	RuleID     string          `json:"ruleId"`
	Level      string          `json:"level"`
	Message    sarifText       `json:"message"`
	Locations  []sarifLocation `json:"locations"`
	Properties map[string]any  `json:"properties,omitempty"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysicalLocation `json:"physicalLocation"`
}

type sarifPhysicalLocation struct {
	ArtifactLocation sarifArtifactLocation `json:"artifactLocation"`
	Region           sarifRegion           `json:"region"`
}

type sarifArtifactLocation struct {
	URI string `json:"uri"`
}

type sarifRegion struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn,omitempty"`
}

// WriteSARIF writes a SARIF 2.1.0 document suitable for GitHub code scanning.
func WriteSARIF(w io.Writer, res analyzer.Result, sum Summary, meta Meta) error {
	doc := sarifDocument{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool:    sarifTool{Driver: buildDriver()},
			Results: buildSARIFResults(res, sum, meta),
		}},
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

func buildDriver() sarifDriver {
	driverRules := make([]sarifReportingRule, 0, len(rules.AllRuleIDs))
	for _, id := range rules.AllRuleIDs {
		driverRules = append(driverRules, sarifReportingRule{
			ID:               string(id),
			Name:             rules.ShortName(id),
			ShortDescription: sarifText{Text: rules.Description(id)},
			FullDescription:  sarifText{Text: rules.Description(id)},
			DefaultConfig:    sarifDefaultConfig{Level: "warning"},
		})
	}
	return sarifDriver{
		Name:           "ai-voice-platform-quality",
		InformationURI: "https://github.com/sourabh/ai-voice-platform",
		Version:        "1.0.0",
		Rules:          driverRules,
	}
}

func buildSARIFResults(res analyzer.Result, sum Summary, meta Meta) []sarifResult {
	results := make([]sarifResult, 0, len(res.Violations))
	for _, v := range res.Violations {
		if meta.Base != "" && !v.IsNew {
			continue
		}
		results = append(results, toSARIFResult(v, sum))
	}
	return results
}

func toSARIFResult(v analyzer.Violation, sum Summary) sarifResult {
	msg := v.Message
	if v.Function != "" {
		msg = fmt.Sprintf("%s (function %s)", msg, v.Function)
	}
	return sarifResult{
		RuleID:    string(v.RuleID),
		Level:     sarifLevel(v.Severity),
		Message:   sarifText{Text: msg},
		Locations: []sarifLocation{sarifLoc(v)},
		Properties: map[string]any{
			"severity":   string(v.Severity),
			"actual":     v.Actual,
			"threshold":  v.Threshold,
			"isNew":      v.IsNew,
			"function":   v.Function,
			"gatePassed": sum.Passed,
		},
	}
}

func sarifLevel(sev rules.Severity) string {
	if sev == rules.SeverityMajor {
		return "error"
	}
	return "warning"
}

func sarifLoc(v analyzer.Violation) sarifLocation {
	return sarifLocation{
		PhysicalLocation: sarifPhysicalLocation{
			ArtifactLocation: sarifArtifactLocation{URI: filepath.ToSlash(v.File)},
			Region: sarifRegion{
				StartLine:   maxInt(1, v.Line),
				StartColumn: maxInt(1, v.Column),
			},
		},
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
