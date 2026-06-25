package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/unbound-force/gaze/internal/crap"
)

// TestCrapWithExternalAnalyzer verifies that runCrap correctly uses
// an external analyzer binary via the --analyzer flag. The fake
// analyzer provides canned complexity and coverage data:
//
//   - add:      complexity=2, coverage=90%
//   - multiply: complexity=3, coverage=60%
//   - divide:   complexity=5, coverage=0%
//
// CRAP scores are computed from these values using the standard
// formula: CRAP(c,cov) = c² × (1 - cov)³ + c.
func TestCrapWithExternalAnalyzer(t *testing.T) {
	var stdout, stderr bytes.Buffer

	// Use a temp directory as the "module root" — the external
	// analyzer doesn't need a real Go module.
	moduleDir := t.TempDir()

	// Create a minimal go.mod so crap.Analyze can resolve patterns.
	// The external providers bypass Go tooling, but the framework
	// still validates the module directory.
	goMod := filepath.Join(moduleDir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "json",
		opts:         opts,
		moduleDir:    moduleDir,
		analyzerFlag: fakeBinaryPath,
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err != nil {
		t.Fatalf("runCrap with external analyzer: %v\nstderr: %s", err, stderr.String())
	}

	// Parse the JSON output to verify CRAP scores.
	var report crap.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("parsing JSON output: %v\nraw: %s", err, stdout.String())
	}

	if len(report.Scores) == 0 {
		t.Fatal("no scores in report")
	}

	// Build a map of function name → CRAP score for verification.
	scores := make(map[string]float64)
	for _, s := range report.Scores {
		scores[s.Function] = s.CRAP
	}

	// Verify CRAP scores match expected values from the fake data.
	// CRAP formula: c² × (1 - cov)³ + c
	//
	// add:      2² × (1 - 0.90)³ + 2 = 4 × 0.001 + 2 = 2.004
	// multiply: 3² × (1 - 0.60)³ + 3 = 9 × 0.064 + 3 = 3.576
	// divide:   5² × (1 - 0.00)³ + 5 = 25 × 1.0 + 5 = 30.0
	wantApprox := map[string]struct {
		min, max float64
	}{
		"add":      {1.5, 3.0},
		"multiply": {3.0, 4.5},
		"divide":   {29.0, 31.0},
	}

	for name, want := range wantApprox {
		got, ok := scores[name]
		if !ok {
			t.Errorf("function %q not found in scores", name)
			continue
		}
		if got < want.min || got > want.max {
			t.Errorf("%s CRAP = %g, want in [%g, %g]", name, got, want.min, want.max)
		}
	}

	// Verify the stderr mentions the external analyzer.
	stderrStr := stderr.String()
	if !bytes.Contains([]byte(stderrStr), []byte("fake-analyzer")) {
		t.Errorf("stderr should mention analyzer name, got: %s", stderrStr)
	}
}

// TestCrapWithExternalAnalyzer_NotFound verifies that a nonexistent
// analyzer binary produces a clear error.
func TestCrapWithExternalAnalyzer_NotFound(t *testing.T) {
	var stdout, stderr bytes.Buffer

	opts := crap.DefaultOptions()
	opts.Stderr = &stderr

	err := runCrap(crapParams{
		patterns:     []string{"./..."},
		format:       "text",
		opts:         opts,
		moduleDir:    t.TempDir(),
		analyzerFlag: "/nonexistent/analyzer",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for nonexistent analyzer")
	}
}

// TestQualityWithExternalAnalyzer_Rejected verifies that --analyzer
// on gaze quality produces an informative error (deferred per D12).
func TestQualityWithExternalAnalyzer_Rejected(t *testing.T) {
	var stdout, stderr bytes.Buffer

	err := runQuality(qualityParams{
		patterns:     []string{"./..."},
		format:       "text",
		analyzerFlag: "some-analyzer",
		stdout:       &stdout,
		stderr:       &stderr,
	})
	if err == nil {
		t.Fatal("expected error for --analyzer on quality")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("not yet supported")) {
		t.Errorf("error should mention 'not yet supported', got: %s", err.Error())
	}
}
