## ADDED Requirements

### Requirement: Contractual threshold range validation

`config.Load` MUST reject a `.gaze.yaml` where `classification.thresholds.contractual` is less than 1 or greater than 99. The returned error MUST identify the field name and the invalid value.

#### Scenario: Contractual threshold above upper bound
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 500`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.contractual must be in [1, 99], got 500`

#### Scenario: Contractual threshold at zero
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 0`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.contractual must be in [1, 99], got 0`

#### Scenario: Contractual threshold negative
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: -10`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.contractual must be in [1, 99], got -10`

### Requirement: Incidental threshold range validation

`config.Load` MUST reject a `.gaze.yaml` where `classification.thresholds.incidental` is less than 1 or greater than 99. The returned error MUST identify the field name and the invalid value.

#### Scenario: Incidental threshold above upper bound
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.incidental: 200`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.incidental must be in [1, 99], got 200`

#### Scenario: Incidental threshold at zero
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.incidental: 0`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.incidental must be in [1, 99], got 0`

#### Scenario: Incidental threshold negative
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.incidental: -10`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.incidental must be in [1, 99], got -10`

### Requirement: Threshold coherence validation

`config.Load` MUST reject a `.gaze.yaml` where `classification.thresholds.contractual` is less than or equal to `classification.thresholds.incidental`. The returned error MUST include both values.

#### Scenario: Inverted thresholds
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 40` and `classification.thresholds.incidental: 60`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.contractual (40) must be greater than incidental (60)`

#### Scenario: Equal thresholds
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 50` and `classification.thresholds.incidental: 50`
- **WHEN** `config.Load` is called
- **THEN** it MUST return an error containing `classification.thresholds.contractual (50) must be greater than incidental (50)`

### Requirement: Valid boundary thresholds accepted

`config.Load` MUST accept threshold values at the boundaries of the valid range when coherence is satisfied.

#### Scenario: Maximum contractual with minimum incidental
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 99` and `classification.thresholds.incidental: 1`
- **WHEN** `config.Load` is called
- **THEN** it MUST return a valid `*GazeConfig` with no error

#### Scenario: Adjacent valid thresholds
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 51` and `classification.thresholds.incidental: 50`
- **WHEN** `config.Load` is called
- **THEN** it MUST return a valid `*GazeConfig` with no error

### Requirement: Validation order

Range validation MUST be performed before coherence validation. If a threshold is out of range, the range error MUST be returned without also checking coherence.

#### Scenario: Out-of-range contractual with valid incidental
- **GIVEN** a `.gaze.yaml` with `classification.thresholds.contractual: 500` and `classification.thresholds.incidental: 50`
- **WHEN** `config.Load` is called
- **THEN** it MUST return the range error for `contractual`, not a coherence error

## MODIFIED Requirements

None.

## REMOVED Requirements

None.
