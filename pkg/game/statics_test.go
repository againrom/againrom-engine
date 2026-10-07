package game_test

import (
	"errors"
	"image/color"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
	"againrom/pkg/vfs"
)

// openContainers lays a synthetic archive down as graphics.res in the test's own
// temporary tree and opens the container filesystem over it.
//
// It is a HOST FILE and not an in-memory archive because an address begins with
// the container's identity and an identity comes from a host filename: an archive
// built from bytes has no name for one to be derived from. So this is the route
// the front-end takes, exactly — the filesystem consumes the identity segment and
// the archive resolves the remainder — rather than an in-memory stand-in that
// would have to be handed pre-addressed entries no container ever holds.
//
// The bytes are still the suite's own synthetic ones and no game file is read.
func openContainers(t *testing.T, b []byte) *vfs.FS {
	t.Helper()
	path := filepath.Join(t.TempDir(), game.GraphicsArchive)
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	f, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%s): %v", path, err)
	}
	return f
}

// graphicsEntry turns an address into the entry path inside graphics.res: the
// container's identity segment, which every address the loaders resolve now
// carries, stripped off the front.
//
// A container holds container-relative entries, so that is what a fixture
// writes; the loader resolves addresses. The two differ by exactly that
// segment, and this DERIVES it from the archive name rather than writing a
// second prefix literal — a literal could drift until the fixture keyed
// its entries under a prefix nothing resolves, and the suite would still be
// green.
func graphicsEntry(t *testing.T, address string) string {
	t.Helper()
	identity, err := vfs.Identity(game.GraphicsArchive)
	if err != nil {
		t.Fatalf("vfs.Identity(%q): %v", game.GraphicsArchive, err)
	}
	rel := strings.TrimPrefix(address, identity+"/")
	if rel == address {
		t.Fatalf("address %q carries no %q identity segment", address, identity)
	}
	return rel
}

// The placement bytes the fixture registry answers. A byte is a class's ID plus
// one; staticNoClass is the hole left by the ID no section carries, so a test can
// name the byte that resolves to nothing without recomputing the offset.
const (
	staticGood      = 1  // ID 0: oak, Index 1 — the 4x3 frame
	staticNoPalette = 2  // ID 1: a palette-less sheet
	staticUndecoded = 3  // ID 2: a sheet spr256 refuses
	staticFarIndex  = 4  // ID 3: oak, Index 2 — exactly one past its last frame
	staticNoSheet   = 5  // ID 4: a sheet the archive does not hold
	staticNoClass   = 6  // ID 5: no section carries it
	staticEmpty     = 7  // ID 6: a frame of no area — drawable all the same
	staticFirst     = 8  // ID 7: oak, Index 0 — the 1x1 frame of the SAME sheet
	staticNoFile    = 9  // ID 8: no File key at all
	staticNoIndex   = 10 // ID 9: oak, no Index key — so Index resolves to -1
)

// staticPalette is the fixture sheets' palette. Entries 0, 1 and 7 have pairwise
// distinct R, G and B, so a channel swap anywhere on the path — synth's BGR
// block, spr256's reorder, this loader's copy — moves a colour that is asserted.
//
// Entry 0 is a colour like any other. The shipped palettes reserve index 0 as
// their transparent key, but transparency here is structural, so an opaque pixel
// carrying index 0 must come out painted in this entry rather than as a hole.
func staticPalette() []color.RGBA {
	pal := make([]color.RGBA, 8)
	pal[0] = color.RGBA{R: 0x11, G: 0x22, B: 0x33}
	pal[1] = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
	pal[7] = color.RGBA{R: 0x01, G: 0x02, B: 0x04}
	return pal
}

