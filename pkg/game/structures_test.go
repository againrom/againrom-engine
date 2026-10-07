package game_test

import (
	"errors"
	"image/color"
	"io/fs"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/alm"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The placement keys the fixture registry answers. A structure registry is
// subscripted by ID directly — there is no b-1 offset — so a key IS the ID it
// names (REG-KEY-044).
const (
	structGood      = 1 // a whole class: a 2x1 rectangle of FullHeight 3
	structNoSheet   = 2 // names art the archive does not hold
	structNoPalette = 3 // a palette-less sheet
	structOddFrame  = 4 // a sheet holding one 32x31 frame
	structZeroWide  = 5 // TileWidth 0
	structVariable  = 6 // VariableSize set — a bridge subclass
	structNoClass   = 7 // no section carries it
	structBridge1V  = 33
	structFarID     = 300
)

func structPalette() []color.RGBA {
	pal := make([]color.RGBA, 8)
	pal[0] = color.RGBA{R: 0x11, G: 0x22, B: 0x33}
	pal[1] = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
	pal[7] = color.RGBA{R: 0x01, G: 0x02, B: 0x04}
	return pal
}

func structInt(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func structStr(name, s string) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x00, Str: s}
}

// structSection is one [StructureN] body: the identity, the path and the three
// extents, which is every key the loader reads that this fixture varies.
func structSection(id int32, file string, tw, th, fh int32, extra ...synth.RegNode) []synth.RegNode {
	return append([]synth.RegNode{
		structInt("ID", id), structStr("File", file),
		structInt("TileWidth", tw), structInt("TileHeight", th), structInt("FullHeight", fh),
	}, extra...)
}

// structRegistry writes the fixture structures/structures.reg: the original
// exclusion cases, the supported class-33 vertical bridge, and one ID above the
// placement-key byte domain.
func structRegistry() []byte {
	return synth.StructuresReg(
		// ID 0, which the domain 1..66 does not contain and which no placement
		// key's low byte can reach: byte 0 is *no structure*. It is in the fixture
		// so that "Classes[0] stays nil" is a claim about the WALK rather than
		// about a registry that happened to carry no such class.
		structSection(0, `houses\hut`, 2, 1, 3),
		structSection(structGood, `houses\hut`, 2, 1, 3),
		structSection(structNoSheet, `houses\gone`, 2, 1, 3),
		structSection(structNoPalette, `houses\nopal`, 2, 1, 3),
		structSection(structOddFrame, `houses\odd`, 2, 1, 3),
		structSection(structZeroWide, `houses\hut`, 0, 1, 3),
		structSection(structVariable, `houses\hut`, 1, 1, 1, structInt("VariableSize", 1)),
		structSection(structBridge1V, `bridges\bridge1v`, 1, 1, 1,
			structInt("VariableSize", 1), structInt("Flat", 1)),
		// Above the byte domain: loaded by pkg/data, reachable by no placement
		// key, and so absent from the bundle entirely.
		structSection(structFarID, `houses\hut`, 2, 1, 3),
	)
}

// structSquare is a sheet of n frames, every one CellSize square, each filled
// with its own palette index so a consumer that mixed two grid indices up draws
// a frame whose pixels say which one it is.
func structSquare(n int) []byte {
	return synth.StructureSheet(synth.StructureSheetOptions{
		Frames: n, Width: terrain.CellSize, Height: terrain.CellSize,
		Palette: structPalette(),
		Ink:     func(frame, x, y int) (uint8, bool) { return uint8(frame % 8), true },
	})
}

// structArchiveFiles is the fixture graphics archive: the registry, and every
// sheet its sections name EXCEPT houses\gone.
func structArchiveFiles(t *testing.T) []synth.File {
	t.Helper()
	return []synth.File{
		{Path: graphicsEntry(t, game.StructureRegistry), Data: structRegistry()},
		{Path: "structures/houses/hut.256", Data: structSquare(6)},
		{Path: "structures/bridges/bridge1v.256", Data: structSquare(9)},
		{Path: "structures/houses/nopal.256", Data: synth.StructureSheet(synth.StructureSheetOptions{
			Frames: 6, Width: terrain.CellSize, Height: terrain.CellSize, NoPalette: true,
		})},
		// ONE frame of the six is 32x31. The exclusion is about a frame, so a
		// sheet that broke the size everywhere could not tell a whole-sheet rule
		// from a per-frame one.
		{Path: "structures/houses/odd.256", Data: synth.StructureSheet(synth.StructureSheetOptions{
			Frames: 6, Width: terrain.CellSize, Height: terrain.CellSize,
			Odd:     map[int][2]int{4: {terrain.CellSize, terrain.CellSize - 1}},
			Palette: structPalette(),
		})},
	}
}

