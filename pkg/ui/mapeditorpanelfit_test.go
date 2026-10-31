package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/words"
)

// panelWidths are the right panel's least, two middle and greatest widths:
// every window the panel hosts must fit each of them.
var panelWidths = []int{360, 420, 520, 800}

// setRightPanel restores both panels, the left at 260 and the right at
// width.
func setRightPanel(e *MapEditor, width int) {
	e.SetPanelProfile([]byte(fmt.Sprintf(`{"left":{"minimized":false,"width":260},"right":{"minimized":false,"width":%d}}`, width)), nil, nil)
}

// hostedFitProblems lists what the window the right panel hosts does not
// fit: a piece outside the window or the window outside the panel, a tab,
// button, label or line of text cut, a field narrower than popupMinField,
// two pieces over each other, and a drawn control a press does not reach.
func hostedFitProblems(e *MapEditor) []string {
	s, _, ok := e.objectSpec()
	if !ok || !e.panels.hosted {
		return []string{"no window in the right panel"}
	}
	title, body := e.font, e.bodyFont()
	l := layoutPopup(s, title, body)
	panel := e.sidePanelRect(panelRight)
	var out []string
	bad := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	if !l.Rect.In(panel) {
		bad("window %v outside the panel %v", l.Rect, panel)
	}
	type piece struct {
		what  string
		r     image.Rectangle
		press bool
	}
	var pieces []piece
	text := func(f interface{ Advance(string) int }, what, t string, r image.Rectangle, room int) {
		if f.Advance(t) > room {
			bad("%s %q cut: %d wide in %d", what, t, f.Advance(t), room)
		}
		pieces = append(pieces, piece{what: what + " " + t, r: image.Rect(r.Min.X, r.Min.Y, r.Min.X+min(r.Dx(), f.Advance(t)+2), r.Max.Y)})
	}
	for i, r := range l.Tabs {
		if body.Advance(s.Tabs[i])+8 > r.Dx() {
			bad("tab %q cut in %d", s.Tabs[i], r.Dx())
		}
		pieces = append(pieces, piece{what: "tab " + s.Tabs[i], r: r, press: true})
	}
	for i, r := range l.Buttons {
		if body.Advance(s.Buttons[i])+8 > r.Dx() {
			bad("button %q cut in %d", s.Buttons[i], r.Dx())
		}
		pieces = append(pieces, piece{what: "button " + s.Buttons[i], r: r, press: true})
	}
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		switch {
		case row.Kind == popupHeading:
			text(title, "heading", row.Label, rr.Label, rr.Rect.Dx()-4)
		case row.Kind == popupCard:
		case row.Label != "":
			text(body, "label", row.Label, rr.Label, rr.Label.Dx()-2)
		}
		if !rr.Control.Empty() {
			if rr.Control.Dx() < popupMinField {
				bad("field of %q %d wide", row.Label, rr.Control.Dx())
			}
			pieces = append(pieces, piece{what: "field of " + row.Label, r: rr.Control})
		}
		for k, r := range rr.Buttons {
			caption := rowButtons(s, row)[k]
			if row.Kind != popupCard && body.Advance(caption)+8 > r.Dx() {
				bad("row button %q cut in %d", caption, r.Dx())
			}
			pieces = append(pieces, piece{what: "row button " + caption, r: r, press: true})
		}
	}
	for i, r := range l.Notes {
		text(body, "note", s.Notes[i], r, r.Dx())
	}
	if !l.Card.Empty() {
		pieces = append(pieces, piece{what: "card", r: l.Card})
	}
	if !l.CardRest.Empty() {
		pieces = append(pieces, piece{what: "doll and face", r: l.CardRest})
	}
	for i, a := range pieces {
		if !a.r.In(l.Rect) {
			bad("%s %v outside the window %v", a.what, a.r, l.Rect)
		}
		for _, b := range pieces[i+1:] {
			if a.r.Overlaps(b.r) {
				bad("%s %v over %s %v", a.what, a.r, b.what, b.r)
			}
		}
		if a.press && e.chromeHit(center(a.r)) != popObject {
			bad("a press on %s %v reaches %d", a.what, a.r, e.chromeHit(center(a.r)))
		}
	}
	if p := image.Pt(panel.Min.X-20, (panel.Min.Y+panel.Max.Y)/2); e.chromeHit(p) == popObject || e.chromeHit(p) == popProps {
		bad("a press on the map beside the panel at %v reaches %d", p, e.chromeHit(p))
	}
	return out
}

