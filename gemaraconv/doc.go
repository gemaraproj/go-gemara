// Package gemaraconv provides conversion functions to transform Gemara documents
// into various standard formats.
//
// Primary API (strconv-style):
//   - Direct functions: ToSARIF(), CatalogToOSCAL(), CatalogToMarkdown(), GuidanceToOSCAL(), GuidanceToMarkdown()
//
// Fluent Wrappers (for IDE discoverability):
//   - EvaluationLog(), ControlCatalog(), GuidanceCatalog()
//   - Thin wrappers that delegate to the primary functions
//
// # UUID Generation
//
// OSCAL conversions produce deterministic UUIDv5 values (SHA-1, URL namespace)
// via go-oscal's uuid.NewUUIDWithSource. Each UUID source string follows the
// pattern "gemara:<escaped-identity-fields>/<entity-type>", where identity fields
// are URL-encoded with url.PathEscape to prevent separator ambiguity.
// Catalog and Profile roots use options.version (defaulting to
// metadata.Version, then "0.0.0"); the AssessmentResults root uses
// metadata.Version with the same fallback, and it and its Result also include
// metadata.Date so each run produces fresh values.
// Assessment assembly UUIDs include the log ID for per-document uniqueness, and
// assessment logs sharing a requirement are distinguished by an occurrence
// counter. See docs/dev/deterministic-uuids.md for the
// full mapping.
//
// Examples:
//
//	sarifBytes, err := gemaraconv.ToSARIF(log, gemaraconv.WithArtifactURI("file.md"), gemaraconv.WithCatalog(catalog))
//	oscalCatalog, err := gemaraconv.CatalogToOSCAL(catalog, gemaraconv.WithVersion("1.0"))
//	md, err := gemaraconv.CatalogToMarkdown(ctx, catalog, gemaraconv.WithTOC(true), gemaraconv.WithLexiconAutolink(&fetcher.URI{}))
//	// Or pass list-shaped entries: WithInlineLexicon([]gemaraconv.InlineLexiconTerm{...})
//	converter := gemaraconv.EvaluationLog(log)
//	sarifBytes, err := converter.ToSARIF(gemaraconv.WithArtifactURI("file.md"), gemaraconv.WithCatalog(catalog))
//	md, err := gemaraconv.ControlCatalog(catalog).ToMarkdown(ctx)
package gemaraconv
