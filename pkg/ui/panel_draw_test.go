package ui

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

// The panel on screen: which unit it describes, where it lands, when it is
// rebuilt, and that a viewer with neither font nor selection draws exactly
// the frame it drew before this story (AC-4, AC-5, AC-11, AC-12, AC-14).

const panelGridW, panelGridH = 16, 16

// panelViewer is a viewer over a bare grid, with no font: the state every
// viewer built before this story is in.
func panelViewer(t *testing.T) *Viewer {
	t.Helper()
	v, err := NewViewer("panel", terrain.Grid{
		Width: panelGridW, Height: panelGridH, Tiles: make([]uint16, panelGridW*panelGridH),
	}, &terrain.Tileset{})
	if err != nil {
		t.Fatalf("NewViewer: %v", err)
	}
	v.cam.ViewW, v.cam.ViewH = 640, 480
	return v
}

// panelEntity is one snapshot entry: a named unit at a cell, alive unless its
// health says otherwise.
func panelEntity(id uint32, name string, hp, max, x, y int) MapEntity {
	e := MapEntity{ID: id, Name: name, HP: hp, MaxHP: max, Cell: image.Pt(x, y)}
	switch {
	case hp < 0:
		e.Life = LifeDead
	case hp == 0 && max > 0:
		e.Life = LifeDowned
	}
	return e
}

// AC-4, AC-5 — the subject is the filter's first survivor, in the selection's
// own order.
func TestPanelSubject(t *testing.T) {
	v := panelViewer(t)
	v.SetEntities([]MapEntity{
		panelEntity(1, "one", 10, 10, 1, 1),
		panelEntity(2, "two", -1, 10, 2, 2),
		panelEntity(3, "three", 0, 10, 3, 3),
	})

	t.Run("nothing selected yields no subject", func(t *testing.T) {
		v.sel = nil
		if s, ok := v.panelSubject(); ok {
			t.Errorf("an empty selection yielded the subject %+v", s)
		}
	})

	t.Run("the first survivor of the selection's own order", func(t *testing.T) {
		// 2 is dead and is skipped; 4 is not in the snapshot at all; 3 is
		// DOWNED and is kept, which is what makes it the answer here.
		v.sel = selection{2, 3, 4}
		s, ok := v.panelSubject()
		if !ok {
			t.Fatal("no subject over a selection holding a downed unit")
		}
		if s.ID != 3 || s.Name != "three" || s.HP != 0 || s.MaxHP != 10 || s.Cell != image.Pt(3, 3) {
			t.Errorf("subject = %+v, want entity 3's own name, health pair and cell", s)
		}
	})

	t.Run("several selected: the first, and nothing about the rest", func(t *testing.T) {
		v.sel = selection{1, 3}
		s, ok := v.panelSubject()
		if !ok || s.ID != 1 {
			t.Errorf("subject = %+v (ok=%v), want entity 1 — the selection's own first", s, ok)
		}
	})

	t.Run("a selection none of whose entries survives yields no subject", func(t *testing.T) {
		v.sel = selection{2, 4}
		if s, ok := v.panelSubject(); ok {
			t.Errorf("a dead-and-absent selection yielded the subject %+v", s)
		}
	})
}

