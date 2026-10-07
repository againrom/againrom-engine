package game_test

import (
	"errors"
	"image/color"
	"io/fs"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The class ids the fixture registry answers. The domain is deliberately
// sparse — no two adjacent — so a loader keying by section index instead of ID
// answers nothing at any of them.
const (
	unitFlip0    = 3  // warrior sheet, Flip 0 — frames 3x2 then 5x4
	unitFlip1    = 7  // archer sheet, Flip 1, the spec's example scalars
	unitInherit  = 9  // no File of its own; Parent 3 hands it the warrior sheet
	unitNoPal    = 12 // a palette-less sheet
	unitBad      = 15 // a sheet spr256 refuses
	unitNoSheet  = 20 // a sheet the archive does not hold
	unitNoFrames = 25 // a sheet of no frames: nothing to draw
	unitMissing  = 5  // no section carries it — a lookup miss, not an entry
)

// unitPalette is the fixture sheets' palette; entries 1 and 7 are distinct so
// the sheets are ordinary palette-bearing ones.
func unitPalette() []color.RGBA {
	pal := make([]color.RGBA, 8)
	pal[1] = color.RGBA{R: 0xff, G: 0x00, B: 0x00}
	pal[7] = color.RGBA{R: 0x01, G: 0x02, B: 0x04}
	return pal
}

// warriorSheet holds two DIFFERENTLY SIZED frames: 3x2 at index 0, 5x4 at
// index 1. The size difference is the point — a loader drawing any frame but
// sheet frame 0, under either Flip layout, hands back a 5x4 where a 3x2 is
// due.
func warriorSheet() []byte {
	const op = true
	return synth.Sheet256(synth.Sheet256Options{
		Palette: unitPalette(),
		Frames: []synth.Frame256{
			{Width: 3, Height: 2, Pixels: []synth.Pixel256{
				{Index: 1, Opaque: op}, {}, {Index: 7, Opaque: op},
				{}, {Index: 1, Opaque: op}, {},
			}},
			{Width: 5, Height: 4},
		},
	})
}

// archerSheet is the second drawable sheet — the Flip 1 class's — with its own
// pair of differently sized frames: 2x2 at index 0, 4x1 at index 1.
func archerSheet() []byte {
	const op = true
	return synth.Sheet256(synth.Sheet256Options{
		Palette: unitPalette(),
		Frames: []synth.Frame256{
			{Width: 2, Height: 2, Pixels: []synth.Pixel256{
				{Index: 1, Opaque: op}, {}, {}, {Index: 7, Opaque: op},
			}},
			{Width: 4, Height: 1},
		},
	})
}

// unitFiles is the registry's [Files] table. Index 4 names art the archive
// deliberately does not hold.
var unitFiles = []string{`warrior\walk`, `archer\walk`, `flat\nopal`, `broken\bad`, `missing\gone`, `bare\none`}

// unitRegistry writes the fixture units/units.reg: drawable classes at both
// Flip values, one drawable class inheriting File through Parent, and one
// class per exclusion.
//
// The Flip 1 class carries the 0024 spec's worked example scalars — MB 1,
// MV 2, AT 2, DY 2, BN 2, ID 0, Move Time [2,1] / Frame [0,1] — so the
// descriptor the loader must copy across is the one whose bases, total and
// track the spec itself states as literals.
func unitRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	a := func(name string, vs ...int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x06, Ints: vs}
	}
	return synth.UnitsReg(unitFiles,
		[]synth.RegNode{
			i("ID", unitFlip0), i("File", 0), i("Flip", 0),
			i("Width", 48), i("Height", 56), i("CenterX", 24), i("CenterY", 50),
		},
		[]synth.RegNode{
			i("ID", unitFlip1), i("File", 1), i("Flip", 1),
			i("Palette", 1),
			i("Width", 32), i("Height", 40), i("CenterX", 16), i("CenterY", 36),
			i("MoveBeginPhases", 1), i("MovePhases", 2), i("AttackPhases", 2),
			i("DyingPhases", 2), i("BonePhases", 2), i("IdlePhases", 0),
			a("MoveAnimTime", 2, 1), a("MoveAnimFrame", 0, 1),
		},
		// No File key at all: it resolves through Parent to the warrior sheet
		// — units.reg File INHERITS, unlike objects.reg's — while the canvas
		// stays this class's own.
		[]synth.RegNode{
			i("ID", unitInherit), i("Parent", unitFlip0),
			i("Width", 20), i("Height", 20), i("CenterX", 10), i("CenterY", 18),
		},
		[]synth.RegNode{
			i("ID", unitNoPal), i("File", 2),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		[]synth.RegNode{
			i("ID", unitBad), i("File", 3),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		[]synth.RegNode{
			i("ID", unitNoSheet), i("File", 4),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
		[]synth.RegNode{
			i("ID", unitNoFrames), i("File", 5),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
	)
}

// unitArchiveFiles is the fixture graphics archive: the registry and every
// sheet the [Files] table names EXCEPT missing\gone.
func unitArchiveFiles(t *testing.T) []synth.File {
	t.Helper()
	return []synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: unitRegistry()},
		{Path: graphicsEntry(t, game.UnitOwnerPalette), Data: unitOwnerPalette()},
		{Path: "units/warrior/walk.256", Data: warriorSheet()},
		{Path: "units/archer/walk.256", Data: archerSheet()},
		// The palette-less variant: its frame decodes, only the palette is gone.
		{Path: "units/flat/nopal.256", Data: synth.Sheet256(synth.Sheet256Options{
			NoPalette: true,
			Frames:    []synth.Frame256{{Width: 2, Height: 1, Pixels: []synth.Pixel256{{Index: 1, Opaque: true}, {}}}},
		})},
		// A frame record whose block is 4096 bytes long and one byte wide: the
		// stream cannot be walked at all, so spr256 refuses it.
		{Path: "units/broken/bad.256", Data: synth.Sheet256Raw(
			synth.Palette256(unitPalette()),
			[]synth.Sheet256RawFrame{{Width: 8, Height: 8, DataSize: 4096, Data: []byte{0x88}}},
			0x80000000|1,
		)},
		// A palette-bearing sheet of NO frames: it decodes, but holds no frame
		// 0 — the "sheet holds no such frame" exclusion.
		{Path: "units/bare/none.256", Data: synth.Sheet256(synth.Sheet256Options{Palette: unitPalette()})},
	}
}

// unitOwnerPalette is the shared raw owner palette fixture: sixteen tables at
// byte zero, each entry in [B, G, R, reserved] order. Every table and channel
// differs, so the loader cannot pass by reusing one table or swapping channels.
func unitOwnerPalette() []byte {
	out := make([]byte, 16*256*4)
	for table := 0; table < 16; table++ {
		for i := 0; i < 256; i++ {
			e := out[(table*256+i)*4:]
			e[0] = byte(table*11 + i)
			e[1] = byte(table*17 + 255 - i)
			e[2] = byte(table*23 + i*7)
			e[3] = byte(table*29 + i*13)
		}
	}
	return out
}

// loadFixtureUnits loads the bundle out of the fixture archive.
func loadFixtureUnits(t *testing.T) *terrain.UnitSet {
	t.Helper()
	set, err := game.LoadUnits(openContainers(t, synth.Archive(unitArchiveFiles(t))))
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	if set == nil {
		t.Fatal("LoadUnits returned a nil set with no error")
	}
	return set
}

func TestLoadUnits(t *testing.T) {
	t.Run("the shared owner palette and class arm cross together", func(t *testing.T) {
		set := loadFixtureUnits(t)
		if !set.HasOwnerPalettes {
			t.Fatal("HasOwnerPalettes is false for a complete shared palette")
		}
		raw := unitOwnerPalette()
		for table := 0; table < 16; table++ {
			for _, i := range []int{0, 1, 55, 255} {
				e := raw[(table*256+i)*4:]
				want := color.RGBA{R: e[2], G: e[1], B: e[0], A: 0xff}
				if got := set.OwnerPalettes[table][i]; got != want {
					t.Fatalf("table %d entry %d = %+v, want %+v", table, i, got, want)
				}
			}
		}
		if !set.Classes[unitFlip0].OwnerShaded {
			t.Error("Palette 0 class does not take the shared owner palette")
		}
		if set.Classes[unitFlip1].OwnerShaded {
			t.Error("Palette 1 class takes the shared owner palette")
		}
		if got := set.OwnerPalette(set.Classes[unitFlip0], 17); got != &set.OwnerPalettes[1] {
			t.Fatalf("owner 17 selected %p, want wrapped table 1 at %p", got, &set.OwnerPalettes[1])
		}
		if got := set.OwnerPalette(set.Classes[unitFlip1], 1); got != nil {
			t.Fatalf("Palette 1 class selected owner table %p, want nil", got)
		}
	})

	t.Run("a missing or refused owner palette keeps the drawable bundle", func(t *testing.T) {
		base := unitArchiveFiles(t)
		without := make([]synth.File, 0, len(base)-1)
		for _, file := range base {
			if file.Path != graphicsEntry(t, game.UnitOwnerPalette) {
				without = append(without, file)
			}
		}
		for _, tc := range []struct {
			name  string
			files []synth.File
		}{
			{"missing", without},
			{"short", append(append([]synth.File(nil), without...), synth.File{
				Path: graphicsEntry(t, game.UnitOwnerPalette), Data: unitOwnerPalette()[:len(unitOwnerPalette())-1]})},
		} {
			t.Run(tc.name, func(t *testing.T) {
				set, err := game.LoadUnits(openContainers(t, synth.Archive(tc.files)))
				if err != nil {
					t.Fatalf("LoadUnits: %v", err)
				}
				if set.HasOwnerPalettes {
					t.Error("HasOwnerPalettes is true for a missing/refused resource")
				}
				if got := len(set.Classes[unitFlip0].Frames); got != 2 {
					t.Fatalf("drawable class has %d frames after cosmetic fallback, want 2", got)
				}
				if got := set.OwnerPalette(set.Classes[unitFlip0], 1); got != nil {
					t.Fatalf("fallback selected owner palette %p, want nil", got)
				}
			})
		}
	})

	t.Run("a drawable class is keyed by ID, canvas across, the WHOLE sheet aboard", func(t *testing.T) {
		set := loadFixtureUnits(t)

		if got := len(set.Classes); got != 7 {
			t.Errorf("the set holds %d entries, want 7 — one per loaded class and nothing else", got)
		}

		c := set.Classes[unitFlip0]
		if c == nil {
			t.Fatalf("Classes[%d] is nil; the bundle is keyed by the class's own ID", unitFlip0)
		}
		if c.Width != 48 || c.Height != 56 || c.CenterX != 24 || c.CenterY != 50 {
			t.Errorf("canvas = %dx%d centre (%d, %d), want 48x56 centre (24, 50)",
				c.Width, c.Height, c.CenterX, c.CenterY)
		}
		// EVERY frame, in sheet order, at its own size: the two warrior frames are
		// deliberately unequal, so a loader converting only frame 0 — or
		// reordering — moves a literal here.
		if len(c.Frames) != 2 {
			t.Fatalf("Classes[%d] holds %d frames, want the warrior sheet's 2", unitFlip0, len(c.Frames))
		}
		if f := c.Frames[0]; f.Width != 3 || f.Height != 2 || len(f.Pixels) != 6 {
			t.Errorf("Flip 0 frame 0 = %dx%d with %d pixels, want 3x2 with 6", f.Width, f.Height, len(f.Pixels))
		}
		if f := c.Frames[1]; f.Width != 5 || f.Height != 4 || len(f.Pixels) != 20 {
			t.Errorf("Flip 0 frame 1 = %dx%d with %d pixels, want 5x4 with 20", f.Width, f.Height, len(f.Pixels))
		}

		f := set.Classes[unitFlip1]
		if f == nil || len(f.Frames) != 2 {
			t.Fatalf("Classes[%d] = %v; the Flip 1 class must load with its sheet's 2 frames", unitFlip1, f)
		}
		if g := f.Frames[0]; g.Width != 2 || g.Height != 2 {
			t.Errorf("Flip 1 frame 0 = %dx%d, want 2x2", g.Width, g.Height)
		}
		if g := f.Frames[1]; g.Width != 4 || g.Height != 1 {
			t.Errorf("Flip 1 frame 1 = %dx%d, want 4x1", g.Width, g.Height)
		}
	})

	t.Run("the descriptor rides the entry, copied value for value", func(t *testing.T) {
		set := loadFixtureUnits(t)

		// The spec's worked example class (0024, "I/O example"), every field a
		// hand literal: Flip 1 -> (S, D) = (9, 5); MB 1, MV 2, AT 2, DY 2,
		// BN 2, ID 0 -> MoveBase 9, AttackBase 9+5*3 = 24, DyingBase
		// 9+5*5 = 34, TailBase 9+5*7 = 44, MoveSlot 3, MoveWind 1, IdleSlot 0,
		// Total 9+5*(7+max(2,0)) = 54; Time [2,1] / Frame [0,1] -> track
		// [0,0,1]; MV 2 > 0 gates move on, ID 0 gates idle off.
		a := set.Classes[unitFlip1].Anim
		if a.S != 9 || a.D != 5 {
			t.Errorf("(S, D) = (%d, %d), want (9, 5) at Flip 1", a.S, a.D)
		}
		if a.MoveBase != 9 || a.AttackBase != 24 || a.DyingBase != 34 || a.TailBase != 44 {
			t.Errorf("bases = %d/%d/%d/%d, want 9/24/34/44", a.MoveBase, a.AttackBase, a.DyingBase, a.TailBase)
		}
		if a.MoveSlot != 3 || a.MoveWind != 1 || a.IdleSlot != 0 {
			t.Errorf("MoveSlot/MoveWind/IdleSlot = %d/%d/%d, want 3/1/0", a.MoveSlot, a.MoveWind, a.IdleSlot)
		}
		if a.Total != 54 {
			t.Errorf("Total = %d, want the spec's predicted 54", a.Total)
		}
		if want := []int{0, 0, 1}; len(a.MoveTrack) != 3 || a.MoveTrack[0] != 0 || a.MoveTrack[1] != 0 || a.MoveTrack[2] != 1 {
			t.Errorf("MoveTrack = %v, want %v — Time [2,1] over Frame [0,1] run-length expanded", a.MoveTrack, want)
		}
		if len(a.IdleTrack) != 0 {
			t.Errorf("IdleTrack = %v, want empty — no idle pair was written", a.IdleTrack)
		}
		if !a.MoveOK || a.IdleOK {
			t.Errorf("gates = (move %t, idle %t), want (true, false)", a.MoveOK, a.IdleOK)
		}

		// The Flip 0 half of the layout rule, and the gates over the T1
		// defaults: no phase key set resolves MV -1 and ID 0, both tracks
		// empty, both gates off.
		b := set.Classes[unitFlip0].Anim
		if b.S != 16 || b.D != 8 {
			t.Errorf("(S, D) = (%d, %d), want (16, 8) at Flip 0", b.S, b.D)
		}
		if b.MoveOK || b.IdleOK {
			t.Errorf("gates = (move %t, idle %t) on a class with no anim keys, want (false, false)", b.MoveOK, b.IdleOK)
		}

		// A FRAMELESS entry still carries its descriptor: the derivation is
		// arithmetic over the class, not a fact about any sheet.
		if e := set.Classes[unitNoSheet]; e.Anim.S != 16 || e.Anim.D != 8 {
			t.Errorf("the frameless entry's (S, D) = (%d, %d), want (16, 8) — exclusion is about art alone",
				e.Anim.S, e.Anim.D)
		}
	})

	t.Run("File inherits through Parent, and the shared sheet is ONE slice", func(t *testing.T) {
		set := loadFixtureUnits(t)
		c := set.Classes[unitInherit]
		if c == nil {
			t.Fatalf("Classes[%d] is nil", unitInherit)
		}
		// The canvas is the class's OWN; only File came down the chain.
		if c.Width != 20 || c.Height != 20 || c.CenterX != 10 || c.CenterY != 18 {
			t.Errorf("canvas = %dx%d centre (%d, %d), want the class's own 20x20 centre (10, 18)",
				c.Width, c.Height, c.CenterX, c.CenterY)
		}
		if len(c.Frames) != 2 || c.Frames[0].Width != 3 || c.Frames[0].Height != 2 {
			t.Fatalf("inherited-File Frames = %v; the inherited File must resolve the parent's whole 2-frame sheet", c.Frames)
		}

		// ONE DECODE AND ONE CONVERSION PER DISTINCT PATH (0024 AC-8): both
		// classes naming the warrior sheet hold the SAME slice — the memo's own,
		// first element address included — and therefore the same frame
		// pointers, which is what lets the frame-keyed texture cache upload a
		// shared sheet once. A loader converting per class would hand equal VALUES
		// here with distinct pointers, and fail.
		p := set.Classes[unitFlip0]
		if &c.Frames[0] != &p.Frames[0] {
			t.Errorf("the inherited class holds its own frame slice (%p vs %p); a shared sheet must share the one converted slice",
				&c.Frames[0], &p.Frames[0])
		}
		for i := range p.Frames {
			if c.Frames[i] != p.Frames[i] {
				t.Errorf("frame %d differs in pointer identity between the two classes naming one sheet (%p vs %p)",
					i, c.Frames[i], p.Frames[i])
			}
		}
		// And the two DISTINCT sheets stay distinct: no over-merging.
		if q := set.Classes[unitFlip1]; len(q.Frames) > 0 && len(p.Frames) > 0 && q.Frames[0] == p.Frames[0] {
			t.Error("the warrior and archer sheets share a frame pointer; distinct paths must convert apart")
		}
	})

	t.Run("each exclusion keeps a frameless entry, distinct from a missing id", func(t *testing.T) {
		set := loadFixtureUnits(t)

		for _, tc := range []struct {
			id    int32
			label string
		}{
			{unitNoPal, "a palette-less sheet"},
			{unitBad, "a sheet spr256 refuses"},
			{unitNoSheet, "a sheet the archive does not hold"},
			{unitNoFrames, "a sheet of no frames"},
		} {
			c, ok := set.Classes[tc.id]
			if !ok || c == nil {
				t.Errorf("id %d (%s): the entry is missing; an exclusion is a class with no FRAMES", tc.id, tc.label)
				continue
			}
			if c.Frames != nil {
				t.Errorf("id %d (%s): Frames = %v, want nil", tc.id, tc.label, c.Frames)
			}
			// The canvas still loads: the exclusion is about the art alone.
			if c.Width == 0 && c.Height == 0 && c.CenterX == 0 && c.CenterY == 0 {
				t.Errorf("id %d (%s): the canvas came across as all zeroes", tc.id, tc.label)
			}
		}

		// An id naming no class is the OTHER answer: a clean lookup miss, at
		// either sign, never a frameless entry.
		for _, id := range []int32{unitMissing, -1, 0, 80} {
			if c, ok := set.Classes[id]; ok || c != nil {
				t.Errorf("Classes[%d] = (%v, %t), want a miss — no section carries this id", id, c, ok)
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
				label: "no units/units.reg in the archive",
				files: []synth.File{{Path: "units/warrior/walk.256", Data: warriorSheet()}},
				// The read error stays an *fs.PathError: -check is specified to
				// fail when the registry cannot be read, through errors.Is.
				notExist: true,
			},
			{
				label: "a registry that is not a registry",
				files: []synth.File{{Path: graphicsEntry(t, game.UnitRegistry), Data: []byte("not a .reg stream at all")}},
			},
			{
				label: "a registry that parses but does not resolve",
				// [Global] promises one class and no section carries it.
				files: []synth.File{{Path: graphicsEntry(t, game.UnitRegistry), Data: synth.Reg(0x11, []synth.RegNode{
					{Name: "Global", Kind: 0x01, Children: []synth.RegNode{
						{Name: "UnitCount", Kind: 0x02, Int: 1},
						{Name: "FileCount", Kind: 0x02, Int: 0},
					}},
				})}},
			},
		} {
			t.Run(tc.label, func(t *testing.T) {
				set, err := game.LoadUnits(openContainers(t, synth.Archive(tc.files)))
				if err == nil {
					t.Fatalf("LoadUnits succeeded")
				}
				if set != nil {
					t.Errorf("LoadUnits returned a non-nil set alongside an error")
				}
				if !strings.Contains(err.Error(), game.UnitRegistry) {
					t.Errorf("error %q does not name %q", err, game.UnitRegistry)
				}
				if tc.notExist {
					var pathErr *fs.PathError
					if !errors.As(err, &pathErr) {
						t.Errorf("error %v is not an *fs.PathError; the read error must come back unwrapped", err)
					}
					if !errors.Is(err, fs.ErrNotExist) {
						t.Errorf("error %v does not satisfy errors.Is(err, fs.ErrNotExist)", err)
					}
				}
			})
		}
	})

	t.Run("a nil archive is an error, not a panic", func(t *testing.T) {
		set, err := game.LoadUnits(nil)
		if err == nil {
			t.Fatalf("LoadUnits(nil) succeeded")
		}
		if set != nil {
			t.Errorf("LoadUnits(nil) returned a non-nil set alongside an error")
		}
		if !strings.Contains(err.Error(), game.UnitRegistry) {
			t.Errorf("error %q does not name %q", err, game.UnitRegistry)
		}
	})
}