// oakSheet is the sheet two fixture classes share, its two frames DIFFERENTLY
// SIZED: a 1x1 at index 0 and a 4x3 at index 1.
//
// The size difference is the point (SPR256-FRAME-023). A loader that took the
// sheet's first frame regardless of Index, or that carried frame 0's dimensions
// onto whichever frame it drew, agrees with a same-size sheet on every
// assertion; against this one it hands back a 1x1 where a 4x3 is due.
//
// Frame 1 mixes all three pixel kinds: a hole, an opaque index 7, an opaque
// index 0, and a wholly transparent row.
func oakSheet() []byte {
	const op = true
	return synth.Sheet256(synth.Sheet256Options{
		Palette: staticPalette(),
		Frames: []synth.Frame256{
			{Width: 1, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: op}}},
			{Width: 4, Height: 3, Pixels: []synth.Pixel256{
				{}, {Index: 7, Opaque: op}, {Index: 0, Opaque: op}, {},
				{}, {}, {}, {},
				{Index: 1, Opaque: op}, {Index: 1, Opaque: op}, {Index: 1, Opaque: op}, {Index: 1, Opaque: op},
			}},
		},
	})
}

// staticFiles is the registry's [Files] table. Index 3 names art the archive
// deliberately does not hold — the absent-sheet exclusion needs no builder,
// only an omission.
var staticFiles = []string{`trees\oak`, `flat\nopal`, `broken\bad`, `missing\gone`, `empty\void`}

// staticRegistry writes the fixture objects/objects.reg: nine dense sections
// whose IDs skip 5, so exactly one placement byte in the range names no loaded
// class.
//
// Two of the sections OMIT A KEY rather than setting a bad one, because
// pkg/data's absent-everywhere default for both is -1 and that is the only way
// to reach a negative File or Index at all: the loader's own lower-bound guard
// on Index is unreachable from any section that writes the key.
func staticRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	ints := func(name string, v ...int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x06, Ints: v}
	}
	class := func(id, file, index, w, h, cx, cy int32) []synth.RegNode {
		return []synth.RegNode{
			i("ID", id), i("File", file), i("Index", index),
			i("Width", w), i("Height", h), i("CenterX", cx), i("CenterY", cy),
		}
	}
	return synth.ObjectsReg(staticFiles,
		// The good one, and the ONE fixture class carrying a cycle: the pair
		// (2, 1) over the values (0, 1) expands to [0 0 1], period 3. Both keys
		// are written, because the loader validates a pair to one length and an
		// animation key without its partner is a load error rather than a class
		// with no cycle.
		append(class(0, 0, 1, 64, 80, 30, 70),
			ints("AnimationTime", 2, 1), ints("AnimationFrame", 0, 1), i("FireObject", -2)),
		class(1, 1, 0, 32, 32, 16, 24), // palette-less sheet
		class(2, 2, 0, 32, 32, 16, 24), // undecodable sheet
		class(3, 0, 2, 32, 32, 16, 24), // Index exactly one past the last frame
		class(4, 3, 0, 32, 32, 16, 24), // sheet absent from the archive
		class(6, 4, 0, 16, 16, 8, 8),   // a frame of no area
		class(7, 0, 0, 8, 8, 4, 4),     // the same sheet as ID 0, Index 0
		// No File key at all: pkg/data resolves no sprite base, so the class
		// names no art rather than naming art that is missing.
		[]synth.RegNode{i("ID", 8), i("Index", 0), i("Width", 8), i("Height", 8)},
		// No Index key: it resolves to -1 against a sheet that decodes perfectly
		// well, which is the one fixture that reaches the lower bound.
		[]synth.RegNode{i("ID", 9), i("File", 0), i("Width", 8), i("Height", 8)},
	)
}

// staticArchiveFiles is the fixture graphics archive's contents: the registry,
// and every sheet the [Files] table names EXCEPT missing\gone.
func staticArchiveFiles(t *testing.T) []synth.File {
	t.Helper()
	return []synth.File{
		{Path: graphicsEntry(t, game.ObjectRegistry), Data: staticRegistry()},
		{Path: "objects/trees/oak.256", Data: oakSheet()},
		// The palette-less variant: its frames decode, only the palette is gone.
		{Path: "objects/flat/nopal.256", Data: synth.Sheet256(synth.Sheet256Options{
			NoPalette: true,
			Frames:    []synth.Frame256{{Width: 2, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}}}},
		})},
		// A frame record whose block is 4096 bytes long and one byte wide: the
		// stream cannot be walked at all, so spr256 refuses it.
		{Path: "objects/broken/bad.256", Data: synth.Sheet256Raw(
			synth.Palette256(staticPalette()),
			[]synth.Sheet256RawFrame{{Width: 8, Height: 8, DataSize: 4096, Data: []byte{0x88}}},
			0x80000000|1,
		)},
		// A single frame of no area. It decodes, it carries a palette, and its
		// Index is in range — so it IS drawable under the contract's rule, and a
		// loader that refused it would be inventing a fifth exclusion.
		{Path: "objects/empty/void.256", Data: synth.Sheet256(synth.Sheet256Options{
			Palette: staticPalette(),
			Frames:  []synth.Frame256{{Width: 0, Height: 0}},
		})},
	}
}

