// Package adapter implements external analyzer provider adapters
// that translate JSON-RPC protocol responses into the Phase 1
// provider interface types (crap.ComplexityProvider,
// crap.LineCoverageProvider, crap.SideEffectAnalyzer,
// crap.ContractCoverageProvider).
//
// Each adapter holds a reference to a protocol.Client and calls the
// appropriate protocol method in its interface method. The adapters
// are constructed by Session after the initialize handshake.
//
// Design decision D4: External adapters implement Phase 1 interfaces.
package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/protocol"
)

// ExternalComplexityProvider implements crap.ComplexityProvider by
// calling the "complexity" protocol method on an external analyzer.
type ExternalComplexityProvider struct {
	client *protocol.Client
}

// NewExternalComplexityProvider creates a complexity provider that
// delegates to the given protocol client.
func NewExternalComplexityProvider(client *protocol.Client) *ExternalComplexityProvider {
	return &ExternalComplexityProvider{client: client}
}

// Analyze calls the "complexity" protocol method and converts the
// response to []crap.FunctionComplexity.
func (p *ExternalComplexityProvider) Analyze(patterns []string, rootDir string) ([]crap.FunctionComplexity, error) {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	resp, err := p.client.Call(ctx, protocol.MethodComplexity, protocol.ComplexityParams{
		RootPath: rootDir,
		Patterns: patterns,
	})
	if err != nil {
		return nil, fmt.Errorf("complexity protocol call: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("complexity protocol error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var result protocol.ComplexityResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing complexity result: %w", err)
	}

	return convertComplexity(result.Functions), nil
}

// convertComplexity maps protocol FunctionComplexityData to
// crap.FunctionComplexity.
func convertComplexity(funcs []protocol.FunctionComplexityData) []crap.FunctionComplexity {
	out := make([]crap.FunctionComplexity, len(funcs))
	for i, f := range funcs {
		out[i] = crap.FunctionComplexity{
			Package:    f.Package,
			Function:   f.Name,
			File:       f.File,
			Line:       f.Line,
			Complexity: f.Complexity,
		}
	}
	return out
}
