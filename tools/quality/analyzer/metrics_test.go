package analyzer_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/sourabh/ai-voice-platform/tools/quality/analyzer"
	"github.com/sourabh/ai-voice-platform/tools/quality/config"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func testdataRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Join(filepath.Dir(file), "..", "testdata")
}

func analyzeDir(t *testing.T, sub string) analyzer.Result {
	t.Helper()
	root := testdataRoot(t)
	dir := filepath.Join(root, sub)
	cfg := config.Defaults()
	res, err := analyzer.AnalyzePaths([]string{dir}, analyzer.Options{
		Config: cfg,
		Root:   root,
	})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func hasViolation(res analyzer.Result, id rules.RuleID, sev rules.Severity, fn string) bool {
	for _, v := range res.Violations {
		if v.RuleID == id && v.Severity == sev {
			if fn == "" || v.Function == fn {
				return true
			}
		}
	}
	return false
}

func TestPassFixturesHaveNoViolations(t *testing.T) {
	res := analyzeDir(t, "pass")
	if len(res.Violations) != 0 {
		t.Fatalf("unexpected violations: %+v", res.Violations)
	}
}

func TestSkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	// Deliberately awful test file that would fail every rule if analyzed.
	badTest := `package p
func TooMany(a,b,c,d,e,f,g,h,i int) int {
	if a==0{return 0};if a==1{return 1};if a==2{return 2}
	if a==3{return 3};if a==4{return 4};if a==5{return 5}
	if a==6{return 6};return 7
}
`
	prod := "package p\n\nfunc OK() int { return 1 }\n"
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(prod), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ok_test.go"), []byte(badTest), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := analyzer.AnalyzePaths([]string{dir}, analyzer.Options{
		Config: config.Defaults(),
		Root:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.FilesAnalyzed != 1 {
		t.Fatalf("FilesAnalyzed=%d want 1 (test file should be ignored)", res.FilesAnalyzed)
	}
	if len(res.Violations) != 0 {
		t.Fatalf("test file should not produce violations: %+v", res.Violations)
	}
}


func TestMinorReturns(t *testing.T) {
	res := analyzeDir(t, "minor")
	if !hasViolation(res, rules.ReturnsPerFunction, rules.SeverityMinor, "SixReturns") {
		t.Fatalf("expected minor returns violation: %+v", res.Violations)
	}
}

func TestMinorParams(t *testing.T) {
	res := analyzeDir(t, "minor")
	if !hasViolation(res, rules.Parameters, rules.SeverityMinor, "SixParams") {
		t.Fatalf("expected minor params: %+v", res.Violations)
	}
}

func TestMajorReturnsAndParams(t *testing.T) {
	res := analyzeDir(t, "major")
	if !hasViolation(res, rules.ReturnsPerFunction, rules.SeverityMajor, "EightReturns") {
		t.Fatalf("expected major returns: %+v", res.Violations)
	}
	if !hasViolation(res, rules.Parameters, rules.SeverityMajor, "NineParams") {
		t.Fatalf("expected major params: %+v", res.Violations)
	}
	if !hasViolation(res, rules.Complexity, rules.SeverityMajor, "HighComplexity") {
		t.Fatalf("expected major complexity: %+v", res.Violations)
	}
	if !hasViolation(res, rules.NestingDepth, rules.SeverityMajor, "DeepNesting") {
		t.Fatalf("expected major nesting: %+v", res.Violations)
	}
}

