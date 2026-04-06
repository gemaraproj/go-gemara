// SPDX-License-Identifier: Apache-2.0

package gemara

import (
	"testing"

	"github.com/gemaraproj/go-gemara/internal/codec"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestSControl_RoundTrip(t *testing.T) {
	original := Control{
		Id:        "TEST-01",
		Title:     "Test Control",
		Objective: "Verify round-trip marshaling",
		Group:     "test-group",
		State:     LifecycleActive,
		AssessmentRequirements: []AssessmentRequirement{
			{
				Id:            "TEST-01.01",
				Text:          "Verify the control is active",
				Applicability: []string{"all"},
				State:         LifecycleActive,
			},
		},
	}

	sc := original.Sugar()

	yamlBytes, err := codec.MarshalYAML(sc)
	require.NoError(t, err)

	var roundTripped SControl
	require.NoError(t, codec.UnmarshalYAML(yamlBytes, &roundTripped))

	if diff := cmp.Diff(original, roundTripped.ToBase()); diff != "" {
		t.Errorf("control mismatch (-original +roundtripped):\n%s", diff)
	}
}
