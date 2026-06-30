package adapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/protocol"
)

// ExternalLineCoverageProvider implements crap.LineCoverageProvider
// by calling the "coverage" protocol method on an external analyzer.
type ExternalLineCoverageProvider struct {
	client *protocol.Client
}

// NewExternalLineCoverageProvider creates a line coverage provider
// that delegates to the given protocol client.
func NewExternalLineCoverageProvider(client *protocol.Client) *ExternalLineCoverageProvider {
	return &ExternalLineCoverageProvider{client: client}
}

// Coverage calls the "coverage" protocol method and converts the
// response to []crap.FuncCoverage. The coverProfile parameter is
// ignored for external analyzers — the analyzer manages its own
// coverage generation.
func (p *ExternalLineCoverageProvider) Coverage(patterns []string, rootDir string, coverProfile string) ([]crap.FuncCoverage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	resp, err := p.client.Call(ctx, protocol.MethodCoverage, protocol.CoverageParams{
		RootPath: rootDir,
		Patterns: patterns,
	})
	if err != nil {
		return nil, fmt.Errorf("coverage protocol call: %w", err)
	}
	if resp.Error != nil {
		return nil, fmt.Errorf("coverage protocol error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var result protocol.CoverageResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return nil, fmt.Errorf("parsing coverage result: %w", err)
	}

	return convertCoverage(result.Functions), nil
}

// convertCoverage maps protocol FunctionCoverageData to
// crap.FuncCoverage.
func convertCoverage(funcs []protocol.FunctionCoverageData) []crap.FuncCoverage {
	out := make([]crap.FuncCoverage, len(funcs))
	for i, f := range funcs {
		out[i] = crap.FuncCoverage{
			File:         f.File,
			FuncName:     f.Function,
			StartLine:    int(f.StartLine),
			EndLine:      int(f.EndLine),
			CoveredStmts: f.CoveredStmts,
			TotalStmts:   f.TotalStmts,
			Percentage:   f.Percentage,
		}
	}
	return out
}
