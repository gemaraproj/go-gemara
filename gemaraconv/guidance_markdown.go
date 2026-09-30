package gemaraconv

import (
	"context"

	"github.com/gemaraproj/go-gemara"
	"github.com/gemaraproj/go-gemara/gemaraconv/markdown"
)

// GuidanceToMarkdown renders a GuidanceCatalog as Markdown using embedded templates.
// Only guidelines whose state is LifecycleActive are included (TOC, body, and summary counts).
// [WithApplicabilityMatrix] has no effect: guidance has no assessment requirements.
func GuidanceToMarkdown(ctx context.Context, guidance gemara.GuidanceCatalog, opts ...MarkdownOption) ([]byte, error) {
	o := defaultMarkdownOpts()
	o.apply(opts...)
	return markdown.GuidanceToMarkdown(ctx, guidance, o.config())
}
