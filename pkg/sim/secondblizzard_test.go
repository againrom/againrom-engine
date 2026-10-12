package sim

import (
	"math"
	"testing"
)

// blizzardRow is the second game's row 7 with its stored parameters
// (R2-ENGINE-395): cost 100, Sphere 2, range 10, Distribution 5, Radius 4,
// area duration 8 and damage 10..30.
func blizzardRow(dmin, dmax int32) SpellRule {
	return secondRow(SpellRule{ID: 7, School: 2, ManaCost: 100, MaxRange: 10, Area: true,
		Distribution: distributionStaged, Radius: 4, AreaDuration: 8, DamageMin: dmin, DamageMax: dmax, Damaging: true})
}

func blizzardCaster(x, y int32) Entity {
	return Entity{ID: 1, X: x, Y: y, HP: 100, MaxHP: 100, Owner: 3, TokenSize: 1,
		Mind: 30, Mana: 500, MaxMana: 500, KnownSpells: 1 << 7, ScanRange: 19,
		AttackCharge: 4, AttackRelax: 2}
}

func blizzardWorld(t *testing.T, seed uint64, original bool, rule SpellRule, structures []Structure, ents ...Entity) *World {
	t.Helper()
	w, err := NewStructuredWorld(seed, Bounds{Width: 40, Height: 40}, ModeCanonical, Terrain{}, ents, nil,
		Relations{}, nil, nil, []SpellRule{rule}, GhostTemplate{}, structures)
	if err != nil {
		t.Fatal(err)
	}
	if original {
		w.rng = rng{state: uint64(uint32(seed * 2654435761)), original: true}
	}
	return w
}

// msvcRange is the claims' range call on the original stream, computed here
// apart from pkg/random: s = 214013*s + 2531011, r = (s>>16) & 32767 and
// floor(r*(n+1)/32768) (R2-ENGINE-391).
func msvcRange(state *uint32, n int64) int64 {
	*state = *state*214013 + 2531011
	return int64((*state>>16)&32767) * (n + 1) / 32768
}

// blizzardDraw takes one inclusive RNG(n) draw from a copy of w's stream:
// the claims' own arithmetic on an original stream, the engine's seeded draw
// otherwise.
func blizzardDraw(w *World, r *rng, n int64) int64 {
	if w.rng.original {
		s := uint32(r.state)
		v := msvcRange(&s, n)
		r.state = uint64(s)
		return v
	}
	return int64(r.uniform(int32(n)))
}

// blizzardLanding casts Blizzard at (x, y) and steps until the area stands. It
// returns the landing tick's report and the area's power.
func blizzardLanding(t *testing.T, w *World, x, y int32) (Report, uint16) {
	t.Helper()
	cmds := []Command{CastAt(1, 7, CellPoint{X: x, Y: y})}
	for range 60 {
		report := StepReported(w, cmds)
		cmds = nil
		if areas := w.CellEffects(); len(areas) == 1 {
			if areas[0].Spell != 7 || areas[0].Mode != AreaModeRing || areas[0].X != x || areas[0].Y != y {
				t.Fatalf("Blizzard stands as %+v", areas[0])
			}
			return report, areas[0].Power
		}
	}
	t.Fatal("Blizzard never landed")
	return Report{}, 0
}

// blizzardModel is the claims' loop: countdown +0x50 is decremented but its
// old value tested; an old zero resets it to 2 and attempts a pulse, and
// counter +0x4f counts attempts (R2-ENGINE-392).
type blizzardModel struct{ counter, countdown uint8 }

func (m *blizzardModel) update() bool {
	old := m.countdown
	m.countdown--
	if old != 0 {
		return false
	}
	m.countdown = 2
	m.counter++
	return true
}

