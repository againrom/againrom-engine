package game_test

import (
	"image/color"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/game"
	"againrom/pkg/render/terrain"
)

// The hero body loader (AC-8, AC-9, plan SC-6, SC-7).
//
// EVERY FIXTURE IS SYNTHETIC. An in-memory .res archive holding a synthetic
// units.reg and two synthetic .256 sheets; no game install is read and no window
// is opened.
//
// The fixture's two sheets differ in FRAME COUNT and in every frame's SIZE, and
// that is the whole instrument: the composite has to carry one sheet's frames
// and the other record's geometry, and a loader that took either from the wrong
// side answers a number this file already knows.

const (
	heroBodyKey   = 3 // the class key "swordsman" resolves to
	heroCorpseKey = 4 // what that class's Dying key names
)

// bodyRecordSheet is the sheet the CLASS RECORD's own File names: two frames.
func bodyRecordSheet() []byte {
	pal := make([]color.RGBA, 8)
	pal[1] = color.RGBA{R: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{
			{Width: 3, Height: 3}, {Width: 3, Height: 3},
		},
	})
}

// bodyComposedSheet is the sheet the COMPOSED PATH names: three frames, none of
// them the record sheet's size.
func bodyComposedSheet() []byte {
	pal := make([]color.RGBA, 8)
	pal[2] = color.RGBA{B: 0xff}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{
			{Width: 7, Height: 2}, {Width: 7, Height: 2}, {Width: 7, Height: 2},
		},
	})
}

func heroBodyRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	s := func(name, v string) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x00, Str: v}
	}
	return synth.UnitsReg(
		[]string{`humans\swordsman\swordsman`, `humans\swordsman_\swordsman_`},
		[]synth.RegNode{
			i("ID", heroBodyKey), i("File", 0), s("DescText", "Human Swordsman"),
			// ONE DECLARED TIER, and it is load-bearing rather than decoration:
			// a tier is a colour table for the RECORD's own sheet, so a loader
			// that carried the record's tier slices onto the body would draw a
			// hero's picture through a table built for a different one. With no
			// tier declared here that mistake is invisible, which a mutation
			// found by surviving.
			i("Palette", 1),
			i("Dying", heroCorpseKey), i("Flip", 1),
			i("Width", 48), i("Height", 56), i("CenterX", 24), i("CenterY", 50),
			i("MoveBeginPhases", 1), i("MovePhases", 2), i("AttackPhases", 2),
			i("DyingPhases", 2), i("BonePhases", 2), i("IdlePhases", 0),
		},
		[]synth.RegNode{
			i("ID", heroCorpseKey), i("File", 1), s("DescText", "Human Swordsman with shield"),
			i("Dying", heroCorpseKey),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
	)
}

// loadWithBody builds the fixture bundle and resolves one body into it. files
// beyond the registry are the caller's, so a test can leave one out.
func loadWithBody(t *testing.T, dir string, body data.HeroBody, extra ...synth.File) *terrain.UnitSet {
	t.Helper()
	// The record's own tier table, and it DIFFERS from that sheet's own palette
	// — so the record's tier 1 is a recoloured slice of its own two frames and
	// is distinguishable from everything else in this fixture.
	tier := make([]color.RGBA, 8)
	tier[1] = color.RGBA{G: 0xff}
	files := append([]synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: heroBodyRegistry()},
		{Path: "units/humans/swordsman/swordsman.256", Data: bodyRecordSheet()},
		{Path: "units/humans/swordsman/palette.pal", Data: tierPaletteFile(tier)},
	}, extra...)
	src := openContainers(t, synth.Archive(files))
	set, err := game.LoadUnits(src)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}
	game.LoadHeroBody(src, set, dir, body)
	return set
}

