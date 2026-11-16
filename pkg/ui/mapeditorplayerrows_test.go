package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// playerHead is the shown head control of matrix row k (from 1) of kind,
// a cross for popupText with buttons.
func playerHead(t *testing.T, e *MapEditor, k int, kind popupRowKind, cross bool) (popupRow, popupRowRect, popupLayout) {
	t.Helper()
	s, ok := e.listSpec()
	if !ok || s.Grid == nil {
		t.Fatal("the Players window shows no matrix")
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	for _, rr := range l.Rows {
		row := s.Rows[rr.Index]
		if row.HeadOf == k && row.Kind == kind && (len(row.Buttons) > 0) == cross {
			return row, rr, l
		}
	}
	t.Fatalf("row %d has no %v control (cross %v)", k, kind, cross)
	return popupRow{}, popupRowRect{}, l
}

func playerCalls(c *modesCalls, from int, prefix string) []string {
	var out []string
	for _, line := range c.lines[from:] {
		if strings.HasPrefix(line, prefix) {
			out = append(out, line)
		}
	}
	return out
}

// Each matrix row of the Players window holds the player's own controls
// left of its cells, whole at 1920x1080 in both languages: the
// human/computer switch, the number, the name field and the cross; the
// first player's cross is off.
func TestMapEditorPlayersRowsHoldTheirControls(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			e.SetWords(words.For(words.Language(lang)))
			e.Layout(1920, 1080)
			_ = e.Key("players")
			for k := 1; k <= 2; k++ {
				var boxes []image.Rectangle
				for _, c := range []struct {
					kind  popupRowKind
					cross bool
				}{{popupSwitch, false}, {popupText, false}, {popupEdit, false}, {popupText, true}} {
					row, rr, l := playerHead(t, e, k, c.kind, c.cross)
					if !rr.Rect.In(l.Rect) || rr.Rect.Max.X > l.Grid.Min.X || rr.Rect.Min.Y < l.Grid.Min.Y {
						t.Errorf("row %d control %v at %v stands off its head box (window %v, grid %v)", k, c.kind, rr.Rect, l.Rect, l.Grid)
					}
					for _, b := range boxes {
						if b.Overlaps(rr.Rect) {
							t.Errorf("row %d controls overlap: %v and %v", k, b, rr.Rect)
						}
					}
					boxes = append(boxes, rr.Rect)
					if full, cut := rowFull(e.font, e.bodyFont(), row, rr); cut {
						t.Errorf("row %d shows %q cut", k, full)
					}
					switch {
					case c.kind == popupSwitch && row.Value != e.word("editor.add.control.computer"):
						t.Errorf("row %d switch reads %q", k, row.Value)
					case c.kind == popupText && !c.cross && row.Value != fmt.Sprint(k):
						t.Errorf("row %d number reads %q", k, row.Value)
					case c.kind == popupEdit && row.Value != []string{"Self", "Enemy"}[k-1]:
						t.Errorf("row %d name reads %q", k, row.Value)
					case c.cross && (len(rr.Off) == 0 || rr.Off[0] != (k == 1)):
						t.Errorf("row %d cross off %v", k, rr.Off)
					}
				}
			}
			e.Select(len(e.doc.Records)-1, false)
			if _, _, ok := e.objectSpec(); ok {
				t.Error("a player still opens a record window")
			}
		})
	}
}

// The switch sets the other control and the name field writes the typed
// name on Enter, each as one edit.
func TestMapEditorPlayersRowSwitchAndNameAreOneEditEach(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	_, sw, _ := playerHead(t, e, 2, popupSwitch, false)
	from := len(calls.lines)
	modesClick(e, center(sw.Control))
	if got := calls.lines[from:]; len(got) != 1 || got[0] != "SetField Player 1 control 0" {
		t.Fatalf("the switch wrote %v", got)
	}
	_, name, _ := playerHead(t, e, 2, popupEdit, false)
	modesClick(e, center(name.Control))
	if focus, _ := e.playerTyping(); focus != 1 {
		t.Fatalf("the name field of row 2 does not take the keyboard: %d", focus)
	}
	if m, _ := e.inputModel(); m != nil {
		m.SelectAll()
	}
	e.TextInput("Rival")
	if row, _, _ := playerHead(t, e, 2, popupEdit, false); !row.Focus || row.Edit == nil {
		t.Error("the typed name does not show in its row")
	}
	from = len(calls.lines)
	_ = e.Key("enter")
	if got := calls.lines[from:]; len(got) != 1 || !strings.Contains(got[0], "Rival") {
		t.Fatalf("Enter wrote %v", got)
	}
	if row, _, _ := playerHead(t, e, 2, popupEdit, false); row.Value != "Rival" {
		t.Errorf("the name reads %q after Enter", row.Value)
	}
}

