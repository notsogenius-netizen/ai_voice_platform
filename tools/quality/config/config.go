// Package config loads repository-level quality checker settings.
package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

// DefaultConfigPath is the repository-root config file name.
const DefaultConfigPath = ".quality.yaml"

// File is the on-disk YAML shape.
type File struct {
	Quality QualitySection `yaml:"quality"`
	Rules   RulesSection   `yaml:"rules"`
}

// QualitySection holds gate-level settings.
type QualitySection struct {
	FailOn string `yaml:"fail_on"`
}

// RulesSection holds per-rule thresholds keyed by short names.
type RulesSection struct {
	MethodsPerFile     *rules.Thresholds `yaml:"methods_per_file"`
	ReturnsPerFunction *rules.Thresholds `yaml:"returns_per_function"`
	Complexity         *rules.Thresholds `yaml:"complexity"`
	LOCPerFile         *rules.Thresholds `yaml:"loc_per_file"`
	FunctionLength     *rules.Thresholds `yaml:"function_length"`
	Parameters         *rules.Thresholds `yaml:"parameters"`
	NestingDepth       *rules.Thresholds `yaml:"nesting_depth"`
}

// Config is the resolved runtime configuration.
type Config struct {
	FailOn     rules.Severity
	Thresholds map[rules.RuleID]rules.Thresholds
	Path       string
}

// Load reads path if it exists; otherwise returns defaults.
func Load(path string) (Config, error) {
	cfg := Defaults()
	cfg.Path = path
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}
	return applyFile(cfg, path, data)
}

func applyFile(cfg Config, path string, data []byte) (Config, error) {
	var raw File
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return Config{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	if err := applyQualitySection(&cfg, raw.Quality); err != nil {
		return Config{}, err
	}
	applyRulesSection(cfg.Thresholds, raw.Rules)
	if err := validate(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func applyQualitySection(cfg *Config, q QualitySection) error {
	if q.FailOn == "" {
		return nil
	}
	failOn, err := parseFailOn(q.FailOn)
	if err != nil {
		return err
	}
	cfg.FailOn = failOn
	return nil
}

func applyRulesSection(dst map[rules.RuleID]rules.Thresholds, raw RulesSection) {
	pairs := []struct {
		id  rules.RuleID
		src *rules.Thresholds
	}{
		{rules.MethodsPerFile, raw.MethodsPerFile},
		{rules.ReturnsPerFunction, raw.ReturnsPerFunction},
		{rules.Complexity, raw.Complexity},
		{rules.LOCPerFile, raw.LOCPerFile},
		{rules.FunctionLength, raw.FunctionLength},
		{rules.Parameters, raw.Parameters},
		{rules.NestingDepth, raw.NestingDepth},
	}
	for _, p := range pairs {
		if p.src != nil {
			dst[p.id] = *p.src
		}
	}
}

// Defaults returns built-in configuration.
func Defaults() Config {
	return Config{
		FailOn:     rules.DefaultFailOn,
		Thresholds: rules.DefaultThresholds(),
	}
}

func parseFailOn(v string) (rules.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "major":
		return rules.SeverityMajor, nil
	case "minor":
		return rules.SeverityMinor, nil
	case "never", "none", "off":
		return rules.SeverityPass, nil
	default:
		return "", fmt.Errorf("invalid quality.fail_on %q (want major, minor, or never)", v)
	}
}

func validate(cfg Config) error {
	for id, t := range cfg.Thresholds {
		if t.Minor < 0 || t.Major < 0 {
			return fmt.Errorf("rule %s: thresholds must be non-negative", id)
		}
		if t.Minor > t.Major {
			return fmt.Errorf("rule %s: minor (%d) must be <= major (%d)", id, t.Minor, t.Major)
		}
	}
	return nil
}