func loadFixtureStructures(t *testing.T) *terrain.StructureSet {
	t.Helper()
	set, err := game.LoadStructures(openContainers(t, synth.Archive(structArchiveFiles(t))))
	if err != nil {
		t.Fatalf("LoadStructures: %v", err)
	}
	if set == nil {
		t.Fatal("LoadStructures returned a nil set with no error")
	}
	return set
}

// AC-1. The seven classes, each exclusion counted apart: the first draws, the
// next five resolve and draw nothing, and the last is reachable by no key.
func TestLoadStructuresExclusions(t *testing.T) {
	set := loadFixtureStructures(t)

	t.Run("the whole class draws", func(t *testing.T) {
		c := set.Classes[structGood]
		if c == nil {
			t.Fatal("the whole class resolved to no class at all")
		}
		if c.TileWidth != 2 || c.TileHeight != 1 || c.FullHeight != 3 {
			t.Errorf("extents = %dx%d, FullHeight %d, want 2x1 and 3", c.TileWidth, c.TileHeight, c.FullHeight)
		}
		if len(c.Frames) != 6 {
			t.Fatalf("%d frames, want the sheet's 6", len(c.Frames))
		}
		for i, f := range c.Frames {
			if f == nil || f.Width != terrain.CellSize || f.Height != terrain.CellSize {
				t.Fatalf("frame %d is not a %d-square cell", i, terrain.CellSize)
			}
		}
		// The palette rides on the frame and is the SHEET's own, at full opacity.
		if got := c.Frames[0].Palette[7]; got != (color.RGBA{R: 0x01, G: 0x02, B: 0x04, A: 0xff}) {
			t.Errorf("palette entry 7 = %+v, want the fixture colour at full alpha", got)
		}
	})

	// Each of the five draws nothing, and none of them is a MISSING class: the
	// census must be able to say "this key named a class we could not draw"
	// apart from "this key named nothing".
	for _, tc := range []struct {
		name string
		key  int
	}{
		{"an absent sheet", structNoSheet},
		{"a palette-less sheet", structNoPalette},
		{"a sheet holding a 32x31 frame", structOddFrame},
		{"a zero extent", structZeroWide},
		{"a VariableSize class", structVariable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c := set.Classes[tc.key]
			if c == nil {
				t.Fatalf("%s resolved to NO CLASS; it must resolve to a class that draws nothing", tc.name)
			}
			if len(c.Frames) != 0 {
				t.Errorf("%s carries %d frames, want none", tc.name, len(c.Frames))
			}
		})
	}

	t.Run("VariableSize is carried as its own answer", func(t *testing.T) {
		if !set.Classes[structVariable].VariableSize {
			t.Error("the bridge subclass does not carry VariableSize; it would be counted as merely undrawable")
		}
		if set.Classes[structGood].VariableSize {
			t.Error("the whole class carries VariableSize")
		}
	})

	t.Run("the vertical wooden bridge carries its exact selector and sheet", func(t *testing.T) {
		c := set.Classes[structBridge1V]
		if c == nil {
			t.Fatal("class 33 resolved to no class")
		}
		if !c.VariableSize || c.VariableLayout != terrain.VariableStructureVerticalNinePatch {
			t.Fatalf("class 33 variable layout = %v/%v, want vertical nine-patch",
				c.VariableSize, c.VariableLayout)
		}
		if len(c.Frames) != 9 {
			t.Fatalf("class 33 frames = %d, want exact nine-frame sheet", len(c.Frames))
		}
	})

	t.Run("a key naming no section resolves to no class", func(t *testing.T) {
		if set.Classes[structNoClass] != nil {
			t.Errorf("key %d resolved to a class; no section carries that ID", structNoClass)
		}
	})

	t.Run("an ID above 255 is reachable by no key", func(t *testing.T) {
		// The whole array, not just the low byte of structFarID: a class outside
		// the byte domain must be absent from EVERY slot, not merely from the
		// one a truncation would have put it in.
		for b, c := range set.Classes {
			if c != nil && (b < structGood || b > structVariable) && b != structBridge1V {
				t.Errorf("Classes[%d] is filled; the fixture's reachable IDs are %d..%d",
					b, structGood, structVariable)
			}
		}
		if set.Classes[structFarID&0xff] != nil {
			t.Errorf("Classes[%d] is filled; an ID of %d is outside the byte domain",
				structFarID&0xff, structFarID)
		}
	})

	t.Run("byte 0 names nothing", func(t *testing.T) {
		// The fixture registry DOES carry a class whose ID is 0, so this is a
		// claim about the walk starting at 1 and not about an empty slot.
		if set.Classes[0] != nil {
			t.Error("Classes[0] is filled; the ID domain is 1..66 and byte 0 can name no class")
		}
	})
}

