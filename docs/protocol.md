# Gaze Analyzer Protocol Specification

**Version**: 1.0.0
**Transport**: JSON-RPC 2.0 over stdin/stdout

## Overview

The Gaze Analyzer Protocol enables external language analyzers to provide side effect detection, complexity analysis, and coverage data to Gaze. Gaze spawns the analyzer as a subprocess, communicates via JSON-RPC 2.0 over stdin/stdout, and uses the responses to compute CRAP scores, GazeCRAP scores, quadrant classifications, and fix strategies.

This protocol follows the same transport model as the Language Server Protocol (LSP): line-delimited JSON over stdin/stdout, with stderr reserved for diagnostics.

## Lifecycle

```text
gaze crap --analyzer snake-eyes ./src
|
+-- 1. Discover analyzer binary (CLI flag / .gaze.yaml / PATH)
+-- 2. Spawn: snake-eyes --stdio
+-- 3. initialize --> capabilities handshake
+-- 4. discover --> find source + test files (optional)
+-- 5. analyze --> detect side effects per function
+-- 6. complexity --> cyclomatic complexity per function
+-- 7. coverage --> line coverage per function
+-- 8. test_mapping --> map assertions to effects (optional)
+-- 9. classify_signals --> classification signals (optional)
+-- 10. shutdown --> clean exit
+-- 11. Gaze computes CRAP, GazeCRAP, quadrants, fix strategies
```

### Discovery

Gaze finds analyzer binaries using a three-tier mechanism:

1. **CLI flag**: `--analyzer <binary>` -- explicit binary name or path
2. **Config file**: `.gaze.yaml` `analyzers.<language>.command`
3. **PATH convention**: `gaze-analyzer-<language>` on PATH

The `--language` flag determines which language key to look up in tiers 2 and 3. When `--analyzer` is specified directly, `--language` is optional.

## Message Format

### Request

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "analyze",
  "params": {
    "root_path": "/path/to/project",
    "patterns": ["./..."]
  }
}
```

### Response (success)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": { ... }
}
```

