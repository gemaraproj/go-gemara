package markdown

import (
	"sort"

	"github.com/gemaraproj/go-gemara"
)

// markdownGuidanceView is the template root for a GuidanceCatalog: the same
// shape as markdownCatalogView where the shared templates (TOC, extends and
// imports, mapping references, lexicon) read it, plus the guidance-only parts.
type markdownGuidanceView struct {
	Title        string
	Metadata     gemara.Metadata
	ShowMetadata bool
	GuidanceType gemara.GuidanceType
	FrontMatter  string
	Extends      []gemara.ArtifactMapping
	Imports      []markdownImportView
	EntryNoun    string
	TOC          bool
	LineEnding   string
	Groups       []markdownGuidanceGroupView
	TOCItems     []markdownTOCItem
	Exemptions   []gemara.Exemption
	// NumGuidelines and NumStatements count only active guidelines.
	NumGuidelines   int
	NumStatements   int
	LexiconGlossary []markdownLexiconGlossaryEntry
}

type markdownGuidanceGroupView struct {
	ID          string
	Title       string
	Description string
	Anchor      string
	IsUngrouped bool
	Guidelines  []markdownGuidelineView
}

// markdownGuidelineView is a Guideline plus its see-also ids resolved to
// heading anchors (Anchor is empty when the id is not in this catalog).
type markdownGuidelineView struct {
	gemara.Guideline
	SeeAlsoLinks []markdownLink
}

type markdownLink struct {
	Label  string
	Anchor string
}

func guidelineHeading(g gemara.Guideline) string { return g.Id + ": " + g.Title }

func buildMarkdownGuidanceView(guidance gemara.GuidanceCatalog, cfg Config, lexGlossary []markdownLexiconGlossaryEntry) markdownGuidanceView {
	known := make(map[string]struct{}, len(guidance.Groups))
	for _, g := range guidance.Groups {
		known[g.Id] = struct{}{}
	}
	anchors := make(map[string]string, len(guidance.Guidelines))

	byGroup := make(map[string][]gemara.Guideline)
	var orphans []gemara.Guideline
	numGuidelines, numStatements := 0, 0
	for _, g := range guidance.Guidelines {
		if g.State != gemara.LifecycleActive {
			continue
		}
		numGuidelines++
		numStatements += len(g.Statements)
		anchors[g.Id] = Anchor(guidelineHeading(g))
		if _, ok := known[g.Group]; ok {
			byGroup[g.Group] = append(byGroup[g.Group], g)
		} else {
			orphans = append(orphans, g)
		}
	}

	var groups []markdownGuidanceGroupView
	var toc []markdownTOCItem
	appendGroup := func(gv markdownGuidanceGroupView, src []gemara.Guideline) {
		sort.Slice(src, func(i, j int) bool { return src[i].Id < src[j].Id })
		for _, g := range src {
			v := markdownGuidelineView{Guideline: g}
			for _, id := range g.SeeAlso {
				v.SeeAlsoLinks = append(v.SeeAlsoLinks, markdownLink{Label: id, Anchor: anchors[id]})
			}
			gv.Guidelines = append(gv.Guidelines, v)
		}
		groups = append(groups, gv)
		if !cfg.TOC {
			return
		}
		toc = append(toc, markdownTOCItem{Label: gv.Title, Anchor: gv.Anchor})
		for _, g := range src {
			toc = append(toc, markdownTOCItem{Label: guidelineHeading(g), Anchor: anchors[g.Id], Indent: 1})
		}
	}

	for _, g := range guidance.Groups {
		if len(byGroup[g.Id]) == 0 {
			continue
		}
		anchor := Anchor(g.Id)
		if anchor == "" {
			anchor = Anchor(g.Title)
		}
		appendGroup(markdownGuidanceGroupView{ID: g.Id, Title: g.Title, Description: g.Description, Anchor: anchor}, append([]gemara.Guideline(nil), byGroup[g.Id]...))
	}
	if len(orphans) > 0 {
		appendGroup(markdownGuidanceGroupView{
			Title:       ungroupedSectionTitle,
			Description: "Guidelines whose group id is not listed in the catalog groups.",
			Anchor:      Anchor(ungroupedSectionTitle),
			IsUngrouped: true,
		}, orphans)
	}

	return markdownGuidanceView{
		Title:           guidance.Title,
		Metadata:        guidance.Metadata,
		ShowMetadata:    cfg.Metadata,
		GuidanceType:    guidance.GuidanceType,
		FrontMatter:     guidance.FrontMatter,
		Extends:         guidance.Extends,
		Imports:         buildImportViews(guidance.Imports, guidance.Metadata),
		EntryNoun:       "guidelines",
		TOC:             cfg.TOC,
		LineEnding:      cfg.LineEnding,
		Groups:          groups,
		TOCItems:        toc,
		Exemptions:      guidance.Exemptions,
		NumGuidelines:   numGuidelines,
		NumStatements:   numStatements,
		LexiconGlossary: lexGlossary,
	}
}