// TestSecondBlizzardPulseScheduleAndCellDraws drives a real cast on both
// streams. The landing is area update 1; each later update is one tick. Every
// attempt takes exactly two draws, x then y, offsets -2..3, also when the cell
// falls outside 8..dimension-9; between attempts no draw is taken; the 32nd
// attempt on update 94 is the last, and the saved pulse counter and countdown
// follow the claims' loop on every update.
func TestSecondBlizzardPulseScheduleAndCellDraws(t *testing.T) {
	for _, mode := range []bool{false, true} {
		for _, centre := range [][2]int32{{20, 20}, {9, 30}} {
			w := blizzardWorld(t, 7, mode, blizzardRow(10, 30), nil, blizzardCaster(centre[0], centre[1]-8))
			report, _ := blizzardLanding(t, w, centre[0], centre[1])
			if len(report.AreaPaints) > 1 {
				t.Fatalf("landing painted %d stages", len(report.AreaPaints))
			}
			model := blizzardModel{}
			model.update()
			attempts, painted, rejected := 1, len(report.AreaPaints), 1-len(report.AreaPaints)
			seen := map[int64]bool{}
			for update := 2; update <= 100; update++ {
				states, err := w.NativeAreaSaveStates()
				if err != nil {
					t.Fatal(err)
				}
				if update <= 94 {
					if len(states) != 1 || states[0].Stage != model.counter || states[0].Remaining != uint16(model.countdown) {
						t.Fatalf("%v %v before update %d: saved %+v, want counter %d countdown %d", mode, centre, update, states, model.counter, model.countdown)
					}
				} else if len(states) != 0 {
					t.Fatalf("%v %v: the area stands at update %d", mode, centre, update)
				}
				r := w.rng
				pulse := update <= 94 && model.update()
				var x, y int64
				if pulse {
					dx, dy := blizzardDraw(w, &r, 5)-2, blizzardDraw(w, &r, 5)-2
					seen[dx], seen[dy] = true, true
					x, y = int64(uint8(int64(centre[0])+dx)), int64(uint8(int64(centre[1])+dy))
				}
				report := StepReported(w, nil)
				if w.rng != r {
					t.Fatalf("%v %v update %d: the stream moved off the claims' draws (pulse %v)", mode, centre, update, pulse)
				}
				if !pulse {
					if len(report.AreaPaints) != 0 {
						t.Fatalf("%v %v update %d painted with no attempt", mode, centre, update)
					}
					continue
				}
				attempts++
				if x < 8 || y < 8 || x > 31 || y > 31 {
					rejected++
					if len(report.AreaPaints) != 0 {
						t.Fatalf("%v %v update %d painted the rejected cell %d,%d", mode, centre, update, x, y)
					}
					continue
				}
				if len(report.AreaPaints) != 1 || len(report.AreaPaints[0].Cells) != 1 || report.AreaPaints[0].Spell != 7 ||
					report.AreaPaints[0].Cells[0] != (CellPoint{X: int32(x), Y: int32(y)}) {
					t.Fatalf("%v %v update %d painted %+v, want %d,%d", mode, centre, update, report.AreaPaints, x, y)
				}
				painted++
			}
			if attempts != 32 || len(seen) != 6 || !seen[-2] || !seen[3] {
				t.Fatalf("%v %v: %d attempts, offsets %v", mode, centre, attempts, seen)
			}
			if centre[0] == 9 && (rejected == 0 || painted+rejected != 32) || centre[0] == 20 && painted != 32 {
				t.Fatalf("%v %v: %d painted, %d rejected after the landing", mode, centre, painted, rejected)
			}
		}
	}
}

// blizzardPair is the claims' cached fill for book power P: f = 1 + P/30,
// B = uint8(trunc(dmin*f)), S = uint8(trunc(dmax*f - B)) (R2-ENGINE-393).
func blizzardPair(dmin, dmax int64, power uint16) (int64, int64) {
	f := 1 + float64(min(power, 255))/30
	b := int64(uint8(math.Trunc(float64(dmin) * f)))
	return b, int64(uint8(math.Trunc(float64(dmax)*f - float64(b))))
}

// TestSecondBlizzardDamagesEveryActorItStrikes fills the 6 by 6 footprint with
// actors, the caster among them, and checks each pulse against the claims:
// the struck actor loses max(trunc((B+U[0,S])*(100-p)/100 + 0.75), 0) for its
// water resistance p held to 0..100, with no hit roll and no caster
// exclusion, and the same actor again on a later pulse.
func TestSecondBlizzardDamagesEveryActorItStrikes(t *testing.T) {
	casterHit, repeated := false, false
	for _, mode := range []bool{false, true} {
		for seed := uint64(1); seed <= 4; seed++ {
			// Ownerless actors take no AI reaction to a blow, so the pulse is
			// the stream's only consumer.
			caster := blizzardCaster(18, 18)
			caster.Owner = SelfSlot
			ents := []Entity{caster}
			water := map[EntityID]int32{}
			for y := int32(18); y <= 23; y++ {
				for x := int32(18); x <= 23; x++ {
					if x == 18 && y == 18 {
						continue
					}
					id := EntityID(len(ents) + 1)
					a := Entity{ID: id, X: x, Y: y, HP: 5000, MaxHP: 5000, TokenSize: 1}
					a.Protection[1] = []int32{0, 40, 100, -20, 150}[int(id)%5]
					water[id] = a.Protection[1]
					ents = append(ents, a)
				}
			}
			w := blizzardWorld(t, seed, mode, blizzardRow(10, 30), nil, ents...)
			_, power := blizzardLanding(t, w, 20, 20)
			base, spread := blizzardPair(10, 30, power)
			model := blizzardModel{}
			model.update()
			hits := map[EntityID]int{}
			for update := 2; update <= 94; update++ {
				before := w.Entities()
				r := w.rng
				pulse := model.update()
				var struck EntityID
				var want int64
				if pulse {
					x := int32(20 + blizzardDraw(w, &r, 5) - 2)
					y := int32(20 + blizzardDraw(w, &r, 5) - 2)
					for _, e := range before {
						if e.X == x && e.Y == y {
							struck = e.ID
						}
					}
					p := int64(min(max(water[struck], 0), 100))
					v := base + blizzardDraw(w, &r, spread)
					want = max(int64(math.Trunc(float64(v*(100-p))/100+0.75)), 0)
				}
				StepReported(w, nil)
				if w.rng != r {
					t.Fatalf("%v seed %d update %d: the stream moved off the claims' draws", mode, seed, update)
				}
				for i, e := range w.Entities() {
					lost := int64(before[i].HP - e.HP)
					if e.ID == struck && lost != want || e.ID != struck && lost != 0 {
						t.Fatalf("%v seed %d update %d: actor %d lost %d, want %d (struck %d)", mode, seed, update, e.ID, lost, want, struck)
					}
				}
				if struck != 0 {
					hits[struck]++
				}
			}
			for id, n := range hits {
				casterHit = casterHit || id == 1
				repeated = repeated || n > 1
			}
		}
	}
	if !casterHit || !repeated {
		t.Fatalf("caster struck %v, an actor struck twice %v", casterHit, repeated)
	}
}

