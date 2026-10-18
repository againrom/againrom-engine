package ui

import (
	"fmt"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// openGroup opens the groups list and chooses the first group row.
func openGroup(t *testing.T, e *MapEditor) {
	t.Helper()
	chromeClick(e, barPoint(t, e, "groups"))
	e.listSpec()
	chromeClick(e, listRowPoint(t, e, 0)) // player 1, group 1: units 0 and 2
}

// groupLinks are the records the group popup's member rows open.
func groupLinks(t *testing.T, e *MapEditor) []int {
	t.Helper()
	_, links, ok := e.groupSpec()
	if !ok {
		t.Fatal("no group popup")
	}
	var out []int
	for _, l := range links[2:] {
		if l < 0 && e.doc.Records[-l-1].Kind == "Unit" {
			out = append(out, -l-1)
		}
	}
	return out
}

func TestEditorLargeGroupKeepsBackInsidePopup(t *testing.T) {
	e, _ := popupEditor(t)
	_ = e.Open("popups")
	for _, height := range []int{800, 480} {
		e.Layout(1280, height)
		g := editorGroup{player: 1, group: 1, set: true}
		for i := 0; i < 32; i++ {
			g.members = append(g.members, 0)
		}
		e.chooseGroup(g)
		e.objectBack = []objectPlace{{kind: "Unit", index: 0}}
		_, l := objectLayout(t, e)
		if len(l.Buttons) != 1 || !l.Buttons[0].In(l.Rect) || !l.Buttons[0].In(e.popupBounds()) {
			t.Fatalf("height %d: Back %v exceeds popup %v or bounds %v", height, l.Buttons, l.Rect, e.popupBounds())
		}
	}
}

func TestEditorGroupPopupFollowsUndoRedoAndEdits(t *testing.T) {
	e, h := popupEditor(t)
	_ = e.Open("popups")
	openGroup(t, e)
	if got := fmt.Sprint(e.GroupMembers()); got != "[0 2]" {
		t.Fatalf("group members %s", got)
	}
	// An edit moves unit 0 out of the group.
	if err := e.adoptEdit(h.regroup(0, 2), nil); err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprint(e.GroupMembers(), groupLinks(t, e)); got != "[2] [2]" {
		t.Fatalf("after the move: members and links %s", got)
	}
	if n := len(e.groupMarkerLines()); n != 4 {
		t.Fatalf("the group boxes %d lines, want one box", n)
	}
	// Undo brings unit 0 back.
	chromeClick(e, barPoint(t, e, "undo"))
	if got := fmt.Sprint(e.GroupMembers(), groupLinks(t, e)); got != "[0 2] [0 2]" {
		t.Fatalf("after Undo: members and links %s", got)
	}
	if n := len(e.groupMarkerLines()); n != 8 {
		t.Fatalf("the group boxes %d lines after Undo, want two boxes", n)
	}
	// Redo moves it out again.
	chromeClick(e, barPoint(t, e, "redo"))
	if got := fmt.Sprint(e.GroupMembers(), groupLinks(t, e)); got != "[2] [2]" {
		t.Fatalf("after Redo: members and links %s", got)
	}
	chromeClick(e, center(objectRowRect(t, e, 2).Rect))
	if e.selected != 2 {
		t.Fatalf("the member row opened record %d", e.selected)
	}
	// A group left with no member closes its popup.
	e.Select(0, false)
	openGroup(t, e)
	if err := e.adoptEdit(h.regroup(2, 2), nil); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := e.groupSpec(); ok || len(e.GroupMembers()) != 0 {
		t.Fatal("the group popup stayed open with no member")
	}
	// The Groups list is rebuilt from the document.
	if e.model.Len() != 1 || e.listKind() != "records" {
		t.Logf("list kind %q rows %d", e.listKind(), e.model.Len())
	}
}

// focusedUnit is an editor with unit 2 applied once (so there is an Undo
// step), its field 0 focused and a value typed that is not applied.
func focusedUnit(t *testing.T, apply bool) (*MapEditor, *popupHost, string) {
	t.Helper()
	e, h := popupEditor(t)
	_ = e.Open("popups")
	e.Select(2, true)
	objectLayout(t, e)
	if apply {
		chromeClick(e, center(fieldRowRect(t, e, 0).Control))
		_ = e.Key("backspace")
		_ = e.Key("backspace")
		e.TextInput("55")
		_ = e.Key("enter")
		if h.units[2]["a"] != "55" {
			t.Fatal("the setup Apply did not write")
		}
	}
	saved := string(e.doc.SourceBytes)
	chromeClick(e, center(fieldRowRect(t, e, 0).Control))
	_ = e.Key("backspace")
	_ = e.Key("backspace")
	e.TextInput("77")
	if !e.objectTyping() {
		t.Fatal("the field is not focused")
	}
	return e, h, saved
}

func assertTypedValueDropped(t *testing.T, e *MapEditor, h *popupHost, calls int) {
	t.Helper()
	if e.objectTyping() {
		t.Fatal("field typing continued after the click")
	}
	if len(h.calls) != calls || h.units[2]["a"] == "77" {
		t.Fatalf("the typed value was applied: calls %v unit %v", h.calls, h.units[2])
	}
}

func TestEditorBarButtonsActWhileAFieldIsFocused(t *testing.T) {
	t.Run("undo", func(t *testing.T) {
		e, h, saved := focusedUnit(t, true)
		chromeClick(e, barPoint(t, e, "undo"))
		if string(e.doc.SourceBytes) == saved || h.cursor != 1 {
			t.Fatalf("Undo did not act: cursor %d", h.cursor)
		}
		assertTypedValueDropped(t, e, h, 1)
	})
	t.Run("redo", func(t *testing.T) {
		e, h, saved := focusedUnit(t, true)
		chromeClick(e, barPoint(t, e, "undo"))
		chromeClick(e, center(fieldRowRect(t, e, 0).Control))
		e.TextInput("7")
		if !e.objectTyping() {
			t.Fatal("the field is not focused")
		}
		chromeClick(e, barPoint(t, e, "redo"))
		if string(e.doc.SourceBytes) != saved || h.cursor != 2 {
			t.Fatalf("Redo did not act: cursor %d", h.cursor)
		}
		assertTypedValueDropped(t, e, h, 1)
	})
	t.Run("save", func(t *testing.T) {
		e, h, _ := focusedUnit(t, true)
		h.target = "mod"
		chromeClick(e, barPoint(t, e, "save"))
		if h.saves != 1 {
			t.Fatalf("Save did not act: %d saves", h.saves)
		}
		assertTypedValueDropped(t, e, h, 1)
	})
}

func TestEditorMapMenuRowsActWhileAFieldIsFocused(t *testing.T) {
	rows := []struct {
		key  string
		dirt bool
		done func(*testing.T, *MapEditor, *popupHost)
	}{
		{"new", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if e.prompt.kind != promptRectangle || !e.catalog {
				t.Fatalf("New map opened prompt %v catalog %v", e.prompt.kind, e.catalog)
			}
		}},
		{"catalog", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if !e.catalog {
				t.Fatal("Open... did not show the picker")
			}
		}},
		{"open", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if e.prompt.kind != promptOpen {
				t.Fatalf("Open file... opened prompt %v", e.prompt.kind)
			}
		}},
		{"saveas", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if h.saves != 1 {
				t.Fatalf("Save wrote %d times", h.saves)
			}
		}},
		{"saveto", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if e.prompt.kind != promptSave {
				t.Fatalf("Save as... opened prompt %v", e.prompt.kind)
			}
		}},
		{"testplay", true, func(t *testing.T, e *MapEditor, h *popupHost) {
			if h.plays != 1 || e.message != "played" {
				t.Fatalf("Test play ran %d times, message %q", h.plays, e.message)
			}
		}},
		{"close", false, func(t *testing.T, e *MapEditor, h *popupHost) {
			if !e.ExitAsked() || e.Update() != ebiten.Termination {
				t.Fatal("Exit did not ask to close the editor")
			}
		}},
	}
	for _, row := range rows {
		t.Run(row.key, func(t *testing.T) {
			e, h, _ := focusedUnit(t, row.dirt)
			h.target = "mod"
			calls := len(h.calls)
			e.menu, e.menuModel = "map", nil
			n := -1
			for i, it := range e.menuItems("map") {
				if it.key == row.key {
					n = i
				}
			}
			if n < 0 {
				t.Fatalf("no menu row %q", row.key)
			}
			chromeClick(e, menuPoint(t, e, n))
			assertTypedValueDropped(t, e, h, calls)
			row.done(t, e, h)
		})
	}
}
