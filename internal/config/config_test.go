package config

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDefaultConfig_Thresholds(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Classification.Thresholds.Contractual != 80 {
		t.Errorf("default contractual threshold = %d, want 80",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 50 {
		t.Errorf("default incidental threshold = %d, want 50",
			cfg.Classification.Thresholds.Incidental)
	}
}

func TestDefaultConfig_ExcludeList(t *testing.T) {
	cfg := DefaultConfig()

	excludes := cfg.Classification.DocScan.Exclude
	expected := []string{
		"vendor/**", "node_modules/**", ".git/**", "testdata/**",
		"CHANGELOG.md", "CONTRIBUTING.md", "CODE_OF_CONDUCT.md",
		"LICENSE", "LICENSE.md",
	}

	if len(excludes) != len(expected) {
		t.Fatalf("default exclude count = %d, want %d",
			len(excludes), len(expected))
	}
	for i, e := range expected {
		if excludes[i] != e {
			t.Errorf("exclude[%d] = %q, want %q", i, excludes[i], e)
		}
	}
}

func TestDefaultConfig_Timeout(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Classification.DocScan.Timeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s",
			cfg.Classification.DocScan.Timeout)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load("testdata/nonexistent.yaml")
	if err != nil {
		t.Fatalf("Load(nonexistent) error: %v", err)
	}

	// Should return defaults.
	if cfg.Classification.Thresholds.Contractual != 80 {
		t.Errorf("missing file: contractual = %d, want 80",
			cfg.Classification.Thresholds.Contractual)
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "valid.yaml"))
	if err != nil {
		t.Fatalf("Load(valid) error: %v", err)
	}

	if cfg.Classification.Thresholds.Contractual != 85 {
		t.Errorf("contractual = %d, want 85",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 45 {
		t.Errorf("incidental = %d, want 45",
			cfg.Classification.Thresholds.Incidental)
	}
	if cfg.Classification.DocScan.Timeout != 15*time.Second {
		t.Errorf("timeout = %v, want 15s",
			cfg.Classification.DocScan.Timeout)
	}
	if len(cfg.Classification.DocScan.Exclude) != 2 {
		t.Errorf("exclude count = %d, want 2",
			len(cfg.Classification.DocScan.Exclude))
	}
}

func TestLoad_EmptyConfig(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "empty.yaml"))
	if err != nil {
		t.Fatalf("Load(empty) error: %v", err)
	}

	// Empty YAML should keep defaults since we unmarshal into
	// a pre-populated struct.
	if cfg.Classification.Thresholds.Contractual != 80 {
		t.Errorf("empty config: contractual = %d, want 80",
			cfg.Classification.Thresholds.Contractual)
	}
}

func TestLoad_CustomThresholds(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "custom-thresholds.yaml"))
	if err != nil {
		t.Fatalf("Load(custom-thresholds) error: %v", err)
	}

	if cfg.Classification.Thresholds.Contractual != 90 {
		t.Errorf("contractual = %d, want 90",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 40 {
		t.Errorf("incidental = %d, want 40",
			cfg.Classification.Thresholds.Incidental)
	}
}

func TestLoad_IncludeOverride(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "include-override.yaml"))
	if err != nil {
		t.Fatalf("Load(include-override) error: %v", err)
	}

	includes := cfg.Classification.DocScan.Include
	if len(includes) != 2 {
		t.Fatalf("include count = %d, want 2", len(includes))
	}
	if includes[0] != "docs/**" {
		t.Errorf("include[0] = %q, want %q", includes[0], "docs/**")
	}
	if includes[1] != "README.md" {
		t.Errorf("include[1] = %q, want %q", includes[1], "README.md")
	}
}

func TestBaselineConfig_Defaults(t *testing.T) {
	// Loading a nonexistent file should return defaults.
	cfg, err := Load("testdata/nonexistent.yaml")
	if err != nil {
		t.Fatalf("Load(nonexistent) error: %v", err)
	}

	if cfg.Baseline.File != ".gaze/baseline.json" {
		t.Errorf("default baseline.file = %q, want %q",
			cfg.Baseline.File, ".gaze/baseline.json")
	}
	if cfg.Baseline.Epsilon != 0.5 {
		t.Errorf("default baseline.epsilon = %g, want 0.5",
			cfg.Baseline.Epsilon)
	}
	if cfg.Baseline.NewFunctionThreshold != 30 {
		t.Errorf("default baseline.new_function_threshold = %g, want 30",
			cfg.Baseline.NewFunctionThreshold)
	}
}