func TestStructureRecordsCarryOnlyTheWholeKind21Extent(t *testing.T) {
	ext := []byte{3, 9, 9, 9, 6, 9, 9, 9}
	recs := game.StructureRecords([]alm.Object{
		{X: 1, Y: 2, Kind: 0x21, Ext: ext},
		{X: 3, Y: 4, Kind: 0x121, Ext: ext},
		{X: 5, Y: 6, Kind: 0x21, Ext: []byte{3, 0, 0, 0}},
		{X: 7, Y: 8, Kind: 0x21, Ext: make([]byte, 8)},
	})
	if got := recs[0]; got.VariableWidth != 3 || got.VariableHeight != 6 {
		t.Fatalf("kind 0x21 extent = %dx%d, want 3x6", got.VariableWidth, got.VariableHeight)
	}
	for i := 1; i < len(recs); i++ {
		if recs[i].VariableWidth != 0 || recs[i].VariableHeight != 0 {
			t.Errorf("record %d extent = %dx%d, want absent", i,
				recs[i].VariableWidth, recs[i].VariableHeight)
		}
	}
}

// AC-1's error clause: ONLY an unreadable or unparseable registry is an error.
// Every class the loader cannot draw is a skip, so a bundle always comes back
// when the registry reads.
func TestLoadStructuresErrorsOnlyOnTheRegistry(t *testing.T) {
	t.Run("an absent registry", func(t *testing.T) {
		_, err := game.LoadStructures(openContainers(t, synth.Archive(
			[]synth.File{{Path: "structures/houses/hut.256", Data: structSquare(6)}})))
		if !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("err = %v, want fs.ErrNotExist", err)
		}
	})

	t.Run("an unparseable registry", func(t *testing.T) {
		_, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
			{Path: graphicsEntry(t, game.StructureRegistry), Data: []byte("not a .reg stream at all")},
		})))
		if err == nil {
			t.Fatal("an unparseable registry loaded without error")
		}
	})

	t.Run("no source at all", func(t *testing.T) {
		if _, err := game.LoadStructures(nil); err == nil {
			t.Fatal("a nil source loaded without error")
		}
	})

	t.Run("an archive holding the registry and no art", func(t *testing.T) {
		// Every class is undrawable and NONE of that is an error: the layer draws
		// what it can and never fails a run on data it cannot use.
		set, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
			{Path: graphicsEntry(t, game.StructureRegistry), Data: structRegistry()},
		})))
		if err != nil {
			t.Fatalf("LoadStructures: %v", err)
		}
		for b, c := range set.Classes {
			if c != nil && len(c.Frames) != 0 {
				t.Errorf("Classes[%d] carries art from an archive holding none", b)
			}
		}
	})
}

// Two classes naming ONE sheet receive the same converted slice, pointer for
// pointer, so the window's frame-keyed texture cache uploads a shared sheet
// once (inherited from the object loader's memo).
func TestLoadStructuresSharesOneSheetPerPath(t *testing.T) {
	set, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
		{Path: graphicsEntry(t, game.StructureRegistry), Data: synth.StructuresReg(
			structSection(1, `houses\hut`, 2, 1, 3),
			structSection(2, `houses\hut`, 3, 2, 1),
		)},
		{Path: "structures/houses/hut.256", Data: structSquare(6)},
	})))
	if err != nil {
		t.Fatalf("LoadStructures: %v", err)
	}
	a, b := set.Classes[1], set.Classes[2]
	if a == nil || b == nil || len(a.Frames) == 0 || len(b.Frames) == 0 {
		t.Fatal("both fixture classes must draw")
	}
	for i := range a.Frames {
		if a.Frames[i] != b.Frames[i] {
			t.Fatalf("frame %d differs by pointer between two classes naming one sheet", i)
		}
	}
}

