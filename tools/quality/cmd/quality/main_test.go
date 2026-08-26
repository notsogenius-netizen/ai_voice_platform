package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunExitCodes(t *testing.T) {
	dir := t.TempDir()
	// Clean passing file.
	pass := "package p\n\nfunc OK() int { return 1 }\n"
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte(pass), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"--root", dir, "--fail-on", "major", dir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("pass exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}

	bad := "package p\n\nfunc Bad(a,b,c,d,e,f,g,h,i int) int { return a }\n"
	if err := os.WriteFile(filepath.Join(dir, "bad.go"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"--root", dir, "--fail-on", "major", dir}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("fail exit=%d stderr=%s stdout=%s", code, stderr.String(), stdout.String())
	}
}

func TestRunJSONFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\nfunc OK() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "report.json")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--root", dir,
		"--format", "json",
		"--output", outFile,
		"--quiet",
		dir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"result"`)) {
		t.Fatalf("json missing result: %s", data)
	}
}

func TestRunSARIFFormat(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ok.go"), []byte("package p\nfunc OK() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	outFile := filepath.Join(dir, "report.sarif")
	var stdout, stderr bytes.Buffer
	code := run([]string{
		"--root", dir,
		"--format", "sarif",
		"--output", outFile,
		"--quiet",
		dir,
	}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit=%d stderr=%s", code, stderr.String())
	}
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"version": "2.1.0"`)) {
		t.Fatalf("sarif missing version: %s", data)
	}
}

func TestRunBadFlag(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--fail-on", "nope"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("exit=%d", code)
	}
}
