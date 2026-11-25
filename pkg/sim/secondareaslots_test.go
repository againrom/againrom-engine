package sim

import (
	"slices"
	"testing"
)

func TestSecondBlizzardStrikesTheGroundAirAndStructureSlotsTogether(t *testing.T) {
	struck := map[string]int{}
	for seed := uint64(1); seed <= 8; seed++ {
		ground := Entity{ID: 2, X: 20, Y: 20, HP: 5000, MaxHP: 5000, TokenSize: 1}
		air := Entity{ID: 3, X: 20, Y: 20, HP: 5000, MaxHP: 5000, TokenSize: 1, Domain: DomainAir}
		ghost := Entity{ID: 4, X: 19, Y: 19, HP: 5000, MaxHP: 5000, TokenSize: 1, Domain: DomainGhost}
		hall := Structure{ID: 1, Field42: 4000, MaxHealth: 4000, Col: 19, Row: 19, Width: 2, Height: 2, Attach: 15}
		w := blizzardWorld(t, true, seed, false, blizzardRow(10, 30), []Structure{hall}, blizzardCaster(20, 12), ground, air, ghost)
		blizzardLanding(t, w, 20, 20)
		for update := 2; update <= 94; update++ {
			before, hp := w.Entities(), w.structures[0].Field42
			StepReported(w, nil)
			lost := func(i int) bool { return w.Entities()[i].HP < before[i].HP }
			g, a, h, s := lost(1), lost(2), lost(3), w.structures[0].Field42 < hp
			if g != a || (g || h) && !s {
				t.Fatalf("seed %d update %d: ground %v, air %v, ghost %v, hall %v", seed, update, g, a, h, s)
			}
			for name, hit := range map[string]bool{"ground": g, "air": a, "ghost": h, "hall": s} {
				if hit {
					struck[name]++
				}
			}
		}
	}
	for _, name := range []string{"ground", "air", "ghost", "hall"} {
		if struck[name] == 0 {
			t.Errorf("no pulse struck the %s occupant in eight casts", name)
		}
	}
}

func TestLightReachesOnlyTheGroundSlotInBothGames(t *testing.T) {
	for _, second := range []bool{false, true} {
		rule := darkSightRow(second, false)
		ghost, air := darkSightCreature(3, 40, 41), darkSightCreature(4, 41, 40)
		ghost.Domain, air.Domain = DomainGhost, DomainAir
		w := darkSightWorld(t, second, rule, darkSightCreature(2, 40, 40), ghost, air)
		if !w.landArea(rule, 20, 1, true, 2, 2, 40, 40, nil) {
			t.Fatal("fixture: the cloud did not land")
		}
		reached := map[EntityID]bool{}
		for k := 0; k < 48; k++ {
			Step(w, nil)
			for _, id := range []EntityID{2, 3, 4} {
				reached[id] = reached[id] || scanEffectOn(w, id)
			}
		}
		if !reached[2] || !reached[3] || reached[4] {
			t.Errorf("second %v: ground %v, ghost %v, air %v; want only the ground slot reached", second, reached[2], reached[3], reached[4])
		}
		if got := w.Entities()[3].ScanRange; got != 8 {
			t.Errorf("second %v: the air creature's sight is %d, want 8", second, got)
		}
	}
}

func wallOfFireRow(second bool) SpellRule {
	r := SpellRule{ID: 3, School: 1, Area: true, Distribution: distributionWall, AreaDuration: 10,
		DamageMin: 2, DamageMax: 4, Damaging: true, TargetsUnit: true}
	if second {
		return secondRow(r)
	}
	return r
}

