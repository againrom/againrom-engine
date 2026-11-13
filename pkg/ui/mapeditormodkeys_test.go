package ui

import (
	"image"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// While a field of the mod window holds the keyboard, an arrow and a letter
// are the field's: the map does not scroll and no brush arms. A press on the
// map puts the window's focus down, the window stays open, and the same keys
// are the map's.
func TestModWindowKeepsFieldKeysAndAMapPressGivesThemToTheMap(t *testing.T) {
	e, _ := newModesEditor(t, "workspace")
	if err := e.Key("package"); err != nil || !e.modWindow {
		t.Fatalf("W: %v, open %v", err, e.modWindow)
	}
	if !e.FocusModField("title") || !e.objectTyping() {
		t.Fatal("the title field does not hold the keyboard")
	}
	pop, _ := e.PopupRect()
	area := e.mapArea()
	p, found := image.Point{}, false
	for _, c := range []image.Point{{area.Min.X + 40, area.Min.Y + 40}, {area.Max.X - 40, area.Min.Y + 40}, {area.Min.X + 40, area.Max.Y - 40}, {area.Max.X - 40, area.Max.Y - 40}} {
		if !c.In(pop) && e.chromeHit(c) == popNone {
			p, found = c, true
			break
		}
	}
	if !found {
		t.Fatalf("the window %v covers the map %v", pop, area)
	}
	update := func(primary bool, keys ...ebiten.Key) {
		t.Helper()
		down := map[ebiten.Key]bool{}
		for _, k := range keys {
			down[k] = true
		}
		pressed := func(k ebiten.Key) bool { return down[k] }
		e.nativeInput = func() editorNativeInput {
			return editorNativeInput{pressed: pressed, justPressed: pressed, keysDown: len(keys) > 0,
				canvas: Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: primary, PanLeft: down[ebiten.KeyArrowLeft]}}
		}
		if err := e.Update(); err != nil {
			t.Fatal(err)
		}
	}
	c := e.doc.Viewer.Camera()
	c.Zoom, c.X, c.Y = 2, 400, 400
	c.Clamp()
	x := c.X
	update(false, ebiten.KeyArrowLeft)
	update(false, ebiten.KeyT)
	if c.X != x || e.brush.mode != brushOff || !e.objectTyping() {
		t.Fatalf("a focused field's keys reached the map: camera %g from %g, brush %v", c.X, x, e.brush.mode)
	}
	update(true)
	update(false)
	if e.objectTyping() || !e.modWindow {
		t.Fatalf("after a press on the map: typing %v, window %v", e.objectTyping(), e.modWindow)
	}
	update(false, ebiten.KeyArrowLeft)
	if c.X == x {
		t.Error("the arrow did not scroll the map")
	}
	update(false, ebiten.KeyT)
	if e.brush.mode == brushOff {
		t.Error("T did not arm the terrain brush")
	}
}