// The address the bundle is loaded from carries the graphics container's own
// identity segment, so this package states which container its classes come
// from rather than leaving a caller to know it out of band.
func TestStructureRegistryIsAddressedByContainer(t *testing.T) {
	if got, want := game.StructureRegistry, "graphics/structures/structures.reg"; got != want {
		t.Errorf("StructureRegistry = %q, want %q", got, want)
	}
}

// structuresMapKey is the class key every record of the fixture map carries, and
// structuresMap is a 4x3 map placing three type-4 records under it.
//
// It places records and NO object grid, which is what keeps the two layers'
// assertions apart: a structure comes from a record list and an object from a
// per-cell byte, and a fixture carrying both could not say which layer answered.
const structuresMapKey = structGood

func structuresMap() []byte {
	return synth.ALM(synth.ALMOptions{
		Width: 4, Height: 3, Name: "Structures",
		Objects: []synth.ALMObject{
			{X: 0x0000, Y: 0x0000, Kind: structuresMapKey},
			{X: 0x0200, Y: 0x0100, Kind: structuresMapKey},
			{X: 0x0300, Y: 0x0200, Kind: structuresMapKey},
		},
	})
}

// AC-9's three no-bundle shapes, at the LOAD PATH: a map loaded with no
// structure layer at all, one with an explicitly nil bundle, and one with a
// bundle no placement resolves in. All three place no entry and report a zero
// census, with no install and no window.
func TestLoadMapViewerWithoutAStructureBundle(t *testing.T) {
	tiles := &terrain.Tileset{}
	for _, tc := range []struct {
		name  string
		layer game.StructureLayer
	}{
		{"no layer at all", game.StructureLayer{}},
		{"an explicitly nil bundle with art on", game.StructureLayer{Set: nil, Art: true}},
		{"a bundle no key resolves in", game.StructureLayer{Set: new(terrain.StructureSet), Art: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mv, err := game.LoadMapViewer(tiles, structuresMap(), "x", game.Markers{}, game.StaticLayer{}, tc.layer)
			if err != nil {
				t.Fatalf("LoadMapViewer: %v", err)
			}
			entries, counts := mv.Viewer.Structures()
			if entries != 0 {
				t.Errorf("%d entries, want none", entries)
			}
			if counts.Placements.Drawn != 0 || counts.Cells.Frames != 0 {
				t.Errorf("census = %+v, want nothing drawn", counts)
			}
		})
	}
}

// The type-4 records reach the viewer through the Grid UNCONDITIONALLY: with a
// bundle the map's own placements draw, and the count is the map's records
// rather than anything the caller passed.
func TestLoadMapViewerPlacesTheMapsOwnRecords(t *testing.T) {
	set := new(terrain.StructureSet)
	set.Classes[structuresMapKey] = &terrain.StructureClass{
		TileWidth: 1, TileHeight: 1, FullHeight: 1,
		Frames: []*terrain.StaticFrame{{Width: terrain.CellSize, Height: terrain.CellSize}},
	}
	mv, err := game.LoadMapViewer(&terrain.Tileset{}, structuresMap(), "x", game.Markers{}, game.StaticLayer{},
		game.StructureLayer{Set: set, Art: true})
	if err != nil {
		t.Fatalf("LoadMapViewer: %v", err)
	}
	entries, counts := mv.Viewer.Structures()
	if want := len(mv.Map.Objects); counts.Placements.Drawn != want {
		t.Errorf("%d drawn, want the map's %d type-4 records", counts.Placements.Drawn, want)
	}
	// A 1x1 class of FullHeight 1 is one frame per record, so entries and drawn
	// placements coincide here — which is the ONLY shape they do coincide in.
	if entries != counts.Placements.Drawn {
		t.Errorf("%d entries against %d drawn; a 1x1x1 class is one frame per placement",
			entries, counts.Placements.Drawn)
	}
}

// --- the animation gate (AC-2), and the omitted-key indifference ---

