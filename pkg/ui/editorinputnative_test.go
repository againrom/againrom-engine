package ui

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

func updateEditorKeys(t *testing.T, e *MapEditor, keys ...ebiten.Key) {
	t.Helper()
	down := map[ebiten.Key]bool{}
	for _, k := range keys {
		down[k] = true
	}
	pressed := func(k ebiten.Key) bool { return down[k] }
	a := e.mapArea()
	e.nativeInput = func() editorNativeInput {
		return editorNativeInput{pressed: pressed, justPressed: pressed, keysDown: len(keys) > 0,
			canvas: Input{CursorX: a.Min.X + 80, CursorY: a.Min.Y + 80,
				PanLeft: down[ebiten.KeyArrowLeft], PanRight: down[ebiten.KeyArrowRight],
				PanUp: down[ebiten.KeyArrowUp], PanDown: down[ebiten.KeyArrowDown],
				Ctrl: down[ebiten.KeyControl], Shift: down[ebiten.KeyShift]}}
	}
	if err := e.Update(); err != nil {
		t.Fatal(err)
	}
}

func TestEditorNativeAddCellArrowsPan(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyArrowLeft, ebiten.KeyArrowRight, ebiten.KeyArrowUp, ebiten.KeyArrowDown} {
		t.Run(key.String(), func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			_ = e.Key("sack")
			if !e.addPicking() || e.blocksCanvas() {
				t.Fatal("not waiting for a cell")
			}
			e.prompt.path = "25,20"
			m, _ := e.inputModel()
			caret, anchor := m.Caret, m.Anchor
			c := e.doc.Viewer.Camera()
			c.Zoom, c.X, c.Y = 2, 400, 400
			c.Clamp()
			x, y := c.X, c.Y
			updateEditorKeys(t, e, key)
			if c.X == x && c.Y == y {
				t.Error("Add cell arrow did not pan the map")
			}
			if e.prompt.path != "25,20" || m.Caret != caret || m.Anchor != anchor {
				t.Errorf("map arrow edited coordinates: %q caret %d anchor %d", e.prompt.path, m.Caret, m.Anchor)
			}
		})
	}
	e, _ := newModesEditor(t, "full")
	_ = e.Key("sack")
	e.prompt.path = "25,20"
	for _, key := range []ebiten.Key{ebiten.KeyHome, ebiten.KeyEnd, ebiten.KeyBackspace} {
		updateEditorKeys(t, e, key)
	}
	if e.prompt.path != "25,2" || e.input.model.Caret != 4 {
		t.Fatal("coordinate editing keys lost", e.prompt.path, e.input.model.Caret)
	}
}

func TestEditorNativePopupArrowsOwnCaret(t *testing.T) {
	e, _ := popupEditor(t)
	if err := e.Open("popups"); err != nil {
		t.Fatal(err)
	}
	e.Select(0, false)
	e.loadObject()
	e.obj.focus = 2
	m, _ := e.inputModel()
	c := e.doc.Viewer.Camera()
	c.Zoom, c.X, c.Y = 2, 400, 400
	c.Clamp()
	x, y := c.X, c.Y
	updateEditorKeys(t, e, ebiten.KeyArrowLeft)
	if m.Caret != 1 || c.X != x || c.Y != y || e.obj.buf[2] != "u0" {
		t.Fatalf("focused arrow: caret %d camera %g,%g text %q", m.Caret, c.X, c.Y, e.obj.buf[2])
	}
	updateEditorKeys(t, e)
	updateEditorKeys(t, e, ebiten.KeyArrowRight)
	if m.Caret != 2 || c.X != x || c.Y != y {
		t.Fatal("Right did not return the caret without map pan", m.Caret, c.X, c.Y)
	}
	updateEditorKeys(t, e, ebiten.KeyEscape)
	updateEditorKeys(t, e, ebiten.KeyArrowLeft)
	if e.objectTyping() || c.X >= x {
		t.Fatal("map pan did not resume after field lost focus", c.X, x)
	}
}

func TestEditorNativeModifiedPromptNavigation(t *testing.T) {
	for _, key := range []ebiten.Key{ebiten.KeyArrowUp, ebiten.KeyArrowDown} {
		t.Run("single-line/"+key.String(), func(t *testing.T) {
			e, _ := popupEditor(t)
			if err := e.Open("popups"); err != nil {
				t.Fatal(err)
			}
			e.Select(0, false)
			e.openPrompt(editorPrompt{kind: promptField, field: 1, path: "2"})
			want := "1"
			if key == ebiten.KeyArrowDown {
				want = "3"
			}
			updateEditorKeys(t, e, ebiten.KeyControl, key)
			if e.prompt.path != want {
				t.Fatalf("Ctrl+%s lost plain prompt binding: %q want %q", key, e.prompt.path, want)
			}
		})
	}
	for _, test := range []struct {
		key, modifier ebiten.Key
		start, want   int
	}{
		{ebiten.KeyArrowUp, ebiten.KeyShift, 1, 0},
		{ebiten.KeyArrowDown, ebiten.KeyShift, 0, 1},
		{ebiten.KeyHome, ebiten.KeyControl, 1, 0},
		{ebiten.KeyEnd, ebiten.KeyControl, 0, 1},
	} {
		t.Run("Add/"+test.modifier.String()+"+"+test.key.String(), func(t *testing.T) {
			e, _ := newModesEditor(t, "full")
			_ = e.Key("unit")
			_ = e.Key("escape")
			e.addTake(test.start)
			updateEditorKeys(t, e, test.modifier, test.key)
			r, ok := e.addRunning()
			if !ok || r.list.Selection() != test.want || e.prompt.path != []string{"1", "2"}[test.want] {
				t.Fatalf("modified Add list binding lost: %q, run %+v", e.prompt.path, r)
			}
		})
	}
}
