# Design

## Context

See `proposal.md` for the motivation and upstream issue. `buildReviewedControls` in `gemaraconv/assessment_results.go` always takes the address of its selector slice, including when the slice is empty. The OSCAL Go model therefore serializes `include-controls: []` rather than omitting the optional property.

The converter already retains one result and the target inventory item for empty logs. Tests in `gemaraconv/assessment_results_test.go` use `assertValidOSCAL`, which marshals an OSCAL document, round-trips the JSON, and calls the existing OSCAL validator. The validator uses the project's pinned OSCAL 1.1.3 model. The existing helper validates the original model rather than the round-tripped document.

## Goals / Non-Goals

**Goals:**

- Fix the optional-field representation at the reviewed-controls construction boundary.
- Test serialized field presence separately from schema validity so a round-trip cannot hide an empty-array regression.
- Keep the existing populated-result conversion path and public API intact.

**Non-Goals:**

- Reject empty logs, suppress their result, or fabricate control evaluations.
- Change OSCAL versions, converter options, or downstream complyctl behavior.
- Add runtime schema validation or new dependencies.

## Decisions

### Use a nil optional selector pointer for zero controls

Build selectors as today, but assign `IncludeControls` only when selectors are present, using the existing `oscalUtils.NilIfEmpty` convention where appropriate. Retain the required `ControlSelections` collection with one selection object; its optional `include-controls` property is absent for zero controls.

Returning an error would reject a valid input state. Using `include-all` would claim a scope the log does not establish. Removing the required control-selection collection would risk a different schema violation. The regression test must validate the complete document to verify the empty selection representation against OSCAL 1.1.3.

### Exercise the public converter with nil and empty slices

Use valid metadata and target fixtures from the existing converter tests, varying only `Evaluations` between nil and an explicitly empty slice. Check successful conversion, retained result and target data, absent findings, and nil `IncludeControls`. Decode the serialized JSON into a generic object to assert that `include-controls` is absent rather than null or empty and that no `include-all` property is introduced.

Direct helper tests alone would not establish that the public converter emits a valid complete document. Raw string comparisons would be brittle around generated UUIDs and timestamps.

### Reuse schema validation on the serialized document

Strengthen `assertValidOSCAL` to validate its already decoded round-trip model through `oscalUtils.Validate`. Pair it with explicit serialized-key assertions in the empty-log regression cases. Retain the existing populated-log test that checks `CTRL-1` and schema validity.

This uses the existing validator and model wrapper rather than adding a second schema tool or network-fetched fixture. Confirm the new empty-log cases fail before the production fix and pass afterward.

## Risks / Trade-offs

- [A Go zero value could still emit an invalid required structure] → Validate complete serialized nil/empty cases against OSCAL 1.1.3, not just pointer values.
- [A fix could accidentally remove populated selectors] → Keep the existing `CTRL-1` assertions and run the complete converter test suite.
- [An invalid test fixture could mask the reported defect] → Reuse valid metadata and target data; confirm the pre-fix failure identifies the empty `include-controls` array.

## Migration Plan

No API or data migration is required. Ship the converter fix and regression tests together through the normal release process; downstream consumers can then update their go-gemara revision. Reverting the focused change restores the previous serialization behavior if necessary.
