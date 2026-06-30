## ADDED Requirements

### Requirement: Empty-patterns guard in resolvePackagePaths

`resolvePackagePaths` MUST return an empty slice and nil error when called with an empty or nil patterns slice. It MUST NOT call `packages.Load` when there are no patterns to resolve.

#### Scenario: Empty slice input
- **GIVEN** `resolvePackagePaths` is called with `patterns = []string{}` and a valid module directory
- **WHEN** the function executes
- **THEN** it MUST return `(nil, nil)` without invoking `packages.Load`

#### Scenario: Nil slice input
- **GIVEN** `resolvePackagePaths` is called with `patterns = nil` and a valid module directory
- **WHEN** the function executes
- **THEN** it MUST return `(nil, nil)` without invoking `packages.Load`

### Requirement: Symmetric test coverage for crap package

The `internal/crap/contract_test.go` file MUST include a `TestResolvePackagePaths_EmptyPatterns` test that verifies the same empty-input contract as the existing test in `internal/aireport/runner_steps_test.go`.

#### Scenario: crap package empty-patterns test
- **GIVEN** `resolvePackagePaths` in `internal/crap/contract.go` is called with an empty slice
- **WHEN** the test executes
- **THEN** the result MUST be an empty slice with nil error

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
