package gemaraconv

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/gemaraproj/go-gemara"
)

var emptyArtifactURIMessage = "no file associated with this alert"

// ToSARIF converts a Gemara EvaluationLog to a SARIF document (v2.1.0).
// Each AssessmentLog is emitted as a SARIF result. The rule id is derived from
// the control id and requirement id.
//
// When an AssessmentLog carries Evidence with a source, real artifact URIs are
// resolved from the evidence source and metadata mapping references, replacing
// synthetic artifact URIs. The evidence digest is recorded on the artifactLocation
// properties and mapped into run.artifacts hashes. If evidence or its source is
// absent, PhysicalLocation falls back to WithArtifactURI (or a placeholder).
//
// Use WithArtifactURI to set the fallback PhysicalLocation (defaults to a placeholder).
// Use WithCatalog to enrich rules with requirement text and recommendations.
// Use WithExcludedStatuses to omit results by Gemara status.
//
// Each result carries a kind reflecting its Gemara status (fail, review, open,
// or pass); per SARIF v2.1.0 the level is only set (to "error") when the kind
// is "fail". The originating status string is recorded in the result property
// bag under "gemara/result", and a partial fingerprint keyed on the requirement
// id keeps alert identity stable across runs for viewers that track results.
//
// PhysicalLocation identifies the artifact (file/repository) where the result was found.
// LogicalLocation identifies the logical component (assessment step) that produced the result.
// Region is left nil as we don't have file-specific line/column data.
func ToSARIF(evaluationLog gemara.EvaluationLog, opts ...EvalOption) ([]byte, error) {
	options := defaultEvalOpts()
	for _, opt := range opts {
		opt(&options)
	}
	defaultArtifactURI := options.artifactURI
	if defaultArtifactURI == "" {
		defaultArtifactURI = emptyArtifactURIMessage
	}
	catalog := options.catalog
	report := &SarifReport{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/123e95847b13fbdd4cbe2120fa5e33355d4a042b/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
	}
	driver := ToolComponent{
		Name:           evaluationLog.Metadata.Author.Name,
		InformationURI: evaluationLog.Metadata.Author.Uri,
		Version:        evaluationLog.Metadata.Author.Version,
	}
	run := Run{Tool: Tool{Driver: driver}}

	refMap := make(map[string]gemara.MappingReference, len(evaluationLog.Metadata.MappingReferences))
	for _, ref := range evaluationLog.Metadata.MappingReferences {
		if ref.Id != "" {
			refMap[ref.Id] = ref
		}
	}

	var artifacts []Artifact
	artifactIndexMap := make(map[string]int)

	// Build a simple in-memory set of rules to avoid duplicates
	ruleIdSeen := map[string]bool{}
	rules := []ReportingDescriptor{}

	for _, evaluation := range evaluationLog.Evaluations {
		for _, log := range evaluation.AssessmentLogs {
			if log == nil {
				continue
			}

			// Skip NotRun and NotApplicable results - only include Passed, Failed, NeedsReview, Unknown
			if log.Result == gemara.NotRun || log.Result == gemara.NotApplicable {
				continue
			}

			if slices.Contains(options.excludedStatuses, log.Result) {
				continue
			}

			ruleID := log.Requirement.EntryId
			if !ruleIdSeen[ruleID] {
				rule := ReportingDescriptor{ID: ruleID}
				if log.Description != "" {
					rule.Name = log.Description
				} else if evaluation.Name != "" {
					rule.Name = evaluation.Name
				}

				// Enrich rule with catalog data if available
				if catalog != nil {
					control, requirement := findControlAndRequirement(catalog, evaluation.Control.EntryId, log.Requirement.EntryId)
					if control != nil {
						// Use control title if name is still empty
						if rule.Name == "" {
							rule.Name = control.Title
						}

						// Add requirement text as short description
						if requirement != nil && requirement.Text != "" {
							rule.ShortDescription = &Message{Text: requirement.Text}
						}

						// Add full description: control objective + requirement text
						if control.Objective != "" {
							fullDesc := control.Objective
							if requirement != nil && requirement.Text != "" {
								fullDesc = fmt.Sprintf("%s\n\nRequirement: %s", control.Objective, requirement.Text)
							}
							rule.FullDescription = &Message{Text: fullDesc}
						}

						// Add recommendation as help text
						if log.Recommendation != "" {
							rule.Help = &Message{Text: log.Recommendation}
						} else if requirement != nil && requirement.Recommendation != "" {
							rule.Help = &Message{Text: requirement.Recommendation}
						}

						// HelpUri is left empty - catalog-specific URI generation should be handled by the caller
					}
				}

				rules = append(rules, rule)
				ruleIdSeen[ruleID] = true
			}

			kind, level := mapResultToKindLevel(log.Result)

			// Message: prefer specific message, fallback to description
			msg := log.Message
			if msg == "" {
				msg = log.Description
			}

			// Use the last AssessmentStep for LogicalLocation (the location is for the entire evaluation)
			logicalLocationName := ruleID
			if len(log.Steps) > 0 {
				lastStep := log.Steps[len(log.Steps)-1]
				if lastStep != nil {
					logicalLocationName = lastStep.String()
				}
			}

			var locations []Location
			if len(log.Evidence) > 0 {
				locations = buildLocationsFromEvidence(log.Evidence, refMap, logicalLocationName, &artifacts, artifactIndexMap, defaultArtifactURI)
			}
			if len(locations) == 0 {
				locations = []Location{
					{
						PhysicalLocation: &PhysicalLocation{
							ArtifactLocation: ArtifactLocation{
								URI: defaultArtifactURI,
							},
						},
						LogicalLocations: []LogicalLocation{
							{FullyQualifiedName: logicalLocationName},
						},
					},
				}
			}

			result := ResultEntry{
				RuleID:    ruleID,
				Kind:      kind,
				Level:     level,
				Message:   Message{Text: msg},
				Locations: locations,
				PartialFingerprints: map[string]string{
					"gemara/requirementId/v1": ruleID,
				},
				Properties: map[string]any{
					"gemara/result": log.Result.String(),
				},
			}
			run.Results = append(run.Results, result)
		}
	}

	// attach rules if any
	if len(rules) > 0 {
		run.Tool.Driver.Rules = rules
	}

	// attach artifacts if any
	if len(artifacts) > 0 {
		run.Artifacts = artifacts
	}

	report.Runs = append(report.Runs, run)
	return json.Marshal(report)
}

