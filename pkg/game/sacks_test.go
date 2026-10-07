package game

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// sackGraphicsContainers lays a synthetic archive down as graphics.res in the
// test's own temporary tree and opens the container filesystem over it — the
// same route NewFrontEnd's own read takes (openContainers' precedent,
// statics_test.go, rebuilt here because that helper lives in package
// game_test and this file needs package game's unexported seams beside it).
func sackGraphicsContainers(t *testing.T, files []synth.File) *vfs.FS {
	t.Helper()
	path := filepath.Join(t.TempDir(), GraphicsArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	f, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return f
}

// sackSheetFixture is a three-frame sheet, every frame a DIFFERENT size —
// oakSheet's own reason (statics_test.go): a loader that reordered the sheet,
// or carried one frame's dimensions onto another, is caught by nothing else.
func sackSheetFixture() []byte {
	pal := make([]color.RGBA, 4)
	pal[1] = color.RGBA{R: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{
			{Width: 1, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}}},
			{Width: 2, Height: 3, Pixels: make([]synth.Pixel256, 6)},
			{Width: 4, Height: 2, Pixels: make([]synth.Pixel256, 8)},
		},
	})
}

func TestTheSackSheetAddressIsContainerRelative(t *testing.T) {
	if got := graphicsPrefix + sackSheetPath; got != "graphics/backpack/sprites.256" {
		t.Errorf("the sheet is read from %q, want graphics/backpack/sprites.256 — "+
			"sackSheetPath is %q and must be the container-relative remainder alone",
			got, sackSheetPath)
	}
}

func TestLoadSackFrames(t *testing.T) {
	t.Run("every frame loads, in the sheet's own order", func(t *testing.T) {
		src := sackGraphicsContainers(t, []synth.File{{Path: sackSheetPath, Data: sackSheetFixture()}})
		frames := LoadSackFrames(src)
		if len(frames) != 3 {
			t.Fatalf("%d frame(s), want 3", len(frames))
		}
		want := [3][2]int{{1, 1}, {2, 3}, {4, 2}}
		for i, w := range want {
			if frames[i].Width != w[0] || frames[i].Height != w[1] {
				t.Errorf("frame %d is %dx%d, want %dx%d", i, frames[i].Width, frames[i].Height, w[0], w[1])
			}
		}
	})

	t.Run("an archive holding no sheet at all yields none, not an error (AC-2)", func(t *testing.T) {
		src := sackGraphicsContainers(t, nil)
		if frames := LoadSackFrames(src); frames != nil {
			t.Errorf("LoadSackFrames = %v, want nil for an absent entry", frames)
		}
	})

	t.Run("a stream the decoder refuses yields none (AC-2)", func(t *testing.T) {
		// A frame record whose block is 4096 bytes long and one byte wide, on
		// staticArchiveFiles' own broken-sheet fixture (statics_test.go): the
		// stream cannot be walked at all, so spr256 refuses it.
		src := sackGraphicsContainers(t, []synth.File{{Path: sackSheetPath, Data: synth.Sheet256Raw(
			synth.Palette256(make([]color.RGBA, 4)),
			[]synth.Sheet256RawFrame{{Width: 8, Height: 8, DataSize: 4096, Data: []byte{0x88}}},
			0x80000000|1)}})
		if frames := LoadSackFrames(src); frames != nil {
			t.Errorf("LoadSackFrames = %v, want nil for a stream spr256 refuses", frames)
		}
	})

	t.Run("a palette-less sheet yields none (AC-2)", func(t *testing.T) {
		src := sackGraphicsContainers(t, []synth.File{{Path: sackSheetPath, Data: synth.Sheet256(synth.Sheet256Options{
			NoPalette: true,
			Frames:    []synth.Frame256{{Width: 2, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}}}},
		})}})
		if frames := LoadSackFrames(src); frames != nil {
			t.Errorf("LoadSackFrames = %v, want nil for a palette-less sheet", frames)
		}
	})

	t.Run("a nil archive yields none, not a panic", func(t *testing.T) {
		if frames := LoadSackFrames(nil); frames != nil {
			t.Errorf("LoadSackFrames(nil) = %v, want nil", frames)
		}
	})
}