// AC-11, AC-14 — a panel only where there is a font AND a subject, at the
// configured corner, and a viewer with neither presents nothing at all.
func TestPanelPresent(t *testing.T) {
	v := panelViewer(t)
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})

	t.Run("no font, no panel", func(t *testing.T) {
		v.sel = selection{1}
		if _, _, ok := v.panelPresent(); ok {
			t.Error("a viewer with no font presented a panel")
		}
		if v.panelBuilds != 0 {
			t.Errorf("a fontless viewer composed %d picture(s)", v.panelBuilds)
		}
	})

	v.SetFont(panelFont())

	t.Run("a font but nothing selected: the empty pane, not nothing", func(t *testing.T) {
		v.sel = nil
		pic, _, ok := v.panelPresent()
		if !ok || pic == nil {
			t.Fatal("a viewer with nothing selected presented no pane")
		}
		// THE CANVAS IS WIDER THAN THE ID-7 SLOT BY characterPaneSeamW (story
		// 1036 round 3, P-A): panelPresent's own doc explains why.
		want := image.Pt(sidebarWidth+characterPaneSeamW, compactPanelH)
		if got := pic.Bounds().Size(); got != want {
			t.Errorf("the empty pane is %v, want the id-7 slot's own %v widened by the seam",
				got, want)
		}
	})

	t.Run("a font and a subject: the picture at the decoded slot", func(t *testing.T) {
		v.sel = selection{1}
		pic, at, ok := v.panelPresent()
		if !ok || pic == nil {
			t.Fatal("no panel over a font and a subject")
		}
		want, ok := characterPanelBoxRect(image.Pt(v.frameW, v.frameH))
		if !ok {
			t.Fatal("setup: no character panel slot at the viewer's own frame size")
		}
		wantAt := image.Pt(want.Min.X-characterPaneSeamW, want.Min.Y)
		wantSize := image.Pt(want.Dx()+characterPaneSeamW, want.Dy())
		if at != wantAt {
			t.Errorf("origin = %v, want %v (characterPanelBoxRect's own slot, less the seam)", at, wantAt)
		}
		if pic.Bounds().Size() != wantSize {
			t.Errorf("picture size = %v, want the slot's own size %v widened by the seam", pic.Bounds().Size(), wantSize)
		}
	})
}

// AC-14 — the panel reaches nothing else the frame draws. The pass slice is
// what Draw consumes and its whole order, so a slice identical with and without
// a font is the whole of "nothing that drew before this story moved".
func TestPanelDoesNotDisturbTheFrame(t *testing.T) {
	ents := []MapEntity{panelEntity(1, "Warrior", 63, 100, 3, 4), panelEntity(2, "Archer", 9, 10, 5, 6)}

	fingerprint := func(v *Viewer) string {
		s := ""
		for _, p := range v.overlayPasses() {
			s += fmt.Sprintf("|%v/%d", p.Color, len(p.Rects))
		}
		return s
	}

	bare := panelViewer(t)
	bare.SetEntities(ents)
	bare.sel = selection{1, 2}
	before := fingerprint(bare)

	withFont := panelViewer(t)
	withFont.SetEntities(ents)
	withFont.sel = selection{1, 2}
	withFont.SetFont(panelFont())
	if got := fingerprint(withFont); got != before {
		t.Errorf("giving a viewer a font changed the pass slice:\n with: %s\n bare: %s", got, before)
	}
	// And the panel really was live on the second one, or the comparison above
	// compares two viewers that both draw no panel.
	if _, _, ok := withFont.panelPresent(); !ok {
		t.Fatal("the font-bearing viewer presented no panel: this case cannot witness AC-14")
	}
	if _, _, ok := bare.panelPresent(); ok {
		t.Error("the fontless viewer presented a panel")
	}
}

