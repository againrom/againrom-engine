package ui

import (
	"image"
	"slices"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// TestMissionLayersNameWhatTheCPUFrameSkips pins the plan: the steps in
// paint order, and a reason beside every step or part of one the CPU frame
// leaves out.
func TestMissionLayersNameWhatTheCPUFrameSkips(t *testing.T) {
	want := []string{"map", "marks", "path strokes", "numerals", "backings", "panel", "spellbook",
		"readout", "frame rate", "minimap", "mission tip", "cursor mode", "column filler", "mission card",
		"structure readout", "worn", "pack bar", "command panel", "gold", "message", "tooltip",
		"popup backdrop", "pointer", "map to window", "canvas to window", "text overlay"}
	var got, skipped []string
	placed := false
	for _, l := range missionLayers {
		got = append(got, l.name)
		if l.window == nil {
			t.Errorf("%s: no window step", l.name)
		}
		if l.cpu == nil && l.cpuSkips == "" {
			t.Errorf("%s: the CPU frame skips it without saying so", l.name)
		}
		if l.cpu == nil {
			skipped = append(skipped, l.name)
		}
		if placed && !l.placed {
			t.Errorf("%s: a canvas step after a placed one", l.name)
		}
		placed = placed || l.placed
		if l.editor != (l.name == "map" || l.placed) {
			t.Errorf("%s: editor %v", l.name, l.editor)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("missionLayers = %q, want %q", got, want)
	}
	wantSkipped := []string{"path strokes", "readout", "frame rate", "mission tip", "cursor mode", "mission card",
		"structure readout", "worn", "gold", "tooltip", "popup backdrop", "pointer", "text overlay"}
	if !slices.Equal(skipped, wantSkipped) {
		t.Fatalf("CPU skips %q, want %q", skipped, wantSkipped)
	}
}

// TestBothExecutorsWalkTheMissionLayers runs Viewer.Draw and the headless
// frame over a recording copy of the list: the window runs every step once
// in order, the CPU every step that has a cpu, in the same order.
func TestBothExecutorsWalkTheMissionLayers(t *testing.T) {
	orig := missionLayers
	t.Cleanup(func() { missionLayers = orig })
	var window, cpu, all, composed []string
	rec := make([]missionLayer, len(orig))
	for i, l := range orig {
		r := l
		r.window = func(v *Viewer, screen *ebiten.Image, f *windowPass) {
			window = append(window, l.name)
			l.window(v, screen, f)
		}
		if l.cpu != nil {
			r.cpu = func(v *Viewer, f *cpuPass) error {
				cpu = append(cpu, l.name)
				return l.cpu(v, f)
			}
			composed = append(composed, l.name)
		}
		all = append(all, l.name)
		rec[i] = r
	}
	missionLayers = rec
	v := zoomViewer(t, false, true, 1.2)
	v.Layout(1920, 1080)
	screen := ebiten.NewImage(1920, 1080)
	v.Draw(screen)
	if _, _, err := v.headlessMissionFrame(image.Pt(1920, 1080)); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(window, all) {
		t.Errorf("window ran %q, want %q", window, all)
	}
	if !slices.Equal(cpu, composed) {
		t.Errorf("CPU ran %q, want %q", cpu, composed)
	}
}
