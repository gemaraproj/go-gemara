// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"fmt"
	"strings"
	"time"

	"github.com/defenseunicorns/go-oscal/src/pkg/uuid"
	oscal "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/gemaraproj/go-gemara"
	oscalUtils "github.com/gemaraproj/go-gemara/internal/oscal"
	"github.com/opencontainers/go-digest"
)

// EvaluationLogToOSCALAssessmentResults converts a Gemara EvaluationLog into an
// OSCAL Assessment Results document.
//
// Use WithImportApHref to set the assessment plan reference (defaults to "#").
// Use WithCatalog to enrich findings with control titles and requirement text.
func EvaluationLogToOSCALAssessmentResults(log gemara.EvaluationLog, opts ...EvalOption) (oscal.AssessmentResults, error) {
	options := defaultEvalOpts()
	for _, opt := range opts {
		opt(&options)
	}

	authorPartyUUID := uuid.NewUUID()
	metadata := createAssessmentResultsMetadata(log, authorPartyUUID)

	evidenceHashes, err := collectEvidenceHashes(log)
	if err != nil {
		return oscal.AssessmentResults{}, fmt.Errorf("converting evaluation log %q: %w", log.Metadata.Id, err)
	}
	backMatter, mappingResourceUUIDs := mappingToBackMatterWithUUIDs(log.Metadata.MappingReferences, evidenceHashes)
	result, err := evaluationLogToResult(log, options.catalog, authorPartyUUID, mappingResourceUUIDs)
	if err != nil {
		return oscal.AssessmentResults{}, fmt.Errorf("converting evaluation log %q: %w", log.Metadata.Id, err)
	}

	return oscal.AssessmentResults{
		UUID:       uuid.NewUUID(),
		Metadata:   metadata,
		ImportAp:   oscal.ImportAp{Href: options.importApHref},
		Results:    []oscal.Result{result},
		BackMatter: backMatter,
	}, nil
}

func evaluationLogToResult(log gemara.EvaluationLog, catalog *gemara.ControlCatalog, authorPartyUUID string, mappingResourceUUIDs map[string]string) (oscal.Result, error) {
	now := time.Now()
	start := oscalUtils.GetTimeWithFallback(string(log.Metadata.Date), now)

	origin := buildOrigin(log.Metadata.Author, authorPartyUUID)

	controlIds := collectControlIds(log)
	reviewedControls := buildReviewedControls(controlIds)

	var observations []oscal.Observation
	var findings []oscal.Finding
	var logEntries []oscal.AssessmentLogEntry

	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}

		finding := buildFinding(eval, catalog, origin)
		var relatedObs []oscal.RelatedObservation

		for _, alog := range eval.AssessmentLogs {
			if alog == nil {
				continue
			}

			obs, err := buildObservation(alog, eval, origin, start, mappingResourceUUIDs)
			if err != nil {
				return oscal.Result{}, fmt.Errorf("converting assessment log for requirement %q: %w", alog.Requirement.EntryId, err)
			}
			observations = append(observations, obs)
			relatedObs = append(relatedObs, oscal.RelatedObservation{ObservationUuid: obs.UUID})

			entry := buildLogEntry(alog, eval, authorPartyUUID, start)
			logEntries = append(logEntries, entry)
		}

		if len(relatedObs) > 0 {
			finding.RelatedObservations = &relatedObs
		}
		findings = append(findings, finding)
	}

	title := fmt.Sprintf("Evaluation: %s", log.Metadata.Id)

	targetItem := buildTargetInventoryItem(log.Target)
	localDefs := oscal.LocalDefinitions{
		InventoryItems: &[]oscal.InventoryItem{targetItem},
	}

	result := oscal.Result{
		UUID:             uuid.NewUUID(),
		Title:            title,
		Description:      fmt.Sprintf("Results from Gemara EvaluationLog %s", log.Metadata.Id),
		Start:            start,
		ReviewedControls: reviewedControls,
		Observations:     oscalUtils.NilIfEmpty(observations),
		Findings:         oscalUtils.NilIfEmpty(findings),
		LocalDefinitions: &localDefs,
		Props: &[]oscal.Property{
			{
				Name:  "aggregate-result",
				Value: log.Result.String(),
				Ns:    oscalUtils.GemaraNamespace,
			},
		},
	}

	if len(logEntries) > 0 {
		result.AssessmentLog = &oscal.AssessmentLog{Entries: logEntries}
	}

	return result, nil
}

