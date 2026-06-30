package crap

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// --- Task 4.3: WriteComparisonJSON tests ---

// buildTestComparisonResult creates a ComparisonResult with all
// status types represented for test coverage.
func buildTestComparisonResult() *ComparisonResult {
	return &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("internal/crap/analyze.go", "Analyze", 12.5, float64Ptr(18.3)),
				makeScore("internal/crap/report.go", "WriteText", 8.0, float64Ptr(10.0)),
				makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				// New functions are in NewFunctions list, but also in
				// Report.Scores since the current report contains them.
				makeScore("internal/crap/helper.go", "helperFunc", 12.0, nil),
				makeScore("internal/crap/complex.go", "complexFunc", 42.0, nil),
			},
			Summary: Summary{
				TotalFunctions:  5,
				AvgComplexity:   4.0,
				AvgLineCoverage: 70.0,
				AvgCRAP:         15.3,
				CRAPload:        1,
				CRAPThreshold:   15,
				WorstCRAP:       nil,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:      makeScore("internal/crap/analyze.go", "Analyze", 9.2, float64Ptr(14.1)),
				Current:       makeScore("internal/crap/analyze.go", "Analyze", 12.5, float64Ptr(18.3)),
				CRAPDelta:     3.3,
				GazeCRAPDelta: float64Ptr(4.2),
				Status:        StatusRegression,
			},
			{
				Baseline:      makeScore("internal/crap/report.go", "WriteText", 12.0, float64Ptr(15.0)),
				Current:       makeScore("internal/crap/report.go", "WriteText", 8.0, float64Ptr(10.0)),
				CRAPDelta:     -4.0,
				GazeCRAPDelta: float64Ptr(-5.0),
				Status:        StatusImprovement,
			},
			{
				Baseline:  makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				Current:   makeScore("internal/crap/crap.go", "Formula", 2.0, float64Ptr(2.0)),
				CRAPDelta: 0.0,
				Status:    StatusUnchanged,
			},
		},
		NewFunctions: []Score{
			makeScore("internal/crap/helper.go", "helperFunc", 12.0, nil),
			makeScore("internal/crap/complex.go", "complexFunc", 42.0, nil),
		},
		RemovedFunctions: []Score{
			makeScore("internal/crap/old.go", "oldFunc", 5.0, nil),
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Improvements:         1,
			NewFunctions:         1,
			NewViolations:        1,
			RemovedFunctions:     1,
			Unchanged:            1,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
			Passed:               false,
		},
	}
}

func TestSC007_WriteComparisonJSON_Structure(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	// Verify top-level keys exist.
	requiredKeys := []string{"scores", "new_functions", "removed_functions", "comparison", "summary"}
	for _, key := range requiredKeys {
		if _, ok := output[key]; !ok {
			t.Errorf("missing top-level key %q in JSON output", key)
		}
	}

	// Verify scores have delta fields.
	var scores []map[string]interface{}
	if err := json.Unmarshal(output["scores"], &scores); err != nil {
		t.Fatalf("parsing scores: %v", err)
	}

	if len(scores) != 3 {
		t.Fatalf("len(scores) = %d, want 3 (new functions should be in new_functions, not scores)", len(scores))
	}

	// Check the regression score has delta fields.
	regressionScore := scores[0]
	deltaFields := []string{"baseline_crap", "crap_delta", "status"}
	for _, field := range deltaFields {
		if _, ok := regressionScore[field]; !ok {
			t.Errorf("regression score missing field %q", field)
		}
	}

	// Verify new_functions has 2 entries.
	var newFuncs []map[string]interface{}
	if err := json.Unmarshal(output["new_functions"], &newFuncs); err != nil {
		t.Fatalf("parsing new_functions: %v", err)
	}
	if len(newFuncs) != 2 {
		t.Errorf("len(new_functions) = %d, want 2", len(newFuncs))
	}

	// Verify removed_functions has 1 entry.
	var removedFuncs []map[string]interface{}
	if err := json.Unmarshal(output["removed_functions"], &removedFuncs); err != nil {
		t.Fatalf("parsing removed_functions: %v", err)
	}
	if len(removedFuncs) != 1 {
		t.Errorf("len(removed_functions) = %d, want 1", len(removedFuncs))
	}
}