// mapResultToKindLevel maps a Gemara result to a SARIF result kind and level.
// Per SARIF v2.1.0 (§3.27.10), level applies only when kind is "fail"; for any
// other kind the level is omitted (equivalent to "none").
func mapResultToKindLevel(r gemara.Result) (kind, level string) {
	switch r {
	case gemara.Failed:
		return "fail", "error"
	case gemara.NeedsReview:
		return "review", ""
	case gemara.Unknown:
		return "open", ""
	case gemara.Passed, gemara.NotApplicable, gemara.NotRun:
		fallthrough
	default:
		return "pass", ""
	}
}

// Minimal SARIF v2.1.0 model we need for export without external deps
type SarifReport struct {
	Schema  string `json:"$schema"`
	Version string `json:"version"`
	Runs    []Run  `json:"runs"`
}

type Run struct {
	Tool      Tool          `json:"tool"`
	Artifacts []Artifact    `json:"artifacts,omitempty"`
	Results   []ResultEntry `json:"results,omitempty"`
}

type Tool struct {
	Driver ToolComponent `json:"driver"`
}

type ToolComponent struct {
	Name                  string                `json:"name"`
	InformationURI        string                `json:"informationUri,omitempty"`
	Version               string                `json:"version,omitempty"`
	SemanticVersion       string                `json:"semanticVersion,omitempty"`
	DottedQuadFileVersion string                `json:"dottedQuadFileVersion,omitempty"`
	Rules                 []ReportingDescriptor `json:"rules,omitempty"`
}

type Artifact struct {
	Description *Message          `json:"description,omitempty"`
	Location    *ArtifactLocation `json:"location,omitempty"`
	Hashes      map[string]string `json:"hashes,omitempty"`
}

type ReportingDescriptor struct {
	ID               string   `json:"id"`
	Name             string   `json:"name,omitempty"`
	ShortDescription *Message `json:"shortDescription,omitempty"`
	FullDescription  *Message `json:"fullDescription,omitempty"`
	Help             *Message `json:"help,omitempty"`
	HelpUri          string   `json:"helpUri,omitempty"`
}

type ResultEntry struct {
	RuleID              string            `json:"ruleId"`
	Kind                string            `json:"kind,omitempty"`
	Level               string            `json:"level,omitempty"`
	Message             Message           `json:"message"`
	Locations           []Location        `json:"locations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints,omitempty"`
	Properties          map[string]any    `json:"properties,omitempty"`
}

type Message struct {
	Text string `json:"text"`
}

type Location struct {
	PhysicalLocation *PhysicalLocation `json:"physicalLocation,omitempty"`
	LogicalLocations []LogicalLocation `json:"logicalLocations,omitempty"`
}

type PhysicalLocation struct {
	ArtifactLocation ArtifactLocation `json:"artifactLocation"`
	Region           *Region          `json:"region,omitempty"`
}

type ArtifactLocation struct {
	URI         string         `json:"uri"`
	URIBaseID   string         `json:"uriBaseId,omitempty"`
	Index       *int           `json:"index,omitempty"`
	Description *Message       `json:"description,omitempty"`
	Properties  map[string]any `json:"properties,omitempty"`
}

type Region struct {
	StartLine   int      `json:"startLine,omitempty"`
	StartColumn int      `json:"startColumn,omitempty"`
	EndLine     int      `json:"endLine,omitempty"`
	EndColumn   int      `json:"endColumn,omitempty"`
	Snippet     *Snippet `json:"snippet,omitempty"`
}

type Snippet struct {
	Text string `json:"text"`
}

type LogicalLocation struct {
	FullyQualifiedName string `json:"fullyQualifiedName,omitempty"`
}

