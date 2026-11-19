# Migrating to v1.5.0

This document lists every breaking or behavioral change in v1.5.0 that
affects downstream consumers of this library, with the replacement for each
removed or changed API.

## gemaraconv: OSCAL UUIDs are now deterministic

Every OSCAL UUID emitted by the converters switches from random v4
(`uuid.NewUUID()`) to deterministic v5 (`uuid.NewUUIDWithSource()`), derived
from each assembly's identity fields. On upgrade, every UUID value in every
converted document changes: there is zero overlap between old and new
output for consumers that persist or correlate findings, observations,
parties, or resources by UUID. Key on stable document identity
(`metadata.id`) instead of the emitted UUIDs for cross-version handles. The
`AssessmentResults` root and `Result` UUIDs also rotate with the assessment
date (`metadata.Date`, or the conversion date when empty) rather than being
arbitrary per run. The full source-string mapping is documented in
[docs/dev/deterministic-uuids.md](deterministic-uuids.md).

## bundle: single-source assembly

`bundle.Bundle.Files` (a `[]File` of primary artifacts) is replaced by
`bundle.Bundle.Source` (a single `File`). The `Assemble` signature changed
to match: the variadic `sources ...File` parameter is now `source File`
followed by options.

```go
// before
b, err := asm.Assemble(ctx, m, fileA, fileB)

// after
b, err := asm.Assemble(ctx, m, sourceFile)
```

A bundle now has exactly one source artifact; everything else it contains is
an import resolved from that source's extends and imports.

## bundle: Assemble fails on unmatched URL-less mapping references

`Assemble` now returns an error when a URL-less mapping-reference matches no
artifact `metadata.id` in the assembled set. Such a reference previously
assembled silently and produced a bundle whose back-matter could not resolve
it. Pass `WithContinueOnError()` to collect these as `Bundle.Warnings`
instead of failing:

```go
b, err := asm.Assemble(ctx, m, source, bundle.WithContinueOnError())
```

## bundle: Unpack rejects multi-source bundles

`Unpack` returns an error ("bundle contains multiple source artifacts")
when a bundle carries more than one source layer. Legacy multi-source
bundles must be re-assembled from their original sources with a single
source file and repacked.

## gemaraconv: WithLexiconAutolink takes a Fetcher

`WithLexiconAutolink(bool)` is now `WithLexiconAutolink(gemara.Fetcher)`,
and it panics when passed a nil Fetcher. The lexicon document is fetched
through the provided Fetcher instead of relying on ambient configuration:

```go
md, err := gemaraconv.CatalogToMarkdown(ctx, catalog,
    gemaraconv.WithLexiconAutolink(&fetcher.URI{}))
```

Because the Fetcher now carries the HTTP client, SSRF posture is the
caller's choice — see `fetcher.ExampleURI_ssrfSafe` for a dialer that
blocks private, loopback, and link-local addresses. Users of the lower-level
`gemaraconv/markdown` package must likewise set `Config.Fetcher` to enable
autolinking.

## gemara: SchemaVersion is a const

`gemara.SchemaVersion` changed from a `var` to a `const` and now reports
`v1.5.0`. Code that assigned to it at runtime no longer compiles; the
supported schema version is fixed per release of this library.

## gemaraconv: EvaluationLogToOSCALAssessmentResults returns errors

The conversion now returns an error when the log's `metadata.id` or
`target.id` is empty, when two control evaluations share the same control
mapping (reference id plus entry id), or when mapping references conflict on
title or URL for the same ID. Each condition would break the deterministic
UUID scheme or the generated back-matter. Logs that previously converted
silently may now require these fields to be filled in first.

## gemara: Evaluate runs every assessment

`ControlEvaluation.Evaluate` no longer halts after the first Failed
assessment: every applicable assessment runs and rolls into the control's
aggregate result (Failed still wins the rollup). Assessments whose
applicability matches none of the requested values are recorded as
`NotApplicable` with a message naming both lists, instead of being left at
`NotRun`.

## YAML/JSON decoding ignores unknown fields

`Load`, `Decode`, and the bundle loaders now ignore unknown fields in YAML
and JSON input. Artifacts written against a newer Gemara schema decode into
the types this version supports; unknown keys are dropped rather than
preserved, so re-serializing writes only the schema version this library is
at. Strict-field consumers should validate input separately.
