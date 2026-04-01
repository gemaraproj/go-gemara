package gemaraconv

import (
	"sort"

	"github.com/gemaraproj/go-gemara"
)

const ungroupedSectionTitle = "Ungrouped"

// markdownCatalogView is the template root: deterministic ordering and explicit Ungrouped bucket.
type markdownCatalogView struct {
	Title       string
	Metadata    gemara.Metadata
	ShowMetadata bool
	Extends     []gemara.ArtifactMapping
	Imports     []gemara.MultiEntryMapping
	TOC         bool
	LineEnding  string
	Groups      []markdownGroupView
	TOCItems    []markdownTOCItem
	NumControls int
	NumARs      int
}

// markdownTOCItem is one line in the table of contents (group or control).
type markdownTOCItem struct {
	Label   string
	Anchor  string
	Indent  int // 0 = group, 1 = control under group
	Control bool
}

type markdownGroupView struct {
	ID          string
	Title       string
	Description string
	Anchor      string
	IsUngrouped bool
	Controls    []gemara.Control
}

func buildMarkdownCatalogView(catalog *gemara.ControlCatalog, opts markdownOpts) markdownCatalogView {
	if catalog == nil {
		return markdownCatalogView{LineEnding: opts.lineEnding}
	}

	known := make(map[string]struct{}, len(catalog.Groups))
	for _, g := range catalog.Groups {
		known[g.Id] = struct{}{}
	}

	byGroup := make(map[string][]gemara.Control)
	var orphans []gemara.Control
	numARs := 0
	numControlsShown := 0
	for _, c := range catalog.Controls {
		if c.State != gemara.LifecycleActive {
			continue
		}
		numControlsShown++
		numARs += len(c.AssessmentRequirements)
		if _, ok := known[c.Group]; ok {
			byGroup[c.Group] = append(byGroup[c.Group], c)
		} else {
			orphans = append(orphans, c)
		}
	}

	var groups []markdownGroupView
	var toc []markdownTOCItem

	appendGroup := func(gv markdownGroupView) {
		groups = append(groups, gv)
		if !opts.toc {
			return
		}
		toc = append(toc, markdownTOCItem{Label: gv.Title, Anchor: gv.Anchor, Indent: 0, Control: false})
		for _, ctl := range gv.Controls {
			toc = append(toc, markdownTOCItem{
				Label:   ctl.Id + ": " + ctl.Title,
				Anchor:  markdownAnchor(ctl.Id + ": " + ctl.Title),
				Indent:  1,
				Control: true,
			})
		}
	}

	for _, g := range catalog.Groups {
		ctrls := append([]gemara.Control(nil), byGroup[g.Id]...)
		if len(ctrls) == 0 {
			continue
		}
		sort.Slice(ctrls, func(i, j int) bool { return ctrls[i].Id < ctrls[j].Id })
		ctrls = copyControlsWithSortedARs(ctrls)
		anchor := markdownAnchor(g.Id)
		if anchor == "" {
			anchor = markdownAnchor(g.Title)
		}
		appendGroup(markdownGroupView{
			ID:          g.Id,
			Title:       g.Title,
			Description: g.Description,
			Anchor:      anchor,
			IsUngrouped: false,
			Controls:    ctrls,
		})
	}

	if len(orphans) > 0 {
		ctrls := append([]gemara.Control(nil), orphans...)
		sort.Slice(ctrls, func(i, j int) bool { return ctrls[i].Id < ctrls[j].Id })
		ctrls = copyControlsWithSortedARs(ctrls)
		uAnchor := markdownAnchor(ungroupedSectionTitle)
		appendGroup(markdownGroupView{
			ID:          "",
			Title:       ungroupedSectionTitle,
			Description: "Controls whose group id is not listed in the catalog groups.",
			Anchor:      uAnchor,
			IsUngrouped: true,
			Controls:    ctrls,
		})
	}

	return markdownCatalogView{
		Title:       catalog.Title,
		Metadata:    catalog.Metadata,
		ShowMetadata: opts.metadata,
		Extends:     catalog.Extends,
		Imports:     catalog.Imports,
		TOC:         opts.toc,
		LineEnding:  opts.lineEnding,
		Groups:      groups,
		TOCItems:    toc,
		NumControls: numControlsShown,
		NumARs:      numARs,
	}
}

// copyControlsWithSortedARs returns a deep copy of ctrls with AssessmentRequirements
// sorted by id for stable Markdown output. The source slice and catalog are not modified.
func copyControlsWithSortedARs(ctrls []gemara.Control) []gemara.Control {
	out := make([]gemara.Control, len(ctrls))
	for i, c := range ctrls {
		out[i] = c
		ars := append([]gemara.AssessmentRequirement(nil), c.AssessmentRequirements...)
		sort.Slice(ars, func(a, b int) bool { return ars[a].Id < ars[b].Id })
		out[i].AssessmentRequirements = ars
	}
	return out
}