func TestSC007_WriteComparisonJSON_AllStatuses(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	outputStr := buf.String()

	// All 6 status values should appear in the output.
	expectedStatuses := []string{
		string(StatusRegression),
		string(StatusImprovement),
		string(StatusUnchanged),
		string(StatusNew),
		string(StatusNewViolation),
		string(StatusRemoved),
	}
	for _, status := range expectedStatuses {
		if !strings.Contains(outputStr, `"`+status+`"`) {
			t.Errorf("output missing status value %q", status)
		}
	}
}

func TestSC007_WriteComparisonJSON_ComparisonPassed(t *testing.T) {
	// Build a result that passes (no regressions, no violations).
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 8.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 12.0, nil),
				Current:   makeScore("a.go", "Func", 8.0, nil),
				CRAPDelta: -4.0,
				Status:    StatusImprovement,
			},
		},
		Summary: ComparisonSummary{
			Improvements:         1,
			Passed:               true,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	var comparison ComparisonSummary
	if err := json.Unmarshal(output["comparison"], &comparison); err != nil {
		t.Fatalf("parsing comparison: %v", err)
	}

	if !comparison.Passed {
		t.Error("comparison.passed = false, want true")
	}
}

func TestSC007_BackwardCompat_NoComparisonFields(t *testing.T) {
	// Render a normal crap.Report through WriteJSON (not
	// WriteComparisonJSON) and verify zero comparison fields
	// appear in the output.
	rpt := &Report{
		Scores: []Score{
			makeScore("a.go", "Func", 5.0, nil),
		},
		Summary: Summary{
			TotalFunctions:  1,
			AvgComplexity:   3.0,
			AvgLineCoverage: 80.0,
			AvgCRAP:         5.0,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, rpt); err != nil {
		t.Fatalf("WriteJSON() error: %v", err)
	}

	outputStr := buf.String()

	// These comparison-specific fields MUST NOT appear in normal
	// WriteJSON output.
	forbiddenFields := []string{
		"baseline_crap",
		"crap_delta",
		"baseline_gaze_crap",
		"gaze_crap_delta",
		"new_functions",
		"removed_functions",
		`"comparison"`,
	}
	for _, field := range forbiddenFields {
		if strings.Contains(outputStr, field) {
			t.Errorf("normal WriteJSON output contains comparison field %q — backward compatibility broken", field)
		}
	}
}

// --- Task 4.4: WriteComparisonText tests ---

func TestSC008_WriteComparisonText_PassHeader(t *testing.T) {
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 8.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 12.0, nil),
				Current:   makeScore("a.go", "Func", 8.0, nil),
				CRAPDelta: -4.0,
				Status:    StatusImprovement,
			},
		},
		Summary: ComparisonSummary{
			Improvements:         1,
			Passed:               true,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	if !strings.Contains(buf.String(), "PASS") {
		t.Error("output missing PASS header")
	}
}

func TestSC008_WriteComparisonText_FailHeader(t *testing.T) {
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 12.5, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 9.0, nil),
				Current:   makeScore("a.go", "Func", 12.5, nil),
				CRAPDelta: 3.5,
				Status:    StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	if !strings.Contains(buf.String(), "FAIL") {
		t.Error("output missing FAIL header")
	}
}

func TestSC008_WriteComparisonText_EmptySections(t *testing.T) {
	// All unchanged — no regressions, improvements, new, or removed
	// sections should appear.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 5.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 1,
				CRAPThreshold:  15,
			},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 5.0, nil),
				Current:   makeScore("a.go", "Func", 5.0, nil),
				CRAPDelta: 0.0,
				Status:    StatusUnchanged,
			},
		},
		Summary: ComparisonSummary{
			Unchanged:            1,
			Passed:               true,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "Regressions:") {
		t.Error("empty regressions section should be omitted")
	}
	if strings.Contains(output, "Improvements:") {
		t.Error("empty improvements section should be omitted")
	}
	if strings.Contains(output, "New Functions") {
		t.Error("empty new functions section should be omitted")
	}
	if strings.Contains(output, "Removed Functions:") {
		t.Error("empty removed functions section should be omitted")
	}
}

