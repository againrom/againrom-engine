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

func TestEditorNativePlacementArrowsPan(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.placeStart(KindUnit, nil)
	c := e.doc.Viewer.Camera()
	c.Zoom, c.X, c.Y = 2, 400, 400
	c.Clamp()
	x := c.X
	updateEditorKeys(t, e, ebiten.KeyArrowLeft)
	if c.X >= x || !e.addPicking() {
		t.Fatal("arrow did not pan while placing")
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
}