// buildLocationsFromEvidence constructs SARIF Location objects from Gemara Evidence items.
func buildLocationsFromEvidence(
	evidence []gemara.Evidence,
	refMap map[string]gemara.MappingReference,
	stepLocationName string,
	artifacts *[]Artifact,
	artifactIndexMap map[string]int,
	defaultArtifactURI string,
) []Location {
	var locations []Location
	seenEvidenceIDs := make(map[string]bool)
	seenAnon := make(map[string]bool)

	for _, item := range evidence {
		if isEmptySource(item.Source) {
			continue
		}

		if item.Id != "" {
			if seenEvidenceIDs[item.Id] {
				continue
			}
			seenEvidenceIDs[item.Id] = true
		} else {
			anonKey := fmt.Sprintf("%s|%s|%s|%s|%s",
				item.Source.ReferenceId,
				item.Source.Coordinate,
				item.Source.EntryId,
				item.Source.Digest,
				item.Description,
			)
			if seenAnon[anonKey] {
				continue
			}
			seenAnon[anonKey] = true
		}

		uri := resolveArtifactURI(refMap, item.Source)
		locURI := uri
		if locURI == "" {
			locURI = defaultArtifactURI
		}

		artLoc := ArtifactLocation{
			URI: locURI,
		}

		desc := item.Description
		if desc == "" {
			desc = item.Source.Remarks
		}
		if desc != "" {
			artLoc.Description = &Message{Text: desc}
		}

		var props map[string]any
		if item.Source.Digest != "" {
			if props == nil {
				props = make(map[string]any)
			}
			props["gemara/digest"] = item.Source.Digest
			props["gemara/source-digest"] = item.Source.Digest
		}
		if item.Source.ReferenceId != "" {
			if props == nil {
				props = make(map[string]any)
			}
			props["gemara/source-reference-id"] = item.Source.ReferenceId
		}
		if item.Source.Coordinate != "" {
			if props == nil {
				props = make(map[string]any)
			}
			props["gemara/source-coordinate"] = item.Source.Coordinate
		}
		if item.Source.EntryId != "" {
			if props == nil {
				props = make(map[string]any)
			}
			props["gemara/source-entry-id"] = item.Source.EntryId
		}
		artLoc.Properties = props

		if uri != "" {
			artifactKey := uri + "@" + item.Source.Digest
			idx, found := artifactIndexMap[artifactKey]
			if !found {
				var hashes map[string]string
				if item.Source.Digest != "" {
					algo, hash, ok := strings.Cut(item.Source.Digest, ":")
					if ok && algo != "" && hash != "" {
						algo = normalizeHashAlgorithm(algo)
						hashes = map[string]string{algo: hash}
					}
				}
				art := Artifact{
					Location: &ArtifactLocation{URI: uri},
					Hashes:   hashes,
				}
				if desc != "" {
					art.Description = &Message{Text: desc}
				}
				idx = len(*artifacts)
				*artifacts = append(*artifacts, art)
				artifactIndexMap[artifactKey] = idx
			}
			idxCopy := idx
			artLoc.Index = &idxCopy
		}

		locations = append(locations, Location{
			PhysicalLocation: &PhysicalLocation{
				ArtifactLocation: artLoc,
			},
			LogicalLocations: []LogicalLocation{
				{FullyQualifiedName: stepLocationName},
			},
		})
	}

	return locations
}

// resolveArtifactURI resolves an artifact URI from an EvidenceMapping and the document's MappingReferences.
// Returns the reference URL if found and non-empty. Coordinate is kept in properties.
func resolveArtifactURI(refMap map[string]gemara.MappingReference, source gemara.EvidenceMapping) string {
	if source.ReferenceId != "" {
		if ref, found := refMap[source.ReferenceId]; found {
			return ref.Url
		}
	}
	return ""
}

func isEmptySource(s gemara.EvidenceMapping) bool {
	return s.ReferenceId == "" && s.Coordinate == "" && s.EntryId == "" && s.Digest == "" && s.Remarks == ""
}

func normalizeHashAlgorithm(algo string) string {
	switch strings.ToLower(algo) {
	case "sha256":
		return "sha-256"
	case "sha512":
		return "sha-512"
	case "sha384":
		return "sha-384"
	case "sha1":
		return "sha-1"
	default:
		return strings.ToLower(algo)
	}
}

// findControlAndRequirement searches the catalog for a control and requirement by their IDs.
// Returns the control and requirement if found, nil otherwise.
func findControlAndRequirement(catalog *gemara.ControlCatalog, controlID, requirementID string) (*gemara.Control, *gemara.AssessmentRequirement) {
	if catalog == nil {
		return nil, nil
	}

	for i := range catalog.Controls {
		control := &catalog.Controls[i]
		if control.Id == controlID {
			// Found the control, now find the requirement
			for j := range control.AssessmentRequirements {
				requirement := &control.AssessmentRequirements[j]
				if requirement.Id == requirementID {
					return control, requirement
				}
			}
			// Control found but requirement not found
			return control, nil
		}
	}

	return nil, nil
}
