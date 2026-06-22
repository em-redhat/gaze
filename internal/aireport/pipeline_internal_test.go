package aireport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/unbound-force/gaze/internal/crap"
)

// fakeSteps returns a pipelineStepFuncs with all four steps returning
// synthetic success results. Individual steps can be overridden after.
func fakeSteps() pipelineStepFuncs {
	return pipelineStepFuncs{
		crapStep: func(_ []string, _ string, _ string, _ io.Writer, _ func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error) {
			return &crapStepResult{
				JSON:           json.RawMessage(`{"crap":"ok"}`),
				CRAPload:       5,
				GazeCRAPload:   intPtr(3),
				TotalFunctions: 20,
			}, nil
		},
		qualityStep: func(_ []string, _ string, _ io.Writer) (*qualityStepResult, error) {
			return &qualityStepResult{
				JSON:                json.RawMessage(`{"quality":"ok"}`),
				AvgContractCoverage: 85,
			}, nil
		},
		classifyStep: func(_ []string, _ string) (*classifyStepResult, error) {
			return &classifyStepResult{
				JSON:        json.RawMessage(`{"classify":"ok"}`),
				Contractual: 10,
				Ambiguous:   3,
				Incidental:  1,
			}, nil
		},
		docscanStep: func(_ string) (json.RawMessage, error) {
			return json.RawMessage(`{"docscan":"ok"}`), nil
		},
	}
}

func TestRunProductionPipeline_AllStepsSucceed(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// All sections should be populated.
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP")
	}
	if payload.Quality == nil {
		t.Error("expected non-nil Quality")
	}
	if payload.Classify == nil {
		t.Error("expected non-nil Classify")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan")
	}

	// No errors should be set.
	if payload.Errors.CRAP != nil {
		t.Errorf("expected nil CRAP error, got: %v", *payload.Errors.CRAP)
	}
	if payload.Errors.Quality != nil {
		t.Errorf("expected nil Quality error, got: %v", *payload.Errors.Quality)
	}
	if payload.Errors.Classify != nil {
		t.Errorf("expected nil Classify error, got: %v", *payload.Errors.Classify)
	}
	if payload.Errors.Docscan != nil {
		t.Errorf("expected nil Docscan error, got: %v", *payload.Errors.Docscan)
	}

	// SSA should not be degraded when all steps succeed.
	if payload.Summary.SSADegraded {
		t.Error("expected SSADegraded=false when all steps succeed")
	}
	if len(payload.Summary.SSADegradedPackages) != 0 {
		t.Errorf("expected empty SSADegradedPackages, got %v", payload.Summary.SSADegradedPackages)
	}

	// Classification counts should be propagated from classify step.
	if payload.Summary.Contractual != 10 {
		t.Errorf("expected Contractual=10, got %d", payload.Summary.Contractual)
	}
	if payload.Summary.Ambiguous != 3 {
		t.Errorf("expected Ambiguous=3, got %d", payload.Summary.Ambiguous)
	}
	if payload.Summary.Incidental != 1 {
		t.Errorf("expected Incidental=1, got %d", payload.Summary.Incidental)
	}
}