func TestSC008_WriteComparisonText_RegressionTable(t *testing.T) {
	result := buildTestComparisonResult()

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	// Verify regression table appears with the regressed function.
	if !strings.Contains(output, "Regressions:") {
		t.Error("output missing 'Regressions:' section")
	}
	if !strings.Contains(output, "Analyze") {
		t.Error("regression table missing function name 'Analyze'")
	}
	// Verify delta format (+3.3).
	if !strings.Contains(output, "+3.3") {
		t.Error("regression table missing delta value '+3.3'")
	}
	// Verify improvements section also appears.
	if !strings.Contains(output, "Improvements:") {
		t.Error("output missing 'Improvements:' section")
	}
	// Verify new function violation appears.
	if !strings.Contains(output, "VIOLATION") {
		t.Error("output missing VIOLATION marker for new function above threshold")
	}
	// Verify removed functions section.
	if !strings.Contains(output, "Removed Functions:") {
		t.Error("output missing 'Removed Functions:' section")
	}
}

// --- Issue #163: GazeCRAP delta table display tests ---

func TestDeltaTable_TwoRowFormat_WithGazeCRAP(t *testing.T) {
	// When a delta has GazeCRAP data, the two-row format should
	// show function name on its own line, then indented GazeCRAP
	// and CRAP rows.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("internal/crap/analyze.go", "Analyze", 12.0, float64Ptr(12.1)),
			},
			Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:      makeScore("internal/crap/analyze.go", "Analyze", 12.0, float64Ptr(8.5)),
				Current:       makeScore("internal/crap/analyze.go", "Analyze", 12.0, float64Ptr(12.1)),
				CRAPDelta:     0.0,
				GazeCRAPDelta: float64Ptr(3.6),
				Status:        StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	// Function name should appear on its own line.
	if !strings.Contains(output, "Analyze (") {
		t.Error("output missing function name")
	}
	// GazeCRAP row with values.
	if !strings.Contains(output, "GazeCRAP") {
		t.Error("output missing GazeCRAP label")
	}
	if !strings.Contains(output, "8.5") {
		t.Error("output missing GazeCRAP baseline value 8.5")
	}
	if !strings.Contains(output, "12.1") {
		t.Error("output missing GazeCRAP current value 12.1")
	}
	if !strings.Contains(output, "+3.6") {
		t.Error("output missing GazeCRAP delta +3.6")
	}
	// CRAP row with values.
	if !strings.Contains(output, "+0.0") {
		t.Error("output missing CRAP delta +0.0")
	}
}

func TestDeltaTable_TwoRowFormat_MixedGazeCRAP(t *testing.T) {
	// One function has GazeCRAP, another doesn't. The one with
	// GazeCRAP gets both rows, the one without gets only CRAP.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "WithGaze", 10.0, float64Ptr(15.0)),
				makeScore("b.go", "NoGaze", 8.0, nil),
			},
			Summary: Summary{TotalFunctions: 2, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:      makeScore("a.go", "WithGaze", 8.0, float64Ptr(10.0)),
				Current:       makeScore("a.go", "WithGaze", 10.0, float64Ptr(15.0)),
				CRAPDelta:     2.0,
				GazeCRAPDelta: float64Ptr(5.0),
				Status:        StatusRegression,
			},
			{
				Baseline:  makeScore("b.go", "NoGaze", 5.0, nil),
				Current:   makeScore("b.go", "NoGaze", 8.0, nil),
				CRAPDelta: 3.0,
				Status:    StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          2,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	// WithGaze should have GazeCRAP row.
	if !strings.Contains(output, "GazeCRAP") {
		t.Error("output missing GazeCRAP label for WithGaze function")
	}

	// NoGaze should NOT have a GazeCRAP row — count occurrences.
	// GazeCRAP label should appear exactly once (for WithGaze only).
	count := strings.Count(output, "GazeCRAP")
	if count != 1 {
		t.Errorf("GazeCRAP label appears %d times, want 1 (only for WithGaze)", count)
	}

	// Both functions should have CRAP rows.
	crapCount := strings.Count(output, "    CRAP")
	if crapCount != 2 {
		t.Errorf("CRAP label appears %d times, want 2 (one per function)", crapCount)
	}
}

