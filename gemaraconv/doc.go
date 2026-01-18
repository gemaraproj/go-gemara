// Package gemaraconv provides conversion functions to transform Gemara documents
// into various standard formats.
//
// Primary API (strconv-style):
//   - Direct functions: ToSARIF(), CatalogToOSCAL(), GuidanceToOSCAL()
//
// Fluent Wrappers (for IDE discoverability):
//   - EvaluationLog(), Catalog(), GuidanceDocument()
//   - Thin wrappers that delegate to the primary functions
//
// Examples:
//
//	sarifBytes, err := gemaraconv.ToSARIF(&log, "file.md", catalog)
//	oscalCatalog, err := gemaraconv.CatalogToOSCAL(catalog, gemaraconv.WithVersion("1.0"))
//	converter := gemaraconv.EvaluationLog(&log)
//	sarifBytes, err := converter.ToSARIF("file.md", catalog)
package gemaraconv