// AC-8 — the composed sheet's frames on the class record's geometry.
func TestAHeroBodyDrawsTheComposedSheetOnTheRecordsGeometry(t *testing.T) {
	set := loadWithBody(t, "heroes_l", "swordsman", synth.File{
		Path: "units/heroes_l/swordsman/sprites.256", Data: bodyComposedSheet()})

	rec := set.Classes[heroBodyKey]
	if rec == nil {
		t.Fatalf("Classes[%d] is nil; the fixture registry did not load", heroBodyKey)
	}
	b := set.Bodies[data.HeroBodyKey("heroes_l", "swordsman")]
	if b == nil {
		t.Fatal(`Bodies[HeroBodyKey("heroes_l", "swordsman")] is nil; the composed sheet is in the archive`)
	}

	// The frames are the COMPOSED sheet's — count and size both, so neither
	// could have come from the record's own.
	if len(b.Frames) != 3 {
		t.Fatalf("the body draws %d frames, want the composed sheet's 3", len(b.Frames))
	}
	for k, f := range b.Frames {
		if f.Width != 7 || f.Height != 2 {
			t.Errorf("body frame %d is %dx%d, want the composed sheet's 7x2", k, f.Width, f.Height)
		}
	}

	// The geometry, the name, the descriptor and the corpse link are the
	// RECORD's, value for value.
	if b.Width != rec.Width || b.Height != rec.Height ||
		b.CenterX != rec.CenterX || b.CenterY != rec.CenterY {
		t.Errorf("body canvas %dx%d centre (%d, %d), want the record's %dx%d centre (%d, %d)",
			b.Width, b.Height, b.CenterX, b.CenterY,
			rec.Width, rec.Height, rec.CenterX, rec.CenterY)
	}
	if b.Name != rec.Name {
		t.Errorf("body name %q, want the record's %q", b.Name, rec.Name)
	}
	if !reflect.DeepEqual(b.Anim, rec.Anim) {
		t.Errorf("body descriptor %+v, want the record's %+v", b.Anim, rec.Anim)
	}
	if b.Corpse != rec.Corpse || b.Corpse != set.Classes[heroCorpseKey] {
		t.Errorf("body corpse %p, want the record's own link %p", b.Corpse, rec.Corpse)
	}
	// A tier is a colour table for the RECORD's sheet and is not carried. The
	// record declares one and it resolved, so this is a real slice being
	// dropped rather than a nil field staying nil.
	if len(rec.Tiers) != 1 || len(rec.Tiers[0]) != 2 {
		t.Fatalf("setup: the record resolved %d tier(s); the fixture declares one of two frames",
			len(rec.Tiers))
	}
	if b.Tiers != nil {
		t.Errorf("the body carries %d tier slice(s); a tier belongs to the record's own sheet", len(b.Tiers))
	}
	if got := b.TierFrames(1); len(got) != 3 {
		t.Errorf("TierFrames(1) gives %d frames, want the body's own 3", len(got))
	}

	if len(rec.Frames) != 2 {
		t.Errorf("the class record now draws %d frames, want its own 2", len(rec.Frames))
	}
}

func TestAHeroBodyRetainsOwnerShadeEligibility(t *testing.T) {
	src := openContainers(t, synth.Archive([]synth.File{{
		Path: "units/heroes_l/swordsman/sprites.256", Data: bodyComposedSheet(),
	}}))
	rec := &terrain.UnitClass{Width: 48, Height: 56, CenterX: 24, CenterY: 50, OwnerShaded: true}
	set := &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{heroBodyKey: rec}}
	game.LoadHeroBody(src, set, "heroes_l", "swordsman")
	body := set.Bodies[data.HeroBodyKey("heroes_l", "swordsman")]
	if body == nil {
		t.Fatal("owner-shaded class produced no composed hero body")
	}
	if !body.OwnerShaded {
		t.Fatal("the structural hero-body copy dropped OwnerShaded")
	}
	if body == rec {
		t.Fatal("the hero body aliases the class record instead of structurally copying it")
	}
}

// AC-9 — every way of not resolving is a skip, and each leaves NO entry rather
// than a half-built one.
func TestAnUnresolvableHeroBodyIsASkip(t *testing.T) {
	for _, tc := range []struct {
		name  string
		dir   string
		body  data.HeroBody
		extra []synth.File
		why   string
	}{
		{name: "no entry", dir: "heroes_l", body: "swordsman",
			why: "the composed address is not in the archive"},
		{name: "wrong directory", dir: "heroes", body: "swordsman",
			extra: []synth.File{{Path: "units/heroes_l/swordsman/sprites.256", Data: bodyComposedSheet()}},
			why:   "the sheet ships under the other directory"},
		{name: "no class record", dir: "heroes_l", body: "xbowman",
			extra: []synth.File{{Path: "units/heroes_l/xbowman/sprites.256", Data: bodyComposedSheet()}},
			why:   "the name resolves to a class this bundle does not hold"},
		{name: "sheet of no frames", dir: "heroes_l", body: "swordsman",
			extra: []synth.File{{Path: "units/heroes_l/swordsman/sprites.256",
				Data: synth.Sheet256(synth.Sheet256Options{Palette: make([]color.RGBA, 8)})}},
			why: "a sheet holding no frame draws nothing"},
		{name: "empty name", dir: "heroes_l", body: "",
			why: "an empty name addresses nothing"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := loadWithBody(t, tc.dir, tc.body, tc.extra...)
			key := data.HeroBodyKey(tc.dir, tc.body)
			if b := set.Bodies[key]; b != nil {
				t.Errorf("Bodies[%q] = %+v, want no entry — %s", key, b, tc.why)
			}
			// And the bundle it was resolved into is untouched: the class
			// record still draws its own art, which is what a missing entry
			// leaves an entity drawing.
			if rec := set.Classes[heroBodyKey]; rec == nil || len(rec.Frames) != 2 {
				t.Errorf("the class record no longer draws its own two frames")
			}
		})
	}
}