// The cross asks through the editor's question box; its button deletes the
// player as one edit and Cancel deletes nothing. Del on a selected player
// asks the same question.
func TestMapEditorPlayersCrossAsksThenDeletes(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			e.SetWords(words.For(words.Language(lang)))
			e.Layout(1920, 1080)
			_ = e.Key("players")
			_, cross, _ := playerHead(t, e, 2, popupText, true)
			modesClick(e, center(cross.Button))
			if e.prompt.kind != promptDelete {
				t.Fatalf("the cross opened prompt %v", e.prompt.kind)
			}
			s, _ := e.promptSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			if s.Title != e.word("editor.players.deletetitle") || !slices.Contains(s.Buttons, e.word("editor.players.deletego")) || !strings.Contains(strings.Join(s.Notes, " "), "Enemy") {
				t.Errorf("the question reads %q %v %v", s.Title, s.Buttons, s.Notes)
			}
			if !l.Rect.In(e.popupBounds()) {
				t.Errorf("the question %v leaves its bounds", l.Rect)
			}
			for k, b := range l.Buttons {
				if e.bodyFont().Advance(s.Buttons[k]) > b.Dx()-4 {
					t.Errorf("button %q is cut", s.Buttons[k])
				}
			}
			from := len(calls.lines)
			modesClick(e, center(l.Buttons[1]))
			if e.prompt.open() || len(calls.lines) != from {
				t.Fatalf("Cancel left %v, wrote %v", e.prompt.kind, calls.lines[from:])
			}
			_ = e.Key("delete")
			if e.prompt.kind == promptDelete || len(calls.lines) != from {
				t.Fatal("Del on the first player asks a question the model refuses")
			}
			e.message = ""
			e.Select(len(e.doc.Records)-1, false)
			_ = e.Key("delete")
			if e.prompt.kind != promptDelete {
				t.Fatal("Del on a player does not ask")
			}
			s, _ = e.promptSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
			if got := calls.lines[from:]; len(got) != 1 || got[0] != "Delete Player 1" {
				t.Fatalf("the question's button wrote %v", got)
			}
			if m := e.relationMatrixOf(); len(m.players) != 1 {
				t.Errorf("%d players left", len(m.players))
			}
		})
	}
}

// A player chosen outside the Players window opens a record window that only
// reads: its fields are text rows, Apply is off and its users stay listed.
// E there opens the Players window and types the name in the player's row.
func TestMapEditorPlayerWindowElsewhereOnlyReads(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	at := slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == KindPlayer && r.Index == 1 })
	e.Select(at, false)
	s, _, ok := e.objectSpec()
	if !ok || e.barWindow() == "players" {
		t.Fatalf("no player window outside the Players window (window %q)", e.barWindow())
	}
	used := false
	for _, r := range s.Rows {
		if r.Kind != popupText && r.Kind != popupHeading && r.Kind != popupLink {
			t.Errorf("the player window edits %q (kind %d)", r.Label, r.Kind)
		}
		used = used || strings.Contains(r.Label, "Used by")
	}
	if !used || len(s.Disabled) == 0 || !s.Disabled[0] {
		t.Errorf("used-by %v, Apply disabled %v", used, s.Disabled)
	}
	from := len(calls.lines)
	_ = e.Key("field")
	if focus, _ := e.playerTyping(); e.barWindow() != "players" || focus != 1 {
		t.Fatalf("E shows %q and types row %d", e.barWindow(), focus)
	}
	if len(calls.lines) != from {
		t.Errorf("E wrote %v", calls.lines[from:])
	}
}

// Add in the Players window adds a player at once, as one edit, with no Add
// window; the window stays and selects the new player.
func TestMapEditorPlayersAddOpensNoAddWindow(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	button, ok := e.FindControl(ControlQuery{Role: ControlListButtonKey, Key: "add"})
	if !ok {
		t.Fatal("the Players window has no Add button")
	}
	from := len(calls.lines)
	modesClick(e, center(button))
	if addKindNow(e) != "" {
		t.Fatalf("Add opened the %s Add window", addKindNow(e))
	}
	if got := playerCalls(calls, from, "AddRecord Player"); len(got) != 1 || len(calls.lines)-from != 1 {
		t.Fatalf("Add wrote %v", calls.lines[from:])
	}
	if e.barWindow() != "players" || len(e.relationMatrixOf().players) != 3 {
		t.Errorf("window %q, %d players", e.barWindow(), len(e.relationMatrixOf().players))
	}
	if kind, index := e.selectedRecord(); kind != KindPlayer || index != 2 {
		t.Errorf("selected %s %d", kind, index)
	}
	_ = e.Key("player")
	if addKindNow(e) != "" || len(playerCalls(calls, from, "AddRecord Player")) != 2 {
		t.Errorf("A opened %q, wrote %v", addKindNow(e), calls.lines[from:])
	}
}