func TestRunProductionPipeline_CRAPStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error) {
		return nil, fmt.Errorf("crap analysis failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	// CRAP error captured.
	if payload.Errors.CRAP == nil {
		t.Fatal("expected non-nil CRAP error")
	}
	if payload.CRAP != nil {
		t.Error("expected nil CRAP payload when step failed")
	}

	// Other sections still populated.
	if payload.Quality == nil {
		t.Error("expected non-nil Quality despite CRAP failure")
	}
	if payload.Classify == nil {
		t.Error("expected non-nil Classify despite CRAP failure")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite CRAP failure")
	}
}

func TestRunProductionPipeline_QualityStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.qualityStep = func(_ []string, _ string, _ io.Writer) (*qualityStepResult, error) {
		return nil, fmt.Errorf("quality analysis failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Quality == nil {
		t.Fatal("expected non-nil Quality error")
	}
	if payload.Quality != nil {
		t.Error("expected nil Quality payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Quality failure")
	}
}

func TestRunProductionPipeline_ClassifyStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.classifyStep = func(_ []string, _ string) (*classifyStepResult, error) {
		return nil, fmt.Errorf("classify failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Classify == nil {
		t.Fatal("expected non-nil Classify error")
	}
	if payload.Classify != nil {
		t.Error("expected nil Classify payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Classify failure")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite Classify failure")
	}
}

func TestRunProductionPipeline_DocscanStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.docscanStep = func(_ string) (json.RawMessage, error) {
		return nil, fmt.Errorf("docscan failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Docscan == nil {
		t.Fatal("expected non-nil Docscan error")
	}
	if payload.Docscan != nil {
		t.Error("expected nil Docscan payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Docscan failure")
	}
	if payload.Quality == nil {
		t.Error("expected non-nil Quality despite Docscan failure")
	}
}

func TestRunProductionPipeline_MultipleStepsFail(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error) {
		return nil, fmt.Errorf("crap failed")
	}
	steps.qualityStep = func(_ []string, _ string, _ io.Writer) (*qualityStepResult, error) {
		return nil, fmt.Errorf("quality failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failures, got: %v", err)
	}

	// Both errors captured.
	if payload.Errors.CRAP == nil {
		t.Error("expected non-nil CRAP error")
	}
	if payload.Errors.Quality == nil {
		t.Error("expected non-nil Quality error")
	}

	// Other sections still populated.
	if payload.Classify == nil {
		t.Error("expected non-nil Classify despite CRAP+Quality failures")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite CRAP+Quality failures")
	}
}

func TestRunProductionPipeline_EmptyPatterns(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	// Track whether any step was called.
	called := false
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error) {
		called = true
		return nil, nil
	}

	_, err := runProductionPipeline([]string{}, "/tmp", "", &stderr, steps)
	if err == nil {
		t.Fatal("expected error for empty patterns")
	}
	if called {
		t.Error("step functions should not be called when patterns are empty")
	}
}

// TestRunProductionPipeline_GazeCRAPloadFlowsThroughPipeline verifies that
// when the crapStep returns a non-zero GazeCRAPload, it propagates to
// payload.Summary.GazeCRAPload. Callback wiring (non-nil callback passed
// to crapStep) is verified by the integration test
// TestSC002_GazeCRAPloadMatchBetweenCrapAndReport in cmd/gaze/main_test.go.
func TestRunProductionPipeline_GazeCRAPloadFlowsThroughPipeline(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	// Override crapStep to return a known GazeCRAPload value.
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error) {
		return &crapStepResult{
			JSON:           json.RawMessage(`{"crap":"ok"}`),
			CRAPload:       2,
			GazeCRAPload:   intPtr(7),
			TotalFunctions: 15,
		}, nil
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if payload.Summary.GazeCRAPload == nil || *payload.Summary.GazeCRAPload != 7 {
		t.Errorf("expected GazeCRAPload 7, got %v", payload.Summary.GazeCRAPload)
	}
}

func TestRunProductionPipeline_SummaryFields(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", &stderr, steps)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if payload.Summary.CRAPload != 5 {
		t.Errorf("expected CRAPload 5, got %d", payload.Summary.CRAPload)
	}
	if payload.Summary.GazeCRAPload == nil || *payload.Summary.GazeCRAPload != 3 {
		t.Errorf("expected GazeCRAPload 3, got %v", payload.Summary.GazeCRAPload)
	}
	if payload.Summary.AvgContractCoverage != 85 {
		t.Errorf("expected AvgContractCoverage 85, got %d", payload.Summary.AvgContractCoverage)
	}
}

// TestResolveModulePackages_EmptyDir verifies that resolveModulePackages
// with an empty moduleDir uses the current working directory and
// successfully loads module packages (when run from a Go module root).
func TestResolveModulePackages_EmptyDir(t *testing.T) {
	pkgs := resolveModulePackages("")
	// When run from the gaze module root, this should return packages.
	// In CI, the working directory is the repo root.
	if pkgs == nil {
		t.Skip("not running from a Go module root — skipping")
	}
	if len(pkgs) == 0 {
		t.Error("expected non-empty package list from module root")
	}
}

// TestResolveModulePackages_InvalidDir verifies that resolveModulePackages
// returns nil for a directory that is not a Go module.
func TestResolveModulePackages_InvalidDir(t *testing.T) {
	pkgs := resolveModulePackages(t.TempDir())
	if pkgs != nil {
		t.Errorf("expected nil for non-module directory, got %d packages", len(pkgs))
	}
}