// A nil source and a nil bundle are refused before anything is read, so a
// hand-assembled front end can call this loader as freely as it calls the rest.
func TestLoadHeroBodyRefusesNilsWithoutPanicking(t *testing.T) {
	game.LoadHeroBody(nil, &terrain.UnitSet{}, "heroes_l", "swordsman")
	game.LoadHeroBody(openContainers(t, synth.Archive(nil)), nil, "heroes_l", "swordsman")
}

// AC-17 — two directories shipping one name each resolve to their own art:
// both entries land in one bundle and neither is mistaken for the other's.
func TestTwoDirectoriesShippingOneNameBothResolveIntoOneBundle(t *testing.T) {
	// heavySheet is the "heroes" directory's own sheet: four frames of 5x5,
	// distinguishable from bodyComposedSheet's three frames of 7x2 by both
	// the count and the size, so a mixed-up read is caught either way.
	heavySheet := func() []byte {
		pal := make([]color.RGBA, 8)
		pal[3] = color.RGBA{G: 0xff}
		return synth.Sheet256(synth.Sheet256Options{
			Palette: pal,
			Frames: []synth.Frame256{
				{Width: 5, Height: 5}, {Width: 5, Height: 5},
				{Width: 5, Height: 5}, {Width: 5, Height: 5},
			},
		})
	}
	tier := make([]color.RGBA, 8)
	tier[1] = color.RGBA{G: 0xff}
	src := openContainers(t, synth.Archive([]synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: heroBodyRegistry()},
		{Path: "units/humans/swordsman/swordsman.256", Data: bodyRecordSheet()},
		{Path: "units/humans/swordsman/palette.pal", Data: tierPaletteFile(tier)},
		{Path: "units/heroes_l/swordsman/sprites.256", Data: bodyComposedSheet()},
		{Path: "units/heroes/swordsman/sprites.256", Data: heavySheet()},
	}))
	set, err := game.LoadUnits(src)
	if err != nil {
		t.Fatalf("LoadUnits: %v", err)
	}

	game.LoadHeroBody(src, set, "heroes_l", "swordsman")
	game.LoadHeroBody(src, set, "heroes", "swordsman")

	light := set.Bodies[data.HeroBodyKey("heroes_l", "swordsman")]
	heavy := set.Bodies[data.HeroBodyKey("heroes", "swordsman")]
	if light == nil {
		t.Fatal(`Bodies[HeroBodyKey("heroes_l", "swordsman")] is nil`)
	}
	if heavy == nil {
		t.Fatal(`Bodies[HeroBodyKey("heroes", "swordsman")] is nil`)
	}
	if light == heavy {
		t.Fatal("the two directories share one entry; one name under two directories collided")
	}
	if len(light.Frames) != 3 {
		t.Errorf("the heroes_l entry draws %d frame(s), want its own composed sheet's 3", len(light.Frames))
	}
	if len(heavy.Frames) != 4 {
		t.Errorf("the heroes entry draws %d frame(s), want its own sheet's 4", len(heavy.Frames))
	}
}

// D-7 — a body once loaded is never decoded twice: a second call for the SAME
// (dir, body) leaves the bundle's own entry standing even when the archive
// handed to it would answer a different sheet.
func TestLoadHeroBodyDoesNotReloadAKeyItAlreadyHolds(t *testing.T) {
	set := loadWithBody(t, "heroes_l", "swordsman", synth.File{
		Path: "units/heroes_l/swordsman/sprites.256", Data: bodyComposedSheet()})
	key := data.HeroBodyKey("heroes_l", "swordsman")
	first := set.Bodies[key]
	if first == nil || len(first.Frames) != 3 {
		t.Fatalf("setup: the first load did not resolve the composed sheet's 3 frames")
	}

	// A second archive over the SAME address, holding a sheet of a DIFFERENT
	// frame count — if this second call reached it, the entry would change
	// from 3 frames to 5.
	src := openContainers(t, synth.Archive([]synth.File{
		{Path: graphicsEntry(t, game.UnitRegistry), Data: heroBodyRegistry()},
		{Path: "units/humans/swordsman/swordsman.256", Data: bodyRecordSheet()},
		{Path: "units/heroes_l/swordsman/sprites.256", Data: synth.Sheet256(synth.Sheet256Options{
			Palette: make([]color.RGBA, 8),
			Frames: []synth.Frame256{
				{Width: 1, Height: 1}, {Width: 1, Height: 1}, {Width: 1, Height: 1},
				{Width: 1, Height: 1}, {Width: 1, Height: 1},
			},
		})},
	}))
	game.LoadHeroBody(src, set, "heroes_l", "swordsman")

	got := set.Bodies[key]
	if got != first {
		t.Errorf("the entry changed to %p, want the first load's own %p left standing", got, first)
	}
	if len(got.Frames) != 3 {
		t.Errorf("the entry now carries %d frame(s), want the first load's own 3 — it was reloaded", len(got.Frames))
	}
}