// loadFixtureStatics loads the bundle out of the fixture archive.
func loadFixtureStatics(t *testing.T) *terrain.StaticSet {
	t.Helper()
	set, err := game.LoadStatics(openContainers(t, synth.Archive(staticArchiveFiles(t))))
	if err != nil {
		t.Fatalf("LoadStatics: %v", err)
	}
	if set == nil {
		t.Fatal("LoadStatics returned a nil set with no error")
	}
	return set
}

func TestLoadStatics(t *testing.T) {
	t.Run("a class is keyed by its placement byte, not by its ID", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// Byte 0 is *no object* and never a class — not even the class whose ID
		// is 0, which byte 1 names.
		if set.Classes[0] != nil {
			t.Errorf("Classes[0] is non-nil; byte 0 is *no object*, not the class whose ID is 0")
		}
		c := set.Classes[staticGood]
		if c == nil {
			t.Fatalf("Classes[%d] is nil; byte b must name the class whose ID is b-1", staticGood)
		}
		if c.Width != 64 || c.Height != 80 || c.CenterX != 30 || c.CenterY != 70 {
			t.Errorf("canvas = %dx%d centre (%d, %d), want 64x80 centre (30, 70)",
				c.Width, c.Height, c.CenterX, c.CenterY)
		}
		if c.FireObject != -2 {
			t.Errorf("FireObject = %d, want the registry's compared value -2", c.FireObject)
		}
	})

	t.Run("a byte naming no loaded class leaves no class at all", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// The distinction this asserts is the one the corpus census reports: a
		// NIL CLASS is map data pointing past the registry, a class with a nil
		// FRAME is art we could not decode. Collapsing them destroys the only
		// fact AC-8 records.
		if set.Classes[staticNoClass] != nil {
			t.Errorf("Classes[%d] is non-nil; no section carries ID %d",
				staticNoClass, staticNoClass-1)
		}
		if c := set.Classes[staticNoSheet]; c == nil || c.Frame != nil {
			t.Errorf("Classes[%d] = %v; a class whose art is absent must be LOADED and frameless, "+
				"not missing", staticNoSheet, c)
		}
		// Nothing outside the fixture's own bytes is loaded, so the rest of the
		// 256-entry table stays empty.
		for b := staticNoIndex + 1; b <= 0xff; b++ {
			if set.Classes[b] != nil {
				t.Fatalf("Classes[%d] is non-nil; no class carries ID %d", b, b-1)
			}
		}
	})

	t.Run("the drawn frame is the one Index selects, at its own size", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// Two classes, one sheet, different Index — and the frames are
		// differently sized, so taking the sheet's first frame regardless, or
		// carrying frame 0's dimensions onto frame 1, is visible here and
		// nowhere else in this file.
		wide := set.Classes[staticGood].Frame
		if wide == nil {
			t.Fatalf("Classes[%d].Frame is nil, want the sheet's 4x3 frame", staticGood)
		}
		if wide.Width != 4 || wide.Height != 3 {
			t.Errorf("Index 1 selected a %dx%d frame, want 4x3", wide.Width, wide.Height)
		}
		if got := len(wide.Pixels); got != 12 {
			t.Errorf("the 4x3 frame holds %d pixels, want 12", got)
		}

		small := set.Classes[staticFirst].Frame
		if small == nil {
			t.Fatalf("Classes[%d].Frame is nil, want the same sheet's 1x1 frame", staticFirst)
		}
		if small.Width != 1 || small.Height != 1 {
			t.Errorf("Index 0 of the same sheet selected a %dx%d frame, want 1x1",
				small.Width, small.Height)
		}
	})

	t.Run("pixels and palette come across, transparency structurally", func(t *testing.T) {
		f := loadFixtureStatics(t).Classes[staticGood].Frame

		for _, tc := range []struct {
			label  string
			at     int
			opaque bool
			index  uint8
		}{
			{"a hole", 0, false, 0},
			{"an opaque index 7", 1, true, 7},
			{"an OPAQUE INDEX 0 — not a hole", 2, true, 0},
			{"a blank row", 4, false, 0},
			{"an opaque index 1", 8, true, 1},
		} {
			px := f.Pixels[tc.at]
			if px.Opaque != tc.opaque || (tc.opaque && px.Index != tc.index) {
				t.Errorf("pixel %d (%s) = %+v, want Opaque=%t Index=%d",
					tc.at, tc.label, px, tc.opaque, tc.index)
			}
		}

		// The palette is the SHEET's, at full opacity. The format's fourth byte
		// is reserved and not an alpha channel, so an entry a loader left at
		// alpha 0 would paint invisible pixels over the terrain.
		want := staticPalette()
		for _, i := range []int{0, 1, 7} {
			got := f.Palette[i]
			if got != (color.RGBA{R: want[i].R, G: want[i].G, B: want[i].B, A: 0xff}) {
				t.Errorf("Palette[%d] = %+v, want %+v at alpha 0xff", i, got, want[i])
			}
		}
		if f.Palette[200].A != 0xff {
			t.Errorf("Palette[200].A = %#x; an entry the sheet left black is still opaque", f.Palette[200].A)
		}
	})

	t.Run("each exclusion leaves its class artless and counted, never an error", func(t *testing.T) {
		set := loadFixtureStatics(t)

		for _, tc := range []struct {
			code  byte
			label string
		}{
			{staticNoPalette, "a palette-less sheet"},
			{staticUndecoded, "a sheet spr256 refuses"},
			{staticFarIndex, "an Index one past the sheet's last frame"},
			{staticNoSheet, "a sheet the archive does not hold"},
			{staticNoFile, "a class naming no art at all"},
			{staticNoIndex, "an Index that resolves to -1"},
		} {
			c := set.Classes[tc.code]
			if c == nil {
				t.Errorf("byte %d (%s): the class is missing; an exclusion is a class with no FRAME",
					tc.code, tc.label)
				continue
			}
			if c.Frame != nil {
				t.Errorf("byte %d (%s): Frame is non-nil, want nil", tc.code, tc.label)
			}
			// The canvas still loads: the exclusion is about the art, and a
			// class whose geometry vanished with its sheet could not be told
			// from one the registry never described.
			if c.Width == 0 && c.Height == 0 && c.CenterX == 0 && c.CenterY == 0 {
				t.Errorf("byte %d (%s): the canvas came across as all zeroes", tc.code, tc.label)
			}
		}
	})

	t.Run("a frame of no area is drawable", func(t *testing.T) {
		// A non-nil Frame is the WHOLE drawability rule. This frame decodes,
		// carries a palette and is in range, so it loads; the blit draws nothing
		// for it and the builder still places it. Refusing it here would move the
		// decision to where the census can no longer say why the cell drew
		// nothing.
		f := loadFixtureStatics(t).Classes[staticEmpty].Frame
		if f == nil {
			t.Fatalf("Classes[%d].Frame is nil; a frame of no area is not one of the four exclusions",
				staticEmpty)
		}
		if f.Width != 0 || f.Height != 0 || len(f.Pixels) != 0 {
			t.Errorf("frame = %dx%d with %d pixels, want 0x0 with none",
				f.Width, f.Height, len(f.Pixels))
		}
	})

	t.Run("the census counts the two skip kinds apart", func(t *testing.T) {
		// The loader's exclusions, read back through the placement builder —
		// which is the only consumer that can tell a nil class from a frameless
		// one, and the one the corpus criterion reports through.
		set := loadFixtureStatics(t)
		overlay := []uint8{
			0, staticGood, staticNoPalette, staticUndecoded, staticFarIndex,
			staticNoSheet, staticNoClass, staticEmpty, staticFirst, staticNoFile,
			staticNoIndex,
		}
		g := terrain.Grid{Width: len(overlay), Height: 1, Overlay: overlay}

		places, counts, _ := terrain.StaticPlacements(g, set, nil, 0, terrain.AnimGateTiles)
		want := terrain.StaticCounts{Placed: 3, NoClass: 1, NoFrame: 6}
		if counts != want {
			t.Errorf("counts = %+v, want %+v", counts, want)
		}
		if len(places) != want.Placed {
			t.Errorf("%d placements, want %d", len(places), want.Placed)
		}
		for _, p := range places {
			if p.Frame == nil {
				t.Errorf("placement at %v carries a nil frame", p.Cell)
			}
		}
	})

	t.Run("an unreadable registry is the error, and the only one", func(t *testing.T) {
		for _, tc := range []struct {
			label    string
			files    []synth.File
			notExist bool
		}{
			{
				label: "no objects/objects.reg in the archive",
				files: []synth.File{{Path: "objects/trees/oak.256", Data: oakSheet()}},
				// fs.ErrNotExist has to survive the return: cmd/againrom's -check
				// is specified to fail when the registry cannot be READ (AC-7).
				notExist: true,
			},
			{
				label: "a registry that is not a registry",
				files: []synth.File{{Path: graphicsEntry(t, game.ObjectRegistry), Data: []byte("not a .reg stream at all")}},
			},
			{
				label: "a registry that parses but does not resolve",
				// [Global] promises one class and no section carries it.
				files: []synth.File{{Path: graphicsEntry(t, game.ObjectRegistry), Data: synth.Reg(0x11, []synth.RegNode{
					{Name: "Global", Kind: 0x01, Children: []synth.RegNode{
						{Name: "ObjectCount", Kind: 0x02, Int: 1},
						{Name: "FileCount", Kind: 0x02, Int: 0},
					}},
				})}},
			},
		} {
			t.Run(tc.label, func(t *testing.T) {
				set, err := game.LoadStatics(openContainers(t, synth.Archive(tc.files)))
				if err == nil {
					t.Fatalf("LoadStatics succeeded")
				}
				if set != nil {
					t.Errorf("LoadStatics returned a non-nil set alongside an error")
				}
				if !strings.Contains(err.Error(), game.ObjectRegistry) {
					t.Errorf("error %q does not name %q", err, game.ObjectRegistry)
				}
				if tc.notExist && !errors.Is(err, fs.ErrNotExist) {
					t.Errorf("error %v does not satisfy errors.Is(err, fs.ErrNotExist)", err)
				}
			})
		}
	})

	t.Run("a nil archive is an error, not a panic", func(t *testing.T) {
		set, err := game.LoadStatics(nil)
		if err == nil {
			t.Fatalf("LoadStatics(nil) succeeded")
		}
		if set != nil {
			t.Errorf("LoadStatics(nil) returned a non-nil set alongside an error")
		}
	})
}

