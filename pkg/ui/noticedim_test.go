package ui

import (
	"math"
	"testing"
)

// noticeDimGain is the retained menu and custom alpha-wash policy.
const noticeDimGain = 13.0 / 16.0

// gainOfAlpha is the per-channel multiply an 8-bit alpha spells under the
// composite the draw performs. It is the seam's whole arithmetic and is written
// once, so the search and the bound below cannot come to disagree about it.
func gainOfAlpha(a uint8) float64 { return float64(255-a) / 255.0 }

func TestTheDimIsBlackAndNeitherDegenerateStrength(t *testing.T) {
	c := AuthoredNoticeBackdrop()

	if c.R != 0 || c.G != 0 || c.B != 0 {
		t.Errorf("the dim is {R:%d G:%d B:%d}, not black — compositing it would ADD light on a "+
			"channel and shift the map's hue, which is not a per-channel gain", c.R, c.G, c.B)
	}
	if c.A == 0 {
		t.Error("the dim is fully transparent — it would darken nothing")
	}
	if c.A == 0xff {
		t.Error("the dim is fully opaque — it would hide the map rather than darken it")
	}
}

// AC-2 — the shipped alpha is the NEAREST the type can spell to the required
// gain, established by searching all 256 rather than by comparing with a
// remembered number.
//
// THE SEARCH IS THE POINT. Asserting the literal would witness only that nobody
// changed it. This witnesses that no other value the seam can carry is closer,
// which is the actual requirement — and it goes on being the right assertion if
// the gain is amended or the alpha's precision changes.
func TestTheDimIsTheNearestGainAnAlphaCanSpell(t *testing.T) {
	best, bestErr := 0, math.Inf(1)
	for a := 0; a <= 0xff; a++ {
		if e := math.Abs(gainOfAlpha(uint8(a)) - noticeDimGain); e < bestErr {
			best, bestErr = a, e
		}
	}

	got := AuthoredNoticeBackdrop().A
	if int(got) != best {
		t.Errorf("the dim ships alpha 0x%02x (gain %.6f, off by %.6f), but 0x%02x spells "+
			"gain %.6f and is closer to %.6f", got, gainOfAlpha(got),
			math.Abs(gainOfAlpha(got)-noticeDimGain), best, gainOfAlpha(uint8(best)), noticeDimGain)
	}
	t.Logf("gain wanted %.7f; nearest 8-bit alpha 0x%02x spells %d/255 = %.7f (residual %+.7f)",
		noticeDimGain, best, 255-best, gainOfAlpha(uint8(best)), gainOfAlpha(uint8(best))-noticeDimGain)
}

// AC-3 — the residual is BOUNDED, and the bound is expressed in the units the
// divergence is actually felt in: eighths of an 8-bit level.
//
// A relative error means nothing on its own here. What a reader needs to know is
// the largest amount by which a channel can come out wrong, and that is the
// residual gain times the largest channel value — so the bound is stated as a
// fraction of one level, and a quarter of a level is under half a rounding step
// everywhere.
func TestTheDimsResidualIsUnderAQuarterOfOneLevel(t *testing.T) {
	const maxLevel = 255.0
	const bound = 0.25

	got := AuthoredNoticeBackdrop().A
	worst := math.Abs(gainOfAlpha(got)-noticeDimGain) * maxLevel
	if worst >= bound {
		t.Errorf("the shipped gain %.7f is %.4f of one 8-bit level away from %.7f at full scale, "+
			"which is not under the %.2f the contract bounds it by",
			gainOfAlpha(got), worst, noticeDimGain, bound)
	}
	t.Logf("worst per-channel deviation over [0,255]: %.4f of one level (bound %.2f)", worst, bound)
}