func createAssessmentResultsMetadata(log gemara.EvaluationLog, authorPartyUUID string) oscal.Metadata {
	version := log.Metadata.Version
	if version == "" {
		version = oscalUtils.DefaultOSCALVersion
	}

	now := time.Now()
	published := oscalUtils.GetTime(string(log.Metadata.Date))

	metadata := oscal.Metadata{
		Title:        fmt.Sprintf("Assessment Results: %s", log.Metadata.Id),
		OscalVersion: oscal.Version,
		Version:      version,
		Published:    published,
		LastModified: now,
	}

	if log.Metadata.Author.Name != "" {
		authorRole := oscal.Role{
			ID:    "assessor",
			Title: "Assessor",
		}
		party := oscal.Party{
			UUID: authorPartyUUID,
			Type: mapPartyType(log.Metadata.Author.Type),
			Name: log.Metadata.Author.Name,
		}
		metadata.Roles = &[]oscal.Role{authorRole}
		metadata.Parties = &[]oscal.Party{party}
		metadata.ResponsibleParties = &[]oscal.ResponsibleParty{
			{RoleId: authorRole.ID, PartyUuids: []string{party.UUID}},
		}
	}

	return metadata
}

func buildOrigin(author gemara.Actor, partyUUID string) oscal.Origin {
	return oscal.Origin{
		Actors: []oscal.OriginActor{
			{
				ActorUuid: partyUUID,
				Type:      mapActorType(author.Type),
				RoleId:    "assessor",
			},
		},
	}
}

func buildReviewedControls(controlIds []string) oscal.ReviewedControls {
	selectors := make([]oscal.AssessedControlsSelectControlById, 0, len(controlIds))
	for _, id := range controlIds {
		selectors = append(selectors, oscal.AssessedControlsSelectControlById{ControlId: id})
	}
	return oscal.ReviewedControls{
		ControlSelections: []oscal.AssessedControls{
			{IncludeControls: &selectors},
		},
	}
}

func buildFinding(eval *gemara.ControlEvaluation, catalog *gemara.ControlCatalog, origin oscal.Origin) oscal.Finding {
	controlId := eval.Control.EntryId
	status := mapResultToObjectiveStatus(eval.Result)

	description := eval.Message
	if description == "" {
		description = fmt.Sprintf("Evaluation of control %s", controlId)
	}

	title := eval.Name
	if catalog != nil {
		control, _ := findControlAndRequirement(catalog, controlId, "")
		if control != nil && control.Title != "" {
			title = control.Title
		}
	}
	if title == "" {
		title = controlId
	}
	title = sanitizeTitle(title)

	return oscal.Finding{
		UUID:        uuid.NewUUID(),
		Title:       title,
		Description: description,
		Origins:     &[]oscal.Origin{origin},
		Target: oscal.FindingTarget{
			Type:     "objective-id",
			TargetId: controlId,
			Status:   status,
		},
	}
}

func buildObservation(alog *gemara.AssessmentLog, eval *gemara.ControlEvaluation, origin oscal.Origin, fallback time.Time, mappingResourceUUIDs map[string]string) (oscal.Observation, error) {
	collected := oscalUtils.GetTimeWithFallback(string(alog.Start), fallback)

	description := alog.Description
	if description == "" {
		description = alog.Message
	}
	if description == "" {
		description = fmt.Sprintf("Assessment of requirement %s", alog.Requirement.EntryId)
	}

	methods := []string{"EXAMINE"}
	if alog.StepsExecuted > 0 {
		methods = []string{"TEST"}
	}

	obs := oscal.Observation{
		UUID:        uuid.NewUUID(),
		Description: description,
		Collected:   collected,
		Methods:     methods,
		Origins:     &[]oscal.Origin{origin},
		Props: &[]oscal.Property{
			{
				Name:  "result",
				Value: alog.Result.String(),
				Ns:    oscalUtils.GemaraNamespace,
			},
			{
				Name:  "requirement-id",
				Value: alog.Requirement.EntryId,
				Ns:    oscalUtils.GemaraNamespace,
			},
			{
				Name:  "control-id",
				Value: eval.Control.EntryId,
				Ns:    oscalUtils.GemaraNamespace,
			},
		},
	}

	if alog.Recommendation != "" {
		obs.Remarks = alog.Recommendation
	}

	if len(alog.Evidence) > 0 {
		evidence := buildRelevantEvidence(alog.Evidence, mappingResourceUUIDs)
		obs.RelevantEvidence = &evidence
	}

	return obs, nil
}

