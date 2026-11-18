package game

import (
	"fmt"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// modOverview is the overview page of the mod the editor works in: what it
// changes as one count line per page, the check in a modder's words (each
// warning a line that opens its page), the game files it replaces, what
// keeps it from loading, and folded, the game's own check lines and the
// mod's files. counts are the count lines and pages the page each opens.
// It reads no map; W shows it first for a mod no mission of which opens. It
// is empty with no mod open.
func (x *MapInspector) modOverview(counts, countPages []string) ui.ModOverviewPage {
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
	if len(counts) == 0 {
		counts, countPages = []string{x.word("editor.mod.overview.empty")}, nil
	}
	var check, checkPages []string
	for _, w := range m.report.Warnings {
		page, onPage := warningPage(w)
		line := w
		if onPage {
			line = fmt.Sprintf(x.word("editor.mod.overview.warn"), x.word(modPageTab[page]))
		}
		check = append(check, line)
		checkPages = append(checkPages, page)
	}
	if len(issues) == 0 && len(check) == 0 {
		check = []string{x.word("editor.mod.overview.ok")}
	}
	_, enabled := m.firstEnabled()
	page := ui.ModOverviewPage{First: !enabled || m.packed}
	for _, s := range []struct {
		heading string
		lines   []string
		pages   []string
		fold    bool
	}{
		{x.word("editor.mod.overview.kinds"), counts, countPages, false},
		{x.word("editor.mod.overview.report"), check, checkPages, false},
		{x.word("editor.mod.overview.overrides"), append(append([]string(nil), m.report.Overrides...), m.fileOverrides...), nil, false},
		{x.word("editor.mod.overview.issues"), issues, nil, false},
		{x.word("editor.mod.overview.more"), append(append([]string(nil), m.checkLines...), m.report.Warnings...), nil, true},
		{fmt.Sprintf("%s (%d)", x.word("editor.mod.overview.files"), len(m.files)), m.files, nil, true},
	} {
		lines := s.lines
		if len(lines) == 0 {
			lines = []string{x.word("editor.mod.overview.none")}
		}
		page.Sections = append(page.Sections, ui.ModOverviewSection{Heading: s.heading, Lines: lines, Pages: s.pages, Fold: s.fold})
	}
	return page
}

// modPageTab is the tab word of each page an overview line opens.
var modPageTab = map[string]string{
	ui.ModMissionsPage:    "editor.mod.missions",
	ui.ModTerrainScope:    "editor.mod.terrain",
	ui.ModTextsScope:      "editor.mod.texts",
	ui.ModCharactersScope: "editor.mod.characters",
	ui.ModItemsScope:      "editor.mod.items",
	ui.ModSpellsScope:     "editor.mod.spells",
}

// warningPage is the page that edits the file a check warning names, and
// whether it names one; a warning that names none of the pages' files opens
// the Missions page and is shown as its own text.
func warningPage(warning string) (string, bool) {
	for _, p := range []struct {
		page  string
		files []string
	}{
		{ui.ModTerrainScope, []string{mod.TerrainFile}},
		{ui.ModCharactersScope, []string{mod.CharactersFile}},
		{ui.ModItemsScope, []string{mod.ItemsFile}},
		{ui.ModSpellsScope, []string{mod.SpellsFile}},
		{ui.ModTextsScope, textFiles()},
	} {
		for _, f := range p.files {
			if strings.Contains(warning, f) {
				return p.page, true
			}
		}
	}
	return ui.ModMissionsPage, false
}

// modPageCounts are the overview's count lines, one per page that holds any
// of the mod's things ("Items: 4"), and the page each line opens. Each
// count is what its page lists; one overview build reads each file once,
// with the text lookup its page reads names by.
func (s *mapInspectionEdits) modPageCounts() (lines, pages []string) {
	x := s.inspector
	if x.mod == nil {
		return nil, nil
	}
	var d *editorModDocument
	if s.workspace != nil {
		d = s.workspace.Project
	}
	count := func(page string, n int) {
		if n > 0 {
			lines, pages = append(lines, fmt.Sprintf(x.word("editor.mod.overview.count"), x.word(modPageTab[page]), n)), append(pages, page)
		}
	}
	held := func(text rowText, rels ...string) mod.Data {
		data, err := s.heldRows(d, text, rels...)
		if err != nil {
			return mod.Data{}
		}
		return data
	}
	missions := x.mod.document()
	if d != nil {
		missions = d
	}
	count(ui.ModMissionsPage, len(missions.Missions))
	count(ui.ModTerrainScope, len(held(nil, mod.TerrainFile).Terrain.Groups))
	keys := map[string]bool{}
	for _, set := range held(nil, textFiles()...).Strings {
		for k := range set {
			keys[k] = true
		}
	}
	count(ui.ModTextsScope, len(keys))
	count(ui.ModCharactersScope, len(held(s.characterNames, mod.CharactersFile).Characters.Rows))
	count(ui.ModItemsScope, itemThings(held(s.itemNames, mod.ItemsFile).Items))
	count(ui.ModSpellsScope, len(held(nil, mod.SpellsFile).Spells.Rows))
	return lines, pages
}

// ModOverview is W's overview page of the mod the editor works in.
func (s *mapInspectionEdits) ModOverview() ui.ModOverviewPage {
	return s.inspector.modOverview(s.modPageCounts())
}
