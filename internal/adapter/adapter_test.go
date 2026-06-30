package adapter_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/protocol"
)

// fakeBinaryPath is the path to the compiled fake_analyzer binary.
// Built once in TestMain from the protocol testdata.
var fakeBinaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "gaze-adapter-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}

	fakeBinaryPath = filepath.Join(tmpDir, "fake_analyzer")

	// Build the fake analyzer from the protocol testdata directory.
	cmd := exec.Command("go", "build", "-o", fakeBinaryPath,
		"./testdata/fake_analyzer/")
	cmd.Dir = filepath.Join("..", "protocol")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building fake_analyzer: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

// TestExternalComplexityProvider verifies that the complexity adapter
// correctly translates protocol responses to crap.FunctionComplexity.
func TestExternalComplexityProvider(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	mustInitialize(t, client)

	provider := adapter.NewExternalComplexityProvider(client)
	results, err := provider.Analyze([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d functions, want 3", len(results))
	}

	// Verify canned data: add(2), multiply(3), divide(5).
	want := map[string]int{
		"add":      2,
		"multiply": 3,
		"divide":   5,
	}
	for _, r := range results {
		expected, ok := want[r.Function]
		if !ok {
			t.Errorf("unexpected function %q", r.Function)
			continue
		}
		if r.Complexity != expected {
			t.Errorf("%s complexity = %d, want %d", r.Function, r.Complexity, expected)
		}
		if r.Package != "math_utils" {
			t.Errorf("%s package = %q, want %q", r.Function, r.Package, "math_utils")
		}
	}
}

// TestExternalLineCoverageProvider verifies that the coverage adapter
// correctly translates protocol responses to crap.FuncCoverage.
func TestExternalLineCoverageProvider(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	mustInitialize(t, client)

	provider := adapter.NewExternalLineCoverageProvider(client)
	results, err := provider.Coverage([]string{"./..."}, "/tmp/project", "")
	if err != nil {
		t.Fatalf("Coverage: %v", err)
	}

	if len(results) != 3 {
		t.Fatalf("got %d functions, want 3", len(results))
	}

	// Verify canned data: add(90%), multiply(60%), divide(0%).
	want := map[string]float64{
		"add":      90.0,
		"multiply": 60.0,
		"divide":   0.0,
	}
	for _, r := range results {
		expected, ok := want[r.FuncName]
		if !ok {
			t.Errorf("unexpected function %q", r.FuncName)
			continue
		}
		if r.Percentage != expected {
			t.Errorf("%s coverage = %g%%, want %g%%", r.FuncName, r.Percentage, expected)
		}
	}
}

// TestExternalSideEffectAnalyzer verifies that the side effect
// adapter correctly translates protocol responses to
// taxonomy.AnalysisResult with Classification attached.
func TestExternalSideEffectAnalyzer(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	var stderr bytes.Buffer
	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, &stderr,
	)

	results, err := analyzer.Analyze("math_utils")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}

	// Canned data: divide (2 effects), multiply (1 effect), add (0 effects).
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}

	// Find divide — should have ReturnValue and ErrorReturn.
	var divideEffects int
	var multiplyEffects int
	for _, r := range results {
		switch r.Target.Function {
		case "divide":
			divideEffects = len(r.SideEffects)
			// Verify classification is attached.
			for _, e := range r.SideEffects {
				if e.Classification == nil {
					t.Errorf("divide effect %s has nil classification", e.Type)
				}
			}
		case "multiply":
			multiplyEffects = len(r.SideEffects)
		case "add":
			if len(r.SideEffects) != 0 {
				t.Errorf("add has %d effects, want 0", len(r.SideEffects))
			}
		}
	}

	if divideEffects != 2 {
		t.Errorf("divide has %d effects, want 2", divideEffects)
	}
	if multiplyEffects != 1 {
		t.Errorf("multiply has %d effects, want 1", multiplyEffects)
	}
}

// TestExternalSideEffectAnalyzer_FiltersByPackage verifies that
// Analyze filters results by package path.
func TestExternalSideEffectAnalyzer_FiltersByPackage(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	analyzer := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, nil,
	)

	// Query a non-existent package — should return empty.
	results, err := analyzer.Analyze("nonexistent_pkg")
	if err != nil {
		t.Fatalf("Analyze: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("got %d results for nonexistent package, want 0", len(results))
	}
}

// TestSessionLifecycle verifies the full session lifecycle:
// spawn → initialize → providers ready → close.
func TestSessionLifecycle(t *testing.T) {
	var stderr bytes.Buffer
	session := adapter.NewSession(
		fakeBinaryPath, []string{"--stdio"},
		"/tmp/project", []string{"./..."},
		&stderr,
	)

	providers, err := session.Initialize()
	if err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer func() { _ = session.Close() }()

	if providers.AnalyzerName != "fake-analyzer" {
		t.Errorf("AnalyzerName = %q, want %q", providers.AnalyzerName, "fake-analyzer")
	}
	if providers.Language != "python" {
		t.Errorf("Language = %q, want %q", providers.Language, "python")
	}
	if providers.Complexity == nil {
		t.Error("Complexity provider is nil")
	}
	if providers.LineCoverage == nil {
		t.Error("LineCoverage provider is nil")
	}
	// Fake analyzer has test_mapping: true, so ContractCoverage
	// should be non-nil.
	if providers.ContractCoverage == nil {
		t.Error("ContractCoverage provider is nil (test_mapping is true)")
	}
	if !providers.Capabilities.TestMapping {
		t.Error("Capabilities.TestMapping = false, want true")
	}
	if !providers.Capabilities.Discover {
		t.Error("Capabilities.Discover = false, want true")
	}
	if providers.Capabilities.ClassifySignals {
		t.Error("Capabilities.ClassifySignals = true, want false")
	}
}

