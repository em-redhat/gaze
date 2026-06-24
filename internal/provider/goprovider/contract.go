package goprovider

import (
	"io"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/quality"
)

// GoContractCoverageProvider implements crap.ContractCoverageProvider
// by wrapping the existing BuildContractCoverageFunc logic. It
// encapsulates the entire quality pipeline (side effect analysis,
// classification, test loading, assertion mapping, contract coverage
// computation) behind a simple lookup function.
//
// Design decision D6: This is the highest-leverage interface — it
// encapsulates the entire quality pipeline behind a simple lookup
// function. The initial implementation delegates to
// BuildContractCoverageFunc as a thin wrapper to ensure
// byte-identical behavior.
type GoContractCoverageProvider struct {
	// Stderr receives diagnostic output from the quality pipeline.
	// If nil, diagnostics are suppressed.
	Stderr io.Writer

	// AIMapperFunc is an optional AI-assisted assertion mapping
	// callback. When non-nil, it is passed through to the quality
	// pipeline for unmapped assertion fallback.
	AIMapperFunc quality.AIMapperFunc
}

// Ensure GoContractCoverageProvider satisfies
// crap.ContractCoverageProvider.
var _ crap.ContractCoverageProvider = (*GoContractCoverageProvider)(nil)

// NewContractCoverageProvider creates a GoContractCoverageProvider
// with the given stderr writer for diagnostic output and an optional
// AI mapper function for assertion mapping fallback.
func NewContractCoverageProvider(
	stderr io.Writer,
	aiMapperFn ...quality.AIMapperFunc,
) *GoContractCoverageProvider {
	p := &GoContractCoverageProvider{
		Stderr: stderr,
	}
	if len(aiMapperFn) > 0 && aiMapperFn[0] != nil {
		p.AIMapperFunc = aiMapperFn[0]
	}
	return p
}

// Build runs the quality pipeline for the given package patterns and
// returns a contract coverage lookup function, a list of degraded
// package paths, and an error. The lookup function returns
// ContractCoverageInfo for a given (pkg, function) pair, or
// (zero, false) if no quality data exists.
//
// This is a thin wrapper around crap.BuildContractCoverageFunc to
// ensure byte-identical behavior with the existing implementation
// (task 3.2a). The returned error is always nil because
// BuildContractCoverageFunc handles errors internally via
// best-effort semantics.
func (p *GoContractCoverageProvider) Build(
	patterns []string,
	rootDir string,
) (func(pkg, function string) (crap.ContractCoverageInfo, bool), []string, error) {
	stderr := p.Stderr
	if stderr == nil {
		stderr = io.Discard
	}

	var ccFunc func(pkg, function string) (crap.ContractCoverageInfo, bool)
	var degradedPkgs []string

	if p.AIMapperFunc != nil {
		ccFunc, degradedPkgs = crap.BuildContractCoverageFunc(
			patterns, rootDir, stderr, p.AIMapperFunc,
		)
	} else {
		ccFunc, degradedPkgs = crap.BuildContractCoverageFunc(
			patterns, rootDir, stderr,
		)
	}

	return ccFunc, degradedPkgs, nil
}
