package markdown

import (
	"sort"

	"github.com/gemaraproj/go-gemara"
)

// markdownThreatCatalogView is the template root for threat catalog rendering.
type markdownThreatCatalogView struct {
	Title        string
	Metadata     gemara.Metadata
	ShowMetadata bool
	Extends      []gemara.ArtifactMapping
	Imports      []markdownImportView
	TOC          bool
	LineEnding   string
	Groups       []markdownThreatGroupView
	TOCItems     []markdownTOCItem
	NumThreats   int
	// LexiconGlossary is non-empty when lexicon autolink loaded a valid document.
	LexiconGlossary []markdownLexiconGlossaryEntry
}

type markdownThreatGroupView struct {
	ID          string
	Title       string
	Description string
	Anchor      string
	IsUngrouped bool
	Threats     []gemara.Threat
}

func buildMarkdownThreatCatalogView(catalog *gemara.ThreatCatalog, cfg Config, lexGlossary []markdownLexiconGlossaryEntry) markdownThreatCatalogView {
	if catalog == nil {
		return markdownThreatCatalogView{LineEnding: cfg.LineEnding, LexiconGlossary: lexGlossary}
	}

	known := make(map[string]struct{}, len(catalog.Groups))
	for _, g := range catalog.Groups {
		known[g.Id] = struct{}{}
	}

	byGroup := make(map[string][]gemara.Threat)
	var orphans []gemara.Threat
	for _, t := range catalog.Threats {
		if _, ok := known[t.Group]; ok {
			byGroup[t.Group] = append(byGroup[t.Group], t)
		} else {
			orphans = append(orphans, t)
		}
	}

	var groups []markdownThreatGroupView
	var toc []markdownTOCItem

	appendGroup := func(gv markdownThreatGroupView) {
		groups = append(groups, gv)
		if !cfg.TOC {
			return
		}
		toc = append(toc, markdownTOCItem{Label: gv.Title, Anchor: gv.Anchor, Indent: 0})
		for _, t := range gv.Threats {
			toc = append(toc, markdownTOCItem{
				Label:  t.Id + ": " + t.Title,
				Anchor: Anchor(t.Id + ": " + t.Title),
				Indent: 1,
			})
		}
	}

	for _, g := range catalog.Groups {
		threats := append([]gemara.Threat(nil), byGroup[g.Id]...)
		if len(threats) == 0 {
			continue
		}
		sort.Slice(threats, func(i, j int) bool { return threats[i].Id < threats[j].Id })
		anchor := Anchor(g.Id)
		if anchor == "" {
			anchor = Anchor(g.Title)
		}
		appendGroup(markdownThreatGroupView{
			ID:          g.Id,
			Title:       g.Title,
			Description: g.Description,
			Anchor:      anchor,
			Threats:     threats,
		})
	}

	if len(orphans) > 0 {
		sort.Slice(orphans, func(i, j int) bool { return orphans[i].Id < orphans[j].Id })
		appendGroup(markdownThreatGroupView{
			Title:       ungroupedSectionTitle,
			Description: "Threats whose group id is not listed in the catalog groups.",
			Anchor:      Anchor(ungroupedSectionTitle),
			IsUngrouped: true,
			Threats:     orphans,
		})
	}

	return markdownThreatCatalogView{
		Title:           catalog.Title,
		Metadata:        catalog.Metadata,
		ShowMetadata:    cfg.Metadata,
		Extends:         catalog.Extends,
		Imports:         buildThreatImportViews(catalog),
		TOC:             cfg.TOC,
		LineEnding:      cfg.LineEnding,
		Groups:          groups,
		TOCItems:        toc,
		NumThreats:      len(catalog.Threats),
		LexiconGlossary: lexGlossary,
	}
}

// buildThreatImportViews resolves each Import's ReferenceId against Metadata.MappingReferences.
func buildThreatImportViews(catalog *gemara.ThreatCatalog) []markdownImportView {
	if len(catalog.Imports) == 0 {
		return nil
	}
	refMap := make(map[string]gemara.MappingReference, len(catalog.Metadata.MappingReferences))
	for _, ref := range catalog.Metadata.MappingReferences {
		refMap[ref.Id] = ref
	}
	views := make([]markdownImportView, len(catalog.Imports))
	for i, imp := range catalog.Imports {
		v := markdownImportView{
			ReferenceId: imp.ReferenceId,
			Remarks:     imp.Remarks,
			Entries:     imp.Entries,
		}
		if ref, ok := refMap[imp.ReferenceId]; ok {
			v.Title = ref.Title
			v.Url = ref.Url
		}
		views[i] = v
	}
	return views
}
