package terrain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"testing"
)

// secondShieldClaim is R2-ENGINE-693's Shield entries at phase p and scale s,
// over the midpoint circle the program samples.
func secondShieldClaim(p, s, picture int) []EffectMark {
	theta := float64(4*p) * math.Pi / 180
	sin, cos := math.Sin(theta), math.Cos(theta)
	var out []EffectMark
	for _, pt := range sampledMidpointCircle(16*s, 4) {
		x, y := float64(pt[0]), pt[1]
		f := abs(4 - int(float64(5*abs(y))/float64(28*s)))
		dy := -int(1.75 * float64(y))
		out = append(out,
			EffectMark{DX: int(x * cos), DY: dy, Depth: 11*s + int(x*sin), Record: picture, Phase: f},
			EffectMark{DX: -int(x * sin), DY: dy, Depth: 11*s + int(x*cos), Record: picture, Phase: f})
	}
	h := int(float64(28*s) * (float64(float64(p)*(1.0/45)) - 1))
	w := int(float64(16*s) * math.Sin(math.Acos(float64(h)/float64(28*s))))
	k := 4 - abs(p-45)/9
	for _, pt := range sampledMidpointCircle(w, 2*k) {
		out = append(out,
			EffectMark{DX: pt[0], DY: -(11*s + h), Depth: pt[1], Record: picture, Phase: k},
			EffectMark{DX: pt[0], DY: -(11*s - h), Depth: pt[1], Record: picture, Phase: k})
	}
	return out
}

// TestTheSecondGameShieldFollowsItsClaim: every phase of the 90-update cycle
// at three scales builds R2-ENGINE-693's records in order, under the picture
// the caller names.
func TestTheSecondGameShieldFollowsItsClaim(t *testing.T) {
	for s := 1; s <= 3; s++ {
		for p := 0; p < 90; p++ {
			got := EffectMarkRecords(MarkProgramSecondShield, 62, uint16(p), s)
			want := secondShieldClaim(p, s, 62)
			if len(got) != len(want) {
				t.Errorf("s %d phase %d: %d records, want %d", s, p, len(got), len(want))
				continue
			}
			for i := range want {
				if got[i] != want[i] {
					t.Errorf("s %d phase %d record %d = %+v, want %+v", s, p, i, got[i], want[i])
					break
				}
			}
		}
	}
	if a, b := EffectMarkRecords(MarkProgramSecondShield, 62, 0xffff, 1), secondShieldClaim(0xffff%90, 1, 62); len(a) != len(b) {
		t.Errorf("count 0xffff built %d records, want phase 15's %d", len(a), len(b))
	}
}

// TestTheSecondGameBlessRingFollowsItsClaim: R2-ENGINE-693's twenty sparks
// keep their twenty positions 18 degrees apart on the radius-20 ring at every
// count, and each base angle's frame moves 18 degrees round it per update.
func TestTheSecondGameBlessRingFollowsItsClaim(t *testing.T) {
	positions := func(marks []EffectMark) map[[2]int]int {
		out := map[[2]int]int{}
		for _, m := range marks {
			out[[2]int{m.DX, m.Depth}] = m.Phase
		}
		return out
	}
	first := positions(EffectMarkRecords(MarkProgramSecondBless, 48, 0xffff, 2))
	for c := uint16(0xfffe); c >= 0xfff0; c-- {
		marks := EffectMarkRecords(MarkProgramSecondBless, 48, c, 2)
		if len(marks) != 20 {
			t.Fatalf("count %#x built %d sparks, want 20", c, len(marks))
		}
		now := positions(marks)
		if len(now) != 20 {
			t.Fatalf("count %#x put the sparks on %d positions, want 20", c, len(now))
		}
		for pos := range now {
			if _, ok := first[pos]; !ok {
				t.Errorf("count %#x put a spark at %v, off the fixed ring", c, pos)
			}
		}
		for i, m := range marks {
			if m.DY != -64 || m.Record != 48 || m.Phase != 4-(89-18*(i/4))/18 {
				t.Errorf("count %#x spark %d = %+v, want y -64, picture 48, frame %d", c, i, m, 4-(89-18*(i/4))/18)
			}
		}
	}
	a := EffectMarkRecords(MarkProgramSecondBless, 48, 0xffff, 1)
	b := EffectMarkRecords(MarkProgramSecondBless, 48, 0xfffa, 1)
	for i := range a {
		if a[i] != b[i] {
			t.Errorf("spark %d differs across one 5-update period: %+v then %+v", i, a[i], b[i])
		}
	}
}

// TestTheFirstGameMarkRecordsAreTheBases: every first-game kind, as its own
// program and picture, builds the records the base 46c4f6fe built, over the
// countdowns at both ends of the element's life and five scales.
func TestTheFirstGameMarkRecordsAreTheBases(t *testing.T) {
	const want = "750d9fe9b698d6f1f5d2fbe20029af1ff8482cbca3d37cec73545d4d0122d3e5"
	h := sha256.New()
	for kind := 0; kind <= 0x50; kind++ {
		for _, span := range [2][2]int{{0, 400}, {0xffff - 400, 0xffff}} {
			for c := span[0]; c <= span[1]; c++ {
				for tile := 0; tile <= 4; tile++ {
					for _, m := range EffectMarkRecords(kind, kind, uint16(c), tile) {
						fmt.Fprintf(h, "%d,%d,%d,%d,%d,%d,%d,%d;", kind, c, tile, m.DX, m.DY, m.Depth, m.Record, m.Phase)
					}
				}
			}
		}
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != want {
		t.Fatalf("first-game records digest %s, want the base's %s", got, want)
	}
}
