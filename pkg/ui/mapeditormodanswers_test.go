package ui

import (
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// answersEditor is the mod window over a host that gives every page, at
// 1920x1080 in lang, with the calls the host records.
func answersEditor(t *testing.T, lang string) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := newModesEditor(t, "workspace")
	e.font = wordsFont(lang)
	e.SetWords(words.For(lang))
	e.SetHostTextEncoder(func(s string) string {
		encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
		return encoded
	})
	e.Layout(1920, 1080)
	host := e.doc.Edits.(*modesWorkspace)
	host.overview = modesOverview
	host.graph = CampaignGraphPage{Unread: true}
	e.doc.Edits = everyPageHost{host}
	return e, calls
}

// everyPageHost gives each page of W its fields; the Items page holds two
// things, each with its name field and its Remove action.
type everyPageHost struct{ *modesWorkspace }

func (h everyPageHost) WorkspaceFields(scope string) []InspectionField {
	thing := func(key, group string) []InspectionField {
		return []InspectionField{
			{Key: key + ".name", Label: "Name", Text: true, TextValue: group, Group: group},
			{Key: key + ".remove", Label: "Remove from the mod", Action: true, Group: group},
		}
	}
	switch scope {
	case ModItemsScope:
		out := []InspectionField{{Key: "new.armour", Label: "New armour", Choices: []InspectionChoice{{Value: 0, Label: "-"}, {Value: 1, Label: "Coat"}}}}
		return append(append(out, thing("item.coat", "Armour Coat")...), thing("weapon.pike", "Weapon Pike")...)
	case ModTextsScope:
		// One text in two languages, Delete on each language's row.
		return []InspectionField{
			{Key: "en:greet", Label: "English", Text: true, TextValue: "Hello"},
			{Key: "drop.en:greet", Label: "Delete", Action: true, Of: "en:greet"},
			{Key: "ru:greet", Label: "Russian", Text: true, TextValue: "Privet"},
			{Key: "drop.ru:greet", Label: "Delete", Action: true, Of: "ru:greet"},
		}
	case ModTerrainScope, ModCharactersScope, ModSpellsScope:
		return []InspectionField{{Key: scope + ".name", Label: "Name", Text: true, TextValue: scope}}
	}
	return h.modesWorkspace.WorkspaceFields(scope)
}

// turnTo shows W's page n, failing the test when W has no such page.
func turnTo(t *testing.T, e *MapEditor, n int) {
	t.Helper()
	k := slices.Index(e.modPages(), n)
	if k < 0 {
		t.Fatalf("W has no page %d among %v", n, e.modPages())
	}
	e.turnModPage(k)
}

// Answer 1: every page of W is as wide as its tab row needs to keep the
// eight Russian captions on one line, the same width on each page.
func TestModWindowKeepsItsTabsOnOneRowAtOneWidth(t *testing.T) {
	e, _ := answersEditor(t, "ru")
	_ = e.Key("package")
	pages := e.modPages()
	if len(pages) != 8 {
		t.Fatalf("W shows %d pages, want 8: %v", len(pages), pages)
	}
	width := 0
	for k := range pages {
		e.turnModPage(k)
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		for n, tab := range l.Tabs {
			if tab.Min.Y != l.Tabs[0].Min.Y {
				t.Errorf("page %d: tab %d %v stands on another line than tab 0 %v", k, n, tab, l.Tabs[0])
			}
		}
		if k == 0 {
			width = l.Rect.Dx()
		} else if l.Rect.Dx() != width {
			t.Errorf("page %d is %d px wide, page 0 %d", k, l.Rect.Dx(), width)
		}
	}
}

// Answer 2: a page of things shows its adders and the list of its things
// left, and the chosen thing right with its actions on its name line; a
// press on a list row chooses that thing.
func TestModItemsPageListsItsThingsAndShowsTheChosenOne(t *testing.T) {
	e, _ := answersEditor(t, "en")
	_ = e.Key("package")
	turnTo(t, e, modPageItems)
	head := func() popupRow {
		t.Helper()
		s, _, _ := e.modSpec()
		for _, r := range s.Rows {
			if r.Column == 1 && len(r.Buttons) > 0 {
				return r
			}
		}
		t.Fatalf("the right column has no name line with actions: %+v", s.Rows)
		return popupRow{}
	}
	if h := head(); h.Label != "Armour Coat" || !slices.Equal(h.Buttons, []string{"Remove from the mod"}) {
		t.Fatalf("the chosen thing's name line %q %q", h.Label, h.Buttons)
	}
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	pressed := false
	for _, r := range l.Rows {
		if row := s.Rows[r.Index]; row.Column == 0 && row.Label == "Weapon Pike" && !r.Label.Empty() {
			modesClick(e, boxMidpoint(r.Label))
			pressed = true
		}
	}
	if !pressed {
		t.Fatalf("the left column lists no row Weapon Pike: %+v", s.Rows)
	}
	if h := head(); h.Label != "Weapon Pike" {
		t.Fatalf("after a press on its list row the chosen thing is %q", h.Label)
	}
}

