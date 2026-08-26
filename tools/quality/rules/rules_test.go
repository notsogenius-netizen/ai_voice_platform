package rules_test

import (
	"testing"

	"github.com/sourabh/ai-voice-platform/tools/quality/rules"
)

func TestClassifyBoundaries(t *testing.T) {
	thr := rules.Thresholds{Minor: 5, Major: 10}
	cases := []struct {
		actual int
		want   rules.Severity
	}{
		{0, rules.SeverityPass},
		{5, rules.SeverityPass},
		{6, rules.SeverityMinor},
		{10, rules.SeverityMinor},
		{11, rules.SeverityMajor},
	}
	for _, tc := range cases {
		got := thr.Classify(tc.actual)
		if got != tc.want {
			t.Fatalf("Classify(%d)=%s want %s", tc.actual, got, tc.want)
		}
	}
}

func TestSeverityRank(t *testing.T) {
	if rules.SeverityPass.Rank() >= rules.SeverityMinor.Rank() {
		t.Fatal("pass should rank below minor")
	}
	if rules.SeverityMinor.Rank() >= rules.SeverityMajor.Rank() {
		t.Fatal("minor should rank below major")
	}
}

func TestDefaultThresholds(t *testing.T) {
	d := rules.DefaultThresholds()
	for _, id := range rules.AllRuleIDs {
		thr, ok := d[id]
		if !ok {
			t.Fatalf("missing default for %s", id)
		}
		if thr.Minor > thr.Major {
			t.Fatalf("%s: minor > major", id)
		}
	}
}
