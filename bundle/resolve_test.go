// SPDX-License-Identifier: Apache-2.0

package bundle_test

import (
	"context"
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/bundle"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"oras.land/oras-go/v2/content/memory"
)

func TestBundle_ResolveCatalog_WorkedExample(t *testing.T) {
	// Base Control Catalog YAML
	catYAML := `
title: "Container Management Security Controls"
metadata:
  id: "SEC.SLAM.CM"
  type: "ControlCatalog"
  gemara-version: "1.2.0"
  version: "1.0.0"

controls:
  - id: "SEC.SLAM.CM.CTL02"
    title: "Require TLS/SSL with Certificate Pinning"
    objective: "Protect registry traffic and verify artifact integrity."
    assessment-requirements:
      - id: "SEC.SLAM.CM.CTL02.AR01"
        text: "The system MUST use TLS/SSL for all registry communication."
        applicability: ["all_deployments"]
      - id: "SEC.SLAM.CM.CTL02.AR02"
        text: "On untrusted networks, the system MUST use a VPN."
        applicability: ["untrusted_networks"]
      - id: "SEC.SLAM.CM.CTL02.AR03"
        text: "The system MUST verify image digests before execution."
        applicability: ["all_deployments"]
`

	// Policy YAML with overrides and additions
	polYAML := `
title: "Production Infrastructure Security Policy"
metadata:
  id: "org-policy-001"
  type: "Policy"
  gemara-version: "1.2.0"
  mapping-references:
    - id: "SEC.SLAM.CM"
      title: "Container Management Security Controls"
      version: "1.0.0"

imports:
  catalogs:
    - reference-id: "SEC.SLAM.CM"
      assessment-requirement-modifications:
        - id: "MOD-AR01"
          target-id: "SEC.SLAM.CM.CTL02.AR01"
          modification-type: "Override"
          modification-rationale: "Mandate certificate pinning across all environments."
          text: "The system MUST use TLS/SSL and MUST pin to the trusted registry certificate authority."
          applicability: ["all_deployments", "edge_gateways"]

        - id: "MOD-AR02"
          target-id: "SEC.SLAM.CM.CTL02.AR02"
          modification-type: "Modify"
          modification-rationale: "Permit WireGuard tunnels in addition to corporate VPN."
          text: "On untrusted networks, the system MUST use an approved WireGuard or IPSec VPN."

        - id: "MOD-AR04"
          target-id: "SEC.SLAM.CM.CTL02.AR04"
          modification-type: "Add"
          modification-rationale: "Enforce Sigstore cosign cryptographic verification."
          text: "The system MUST verify signatures using the organizational Sigstore keyless cosign root."
          applicability: ["production"]
`

	b := &bundle.Bundle{
		Manifest: bundle.Manifest{
			BundleVersion: "1.0.0",
			GemaraVersion: "1.2.0",
		},
		Source: bundle.File{
			Name: "policy.yaml",
			Type: "Policy",
			Data: []byte(polYAML),
		},
		Imports: []bundle.File{
			{
				Name: "controls.yaml",
				Type: "ControlCatalog",
				Data: []byte(catYAML),
			},
		},
	}

	effCat, err := b.ResolveCatalog()
	require.NoError(t, err)
	require.NotNil(t, effCat)

	assert.Equal(t, "org-policy-001", effCat.ResolvedFromPolicy)
	require.Len(t, effCat.Controls, 1)

	ctrl := effCat.Controls[0]
	assert.Equal(t, "SEC.SLAM.CM.CTL02", ctrl.Id)
	require.Len(t, ctrl.AssessmentRequirements, 4)

	// AR01: Overridden
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR01", ctrl.AssessmentRequirements[0].Id)
	assert.Equal(t, "The system MUST use TLS/SSL and MUST pin to the trusted registry certificate authority.", ctrl.AssessmentRequirements[0].Text)
	assert.Equal(t, []string{"all_deployments", "edge_gateways"}, ctrl.AssessmentRequirements[0].Applicability)

	// AR02: Modified text, preserved applicability
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR02", ctrl.AssessmentRequirements[1].Id)
	assert.Equal(t, "On untrusted networks, the system MUST use an approved WireGuard or IPSec VPN.", ctrl.AssessmentRequirements[1].Text)
	assert.Equal(t, []string{"untrusted_networks"}, ctrl.AssessmentRequirements[1].Applicability)

	// AR03: Unmodified baseline
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR03", ctrl.AssessmentRequirements[2].Id)
	assert.Equal(t, "The system MUST verify image digests before execution.", ctrl.AssessmentRequirements[2].Text)
	assert.Equal(t, []string{"all_deployments"}, ctrl.AssessmentRequirements[2].Applicability)

	// AR04: Added
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR04", ctrl.AssessmentRequirements[3].Id)
	assert.Equal(t, "The system MUST verify signatures using the organizational Sigstore keyless cosign root.", ctrl.AssessmentRequirements[3].Text)
	assert.Equal(t, []string{"production"}, ctrl.AssessmentRequirements[3].Applicability)
}

