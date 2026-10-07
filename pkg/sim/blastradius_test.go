package sim

import (
	"slices"
	"testing"
	"time"
)

func blastWorld(t *testing.T, width, height int32, rule SpellRule, ents ...Entity) *World {
	t.Helper()
	w, err := NewStockedSpelledWorld(7, Bounds{Width: width, Height: height}, ModeCanonical,
		Terrain{}, ents, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("world: %v", err)
	}
	w.SetBurstPhases(11)
	return w
}

func blastRule(radius uint8) SpellRule {
	return SpellRule{ID: fireBallSpell, Area: true, Distribution: distributionDiamond,
		Radius: radius, School: 1, DamageMin: 10, DamageMax: 10, Damaging: true}
}

func wantBlastCells(width, height, cx, cy, r int32) []uint16 {
	var out []uint16
	for x := int32(0); x < width; x++ {
		for y := int32(0); y < height; y++ {
			if x >= cx-r && x <= cx+r && y >= cy-r && y <= cy+r {
				out = append(out, cellKey(x, y))
			}
		}
	}
	return out
}

func TestBlastCellsInMapAreTheCoveredSquareClippedToTheMap(t *testing.T) {
	for _, size := range [][2]int32{{16, 16}, {20, 9}, {256, 256}, {130, 200}} {
		w := blastWorld(t, size[0], size[1], blastRule(1))
		for _, at := range [][2]int32{{0, 0}, {size[0] - 1, size[1] - 1}, {size[0] / 2, size[1] / 2}, {3, size[1] - 2}} {
			for _, r := range []int32{0, 1, 2, 5, 17, 100, 255} {
				got := w.blastCellsInMap(at[0], at[1], r)
				want := wantBlastCells(size[0], size[1], at[0], at[1], r)
				if 2*r+1 <= 256 {
					want = blastCells(at[0], at[1], r)
				}
				if !slices.Equal(got, want) {
					t.Fatalf("map %v anchor %v radius %d: %d cells, want %d", size, at, r, len(got), len(want))
				}
			}
		}
	}
}

func TestBlastCellsInMapEqualTheByteProducerAwayFromTheEdge(t *testing.T) {
	w := blastWorld(t, 64, 64, blastRule(1))
	for _, r := range []int32{0, 1, 2, 7} {
		if got, want := w.blastCellsInMap(20, 30, r), blastCells(20, 30, r); !slices.Equal(got, want) {
			t.Fatalf("radius %d: %x want %x", r, got, want)
		}
	}
}

func burstCells(w *World) [][2]int32 {
	var out [][2]int32
	for _, p := range w.SavedProjectiles().Items {
		if p.Picture == fireBallBurstPicture() {
			out = append(out, [2]int32{p.X / 256, p.Y / 256})
		}
	}
	return out
}

func TestBlastDrawsOneBurstForTheInstalledRadiusAndOnePerTileBeyondIt(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<fireBallSpell)
	for _, tc := range []struct {
		radius uint8
		want   int
	}{{0, 1}, {1, 1}, {2, 9}, {4, 9}, {5, 25}} {
		rule := blastRule(tc.radius)
		w := blastWorld(t, 32, 32, rule, caster)
		w.landAreaFacing(rule, 0, 1, true, 2, 2, 16, 16, 0, false, nil)
		got := burstCells(w)
		if len(got) != tc.want {
			t.Fatalf("radius %d: %d bursts %v, want %d", tc.radius, len(got), got, tc.want)
		}
		if tc.want == 1 && got[0] != [2]int32{16, 16} {
			t.Fatalf("radius %d: burst at %v, want the anchor", tc.radius, got[0])
		}
	}
}

func TestBlastBurstsCoverTheClippedSquareAndStayBounded(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<fireBallSpell)
	for _, at := range [][2]int32{{0, 0}, {5, 11}, {15, 15}, {8, 8}} {
		for _, r := range []uint8{3, 6, 9, 20, 255} {
			rule := blastRule(r)
			w := blastWorld(t, 16, 16, rule, caster)
			w.landAreaFacing(rule, 0, 1, true, 2, 2, at[0], at[1], 0, false, nil)
			got := burstCells(w)
			if len(got) > 36 {
				t.Fatalf("anchor %v radius %d: %d bursts exceed the 36 tiles of the map", at, r, len(got))
			}
			for _, b := range got {
				if b[0] < 0 || b[1] < 0 || b[0] >= 16 || b[1] >= 16 {
					t.Fatalf("burst off the map: %v", b)
				}
			}
			for _, key := range wantBlastCells(16, 16, at[0], at[1], int32(r)) {
				x, y := keyCell(key)
				covered := false
				for _, b := range got {
					if abs32(b[0]-x) <= 1 && abs32(b[1]-y) <= 1 {
						covered = true
					}
				}
				if !covered {
					t.Fatalf("anchor %v radius %d: cell (%d,%d) is under no burst %v", at, r, x, y, got)
				}
			}
		}
	}
}

func TestBlastScorchesExactlyTheCoveredCells(t *testing.T) {
	caster := effectMage(1, 2, 2, 1<<fireBallSpell)
	for _, at := range [][2]int32{{0, 0}, {12, 3}, {8, 8}} {
		for _, r := range []uint8{1, 4, 255} {
			rule := blastRule(r)
			w := blastWorld(t, 16, 16, rule, caster)
			w.landAreaFacing(rule, 0, 1, true, 2, 2, at[0], at[1], 0, false, nil)
			want := wantBlastCells(16, 16, at[0], at[1], int32(r))
			slices.Sort(want)
			if got := w.ScorchedCells(); !slices.Equal(got, want) {
				t.Fatalf("anchor %v radius %d: %d scorched, want %d", at, r, len(got), len(want))
			}
		}
	}
}