// TestSessionLifecycle_NoTestMapping verifies that when test_mapping
// capability is false, ContractCoverage is nil.
func TestSessionLifecycle_NoTestMapping(t *testing.T) {
	// The fake analyzer has test_mapping: true by default.
	// We test the session's behavior by verifying the provider
	// construction logic. Since we can't easily change the fake's
	// capabilities, we verify the positive case above and test
	// the no-op path through the contract provider directly.

	// Create a client and get capabilities.
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	// Use capabilities with test_mapping: false.
	caps := protocol.Capabilities{
		Discover:        true,
		TestMapping:     false,
		ClassifySignals: false,
	}

	provider := adapter.NewExternalContractCoverageProvider(
		client, caps, nil, "/tmp/project", []string{"./..."}, nil,
	)

	lookup, degraded, err := provider.Build([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if degraded != nil {
		t.Errorf("degraded = %v, want nil", degraded)
	}

	// No-op lookup should return (zero, false).
	info, ok := lookup("math_utils", "divide")
	if ok {
		t.Error("no-op lookup returned ok=true, want false")
	}
	if info.Percentage != 0 {
		t.Errorf("no-op lookup percentage = %g, want 0", info.Percentage)
	}
}

// TestContractCoverageProvider_WithTestMapping verifies that when
// test_mapping is supported, the provider calls the method and
// computes contract coverage.
func TestContractCoverageProvider_WithTestMapping(t *testing.T) {
	client := mustNewClient(t)
	defer func() { _ = client.Close() }()

	caps := mustInitialize(t, client)

	sideEffects := adapter.NewExternalSideEffectAnalyzer(
		client, caps, "/tmp/project", []string{"./..."}, nil,
	)

	provider := adapter.NewExternalContractCoverageProvider(
		client, caps, sideEffects,
		"/tmp/project", []string{"./..."}, nil,
	)

	lookup, _, err := provider.Build([]string{"./..."}, "/tmp/project")
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	// The fake analyzer maps test_multiply → multiply ReturnValue.
	// multiply has 1 contractual effect (ReturnValue), which is
	// mapped, so contract coverage should be 100%.
	info, ok := lookup("math_utils", "multiply")
	if !ok {
		t.Fatal("lookup returned ok=false for multiply")
	}
	if info.Percentage != 100.0 {
		t.Errorf("multiply contract coverage = %g%%, want 100%%", info.Percentage)
	}

	// divide has 2 contractual effects but no test mappings target
	// it, so contract coverage should be 0%.
	info, ok = lookup("math_utils", "divide")
	if !ok {
		t.Fatal("lookup returned ok=false for divide")
	}
	if info.Percentage != 0.0 {
		t.Errorf("divide contract coverage = %g%%, want 0%%", info.Percentage)
	}
}

// TestSessionClose_BeforeInitialize verifies that Close is safe
// to call before Initialize.
func TestSessionClose_BeforeInitialize(t *testing.T) {
	session := adapter.NewSession(
		fakeBinaryPath, []string{"--stdio"},
		"/tmp/project", []string{"./..."}, nil,
	)

	// Close without Initialize — should not panic or error.
	if err := session.Close(); err != nil {
		t.Errorf("Close before Initialize: %v", err)
	}
}

// TestSession_BinaryNotFound verifies error when analyzer binary
// does not exist.
func TestSession_BinaryNotFound(t *testing.T) {
	session := adapter.NewSession(
		"/nonexistent/analyzer", []string{"--stdio"},
		"/tmp/project", []string{"./..."}, nil,
	)

	_, err := session.Initialize()
	if err == nil {
		t.Fatal("Initialize with nonexistent binary should fail")
	}
}

// TestErrorPropagation_ComplexityProtocolError verifies that
// protocol errors from required methods are propagated.
func TestErrorPropagation_ComplexityProtocolError(t *testing.T) {
	// Build a fake that returns errors after initialize.
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio", "--error-response")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	// Initialize succeeds.
	ctx := context.Background()
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}

	// Complexity call should get the error response.
	provider := adapter.NewExternalComplexityProvider(client)
	_, err = provider.Analyze([]string{"./..."}, "/tmp/project")
	if err == nil {
		t.Fatal("Analyze should fail with error response")
	}
}

// --- helpers ---

func mustNewClient(t *testing.T) *protocol.Client {
	t.Helper()
	client, err := protocol.NewClient(fakeBinaryPath, "--stdio")
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return client
}

func mustInitialize(t *testing.T, client *protocol.Client) protocol.Capabilities {
	t.Helper()
	ctx := context.Background()
	resp, err := client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: "/tmp/project",
	})
	if err != nil {
		t.Fatalf("initialize: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("initialize error: %s", resp.Error.Message)
	}

	var initResult protocol.InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		t.Fatalf("unmarshal initialize result: %v", err)
	}
	return initResult.Capabilities
}
