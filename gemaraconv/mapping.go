// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"fmt"
	"strings"
	"time"

	"github.com/defenseunicorns/go-oscal/src/pkg/uuid"
	oscal12 "github.com/defenseunicorns/go-oscal/src/types/oscal-1-2-2"
	"github.com/gemaraproj/go-gemara"
	oscalUtils "github.com/gemaraproj/go-gemara/internal/oscal"
)

// Gemara MappingDocuments are designed to capture functional relationships
const matchingRationale = "functional"

// Mapping provenance is always emitted as draft: Gemara has no field to
// record review/approval state, so claiming anything else would be a
// fabricated audit trail.
const mappingStatusDraft = "draft"

// Provenance method values, inferred from the MappingDocument author's
// EntityType unless overridden via WithHumanMethod/WithAutomatedMethod/
// WithMachineAssistedMethod.
const (
	mappingMethodHuman           = "human"
	mappingMethodAutomated       = "automated"
	mappingMethodMachineAssisted = "machine-assisted"
)

// inferMappingMethod derives the provenance method from the MappingDocument
// author's EntityType. Unset/unrecognized author types default to human,
// since that is the safe assumption absent evidence of automation.
func inferMappingMethod(authorType gemara.EntityType) string {
	switch authorType {
	case gemara.Software:
		return mappingMethodAutomated
	case gemara.SoftwareAssisted:
		return mappingMethodMachineAssisted
	default:
		return mappingMethodHuman
	}
}

// gemaraToOSCALRelationship maps Gemara relationship types to the set-theory
// relationship tokens allowed by the OSCAL mapping model's "relationship" flag.
// Only relationships with a well-defined set-theory meaning are convertible;
// directional relationships (implements/supports and their inverses) have no
// OSCAL equivalent and are rejected rather than emitted as invalid documents.
var gemaraToOSCALRelationship = map[gemara.RelationshipType]string{
	gemara.RelEquivalent: "equivalent-to",
	gemara.RelSubsumes:   "superset-of",
	gemara.RelRelatesTo:  "intersects-with",
	gemara.RelNoMatch:    "no-relationship",
}

// gemaraRelationshipToOSCAL converts a Gemara relationship to its OSCAL mapping
// relationship token, returning an error for relationships OSCAL cannot represent.
func gemaraRelationshipToOSCAL(rel gemara.RelationshipType) (string, error) {
	oscalRel, ok := gemaraToOSCALRelationship[rel]
	if !ok {
		return "", fmt.Errorf("relationship %q has no OSCAL mapping equivalent", rel.String())
	}
	return oscalRel, nil
}

// MappingDocumentToOSCAL converts a Gemara MappingDocument to an OSCAL MappingCollection.
func MappingDocumentToOSCAL(doc gemara.MappingDocument, opts ...MappingOption) (oscal12.MappingCollection, error) {
	options := defaultMappingOpts()
	for _, opt := range opts {
		opt(&options)
	}

	if len(doc.Mappings) == 0 {
		return oscal12.MappingCollection{}, fmt.Errorf("mapping document has no mappings")
	}

	sourceRef, err := resolveMappingReference(doc.Metadata.MappingReferences, doc.SourceReference.ReferenceId)
	if err != nil {
		return oscal12.MappingCollection{}, fmt.Errorf("resolving source reference: %w", err)
	}

	targetRef, err := resolveMappingReference(doc.Metadata.MappingReferences, doc.TargetReference.ReferenceId)
	if err != nil {
		return oscal12.MappingCollection{}, fmt.Errorf("resolving target reference: %w", err)
	}

	if options.reverseDoc != nil {
		if err := validateReverseDocument(doc, *options.reverseDoc); err != nil {
			return oscal12.MappingCollection{}, fmt.Errorf("reverse mapping document: %w", err)
		}
	}

	metadata := buildMappingMetadata(doc)
	maps, err := buildMappingMaps(doc)
	if err != nil {
		return oscal12.MappingCollection{}, err
	}

	mapping := oscal12.Mapping{
		UUID: uuid.NewUUID(),
		SourceResource: oscal12.MappingResourceReference{
			Href: sourceRef.Url,
			Type: strings.ToLower(doc.SourceReference.EntryType.String()),
		},
		TargetResource: oscal12.MappingResourceReference{
			Href: targetRef.Url,
			Type: strings.ToLower(doc.TargetReference.EntryType.String()),
		},
		Maps:              maps,
		MatchingRationale: matchingRationale,
		SourceGapSummary:  buildSourceGapSummary(doc.Mappings),
		TargetGapSummary:  buildTargetGapSummary(options.reverseDoc),
	}

	method := options.methodOverride
	if method == "" {
		method = inferMappingMethod(doc.Metadata.Author.Type)
	}

	provenance := oscal12.MappingProvenance{
		Method:             method,
		Status:             mappingStatusDraft,
		MatchingRationale:  matchingRationale,
		MappingDescription: doc.Remarks,
	}

	return oscal12.MappingCollection{
		UUID:       uuid.NewUUID(),
		Metadata:   metadata,
		Mappings:   mapping,
		Provenance: provenance,
		BackMatter: buildMappingBackMatter(doc.Metadata.MappingReferences),
	}, nil
}