func TestBlastHitsCasterAndAlliesOncePerCoveredCell(t *testing.T) {
	caster := effectMage(1, 6, 6, 1<<fireBallSpell)
	ally := spEnt(2, 7, 6)
	ally.Owner = SelfSlot
	far := spEnt(3, 14, 14)
	foe := spEnt(4, 5, 5)
	foe.Owner = 3
	rule := blastRule(1)
	w := blastWorld(t, 16, 16, rule, caster, ally, far, foe)
	w.landAreaFacing(rule, 0, 1, true, 6, 6, 6, 6, 0, false, nil)
	hp := func(id EntityID) int32 { e, _ := w.Entity(id); return e.HP }
	if hp(1) != 90 || hp(2) != 90 || hp(4) != 90 || hp(3) != 100 {
		t.Fatalf("hp caster %d ally %d foe %d far %d", hp(1), hp(2), hp(4), hp(3))
	}

	rule = blastRule(255)
	caster = effectMage(1, 6, 6, 1<<fireBallSpell)
	w = blastWorld(t, 16, 16, rule, caster, spEnt(2, 7, 6), spEnt(3, 14, 14))
	w.landAreaFacing(rule, 0, 1, true, 6, 6, 6, 6, 0, false, nil)
	for _, id := range []EntityID{1, 2, 3} {
		if got := hp(id); got != 90 {
			t.Fatalf("radius 255: unit %d hp %d, want one hit", id, got)
		}
	}
}

func TestBlastAtTheWidestRadiusOnTheLargestMapIsBounded(t *testing.T) {
	caster := effectMage(1, 100, 100, 1<<fireBallSpell)
	rule := blastRule(255)
	var ents []Entity
	ents = append(ents, caster)
	for i := int32(0); i < 200; i++ {
		ents = append(ents, spEnt(EntityID(2+i), (i*7)%256, (i*13)%256))
	}
	w := blastWorld(t, 256, 256, rule, ents...)
	start := time.Now()
	w.landAreaFacing(rule, 0, 1, true, 100, 100, 128, 128, 0, false, nil)
	took := time.Since(start)
	t.Logf("radius 255 on 256x256 with 201 units: %v, %d bursts, %d scorched", took, len(burstCells(w)), len(w.ScorchedCells()))
	if n := len(burstCells(w)); n > 86*86 {
		t.Fatalf("%d bursts", n)
	}
	if len(w.ScorchedCells()) != 256*256 {
		t.Fatalf("scorched %d", len(w.ScorchedCells()))
	}
	if took > 5*time.Second {
		t.Fatalf("cast took %v", took)
	}
}

func TestWideBlastSurvivesTheByteFormInFlightAndDuringItsBursts(t *testing.T) {
	w, r := burstWorld(t)
	r.Radius = 255
	if !w.landArea(r, 0, 1, true, 1, 1, 8, 8, nil) {
		t.Fatal("area preparation")
	}
	for range 4 {
		Step(w, nil)
	}
	inFlight := requireSpellGraphBinary(t, w)
	for range 18 {
		Step(w, nil)
	}
	during := requireSpellGraphBinary(t, w)
	if n := len(burstCells(w)); n == 0 || n > 36 {
		t.Fatalf("%d bursts mid-life", n)
	}
	for range 18 {
		Step(inFlight, nil)
	}
	same := func(a, b *World) bool {
		return slices.Equal(burstCells(a), burstCells(b)) && slices.Equal(a.ScorchedCells(), b.ScorchedCells()) &&
			slices.Equal(a.entities, b.entities)
	}
	if w.Hash() != during.Hash() || !same(w, inFlight) {
		t.Fatal("a LOAD diverged before the bursts ended")
	}
	for k := range 30 {
		Step(w, nil)
		Step(inFlight, nil)
		Step(during, nil)
		if w.Hash() != during.Hash() || !same(w, inFlight) {
			t.Fatalf("a LOAD diverged %d ticks later", k+1)
		}
	}
	if len(w.SavedProjectiles().Items) != 0 || len(inFlight.ScorchedCells()) != 256 {
		t.Fatalf("bursts left %d, scorched %d", len(w.SavedProjectiles().Items), len(inFlight.ScorchedCells()))
	}
}

func TestInstalledRadiusBlastOnA256MapWrapsAsTheByteWalkDoes(t *testing.T) {
	for _, at := range [][2]int32{{0, 0}, {255, 255}, {0, 5}, {8, 8}, {255, 100}} {
		caster := effectMage(1, 100, 100, 1<<fireBallSpell)
		opposite := spEnt(2, (at[0]+255)%256, (at[1]+255)%256)
		rule := blastRule(1)
		w := blastWorld(t, 256, 256, rule, caster, opposite)
		w.landAreaFacing(rule, 0, 1, true, 100, 100, at[0], at[1], 0, false, nil)
		var want []uint16
		for _, key := range blastCells(at[0], at[1], 1) {
			x, y := keyCell(key)
			if x < 256 && y < 256 {
				want = append(want, key)
			}
		}
		slices.Sort(want)
		want = slices.Compact(want)
		if got := w.ScorchedCells(); !slices.Equal(got, want) {
			t.Fatalf("anchor %v: scorched %x, want the byte walk's %x", at, got, want)
		}
		hurt := slices.Contains(want, cellKey(opposite.X, opposite.Y))
		e, _ := w.Entity(2)
		if hurt != (e.HP < 100) {
			t.Fatalf("anchor %v: opposite-edge unit hp %d, wrapped neighbour expected %v", at, e.HP, hurt)
		}
		if n := len(burstCells(w)); n != 1 {
			t.Fatalf("anchor %v: %d bursts", at, n)
		}
	}
}
