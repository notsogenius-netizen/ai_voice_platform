package analyzer

import (
	"fmt"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sourabh/ai-voice-platform/tools/quality/config"
)

// CompareOptions configures PR-vs-base analysis.
type CompareOptions struct {
	Config config.Config
	Root   string
	Base   string
	Paths  []string
}

// AnalyzeWithBase analyzes the working tree and marks violations that are new
// relative to the merge-base with Base. Empty Base reports all violations as new.
func AnalyzeWithBase(opts CompareOptions) (Result, error) {
	absRoot, paths, err := resolveComparePaths(opts)
	if err != nil {
		return Result{}, err
	}
	analyzeOpts := Options{Config: opts.Config, Root: absRoot}
	if opts.Base == "" {
		return AnalyzePaths(paths, analyzeOpts)
	}
	return analyzeAgainstBase(absRoot, opts.Base, paths, analyzeOpts)
}

func resolveComparePaths(opts CompareOptions) (string, []string, error) {
	root := opts.Root
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", nil, err
		}
		root = wd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", nil, err
	}
	paths := opts.Paths
	if len(paths) == 0 {
		paths = []string{absRoot}
	}
	return absRoot, paths, nil
}

func analyzeAgainstBase(absRoot, base string, paths []string, analyzeOpts Options) (Result, error) {
	mergeBase, err := gitMergeBase(absRoot, base)
	if err != nil {
		return Result{}, err
	}
	changed, err := gitChangedGoFiles(absRoot, mergeBase)
	if err != nil {
		return Result{}, err
	}
	changed = filterUnderPaths(absRoot, changed, paths)
	if len(changed) == 0 {
		return Result{}, nil
	}
	current, err := analyzeFiles(absPaths(absRoot, changed), analyzeOpts, nil)
	if err != nil {
		return Result{}, err
	}
	baseContents, _, err := gitShowFiles(absRoot, mergeBase, changed)
	if err != nil {
		return Result{}, err
	}
	baseIdx := collectBaseMetrics(absRoot, changed, baseContents, analyzeOpts)
	markNewViolations(&current, baseIdx)
	return current, nil
}

type metricsIndex map[MetricKey]int

func collectBaseMetrics(root string, changed []string, baseContents map[string][]byte, opts Options) metricsIndex {
	idx := metricsIndex{}
	fset := token.NewFileSet()
	for _, rel := range changed {
		data, ok := baseContents[filepath.ToSlash(rel)]
		if !ok {
			continue
		}
		abs := filepath.Join(root, filepath.FromSlash(rel))
		fr, _ := analyzeSource(fset, abs, filepath.ToSlash(rel), data, opts.Config)
		for _, m := range fr.Metrics {
			idx[m.Key] = m.Actual
		}
	}
	return idx
}

func markNewViolations(current *Result, base metricsIndex) {
	for i := range current.Violations {
		v := &current.Violations[i]
		key := MetricKey{RuleID: v.RuleID, File: v.File, Function: v.Function}
		baseVal, ok := base[key]
		if !ok {
			v.IsNew = true
			continue
		}
		v.IsNew = v.Actual > baseVal
	}
}

func gitMergeBase(root, base string) (string, error) {
	out, err := git(root, "merge-base", "HEAD", base)
	if err != nil {
		return "", fmt.Errorf("git merge-base HEAD %s: %w (fetch the base ref or pass --base)", base, err)
	}
	return strings.TrimSpace(out), nil
}

func gitChangedGoFiles(root, mergeBase string) ([]string, error) {
	out, err := git(root, "diff", "--name-status", "--diff-filter=ACMR", mergeBase, "HEAD")
	if err != nil {
		return nil, fmt.Errorf("git diff: %w", err)
	}
	files := parseNameStatus(out)
	appendDirtyGoFiles(root, &files)
	return files, nil
}

func parseNameStatus(out string) []string {
	seen := map[string]struct{}{}
	var files []string
	for _, line := range strings.Split(out, "\n") {
		rel := pathFromNameStatus(line)
		addGoPath(rel, seen, &files)
	}
	return files
}

func pathFromNameStatus(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return ""
	}
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return ""
	}
	status := parts[0]
	if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
		return parts[len(parts)-1]
	}
	return parts[1]
}

func appendDirtyGoFiles(root string, files *[]string) {
	seen := map[string]struct{}{}
	for _, f := range *files {
		seen[f] = struct{}{}
	}
	for _, args := range [][]string{
		{"diff", "--name-only", "--diff-filter=ACMR", "HEAD"},
		{"diff", "--name-only", "--diff-filter=ACMR", "--cached"},
	} {
		out, err := git(root, args...)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(out, "\n") {
			addGoPath(line, seen, files)
		}
	}
}

func addGoPath(p string, seen map[string]struct{}, files *[]string) {
	p = filepath.ToSlash(strings.TrimSpace(p))
	if p == "" || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
		return
	}
	if shouldSkip(p) {
		return
	}
	if _, ok := seen[p]; ok {
		return
	}
	seen[p] = struct{}{}
	*files = append(*files, p)
}

func gitShowFiles(root, ref string, relPaths []string) (map[string][]byte, []string, error) {
	out := map[string][]byte{}
	var missing []string
	for _, rel := range relPaths {
		data, err := gitBytes(root, "show", ref+":"+filepath.ToSlash(rel))
		if err != nil {
			missing = append(missing, rel)
			continue
		}
		out[filepath.ToSlash(rel)] = data
	}
	return out, missing, nil
}

func git(root string, args ...string) (string, error) {
	b, err := gitBytes(root, args...)
	return string(b), err
}

func gitBytes(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func absPaths(root string, rels []string) []string {
	out := make([]string, 0, len(rels))
	for _, r := range rels {
		out = append(out, filepath.Join(root, filepath.FromSlash(r)))
	}
	return out
}

func filterUnderPaths(root string, rels []string, paths []string) []string {
	if len(paths) == 0 {
		return rels
	}
	absRoots := absPathList(paths)
	var out []string
	for _, rel := range rels {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if underAny(abs, absRoots) {
			out = append(out, rel)
		}
	}
	return out
}

func absPathList(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		abs, err := filepath.Abs(p)
		if err != nil {
			continue
		}
		out = append(out, abs)
	}
	return out
}

func underAny(abs string, roots []string) bool {
	for _, r := range roots {
		if abs == r || strings.HasPrefix(abs, r+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}
