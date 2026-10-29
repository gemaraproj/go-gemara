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

## OSCAL Persistence Rules

The OSCAL metaschema defines two persistence models for UUIDs. The source-string
convention follows both.

### `change-on-write` (document roots)

Catalog and Profile root UUIDs declare `identifier-persistence: change-on-write`
in their metaschema. The [OSCAL Model Overview](https://pages.nist.gov/OSCAL/learn/concepts/layer/overview/)
states: "Every time the content of an OSCAL file changes, a new UUID value must
be generated and assigned to the root element's uuid."

For `gemaraconv`, the conversion is a pure function: same Gemara input at the same
version always produces the same OSCAL. The version bump is the `change-on-write`
trigger. Source string: `gemara:{metadata.Id}/{version}/{document-type}`.

### `per-subject` (assembly UUIDs)

Every inner assembly UUID (finding, observation, result, party, log entry,
resource) declares in its metaschema description: "This UUID should be assigned
per-subject, which means it should be consistently used to identify the same
subject across revisions of the document."

Source strings use identity fields only (IDs, structural keys), not content
fields. Changing a control's description or an assessment's result does not change
the UUID because the subject (the control, the requirement) has not changed.

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
| Catalog (control) | `change-on-write` | `gemara:{id}/{version}/catalog` | Document ID + version |
| Catalog (guidance) | `change-on-write` | `gemara:{id}/{version}/guidance-catalog` | Document ID + version |
| Profile | `change-on-write` | `gemara:{id}/{version}/guidance-profile` | Document ID + version |
| AssessmentResults | `change-on-write` | `gemara:{id}/{version}/assessment-results` | Document ID + version |
| Party | `per-subject` | `gemara:{authorName}/{authorType}/party` | Author name + type |
| Resource | `per-subject` | `gemara:{ref.Id}/resource` | Mapping reference ID |
| Result | `per-subject` | `gemara:{log.Metadata.Id}/result` | Log ID |
| Finding | `per-subject` | `gemara:{logId}/{controlId}/finding` | Log ID + control entry ID |
| Observation | `per-subject` | `gemara:{logId}/{controlId}/{requirementId}/observation` | Log ID + control + requirement |
| LogEntry | `per-subject` | `gemara:{logId}/{controlId}/{requirementId}/log-entry` | Log ID + control + requirement |
| InventoryItem | `per-subject` | `gemara:{target.Id}/inventory-item` | Target resource ID |

All `{field}` placeholders denote `url.PathEscape(field)` values. Document-root
UUIDs use `options.version`, which defaults to `metadata.Version` and falls back
to `"0.0.0"` when both are empty.

### Fields Excluded from Source Strings

Content and descriptive fields are excluded because they are properties of the
subject, not its identity:

- `eval.Result`, `eval.Message` (finding verdict and description)
- `alog.Result`, `alog.Description`, `alog.Recommendation`, `alog.Evidence`
- `ref.Title`, `ref.Url`, `ref.Version`, `ref.Description`
- `target.Name`, `target.Description`
- `confidence-level` (in the Breaking Changes milestone, gemara#497)
- `executor` (attribution, not identity, per gemara PR #506)

## Behavior

- **Same input, same output.** Converting the same Gemara document twice produces
  identical OSCAL UUIDs.
- **Version bump changes document UUID.** Bumping `metadata.version` changes the
  root UUID (Catalog, Profile, AssessmentResults) but assembly UUIDs stay the
  same.
- **Content changes do not change assembly UUIDs.** Editing a finding's message,
  an observation's description, or a resource's title does not change its UUID.
- **Different subjects get different UUIDs.** Each assembly type derives its UUID
  from its own identity fields (see the mapping table above). Different values in
  those fields produce different UUIDs.
- **No collision across assembly types.** A Finding and an Observation for the
  same control+requirement get different UUIDs because of the entity-type suffix.
