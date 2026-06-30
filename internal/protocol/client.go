package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"sync/atomic"
	"time"
)

// Default per-method timeouts. Design decision D10: per-method
// defaults prevent hung analyzers from blocking gaze indefinitely.
const (
	// AnalysisTimeout is the default timeout for analyze, complexity,
	// and coverage methods (5 minutes).
	AnalysisTimeout = 5 * time.Minute

	// ShortTimeout is the default timeout for initialize, discover,
	// and shutdown methods (30 seconds).
	ShortTimeout = 30 * time.Second

)

// Client manages a JSON-RPC 2.0 session with an external analyzer
// subprocess. It spawns the analyzer binary, sends requests via
// stdin, and reads responses from stdout. Stderr is captured for
// diagnostics.
//
// The client is sequential — it sends one request at a time and
// waits for the matching response before sending the next. No
// multiplexing or concurrent requests.
//
// Design decision D1: JSON-RPC 2.0 over stdin/stdout.
// Design decision D10: Call accepts context.Context for timeouts.
type Client struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	stderr *bytes.Buffer

	// nextID is an atomic counter for generating unique request IDs.
	nextID atomic.Int64

	// mu serializes Call invocations. The protocol is sequential —
	// one request/response pair at a time.
	mu sync.Mutex

	// closed tracks whether Close has been called.
	closed atomic.Bool
}

// NewClient spawns an external analyzer binary as a subprocess and
// returns a Client for communicating with it via JSON-RPC 2.0.
//
// The binary is resolved via exec.LookPath. The args slice is passed
// directly to the subprocess (typically ["--stdio"]).
//
// Returns an error if the binary is not found or the subprocess
// fails to start.
func NewClient(binary string, args ...string) (*Client, error) {
	binPath, err := exec.LookPath(binary)
	if err != nil {
		return nil, fmt.Errorf("analyzer binary %q not found: %w", binary, err)
	}

	cmd := exec.Command(binPath, args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdin pipe for %s: %w", binary, err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("creating stdout pipe for %s: %w", binary, err)
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("starting analyzer %s: %w", binary, err)
	}

	scanner := bufio.NewScanner(stdout)
	// Allow up to 10 MiB per line for large analysis responses.
	const maxScanSize = 10 * 1024 * 1024
	scanner.Buffer(make([]byte, 64*1024), maxScanSize)

	c := &Client{
		cmd:    cmd,
		stdin:  stdin,
		stdout: scanner,
		stderr: &stderr,
	}

	return c, nil
}

// Call sends a JSON-RPC 2.0 request to the analyzer and reads the
// response. The method and params are encoded into a Request, sent
// as a single line of JSON to the subprocess's stdin, and the
// response is read from stdout.
//
// The context controls the deadline. When the context is cancelled
// or its deadline expires, the subprocess is killed and a timeout
// error is returned.
//
// Returns an error if:
//   - The request cannot be marshaled
//   - Writing to stdin fails (subprocess crashed)
//   - Reading from stdout fails (subprocess crashed)
//   - The response JSON is malformed
//   - The response ID does not match the request ID
//   - The context deadline expires
//
// If the response contains a JSON-RPC error, it is returned as a
// *Error (which implements the error interface) in the Response.
// The caller should check Response.Error before using Response.Result.
func (c *Client) Call(ctx context.Context, method string, params any) (*Response, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed.Load() {
		return nil, fmt.Errorf("protocol client is closed")
	}

	id := c.nextID.Add(1)

	req := Request{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling %s request: %w", method, err)
	}

	// Use a channel to coordinate the write+read with context
	// cancellation. This allows us to kill the subprocess if the
	// context deadline expires.
	type callResult struct {
		resp *Response
		err  error
	}
	resultCh := make(chan callResult, 1)

	go func() {
		// Write request as a single line.
		reqBytes = append(reqBytes, '\n')
		if _, werr := c.stdin.Write(reqBytes); werr != nil {
			resultCh <- callResult{err: fmt.Errorf("writing %s request to analyzer stdin: %w", method, werr)}
			return
		}

		// Read response line.
		if !c.stdout.Scan() {
			scanErr := c.stdout.Err()
			// Note: we do NOT read c.stderr here because exec.Cmd
			// writes to it concurrently. Stderr is only safe to
			// read after cmd.Wait() returns (see Stderr() method).
			if scanErr != nil {
				resultCh <- callResult{err: fmt.Errorf("reading %s response from analyzer: %w", method, scanErr)}
			} else {
				resultCh <- callResult{err: fmt.Errorf("analyzer closed stdout while waiting for %s response (process may have crashed; check Stderr())", method)}
			}
			return
		}

		// Copy the scanned bytes — the scanner's buffer is reused
		// on the next Scan() call.
		line := make([]byte, len(c.stdout.Bytes()))
		copy(line, c.stdout.Bytes())

		var resp Response
		if uerr := json.Unmarshal(line, &resp); uerr != nil {
			resultCh <- callResult{err: fmt.Errorf("parsing %s response JSON: %w (raw: %s)", method, uerr, truncate(line, 200))}
			return
		}

		if resp.ID != id {
			resultCh <- callResult{err: fmt.Errorf("response ID mismatch for %s: expected %d, got %d", method, id, resp.ID)}
			return
		}

		resultCh <- callResult{resp: &resp}
	}()

	select {
	case <-ctx.Done():
		// Context expired — kill the subprocess.
		if c.cmd.Process != nil {
			_ = c.cmd.Process.Kill()
		}
		return nil, fmt.Errorf("%s: %w", method, ctx.Err())
	case result := <-resultCh:
		return result.resp, result.err
	}
}

// Close sends a shutdown request to the analyzer and waits for the
// subprocess to exit. If the subprocess does not exit cleanly, it
// is killed.
//
// Close is idempotent — calling it multiple times is safe.
func (c *Client) Close() error {
	if c.closed.Swap(true) {
		return nil // already closed
	}

	// Send shutdown request directly, bypassing the closed check
	// in Call(). This is necessary because we just set closed=true
	// above to prevent concurrent callers, but we still need to
	// send the final shutdown message.
	c.sendShutdown()

	// Close stdin to signal EOF to the subprocess.
	_ = c.stdin.Close()

	// Wait for the subprocess to exit.
	err := c.cmd.Wait()

	return err
}

// sendShutdown sends the shutdown JSON-RPC request directly to the
// subprocess stdin, bypassing Call()'s closed-state guard. This is
// the only method allowed to write after closed=true — it exists
// so Close() can send the final shutdown message before closing stdin.
//
// Best-effort: if the analyzer is already dead or the pipe is broken,
// the write fails silently and Close() proceeds to cleanup.
func (c *Client) sendShutdown() {
	c.mu.Lock()
	defer c.mu.Unlock()

	id := c.nextID.Add(1)
	req := Request{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Method:  MethodShutdown,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return // best-effort
	}
	data = append(data, '\n')
	_, _ = c.stdin.Write(data)
	// Don't wait for response — analyzer may exit immediately.
}

// Stderr returns the accumulated stderr output from the analyzer
// subprocess. Useful for diagnostics when errors occur.
func (c *Client) Stderr() string {
	return c.stderr.String()
}

// truncate returns the first n bytes of b as a string, appending
// "..." if truncated.
func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