// TestSackFrameIndexWalksTheDecodedValueLadder — ITEM-136, ITEM-137,
// ITEM-138, SPR256-077 through sacks.go. The frame is a ladder at the powers
// of ten over the sack's own value, gold plus each item's own Price
// (ITEM-SACK-010), clamped at 5.
//
// EVERY BAND IS PINNED AT BOTH EDGES, because an off-by-one at a threshold is
// exactly what a log10-shaped rule gets wrong: 9 and 10 must differ, 10 and 99
// must not, and 99999 must still be 4. A test that only sampled band interiors
// would pass against `value/10` and against a float log10 that rounds 1000
// down to 2.9999.
func TestSackFrameIndexWalksTheDecodedValueLadder(t *testing.T) {
	for _, c := range []struct {
		gold  uint32
		want  int
		label string
	}{
		{0, 0, "empty: not decoded, authored to the lowest frame (DIV-1358)"},
		{1, 0, "band 0 lower edge"},
		{9, 0, "band 0 upper edge"},
		{10, 1, "band 1 lower edge"},
		{99, 1, "band 1 upper edge"},
		{100, 2, "band 2 lower edge"},
		{999, 2, "band 2 upper edge"},
		{1000, 3, "band 3 lower edge"},
		{9999, 3, "band 3 upper edge"},
		{10000, 4, "band 4 lower edge"},
		{99999, 4, "band 4 upper edge"},
		{100000, 5, "band 5 lower edge, the clamp"},
		{999999, 5, "well past the clamp"},
		{4000000000, 5, "a value no sheet frame can exceed"},
	} {
		if got := sackFrameIndex(sim.Sack{Gold: c.gold}); got != c.want {
			t.Errorf("sackFrameIndex(gold %d) = %d, want %d (%s)", c.gold, got, c.want, c.label)
		}
	}

	// ITEM-SACK-010: the value is gold PLUS each contained item's own value,
	// so items alone can raise the frame and the two sources add rather than
	// one masking the other. 60 gold and two 20-value items make 100, the
	// exact band-2 edge, which neither source reaches by itself.
	mixed := sim.Sack{Gold: 60, ItemInstances: []sim.ItemInstance{
		{Code: 1, Price: 20}, {Code: 2, Price: 20},
	}}
	if got := sackFrameIndex(mixed); got != 2 {
		t.Errorf("sackFrameIndex(60 gold + 2x20) = %d, want 2 (gold and item value add)", got)
	}
	if got := sackFrameIndex(sim.Sack{Gold: 60}); got != 1 {
		t.Errorf("the same sack without its items = %d, want 1 (items must contribute)", got)
	}
	if got := sackFrameIndex(sim.Sack{ItemInstances: mixed.ItemInstances}); got != 1 {
		t.Errorf("the same items without the gold = %d, want 1 (gold must contribute)", got)
	}

	// An item this build cannot price contributes zero rather than refusing:
	// it can only push a sack DOWN a band, never up.
	unpriced := sim.Sack{Gold: 100, ItemInstances: []sim.ItemInstance{{Code: 0xffff}}}
	if got := sackFrameIndex(unpriced); got != 2 {
		t.Errorf("sackFrameIndex(100 gold + one unpriced item) = %d, want 2", got)
	}

	// ITEM-138: the ladder ascends. Walked over the whole decoded domain, the
	// index never falls.
	last := 0
	for value := int64(1); value <= 2000000; value += value/3 + 1 {
		got := sackFrameIndex(sim.Sack{Gold: uint32(value)})
		if got < last {
			t.Fatalf("value %d gave frame %d after %d: the ladder must ascend (ITEM-138)", value, got, last)
		}
		if got < 0 || got >= sackFrameCount {
			t.Fatalf("value %d gave frame %d, outside the sheet's %d frames", value, got, sackFrameCount)
		}
		last = got
	}
	if last != sackFrameCount-1 {
		t.Fatalf("the walk ended at frame %d; it must reach the clamp at %d", last, sackFrameCount-1)
	}
}

// sackDrawsWorld is a small world holding three ground sacks at cells whose
// (Y, X) ordering DIFFERS from the order they are named in — sim.normaliseSacks
// sorts ascending by (Y, X) before the world ever holds them (pkg/sim/sack.go),
// so this is the discriminating fixture: a builder that walked its OWN
// argument order, rather than reading the world's answer back, would still
// pass a fixture given in sorted order and fail only this one.
func sackDrawsWorld(t *testing.T) *sim.World {
	t.Helper()
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: 20, Height: 20}, sim.ModeCanonical,
		sim.Terrain{}, nil, nil, sim.Relations{}, []sim.Sack{
			{X: 5, Y: 9, Gold: 40, Items: []uint16{7}},
			{X: 2, Y: 3, Items: []uint16{1, 2}},
			{X: 11, Y: 3},
		})
	if err != nil {
		t.Fatalf("NewLootWorld: %v", err)
	}
	return w
}

