// Command quality is a static Go code-quality analyzer for this repository.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/config"
	"github.com/sourabh/ai-voice-platform/tools/quality/reporter"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

type cliFlags struct {
	configPath string
	base       string
	format     string
	output     string
	failOn     string
	repoRoot   string
	quiet      bool
	paths      []string
}

func run(args []string, stdout, stderr io.Writer) int {
	flags, err := parseFlags(args, stderr)
	if err != nil {
		return 2
	}
	root, cfg, paths, err := prepareRun(flags, stderr)
	if err != nil {
		return 2
	}
	result, err := analyzer.AnalyzeWithBase(analyzer.CompareOptions{
		Config: cfg,
		Root:   root,
		Base:   flags.base,
		Paths:  paths,
	})
	if err != nil {
		fmt.Fprintf(stderr, "quality: %v\n", err)
		return 2
	}
	return writeAndExit(result, cfg, flags, root, stdout, stderr)
}

func prepareRun(flags cliFlags, stderr io.Writer) (string, config.Config, []string, error) {
	root, err := resolveRoot(flags.repoRoot)
	if err != nil {
		fmt.Fprintf(stderr, "quality: %v\n", err)
		return "", config.Config{}, nil, err
	}
	cfg, err := loadConfig(root, flags)
	if err != nil {
		fmt.Fprintf(stderr, "quality: %v\n", err)
		return "", config.Config{}, nil, err
	}
	paths := flags.paths
	if len(paths) == 0 {
		paths = defaultScanPaths(root)
	} else {
		paths = absAll(root, paths)
	}
	return root, cfg, paths, nil
}

func parseFlags(args []string, stderr io.Writer) (cliFlags, error) {
	fs := flag.NewFlagSet("quality", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var f cliFlags
	fs.StringVar(&f.configPath, "config", "", "path to .quality.yaml")
	fs.StringVar(&f.base, "base", "", "git base ref for PR comparison")
	fs.StringVar(&f.format, "format", "terminal", "output format: terminal|json|sarif")
	fs.StringVar(&f.output, "output", "", "write report to file")
	fs.StringVar(&f.failOn, "fail-on", "", "override fail_on: major|minor|never")
	fs.StringVar(&f.repoRoot, "root", "", "repository root")
	fs.BoolVar(&f.quiet, "quiet", false, "suppress terminal summary when writing files")
	if err := fs.Parse(args); err != nil {
		return cliFlags{}, err
	}
	f.paths = fs.Args()
	return f, nil
}

func loadConfig(root string, flags cliFlags) (config.Config, error) {
	cfgFile := flags.configPath
	if cfgFile == "" {
		cfgFile = filepath.Join(root, config.DefaultConfigPath)
	}
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return config.Config{}, err
	}
	if flags.failOn == "" {
		return cfg, nil
	}
	sev, err := parseFailOnFlag(flags.failOn)
	if err != nil {
		return config.Config{}, err
	}
	cfg.FailOn = sev
	return cfg, nil
}

func writeAndExit(
	result analyzer.Result,
	cfg config.Config,
	flags cliFlags,
	root string,
	stdout, stderr io.Writer,
) int {
	sum := reporter.BuildSummary(result, cfg.FailOn)
	meta := reporter.Meta{
		Repository: gitRemote(root),
		Commit:     gitHead(root),
		Base:       flags.base,
	}
	if err := writeReport(result, sum, meta, flags, stdout, stderr); err != nil {
		fmt.Fprintf(stderr, "quality: %v\n", err)
		return 2
	}
	if !sum.Passed {
		return 1
	}
	return 0
}

func writeReport(
	result analyzer.Result,
	sum reporter.Summary,
	meta reporter.Meta,
	flags cliFlags,
	stdout, stderr io.Writer,
) error {
	out := stdout
	var closer io.Closer
	if flags.output != "" {
		f, err := os.Create(flags.output)
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		closer = f
		out = f
	}
	if closer != nil {
		defer closer.Close()
	}
	if err := encodeFormat(out, result, sum, meta, flags.format); err != nil {
		return err
	}
	if !flags.quiet && flags.output != "" && flags.format != "terminal" && flags.format != "text" {
		_ = reporter.WriteTerminal(stdout, result, sum, flags.base)
	}
	_ = stderr
	return nil
}

func encodeFormat(out io.Writer, result analyzer.Result, sum reporter.Summary, meta reporter.Meta, format string) error {
	switch strings.ToLower(format) {
	case "terminal", "text":
		return reporter.WriteTerminal(out, result, sum, meta.Base)
	case "json":
		return reporter.WriteJSON(out, result, sum, meta)
	case "sarif":
		return reporter.WriteSARIF(out, result, sum, meta)
	default:
		return fmt.Errorf("unknown format %q", format)
	}
}

func parseFailOnFlag(v string) (rules.Severity, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "major":
		return rules.SeverityMajor, nil
	case "minor":
		return rules.SeverityMinor, nil
	case "never", "none", "off":
		return rules.SeverityPass, nil
	default:
		return "", fmt.Errorf("invalid --fail-on %q", v)
	}
}

func resolveRoot(explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = wd
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), nil
	}
	return wd, nil
}

func defaultScanPaths(root string) []string {
	candidates := []string{
		filepath.Join(root, "services"),
		filepath.Join(root, "pkg"),
		filepath.Join(root, "tools"),
	}
	var paths []string
	for _, c := range candidates {
		if st, err := os.Stat(c); err == nil && st.IsDir() {
			paths = append(paths, c)
		}
	}
	if len(paths) == 0 {
		return []string{root}
	}
	return paths
}

func absAll(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if filepath.IsAbs(p) {
			out = append(out, p)
			continue
		}
		out = append(out, filepath.Join(root, p))
	}
	return out
}

func gitHead(root string) string {
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitRemote(root string) string {
	out, err := exec.Command("git", "-C", root, "config", "--get", "remote.origin.url").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
