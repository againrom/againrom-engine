package data

import (
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/reg"
)

// The projectile registry and a cast's own arithmetic. Every fixture is
// synthetic: internal/synth writes the .reg byte stream and pkg/formats/reg
// parses it back, so the loader only ever sees a tree it could have got from
// a real one. Nothing here reads an install.

func projRow(id int32, file string, keys ...synth.RegNode) []synth.RegNode {
	row := []synth.RegNode{
		{Name: "ID", Kind: 0x02, Int: id},
		{Name: "File", Kind: 0x00, Str: file},
	}
	return append(row, keys...)
}

func projKey(name string, v int32) synth.RegNode {
	return synth.RegNode{Name: name, Kind: 0x02, Int: v}
}

func projReg(t *testing.T, count int32, rows ...[]synth.RegNode) *reg.Reg {
	t.Helper()
	r, err := reg.Parse(synth.ProjectilesReg(count, rows...))
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	return r
}

// TestAProjectileRowTakesTheEnginesOwnDefaults is AC-1: a key a row omits is
// live at the value the engine's loader pushes before reading it, and on the
// shipped file that is eight rows for four of these keys.
func TestAProjectileRowTakesTheEnginesOwnDefaults(t *testing.T) {
	t.Parallel()

	r := projReg(t, 2,
		projRow(10, "firebolt\\sprites",
			projKey("Phases", 4), projKey("RotationPhases", 16),
			projKey("Width", 128), projKey("Height", 128),
			projKey("Palette", 1), projKey("Homing", 1), projKey("Flip", 1),
			projKey("SFX", 1), projKey("A16", 1)),
		projRow(20, "healing\\sprites"),
	)
	p, err := LoadProjectiles(r)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	if p.Len() != 2 {
		t.Fatalf("two rows loaded %d records", p.Len())
	}

	full, ok := p.ByID(10)
	if !ok {
		t.Fatal("the row stating every key is not addressable at its own ID")
	}
	if full.Phases != 4 || full.RotationPhases != 16 || full.Width != 128 ||
		full.Flip != 1 || full.A16 != 1 || full.Palette != 1 {
		t.Errorf("the stated row loaded as %+v", full)
	}

	bare, ok := p.ByID(20)
	if !ok {
		t.Fatal("the row stating two keys is not addressable at its own ID")
	}
	if bare.Phases != projectilePhases {
		t.Errorf("an absent Phases is %d, want the engine's %d", bare.Phases, projectilePhases)
	}
	if bare.RotationPhases != projectileRotationPhases {
		t.Errorf("an absent RotationPhases is %d, want %d",
			bare.RotationPhases, projectileRotationPhases)
	}
	if bare.Width != projectileWidth || bare.Height != projectileHeight {
		t.Errorf("absent Width/Height are %d/%d, want %d/%d",
			bare.Width, bare.Height, projectileWidth, projectileHeight)
	}
	if bare.Palette != 0 || bare.Homing != 0 || bare.Flip != 0 || bare.SFX != 0 || bare.A16 != 0 {
		t.Errorf("the four zero-defaulted keys loaded as %+v", bare)
	}
}

func TestARegistryIsKeyedByIDAndNotBySectionNumber(t *testing.T) {
	t.Parallel()

	r := projReg(t, 2,
		projRow(25, "poison\\sprites"),
		projRow(15, "firewall\\sprites"),
	)
	p, err := LoadProjectiles(r)
	if err != nil {
		t.Fatalf("LoadProjectiles: %v", err)
	}
	if got, ok := p.ByID(25); !ok || got.File != "poison\\sprites" {
		t.Errorf("section 0's row is not at ID 25")
	}
	if _, ok := p.ByID(0); ok {
		t.Error("a record is addressable at its section number")
	}
	if _, ok := p.ByID(1); ok {
		t.Error("a record is addressable at its section number")
	}
}

// TestARegistryWithNoCountIsRefusedAndAShortOneIsNot separates the one failure
// from the two skips: a registry that does not say how many rows it has is not
// this registry, and a Count overshooting its sections is readable data.
func TestARegistryWithNoCountIsRefusedAndAShortOneIsNot(t *testing.T) {
	t.Parallel()

	bare, err := reg.Parse(synth.Reg(0x11, []synth.RegNode{
		{Name: "Projectile0", Kind: 0x01, Children: projRow(10, "x")},
	}))
	if err != nil {
		t.Fatalf("reg.Parse: %v", err)
	}
	if _, err := LoadProjectiles(bare); err == nil {
		t.Error("a registry with no [Global] Count loaded without error")
	}

	// A Count of nine over two sections, the second of which states no ID.
	over := projReg(t, 9,
		projRow(10, "x"),
		[]synth.RegNode{{Name: "File", Kind: 0x00, Str: "no id here"}},
	)
	p, err := LoadProjectiles(over)
	if err != nil {
		t.Fatalf("a Count overshooting its sections failed the load: %v", err)
	}
	if p.Len() != 1 {
		t.Errorf("the short registry loaded %d records, want the one row that states an ID", p.Len())
	}
}

