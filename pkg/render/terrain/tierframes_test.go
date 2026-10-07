package terrain_test

import (
	"image/color"
	"testing"

	"againrom/pkg/render/terrain"
)

// The tier selector, over a hand-built class: this tier is handed filled
// slices and builds none, so the fixture is three slices of frames written
// here and nothing is loaded, decoded or recoloured (0057 AC-6, AC-11).

// tierFrame is one frame whose palette says which slice it came from, so an
// assertion can name the answer rather than count pointers.
func tierFrame(mark uint8) *terrain.StaticFrame {
	f := &terrain.StaticFrame{
		Width:  1,
		Height: 1,
		Pixels: []terrain.StaticPixel{{Index: 7, Opaque: true}},
	}
	f.Palette[7] = color.RGBA{R: mark, A: 0xff}
	return f
}

func tierClass() *terrain.UnitClass {
	base := []*terrain.StaticFrame{tierFrame(0)}
	return &terrain.UnitClass{
		Frames: base,
		Tiers: [][]*terrain.StaticFrame{
			base,           // tier 1: the sheet's own, by identity
			{tierFrame(2)}, // tier 2: its own colours
			nil,            // tier 3: absent
			{},             // tier 4: refused
		},
	}
}

// same reports whether two slices are the same slice: same backing array and
// same length.
func same(a, b []*terrain.StaticFrame) bool {
	if len(a) != len(b) || len(a) == 0 {
		return len(a) == len(b)
	}
	return &a[0] == &b[0]
}

func TestTierFramesAnswersTheTiersOwnSlice(t *testing.T) {
	c := tierClass()
	if got := c.TierFrames(2); same(got, c.Frames) {
		t.Fatal("tier 2 answered the base slice")
	} else if got[0].Palette[7].R != 2 {
		t.Fatalf("tier 2 frame palette mark %d, want 2", got[0].Palette[7].R)
	}
	// Tier 1 is the base slice BY IDENTITY here, which is what the loader
	// produces wherever a tier's table equals the sheet's own.
	if !same(c.TierFrames(1), c.Frames) {
		t.Fatal("tier 1 is not the base slice by identity")
	}
}

// AC-6, AC-11. Every way of having no tier of one's own is one arm: below the
// range, above it, an absent slice and an empty one.
func TestTierFramesIsTotalAndFallsBackToTheSheet(t *testing.T) {
	c := tierClass()
	for _, tier := range []int{-4096, -1, 0, 3, 4, 5, 1 << 30} {
		if !same(c.TierFrames(tier), c.Frames) {
			t.Errorf("tier %d did not answer the base slice", tier)
		}
	}
}

// A class with no tiers at all draws exactly what it drew before this story —
// its own frames, by identity, at every tier anyone can name (AC-11).
func TestTierFramesOnAClassWithNoTiers(t *testing.T) {
	c := &terrain.UnitClass{Frames: []*terrain.StaticFrame{tierFrame(9)}}
	for tier := -2; tier <= 6; tier++ {
		if !same(c.TierFrames(tier), c.Frames) {
			t.Errorf("tier %d did not answer the base slice", tier)
		}
	}
}

// The two nil answers: a nil receiver, and a resolved class holding no frames.
// Neither is an error and neither panics — the caller that produced them needed
// no guard, and neither does this.
func TestTierFramesOnNothing(t *testing.T) {
	var nilClass *terrain.UnitClass
	if got := nilClass.TierFrames(1); got != nil {
		t.Errorf("nil class: TierFrames(1) = %v, want nil", got)
	}
	frameless := &terrain.UnitClass{Tiers: [][]*terrain.StaticFrame{nil}}
	for _, tier := range []int{0, 1, 2} {
		if got := frameless.TierFrames(tier); got != nil {
			t.Errorf("frameless class: TierFrames(%d) = %v, want nil", tier, got)
		}
	}
}

func TestTierFramesIsPure(t *testing.T) {
	c := tierClass()
	for tier := -1; tier <= 6; tier++ {
		first := c.TierFrames(tier)
		for probe := -1; probe <= 6; probe++ {
			c.TierFrames(probe)
		}
		if !same(c.TierFrames(tier), first) {
			t.Fatalf("tier %d moved after other calls", tier)
		}
	}
}
