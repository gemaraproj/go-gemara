// Package gemaraconv provides conversion functions to transform Gemara documents
// into various standard formats.
//
// OSCAL (Open Security Controls Assessment Language) conversion:
// OSCAL is a set of standardized formats for expressing security controls,
// assessments, and related information in a machine-readable format. This package
// supports converting:
//
//   - Layer 1 GuidanceDocument to OSCAL Profile and Catalog
//   - Layer 2 Catalog to OSCAL Catalog
//
// SARIF (Static Analysis Results Interchange Format) conversion:
// SARIF is a standard format for static analysis tool output, enabling
// integration with code scanning platforms like GitHub Code Scanning,
// Azure DevOps, and other security analysis tools.
//
// This package converts EvaluationLog entries into SARIF v2.1.0 format,
// where each AssessmentLog becomes a SARIF result. The conversion supports
// optional catalog enrichment to include control and requirement details
// in the SARIF output.
package gemaraconv