// structAnimReg writes a one-class registry carrying the given animation keys
// over a 2x1 rectangle of FullHeight 3 — six grid cells, so a mask of length 6
// is the grid's own and one of length 2 is the RECTANGLE's, which is the rival
// reading the contract refuses.
func structAnimReg(keys ...synth.RegNode) []byte {
	return synth.StructuresReg(structSection(1, `houses\hut`, 2, 1, 3, keys...))
}

func structAnimSet(t *testing.T, keys ...synth.RegNode) *terrain.StructureClass {
	t.Helper()
	set, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
		{Path: graphicsEntry(t, game.StructureRegistry), Data: structAnimReg(keys...)},
		{Path: "structures/houses/hut.256", Data: structSquare(24)},
	})))
	if err != nil {
		t.Fatalf("LoadStructures: %v", err)
	}
	if set.Classes[1] == nil {
		t.Fatal("the fixture class did not load")
	}
	return set.Classes[1]
}

func structInts(name string, v ...int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x06, Ints: v}
}

// AC-2. The four precondition shapes: only the one meeting all three carries a
// non-empty timeline, and the wrong-length mask is refused WHOLE.
func TestStructureAnimationGate(t *testing.T) {
	// The pair (2, 1) over the values (1, 2) expands to [1 1 2], period 3.
	timeKeys := []synth.RegNode{structInts("AnimTime", 2, 1), structInts("AnimFrame", 1, 2)}
	gridMask := structStr("AnimMask", "x-x-x-") // TileWidth * FullHeight = 6
	rectMask := structStr("AnimMask", "x-")     // TileWidth * TileHeight = 2

	t.Run("Phases 1 with a mask", func(t *testing.T) {
		c := structAnimSet(t, append([]synth.RegNode{structInt("Phases", 1), gridMask}, timeKeys...)...)
		if len(c.Timeline) != 0 {
			t.Errorf("timeline = %v, want none — one phase is not a cycle", c.Timeline)
		}
	})

	t.Run("a mask of the rectangle's length is refused one tier down", func(t *testing.T) {
		// THE REGISTRY ITSELF IS REFUSED, and this is the one place the tree is
		// stricter than 0054's own contract: pkg/data's validation carries the
		// identical TileWidth * FullHeight rule and makes a mask of any other
		// length a LOAD ERROR, so no bundle is produced at all and the gate in
		// LoadStructures never sees such a class.
		//
		// Which is a stronger form of "refused whole", not a weaker one: nothing
		// truncates the mask, pads it or indexes it, because nothing gets to it.
		// The gate's own arm is exercised directly in structureanim_test.go, over
		// a resolved class built there, since this door cannot deliver one.
		_, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
			{Path: graphicsEntry(t, game.StructureRegistry), Data: structAnimReg(
				append([]synth.RegNode{structInt("Phases", 3), rectMask}, timeKeys...)...)},
			{Path: "structures/houses/hut.256", Data: structSquare(24)},
		})))
		if err == nil {
			t.Fatal("a registry whose AnimMask is the RECTANGLE's length loaded without error")
		}
		if !strings.Contains(err.Error(), "AnimMask") {
			t.Errorf("err = %v, want one naming AnimMask", err)
		}
	})

	t.Run("no mask at all", func(t *testing.T) {
		c := structAnimSet(t, append([]synth.RegNode{structInt("Phases", 3)}, timeKeys...)...)
		if len(c.Timeline) != 0 {
			t.Errorf("timeline = %v, want none — a class with no mask does not animate", c.Timeline)
		}
	})

	t.Run("all three present", func(t *testing.T) {
		c := structAnimSet(t, append([]synth.RegNode{structInt("Phases", 3), gridMask}, timeKeys...)...)
		if want := []int{1, 1, 2}; len(c.Timeline) != len(want) {
			t.Fatalf("timeline = %v, want %v", c.Timeline, want)
		} else {
			for i := range want {
				if c.Timeline[i] != want[i] {
					t.Fatalf("timeline = %v, want %v", c.Timeline, want)
				}
			}
		}
		// The mask read once: '-' retires a cell, and the rank is EXCLUSIVE, so
		// the first live cell ranks 0.
		wantRank := []int{0, -1, 1, -1, 2, -1}
		if len(c.Rank) != len(wantRank) {
			t.Fatalf("rank = %v, want %v", c.Rank, wantRank)
		}
		for i := range wantRank {
			if c.Rank[i] != wantRank[i] {
				t.Fatalf("rank = %v, want %v", c.Rank, wantRank)
			}
		}
		if c.Live != 3 {
			t.Errorf("live = %d, want 3", c.Live)
		}
	})

	t.Run("Phases above 1 with no animation keys at all", func(t *testing.T) {
		// Ten shipped classes spell Phases > 1 and then spell no timeline, so the
		// scalar alone must not open the gate.
		c := structAnimSet(t, structInt("Phases", 5))
		if len(c.Timeline) != 0 {
			t.Errorf("timeline = %v, want none", c.Timeline)
		}
	})
}

