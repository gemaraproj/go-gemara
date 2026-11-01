// SPDX-License-Identifier: Apache-2.0

package gemara_test

import (
	"testing"

	"github.com/gemaraproj/go-gemara"
	"github.com/goccy/go-yaml"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRequirements_WorkedExample(t *testing.T) {
	// Base Control Catalog: SEC.SLAM.CM
	baseCatalog := &gemara.ControlCatalog{
		Title: "Container Management Security Controls",
		Metadata: gemara.Metadata{
			Id:            "SEC.SLAM.CM",
			Type:          gemara.ControlCatalogArtifact,
			GemaraVersion: "1.2.0",
			Version:       "1.0.0",
		},
		Controls: []gemara.Control{
			{
				Id:        "SEC.SLAM.CM.CTL02",
				Title:     "Require TLS/SSL with Certificate Pinning",
				Objective: "Protect registry traffic and verify artifact integrity.",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{
						Id:            "SEC.SLAM.CM.CTL02.AR01",
						Text:          "The system MUST use TLS/SSL for all registry communication.",
						Applicability: []string{"all_deployments"},
					},
					{
						Id:            "SEC.SLAM.CM.CTL02.AR02",
						Text:          "On untrusted networks, the system MUST use a VPN.",
						Applicability: []string{"untrusted_networks"},
					},
					{
						Id:            "SEC.SLAM.CM.CTL02.AR03",
						Text:          "The system MUST verify image digests before execution.",
						Applicability: []string{"all_deployments"},
					},
				},
			},
		},
	}

	// Policy: org-policy-001
	policy := &gemara.Policy{
		Title: "Production Infrastructure Security Policy",
		Metadata: gemara.Metadata{
			Id:            "org-policy-001",
			Type:          gemara.PolicyArtifact,
			GemaraVersion: "1.2.0",
			MappingReferences: []gemara.MappingReference{
				{
					Id:      "SEC.SLAM.CM",
					Title:   "Container Management Security Controls",
					Version: "1.0.0",
				},
			},
		},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{
				{
					ReferenceId: "SEC.SLAM.CM",
					AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
						// Override AR01
						{
							Id:                    "MOD-AR01",
							TargetId:              "SEC.SLAM.CM.CTL02.AR01",
							ModificationType:      gemara.ModOverride,
							ModificationRationale: "Mandate certificate pinning across all environments.",
							Text:                  "The system MUST use TLS/SSL and MUST pin to the trusted registry certificate authority.",
							Applicability:         []string{"all_deployments", "edge_gateways"},
						},
						// Modify AR02: Update text only (applicability preserved!)
						{
							Id:                    "MOD-AR02",
							TargetId:              "SEC.SLAM.CM.CTL02.AR02",
							ModificationType:      gemara.ModModify,
							ModificationRationale: "Permit WireGuard tunnels in addition to corporate VPN.",
							Text:                  "On untrusted networks, the system MUST use an approved WireGuard or IPSec VPN.",
						},
						// Add AR04: Inject a new requirement
						{
							Id:                    "MOD-AR04",
							TargetId:              "SEC.SLAM.CM.CTL02.AR04",
							ModificationType:      gemara.ModAdd,
							ModificationRationale: "Enforce Sigstore cosign cryptographic verification.",
							Text:                  "The system MUST verify signatures using the organizational Sigstore keyless cosign root.",
							Applicability:         []string{"production"},
						},
					},
				},
			},
		},
	}

	eff, err := gemara.ResolveRequirements(policy, baseCatalog, "SEC.SLAM.CM")
	require.NoError(t, err)
	require.NotNil(t, eff)

	assert.Equal(t, "org-policy-001", eff.ResolvedFromPolicy)
	require.Len(t, eff.Controls, 1)

	ctrl := eff.Controls[0]
	assert.Equal(t, "SEC.SLAM.CM.CTL02", ctrl.Id)
	require.Len(t, ctrl.AssessmentRequirements, 4)

	// AR01: Overridden
	ar01 := ctrl.AssessmentRequirements[0]
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR01", ar01.Id)
	assert.Equal(t, "The system MUST use TLS/SSL and MUST pin to the trusted registry certificate authority.", ar01.Text)
	assert.Equal(t, []string{"all_deployments", "edge_gateways"}, ar01.Applicability)

	// AR02: Modified text, preserved applicability
	ar02 := ctrl.AssessmentRequirements[1]
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR02", ar02.Id)
	assert.Equal(t, "On untrusted networks, the system MUST use an approved WireGuard or IPSec VPN.", ar02.Text)
	assert.Equal(t, []string{"untrusted_networks"}, ar02.Applicability)

	// AR03: Unmodified
	ar03 := ctrl.AssessmentRequirements[2]
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR03", ar03.Id)
	assert.Equal(t, "The system MUST verify image digests before execution.", ar03.Text)
	assert.Equal(t, []string{"all_deployments"}, ar03.Applicability)

	// AR04: Added
	ar04 := ctrl.AssessmentRequirements[3]
	assert.Equal(t, "SEC.SLAM.CM.CTL02.AR04", ar04.Id)
	assert.Equal(t, "The system MUST verify signatures using the organizational Sigstore keyless cosign root.", ar04.Text)
	assert.Equal(t, []string{"production"}, ar04.Applicability)

	// Provenance checks
	prov01 := eff.GetProvenance("SEC.SLAM.CM.CTL02.AR01")
	require.NotNil(t, prov01)
	assert.Equal(t, "MOD-AR01", prov01.ModifierID)
	assert.Equal(t, gemara.ModOverride, prov01.ModificationType)

	prov04 := eff.GetProvenance("SEC.SLAM.CM.CTL02.AR04")
	require.NotNil(t, prov04)
	assert.Equal(t, "MOD-AR04", prov04.ModifierID)
	assert.Equal(t, gemara.ModAdd, prov04.ModificationType)
}