func TestBundle_ResolvePolicy(t *testing.T) {
	catYAML := `
metadata:
  id: "CAT-1"
  type: "ControlCatalog"
controls:
  - id: "C1"
    assessment-requirements:
      - id: "C1.AR1"
        text: "Baseline req"
        applicability: ["default"]
`
	polYAML := `
metadata:
  id: "POL-1"
  type: "Policy"
imports:
  catalogs:
    - reference-id: "CAT-1"
      assessment-requirement-modifications:
        - id: "MOD-1"
          target-id: "C1.AR1"
          modification-type: "Modify"
          text: "Updated req text"
`
	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "policy.yaml",
			Type: "Policy",
			Data: []byte(polYAML),
		},
		Imports: []bundle.File{
			{
				Name: "cat.yaml",
				Type: "ControlCatalog",
				Data: []byte(catYAML),
			},
		},
	}

	effPol, err := b.ResolvePolicy()
	require.NoError(t, err)
	require.NotNil(t, effPol)

	assert.Equal(t, "POL-1", effPol.Metadata.Id)
	require.Len(t, effPol.ResolvedCatalogs, 1)

	effCat := effPol.Catalog("CAT-1")
	require.NotNil(t, effCat)
	assert.Equal(t, "Updated req text", effCat.Controls[0].AssessmentRequirements[0].Text)
}

var testActor = gemara.Actor{
	Id:   "test-actor",
	Name: "Test Actor",
	Type: gemara.Human,
}

func TestBundle_ResolveCatalog_Extends(t *testing.T) {
	baseCat := gemara.ControlCatalog{
		Title: "Base Catalog",
		Metadata: gemara.Metadata{
			Id:     "BASE-CAT",
			Type:   gemara.ControlCatalogArtifact,
			Author: testActor,
		},
		Groups: []gemara.Group{
			{Id: "grp-1", Title: "Group 1", Description: "Base group"},
		},
		Controls: []gemara.Control{
			{
				Id:    "CTL-1",
				Title: "Control 1 Base",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-1.AR1", Text: "Base AR1"},
				},
			},
			{
				Id:    "CTL-2",
				Title: "Control 2 Base",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-2.AR1", Text: "Base AR2"},
				},
			},
		},
	}

	childCat := gemara.ControlCatalog{
		Title: "Child Catalog",
		Metadata: gemara.Metadata{
			Id:     "CHILD-CAT",
			Type:   gemara.ControlCatalogArtifact,
			Author: testActor,
			MappingReferences: []gemara.MappingReference{
				{Id: "BASE-CAT"},
			},
		},
		Extends: []gemara.ArtifactMapping{
			{ReferenceId: "BASE-CAT"},
		},
		Controls: []gemara.Control{
			// Overrides CTL-1
			{
				Id:    "CTL-1",
				Title: "Control 1 Child Override",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-1.AR1", Text: "Child overridden AR1"},
				},
			},
			// Adds CTL-3
			{
				Id:    "CTL-3",
				Title: "Control 3 Child",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTL-3.AR1", Text: "Child AR3"},
				},
			},
		},
	}

	baseBytes, err := yaml.Marshal(baseCat)
	require.NoError(t, err)
	childBytes, err := yaml.Marshal(childCat)
	require.NoError(t, err)

	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "child.yaml",
			Type: "ControlCatalog",
			Data: childBytes,
		},
		Imports: []bundle.File{
			{
				Name: "base.yaml",
				Type: "ControlCatalog",
				Data: baseBytes,
			},
		},
	}

	effCat, err := b.ResolveCatalog()
	require.NoError(t, err)
	require.NotNil(t, effCat)

	assert.Equal(t, "Child Catalog", effCat.Title)
	require.Len(t, effCat.Groups, 1)
	assert.Equal(t, "grp-1", effCat.Groups[0].Id)

	// Should have CTL-1 (overridden), CTL-2 (inherited from base), CTL-3 (added by child)
	require.Len(t, effCat.Controls, 3)

	ctl1 := effCat.FindControl("CTL-1")
	require.NotNil(t, ctl1)
	assert.Equal(t, "Control 1 Child Override", ctl1.Title)
	assert.Equal(t, "Child overridden AR1", ctl1.AssessmentRequirements[0].Text)

	ctl2 := effCat.FindControl("CTL-2")
	require.NotNil(t, ctl2)
	assert.Equal(t, "Control 2 Base", ctl2.Title)

	ctl3 := effCat.FindControl("CTL-3")
	require.NotNil(t, ctl3)
	assert.Equal(t, "Control 3 Child", ctl3.Title)
}