// TestSecondBlizzardStructureLosesTheRollLessFive checks a structure under
// the pulses: a positive spread takes max(B+U[0,S]-5, 0) from its health word
// per struck cell, and a zero spread takes nothing and draws nothing
// (R2-ENGINE-393).
func TestSecondBlizzardStructureLosesTheRollLessFive(t *testing.T) {
	for _, pair := range [][2]int32{{10, 30}, {10, 10}} {
		hall := Structure{ID: 1, Field42: 4000, MaxHealth: 4000, Col: 19, Row: 19, Width: 2, Height: 2, Attach: 15}
		w := blizzardWorld(t, 3, false, blizzardRow(pair[0], pair[1]), []Structure{hall}, blizzardCaster(20, 12))
		_, power := blizzardLanding(t, w, 20, 20)
		base, spread := blizzardPair(int64(pair[0]), int64(pair[1]), power)
		model := blizzardModel{}
		model.update()
		struck := 0
		for update := 2; update <= 94; update++ {
			before := w.structures[0].Field42
			r := w.rng
			var want int64
			if model.update() {
				x := 20 + blizzardDraw(w, &r, 5) - 2
				y := 20 + blizzardDraw(w, &r, 5) - 2
				if x >= 19 && x <= 20 && y >= 19 && y <= 20 {
					struck++
					if spread > 0 {
						want = max(base+blizzardDraw(w, &r, spread)-5, 0)
					}
				}
			}
			StepReported(w, nil)
			if w.rng != r || int64(before)-int64(w.structures[0].Field42) != want {
				t.Fatalf("pair %v update %d: structure %d -> %d, want a loss of %d", pair, update, before, w.structures[0].Field42, want)
			}
		}
		if struck == 0 {
			t.Fatalf("pair %v: no pulse struck the structure", pair)
		}
	}
}

// TestSampledRingCellsAdmitOnlyAStageCell holds a SAVE's carried cells of a
// sampled staged record to what one stage can paint: none or one cell, -2..3
// from the anchor, inside the 8..dimension-9 inset.
func TestSampledRingCellsAdmitOnlyAStageCell(t *testing.T) {
	w := blizzardWorld(t, 1, false, blizzardRow(10, 30), nil, blizzardCaster(20, 12))
	e := cellEffect{Key: cellKey(20, 20), Spell: 7, Mode: areaModeRing}
	for _, tc := range []struct {
		cells []uint16
		want  bool
	}{
		{nil, true},
		{[]uint16{cellKey(18, 18)}, true},
		{[]uint16{cellKey(23, 23)}, true},
		{[]uint16{cellKey(17, 20)}, false},
		{[]uint16{cellKey(20, 24)}, false},
		{[]uint16{cellKey(18, 18), cellKey(19, 19)}, false},
	} {
		if got := w.sampledRingCells(e, tc.cells); got != tc.want {
			t.Errorf("cells %04x admitted %v, want %v", tc.cells, got, tc.want)
		}
	}
	edge := cellEffect{Key: cellKey(9, 30), Spell: 7, Mode: areaModeRing}
	if w.sampledRingCells(edge, []uint16{cellKey(7, 30)}) || w.sampledRingCells(edge, []uint16{cellKey(9, 32)}) || !w.sampledRingCells(edge, []uint16{cellKey(8, 31)}) {
		t.Error("the inset 8..dimension-9 does not bound the carried cell")
	}
}