func TestOpenGraphics(t *testing.T) {
	t.Run("one open yields both the filesystem and the tileset", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, game.GraphicsArchive)
		if err := os.WriteFile(path, synth.Archive(staticArchiveFiles(t)), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}

		containers, tiles, err := game.OpenGraphics(path)
		if err != nil {
			t.Fatalf("OpenGraphics: %v", err)
		}
		if containers == nil || tiles == nil {
			t.Fatalf("OpenGraphics: containers=%v tiles=%v, want both non-nil", containers != nil, tiles != nil)
		}
		// The fixture holds no terrain strips, so nothing is loaded — a tileset
		// with absent slots draws placeholders rather than failing.
		if tiles.Loaded != 0 {
			t.Errorf("Loaded = %d on an archive with no tile strips, want 0", tiles.Loaded)
		}

		// The bundle comes off THE FILESYSTEM THIS CALL RETURNED, which is what
		// the developer front-ends now do: the tileset and the object art are both
		// addressed by container, and one open over one file serves both — the
		// transitional second read of the same host is gone with the archive that
		// was its only reason.
		set, err := game.LoadStatics(containers)
		if err != nil {
			t.Fatalf("LoadStatics off the same host path: %v", err)
		}
		if set.Classes[staticGood] == nil || set.Classes[staticGood].Frame == nil {
			t.Errorf("the bundle loaded off that path's container filesystem has no art for byte %d", staticGood)
		}
	})

	t.Run("the frozen wording is one string, shared with OpenContainers", func(t *testing.T) {
		// `open <path>: <err>` is contract. OpenGraphics is written OVER
		// OpenContainers, so the two cannot drift — asserted by equality rather
		// than by two copies of the expected text. OpenTileset used to be the
		// third string this compared; the teardown removed it, so the wording now
		// has one fewer place it could drift in rather than one more.
		path := filepath.Join(t.TempDir(), "absent.res")

		containers, tiles, err := game.OpenGraphics(path)
		if err == nil {
			t.Fatalf("OpenGraphics succeeded on a missing archive")
		}
		if containers != nil || tiles != nil {
			t.Errorf("OpenGraphics returned containers=%v tiles=%v alongside an error",
				containers != nil, tiles != nil)
		}
		if !strings.HasPrefix(err.Error(), "open ") || !strings.Contains(err.Error(), path) {
			t.Errorf("error %q is not `open <path>: <err>` for %q", err, path)
		}

		bare, bareErr := game.OpenContainers(path)
		if bareErr == nil {
			t.Fatalf("OpenContainers succeeded on a missing archive")
		}
		if bare != nil {
			t.Errorf("OpenContainers returned a non-nil filesystem alongside an error")
		}
		if bareErr.Error() != err.Error() {
			t.Errorf("OpenContainers error %q != OpenGraphics error %q", bareErr, err)
		}
	})
}

