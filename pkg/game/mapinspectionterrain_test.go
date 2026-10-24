package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// openNewHost opens the new map of source with no install behind it.
func openNewHost(t *testing.T, source string) *mapInspectionEdits {
	t.Helper()
	x := &MapInspector{tiles: &terrain.Tileset{}}
	d, err := x.Open(source)
	if err != nil {
		t.Fatalf("%s: %v", source, err)
	}
	return d.Edits.(*mapInspectionEdits)
}

// The window's height constants are the format's: 64 by default, 126 at most.
func TestNewMapHeightWordsAreTheFormats(t *testing.T) {
	if ui.NewMapDefaultAltitude != int(alm.DefaultNewMap().Altitude) || ui.NewMapTopAltitude != alm.NewMapMaxAltitude {
		t.Fatalf("window %d..%d, format %d..%d", ui.NewMapDefaultAltitude, ui.NewMapTopAltitude, alm.DefaultNewMap().Altitude, alm.NewMapMaxAltitude)
	}
}

// A new-map source with a starting height opens a map whose every height cell
// holds it; one without starts at 64, and the multiplayer ending still
// follows. A height outside 0..126 or a malformed one is refused.
func TestNewMapSourceCarriesTheStartingHeight(t *testing.T) {
	for _, c := range []struct {
		source string
		height uint8
		mode   uint32
	}{
		{"new/32x32", 64, ui.MapModeSingle},
		{"new/32x32/altitude0", 0, ui.MapModeSingle},
		{"new/32x48/altitude126", 126, ui.MapModeSingle},
		{"new/32x32/altitude40" + ui.NewMapMultiplayer, 40, ui.MapModeMultiplayer},
	} {
		s := openNewHost(t, c.source)
		m, err := s.model.Map()
		if err != nil {
			t.Fatal(err)
		}
		for i, h := range m.Altitudes {
			if h != c.height {
				t.Fatalf("%s: cell %d holds %d, want %d", c.source, i, h, c.height)
			}
		}
		if m.Meta.Word70 != c.mode {
			t.Fatalf("%s: mode %d", c.source, m.Meta.Word70)
		}
	}
	plain, high := openNewHost(t, "new/32x32"), openNewHost(t, "new/32x32/altitude64")
	if !bytes.Equal(plain.model.Bytes(), high.model.Bytes()) {
		t.Fatal("the default height written out differs from the default")
	}
	for _, bad := range []string{"new/32x32/altitude127", "new/32x32/altitude-1", "new/32x32/altitude", "new/32x32/altitude4x", "new/32x32/altitude40/single"} {
		if _, _, err := newMapSpec(bad); err == nil {
			t.Errorf("%s was accepted", bad)
		}
	}
}

// The whole-map commands through the host: one seed gives one map, another
// seed another, each is one undo step, and the seed reaches the model's
// source as the brush's seeded source.
func TestHostWholeMapCommandsFollowTheSeed(t *testing.T) {
	for _, run := range []struct {
		name string
		call func(*mapInspectionEdits, uint32) (*ui.InspectionDocument, error)
		draw func(*mapedit.Editor, func(int) int) (bool, error)
	}{
		{"terrain", (*mapInspectionEdits).RandomTerrain, (*mapedit.Editor).RandomizeTerrain},
		{"heights", (*mapInspectionEdits).RandomHeights, (*mapedit.Editor).RandomizeHeights},
	} {
		a, b, c := openNewHost(t, "new/48x48"), openNewHost(t, "new/48x48"), openNewHost(t, "new/48x48")
		before := a.model.Bytes()
		for _, s := range []struct {
			h    *mapInspectionEdits
			seed uint32
		}{{a, 9}, {b, 9}, {c, 10}} {
			if d, err := run.call(s.h, s.seed); err != nil || d == nil {
				t.Fatalf("%s seed %d: %v %v", run.name, s.seed, d, err)
			}
		}
		if !bytes.Equal(a.model.Bytes(), b.model.Bytes()) || bytes.Equal(a.model.Bytes(), c.model.Bytes()) {
			t.Fatalf("%s: seed 9 twice equal %v, seeds 9 and 10 equal %v", run.name, bytes.Equal(a.model.Bytes(), b.model.Bytes()), bytes.Equal(a.model.Bytes(), c.model.Bytes()))
		}
		want, err := mapedit.New(before)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := run.draw(want, seedSource(9)); err != nil || !bytes.Equal(want.Bytes(), a.model.Bytes()) {
			t.Fatalf("%s: the host's map is not the model's for seed 9 (%v)", run.name, err)
		}
		if _, err := a.Undo(); err != nil || !bytes.Equal(a.model.Bytes(), before) {
			t.Fatalf("%s: one undo did not give back the map (%v)", run.name, err)
		}
	}
}

// An area copy with no records pastes its terrain, heights and objects at
// the target cell as one undo step, and the clip counts the patch.
func TestHostAreaCopyPastesTheTerrain(t *testing.T) {
	s := openNewHost(t, "new/48x48")
	if _, err := s.RandomHeights(3); err != nil {
		t.Fatal(err)
	}
	if _, err := s.change(func(m *mapedit.Editor) (bool, error) {
		for x := 4; x < 12; x += 2 {
			if err := m.SetOverlay(x, 6, uint8(x)); err != nil {
				return false, err
			}
		}
		return true, nil
	}); err != nil {
		t.Fatal(err)
	}
	box := image.Rect(3, 4, 13, 9)
	clip, note, err := s.CopyArea(nil, box)
	if err != nil || clip.Len() != 1 || note == "" {
		t.Fatalf("clip %v note %q: %v", clip, note, err)
	}
	src, _ := s.model.Map()
	before := s.model.Bytes()
	r, err := s.PasteRecords(clip, image.Pt(30, 20))
	if err != nil || r.Doc == nil {
		t.Fatalf("paste: %+v %v", r, err)
	}
	now, _ := s.model.Map()
	for y := 0; y < box.Dy(); y++ {
		for x := 0; x < box.Dx(); x++ {
			from, to := (box.Min.Y+y)*48+box.Min.X+x, (20+y)*48+30+x
			if now.Altitudes[to] != src.Altitudes[from] || now.Overlay[to] != src.Overlay[from] || now.Tiles[to] != src.Tiles[from] {
				t.Fatalf("cell (%d,%d) of the patch is not pasted", x, y)
			}
		}
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(s.model.Bytes(), before) {
		t.Fatalf("one undo did not give back the map (%v)", err)
	}
}