func TestGroupedParameters(t *testing.T) {
	root := testdataRoot(t)
	cfg := config.Defaults()
	// Lower thresholds so Grouped(3) is still pass, Mixed(5) pass, but we assert metrics.
	res, err := analyzer.AnalyzePaths([]string{filepath.Join(root, "params")}, analyzer.Options{
		Config: cfg,
		Root:   root,
	})
	if err != nil {
		t.Fatal(err)
	}
	// With defaults, Mixed has 5 params → pass. Force check via lowered thresholds.
	cfg.Thresholds[rules.Parameters] = rules.Thresholds{Minor: 2, Major: 4}
	res, err = analyzer.AnalyzePaths([]string{filepath.Join(root, "params")}, analyzer.Options{
		Config: cfg,
		Root:   root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasViolation(res, rules.Parameters, rules.SeverityMinor, "Grouped") {
		t.Fatalf("Grouped should count as 3 params → minor: %+v", res.Violations)
	}
	if !hasViolation(res, rules.Parameters, rules.SeverityMajor, "Mixed") {
		t.Fatalf("Mixed should count as 5 params → major: %+v", res.Violations)
	}
}

func TestCommentsExcludedFromLOC(t *testing.T) {
	root := testdataRoot(t)
	src, err := os.ReadFile(filepath.Join(root, "comments", "comments.go"))
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "comments.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	// Use package-level helpers via AnalyzePaths and inspect methods_per_file / loc.
	cfg := config.Defaults()
	cfg.Thresholds[rules.LOCPerFile] = rules.Thresholds{Minor: 1, Major: 2}
	res, err := analyzer.AnalyzePaths([]string{filepath.Join(root, "comments")}, analyzer.Options{
		Config: cfg,
		Root:   root,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = file
	var loc int
	for _, v := range res.Violations {
		if v.RuleID == rules.LOCPerFile {
			loc = v.Actual
		}
	}
	// Even with low thresholds we mainly assert LOC is small (not inflated by comments/imports).
	if loc == 0 {
		// Might be below threshold; compute by analyzing with fail thresholds.
		cfg.Thresholds[rules.LOCPerFile] = rules.Thresholds{Minor: 0, Major: 0}
		res, err = analyzer.AnalyzePaths([]string{filepath.Join(root, "comments")}, analyzer.Options{
			Config: cfg,
			Root:   root,
		})
		if err != nil {
			t.Fatal(err)
		}
		for _, v := range res.Violations {
			if v.RuleID == rules.LOCPerFile {
				loc = v.Actual
			}
		}
	}
	if loc < 3 || loc > 20 {
		t.Fatalf("unexpected LOC=%d (comments/imports should be excluded)", loc)
	}
}

func TestMethodsPerFile(t *testing.T) {
	root := testdataRoot(t)
	cfg := config.Defaults()
	cfg.Thresholds[rules.MethodsPerFile] = rules.Thresholds{Minor: 3, Major: 5}
	res, err := analyzer.AnalyzePaths([]string{filepath.Join(root, "methods")}, analyzer.Options{
		Config: cfg,
		Root:   root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasViolation(res, rules.MethodsPerFile, rules.SeverityMajor, "") {
		t.Fatalf("expected methods_per_file major: %+v", res.Violations)
	}
}

func TestMalformedFileDoesNotPanic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.go")
	if err := os.WriteFile(path, []byte("package x\nfunc ({\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := analyzer.AnalyzePaths([]string{dir}, analyzer.Options{
		Config: config.Defaults(),
		Root:   dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.ParseErrors) == 0 {
		t.Fatal("expected parse error")
	}
}

func TestMultiViolations(t *testing.T) {
	res := analyzeDir(t, "multi")
	if !hasViolation(res, rules.ReturnsPerFunction, rules.SeverityMajor, "TooManyReturns") {
		t.Fatalf("missing returns major: %+v", res.Violations)
	}
	if !hasViolation(res, rules.Parameters, rules.SeverityMajor, "TooManyParams") {
		t.Fatalf("missing params major: %+v", res.Violations)
	}
}

func TestFunctionLengthThresholds(t *testing.T) {
	res := analyzeDir(t, "minor")
	if !hasViolation(res, rules.FunctionLength, rules.SeverityMinor, "MildLength") {
		t.Fatalf("expected mild function length: %+v", res.Violations)
	}
}
