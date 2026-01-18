package gemaraconv

import (
	oscal "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/gemaraproj/go-gemara"
)

// EvaluationLogConverter define a converter object for converting EvaluationLog.
type EvaluationLogConverter struct {
	log *gemara.EvaluationLog
}

// EvaluationLog creates a new EvaluationLogConverter struct.
func EvaluationLog(log *gemara.EvaluationLog) *EvaluationLogConverter {
	return &EvaluationLogConverter{log: log}
}

// ToSARIF converts the EvaluationLog to SARIF format.
func (c *EvaluationLogConverter) ToSARIF(artifactURI string, catalog *gemara.Catalog) ([]byte, error) {
	return ToSARIF(*c.log, artifactURI, catalog)
}

// CatalogConverter defines a converter for converting Catalog.
type CatalogConverter struct {
	catalog *gemara.Catalog
}

// Catalog creates a new CatalogConverter struct.
func Catalog(catalog *gemara.Catalog) *CatalogConverter {
	return &CatalogConverter{catalog: catalog}
}

// ToOSCAL converts the Catalog to OSCAL format.
func (c *CatalogConverter) ToOSCAL(opts ...GenerateOption) (oscal.Catalog, error) {
	return CatalogToOSCAL(c.catalog, opts...)
}

// GuidanceDocumentConverter defines a converter for converting GuidanceDocument.
type GuidanceDocumentConverter struct {
	guidance *gemara.GuidanceDocument
}

// GuidanceDocument creates a new GuidanceDocumentConverter struct.
func GuidanceDocument(guidance *gemara.GuidanceDocument) *GuidanceDocumentConverter {
	return &GuidanceDocumentConverter{guidance: guidance}
}

// ToOSCAL converts the GuidanceDocument to an OSCAL Catalog and Profile.
func (c *GuidanceDocumentConverter) ToOSCAL(guidanceDocHref string, opts ...GenerateOption) (oscal.Catalog, oscal.Profile, error) {
	return GuidanceToOSCAL(c.guidance, guidanceDocHref, opts...)
}