// TestLoadStaticsCarriesTheWholeSheet is 0031 AC-6 and the loader invariant T3
// pins: a drawing class carries every frame of its sheet, two classes naming one
// sheet receive ONE converted slice, and Frame is non-nil exactly when Frames is
// non-nil and Index lies inside it.
//
// The fixture archive is the same one the 0017 tests use, so what changes here
// is what the loader now carries and not what it accepts: the four exclusions,
// the two artless answers and the census that counts them apart are asserted
// unmoved above.
func TestLoadStaticsCarriesTheWholeSheet(t *testing.T) {
	t.Run("two classes naming one sheet share it pointer for pointer", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// staticGood is oak at Index 1, staticFirst is oak at Index 0 — one
		// sheet, two classes, two different drawn frames. The memo is per PATH,
		// so the converted slice must be shared: the frame-keyed texture cache
		// downstream uploads a sheet once because these pointers are equal.
		a, b := set.Classes[staticGood], set.Classes[staticFirst]
		if a == nil || b == nil {
			t.Fatalf("Classes[%d]=%v Classes[%d]=%v, want both loaded", staticGood, a, staticFirst, b)
		}
		if len(a.Frames) != 2 || len(b.Frames) != 2 {
			t.Fatalf("Frames lengths %d and %d, want 2 each — the oak sheet's whole frame count",
				len(a.Frames), len(b.Frames))
		}
		for i := range a.Frames {
			if a.Frames[i] != b.Frames[i] {
				t.Errorf("Frames[%d] differs between two classes of one sheet: %p vs %p",
					i, a.Frames[i], b.Frames[i])
			}
		}

		// Every frame of the sheet is drawable, at its OWN size — which is the
		// whole reason the sheet has to be carried: a cycle stands frames of
		// different sizes on one ground point.
		if a.Frames[0].Width != 1 || a.Frames[0].Height != 1 {
			t.Errorf("Frames[0] = %dx%d, want 1x1", a.Frames[0].Width, a.Frames[0].Height)
		}
		if a.Frames[1].Width != 4 || a.Frames[1].Height != 3 {
			t.Errorf("Frames[1] = %dx%d, want 4x3", a.Frames[1].Width, a.Frames[1].Height)
		}
	})

	t.Run("Index rides across and selects inside Frames", func(t *testing.T) {
		set := loadFixtureStatics(t)

		for _, tc := range []struct {
			code byte
			want int
		}{
			{staticGood, 1},
			{staticFirst, 0},
			{staticFarIndex, 2},  // one past the sheet's last frame
			{staticNoIndex, -1},  // the key is absent, so it resolves to -1
			{staticNoPalette, 0}, // in range, but the sheet excludes itself
		} {
			c := set.Classes[tc.code]
			if c == nil {
				t.Fatalf("Classes[%d] is nil", tc.code)
			}
			if c.Index != tc.want {
				t.Errorf("Classes[%d].Index = %d, want %d", tc.code, c.Index, tc.want)
			}
		}

		// The drawn frame IS the one Index names inside the carried sheet. The
		// two are produced by two calls, so this is the assertion that they
		// agree about which frame a class draws rather than only about whether
		// it draws one.
		for _, code := range []byte{staticGood, staticFirst} {
			c := set.Classes[code]
			got := c.Frames[c.Index]
			if got.Width != c.Frame.Width || got.Height != c.Frame.Height {
				t.Errorf("Classes[%d]: Frames[Index] is %dx%d, Frame is %dx%d",
					code, got.Width, got.Height, c.Frame.Width, c.Frame.Height)
			}
		}
	})

	t.Run("Frame is non-nil exactly when Frames holds Index", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// Every class the fixture loads, the four exclusions among them, over
		// the invariant stated as an EQUIVALENCE rather than as two one-way
		// checks: an implication in one direction alone is satisfied by a
		// loader that never fills Frames at all.
		for b := 1; b <= 0xff; b++ {
			c := set.Classes[b]
			if c == nil {
				continue
			}
			inside := c.Frames != nil && c.Index >= 0 && c.Index < len(c.Frames)
			if (c.Frame != nil) != inside {
				t.Errorf("Classes[%d]: Frame non-nil = %t but Frames holds Index = %t (Index %d, %d frames)",
					b, c.Frame != nil, inside, c.Index, len(c.Frames))
			}
		}

		// The two whole-sheet exclusions leave NO sheet at all, which is
		// stronger than leaving no drawn frame: every frame of a palette-less
		// or undecodable sheet selects in a palette the format did not deliver.
		for _, tc := range []struct {
			code  byte
			label string
		}{
			{staticNoPalette, "a palette-less sheet"},
			{staticUndecoded, "a sheet spr256 refuses"},
			{staticNoSheet, "a sheet the archive does not hold"},
			{staticNoFile, "a class naming no art at all"},
		} {
			if c := set.Classes[tc.code]; c == nil || c.Frames != nil {
				t.Errorf("byte %d (%s): Frames = %v, want nil", tc.code, tc.label, c.Frames)
			}
		}

		// The range exclusion is the other kind: the SHEET is fine and is
		// carried, and only the class's own selection falls outside it. A
		// loader that dropped the sheet with the frame would make this class
		// indistinguishable from a palette-less one.
		if c := set.Classes[staticFarIndex]; c == nil || c.Frame != nil || len(c.Frames) != 2 {
			t.Errorf("byte %d: Frame=%v Frames=%d, want a nil frame beside the sheet's 2 frames",
				staticFarIndex, c.Frame, len(c.Frames))
		}
	})

	t.Run("the class's timeline comes across expanded", func(t *testing.T) {
		set := loadFixtureStatics(t)

		// The one fixture class carrying a pair: AnimationTime (2, 1) over
		// AnimationFrame (0, 1) expands to [0 0 1], written out by hand here
		// rather than taken from the expansion under test.
		if got, want := set.Classes[staticGood].Timeline, []int{0, 0, 1}; !slices.Equal(got, want) {
			t.Errorf("Classes[%d].Timeline = %v, want %v", staticGood, got, want)
		}

		// Every other class carries no pair, so it has NO cycle — the period is
		// the timeline's length and nothing else answers for it.
		for b := 1; b <= 0xff; b++ {
			if b == staticGood {
				continue
			}
			if c := set.Classes[b]; c != nil && len(c.Timeline) != 0 {
				t.Errorf("Classes[%d].Timeline = %v, want empty — the class carries no pair",
					b, c.Timeline)
			}
		}
	})
}
