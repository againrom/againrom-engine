package ui

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// With one player the Players window still shows that player's switch and
// name field, with no relation matrix, and an edit through them writes one
// edit on player 0; with two players the grid keeps its 8 head rows.
func TestMapEditorPlayersOnePlayerKeepsItsRowControls(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, calls := newModesEditor(t, "full")
			e.SetWords(words.For(words.Language(lang)))
			e.Layout(1920, 1080)
			_ = e.Key("players")
			s, ok := e.listSpec()
			if heads := 0; !ok || s.Grid == nil {
				t.Fatal("two players show no grid")
			} else {
				for _, r := range s.Rows {
					if r.HeadOf > 0 {
						heads++
					}
				}
				if heads != 8 {
					t.Errorf("two players: %d head rows", heads)
				}
			}
			e.Select(len(e.doc.Records)-1, false)
			_ = e.Key("delete")
			s, _ = e.promptSpec()
			modesClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[0]))
			if m := e.relationMatrixOf(); len(m.players) != 1 {
				t.Fatalf("%d players left", len(m.players))
			}
			e.Select(slices.IndexFunc(e.doc.Records, func(r InspectionRecord) bool { return r.Kind == KindPlayer }), false)
			if s, ok = e.listSpec(); !ok || s.Grid == nil || len(s.Grid.ColHeads) != 0 {
				t.Fatal("one player: no row grid, or it still heads columns")
			}
			_, sw, _ := playerHead(t, e, 1, popupSwitch, false)
			from := len(calls.lines)
			modesClick(e, center(sw.Control))
			if got := calls.lines[from:]; len(got) != 1 || !strings.HasPrefix(got[0], "SetField Player 0 control") {
				t.Fatalf("the switch wrote %v", got)
			}
			_, name, _ := playerHead(t, e, 1, popupEdit, false)
			modesClick(e, center(name.Control))
			if m, _ := e.inputModel(); m != nil {
				m.SelectAll()
			}
			e.TextInput("Zed")
			from = len(calls.lines)
			_ = e.Key("enter")
			if got := playerCalls(calls, from, "SetTextField"); len(got) != 1 || len(calls.lines[from:]) != 1 {
				t.Fatalf("Enter wrote %v", calls.lines[from:])
			}
		})
	}
}

// Del asks about a player only while the Players window is shown.
func TestMapEditorPlayerDeleteNeedsThePlayersWindow(t *testing.T) {
	e, calls := newModesEditor(t, "full")
	_ = e.Key("players")
	e.Select(len(e.doc.Records)-1, false)
	_ = e.Key("players")
	if e.barWindow() == "players" {
		t.Fatal("P left the Players window open")
	}
	from := len(calls.lines)
	_ = e.Key("delete")
	_ = e.Key("enter")
	if e.prompt.open() || len(calls.lines) != from {
		t.Fatalf("Del outside the window asked %v, wrote %v", e.prompt.kind, calls.lines[from:])
	}
	_ = e.Key("players")
	_ = e.Key("delete")
	if e.prompt.kind != promptDelete {
		t.Fatal("Del in the Players window does not ask")
	}
	_ = e.Key("enter")
	if got := calls.lines[from:]; len(got) != 1 || got[0] != "Delete Player 1" {
		t.Fatalf("Enter wrote %v", got)
	}
}