func TestWallOfFireBurnsOnlyTheGroundSlotOnItsPulseCount(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, power := range []uint16{0, 9, 10, 100, 255} {
			ground := Entity{ID: 2, X: 20, Y: 20, HP: 5000, MaxHP: 5000, TokenSize: 1}
			air := Entity{ID: 3, X: 20, Y: 20, HP: 5000, MaxHP: 5000, TokenSize: 1, Domain: DomainAir}
			hall := Structure{ID: 1, Field42: 4000, MaxHealth: 4000, Col: 21, Row: 20, Width: 2, Height: 1, Attach: 15}
			w := blizzardWorld(t, second, 5, false, wallOfFireRow(second), []Structure{hall}, ground, air)
			if !w.landArea(w.spells[0], power, 0, false, 20, 12, 20, 20, nil) {
				t.Fatalf("second %v power %d: the wall did not land", second, power)
			}
			cells := w.CellEffects()[0].Cells
			if len(cells) != 10 || !slices.Contains(cells, [2]int32{21, 20}) {
				t.Fatalf("second %v power %d: the wall paints %v, want ten cells over the hall", second, power, cells)
			}
			pulses := 0
			for k := 0; k < 16*40 && len(w.effects) != 0; k++ {
				before := w.Entities()[0].HP
				Step(w, nil)
				if w.Entities()[0].HP < before {
					pulses++
				}
			}
			want := 10 + (int(power)*16/10+15)/16
			if pulses != want || w.Entities()[1].HP != 5000 || w.structures[0].Field42 != 4000 {
				t.Errorf("second %v power %d: %d pulses (want %d), air %d, hall %d", second, power, pulses, want,
					w.Entities()[1].HP, w.structures[0].Field42)
			}
		}
	}
}

func TestSecondAreaShapesHoldThePublishedCellCounts(t *testing.T) {
	w := blizzardWorld(t, true, 1, false, wallOfFireRow(true), nil)
	if n := len(w.diamondCells(20, 20, 2)); n != 21 {
		t.Errorf("radius 2 diamond: %d cells, want 21", n)
	}
	if n := len(w.diamondCells(20, 20, 4)); n != 57 {
		t.Errorf("radius 4 diamond: %d cells, want 57", n)
	}
	if n := len(w.blastCellsInMap(20, 20, 1)); n != 9 {
		t.Errorf("Fire Ball square: %d cells, want 9", n)
	}
	for d := uint8(0); d < 8; d++ {
		want, cone := 10, 25
		if wallDirs[d].diagonal {
			want = 9
		}
		if d&1 != 0 {
			cone = 21
		}
		if n := len(w.wallCells(20, 20, d)); n != want {
			t.Errorf("facing %d: wall of %d cells, want %d", d, n, want)
		}
		total := 0
		for stage := 0; stage < 6; stage++ {
			total += len(acidStreamStage(w, cellEffect{Direction: d}, stage))
		}
		if total != cone {
			t.Errorf("facing %d: Acid Stream covers %d cells, want %d", d, total, cone)
		}
	}
}

func TestAcidStreamStraightPulseTwoFollowsTheBoundRows(t *testing.T) {
	w := blizzardWorld(t, true, 1, false, wallOfFireRow(true), nil)
	strikes := func(w *World) map[[2]int32]int {
		got := map[[2]int32]int{}
		for _, c := range acidStreamStage(w, cellEffect{Direction: 0}, 2) {
			got[c]++
		}
		return got
	}
	ru := strikes(w)
	for x := int32(-2); x <= 2; x++ {
		if ru[[2]int32{x, -2}] != 1 || len(ru) != 5 {
			t.Fatalf("unbound rows: %v, want each of x -2..2 once", ru)
		}
	}
	w.SetStraightCone(StraightCone{2: {-1, -1, 0, 1, 2}})
	en := strikes(w)
	if en[[2]int32{-1, -2}] != 2 || en[[2]int32{-2, -2}] != 0 || len(en) != 4 {
		t.Fatalf("bound rows: %v, want x -1 twice, 0, 1, 2 once and no -2", en)
	}
	if n := len(acidStreamStage(w, cellEffect{Direction: 0}, 3)); n != 7 {
		t.Errorf("a row left nil keeps the symmetric wedge: %d cells, want 7", n)
	}
}