func sackDrawsDriver(t *testing.T) *mapWorld {
	t.Helper()
	m := worldFixtureMap()
	return newMapWorld(sackDrawsWorld(t), nil, nil, worldFixtureViewer(t, m))
}

func TestSackDrawsBuildsOneRecordPerWorldEntryInTheWorldsOwnOrder(t *testing.T) {
	mw := sackDrawsDriver(t)
	got := mw.sackDraws()

	want := []ui.MapSack{
		{Cell: image.Pt(2, 3), FrameIndex: 0},
		{Cell: image.Pt(11, 3), FrameIndex: 0},
		// 40 gold, the only entry with any value: band 1, not band 0.
		{Cell: image.Pt(5, 9), FrameIndex: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sackDraws() = %+v\nwant %+v (the world's own ascending-(Y,X) order)", got, want)
	}

	sacks := mw.world.Sacks()
	if len(got) != len(sacks) {
		t.Fatalf("sackDraws() returned %d record(s), want %d (one per world sack)", len(got), len(sacks))
	}
	for i, s := range sacks {
		if want := (image.Point{X: int(s.X), Y: int(s.Y)}); got[i].Cell != want {
			t.Errorf("record %d cell = %v, want the world's own %v", i, got[i].Cell, want)
		}
	}
}

func TestSackDrawsOnAWorldWithNoSacksDrawsNone(t *testing.T) {
	m := worldFixtureMap()
	w, err := sim.NewWorld(1, sim.Bounds{Width: int32(m.Width), Height: int32(m.Height)}, sim.ModeCanonical, nil, nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	mw := newMapWorld(w, nil, nil, worldFixtureViewer(t, m))
	if got := mw.sackDraws(); len(got) != 0 {
		t.Errorf("sackDraws() = %+v on a world with no sacks, want none", got)
	}
}

func TestSackDrawsIsIdempotentAcrossTwoRefreshesOfAnUnchangedWorld(t *testing.T) {
	mw := sackDrawsDriver(t)
	first := mw.sackDraws()
	second := mw.sackDraws()
	if !reflect.DeepEqual(first, second) {
		t.Errorf("two refreshes of an unchanged world differ:\n first  %+v\n second %+v", first, second)
	}
}

func TestPushDoesNotDisturbWorldState(t *testing.T) {
	mw := sackDrawsDriver(t)
	before := mw.world.Hash()
	mw.push()
	mw.push()
	if got := mw.world.Hash(); got != before {
		t.Errorf("two pushes moved the digest to %#016x from %#016x", got, before)
	}
	if got := mw.world.Tick(); got != 0 {
		t.Errorf("push advanced the tick to %d, want 0 — push is a pure read", got)
	}
}

func TestPushHandsTheDrawnSacksToTheViewer(t *testing.T) {
	mw := sackDrawsDriver(t)
	want := len(mw.sackDraws())
	if want == 0 {
		t.Fatal("setup: the fixture world holds no sack, so this test measures nothing")
	}
	if got, _ := mw.view.SackMarkers(); got != want {
		t.Errorf("the viewer holds %d sack(s) after the constructor's push, want the "+
			"world's %d — push did not hand them over", got, want)
	}
	mw.push()
	if got, _ := mw.view.SackMarkers(); got != want {
		t.Errorf("the viewer holds %d sack(s) after a second push, want %d", got, want)
	}
}

// looseSackMap is the picker-path map: one loose .alm file, reached through
// the LOOSE filesystem exactly as looseOver's own callers reach theirs
// (frontend_test.go).
func looseSackMap(t *testing.T, dir string) {
	t.Helper()
	data := synth.ALM(synth.ALMOptions{Width: 6, Height: 5, Name: "Loot",
		Units: []synth.ALMUnit{{X: 0x0180, Y: 0x0300}}})
	if err := os.WriteFile(filepath.Join(dir, "a.alm"), data, 0o644); err != nil {
		t.Fatalf("write a.alm: %v", err)
	}
}

// sackFrontEnd is a front-end whose graphics container holds the sack sheet
// fixture (decoded through LoadSackFrames, the same call NewFrontEnd itself
// makes), whose scenario container holds mission 10, and whose loose
// filesystem holds one picker row — so BOTH opener paths plan R-4 names can
// be driven from one fixture.
func sackFrontEnd(t *testing.T) *FrontEnd {
	t.Helper()
	dir := t.TempDir()
	looseSackMap(t, dir)

	graphics := synth.Archive([]synth.File{{Path: sackSheetPath, Data: sackSheetFixture()}})
	if err := os.WriteFile(filepath.Join(dir, GraphicsArchive), graphics, 0o644); err != nil {
		t.Fatalf("write %s: %v", GraphicsArchive, err)
	}
	scenario := synth.Archive([]synth.File{
		{Path: "10.alm", Data: synth.ALM(synth.ALMOptions{Width: 128, Height: 128,
			Units: []synth.ALMUnit{{X: 0x1480, Y: 0x1480}}})},
		{Path: "npc.reg", Data: synth.NPCReg(nil)},
	})
	if err := os.WriteFile(filepath.Join(dir, ScenarioArchive), scenario, 0o644); err != nil {
		t.Fatalf("write %s: %v", ScenarioArchive, err)
	}

	containers, err := vfs.Open([]string{
		filepath.Join(dir, GraphicsArchive),
		filepath.Join(dir, ScenarioArchive),
	}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}

	frames := LoadSackFrames(containers)
	if len(frames) != 3 {
		t.Fatalf("setup: LoadSackFrames returned %d frame(s), want the fixture's 3", len(frames))
	}

	return &FrontEnd{InstallResources: InstallResources{Archives: &Archives{Root: dir, Containers: containers, Loose: looseOver(t, dir)}, Tiles: &terrain.Tileset{}, Maps: []MapEntry{{Source: "a.alm", Name: "Loot"}}, SackFrames: frames}}
}

func TestTheSackSheetInstallsOnBothOpenerPaths(t *testing.T) {
	f := sackFrontEnd(t)

	t.Run("the picker's path (loadMap)", func(t *testing.T) {
		v, tick, _, _, _, _, _, _, _, _, err := f.loadMap(0)
		if err != nil {
			t.Fatalf("loadMap: %v", err)
		}
		if v == nil {
			t.Fatal("loadMap returned a nil viewer")
		}
		tick()
		if got := v.EntityMarkers(); got != 1 {
			t.Errorf("the viewer holds %d entity, want 1 — the map did not open cleanly", got)
		}
		if _, frames := v.SackMarkers(); frames != 3 {
			t.Errorf("the picker's viewer holds %d sack frame(s), want the fixture's 3 — "+
				"loadMap did not install the sheet", frames)
		}
	})

	t.Run("the mission path (MissionOpener) — the sharp call site", func(t *testing.T) {
		v, tick, _, _, _, _, _, _, _, _, err := f.MissionOpener(10)()
		if err != nil {
			t.Fatalf("MissionOpener(10): %v", err)
		}
		if v == nil {
			t.Fatal("MissionOpener returned a nil viewer")
		}
		tick()
		// The map's own one unit PLUS the party member StartMissionFrom adds
		// (TestTheMissionPartyIsOneMemberAtTheStartCell's own count) — two, not
		// one: a mission world always holds one more entity than its map alone
		// placed.
		if got := v.EntityMarkers(); got != 2 {
			t.Errorf("the viewer holds %d entity/entities, want 2 — the mission did not open cleanly", got)
		}
		if _, frames := v.SackMarkers(); frames != 3 {
			t.Errorf("the mission's viewer holds %d sack frame(s), want the fixture's 3 — "+
				"MissionOpener did not install the sheet, and every mission would draw no "+
				"sack while the picker's path went on working (plan R-4)", frames)
		}
	})
}

func TestAFramelessSackSheetStillOpensBothPaths(t *testing.T) {
	f := sackFrontEnd(t)
	f.SackFrames = nil

	if _, _, _, _, _, _, _, _, _, _, err := f.loadMap(0); err != nil {
		t.Errorf("loadMap with no sack sheet: %v", err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(10)(); err != nil {
		t.Errorf("MissionOpener with no sack sheet: %v", err)
	}
}
