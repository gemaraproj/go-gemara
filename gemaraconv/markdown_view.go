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

	var groups []markdownGroupView
	var toc []markdownTOCItem
	numARs := 0
	for _, c := range catalog.Controls {
		numARs += len(c.AssessmentRequirements)
	}

	appendGroup := func(gv markdownGroupView) {
		groups = append(groups, gv)
		if !opts.toc {
			return
		}
		toc = append(toc, markdownTOCItem{Label: gv.Title, Anchor: gv.Anchor, Indent: 0, Control: false})
		for _, ctl := range gv.Controls {
			toc = append(toc, markdownTOCItem{
				Label:   ctl.Id + " — " + ctl.Title,
				Anchor:  markdownAnchor(ctl.Id),
				Indent:  1,
				Control: true,
			})
		}
	}

	// Known groups in catalog order (skip empty).
	for _, g := range catalog.Groups {
		// Copy before sort so we never mutate a future controls_cache slice in place.
		ctrls := append([]gemara.Control(nil), catalog.GetControlsForGroup(g.Id)...)
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

	// Controls whose Group id is not in catalog.Groups.
	var ungrouped []gemara.Control
	for _, c := range catalog.Controls {
		if _, ok := known[c.Group]; !ok {
			ungrouped = append(ungrouped, c)
		}
	}
	sort.Slice(ungrouped, func(i, j int) bool { return ungrouped[i].Id < ungrouped[j].Id })
	if len(ungrouped) > 0 {
		ungrouped = copyControlsWithSortedARs(ungrouped)
		uAnchor := markdownAnchor(ungroupedSectionTitle)
		appendGroup(markdownGroupView{
			ID:          "",
			Title:       ungroupedSectionTitle,
			Description: "Controls whose group id is not listed in the catalog groups.",
			Anchor:      uAnchor,
			IsUngrouped: true,
			Controls:    ungrouped,
		})
	}

	return markdownCatalogView{
		Title:       catalog.Title,
		Metadata:    catalog.Metadata,
		Extends:     catalog.Extends,
		Imports:     catalog.Imports,
		TOC:         opts.toc,
		LineEnding:  opts.lineEnding,
		Groups:      groups,
		TOCItems:    toc,
		NumControls: len(catalog.Controls),
		NumARs:      numARs,
	}
}

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