func TestResolveRequirements_Modifications(t *testing.T) {
	baseCatalog := &gemara.ControlCatalog{
		Metadata: gemara.Metadata{
			Id:     "CAT1",
			Type:   gemara.ControlCatalogArtifact,
			Author: gemara.Actor{Type: gemara.Human},
		},
		Controls: []gemara.Control{
			{
				Id: "CTRL1",
				AssessmentRequirements: []gemara.AssessmentRequirement{
					{Id: "CTRL1.AR1", Text: "Original AR1", Applicability: []string{"app1"}},
					{Id: "CTRL1.AR2", Text: "Original AR2", Applicability: []string{"app2"}},
					{Id: "CTRL1.AR3", Text: "Original AR3", Applicability: []string{"app3"}},
				},
			},
		},
	}

	t.Run("remove requirement", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:               "MOD-REM",
								TargetId:         "CTRL1.AR2",
								ModificationType: gemara.ModRemove,
							},
						},
					},
				},
			},
		}

		eff, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		require.NoError(t, err)
		assert.Len(t, eff.Controls[0].AssessmentRequirements, 2)
		assert.Equal(t, "CTRL1.AR1", eff.Controls[0].AssessmentRequirements[0].Id)
		assert.Equal(t, "CTRL1.AR3", eff.Controls[0].AssessmentRequirements[1].Id)
	})

	t.Run("replace requirement", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:               "MOD-REP",
								TargetId:         "CTRL1.AR1",
								ModificationType: gemara.ModReplace,
								Text:             "Replaced AR1 text",
								Applicability:    []string{"new_app"},
							},
						},
					},
				},
			},
		}

		eff, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		require.NoError(t, err)
		assert.Equal(t, "Replaced AR1 text", eff.Controls[0].AssessmentRequirements[0].Text)
		assert.Equal(t, []string{"new_app"}, eff.Controls[0].AssessmentRequirements[0].Applicability)
	})

	t.Run("override requires rationale", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:               "MOD-OVR",
								TargetId:         "CTRL1.AR1",
								ModificationType: gemara.ModOverride,
								Text:             "Override text",
							},
						},
					},
				},
			},
		}

		_, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		assert.ErrorContains(t, err, "modification-rationale is required")
	})

	t.Run("add existing requirement fails", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:               "MOD-ADD",
								TargetId:         "CTRL1.AR1",
								ModificationType: gemara.ModAdd,
								Text:             "New text",
							},
						},
					},
				},
			},
		}

		_, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		assert.ErrorContains(t, err, "already exists")
	})

	t.Run("modify non-existent requirement fails", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:               "MOD-MOD",
								TargetId:         "CTRL1.NONEXISTENT",
								ModificationType: gemara.ModModify,
								Text:             "New text",
							},
						},
					},
				},
			},
		}

		_, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		assert.ErrorContains(t, err, "not found")
	})

	t.Run("exclusions applied before modifications", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						Exclusions:  []string{"CTRL1.AR3"},
					},
				},
			},
		}

		eff, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		require.NoError(t, err)
		assert.Len(t, eff.Controls[0].AssessmentRequirements, 2)
		assert.Len(t, eff.FindControl("CTRL1").AssessmentRequirements, 2)
		_, req3 := eff.FindRequirement("CTRL1.AR3")
		assert.Nil(t, req3)
	})

	t.Run("constraint validation fails when target missing", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						Constraints: []gemara.Constraint{
							{
								Id:       "C1",
								TargetId: "NON_EXISTENT",
								Text:     "Must exist",
							},
						},
					},
				},
			},
		}

		_, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		assert.ErrorContains(t, err, "target \"NON_EXISTENT\" not found")
	})

	t.Run("yaml round trip preserves effective metadata", func(t *testing.T) {
		policy := &gemara.Policy{
			Metadata: gemara.Metadata{Id: "P1"},
			Imports: gemara.Imports{
				Catalogs: []gemara.CatalogImport{
					{
						ReferenceId: "CAT1",
						AssessmentRequirementModifications: []gemara.AssessmentRequirementModifier{
							{
								Id:                    "MOD-1",
								TargetId:              "CTRL1.AR1",
								ModificationType:      gemara.ModOverride,
								ModificationRationale: "Mandatory override",
								Text:                  "New text",
							},
						},
					},
				},
			},
		}

		eff, err := gemara.ResolveRequirements(policy, baseCatalog, "CAT1")
		require.NoError(t, err)

		data, err := yaml.Marshal(eff)
		require.NoError(t, err)

		var roundTripped gemara.EffectiveCatalog
		err = yaml.Unmarshal(data, &roundTripped)
		require.NoError(t, err)

		assert.Equal(t, eff.ResolvedFromPolicy, roundTripped.ResolvedFromPolicy)
		require.Len(t, roundTripped.Provenance, len(eff.Provenance))
		assert.Equal(t, eff.Provenance[0].RequirementID, roundTripped.Provenance[0].RequirementID)
		assert.Equal(t, eff.Controls[0].AssessmentRequirements[0].Text, roundTripped.Controls[0].AssessmentRequirements[0].Text)
	})
}
