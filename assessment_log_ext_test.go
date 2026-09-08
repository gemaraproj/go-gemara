package gemara_test

import (
	"testing"

	gemara "github.com/gemaraproj/go-gemara"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStepIdentitySurvivesInlining lives in an external test package because
// the defect it guards only appears at a consumer's call site. With decodedStep
// and NamedStep inlinable, a call inside a go or defer literal gets its own
// copy of the closure at a different code pointer, so isDecoded and isNamed
// miss it and String falls back to the closure's symbol. The in-package tests
// cannot see this; this one fails if either go:noinline directive is removed.
func TestStepIdentitySurvivesInlining(t *testing.T) {
	passing := func(interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
		return gemara.Passed, "ok", gemara.High
	}

	t.Run("NamedStep in a go literal", func(t *testing.T) {
		done := make(chan gemara.AssessmentStep)
		go func() { done <- gemara.NamedStep("recorded", passing) }()
		assert.Equal(t, "recorded", (<-done).String())
	})

	t.Run("UnmarshalText in a defer literal", func(t *testing.T) {
		var step gemara.AssessmentStep
		func() {
			defer func() { require.NoError(t, step.UnmarshalText([]byte("recorded"))) }()
		}()
		assert.Equal(t, "recorded", step.String())
	})
}