func resolveMappingReference(refs []gemara.MappingReference, refId string) (gemara.MappingReference, error) {
	for _, ref := range refs {
		if ref.Id == refId {
			return ref, nil
		}
	}
	return gemara.MappingReference{}, fmt.Errorf("mapping reference %q not found in metadata", refId)
}

func validateReverseDocument(forward, reverse gemara.MappingDocument) error {
	if forward.SourceReference.ReferenceId != reverse.TargetReference.ReferenceId {
		return fmt.Errorf("reverse target-reference %q does not match forward source-reference %q",
			reverse.TargetReference.ReferenceId, forward.SourceReference.ReferenceId)
	}
	if forward.TargetReference.ReferenceId != reverse.SourceReference.ReferenceId {
		return fmt.Errorf("reverse source-reference %q does not match forward target-reference %q",
			reverse.SourceReference.ReferenceId, forward.TargetReference.ReferenceId)
	}
	return nil
}

func buildMappingMetadata(doc gemara.MappingDocument) oscal12.Metadata {
	now := time.Now()
	version := doc.Metadata.Version
	if version == "" {
		version = oscalUtils.DefaultOSCALVersion
	}

	published := oscalUtils.GetTime(string(doc.Metadata.Date))

	metadata := oscal12.Metadata{
		Title:        doc.Title,
		OscalVersion: oscal12.Version,
		Version:      version,
		Published:    published,
		LastModified: now,
	}

	if doc.Metadata.Author.Name != "" {
		authorRole := oscal12.Role{
			ID:          "author",
			Description: "Author and owner of the document",
			Title:       "Author",
		}
		author := oscal12.Party{
			UUID: uuid.NewUUID(),
			Type: "person",
			Name: doc.Metadata.Author.Name,
		}
		responsibleParty := oscal12.ResponsibleParty{
			PartyUuids: []string{author.UUID},
			RoleId:     authorRole.ID,
		}
		metadata.Parties = &[]oscal12.Party{author}
		metadata.Roles = &[]oscal12.Role{authorRole}
		metadata.ResponsibleParties = &[]oscal12.ResponsibleParty{responsibleParty}
	}

	return metadata
}

func buildMappingMaps(doc gemara.MappingDocument) ([]oscal12.Map, error) {
	maps := make([]oscal12.Map, 0, len(doc.Mappings))
	for _, m := range doc.Mappings {
		if m.Relationship == gemara.RelNoMatch {
			continue
		}

		relationship, err := gemaraRelationshipToOSCAL(m.Relationship)
		if err != nil {
			return nil, fmt.Errorf("mapping %q: %w", m.Id, err)
		}

		targets := make([]oscal12.MappingItem, 0, len(m.Targets))
		var rationales []string
		for _, t := range m.Targets {
			targets = append(targets, oscal12.MappingItem{
				IDRef: t.EntryId,
				Type:  strings.ToLower(doc.TargetReference.EntryType.String()),
			})
			if t.Rationale != "" {
				rationales = append(rationales, t.Rationale)
			}
		}

		remarks := strings.Join(rationales, "\n")
		if m.Remarks != "" {
			if remarks != "" {
				remarks = remarks + "\n" + m.Remarks
			} else {
				remarks = m.Remarks
			}
		}

		oscalMap := oscal12.Map{
			UUID: uuid.NewUUID(),
			Sources: []oscal12.MappingItem{
				{
					IDRef: m.Source,
					Type:  strings.ToLower(doc.SourceReference.EntryType.String()),
				},
			},
			Targets:      targets,
			Relationship: relationship,
			Remarks:      remarks,
		}

		maps = append(maps, oscalMap)
	}
	return maps, nil
}

func buildSourceGapSummary(mappings []gemara.Mapping) *oscal12.GapSummary {
	var unmappedIds []string
	for _, m := range mappings {
		if m.Relationship == gemara.RelNoMatch {
			unmappedIds = append(unmappedIds, m.Source)
		}
	}
	if len(unmappedIds) == 0 {
		return nil
	}
	return &oscal12.GapSummary{
		UUID: uuid.NewUUID(),
		UnmappedControls: []oscal12.SelectControlById{
			{WithIds: &unmappedIds},
		},
	}
}

func buildTargetGapSummary(reverseDoc *gemara.MappingDocument) *oscal12.GapSummary {
	if reverseDoc == nil {
		return nil
	}
	return buildSourceGapSummary(reverseDoc.Mappings)
}

func buildMappingBackMatter(refs []gemara.MappingReference) *oscal12.BackMatter {
	if len(refs) == 0 {
		return nil
	}
	var resources []oscal12.Resource
	for _, ref := range refs {
		resource := oscal12.Resource{
			UUID:        uuid.NewUUID(),
			Title:       ref.Title,
			Description: ref.Description,
			Props: &[]oscal12.Property{
				{
					Name:  "id",
					Value: ref.Id,
					Ns:    oscalUtils.GemaraNamespace,
				},
			},
			Rlinks: &[]oscal12.ResourceLink{
				{Href: ref.Url},
			},
			Citation: &oscal12.Citation{
				Text: fmt.Sprintf("*%s*. %s", ref.Title, ref.Url),
			},
		}
		resources = append(resources, resource)
	}
	return &oscal12.BackMatter{Resources: &resources}
}
