package adapter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/protocol"
)

// Providers holds the constructed provider adapters ready to be
// passed to crap.Options.
type Providers struct {
	// Complexity is the external complexity provider.
	Complexity crap.ComplexityProvider

	// LineCoverage is the external line coverage provider.
	LineCoverage crap.LineCoverageProvider

	// ContractCoverage is the external contract coverage provider.
	// Nil when test_mapping capability is false.
	ContractCoverage crap.ContractCoverageProvider

	// Capabilities is the analyzer's declared capabilities from
	// the initialize handshake.
	Capabilities protocol.Capabilities

	// AnalyzerName is the human-readable analyzer name.
	AnalyzerName string

	// Language is the primary language the analyzer targets.
	Language string
}

// Session manages the full protocol lifecycle with an external
// analyzer: spawn binary, initialize (get capabilities), construct
// provider adapters, and shutdown.
//
// Design decision D2: Protocol lifecycle matches Issue #95.
type Session struct {
	client   *protocol.Client
	binary   string
	args     []string
	rootDir  string
	patterns []string
	stderr   io.Writer

	// caps is populated after Initialize.
	caps     protocol.Capabilities
	initDone bool
}

// NewSession creates a new session for the given analyzer binary.
// The binary is not spawned until Initialize is called.
func NewSession(binary string, args []string, rootDir string, patterns []string, stderr io.Writer) *Session {
	return &Session{
		binary:   binary,
		args:     args,
		rootDir:  rootDir,
		patterns: patterns,
		stderr:   stderr,
	}
}

// Initialize spawns the analyzer binary and performs the initialize
// handshake. Returns the Providers struct with all adapters ready
// for use with crap.Options.
//
// The caller must call Close when done to shut down the analyzer.
func (s *Session) Initialize() (*Providers, error) {
	client, err := protocol.NewClient(s.binary, s.args...)
	if err != nil {
		return nil, fmt.Errorf("spawning analyzer %s: %w", s.binary, err)
	}
	s.client = client

	// Initialize handshake with short timeout (D10).
	ctx, cancel := context.WithTimeout(context.Background(), protocol.ShortTimeout)
	defer cancel()

	resp, err := s.client.Call(ctx, protocol.MethodInitialize, protocol.InitializeParams{
		RootPath: s.rootDir,
	})
	if err != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("initialize handshake: %w", err)
	}
	if resp.Error != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("initialize error: %s (code %d)", resp.Error.Message, resp.Error.Code)
	}

	var initResult protocol.InitializeResult
	if err := json.Unmarshal(resp.Result, &initResult); err != nil {
		_ = s.client.Close()
		return nil, fmt.Errorf("parsing initialize result: %w", err)
	}

	s.caps = initResult.Capabilities
	s.initDone = true

	// Construct provider adapters.
	complexityProvider := NewExternalComplexityProvider(s.client)
	coverageProvider := NewExternalLineCoverageProvider(s.client)

	sideEffectAnalyzer := NewExternalSideEffectAnalyzer(
		s.client, s.caps, s.rootDir, s.patterns, s.stderr,
	)

	var contractProvider crap.ContractCoverageProvider
	if s.caps.TestMapping {
		contractProvider = NewExternalContractCoverageProvider(
			s.client, s.caps, sideEffectAnalyzer,
			s.rootDir, s.patterns, s.stderr,
		)
	}

	return &Providers{
		Complexity:       complexityProvider,
		LineCoverage:     coverageProvider,
		ContractCoverage: contractProvider,
		Capabilities:     initResult.Capabilities,
		AnalyzerName:     initResult.AnalyzerName,
		Language:         initResult.Language,
	}, nil
}

// Close sends a shutdown request to the analyzer and waits for the
// subprocess to exit. Safe to call even if Initialize was not called
// or failed.
func (s *Session) Close() error {
	if s.client == nil {
		return nil
	}
	return s.client.Close()
}
