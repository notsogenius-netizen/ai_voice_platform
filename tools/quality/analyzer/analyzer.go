package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sourabh/ai-voice-platform/tools/quality/config"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

// Violation is a single rule failure (minor or major).
type Violation struct {
	RuleID    rules.RuleID   `json:"rule_id"`
	Severity  rules.Severity `json:"severity"`
	File      string         `json:"file"`
	Line      int            `json:"line"`
	Column    int            `json:"column"`
	Function  string         `json:"function,omitempty"`
	Actual    int            `json:"actual"`
	Threshold int            `json:"threshold"`
	Message   string         `json:"message"`
	IsNew     bool           `json:"is_new"`
}

// MetricKey uniquely identifies a measured entity for baseline comparison.
type MetricKey struct {
	RuleID   rules.RuleID
	File     string
	Function string
}

// Metric is a measured value before severity classification.
type Metric struct {
	Key    MetricKey
	Actual int
	Line   int
	Column int
}

// FileResult holds analysis for one Go file.
type FileResult struct {
	Path     string
	Metrics  []Metric
	ParseErr string
}

// Result is a full analysis run.
type Result struct {
	FilesAnalyzed     int
	FunctionsAnalyzed int
	Violations        []Violation
	ParseErrors       []string
}

// Options controls an analysis run.
type Options struct {
	Config config.Config
	Root   string
}

// AnalyzePaths parses and evaluates all .go files under the given paths.
func AnalyzePaths(paths []string, opts Options) (Result, error) {
	files, err := collectGoFiles(paths)
	if err != nil {
		return Result{}, err
	}
	return analyzeFiles(files, opts, nil)
}

func analyzeFiles(files []string, opts Options, contentByPath map[string][]byte) (Result, error) {
	var out Result
	fset := token.NewFileSet()
	for _, path := range files {
		rel := relPath(opts.Root, path)
		if shouldSkip(rel) {
			continue
		}
		analyzeOneFile(fset, path, rel, contentByPath, opts, &out)
	}
	sortViolations(out.Violations)
	return out, nil
}

func analyzeOneFile(
	fset *token.FileSet,
	path, rel string,
	contentByPath map[string][]byte,
	opts Options,
	out *Result,
) {
	src, err := readSource(path, contentByPath)
	if err != nil {
		out.ParseErrors = append(out.ParseErrors, path+": "+err.Error())
		return
	}
	fr, fnCount := analyzeSource(fset, path, rel, src, opts.Config)
	out.FilesAnalyzed++
	out.FunctionsAnalyzed += fnCount
	if fr.ParseErr != "" {
		out.ParseErrors = append(out.ParseErrors, fr.ParseErr)
		return
	}
	out.Violations = append(out.Violations, metricsToViolations(fr.Metrics, opts.Config)...)
}

func metricsToViolations(ms []Metric, cfg config.Config) []Violation {
	var out []Violation
	for _, m := range ms {
		thr := cfg.Thresholds[m.Key.RuleID]
		sev := thr.Classify(m.Actual)
		if sev == rules.SeverityPass {
			continue
		}
		out = append(out, Violation{
			RuleID:    m.Key.RuleID,
			Severity:  sev,
			File:      m.Key.File,
			Line:      m.Line,
			Column:    m.Column,
			Function:  m.Key.Function,
			Actual:    m.Actual,
			Threshold: thr.ThresholdFor(sev),
			Message:   rules.Message(m.Key.RuleID, m.Actual, sev, thr),
			IsNew:     true,
		})
	}
	return out
}

func readSource(path string, contentByPath map[string][]byte) ([]byte, error) {
	if contentByPath != nil {
		if data, ok := contentByPath[path]; ok {
			return data, nil
		}
		if data, ok := contentByPath[filepath.ToSlash(path)]; ok {
			return data, nil
		}
	}
	return os.ReadFile(path)
}

