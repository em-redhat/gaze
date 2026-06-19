## 1. Test Fixtures

- [x] 1.1 Create `internal/config/testdata/out-of-range-contractual.yaml` with `classification.thresholds.contractual: 500` and valid `incidental: 50`
- [x] 1.2 Create `internal/config/testdata/out-of-range-incidental.yaml` with `classification.thresholds.incidental: 200` and valid `contractual: 80`
- [x] 1.3 Create `internal/config/testdata/zero-contractual.yaml` with `classification.thresholds.contractual: 0` and valid `incidental: 50`
- [x] 1.4 Create `internal/config/testdata/negative-incidental.yaml` with `classification.thresholds.incidental: -10` and valid `contractual: 80`
- [x] 1.5 Create `internal/config/testdata/inverted-thresholds.yaml` with `classification.thresholds.contractual: 40` and `incidental: 60`
- [x] 1.6 Create `internal/config/testdata/equal-thresholds.yaml` with `classification.thresholds.contractual: 50` and `incidental: 50`
- [x] 1.7 Create `internal/config/testdata/boundary-thresholds.yaml` with `classification.thresholds.contractual: 99` and `incidental: 1`
- [x] 1.8 Create `internal/config/testdata/adjacent-thresholds.yaml` with `classification.thresholds.contractual: 51` and `incidental: 50`
- [x] 1.9 Create `internal/config/testdata/negative-contractual.yaml` with `classification.thresholds.contractual: -10` and valid `incidental: 50`
- [x] 1.10 Create `internal/config/testdata/zero-incidental.yaml` with `classification.thresholds.incidental: 0` and valid `contractual: 80`

## 2. Validation Logic

- [x] 2.1 Add contractual range check to `config.Load` (`internal/config/config.go`): after baseline validation (line 136), check `cfg.Classification.Thresholds.Contractual` is in `[1, 99]`; return `fmt.Errorf("classification.thresholds.contractual must be in [1, 99], got %d", ct)` if not
- [x] 2.2 Add incidental range check to `config.Load`: check `cfg.Classification.Thresholds.Incidental` is in `[1, 99]`; return `fmt.Errorf("classification.thresholds.incidental must be in [1, 99], got %d", it)` if not
- [x] 2.3 Add coherence check to `config.Load`: after both range checks pass, check `contractual > incidental`; return `fmt.Errorf("classification.thresholds.contractual (%d) must be greater than incidental (%d)", ct, it)` if not

## 3. Unit Tests

- [x] 3.1 Add `TestLoad_ContractualOutOfRange` in `internal/config/config_test.go`: load `out-of-range-contractual.yaml`, assert error contains `classification.thresholds.contractual must be in [1, 99], got 500`
- [x] 3.2 Add `TestLoad_IncidentalOutOfRange`: load `out-of-range-incidental.yaml`, assert error contains `classification.thresholds.incidental must be in [1, 99], got 200`
- [x] 3.3 Add `TestLoad_ZeroContractual`: load `zero-contractual.yaml`, assert error contains `classification.thresholds.contractual must be in [1, 99], got 0`
- [x] 3.4 Add `TestLoad_NegativeIncidental`: load `negative-incidental.yaml`, assert error contains `classification.thresholds.incidental must be in [1, 99], got -10`
- [x] 3.5 Add `TestLoad_InvertedThresholds`: load `inverted-thresholds.yaml`, assert error contains `classification.thresholds.contractual (40) must be greater than incidental (60)`
- [x] 3.6 Add `TestLoad_EqualThresholds`: load `equal-thresholds.yaml`, assert error contains `classification.thresholds.contractual (50) must be greater than incidental (50)`
- [x] 3.7 Add `TestLoad_BoundaryValid`: load `boundary-thresholds.yaml`, assert no error, `Contractual == 99`, `Incidental == 1`
- [x] 3.8 Add `TestLoad_AdjacentValid`: load `adjacent-thresholds.yaml`, assert no error, `Contractual == 51`, `Incidental == 50`
- [x] 3.9 Add `TestLoad_NegativeContractual`: load `negative-contractual.yaml`, assert error contains `classification.thresholds.contractual must be in [1, 99], got -10`
- [x] 3.10 Add `TestLoad_ZeroIncidental`: load `zero-incidental.yaml`, assert error contains `classification.thresholds.incidental must be in [1, 99], got 0`

## 4. Verification

- [x] 4.1 Run `go test -race -count=1 ./internal/config/...` and confirm all tests pass
- [x] 4.2 Run `go test -race -count=1 -short ./...` and confirm no regressions across the full module
- [x] 4.3 Run `golangci-lint run` and confirm no lint violations
- [x] 4.4 Update `TestLoadConfig_YAMLInvertedThresholdsRejected` in `cmd/gaze/main_test.go`: after this change, `config.Load` returns the coherence error before `loadConfig`'s own coherence check runs (line 154-156), so the error message format changes from `"config file"` to the `config.Load` format (`classification.thresholds.contractual`). Update the test assertion on line 653 from `strings.Contains(err.Error(), "config file")` to `strings.Contains(err.Error(), "classification.thresholds.contractual")` and verify the test passes

## 5. Constitution Alignment Verification

- [x] 5.1 Verify Principle I (Accuracy): confirm that invalid thresholds are rejected before they can corrupt classification output
- [x] 5.2 Verify Principle II (Minimal Assumptions): confirm no new assumptions introduced; validation enforces the same constraints already documented in the CLI path
- [x] 5.3 Verify Principle III (Actionable Output): confirm all error messages name the invalid field, show the received value, and state the valid range
- [x] 5.4 Verify Principle IV (Testability): confirm all validation branches are covered by dedicated test cases with isolated fixture files
<!-- spec-review: passed -->
<!-- code-review: passed -->