func TestDeltaTable_SingleRowFormat_NoGazeCRAP(t *testing.T) {
	// When no function has GazeCRAP data, use the single-row
	// format (backward compatible).
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "FuncA", 15.0, nil),
			},
			Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "FuncA", 10.0, nil),
				Current:   makeScore("a.go", "FuncA", 15.0, nil),
				CRAPDelta: 5.0,
				Status:    StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	// Should NOT contain indented metric labels (two-row format).
	if strings.Contains(output, "    CRAP") {
		t.Error("single-row format should not contain indented CRAP label")
	}
	if strings.Contains(output, "GazeCRAP") {
		t.Error("single-row format should not contain GazeCRAP label")
	}

	// Should contain the function with values on one line.
	if !strings.Contains(output, "FuncA") {
		t.Error("output missing function name")
	}
	if !strings.Contains(output, "+5.0") {
		t.Error("output missing delta +5.0")
	}
}

func TestDeltaTable_GazeCRAPBeforeCRAP(t *testing.T) {
	// Verify GazeCRAP row appears before CRAP row (D1).
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Func", 10.0, float64Ptr(20.0)),
			},
			Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:      makeScore("a.go", "Func", 8.0, float64Ptr(15.0)),
				Current:       makeScore("a.go", "Func", 10.0, float64Ptr(20.0)),
				CRAPDelta:     2.0,
				GazeCRAPDelta: float64Ptr(5.0),
				Status:        StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()

	gazeIdx := strings.Index(output, "GazeCRAP")
	crapIdx := strings.Index(output, "    CRAP")
	if gazeIdx < 0 {
		t.Fatal("output missing GazeCRAP label")
	}
	if crapIdx < 0 {
		t.Fatal("output missing indented CRAP label")
	}
	if gazeIdx >= crapIdx {
		t.Errorf("GazeCRAP (pos %d) should appear before CRAP (pos %d)",
			gazeIdx, crapIdx)
	}
}

func TestDeltaTable_TwoRowFormat_WidthCompliance(t *testing.T) {
	// Verify no line in the comparison section (after "--- Baseline
	// Comparison") exceeds 80 characters. The preceding WriteText
	// output has its own width constraints.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("internal/crap/analyze.go", "AnalyzeVeryLongFunctionName", 12.5, float64Ptr(18.3)),
			},
			Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:      makeScore("internal/crap/analyze.go", "AnalyzeVeryLongFunctionName", 9.2, float64Ptr(14.1)),
				Current:       makeScore("internal/crap/analyze.go", "AnalyzeVeryLongFunctionName", 12.5, float64Ptr(18.3)),
				CRAPDelta:     3.3,
				GazeCRAPDelta: float64Ptr(4.2),
				Status:        StatusRegression,
			},
		},
		Summary: ComparisonSummary{
			Regressions:          1,
			Passed:               false,
			Epsilon:              0.5,
			NewFunctionThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()
	comparisonStart := strings.Index(output, "--- Baseline Comparison")
	if comparisonStart < 0 {
		t.Fatal("output missing '--- Baseline Comparison' header")
	}

	comparisonSection := output[comparisonStart:]
	lines := strings.Split(comparisonSection, "\n")
	for i, line := range lines {
		if len(line) > 80 {
			t.Errorf("comparison line %d exceeds 80 columns (%d chars): %q",
				i+1, len(line), line)
		}
	}
}

// --- Issue #164: GazeCRAP new-function threshold report tests ---

func TestSC006_TextReport_ViolationShowsGazeCRAP(t *testing.T) {
	// A new-function violation with GazeCRAP available should
	// display both CRAP and GazeCRAP scores in the text output.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Existing", 5.0, nil),
				makeScore("b.go", "NewFunc", 25.0, float64Ptr(40.0)),
			},
			Summary: Summary{
				TotalFunctions: 2,
				CRAPThreshold:  15,
			},
		},
		NewFunctions: []Score{
			makeScore("b.go", "NewFunc", 25.0, float64Ptr(40.0)),
		},
		Summary: ComparisonSummary{
			NewViolations:                1,
			Passed:                       false,
			Epsilon:                      0.5,
			NewFunctionThreshold:         30,
			NewFunctionGazeCRAPThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CRAP: 25.0") {
		t.Error("violation output missing CRAP score")
	}
	if !strings.Contains(output, "GazeCRAP: 40.0") {
		t.Error("violation output missing GazeCRAP score")
	}
	if !strings.Contains(output, "VIOLATION") {
		t.Error("violation output missing [VIOLATION] marker")
	}
}

