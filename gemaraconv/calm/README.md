# ControlCatalog → CALM controls

Converts a Gemara `ControlCatalog` into a FINOS CALM (release/1.2) `controls` block. Entry: `gemaraconv.ControlCatalog(c).ToCALM()`. One opinionated conversion, no options.

Each `config` conforms to the Gemara CALM control-requirement Standard, published at its canonical `$id` `https://gemara.openssf.org/schema/calm/v1/gemara-control-requirement.json`, which CALM's `requirement-url` points at so downstream validators can confirm this library produces a valid artifact. The Standard is also embedded in the package and validated offline in tests, so conversion does not depend on the URL resolving.

Unlike other gemaraconv processes, this is a **deliberately lossy conversion** focusing on producing a library of requirements, with links to the source artifact for additional context. CALM does not support complete Gemara control catalog artifacts.

## Shape

- One CALM **control-name per Gemara control** — key = normalized control id,
  description = `title — objective` — with each of the control's **assessment
  requirements** as a `control-detail` (always an inline `config`, never
  `config-url`).
- This maps the two most meaningful source levels (control, requirement) onto
  CALM's two-level `name → requirements[]` — the highest-fidelity projection the
  shape allows. (Catalog and group survive as config fields, not structure.)
- Each `config` is self-contained: identity + provenance repeat per requirement so
  a single requirement resolves standalone.

## Carried

`control-id` (AR id), `gemara-control-id`, `name` (control title), `description` (AR text), `group`, `applicability`, `recommendation`, `state` (AR lifecycle), `catalog-author` (`metadata.author.id`), `catalog-id`, `catalog-version`.

## Omitted — intentional, not oversight

| Dropped | Why |
|---|---|
| `threats`, `guidelines` (framework crosswalk) | mappings; no CALM controls-block home |
| `extends` / `imports` | out of scope until go-gemara has better support for catalog composition |
| `replaced-by` | mapping |
| control-level `state` | redundant with AR-level `state` |
| `objective` | folded into the control-name description (`title — objective`) |
| catalog `Groups` (per-group title/description) | only the bare group id is carried in each config; ids are stable join keys |
| applicability / mapping-reference dictionaries | some precision loss, but ids are stable enough join keys for this purpose |

If we want to revise to include dropped data in the future: Extend the Standard (`allOf`, no `additionalProperties:false` — additive, non-breaking).

## Provenance

`catalog-author` (`metadata.author.id`), `catalog-id` (`metadata.id`), **and** `catalog-version` are all **required** (Standard: `minLength 1` each) and together are the traceability anchor — a consuming tool resolves them back to the source release (e.g. builds the grc.store release URL `{author}/{id}/versions/{version}`). No `source-url` is emitted: the URL is derivable from the coordinates, so carrying it would be redundant.

The three are carried **verbatim as independent values** — we don't merge or reformat them, because a registry composes them its own way (grc.store uses `{author}/{id}`) and we can't assume how. `catalog-id` alone doesn't name *which* registry to resolve against, so the coordinates disambiguate a release within a known registry, not across registries.

Input is trusted as a valid, vetted Gemara catalog: the converter does no field validation and carries values through as-is.

## Gotchas

- **Control-name keys must be unique, or it errors — never silently merges.**
  Distinct control ids that normalize alike (`a.b` vs `a b` → `a-b`) *and*
  exact-duplicate control ids both error.
- Prose fields (`name`, `description`, `recommendation`, and the control-name
  description) have folded-YAML newlines collapsed to single spaces.
- `calm validate` does **not** check configs against `requirement-url`; the
  conformance guard lives in `calm_test.go` (validates against the Standard *and*
  the vendored upstream `control.json`).
- `testdata/` holds two kinds of file — don't hand-edit either:
  - **Generated golden** — `ccc.marefarc.cn.controls.json`, the full conversion of
    `ccc.marefarc.cn.yaml`. Named as the real emitted artifact, not `*.golden.*`.
    Regenerate with `go test ./gemaraconv/ -run CCCFixture -update`.
  - **Vendored upstream schemas** — `control.json`, `control-requirement.json`,
    pristine copies from CALM for offline validation / drift comparison.
