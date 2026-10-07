package terrain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
)

// TestTheFourProtectionsFormTheDiamond — MAGIC-PROT-062: one record each, all
// four `TileSize*32` above the anchor, all four at depth 0, and 6 pixels left,
// right, up and down of one another. The record index is the kind and the phase
// is `countdown mod 6`.
func TestTheFourProtectionsFormTheDiamond(t *testing.T) {
	const tile, countdown = 2, 65533
	base := tile * 32
	want := map[int][2]int{
		MarkProtectionFire:  {-6, -base},
		MarkProtectionWater: {6, -base},
		MarkProtectionAir:   {0, -base - 6},
		MarkProtectionEarth: {0, -base + 6},
	}
	for kind, offset := range want {
		got := EffectMarkRecords(kind, countdown, tile)
		if len(got) != 1 {
			t.Fatalf("kind %#x built %d records, want 1", kind, len(got))
		}
		m := got[0]
		if m.DX != offset[0] || m.DY != offset[1] {
			t.Errorf("kind %#x placed at (%d,%d), want (%d,%d)", kind, m.DX, m.DY, offset[0], offset[1])
		}
		if m.Depth != 0 {
			t.Errorf("kind %#x carries depth %d, want 0 — all four draw after the actor sprite", kind, m.Depth)
		}
		if m.Record != kind {
			t.Errorf("kind %#x wrote record index %#x, want its own kind", kind, m.Record)
		}
		if m.Phase != countdown%6 {
			t.Errorf("kind %#x phase %d, want countdown mod 6 = %d", kind, m.Phase, countdown%6)
		}
	}
}

// TestPoisonCloudPutsOneMarkAboveTheActor — MAGIC-CLOUD-065's first arm.
func TestPoisonCloudPutsOneMarkAboveTheActor(t *testing.T) {
	got := EffectMarkRecords(MarkPoisonCloud, 100, 1)
	if len(got) != 1 {
		t.Fatalf("built %d records, want 1", len(got))
	}
	want := EffectMark{DY: -32, Record: MarkPoisonCloud, Phase: 100 % 6}
	if got[0] != want {
		t.Errorf("record = %+v, want %+v", got[0], want)
	}
}

// TestTheNonMarkingSpellsBuildNothing — MAGIC-MARK-061's negative half: the 35
// kinds in range that reach the arm appending nothing, and the five whose kind
// falls outside the dispatch range entirely. `stone_curse` and `invisibility`
// are among them: MAGIC-ACTOR-066 establishes that they change the actor's own
// draw instead, which is not this function's business.
func TestTheNonMarkingSpellsBuildNothing(t *testing.T) {
	// freezing_cloud, light, lightning, prismatic_spray, acid_stream,
	// invisibility, darkness, wall_of_earth, stone_curse, meteor_storm, haste,
	// control_spirit, teleport — in range and unmarked.
	inRange := []int{0x16, 0x20, 0x22, 0x24, 0x1a, 0x26, 0x2a, 0x2e, 0x30, 0x32, 0x38, 0x3a, 0x3c}
	// slow, fire_arrow, fire_ball, wall_of_fire, fire_sacrifice — outside it.
	outside := []int{0x40, 0x0a, 0x0c, 0x0e, 0x10}
	for _, kind := range append(inRange, outside...) {
		if got := EffectMarkRecords(kind, 500, 1); len(got) != 0 {
			t.Errorf("kind %#x built %d records, want none", kind, len(got))
		}
	}
	// And the marking set, for the same sweep's control: each builds at least
	// one record, so the test above is discriminating and not vacuous.
	for _, kind := range []int{MarkProtectionFire, MarkProtectionWater, MarkProtectionAir,
		MarkProtectionEarth, MarkPoisonCloud, MarkShield, MarkBless, MarkCurse} {
		if got := EffectMarkRecords(kind, 500, 1); len(got) == 0 {
			t.Errorf("kind %#x built no records, want at least one", kind)
		}
	}
}

// TestBlessAndCurseSweepOppositeWays — MAGIC-BLESS-064: twenty marks each and
// the sine carried in depth so the ring splits around the actor. Frame animation
// is covered separately because it is owner-authored rather than part of the
// decoded geometry.
func TestBlessAndCurseSweepOppositeWays(t *testing.T) {
	for _, tc := range []struct{ kind int }{{MarkBless}, {MarkCurse}} {
		got := EffectMarkRecords(tc.kind, 300, 1)
		if len(got) != 20 {
			t.Fatalf("kind %#x built %d records, want 20", tc.kind, len(got))
		}
		for step := 0; step < 5; step++ {
			for i := 0; i < 4; i++ {
				m := got[step*4+i]
				if m.Phase < 0 || m.Phase > 4 {
					t.Errorf("kind %#x step %d record %d frame %d, want 0..4",
						tc.kind, step, i, m.Phase)
				}
				if m.Record != tc.kind {
					t.Errorf("kind %#x wrote record index %#x", tc.kind, m.Record)
				}
				if m.DY != -32 {
					t.Errorf("kind %#x dy %d, want the -TileSize*32 base", tc.kind, m.DY)
				}
			}
		}
		var above, below int
		for _, m := range got {
			if m.Depth > 0 {
				above++
			} else {
				below++
			}
		}
		if above == 0 || below == 0 {
			t.Errorf("kind %#x put %d records behind the actor and %d in front; the ring must split",
				tc.kind, above, below)
		}
	}
}

