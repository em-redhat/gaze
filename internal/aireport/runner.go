package aireport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/unbound-force/gaze/internal/crap"
)

// RunnerOptions configures the report pipeline runner.
type RunnerOptions struct {
	// Patterns is the package pattern(s) to analyze (e.g., "./...").
	Patterns []string

	// ModuleDir is the root of the Go module being analyzed.
	ModuleDir string

	// Adapter is the AI adapter to use for formatting.
	// Required when Format is "text"; ignored when Format is "json".
	Adapter AIAdapter

	// AdapterCfg holds adapter-specific configuration (timeout, model, etc.).
	AdapterCfg AdapterConfig

	// SystemPrompt is the formatting instructions for the AI adapter.
	// Loaded from .opencode/agents/gaze-reporter.md (stripped of YAML
	// frontmatter) or the embedded default prompt.
	SystemPrompt string

	// Format is "text" (default) or "json".
	Format string

	// Stdout receives the formatted report (text mode) or combined JSON
	// payload (json mode).
	Stdout io.Writer

	// Stderr receives progress signals, threshold summaries, and warnings.
	Stderr io.Writer

	// Thresholds holds the CI gate configuration. EvaluateThresholds is
	// called after the pipeline completes. Results are printed to Stderr.
	// When any threshold fails, Run returns a non-nil error.
	Thresholds ThresholdConfig

	// StepSummaryPath is the value of $GITHUB_STEP_SUMMARY, if set.
	// Empty means Step Summary output is disabled.
	StepSummaryPath string

	// CoverProfile is the path to a pre-generated Go coverage profile.
	// When non-empty, the CRAP analysis step uses this file directly
	// instead of spawning go test internally (FR-001, FR-002).
	// Empty string means "generate internally" (default behavior, FR-003).
	CoverProfile string

	// AnalyzeFunc overrides the analysis pipeline for testing.
	// When nil, the production pipeline is called.
	AnalyzeFunc func(patterns []string, moduleDir string) (*ReportPayload, error)
}

// validateRunnerOpts applies defaults and validates required fields.
// In text mode, verifies the adapter is non-nil and the binary is on PATH.
func validateRunnerOpts(opts *RunnerOptions) error {
	if opts.Format != "text" && opts.Format != "json" {
		opts.Format = "text"
	}
	if opts.Format == "text" && opts.Adapter == nil {
		return fmt.Errorf("text format requires a non-nil Adapter")
	}
	if opts.Format == "text" {
		if err := ValidateAdapterBinary(opts.Adapter); err != nil {
			return err
		}
	}
	return nil
}