func TestStructureContractIsIndifferentToTheOmittedKeyDefault(t *testing.T) {
	// Every scalar key the loader reads, with the value the registry's own
	// default resolves it to.
	for _, tc := range []struct {
		key     string
		decoded int32
		// withAnim spells the three animation keys beside the rest. It is off for
		// the two extents the mask's length is measured against, because pkg/data
		// refuses a registry whose mask does not match TileWidth * FullHeight and
		// an omitted extent makes that product 0 — so those two rows would test
		// the registry's own refusal a second time instead of this contract's
		// indifference.
		withAnim bool
	}{
		{"ID", -1, true},
		{"TileWidth", -1, false},
		{"TileHeight", -1, true},
		{"FullHeight", -1, false},
		{"Phases", -1, true},
		{"Indestructible", 0, true},
		{"Flat", 0, true},
		{"VariableSize", 0, true},
	} {
		t.Run(tc.key, func(t *testing.T) {
			// A class spelling every key EXCEPT this one, and the same class
			// spelling it at the registry's own default.
			// spell false leaves the key OUT, which is what the tree resolves
			// to the Go zero today; spell true writes it at the registry's own
			// decoded default. Every other key is spelled either way, so the two
			// registries differ in exactly one key.
			base := func(omit string, spell bool, v int32) []synth.RegNode {
				keys := []synth.RegNode{
					structInt("ID", 1), structStr("File", `houses\hut`),
					structInt("TileWidth", 2), structInt("TileHeight", 1), structInt("FullHeight", 3),
					structInt("Phases", 3), structInt("Indestructible", 0),
					structInt("Flat", 0), structInt("VariableSize", 0),
				}
				if tc.withAnim {
					keys = append(keys,
						structStr("AnimMask", "x-x-x-"),
						structInts("AnimTime", 2, 1), structInts("AnimFrame", 1, 2))
				}
				out := make([]synth.RegNode, 0, len(keys))
				for _, k := range keys {
					if k.Name == omit {
						if spell {
							out = append(out, structInt(omit, v))
						}
						continue
					}
					out = append(out, k)
				}
				return out
			}

			load := func(keys []synth.RegNode) (*terrain.StructureSet, []terrain.StructurePlacement, terrain.StructureCounts) {
				t.Helper()
				set, err := game.LoadStructures(openContainers(t, synth.Archive([]synth.File{
					{Path: graphicsEntry(t, game.StructureRegistry), Data: synth.StructuresReg(keys)},
					{Path: "structures/houses/hut.256", Data: structSquare(24)},
				})))
				if err != nil {
					t.Fatalf("LoadStructures: %v", err)
				}
				g := terrain.Grid{Width: 8, Height: 8, Structures: []terrain.StructureRecord{
					{X: 1 << 8, Y: 1 << 8, Key: 1},
				}}
				places, counts, _ := terrain.StructurePlacements(g, set, nil, 0)
				return set, places, counts
			}

			// "Omitted": the key is absent, and the tree resolves it to the Go
			// zero. "Decoded": the key is spelled at the registry's own default.
			omittedSet, omittedList, omittedCounts := load(base(tc.key, false, 0))
			decodedSet, decodedList, decodedCounts := load(base(tc.key, true, tc.decoded))

			if !reflect.DeepEqual(omittedSet, decodedSet) {
				t.Fatalf("%s: the bundle differs between the two candidate values of an omitted key", tc.key)
			}
			if !reflect.DeepEqual(omittedList, decodedList) {
				t.Fatalf("%s: the placement list differs between the two candidate values", tc.key)
			}
			if omittedCounts != decodedCounts {
				t.Fatalf("%s: the census differs: %+v against %+v", tc.key, omittedCounts, decodedCounts)
			}
		})
	}
}
