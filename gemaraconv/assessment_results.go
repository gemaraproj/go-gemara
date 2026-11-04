// SPDX-License-Identifier: Apache-2.0

package gemaraconv

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/defenseunicorns/go-oscal/src/pkg/uuid"
	oscal "github.com/defenseunicorns/go-oscal/src/types/oscal-1-1-3"
	"github.com/gemaraproj/go-gemara"
	oscalUtils "github.com/gemaraproj/go-gemara/internal/oscal"
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

	seenControls := make(map[string]struct{}, len(log.Evaluations))
	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}
		controlId := eval.Control.EntryId
		if _, exists := seenControls[controlId]; exists {
			return oscal.AssessmentResults{}, fmt.Errorf("duplicate control evaluation for control %s", controlId)
		}
		seenControls[controlId] = struct{}{}
	}

	authorPartyUUID := uuid.NewUUIDWithSource(partyUUIDSource(log.Metadata.Author))
	metadata := createAssessmentResultsMetadata(log, authorPartyUUID)

	backMatter, mappingResourceUUIDs, err := mappingToBackMatterWithUUIDs(log.Metadata.MappingReferences)
	if err != nil {
		return oscal.AssessmentResults{}, err
	}
	result := evaluationLogToResult(log, options.catalog, authorPartyUUID, mappingResourceUUIDs, log.Metadata.Id)

	version := log.Metadata.Version
	if version == "" {
		version = oscalUtils.DefaultOSCALVersion
	}

	return oscal.AssessmentResults{
		UUID:       uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/%s/assessment-results", url.PathEscape(log.Metadata.Id), url.PathEscape(version), url.PathEscape(string(log.Metadata.Date)))),
		Metadata:   metadata,
		ImportAp:   oscal.ImportAp{Href: options.importApHref},
		Results:    []oscal.Result{result},
		BackMatter: backMatter,
	}, nil
}

// assessmentKey identifies the (control, requirement) pair an assessment log
// evaluates. Assessment logs that share a key are distinguished by an
// occurrence counter rather than identity fields alone.
type assessmentKey struct {
	controlId     string
	requirementId string
}

func evaluationLogToResult(log gemara.EvaluationLog, catalog *gemara.ControlCatalog, authorPartyUUID string, mappingResourceUUIDs map[string]string, logId string) oscal.Result {
	now := time.Now()
	start := oscalUtils.GetTimeWithFallback(string(log.Metadata.Date), now)

	origin := buildOrigin(log.Metadata.Author, authorPartyUUID)

	controlIds := collectControlIds(log)
	reviewedControls := buildReviewedControls(controlIds)

	var observations []oscal.Observation
	var findings []oscal.Finding
	var logEntries []oscal.AssessmentLogEntry

	// Occurrence counter for assessment logs sharing the same
	// (control, requirement) key. OSCAL requires per-document UUID
	// uniqueness, and a requirement can be evaluated in more than one way.
	occurrences := make(map[assessmentKey]int)

	for _, eval := range log.Evaluations {
		if eval == nil {
			continue
		}

		finding := buildFinding(eval, catalog, origin, logId)
		var relatedObs []oscal.RelatedObservation

		for _, alog := range eval.AssessmentLogs {
			if alog == nil {
				continue
			}

			key := assessmentKey{controlId: eval.Control.EntryId, requirementId: alog.Requirement.EntryId}
			occurrences[key]++

			obs := buildObservation(alog, eval, origin, start, mappingResourceUUIDs, logId, occurrences[key])
			observations = append(observations, obs)
			relatedObs = append(relatedObs, oscal.RelatedObservation{ObservationUuid: obs.UUID})

			entry := buildLogEntry(alog, eval, authorPartyUUID, start, logId, occurrences[key])
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
		UUID:             uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/result", url.PathEscape(log.Metadata.Id), url.PathEscape(string(log.Metadata.Date)))),
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

	return result
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
			{IncludeControls: oscalUtils.NilIfEmpty(selectors)},
		},
	}
}