// runJSONPath writes the payload as formatted JSON to stdout and
// evaluates thresholds.
func runJSONPath(payload *ReportPayload, opts RunnerOptions) error {
	enc := json.NewEncoder(opts.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(payload); err != nil {
		return err
	}
	return evaluateAndPrintThresholds(opts.Thresholds, payload, opts.Stderr)
}

// runTextPath invokes the AI adapter with the payload, writes the
// formatted report to stdout and optionally to GITHUB_STEP_SUMMARY,
// and evaluates thresholds.
func runTextPath(payload *ReportPayload, opts RunnerOptions) error {
	_, _ = fmt.Fprintln(opts.Stderr, "Formatting report...")

	payloadBytes, err := payload.CompactForAI()
	if err != nil {
		return fmt.Errorf("compacting payload for AI adapter: %w", err)
	}
	_, _ = fmt.Fprintf(opts.Stderr, "Payload: %d bytes (compact)\n", len(payloadBytes))

	timeout := opts.AdapterCfg.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	formatted, err := opts.Adapter.Format(ctx, opts.SystemPrompt, bytes.NewReader(payloadBytes))
	if err != nil {
		return fmt.Errorf("AI formatting failed: %w", err)
	}
	if strings.TrimSpace(formatted) == "" {
		return fmt.Errorf("AI adapter returned empty output (FR-016): ensure the adapter is configured and working correctly")
	}

	if _, err := fmt.Fprint(opts.Stdout, formatted); err != nil {
		return fmt.Errorf("writing report to stdout: %w", err)
	}

	if opts.StepSummaryPath != "" {
		_, _ = fmt.Fprintln(opts.Stderr, "Writing Step Summary...")
		WriteStepSummary(opts.StepSummaryPath, formatted, opts.Stderr)
	}

	return evaluateAndPrintThresholds(opts.Thresholds, payload, opts.Stderr)
}

// Run executes the report pipeline according to opts.
//
// In --format=json mode it assembles a ReportPayload from the four analysis
// steps (CRAP, Quality, Classification, Docscan) and writes the combined
// JSON to opts.Stdout. Each step's failure is recorded in PayloadErrors and
// the remaining steps still run (partial-failure mode).
//
// In --format=text mode (default) it additionally calls opts.Adapter.Format
// with the system prompt and JSON payload to produce a markdown report, then
// writes that to opts.Stdout and optionally to $GITHUB_STEP_SUMMARY.
//
// Returns an error when:
//   - The AI adapter binary is not on PATH (FR-012, text mode only; checked before analysis).
//   - No packages match the patterns (FR-013).
//   - The AI adapter returns empty output (FR-016, text mode only).
//   - The AI adapter invocation fails (text mode only).
func Run(opts RunnerOptions) error {
	if err := validateRunnerOpts(&opts); err != nil {
		return err
	}

	analyzeFunc := opts.AnalyzeFunc
	if analyzeFunc == nil {
		analyzeFunc = func(patterns []string, moduleDir string) (*ReportPayload, error) {
			return runProductionPipeline(patterns, moduleDir, opts.CoverProfile, opts.Stderr, pipelineStepFuncs{})
		}
	}

	payload, err := analyzeFunc(opts.Patterns, opts.ModuleDir)
	if err != nil {
		return err
	}

	if opts.Format == "json" {
		return runJSONPath(payload, opts)
	}
	return runTextPath(payload, opts)
}

// evaluateAndPrintThresholds evaluates cfg thresholds against payload and
// prints results to stderr. Returns a non-nil error if any threshold failed.
// When cfg has no thresholds set, it is a no-op.
func evaluateAndPrintThresholds(cfg ThresholdConfig, payload *ReportPayload, stderr io.Writer) error {
	results, allPassed := EvaluateThresholds(cfg, payload)
	for _, r := range results {
		status := "PASS"
		if !r.Passed {
			status = "FAIL"
		}
		_, _ = fmt.Fprintf(stderr, "%s: %d/%d (%s)\n", r.Name, r.Actual, r.Limit, status)
	}
	if !allPassed {
		return fmt.Errorf("one or more quality thresholds failed")
	}
	return nil
}

// errString converts an error to a *string for PayloadErrors.
func errString(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}

// pipelineStepFuncs holds injectable step function references for
// runProductionPipeline. When a field is nil, the real step function
// is used. This enables unit testing of the orchestration logic
// (partial failures, error capture, payload assembly) without running
// real analysis.
type pipelineStepFuncs struct {
	crapStep     func([]string, string, string, io.Writer, func(string, string) (crap.ContractCoverageInfo, bool)) (*crapStepResult, error)
	qualityStep  func([]string, string, io.Writer) (*qualityStepResult, error)
	classifyStep func([]string, string) (*classifyStepResult, error)
	docscanStep  func(string) (json.RawMessage, error)
}

// runProductionPipeline runs the four-step analysis pipeline and returns
// a ReportPayload. Each step failure is recorded as a non-nil PayloadErrors
// field; remaining steps still run.
//
// coverProfile is forwarded to runCRAPStep (FR-001, FR-002). Empty string
// means generate coverage internally (FR-003).
//
// The steps parameter allows injection of fake step functions for testing.
// Pass pipelineStepFuncs{} (zero value) for production behavior.
func runProductionPipeline(patterns []string, moduleDir string, coverProfile string, stderr io.Writer, steps pipelineStepFuncs) (*ReportPayload, error) {
	// Default nil step functions to real implementations.
	if steps.crapStep == nil {
		steps.crapStep = runCRAPStep
	}
	if steps.qualityStep == nil {
		steps.qualityStep = runQualityStep
	}
	if steps.classifyStep == nil {
		steps.classifyStep = runClassifyStep
	}
	if steps.docscanStep == nil {
		steps.docscanStep = runDocscanStep
	}

	payload := &ReportPayload{}

	// Validate patterns are non-empty.
	if len(patterns) == 0 {
		return nil, fmt.Errorf("no package patterns specified")
	}

	// Build contract coverage callback for GazeCRAP scoring (spec 022).
	// This runs the quality pipeline per-package to build a lookup
	// closure. Best-effort: returns nil if all packages fail.
	ccFunc, degradedPkgs := crap.BuildContractCoverageFunc(patterns, moduleDir, stderr)
	if len(degradedPkgs) > 0 {
		payload.Summary.SSADegraded = true
		payload.Summary.SSADegradedPackages = append(
			payload.Summary.SSADegradedPackages, degradedPkgs...)
	}

	// Step 1: CRAP analysis.
	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (CRAP)")
	if crapRes, err := steps.crapStep(patterns, moduleDir, coverProfile, stderr, ccFunc); err != nil {
		payload.Errors.CRAP = errString(err)
	} else {
		payload.CRAP = crapRes.JSON
		payload.Summary.CRAPload = crapRes.CRAPload
		payload.Summary.GazeCRAPload = crapRes.GazeCRAPload
	}

	// Step 2: Quality analysis.
	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (Quality)")
	if qualRes, err := steps.qualityStep(patterns, moduleDir, stderr); err != nil {
		payload.Errors.Quality = errString(err)
	} else {
		payload.Quality = qualRes.JSON
		payload.Summary.AvgContractCoverage = qualRes.AvgContractCoverage
		payload.Summary.SSADegraded = payload.Summary.SSADegraded || qualRes.SSADegraded
		payload.Summary.SSADegradedPackages = append(
			payload.Summary.SSADegradedPackages, qualRes.SSADegradedPackages...)
	}

	// Step 3: Classification analysis.
	_, _ = fmt.Fprintln(stderr, "Analyzing packages... (Classification)")
	if classifyRes, err := steps.classifyStep(patterns, moduleDir); err != nil {
		payload.Errors.Classify = errString(err)
	} else {
		payload.Classify = classifyRes.JSON
		payload.Summary.Contractual = classifyRes.Contractual
		payload.Summary.Ambiguous = classifyRes.Ambiguous
		payload.Summary.Incidental = classifyRes.Incidental
	}

	// Step 4: Documentation scan.
	_, _ = fmt.Fprintln(stderr, "Scanning documentation...")
	if docscanJSON, err := steps.docscanStep(moduleDir); err != nil {
		payload.Errors.Docscan = errString(err)
	} else {
		payload.Docscan = docscanJSON
	}

	return payload, nil
}

// captureJSON runs fn writing JSON to a buffer and returns the bytes.
func captureJSON(fn func(w io.Writer) error) (json.RawMessage, error) {
	var buf bytes.Buffer
	if err := fn(&buf); err != nil {
		return nil, err
	}
	return json.RawMessage(buf.Bytes()), nil
}