func TestEveryBlessStarAnimatesFromItsOwnStartingFrame(t *testing.T) {
	start := EffectMarkRecords(MarkBless, 0xffff, 1)
	next := EffectMarkRecords(MarkBless, 0xfffe, 1)
	if len(start) != 20 || len(next) != 20 {
		t.Fatalf("Bless built %d then %d stars, want 20 on both ticks", len(start), len(next))
	}
	wantFirstEight := [8]int{3, 2, 0, 3, 4, 3, 0, 1}
	for i := range wantFirstEight {
		if start[i].Phase != wantFirstEight[i] {
			t.Errorf("star %d starts on frame %d, want the fixed presentation phase %d",
				i, start[i].Phase, wantFirstEight[i])
		}
	}
	for i := range start {
		if next[i].Phase != (start[i].Phase+1)%5 {
			t.Errorf("star %d advanced %d -> %d, want one shared tick step",
				i, start[i].Phase, next[i].Phase)
		}
	}
}

// TestShieldUsesTheTwoExactDecodedComponents pins MAGIC-091 and corrected
// MAGIC-092's ordered producer results without deriving counts or hashes from
// the implementation.
func TestShieldUsesTheTwoExactDecodedComponents(t *testing.T) {
	cases := []struct {
		tile, phase, envelope, count int
		hash                         string
	}{
		{1, 0, 28, 48, "21960277b6b1d5d9ca037cb51184c738dd9ef3d643aba963b69088ada85b529e"},
		{1, 45, 0, 72, "5d77e701e6a73726939e1337a110acc5f2e4ee283f2d4dc9aa2a30c12da9d6b6"},
		{3, 15, 56, 336, "4427ea4296ccc5322eaf8e2ade9afabff036f12c62bc41cdeb621c10352ff199"},
		{3, 30, 27, 224, "4c358c15bfe73930cf3edc9aec20cb44588363e065de229ceaf522f42879e60d"},
		{3, 60, 28, 224, "7ebdb38f9376965cbb3253a34b6aa4888232c411cf3713b44f0dfc1b39ab624c"},
	}
	for _, tc := range cases {
		got := EffectMarkRecords(MarkShield, uint16(tc.phase), tc.tile)
		if len(got) != tc.count {
			t.Errorf("T=%d p=%d records=%d, want %d", tc.tile, tc.phase, len(got), tc.count)
		}
		if e := shieldEnvelopePC53(tc.tile, tc.phase); e != tc.envelope {
			t.Errorf("T=%d p=%d envelope=%d, want %d", tc.tile, tc.phase, e, tc.envelope)
		}
		if hash := effectMarkHash(got); hash != tc.hash {
			t.Errorf("T=%d p=%d ordered hash=%s, want %s", tc.tile, tc.phase, hash, tc.hash)
		}
	}
}

// TestShieldComponentOrderAndZeroRadius are the two structural discriminators:
// every fixed-radius A record precedes B, and phase zero emits no B record
// instead of drawing frame -1.
func TestShieldComponentOrderAndZeroRadius(t *testing.T) {
	zero := EffectMarkRecords(MarkShield, 0, 1)
	if len(zero) != 48 {
		t.Fatalf("phase zero records=%d, want Component A's exact 48 and no Component B", len(zero))
	}
	wantA := EffectMark{DX: 0, DY: -28, Depth: 11, Record: MarkShield, Phase: 1}
	if zero[0] != wantA || zero[1] != wantA {
		t.Fatalf("first sampled A pair=%+v %+v, want duplicate %+v", zero[0], zero[1], wantA)
	}

	flat := EffectMarkRecords(MarkShield, 45, 1)
	if len(flat) != 72 {
		t.Fatalf("phase 45 records=%d, want 48 A followed by 24 B", len(flat))
	}
	wantB := EffectMark{DX: 0, DY: -11, Depth: 16, Record: MarkShield, Phase: 4}
	if flat[48] != wantB || flat[49] != wantB {
		t.Fatalf("first B pair=%+v %+v, want duplicate %+v after all A records", flat[48], flat[49], wantB)
	}
}

func effectMarkHash(marks []EffectMark) string {
	h := sha256.New()
	for _, m := range marks {
		fmt.Fprintf(h, "%d,%d,%d,%d,%d;", m.DX, m.DY, m.Depth, m.Record, m.Phase)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// TestTileSizeScalesEveryOffset — MAGIC-MARK-059: the vertical base is the
// class's own TileSize key times 32, so every mark offset scales with the
// actor's footprint. A TileSize below one is treated as one, the key's own
// registry default.
func TestTileSizeScalesEveryOffset(t *testing.T) {
	one := EffectMarkRecords(MarkPoisonCloud, 0, 1)[0]
	three := EffectMarkRecords(MarkPoisonCloud, 0, 3)[0]
	if three.DY != 3*one.DY {
		t.Errorf("TileSize 3 put the mark at dy %d, want three times TileSize 1's %d", three.DY, one.DY)
	}
	if zero := EffectMarkRecords(MarkPoisonCloud, 0, 0)[0]; zero != one {
		t.Errorf("TileSize 0 built %+v, want TileSize 1's %+v", zero, one)
	}
}

// TestMarkKindIsTheEvenPictureID — MAGIC-MARK-061: `kind = 2*spellId + 8`, the
// even half of MAGIC-PIC-026's pair, checked at the ten marking spells.
func TestMarkKindIsTheEvenPictureID(t *testing.T) {
	want := map[int]int{5: MarkProtectionFire, 6: MarkHeal, 8: MarkPoisonCloud,
		10: MarkProtectionWater, 11: MarkDrainLife, 15: MarkInvisibility,
		16: MarkProtectionAir, 18: MarkShield, 20: MarkStoneCurse,
		22: MarkProtectionEarth, 23: MarkBless, 27: MarkCurse}
	for spell, kind := range want {
		if got := MarkKind(spell); got != kind {
			t.Errorf("MarkKind(%d) = %#x, want %#x", spell, got, kind)
		}
	}
}
