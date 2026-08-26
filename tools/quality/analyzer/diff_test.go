package analyzer

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sourabh/ai-voice-platform/tools/quality/config"
	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func TestMarkNewViolations(t *testing.T) {
	base := metricsIndex{
		{RuleID: rules.Complexity, File: "a.go", Function: "F"}: 8,
		{RuleID: rules.MethodsPerFile, File: "a.go"}:            28,
	}
	cases := []struct {
		name string
		v    Violation
		want bool
	}{
		{"same complexity", Violation{RuleID: rules.Complexity, File: "a.go", Function: "F", Actual: 8}, false},
		{"worse complexity", Violation{RuleID: rules.Complexity, File: "a.go", Function: "F", Actual: 12}, true},
		{"same methods", Violation{RuleID: rules.MethodsPerFile, File: "a.go", Actual: 28}, false},
		{"more methods", Violation{RuleID: rules.MethodsPerFile, File: "a.go", Actual: 29}, true},
		{"brand new", Violation{RuleID: rules.Parameters, File: "b.go", Function: "New", Actual: 9}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := Result{Violations: []Violation{tc.v}}
			markNewViolations(&r, base)
			if r.Violations[0].IsNew != tc.want {
				t.Fatalf("IsNew=%v want %v", r.Violations[0].IsNew, tc.want)
			}
		})
	}
}

func TestAnalyzeWithBaseGit(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	run("init", "--template=", "-b", "main")

	src := "package p\n\nfunc Mild(x int) int {\n" +
		"\tif x == 0 { return 0 }\n" +
		"\tif x == 1 { return 1 }\n" +
		"\tif x == 2 { return 2 }\n" +
		"\tif x == 3 { return 3 }\n" +
		"\tif x == 4 { return 4 }\n" +
		"\treturn 5\n}\n"
	path := filepath.Join(root, "svc.go")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "svc.go")
	run("commit", "-m", "base")

	run("checkout", "-b", "pr")
	worse := "package p\n\nfunc Mild(x int) int {\n" +
		"\tif x == 0 { return 0 }\n" +
		"\tif x == 1 { return 1 }\n" +
		"\tif x == 2 { return 2 }\n" +
		"\tif x == 3 { return 3 }\n" +
		"\tif x == 4 { return 4 }\n" +
		"\tif x == 5 { return 5 }\n" +
		"\tif x == 6 { return 6 }\n" +
		"\treturn 7\n}\n"
	if err := os.WriteFile(path, []byte(worse), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "svc.go")
	run("commit", "-m", "worse")

	cfg := config.Defaults()
	res, err := AnalyzeWithBase(CompareOptions{
		Config: cfg,
		Root:   root,
		Base:   "main",
		Paths:  []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	foundNewMajor := false
	for _, v := range res.Violations {
		if v.RuleID == rules.ReturnsPerFunction && v.IsNew && v.Severity == rules.SeverityMajor {
			foundNewMajor = true
		}
	}
	if !foundNewMajor {
		t.Fatalf("expected new major returns violation: %+v", res.Violations)
	}
}

func TestAnalyzeWithBaseUnchangedNotNew(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@e",
			"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@e",
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v (%s)", args, err, out)
		}
	}
	run("init", "--template=", "-b", "main")
	src := "package p\n\nfunc Mild(x int) int {\n" +
		"\tif x == 0 { return 0 }\n" +
		"\tif x == 1 { return 1 }\n" +
		"\tif x == 2 { return 2 }\n" +
		"\tif x == 3 { return 3 }\n" +
		"\tif x == 4 { return 4 }\n" +
		"\treturn 5\n}\n"
	if err := os.WriteFile(filepath.Join(root, "svc.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "svc.go")
	run("commit", "-m", "base")
	run("checkout", "-b", "pr")
	// Touch an unrelated file so the PR has a change, but svc.go metrics stay the same.
	if err := os.WriteFile(filepath.Join(root, "readme.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "readme.txt")
	run("commit", "-m", "docs")

	res, err := AnalyzeWithBase(CompareOptions{
		Config: config.Defaults(),
		Root:   root,
		Base:   "main",
		Paths:  []string{root},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range res.Violations {
		if v.File == "svc.go" && v.IsNew {
			t.Fatalf("unchanged svc.go violation should not be new: %+v", v)
		}
	}
}

func TestCyclomaticAndNesting(t *testing.T) {
	src := `package p
func F(a, b bool, xs []int) {
	if a && b {
		for _, x := range xs {
			switch x {
			case 1:
				_ = x
			default:
				_ = x
			}
		}
	}
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	fd := file.Decls[0].(*ast.FuncDecl)
	c := cyclomaticComplexity(fd)
	// base 1 + if + && + range + case + default = 6
	if c < 5 {
		t.Fatalf("complexity=%d, want >= 5", c)
	}
	d := maxNestingDepth(fd)
	if d < 3 {
		t.Fatalf("nesting=%d, want >= 3", d)
	}
}

func TestCountParamsGrouped(t *testing.T) {
	src := `package p
func Grouped(a, b, c int) {}
func Mixed(a, b int, c string) {}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "x.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	if got := countParams(file.Decls[0].(*ast.FuncDecl)); got != 3 {
		t.Fatalf("Grouped params=%d", got)
	}
	if got := countParams(file.Decls[1].(*ast.FuncDecl)); got != 3 {
		t.Fatalf("Mixed params=%d", got)
	}
}