func analyzeSource(fset *token.FileSet, absPath, rel string, src []byte, cfg config.Config) (FileResult, int) {
	fr := FileResult{Path: rel}
	file, err := parser.ParseFile(fset, absPath, src, parser.ParseComments)
	if err != nil {
		fr.ParseErr = rel + ": " + err.Error()
		return fr, 0
	}
	_ = cfg
	fr.Metrics = append(fr.Metrics, fileLOCMetric(fset, file, rel, src))
	fnCount := 0
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		fnCount++
		fr.Metrics = append(fr.Metrics, functionMetrics(fset, file, fd, rel, src)...)
	}
	fr.Metrics = append(fr.Metrics, Metric{
		Key:    MetricKey{RuleID: rules.MethodsPerFile, File: rel},
		Actual: fnCount,
		Line:   1,
		Column: 1,
	})
	return fr, fnCount
}

func fileLOCMetric(fset *token.FileSet, file *ast.File, rel string, src []byte) Metric {
	pos := fset.Position(file.Pos())
	return Metric{
		Key:    MetricKey{RuleID: rules.LOCPerFile, File: rel},
		Actual: countFileLOC(fset, file, src),
		Line:   pos.Line,
		Column: pos.Column,
	}
}

func functionMetrics(fset *token.FileSet, file *ast.File, fd *ast.FuncDecl, rel string, src []byte) []Metric {
	name := funcName(fd)
	fnPos := fset.Position(fd.Pos())
	values := []struct {
		id     rules.RuleID
		actual int
	}{
		{rules.ReturnsPerFunction, countReturns(fd)},
		{rules.Complexity, cyclomaticComplexity(fd)},
		{rules.FunctionLength, countFunctionLOC(fset, file, fd, src)},
		{rules.Parameters, countParams(fd)},
		{rules.NestingDepth, maxNestingDepth(fd)},
	}
	out := make([]Metric, 0, len(values))
	for _, v := range values {
		out = append(out, Metric{
			Key:    MetricKey{RuleID: v.id, File: rel, Function: name},
			Actual: v.actual,
			Line:   fnPos.Line,
			Column: fnPos.Column,
		})
	}
	return out
}

func funcName(fd *ast.FuncDecl) string {
	name := "func"
	if fd.Name != nil {
		name = fd.Name.Name
	}
	if fd.Recv == nil || len(fd.Recv.List) == 0 {
		return name
	}
	return receiverTypeName(fd.Recv.List[0].Type) + "." + name
}

func receiverTypeName(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + receiverTypeName(t.X)
	case *ast.IndexExpr:
		return receiverTypeName(t.X)
	case *ast.IndexListExpr:
		return receiverTypeName(t.X)
	default:
		return "T"
	}
}

func collectGoFiles(paths []string) ([]string, error) {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range paths {
		if err := walkGoPath(p, seen, &out); err != nil {
			return nil, err
		}
	}
	sort.Strings(out)
	return out, nil
}

func walkGoPath(root string, seen map[string]struct{}, out *[]string) error {
	return filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipDirIfNeeded(d.Name())
		}
		return considerGoFile(path, seen, out)
	})
}

func skipDirIfNeeded(name string) error {
	switch name {
	case "vendor", ".git", "testdata", "node_modules":
		return filepath.SkipDir
	default:
		return nil
	}
}

func considerGoFile(path string, seen map[string]struct{}, out *[]string) error {
	if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
		return nil
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	if _, ok := seen[abs]; ok {
		return nil
	}
	seen[abs] = struct{}{}
	*out = append(*out, abs)
	return nil
}

func shouldSkip(rel string) bool {
	rel = filepath.ToSlash(rel)
	return strings.Contains(rel, "/testdata/") || strings.HasPrefix(rel, "testdata/")
}

func relPath(root, path string) string {
	if root == "" {
		return filepath.ToSlash(path)
	}
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(rel)
}

func sortViolations(vs []Violation) {
	sort.Slice(vs, func(i, j int) bool {
		a, b := vs[i], vs[j]
		if a.Severity.Rank() != b.Severity.Rank() {
			return a.Severity.Rank() > b.Severity.Rank()
		}
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		if a.RuleID != b.RuleID {
			return a.RuleID < b.RuleID
		}
		return a.Function < b.Function
	})
}