// Answer 4: the campaign chain line stands in the Mission block, whole in
// its tip, with Move up and Move down on it; Move down runs its key.
func TestModMissionBlockCarriesTheChainLine(t *testing.T) {
	e, calls := answersEditor(t, "en")
	host := e.doc.Edits.(everyPageHost)
	line := "Campaign 20 -> Crossing 25 -> Watch 26"
	host.fields = append(host.fields, InspectionField{Key: "chain", ReadOnly: true, Value: 1, Column: 1, Lines: []InspectionLine{{Text: line}}})
	_ = e.Key("package")
	turnTo(t, e, modPageFields)
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, r := range l.Rows {
		row := s.Rows[r.Index]
		if row.Label != line {
			continue
		}
		if row.Tip != line || !slices.Equal(row.Buttons, []string{e.word("editor.mod.up"), e.word("editor.mod.down")}) || len(r.Buttons) != 2 {
			t.Fatalf("the chain line's tip %q, buttons %q", row.Tip, row.Buttons)
		}
		modesClick(e, boxMidpoint(r.Buttons[1]))
		if got := strings.Join(calls.lines, "\n"); !strings.Contains(got, "package down") {
			t.Fatalf("Move down on the chain line ran no move: %s", got)
		}
		for _, note := range s.Notes {
			if strings.Contains(note, "Crossing 25") {
				t.Fatalf("the notes hold the chain line: %q", s.Notes)
			}
		}
		return
	}
	t.Fatalf("no row holds the chain line: %+v", s.Rows)
}

// Answer 10: a long mission name on the Missions page is one line, cut to
// its width, the whole name in its tip.
func TestModMissionRowShowsALongNameWholeInItsTip(t *testing.T) {
	e, _ := answersEditor(t, "en")
	host := e.doc.Edits.(everyPageHost)
	long := strings.Repeat("The long road over the northern pass ", 8) + "(25)"
	for i := range host.fields {
		if host.fields[i].Key == "source" {
			host.fields[i].Choices[0].Label = long
		}
	}
	_ = e.Key("package")
	turnTo(t, e, modPageFields)
	s, _, _ := e.modSpec()
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, r := range l.Rows {
		row := s.Rows[r.Index]
		if row.Kind != popupText || !strings.HasPrefix(long, strings.TrimSuffix(row.Label, "...")) || row.Label == "" {
			continue
		}
		if row.Tip != long {
			t.Fatalf("the mission row %q has the tip %q", row.Label, row.Tip)
		}
		if r.Rect.Dy() > fontHeight(e.bodyFont(), 15)+popupRowExtra {
			t.Fatalf("the mission row takes %d px, more than one line", r.Rect.Dy())
		}
		if len(r.Buttons) != 1 || r.Label.Max.X > r.Buttons[0].Min.X {
			t.Fatalf("the mission row's label %v runs under its Open button %v", r.Label, r.Buttons)
		}
		return
	}
	t.Fatalf("no mission row shows the long name: %+v", s.Rows)
}

// Answer 3: Remove stands beside the mission and Add map on the map rows of
// the Missions page; a press on each reaches the host; the window's button
// row is its own four.
func TestModMissionsPageCarriesRemoveAndAddMap(t *testing.T) {
	e, p := modMapActionsEditor(t, "en", image.Pt(1920, 1080))
	turnTo(t, e, modPageFields)
	press := func(want string, on func(r popupRowRect) (image.Rectangle, bool)) {
		t.Helper()
		s, _, _ := e.modSpec()
		l := layoutPopup(s, e.font, e.bodyFont())
		if len(s.Buttons) != len(modButtons) {
			t.Fatalf("the window's buttons %q", s.Buttons)
		}
		for _, r := range l.Rows {
			if b, ok := on(r); ok {
				count := len(p.calls.lines)
				modesClick(e, boxMidpoint(b))
				if got := strings.Join(p.calls.lines[count:], "\n"); !strings.Contains(got, want) {
					t.Fatalf("the press ran %q, want %q", got, want)
				}
				return
			}
		}
		t.Fatalf("no row carries the button for %q: %+v", want, s.Rows)
	}
	press("package remove", func(r popupRowRect) (image.Rectangle, bool) {
		i, ok := e.obj.rowField[r.Index]
		if !ok || e.obj.fields[i].Key != "source" || len(r.Buttons) != 1 {
			return image.Rectangle{}, false
		}
		s, _, _ := e.modSpec()
		return r.Buttons[0], s.Rows[r.Index].Buttons[0] == e.word("editor.mod.remove")
	})
	press("package addmap", func(r popupRowRect) (image.Rectangle, bool) {
		s, _, _ := e.modSpec()
		row := s.Rows[r.Index]
		return firstButton(r), row.Label == e.word("editor.mod.addmap.label") && slices.Equal(row.Buttons, []string{e.word("editor.mod.addmap")})
	})
}

// firstButton is r's first row button, empty for none.
func firstButton(r popupRowRect) image.Rectangle {
	if len(r.Buttons) == 0 {
		return image.Rectangle{}
	}
	return r.Buttons[0]
}
