package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCollapseExtraNewlines(t *testing.T) {
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\n\nb"))
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\n\n\n\n\n\nb"))
	assert.Equal(t, "a\n\nb", collapseExtraNewlines("a\n\nb"))
}