### Response (error)

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "error": {
    "code": -32603,
    "message": "Internal error",
    "data": "optional details"
  }
}
```

### Error Codes

Standard JSON-RPC 2.0 error codes:

| Code | Meaning |
|------|---------|
| -32700 | Parse error |
| -32600 | Invalid request |
| -32601 | Method not found |
| -32602 | Invalid params |
| -32603 | Internal error |

## Methods

### Required Methods

Every analyzer **must** implement these 5 methods.

---

### `initialize`

Handshake method. Must be the first method called. Returns the analyzer's capabilities and identity.

**Timeout**: 30 seconds

**Request params**:

```json
{
  "root_path": "/absolute/path/to/project",
  "config": {}
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `root_path` | string | yes | Absolute path to the project root |
| `config` | object | no | Analyzer-specific config from `.gaze.yaml` |

**Response result**:

```json
{
  "capabilities": {
    "discover": true,
    "test_mapping": true,
    "classify_signals": false
  },
  "protocol_version": "1.0.0",
  "analyzer_name": "snake-eyes",
  "language": "python"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `capabilities.discover` | boolean | Supports the `discover` method |
| `capabilities.test_mapping` | boolean | Supports the `test_mapping` method |
| `capabilities.classify_signals` | boolean | Supports the `classify_signals` method |
| `protocol_version` | string | Protocol version (semver) |
| `analyzer_name` | string | Human-readable analyzer name |
| `language` | string | Primary language (e.g., "python", "rust") |

---

### `analyze`

Detect side effects for all functions in the project.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "name": "divide",
      "package": "math_utils",
      "file": "math_utils/ops.py",
      "line": 20,
      "side_effects": [
        {
          "type": "ReturnValue",
          "description": "returns division result",
          "location": "math_utils/ops.py:25:5",
          "target": "result",
          "classification": {
            "label": "contractual",
            "confidence": 90
          }
        },
        {
          "type": "ErrorReturn",
          "description": "raises ZeroDivisionError",
          "location": "math_utils/ops.py:22:9",
          "target": "ZeroDivisionError",
          "classification": {
            "label": "contractual",
            "confidence": 85
          }
        }
      ]
    }
  ]
}
```

#### Side Effect Types

Analyzers map their language concepts to Gaze's taxonomy. Common mappings:

| Gaze Type | Python | Rust | Description |
|-----------|--------|------|-------------|
| `ReturnValue` | `return`, `yield` | `return`, `Ok(v)` | Function returns a value |
| `ErrorReturn` | `raise`, exception | `Err(e)`, `panic!` | Function signals an error |
| `ReceiverMutation` | `self.x = ...` | `&mut self` | Mutates the receiver/self |
| `PointerArgMutation` | mutable arg | `&mut` param | Mutates a parameter |
| `GlobalMutation` | module-level write | `static mut` | Modifies global state |
| `FileSystemWrite` | `open().write()` | `fs::write()` | Writes to filesystem |
| `FileSystemRead` | `open().read()` | `fs::read()` | Reads from filesystem |
| `NetworkCall` | `requests.get()` | `reqwest::get()` | Makes network request |
| `GoroutineSpawn` | `threading.Thread` | `tokio::spawn` | Spawns concurrent task |

The full taxonomy is defined in `internal/taxonomy/types.go`. Unknown types default to P4 tier with a warning.

#### Classification

Each side effect may include a `classification` object:

| Field | Type | Description |
|-------|------|-------------|
| `label` | string | `"contractual"`, `"incidental"`, or `"ambiguous"` |
| `confidence` | integer | 0-100 confidence score |

When `classification` is null, Gaze uses default classification based on the effect type's tier.

---

### `complexity`

Compute cyclomatic complexity for all functions.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "name": "divide",
      "package": "math_utils",
      "file": "math_utils/ops.py",
      "line": 20,
      "complexity": 5
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `name` | string | Function or method name |
| `package` | string | Package/module path |
| `file` | string | Source file path (relative or absolute) |
| `line` | integer | Line number of function declaration |
| `complexity` | integer | Cyclomatic complexity value |

---

### `coverage`

Produce per-function line coverage data. The analyzer is responsible for running tests and collecting coverage internally.

**Timeout**: 5 minutes

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "functions": [
    {
      "file": "math_utils/ops.py",
      "function": "divide",
      "start_line": 20,
      "end_line": 30,
      "covered_stmts": 0,
      "total_stmts": 10,
      "percentage": 0.0
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `file` | string | Source file path |
| `function` | string | Function or method name |
| `start_line` | integer | Function declaration start line |
| `end_line` | integer | Function body end line |
| `covered_stmts` | integer | Number of statements covered by tests |
| `total_stmts` | integer | Total number of statements |
| `percentage` | number | Coverage percentage (0.0-100.0) |

---

### `shutdown`

Request the analyzer to shut down cleanly. The analyzer should finish any pending work and exit.

**Timeout**: 10 seconds

**Request params**: none (or `null`)

**Response result**: `{}`

After receiving the shutdown response, Gaze closes stdin and waits for the process to exit.

---

### Optional Methods

These methods are declared via capabilities in the `initialize` response. Gaze only calls them when the corresponding capability is `true`.

---

### `discover` (optional)

Find source and test files in the project. Reserved for future use -- currently not consumed by Gaze's provider interfaces.

**Capability**: `discover`

**Request params**:

```json
{
  "root_path": "/path/to/project"
}
```

**Response result**:

```json
{
  "source_files": ["math_utils/ops.py", "math_utils/helpers.py"],
  "test_files": ["tests/test_ops.py"],
  "framework": "pytest"
}
```

---

### `test_mapping` (optional)

Map test assertions to side effects. When supported, this enables GazeCRAP scoring (contract coverage).

**Capability**: `test_mapping`

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "mappings": [
    {
      "test_function": "test_multiply",
      "test_file": "tests/test_ops.py",
      "assertion_location": "tests/test_ops.py:10",
      "assertion_type": "equality",
      "target_function": "multiply",
      "target_package": "math_utils",
      "side_effect_type": "ReturnValue",
      "confidence": 80
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `test_function` | string | Test function name |
| `test_file` | string | Test file path |
| `assertion_location` | string | Source position of the assertion |
| `assertion_type` | string | Kind of assertion (e.g., "equality", "error_check") |
| `target_function` | string | Function under test |
| `target_package` | string | Package of the function under test |
| `side_effect_type` | string | Type of side effect being asserted on |
| `confidence` | integer | Mapping confidence (0-100) |

---

### `classify_signals` (optional)

Provide raw classification signals for Gaze's scoring engine. When supported, these signals are fed into `classify.ComputeScore` alongside Gaze's built-in signals.

**Capability**: `classify_signals`

**Request params**:

```json
{
  "root_path": "/path/to/project",
  "patterns": ["./..."]
}
```

**Response result**:

```json
{
  "signals": [
    {
      "function": "divide",
      "package": "math_utils",
      "side_effect_type": "ErrorReturn",
      "source": "docstring",
      "weight": 15,
      "reasoning": "docstring mentions ZeroDivisionError"
    }
  ]
}
```

| Field | Type | Description |
|-------|------|-------------|
| `function` | string | Function name |
| `package` | string | Package path |
| `side_effect_type` | string | Side effect type this signal relates to |
| `source` | string | Signal type (e.g., "docstring", "type_annotation", "decorator") |
| `weight` | integer | Numeric contribution to confidence score |
| `reasoning` | string | (optional) Explanation of why this signal was applied |

## Error Handling

### Required method errors

When a required method (`analyze`, `complexity`, `coverage`) returns a JSON-RPC error, Gaze propagates the error and exits non-zero. The error message is displayed to the user.

### Optional method errors

When an optional method (`discover`, `test_mapping`, `classify_signals`) returns an error, Gaze logs a warning to stderr and degrades gracefully:

- `discover` error: no impact (not currently consumed)
- `test_mapping` error: GazeCRAP is unavailable
- `classify_signals` error: uses pre-classified effects from `analyze`

### Process crashes

If the analyzer process exits unexpectedly (crash, segfault), Gaze detects the closed stdin/stdout pipes and returns an error with the process's stderr output for diagnostics.

### Timeouts

Each method has a default timeout (30s for handshake methods, 5min for analysis methods). When the timeout expires, Gaze kills the subprocess and returns a timeout error.

## Configuration

### `.gaze.yaml`

```yaml
analyzers:
  python:
    command: snake-eyes
    args: ["--stdio"]
  rust:
    command: /usr/local/bin/gaze-analyzer-rust
    args: ["--stdio", "--edition=2021"]
```

### CLI Flags

| Flag | Description |
|------|-------------|
| `--analyzer <binary>` | Explicit analyzer binary name or path |
| `--language <lang>` | Target language for config/PATH discovery |

## Building an Analyzer

To build a Gaze-compatible analyzer:

1. **Accept `--stdio` flag**: Read JSON-RPC requests from stdin, write responses to stdout, diagnostics to stderr.
2. **Implement the 5 required methods**: `initialize`, `analyze`, `complexity`, `coverage`, `shutdown`.
3. **Declare capabilities**: In the `initialize` response, set `test_mapping: true` if you can map assertions to effects (enables GazeCRAP).
4. **Map to Gaze's taxonomy**: Use Gaze's `SideEffectType` constants for the `type` field in `analyze` responses.
5. **Follow naming convention**: Name your binary `gaze-analyzer-<language>` for automatic PATH discovery.

### Minimal Example (Python)

```python
#!/usr/bin/env python3
"""Minimal Gaze analyzer for demonstration."""
import json
import sys

def handle(request):
    method = request["method"]
    rid = request["id"]

    if method == "initialize":
        return {"jsonrpc": "2.0", "id": rid, "result": {
            "capabilities": {"discover": False, "test_mapping": False, "classify_signals": False},
            "protocol_version": "1.0.0",
            "analyzer_name": "minimal-python",
            "language": "python"
        }}
    elif method == "analyze":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "complexity":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "coverage":
        return {"jsonrpc": "2.0", "id": rid, "result": {"functions": []}}
    elif method == "shutdown":
        return {"jsonrpc": "2.0", "id": rid, "result": {}}
    else:
        return {"jsonrpc": "2.0", "id": rid, "error": {
            "code": -32601, "message": f"Method not found: {method}"
        }}

for line in sys.stdin:
    request = json.loads(line.strip())
    response = handle(request)
    print(json.dumps(response), flush=True)
    if request["method"] == "shutdown":
        break
```

### Testing Your Analyzer

Use Gaze's fake analyzer as a reference implementation: `internal/protocol/testdata/fake_analyzer/main.go`.

```bash
# Test with gaze crap
gaze crap --analyzer ./my-analyzer --language python ./src

# Test with JSON output (no AI adapter needed)
gaze report --analyzer ./my-analyzer --format=json ./src
```