func TestBundle_ResolveCatalog_ExtendsCycleDetection(t *testing.T) {
	catA := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:     "CAT-A",
			Type:   gemara.ControlCatalogArtifact,
			Author: testActor,
			MappingReferences: []gemara.MappingReference{
				{Id: "CAT-B"},
			},
		},
		Extends: []gemara.ArtifactMapping{{ReferenceId: "CAT-B"}},
	}
	catB := gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:     "CAT-B",
			Type:   gemara.ControlCatalogArtifact,
			Author: testActor,
			MappingReferences: []gemara.MappingReference{
				{Id: "CAT-A"},
			},
		},
		Extends: []gemara.ArtifactMapping{{ReferenceId: "CAT-A"}},
	}

	bytesA, _ := yaml.Marshal(catA)
	bytesB, _ := yaml.Marshal(catB)

	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "a.yaml",
			Type: "ControlCatalog",
			Data: bytesA,
		},
		Imports: []bundle.File{
			{
				Name: "b.yaml",
				Type: "ControlCatalog",
				Data: bytesB,
			},
		},
	}

	_, err := b.ResolveCatalog()
	assert.ErrorContains(t, err, "cyclic dependency detected in catalog extends")
}

func TestBundle_ResolvePolicy_CycleDetection(t *testing.T) {
	polA := gemara.Policy{
		Metadata: gemara.Metadata{
			Id:     "POL-A",
			Type:   gemara.PolicyArtifact,
			Author: testActor,
		},
		Imports: gemara.Imports{
			Policies: []gemara.ArtifactMapping{{ReferenceId: "POL-B"}},
		},
	}
	polB := gemara.Policy{
		Metadata: gemara.Metadata{
			Id:     "POL-B",
			Type:   gemara.PolicyArtifact,
			Author: testActor,
		},
		Imports: gemara.Imports{
			Policies: []gemara.ArtifactMapping{{ReferenceId: "POL-A"}},
		},
	}

	bytesA, _ := yaml.Marshal(polA)
	bytesB, _ := yaml.Marshal(polB)

	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "polA.yaml",
			Type: "Policy",
			Data: bytesA,
		},
		Imports: []bundle.File{
			{
				Name: "polB.yaml",
				Type: "Policy",
				Data: bytesB,
			},
		},
	}

	_, err := b.ResolvePolicy()
	assert.ErrorContains(t, err, "cyclic dependency detected in policy imports")
}

