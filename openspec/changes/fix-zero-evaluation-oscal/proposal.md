# Proposal

## Why

Evaluation logs with no control evaluations currently convert successfully but serialize an empty `include-controls` array that violates OSCAL 1.1.3. Fixing upstream issue https://github.com/gemaraproj/go-gemara/issues/155 lets downstream consumers export valid reports for targets with no assessments.

## What Changes

- Omit `include-controls` when an evaluation log has no evaluated controls.
- Preserve control selectors and existing output behavior for non-empty evaluation logs.
- Add regression coverage for nil and empty evaluation slices, validating the serialized assessment-results document against OSCAL 1.1.3.

## Capabilities

### New Capabilities

- `oscal-assessment-results`: Conversion of Gemara evaluation logs into OSCAL assessment-results documents, including empty evaluation results.

### Modified Capabilities

None. There are no existing OpenSpec capability specs in this repository.

## Impact

- Affects `gemaraconv/assessment_results.go` and `gemaraconv/assessment_results_test.go`.
- Keeps the public converter signatures and OSCAL version unchanged.
- Uses the existing OSCAL types and schema-validation utilities; no new dependencies are needed.
- Downstream consumers such as complyctl can export zero-evaluation logs without receiving schema-invalid output.
