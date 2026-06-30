package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/unbound-force/gaze/internal/protocol"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// ExternalSideEffectAnalyzer implements crap.SideEffectAnalyzer by
// calling the "analyze" protocol method on an external analyzer.
//
// The analyzer returns per-function results for the whole project.
// This adapter caches the full result set and filters by pkgPath
// when Analyze(pkgPath) is called. Design decision D4: the adapter
// is a composition dependency of ExternalContractCoverageProvider,
// not a standalone adapter passed to crap.Options.
type ExternalSideEffectAnalyzer struct {
	client *protocol.Client
	caps   protocol.Capabilities
	stderr io.Writer

	// rootDir and patterns are set during session initialization
	// and used for the analyze call.
	rootDir  string
	patterns []string

	// mu protects the cached results.
	mu     sync.Mutex
	cached []taxonomy.AnalysisResult
	loaded bool
}

// NewExternalSideEffectAnalyzer creates a side effect analyzer that
// delegates to the given protocol client. The capabilities determine
// whether classify_signals is also called.
func NewExternalSideEffectAnalyzer(
	client *protocol.Client,
	caps protocol.Capabilities,
	rootDir string,
	patterns []string,
	stderr io.Writer,
) *ExternalSideEffectAnalyzer {
	return &ExternalSideEffectAnalyzer{
		client:   client,
		caps:     caps,
		rootDir:  rootDir,
		patterns: patterns,
		stderr:   stderr,
	}
}

// Analyze returns side effect analysis results for the given package
// path. On the first call, it fetches all results from the external
// analyzer and caches them. Subsequent calls filter the cache.
func (a *ExternalSideEffectAnalyzer) Analyze(pkgPath string) ([]taxonomy.AnalysisResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.loaded {
		if err := a.loadAll(); err != nil {
			return nil, err
		}
		a.loaded = true
	}

	// Filter cached results by package path.
	var filtered []taxonomy.AnalysisResult
	for _, r := range a.cached {
		if r.Target.Package == pkgPath {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// AllResults returns all cached analysis results across all packages.
// Must be called after at least one Analyze call. This is used by
// ExternalContractCoverageProvider to access the full result set.
func (a *ExternalSideEffectAnalyzer) AllResults() ([]taxonomy.AnalysisResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.loaded {
		if err := a.loadAll(); err != nil {
			return nil, err
		}
		a.loaded = true
	}
	return a.cached, nil
}

// loadAll fetches all analysis results from the external analyzer.
// Must be called with a.mu held.
func (a *ExternalSideEffectAnalyzer) loadAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	resp, err := a.client.Call(ctx, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: a.rootDir,
		Patterns: a.patterns,
	})
	if err != nil {
		return fmt.Errorf("analyze protocol call: %w", err)
	}
	if resp.Error != nil {
		return fmt.Errorf("analyze protocol error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var result protocol.AnalyzeResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return fmt.Errorf("parsing analyze result: %w", err)
	}

	a.cached = convertAnalysisResults(result.Functions, a.stderr)
	return nil
}

// convertAnalysisResults maps protocol AnalyzedFunction entries to
// taxonomy.AnalysisResult. Unknown SideEffectType values are
// included with a warning logged to stderr (design decision D9).
func convertAnalysisResults(funcs []protocol.AnalyzedFunction, stderr io.Writer) []taxonomy.AnalysisResult {
	results := make([]taxonomy.AnalysisResult, len(funcs))
	for i, f := range funcs {
		effects := make([]taxonomy.SideEffect, len(f.SideEffects))
		for j, se := range f.SideEffects {
			effectType := taxonomy.SideEffectType(se.Type)
			tier := taxonomy.TierOf(effectType)

			// Warn on unknown side effect types (they default to P4).
			if tier == taxonomy.TierP4 && !isKnownP4(effectType) && stderr != nil {
				_, _ = fmt.Fprintf(stderr, "warning: unknown side effect type %q from external analyzer (defaulting to P4)\n", se.Type)
			}

			effect := taxonomy.SideEffect{
				ID:          taxonomy.GenerateID(f.Package, f.Name, se.Type, se.Location),
				Type:        effectType,
				Tier:        tier,
				Location:    se.Location,
				Description: se.Description,
				Target:      se.Target,
			}

			if se.Classification != nil {
				effect.Classification = &taxonomy.Classification{
					Label:      taxonomy.ClassificationLabel(se.Classification.Label),
					Confidence: se.Classification.Confidence,
				}
			}

			effects[j] = effect
		}

		results[i] = taxonomy.AnalysisResult{
			Target: taxonomy.FunctionTarget{
				Package:  f.Package,
				Function: f.Name,
				Location: fmt.Sprintf("%s:%d", f.File, f.Line),
			},
			SideEffects: effects,
		}
	}
	return results
}

// isKnownP4 checks whether a side effect type is a known P4 type
// (as opposed to an unknown type that defaults to P4).
func isKnownP4(t taxonomy.SideEffectType) bool {
	switch t {
	case taxonomy.ReflectionMutation,
		taxonomy.UnsafeMutation,
		taxonomy.CgoCall,
		taxonomy.FinalizerRegistration,
		taxonomy.SyncPoolOp,
		taxonomy.ClosureCaptureMutation:
		return true
	}
	return false
}
