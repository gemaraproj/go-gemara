# Spec Delta

## Purpose

Define the OSCAL assessment-results output that consumers receive when converting Gemara evaluation logs, including logs with no control evaluations.

## ADDED Requirements

### Requirement: Schema-valid zero-evaluation results

The converter SHALL successfully convert an otherwise valid evaluation log with zero control evaluations into an assessment-results document whose serialized JSON validates against OSCAL 1.1.3. The output SHALL retain a result describing the evaluation and its target without inventing evaluated controls or findings.

#### Scenario: Nil evaluation collection

- **WHEN** an evaluation log with valid metadata and target data has a nil evaluation collection and is converted to OSCAL assessment-results
- **THEN** conversion succeeds and the serialized document validates against OSCAL 1.1.3
- **AND** the output contains a result and its target inventory item without evaluated controls or findings

#### Scenario: Empty evaluation collection

- **WHEN** an evaluation log with valid metadata and target data has an explicitly empty evaluation collection and is converted to OSCAL assessment-results
- **THEN** conversion succeeds and the serialized document validates against OSCAL 1.1.3
- **AND** the output contains a result and its target inventory item without evaluated controls or findings

### Requirement: Omit empty include-controls

The serialized assessment-results document SHALL omit `include-controls` when no evaluated control identifiers exist. It MUST NOT serialize an empty array, null value, or synthetic include-all selection in place of absent evaluated controls.

#### Scenario: No evaluated controls

- **WHEN** a zero-evaluation log is converted and serialized
- **THEN** its reviewed-control selection omits the `include-controls` property
- **AND** no include-all selection is added

### Requirement: Preserve populated control selections

For otherwise valid logs with evaluated controls, the converter SHALL retain the existing control identifiers in `include-controls` and produce serialized JSON that validates against OSCAL 1.1.3.

#### Scenario: Evaluated control remains selected

- **WHEN** a log with an evaluation of control `CTRL-1` is converted and serialized
- **THEN** `include-controls` contains a selector for `CTRL-1`
- **AND** the serialized document validates against OSCAL 1.1.3