func buildRelevantEvidence(evidence []gemara.Evidence, mappingResourceUUIDs map[string]string) []oscal.RelevantEvidence {
	relevantEvidence := make([]oscal.RelevantEvidence, 0, len(evidence))
	for _, item := range evidence {
		props := []oscal.Property{
			{Name: "id", Value: item.Id, Ns: oscalUtils.GemaraNamespace},
			{Name: "type", Value: string(item.Type), Ns: oscalUtils.GemaraNamespace},
			{Name: "collected-at", Value: string(item.CollectedAt), Ns: oscalUtils.GemaraNamespace},
		}
		if item.Source.ReferenceId != "" {
			props = append(props, oscal.Property{Name: "source-reference-id", Value: item.Source.ReferenceId, Ns: oscalUtils.GemaraNamespace})
		}
		if item.Source.Coordinate != "" {
			props = append(props, oscal.Property{Name: "source-coordinate", Value: item.Source.Coordinate, Ns: oscalUtils.GemaraNamespace})
		}
		if item.Source.EntryId != "" {
			props = append(props, oscal.Property{Name: "source-entry-id", Value: item.Source.EntryId, Ns: oscalUtils.GemaraNamespace})
		}
		itemEvidence := oscal.RelevantEvidence{
			Description: item.Description,
			Props:       &props,
			Remarks:     item.Source.Remarks,
		}
		if resourceUUID, found := mappingResourceUUIDs[item.Source.ReferenceId]; found {
			itemEvidence.Links = &[]oscal.Link{{Href: "#" + resourceUUID, Rel: "reference"}}
		}
		relevantEvidence = append(relevantEvidence, itemEvidence)
	}
	return relevantEvidence
}

func collectEvidenceHashes(log gemara.EvaluationLog) (map[string][]oscal.Hash, error) {
	hashesByReference := make(map[string][]oscal.Hash)
	seen := make(map[string]map[string]struct{})
	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}
		for _, assessmentLog := range eval.AssessmentLogs {
			if assessmentLog == nil {
				continue
			}
			for _, evidence := range assessmentLog.Evidence {
				rawDigest := evidence.Source.Digest
				if rawDigest == "" {
					continue
				}
				parsedDigest, err := digest.Parse(rawDigest)
				if err != nil {
					return nil, fmt.Errorf("parsing evidence digest %q: %w", rawDigest, err)
				}
				referenceID := evidence.Source.ReferenceId
				if _, exists := seen[referenceID]; !exists {
					seen[referenceID] = make(map[string]struct{})
				}
				if _, exists := seen[referenceID][rawDigest]; exists {
					continue
				}
				seen[referenceID][rawDigest] = struct{}{}
				hashesByReference[referenceID] = append(hashesByReference[referenceID], oscal.Hash{Algorithm: parsedDigest.Algorithm().String(), Value: parsedDigest.Encoded()})
			}
		}
	}
	return hashesByReference, nil
}

func buildLogEntry(alog *gemara.AssessmentLog, eval *gemara.ControlEvaluation, partyUUID string, fallback time.Time) oscal.AssessmentLogEntry {
	start := oscalUtils.GetTimeWithFallback(string(alog.Start), fallback)

	title := fmt.Sprintf("%s / %s", eval.Control.EntryId, alog.Requirement.EntryId)

	entry := oscal.AssessmentLogEntry{
		UUID:        uuid.NewUUID(),
		Title:       title,
		Start:       start,
		Description: alog.Description,
		LoggedBy: &[]oscal.LoggedBy{
			{PartyUuid: partyUUID, RoleId: "assessor"},
		},
	}

	if end := oscalUtils.GetTime(string(alog.End)); end != nil {
		entry.End = end
	}

	return entry
}

func buildTargetInventoryItem(target gemara.Resource) oscal.InventoryItem {
	description := target.Description
	if description == "" {
		description = target.Name
	}

	return oscal.InventoryItem{
		UUID:        uuid.NewUUID(),
		Description: description,
		Props: &[]oscal.Property{
			{
				Name:  "gemara-resource-id",
				Value: target.Id,
				Ns:    oscalUtils.GemaraNamespace,
			},
			{
				Name:  "name",
				Value: target.Name,
				Ns:    oscalUtils.GemaraNamespace,
			},
		},
	}
}

func collectControlIds(log gemara.EvaluationLog) []string {
	seen := make(map[string]struct{})
	var ids []string
	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}
		id := eval.Control.EntryId
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids
}

func mapResultToObjectiveStatus(r gemara.Result) oscal.ObjectiveStatus {
	switch r {
	case gemara.Passed:
		return oscal.ObjectiveStatus{State: "satisfied"}
	case gemara.Failed:
		return oscal.ObjectiveStatus{State: "not-satisfied"}
	default:
		return oscal.ObjectiveStatus{
			State:  "not-satisfied",
			Reason: r.String(),
		}
	}
}

// sanitizeTitle collapses newlines and surrounding whitespace into a single
// space so the value satisfies the OSCAL StringDatatype pattern (^[^\n]+$).
func sanitizeTitle(s string) string {
	parts := strings.Fields(s)
	return strings.Join(parts, " ")
}

func mapActorType(t gemara.EntityType) string {
	switch t {
	case gemara.Human:
		return "person"
	default:
		return "tool"
	}
}

func mapPartyType(t gemara.EntityType) string {
	switch t {
	case gemara.Human:
		return "person"
	default:
		return "organization"
	}
}