func buildFinding(eval *gemara.ControlEvaluation, catalog *gemara.ControlCatalog, origin oscal.Origin, logId string) oscal.Finding {
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
		UUID:        uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/finding", url.PathEscape(logId), url.PathEscape(controlId))),
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

func buildObservation(alog *gemara.AssessmentLog, eval *gemara.ControlEvaluation, origin oscal.Origin, fallback time.Time, mappingResourceUUIDs map[string]string, logId string, occurrence int) oscal.Observation {
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
		UUID:        uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/%s/%d/observation", url.PathEscape(logId), url.PathEscape(eval.Control.EntryId), url.PathEscape(alog.Requirement.EntryId), occurrence)),
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

	return obs
}

func buildRelevantEvidence(evidence []gemara.Evidence, mappingResourceUUIDs map[string]string) []oscal.RelevantEvidence {
	relevantEvidence := make([]oscal.RelevantEvidence, 0, len(evidence))
	for _, item := range evidence {
		var props []oscal.Property
		if item.Id != "" {
			props = append(props, oscal.Property{Name: "id", Value: item.Id, Ns: oscalUtils.GemaraNamespace})
		}
		if item.Type != "" {
			props = append(props, oscal.Property{Name: "type", Value: string(item.Type), Ns: oscalUtils.GemaraNamespace})
		}
		if item.CollectedAt != "" {
			props = append(props, oscal.Property{Name: "collected-at", Value: string(item.CollectedAt), Ns: oscalUtils.GemaraNamespace})
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
		if item.Source.Digest != "" {
			props = append(props, oscal.Property{Name: "source-digest", Value: item.Source.Digest, Ns: oscalUtils.GemaraNamespace})
		}
		itemEvidence := oscal.RelevantEvidence{
			Description: item.Description,
			Props:       oscalUtils.NilIfEmpty(props),
			Remarks:     item.Source.Remarks,
		}
		if item.Source.ReferenceId != "" {
			if resourceUUID, found := mappingResourceUUIDs[item.Source.ReferenceId]; found {
				itemEvidence.Links = &[]oscal.Link{{Href: "#" + resourceUUID, Rel: "resource"}}
			}
		}
		relevantEvidence = append(relevantEvidence, itemEvidence)
	}
	return relevantEvidence
}

func buildLogEntry(alog *gemara.AssessmentLog, eval *gemara.ControlEvaluation, partyUUID string, fallback time.Time, logId string, occurrence int) oscal.AssessmentLogEntry {
	start := oscalUtils.GetTimeWithFallback(string(alog.Start), fallback)

	title := fmt.Sprintf("%s / %s", eval.Control.EntryId, alog.Requirement.EntryId)

	entry := oscal.AssessmentLogEntry{
		UUID:        uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/%s/%s/%d/log-entry", url.PathEscape(logId), url.PathEscape(eval.Control.EntryId), url.PathEscape(alog.Requirement.EntryId), occurrence)),
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
		UUID:        uuid.NewUUIDWithSource(fmt.Sprintf("gemara:%s/inventory-item", url.PathEscape(target.Id))),
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

// mapResultToObjectiveStatus maps a Gemara result to an OSCAL objective status.
// OSCAL types the reason as a token, so Result.String() ("Not Run") cannot be
// used. OSCAL suggests pass, fail and other but accepts any token, so results
// other than Passed and Failed get a descriptive reason. A value outside the
// Result enum cannot come from a valid EvaluationLog, so it is "invalid".
func mapResultToObjectiveStatus(r gemara.Result) oscal.ObjectiveStatus {
	switch r {
	case gemara.Passed:
		return oscal.ObjectiveStatus{State: "satisfied"}
	case gemara.Failed:
		return oscal.ObjectiveStatus{State: "not-satisfied"}
	case gemara.NotRun:
		return oscal.ObjectiveStatus{State: "not-satisfied", Reason: "not-run"}
	case gemara.NeedsReview:
		return oscal.ObjectiveStatus{State: "not-satisfied", Reason: "needs-review"}
	case gemara.NotApplicable:
		return oscal.ObjectiveStatus{State: "not-satisfied", Reason: "not-applicable"}
	case gemara.Unknown:
		return oscal.ObjectiveStatus{State: "not-satisfied", Reason: "unknown"}
	default:
		return oscal.ObjectiveStatus{State: "not-satisfied", Reason: "invalid"}
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