func TestBaselineConfig_Override(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "baseline-config.yaml"))
	if err != nil {
		t.Fatalf("Load(baseline-config) error: %v", err)
	}

	if cfg.Baseline.File != "custom/baseline.json" {
		t.Errorf("baseline.file = %q, want %q",
			cfg.Baseline.File, "custom/baseline.json")
	}
	if cfg.Baseline.Epsilon != 1.0 {
		t.Errorf("baseline.epsilon = %g, want 1.0",
			cfg.Baseline.Epsilon)
	}
	if cfg.Baseline.NewFunctionThreshold != 20 {
		t.Errorf("baseline.new_function_threshold = %g, want 20",
			cfg.Baseline.NewFunctionThreshold)
	}
}

func TestBaselineConfig_InvalidEpsilon(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid-baseline.yaml"))
	if err == nil {
		t.Fatal("expected error for negative epsilon, got nil")
	}

	want := "baseline.epsilon must be >= 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestBaselineConfig_InvalidThreshold(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid-baseline-threshold.yaml"))
	if err == nil {
		t.Fatal("expected error for zero threshold, got nil")
	}

	want := "baseline.new_function_threshold must be > 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_ContractualOutOfRange(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "out-of-range-contractual.yaml"))
	if err == nil {
		t.Fatal("expected error for contractual=500, got nil")
	}

	want := "classification.thresholds.contractual must be in [1, 99], got 500"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_IncidentalOutOfRange(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "out-of-range-incidental.yaml"))
	if err == nil {
		t.Fatal("expected error for incidental=200, got nil")
	}

	want := "classification.thresholds.incidental must be in [1, 99], got 200"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_ZeroContractual(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "zero-contractual.yaml"))
	if err == nil {
		t.Fatal("expected error for contractual=0, got nil")
	}

	want := "classification.thresholds.contractual must be in [1, 99], got 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_NegativeIncidental(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "negative-incidental.yaml"))
	if err == nil {
		t.Fatal("expected error for incidental=-10, got nil")
	}

	want := "classification.thresholds.incidental must be in [1, 99], got -10"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_InvertedThresholds(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "inverted-thresholds.yaml"))
	if err == nil {
		t.Fatal("expected error for inverted thresholds, got nil")
	}

	want := "classification.thresholds.contractual (40) must be greater than incidental (60)"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_EqualThresholds(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "equal-thresholds.yaml"))
	if err == nil {
		t.Fatal("expected error for equal thresholds, got nil")
	}

	want := "classification.thresholds.contractual (50) must be greater than incidental (50)"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_BoundaryValid(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "boundary-thresholds.yaml"))
	if err != nil {
		t.Fatalf("Load(boundary-thresholds) error: %v", err)
	}

	if cfg.Classification.Thresholds.Contractual != 99 {
		t.Errorf("contractual = %d, want 99",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 1 {
		t.Errorf("incidental = %d, want 1",
			cfg.Classification.Thresholds.Incidental)
	}
}

func TestLoad_AdjacentValid(t *testing.T) {
	cfg, err := Load(filepath.Join("testdata", "adjacent-thresholds.yaml"))
	if err != nil {
		t.Fatalf("Load(adjacent-thresholds) error: %v", err)
	}

	if cfg.Classification.Thresholds.Contractual != 51 {
		t.Errorf("contractual = %d, want 51",
			cfg.Classification.Thresholds.Contractual)
	}
	if cfg.Classification.Thresholds.Incidental != 50 {
		t.Errorf("incidental = %d, want 50",
			cfg.Classification.Thresholds.Incidental)
	}
}

func TestLoad_NegativeContractual(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "negative-contractual.yaml"))
	if err == nil {
		t.Fatal("expected error for contractual=-10, got nil")
	}

	want := "classification.thresholds.contractual must be in [1, 99], got -10"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

// --- Issue #164: GazeCRAP new-function threshold tests ---

func TestSC004_ConfigDefault_GazeCRAPThreshold(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Baseline.NewFunctionGazeCRAPThreshold != 30 {
		t.Errorf("default NewFunctionGazeCRAPThreshold = %g, want 30",
			cfg.Baseline.NewFunctionGazeCRAPThreshold)
	}
}

func TestSC007_ConfigValidation_ZeroGazeCRAPThreshold(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid-baseline-gaze-threshold-zero.yaml"))
	if err == nil {
		t.Fatal("expected error for zero gaze_crap threshold, got nil")
	}

	want := "baseline.new_function_gaze_crap_threshold must be > 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestSC007_ConfigValidation_NegativeGazeCRAPThreshold(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "invalid-baseline-gaze-threshold-negative.yaml"))
	if err == nil {
		t.Fatal("expected error for negative gaze_crap threshold, got nil")
	}

	want := "baseline.new_function_gaze_crap_threshold must be > 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}

func TestLoad_ZeroIncidental(t *testing.T) {
	_, err := Load(filepath.Join("testdata", "zero-incidental.yaml"))
	if err == nil {
		t.Fatal("expected error for incidental=0, got nil")
	}

	want := "classification.thresholds.incidental must be in [1, 99], got 0"
	if !strings.Contains(err.Error(), want) {
		t.Errorf("error = %q, want to contain %q", err, want)
	}
}