func dyingBodySheet(w int) []byte {
	pal := make([]color.RGBA, 8)
	pal[3] = color.RGBA{R: 0x80, G: 0x80}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: pal,
		Frames: []synth.Frame256{
			{Width: w, Height: 4}, {Width: w, Height: 4}, {Width: w, Height: 4}, {Width: w, Height: 4},
		},
	})
}

func dyingBodyRegistry() []byte {
	i := func(name string, v int32) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x02, Int: v}
	}
	s := func(name, v string) synth.RegNode {
		return synth.RegNode{Name: name, Kind: 0x00, Str: v}
	}
	return synth.UnitsReg(
		[]string{`humans\unarmed\unarmed`, `humans\swordsman\swordsman`, `humans\mage\mage`, `humans\mage\mage_st`, `humans\swordsman_\swordsman_`},
		[]synth.RegNode{
			i("ID", 1), i("File", 0), s("DescText", "Unarmed"), i("Dying", 1),
			i("Width", 30), i("Height", 40), i("CenterX", 15), i("CenterY", 36),
			i("DyingPhases", 1), i("BonePhases", 1),
		},
		[]synth.RegNode{
			i("ID", 3), i("File", 0), s("DescText", "Swordsman"), i("Dying", 4),
			i("Width", 48), i("Height", 56), i("CenterX", 24), i("CenterY", 50),
			i("DyingPhases", 1), i("BonePhases", 1),
		},
		[]synth.RegNode{
			i("ID", 23), i("File", 0), s("DescText", "Mage"), i("Dying", 24),
			i("Width", 44), i("Height", 52), i("CenterX", 22), i("CenterY", 46),
			i("DyingPhases", 1), i("BonePhases", 1),
		},
		[]synth.RegNode{
			i("ID", 24), i("File", 0), s("DescText", "Mage staff"), i("Dying", 24),
			i("Width", 46), i("Height", 54), i("CenterX", 23), i("CenterY", 48),
			i("DyingPhases", 1), i("BonePhases", 1),
		},
		[]synth.RegNode{
			i("ID", 4), i("File", 0), s("DescText", "Swordsman with shield"), i("Dying", 4),
			i("Width", 16), i("Height", 16), i("CenterX", 8), i("CenterY", 14),
		},
	)
}

// A fallen hero is drawn from the composed sheet of the forced dying body, not
// from the class record's own corpse sheet.
func TestAFallenHeroDrawsTheForcedDyingBodyOfItsDirectory(t *testing.T) {
	for _, tc := range []struct {
		body  data.HeroBody
		dying string
		class int32
		width int
	}{
		{"swordsman", "unarmed", 1, 9},
		{"mage", "mage_st", 24, 11},
	} {
		t.Run(string(tc.body), func(t *testing.T) {
			src := openContainers(t, synth.Archive([]synth.File{
				{Path: graphicsEntry(t, game.UnitRegistry), Data: dyingBodyRegistry()},
				{Path: "units/heroes/" + string(tc.body) + "/sprites.256", Data: dyingBodySheet(5)},
				{Path: "units/heroes/" + tc.dying + "/sprites.256", Data: dyingBodySheet(tc.width)},
			}))
			set, err := game.LoadUnits(src)
			if err != nil {
				t.Fatalf("LoadUnits: %v", err)
			}
			game.LoadHeroBody(src, set, "heroes", tc.body)
			b := set.Bodies[data.HeroBodyKey("heroes", tc.body)]
			if b == nil {
				t.Fatal("no composed live body")
			}
			c := b.Corpse
			if c == nil || len(c.Frames) != 4 || c.Frames[0].Width != tc.width {
				t.Fatalf("fallen body = %+v, want the %s sheet of width %d", c, tc.dying, tc.width)
			}
			want := set.Classes[tc.class]
			if c.Width != want.Width || c.Height != want.Height {
				t.Errorf("fallen body geometry %dx%d, want class %d's %dx%d", c.Width, c.Height, tc.class, want.Width, want.Height)
			}
		})
	}
}
