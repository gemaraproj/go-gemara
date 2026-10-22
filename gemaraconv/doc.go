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
// pattern "gemara:<identity-fields>/<entity-type>". Document-root UUIDs use
// metadata.Id + version (change-on-write). Assembly UUIDs use identity fields
// only (per-subject). See docs/dev/deterministic-uuids.md for the full mapping.
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