func TestSC006_TextReport_ViolationWithoutGazeCRAP(t *testing.T) {
	// A violation with nil GazeCRAP should show only CRAP, no
	// GazeCRAP label.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Existing", 5.0, nil),
				makeScore("b.go", "NewFunc", 42.0, nil),
			},
			Summary: Summary{
				TotalFunctions: 2,
				CRAPThreshold:  15,
			},
		},
		NewFunctions: []Score{
			makeScore("b.go", "NewFunc", 42.0, nil),
		},
		Summary: ComparisonSummary{
			NewViolations:                1,
			Passed:                       false,
			Epsilon:                      0.5,
			NewFunctionThreshold:         30,
			NewFunctionGazeCRAPThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonText(&buf, result); err != nil {
		t.Fatalf("WriteComparisonText() error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "CRAP: 42.0") {
		t.Error("violation output missing CRAP score")
	}
	if strings.Contains(output, "GazeCRAP:") {
		t.Error("violation output should NOT contain GazeCRAP when GazeCRAP is nil")
	}
	if !strings.Contains(output, "VIOLATION") {
		t.Error("violation output missing [VIOLATION] marker")
	}
}

func TestSC005_JSONOutput_SummaryIncludesGazeCRAPThreshold(t *testing.T) {
	// The comparison JSON output must include the
	// new_function_gaze_crap_threshold field.
	result := &ComparisonResult{
		Report: &Report{
			Scores:  []Score{makeScore("a.go", "Func", 5.0, nil)},
			Summary: Summary{TotalFunctions: 1, CRAPThreshold: 15},
		},
		Deltas: []FunctionDelta{
			{
				Baseline:  makeScore("a.go", "Func", 5.0, nil),
				Current:   makeScore("a.go", "Func", 5.0, nil),
				CRAPDelta: 0,
				Status:    StatusUnchanged,
			},
		},
		Summary: ComparisonSummary{
			Unchanged:                    1,
			Passed:                       true,
			Epsilon:                      0.5,
			NewFunctionThreshold:         30,
			NewFunctionGazeCRAPThreshold: 40,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	var comparison ComparisonSummary
	if err := json.Unmarshal(output["comparison"], &comparison); err != nil {
		t.Fatalf("parsing comparison: %v", err)
	}

	if comparison.NewFunctionGazeCRAPThreshold != 40 {
		t.Errorf("comparison.new_function_gaze_crap_threshold = %g, want 40",
			comparison.NewFunctionGazeCRAPThreshold)
	}
}

func TestSC005_JSONOutput_NewFunctionStatusUsesGazeCRAP(t *testing.T) {
	// A new function with CRAP below threshold but GazeCRAP above
	// threshold should have status "new_violation" in JSON output.
	result := &ComparisonResult{
		Report: &Report{
			Scores: []Score{
				makeScore("a.go", "Existing", 5.0, nil),
				makeScore("b.go", "NewFunc", 25.0, float64Ptr(40.0)),
			},
			Summary: Summary{TotalFunctions: 2, CRAPThreshold: 15},
		},
		NewFunctions: []Score{
			makeScore("b.go", "NewFunc", 25.0, float64Ptr(40.0)),
		},
		Summary: ComparisonSummary{
			NewViolations:                1,
			Passed:                       false,
			Epsilon:                      0.5,
			NewFunctionThreshold:         30,
			NewFunctionGazeCRAPThreshold: 30,
		},
	}

	var buf bytes.Buffer
	if err := WriteComparisonJSON(&buf, result); err != nil {
		t.Fatalf("WriteComparisonJSON() error: %v", err)
	}

	var output map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	var newFuncs []map[string]interface{}
	if err := json.Unmarshal(output["new_functions"], &newFuncs); err != nil {
		t.Fatalf("parsing new_functions: %v", err)
	}

	if len(newFuncs) != 1 {
		t.Fatalf("len(new_functions) = %d, want 1", len(newFuncs))
	}

	status, ok := newFuncs[0]["status"].(string)
	if !ok {
		t.Fatal("new_functions[0].status is not a string")
	}
	if status != string(StatusNewViolation) {
		t.Errorf("new_functions[0].status = %q, want %q",
			status, StatusNewViolation)
	}
}
