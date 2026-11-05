# Deterministic UUIDs in OSCAL Conversions

This document explains how `gemaraconv` generates UUIDs for OSCAL assemblies
and the conventions that govern them.

## Background

OSCAL documents contain UUIDs on every assembly (catalog, finding, observation,
party, resource, etc.). Before this change, `gemaraconv` generated random UUIDv4
values on every conversion. The same Gemara input produced byte-different OSCAL
output on each run, breaking reproducibility, diffing, and downstream
correlation.

The fix switches from `uuid.NewUUID()` (random v4) to `uuid.NewUUIDWithSource()`
(deterministic v5, SHA-1, URL namespace) from the `go-oscal` library. Each call
site passes a source string derived from the assembly's identity fields in the
Gemara input.

## UUID Persistence

The OSCAL 1.1.3 schemas bundled with go-oscal describe most UUIDs as
per-subject: "This UUID should be assigned per-subject, which means it should
be consistently used to identify the same subject across revisions of the
document." That sentence appears on the AssessmentResults root, the Result,
and the finding, observation, log-entry, and inventory-item assemblies. Party
and back-matter resource UUIDs are described only as plain unique identifiers
("A unique identifier for the party." / "A unique identifier for a
resource."); treating them as per-subject is this project's convention, not a
schema mandate. The catalog and profile root UUIDs carry no persistence
wording at all; their schema description is only a "globally unique means to
identify a given catalog instance."

The convention below has three tiers.

### Document roots: stable per identity

Catalog and Profile roots derive from `metadata.Id` and `version`:
`gemara:{metadata.Id}/{version}/{document-type}`. The conversion is a pure
function, so the same document at the same version always produces the same
root, and a version bump mints a new one.

### Assessment Results roots and Results: per run

The AssessmentResults root and its Result additionally include
`metadata.Date`:
`gemara:{metadata.Id}/{version}/{metadata.Date}/assessment-results`. Each
conversion of an evaluation log is a new assessment run, even when the log id
and version are unchanged (Privateer, for example, leaves the version empty),
so the root and the Result change with the date.

This is a deliberate departure from the per-subject wording above: a re-run
on a new date is a new write of the results document, not a revision of the
same one. Consumers that need a stable handle on the assessment itself should
key on the log's `metadata.Id`, not on the root UUID.

### Assemblies: per-subject

Every inner assembly UUID (finding, observation, party, resource, log entry,
inventory item) uses identity fields only (IDs, structural keys), not content
fields. Changing a control's description or an assessment's result does not
change the UUID, because the subject (the control, the requirement) has not
changed. This is the per-subject rule as the schema states it.

The Result is the exception: a Result is one run of the log, so its source
includes `metadata.Date` alongside the log ID.

## Source String Convention

All source strings follow the format:

```
gemara:<identity-fields>/<entity-type>
```

The `gemara:` prefix prevents collision with other UUIDv5 URL-namespace users.
The entity-type suffix prevents collision between assemblies that share the same
identity fields (a finding and an observation for the same control+requirement).

### Field Escaping

Identity fields are URL-encoded with `url.PathEscape()` before concatenation.
This prevents ambiguity when a field contains `/`, which is also the separator
between fields. For example, a catalog with `metadata.Id = "A/B"` and
`version = "C"` produces `gemara:A%2FB/C/catalog`, distinct from `metadata.Id = "A"`
with `version = "B/C"` which produces `gemara:A/B%2FC/catalog`.

### Complete Mapping

| OSCAL Assembly | Persistence | Source String | Identity Fields |
|---|---|---|---|
| Catalog (control) | per (id, version) | `gemara:{id}/{version}/catalog` | Document ID + version |
| Catalog (guidance) | per (id, version) | `gemara:{id}/{version}/guidance-catalog` | Document ID + version |
| Profile | per (id, version) | `gemara:{id}/{version}/guidance-profile` | Document ID + version |
| AssessmentResults | per run | `gemara:{id}/{version}/{date}/assessment-results` | Document ID + version + date |
| Party | `per-subject` | `gemara:{actor-id}/party` (name + type when the id is empty) | Actor ID, or author name + type when absent |
| Resource | `per-subject` | `gemara:{ref.Id}/resource` | Mapping reference ID |
| Result | per-run | `gemara:{log.Metadata.Id}/{date}/result` | Log ID + date |
| Finding | `per-subject` | `gemara:{logId}/{controlId}/finding` | Log ID + control entry ID |
| Observation | `per-subject` | `gemara:{logId}/{controlId}/{requirementId}/{occurrence}/observation` | Log ID + control + requirement + occurrence |
| LogEntry | `per-subject` | `gemara:{logId}/{controlId}/{requirementId}/{occurrence}/log-entry` | Log ID + control + requirement + occurrence |
| InventoryItem | `per-subject` | `gemara:{target.Id}/inventory-item` | Target resource ID |

All `{field}` placeholders denote `url.PathEscape(field)` values. Catalog and
Profile roots use `options.version` (`WithVersion`, else `metadata.Version`,
else `"0.0.0"`). The AssessmentResults root has no version option; it uses
`log.Metadata.Version` with the same `"0.0.0"` fallback.

The `{occurrence}` field is a 1-based counter over assessment logs that share
the same (control, requirement) key, counted in input order. A requirement can
be evaluated in more than one way, and OSCAL requires UUIDs to be unique within
a document, so the counter keeps the observation and log-entry UUIDs distinct
while staying reproducible. It can be swapped for plan and method ids if
gemaraproj/gemara#506 lands.

Two sources deliberately omit the document id: Resource keys on the mapping
reference ID and InventoryItem on the target resource ID. Both identify the
same subject regardless of which document mentions it, so their UUIDs match
across documents that reference them. The document id is part of the Finding,
Observation, and LogEntry sources because those assemblies are scoped to a
single evaluation log.

### Fields Excluded from Source Strings

Content and descriptive fields are excluded because they are properties of the
subject, not its identity:

- `eval.Result`, `eval.Message` (finding verdict and description)
- `alog.Result`, `alog.Description`, `alog.Recommendation`, `alog.Evidence`
- `ref.Title`, `ref.Url`, `ref.Version`, `ref.Description`
- `target.Name`, `target.Description`
- `confidence-level` (in the Breaking Changes milestone, gemara#497)
- `executor` (attribution, not identity, per gemara PR #506)

## Stability Across Runs

Which UUIDs survive a re-run of the same log and which change:

| UUID | Same input, new run | Changes with |
|---|---|---|
| AssessmentResults root | changes when the date changes | `metadata.Date`, version |
| Result | changes when the date changes | `metadata.Date` |
| Finding, Observation, LogEntry | stable | identity fields only |
| Party, Resource, InventoryItem | stable | identity fields only |
| Catalog / Profile root | stable | `metadata.Id`, version |

Per-subject assemblies staying stable across runs is what makes run-to-run
diffs readable: only the root, the Result, and any changed content move.

## Behavior

- **Same input, same output.** Converting the same Gemara document twice produces
  identical OSCAL UUIDs.
- **Version bump changes document UUID.** Bumping `metadata.version` changes the
  root UUID (Catalog, Profile, AssessmentResults) but assembly UUIDs stay the
  same. For Assessment Results, a new `metadata.Date` also changes the root and
  Result UUIDs.
- **Content changes do not change assembly UUIDs.** Editing a finding's message,
  an observation's description, or a resource's title does not change its UUID.
- **Different subjects get different UUIDs.** Each assembly type derives its UUID
  from its own identity fields (see the mapping table above). Different values in
  those fields produce different UUIDs.
- **No collision across assembly types.** A Finding and an Observation for the
  same control+requirement get different UUIDs because of the entity-type suffix.
- **No collision within a document.** Assessment logs that evaluate the same
  requirement under the same control get distinct observation and log-entry
  UUIDs via the occurrence counter.
- **Duplicate control IDs are rejected.** `EvaluationLogToOSCALAssessmentResults`
  returns an error when two control evaluations share a control ID. The Privateer
  SDK emits one evaluation per catalog control, so a duplicate is a catalog
  authoring error, and duplicate findings would corrupt result rollups.
- **Conflicting mapping references are rejected.** A repeated mapping reference ID
  with the same title and URL is deduplicated. A repeated ID with a different
  title or URL returns an error, since one back-matter resource cannot represent
  two distinct sources.
- **Authors without a name carry no party.** The metadata party, the origin
  actors on findings and observations, and the logged-by entry on assessment
  log entries all gate on a non-empty author name, on every conversion path.
  An author with neither `id` nor `name` has no identity to key a party UUID
  on; deriving one would collapse every anonymous actor of the same entity
  type onto a single shared party across all documents. An author with an
  `id` but no name likewise carries no party and no origin actors, so no
  output ever references a party the document does not define.
- **Party.Type follows the author's entity type.** Author parties are typed via
  `mapPartyType` (person for human actors, organization otherwise) on every
  conversion path. Catalog and guidance author parties were previously
  hardcoded to "person" regardless of the author's entity type.