// TestANilRegistryLoadsNothingAndFailsNot is the shape LoadSpells already uses
// for an absent collection.
func TestANilRegistryLoadsNothingAndFailsNot(t *testing.T) {
	t.Parallel()

	p, err := LoadProjectiles(nil)
	if err != nil || p.Len() != 0 {
		t.Errorf("LoadProjectiles(nil) = %v, %v; want an empty set and no error", p.Len(), err)
	}
	if _, ok := (*Projectiles)(nil).ByID(10); ok {
		t.Error("a nil set answered a record")
	}
	if (*Projectiles)(nil).Len() != 0 {
		t.Error("a nil set has a length")
	}
}

func TestASheetAddressIsTheFilePathUnderTheProjectileDirectory(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		p    Projectile
		want string
	}{
		{"a .16a row", Projectile{File: "firebolt\\sprites", A16: 1},
			"projectiles/firebolt/sprites.16a"},
		{"an indexed row, A16 0", Projectile{File: "archer\\arrow", A16: 0},
			"projectiles/archer/arrow.256"},
		{"a row naming no file", Projectile{A16: 1}, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.p.SpritePath(); got != tc.want {
				t.Errorf("SpritePath() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestASpellComputesItsTwoPictures is AC-2: no Spells column carries a picture
// id, so both are arithmetic, and seven of the 28 spells compute an id the game
// ships no art at either parity for.
func TestASpellComputesItsTwoPictures(t *testing.T) {
	t.Parallel()

	for id := range 29 {
		if got, want := CastPicture(id), 2*id+8; got != want {
			t.Errorf("CastPicture(%d) = %d, want %d", id, got, want)
		}
		if got, want := BurstPicture(id), 2*id+9; got != want {
			t.Errorf("BurstPicture(%d) = %d, want %d", id, got, want)
		}
	}

	// The shipped registry's own id set, transcribed. A spell resolves art when
	// either of its two pictures is in it.
	shipped := map[int]bool{}
	for _, id := range []int{1, 2, 3, 6, 7, 10, 12, 13, 15, 17, 18, 20, 23, 24, 25, 27, 28, 30,
		34, 36, 40, 44, 47, 51, 52, 54, 60, 62} {
		shipped[id] = true
	}
	var bare []int
	for id := 1; id <= 28; id++ {
		if !shipped[CastPicture(id)] && !shipped[BurstPicture(id)] {
			bare = append(bare, id)
		}
	}
	want := []int{12, 15, 17, 20, 24, 25, 28}
	if len(bare) != len(want) {
		t.Fatalf("%v spells resolve no art at either parity, want %v", bare, want)
	}
	for i, id := range want {
		if bare[i] != id {
			t.Errorf("the spells with no art are %v, want %v", bare, want)
			break
		}
	}
}

// TestOnlySevenPicturesHaveAFlightLength is AC-3: the engine's own 51-byte index
// table has six arms and the zero arm covers 44 of the pictures.
func TestOnlySevenPicturesHaveAFlightLength(t *testing.T) {
	t.Parallel()

	const fiveCells = 5 * PictureCellUnits
	want := map[int]int{10: 6, 12: 3, 20: 1, 30: 1, 34: 13, 36: 13, 60: 21}
	for picture := 0; picture <= 65; picture++ {
		got := CastFlight(picture, fiveCells)
		if w, ok := want[picture]; ok {
			if got != w {
				t.Errorf("picture %d flies for %d ticks over five cells, want %d", picture, got, w)
			}
			if !CastFlies(picture) {
				t.Errorf("picture %d does not report as flying", picture)
			}
			continue
		}
		if got != 0 {
			t.Errorf("picture %d flies for %d ticks, want 0 — it spawns nothing", picture, got)
		}
		if CastFlies(picture) {
			t.Errorf("picture %d reports as flying", picture)
		}
	}
}

func TestADividedFlightLengthIsNeverZero(t *testing.T) {
	t.Parallel()

	for _, picture := range []int{10, 12} {
		for _, dist := range []int{-99, 0, 1, 199} {
			if got := CastFlight(picture, dist); got != 1 {
				t.Errorf("picture %d at distance %d flies for %d ticks, want 1", picture, dist, got)
			}
		}
	}
}

func TestTwoBurstsAreGivenALifeOfTwiceTheirPhaseCount(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ picture, want int }{
		{13, 22}, {27, 18}, {15, 16}, {17, 16}, {23, 16}, {25, 16}, {47, 16}, {51, 16},
		// A picture with no burst art still answers a life; what makes nothing
		// drawn is the missing record.
		{11, 16}, {21, 16},
	} {
		if got := BurstLife(tc.picture); got != tc.want {
			t.Errorf("BurstLife(%d) = %d, want %d", tc.picture, got, tc.want)
		}
	}
}