func TestResolveCatalog_OCI(t *testing.T) {
	ctx := context.Background()
	target := memory.New()

	catYAML := `
title: "OCI Test Catalog"
metadata:
  id: "CAT-OCI"
  type: "ControlCatalog"
controls:
  - id: "CTRL-1"
    title: "OCI Control 1"
    assessment-requirements:
      - id: "CTRL-1.AR1"
        text: "Must verify OCI resolution."
        applicability: ["cloud"]
`
	polYAML := `
title: "OCI Test Policy"
metadata:
  id: "POL-OCI"
  type: "Policy"
imports:
  catalogs:
    - reference-id: "CAT-OCI"
      assessment-requirement-modifications:
        - id: "MOD-1"
          target-id: "CTRL-1.AR1"
          modification-type: "Override"
          modification-rationale: "Organizational mandate"
          text: "Overridden text for OCI."
          applicability: ["cloud", "on-prem"]
`
	b := &bundle.Bundle{
		Manifest: bundle.Manifest{
			BundleVersion: "1.0.0",
			GemaraVersion: "1.2.0",
		},
		Source: bundle.File{
			Name: "policy.yaml",
			Type: "Policy",
			Data: []byte(polYAML),
		},
		Imports: []bundle.File{
			{
				Name: "controls.yaml",
				Type: "ControlCatalog",
				Data: []byte(catYAML),
			},
		},
	}

	tag := "v1.0.0"
	desc, err := bundle.Pack(ctx, target, b)
	require.NoError(t, err)
	require.NoError(t, target.Tag(ctx, desc, tag))

	// Resolve via Resolver struct
	resolver := bundle.NewResolver(bundle.WithTarget(target))
	effCat, err := resolver.ResolveCatalog(ctx, tag)
	require.NoError(t, err)
	require.NotNil(t, effCat)

	assert.Equal(t, "POL-OCI", effCat.ResolvedFromPolicy)
	assert.Equal(t, "Overridden text for OCI.", effCat.Controls[0].AssessmentRequirements[0].Text)
	assert.Equal(t, []string{"cloud", "on-prem"}, effCat.Controls[0].AssessmentRequirements[0].Applicability)

	// Resolve Policy directly
	effPol, err := bundle.ResolvePolicy(ctx, target, tag)
	require.NoError(t, err)
	require.NotNil(t, effPol)
	assert.Equal(t, "POL-OCI", effPol.Metadata.Id)
	require.Len(t, effPol.ResolvedCatalogs, 1)
}

func TestBundle_ResolveCatalog_MultipleImports(t *testing.T) {
	cat1YAML := `
metadata:
  id: "CAT-1"
  type: "ControlCatalog"
controls:
  - id: "C1"
`
	cat2YAML := `
metadata:
  id: "CAT-2"
  type: "ControlCatalog"
controls:
  - id: "C2"
`
	polYAML := `
metadata:
  id: "POL-MULTI"
  type: "Policy"
imports:
  catalogs:
    - reference-id: "CAT-1"
    - reference-id: "CAT-2"
`
	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "policy.yaml",
			Type: "Policy",
			Data: []byte(polYAML),
		},
		Imports: []bundle.File{
			{Name: "cat1.yaml", Type: "ControlCatalog", Data: []byte(cat1YAML)},
			{Name: "cat2.yaml", Type: "ControlCatalog", Data: []byte(cat2YAML)},
		},
	}

	// Without WithCatalogID should fail when multiple catalogs exist
	_, err := b.ResolveCatalog()
	assert.ErrorContains(t, err, "use WithCatalogID to select which one to resolve")

	// With WithCatalogID selecting CAT-2
	effCat, err := b.ResolveCatalog(bundle.WithCatalogID("CAT-2"))
	require.NoError(t, err)
	assert.Equal(t, "CAT-2", effCat.Metadata.Id)
	assert.Equal(t, "C2", effCat.Controls[0].Id)
}

func TestBundle_ResolveGuidance(t *testing.T) {
	guideYAML := `
metadata:
  id: "GUIDE-1"
  type: "GuidanceCatalog"
guidelines:
  - id: "GL-1"
    text: "Guideline 1 text"
  - id: "GL-2"
    text: "Guideline 2 text"
`
	polYAML := `
metadata:
  id: "POL-GUIDE"
  type: "Policy"
imports:
  guidance:
    - reference-id: "GUIDE-1"
      exclusions:
        - "GL-2"
`
	b := &bundle.Bundle{
		Source: bundle.File{
			Name: "policy.yaml",
			Type: "Policy",
			Data: []byte(polYAML),
		},
		Imports: []bundle.File{
			{Name: "guide.yaml", Type: "GuidanceCatalog", Data: []byte(guideYAML)},
		},
	}

	effGuide, err := b.ResolveGuidance()
	require.NoError(t, err)
	assert.Equal(t, "GUIDE-1", effGuide.Metadata.Id)
	require.Len(t, effGuide.Guidelines, 1)
	assert.Equal(t, "GL-1", effGuide.Guidelines[0].Id)

	effPol, err := b.ResolvePolicy()
	require.NoError(t, err)
	require.Len(t, effPol.ResolvedGuidance, 1)
	assert.Equal(t, "GUIDE-1", effPol.Guidance("GUIDE-1").Metadata.Id)
}