// Each window the right panel hosts fits the panel at each of its widths,
// in English and Russian: every tab, button, label and line inside the
// window and whole, none over another, and a press on a drawn control
// reaches the window while a press beside the panel reaches the map. Fails
// when a window lays its button row, tabs or labels out at its floating
// width, which put Apply on the map at 360 and 420.
func TestMapEditorHostedWindowsFitEveryPanelWidth(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, width := range panelWidths {
			for _, kind := range []string{"Unit", "Structure", "Trigger", "Player", "group"} {
				e, _, _ := panelsEditor(t, 1600, 900, lang)
				setRightPanel(e, width)
				if got := e.panelsShown()[panelRight]; got != width {
					t.Fatalf("the right panel is %d wide, want %d", got, width)
				}
				if kind == "group" {
					e.chooseGroup(e.groups()[0], false)
				} else {
					e.Select(slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == kind }), false)
				}
				for _, p := range hostedFitProblems(e) {
					t.Errorf("%s %s at %d: %s", lang, kind, width, p)
				}
			}
			// The person window, each section it hosts.
			e, host := popupEditor(t)
			_ = e.Open("popups")
			e.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
			e.SetWords(words.For(lang))
			e.SetHostTextEncoder(func(s string) string {
				encoded, _ := textinput.EncodeText(s, e.font.Selector, textinput.Substitute)
				return encoded
			})
			e.Layout(1600, 900)
			setRightPanel(e, width)
			e.Select(0, false)
			for section := range 4 {
				e.obj.section = section
				if _, _, _ = e.objectSpec(); !e.panels.hosted {
					continue // the bag keeps its floating window
				}
				for _, p := range hostedFitProblems(e) {
					t.Errorf("%s person section %d at %d: %s", lang, section, width, p)
				}
			}
		}
	}
}

// At the widths the review measured, with the left panel at 260 in a
// 1280x800 window, a press on the trigger window's and the person window's
// first button acts on the window and keeps the selection: it once fell to
// the map and cleared it.
func TestMapEditorPanelButtonPressActsOnTheWindow(t *testing.T) {
	for _, width := range []int{360, 420, 520} {
		e, _, _ := panelsEditor(t, 1280, 800, "ru")
		setRightPanel(e, width)
		trigger := slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == "Trigger" })
		e.Select(trigger, false)
		pressFirstButton(t, e, fmt.Sprint("trigger at ", width), trigger)

		f, host := popupEditor(t)
		_ = f.Open("popups")
		f.doc.Edits = &personSettingsProbe{&settingsPreviewProbe{popupHost: host}}
		f.SetWords(words.For("ru"))
		setRightPanel(f, width)
		f.Select(0, false)
		pressFirstButton(t, f, fmt.Sprint("person at ", width), 0)
	}
}

func pressFirstButton(t *testing.T, e *MapEditor, name string, selected int) {
	t.Helper()
	s, _, ok := e.objectSpec()
	if !ok || !e.panels.hosted {
		t.Fatalf("%s: the window is not in the right panel", name)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	if len(l.Buttons) == 0 || !l.Buttons[0].In(e.sidePanelRect(panelRight)) {
		t.Fatalf("%s: buttons %v outside the panel %v", name, l.Buttons, e.sidePanelRect(panelRight))
	}
	chromeClick(e, center(l.Buttons[0]))
	if e.selected != selected {
		t.Fatalf("%s: a press on %q left selection %d, want %d", name, s.Buttons[0], e.selected, selected)
	}
}

// Every group, a group numbered 10 or more and another player's group of
// the same number among them, opens from the tree and deletes from its
// window at each window width from the least the editor admits: the left
// panel narrows to fit beside the map and the right panel's strip instead
// of folding to its own strip. F6 twice gives the panel back narrowed.
func TestMapEditorEveryGroupOpensAndDeletesAtNarrowWindows(t *testing.T) {
	for _, w := range []int{640, 700, 760, 800} {
		e, p, _ := panelsEditor(t, w, 480, "")
		p.units[2].group, p.units[4].group = 30, 30
		if err := e.Open("map0"); err != nil {
			t.Fatal(err)
		}
		e.Layout(w, 480)
		_ = e.Key("objectspanel")
		_ = e.Key("objectspanel")
		if !e.panelOpen(panelLeft) || e.panelsShown()[panelLeft] < sidePanels[panelLeft].least {
			t.Fatalf("width %d: the left panel shows %v", w, e.panelsShown())
		}
		for _, g := range []editorGroup{{player: 1, group: 30}, {player: 2, group: 30}, {player: 1, group: 1}, {player: 2, group: 2}} {
			chromeClick(e, barPoint(t, e, "groups"))
			row := slices.IndexFunc(e.treeRowsNow(), func(r treeRow) bool {
				return r.group.set && r.group.player == g.player && r.group.group == g.group
			})
			if row < 0 {
				t.Fatalf("width %d: the tree has no row for group %d of player %d: %q", w, g.group, g.player, e.TreeLines())
			}
			clickTreeRow(t, e, row)
			if !e.groupSel.set || e.groupSel.player != g.player || e.groupSel.group != g.group {
				t.Fatalf("width %d: the tree row of group %d of player %d chose %+v", w, g.group, g.player, e.groupSel)
			}
			if g.group != 30 {
				continue
			}
			r, ok := e.GroupDeleteRect()
			if !ok || !r.In(image.Rectangle{Max: image.Pt(w, 480)}) || e.chromeHit(center(r)) != popObject {
				t.Fatalf("width %d: Delete group at %v reaches %d", w, r, e.chromeHit(center(r)))
			}
			before := len(p.calls.lines)
			chromeClick(e, center(r))
			unit := 2
			if g.player == 2 {
				unit = 4
			}
			if got := strings.Join(p.calls.lines[before:], "|"); got != fmt.Sprintf("SetUnitsGroup [{Unit %d}] 0", unit) {
				t.Fatalf("width %d: Delete group of group 30 of player %d called %q", w, g.player, got)
			}
		}
	}
}
