package game

import (
	"bytes"
	"testing"
)

// TestReleaseSecondMissionTenMiddleDragAndAltEnter runs the window toggle and
// the middle-button drag on the second game's mission 10, through the App the
// game runs. The drag moves the view so that the hero's cell stays under the
// pointer, and the world after the drag equals the world of a session that
// only hovered over the same points. Then Alt+Enter flips the window form
// twice and leaves the mission screen and the camera where they were.
func TestReleaseSecondMissionTenMiddleDragAndAltEnter(t *testing.T) {
	secondGameRoot(t)
	run := func(middle bool) (uint64, []byte, [2]int, [2]int) {
		f, app := secondMissionSettled(t)
		hero := f.live.mission.ids[0]
		e, _ := f.live.world.Entity(hero)
		f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
		x0, y0, err := app.HeadlessCellPoint(int(e.X), int(e.Y))
		if err != nil {
			t.Fatal(err)
		}
		press, move, release := "hover", "hover", "hover"
		if middle {
			press, move, release = "middle-press", "middle-move", "middle-release"
		}
		const travelX, travelY = -40, -30
		for _, step := range []struct {
			action string
			x, y   int
		}{{press, x0, y0}, {move, x0 + travelX/2, y0 + travelY/2}, {move, x0 + travelX, y0 + travelY}, {release, x0 + travelX, y0 + travelY}} {
			if err := app.HeadlessPointer(step.action, step.x, step.y); err != nil {
				t.Fatal(err)
			}
		}
		x1, y1, err := app.HeadlessCellPoint(int(e.X), int(e.Y))
		if err != nil {
			t.Fatal(err)
		}
		hash, world := f.live.world.Hash(), marshalWorld(t, f.live.world)
		if middle {
			cam := f.live.view.Camera()
			cx, cy, zoom := cam.X, cam.Y, cam.Zoom
			screen := app.HeadlessGameplayScreen()
			windowed := app.Windowed()
			for range 2 {
				if err := app.HeadlessKey("alt-enter"); err != nil {
					t.Fatal(err)
				}
				if app.Windowed() == windowed {
					t.Fatal("Alt+Enter did not switch the window form")
				}
				windowed = app.Windowed()
				if app.HeadlessGameplayScreen() != screen || cam.X != cx || cam.Y != cy || cam.Zoom != zoom {
					t.Fatal("Alt+Enter moved the screen or the camera")
				}
			}
		}
		return hash, world, [2]int{x0, y0}, [2]int{x1, y1}
	}
	hoveredHash, hovered, h0, h1 := run(false)
	if h0 != h1 {
		t.Fatalf("hovering moved the hero's cell from %v to %v", h0, h1)
	}
	draggedHash, dragged, d0, d1 := run(true)
	if d0 != h0 {
		t.Fatalf("the two sessions started with the hero's cell at %v and %v", h0, d0)
	}
	if want := [2]int{d0[0] - 40, d0[1] - 30}; d1 != want {
		t.Fatalf("after the middle drag the hero's cell is at %v, want %v under the pointer", d1, want)
	}
	if hoveredHash != draggedHash || !bytes.Equal(hovered, dragged) {
		t.Fatalf("the middle drag changed the world: hash %#x against %#x", draggedHash, hoveredHash)
	}
}