// AC-12 — the picture is rebuilt on a change of what it would show, and on no
// other frame.
func TestPanelRebuildsOnlyOnAChange(t *testing.T) {
	v := panelViewer(t)
	v.SetFont(panelFont())
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
	v.sel = selection{1}

	if _, _, ok := v.panelPresent(); !ok {
		t.Fatal("no panel to begin with")
	}
	if v.panelBuilds != 1 {
		t.Fatalf("the first frame composed %d picture(s), want 1", v.panelBuilds)
	}

	// Frames in which nothing changes.
	for i := 0; i < 5; i++ {
		v.panelPresent()
	}
	if v.panelBuilds != 1 {
		t.Errorf("five unchanged frames composed %d picture(s) in total, want 1", v.panelBuilds)
	}

	// Only the camera moves. The pixels do not depend on it.
	v.cam.X, v.cam.Y, v.cam.Zoom = 3, 4, 2
	v.panelPresent()
	if v.panelBuilds != 1 {
		t.Errorf("a camera move composed a picture: %d builds, want 1", v.panelBuilds)
	}

	// A stated value changes.
	v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 53, 100, 12, 34)})
	v.panelPresent()
	if v.panelBuilds != 2 {
		t.Errorf("a health change gave %d builds, want 2", v.panelBuilds)
	}

	// A DIFFERENT unit stating identical values. The picture is byte-identical,
	// so nothing about the pixels could witness this — only the id in the key
	// can, which is why it is in the key.
	v.SetEntities([]MapEntity{panelEntity(9, "Warrior", 53, 100, 12, 34)})
	v.sel = selection{9}
	v.panelPresent()
	if v.panelBuilds != 3 {
		t.Errorf("a new subject stating identical values gave %d builds, want 3 — the subject's "+
			"identity is not in the key", v.panelBuilds)
	}

	v.cam.ViewW, v.cam.ViewH = 800, 600
	v.panelPresent()
	if v.panelBuilds != 3 {
		t.Errorf("a camera-only resize gave %d builds, want 3 — unchanged", v.panelBuilds)
	}

	// THE FRAME RESIZING IS WHAT MOVES THE SLOT, and that is what the key
	// tracks now.
	v.frameW, v.frameH = 1200, 900
	v.panelPresent()
	if v.panelBuilds != 4 {
		t.Errorf("a frame resize gave %d builds, want 4", v.panelBuilds)
	}

	// Every rebuild marks the picture as not yet uploaded, and a frame that
	// rebuilds nothing leaves that mark alone. The upload itself is one engine
	// call inside Draw and is not reachable here; this flag is the whole of
	// what couples it to the picture, so it is asserted directly.
	v.panelFresh = false
	v.panelPresent()
	if v.panelFresh {
		t.Error("a frame that rebuilt nothing marked the picture for re-upload")
	}
	v.SetEntities([]MapEntity{panelEntity(9, "Warrior", 1, 100, 12, 34)})
	v.panelPresent()
	if !v.panelFresh {
		t.Error("a rebuilt picture was not marked for re-upload: the texture would stay stale")
	}

	// The layout is replaced.
	v.SetPanelLayout(AuthoredPanelLayout())
	v.panelPresent()
	if v.panelBuilds != 6 {
		t.Errorf("a replaced layout gave %d builds, want 6", v.panelBuilds)
	}
}

func TestPanelIsAFunctionOfItsLayout(t *testing.T) {
	build := func(l PanelLayout) *image.RGBA {
		v := panelViewer(t)
		v.SetFont(panelFont())
		v.SetPanelLayout(l)
		v.toggleHudPanel(hudPanelDoll)
		v.SetEntities([]MapEntity{panelEntity(1, "Warrior", 63, 100, 12, 34)})
		v.sel = selection{1}
		pic, _, ok := v.panelPresent()
		if !ok {
			t.Fatal("no panel")
		}
		return pic
	}

	same := func(a, b *image.RGBA) bool {
		if a.Bounds() != b.Bounds() {
			return false
		}
		for i := range a.Pix {
			if a.Pix[i] != b.Pix[i] {
				return false
			}
		}
		return true
	}

	if !same(build(AuthoredPanelLayout()), build(AuthoredPanelLayout())) {
		t.Error("two viewers sharing a layout drew different panels for the same subject")
	}

	other := AuthoredPanelLayout()
	other.Rows = []PanelRow{{Field: PanelFieldHealth, Label: "LIFE"}}
	if same(build(AuthoredPanelLayout()), build(other)) {
		t.Error("two viewers differing in their layout drew the same panel")
	}

	// The shipped layout is handed out fresh, so a caller mutating what it got
	// cannot reach anyone else's.
	a, b := AuthoredPanelLayout(), AuthoredPanelLayout()
	// Read what the shipped row says BEFORE the mutation rather than naming it
	// here: which row is first, and what it is labelled, is the authored
	// layout's business and moves when a story adds a row.
	was := b.Rows[0].Label
	a.Rows[0].Label = "mutated"
	if b.Rows[0].Label != was {
		t.Error("mutating one AuthoredPanelLayout's rows reached another's: the slice is shared")
	}
}
