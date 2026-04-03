package markdown

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeMarkdownTableCell_newlinesAndPipe(t *testing.T) {
	assert.Equal(t, `Enforce Least Privilege on CI/CD Pipelines`, sanitizeMarkdownTableCell("Enforce Least Privilege on CI/CD Pipelines\n"))
	assert.Equal(t, `a \| b`, sanitizeMarkdownTableCell("a | b"))
	assert.Equal(t, `one two`, sanitizeMarkdownTableCell("one\n\ttwo"))
}
