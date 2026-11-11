package game

import "againrom/pkg/ui"

// modOverview is the overview page of the mod the editor works in: the
// kinds whose files it holds, the report againrom -check prints for it, its
// files, the overrides and warnings applying its data gave, and what keeps
// it from loading or from being written. It reads no map; W shows it first
// for a mod no mission of which opens. It is empty with no mod open.
func (x *MapInspector) modOverview() ui.ModOverviewPage {
	m := x.mod
	if m == nil {
		return ui.ModOverviewPage{}
	}
	var issues []string
	if err := x.modReadOnly(); err != nil {
		issues = append(issues, err.Error())
	}
	if refusal := x.modContextRefusal(); refusal != "" {
		issues = append(issues, refusal)
	}
	if m.refusal != "" {
		issues = append(issues, x.word("editor.mod.overview.refused")+" "+m.refusal)
	}
	_, enabled := m.firstEnabled()
	page := ui.ModOverviewPage{First: !enabled || m.packed}
	for _, s := range []struct {
		heading string
		lines   []string
	}{
		{x.word("editor.mod.overview.kinds"), m.kinds},
		{x.word("editor.mod.overview.report"), m.checkLines},
		{x.word("editor.mod.overview.files"), m.files},
		{x.word("editor.mod.overview.overrides"), append(append([]string(nil), m.report.Overrides...), m.fileOverrides...)},
		{x.word("editor.mod.overview.warnings"), m.report.Warnings},
		{x.word("editor.mod.overview.issues"), issues},
	} {
		lines := s.lines
		if len(lines) == 0 {
			lines = []string{x.word("editor.mod.overview.none")}
		}
		page.Sections = append(page.Sections, ui.ModOverviewSection{Heading: s.heading, Lines: lines})
	}
	return page
}

// ModOverview is W's overview page of the mod the editor works in.
func (s *mapInspectionEdits) ModOverview() ui.ModOverviewPage { return s.inspector.modOverview() }
