package sim

import "testing"

// darkSightRow is the Darkness (dark) or Light cloud row of either game in
// the shipped durations: the first game's ids 17 and 12, area column 20 and
// the effect counter 16 its loader makes of `duration 1` (MAGIC-398); the
// second game's 14 and 15 on arms 17 and 12, area column 15 (R2-ENGINE-024)
// and the raw `duration 1` (R2-ASSET-178).
func darkSightRow(second, dark bool) SpellRule {
	id := uint16(12)
	switch {
	case dark && second:
		id = 14
	case dark:
		id = 17
	case second:
		id = 15
	}
	rows := []SpellRule{{ID: id, Area: true, Distribution: distributionDiamond, Radius: 2, AreaDuration: 20,
		TargetsUnit: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 16}}
	if second {
		AssignSecondGameArms(rows)
		rows[0].AreaDuration, rows[0].EffectDuration = 15, 1
	}
	return rows[0]
}

// darkSightWorld is a 64x64 flat world of either game holding a caster of
// owner 2 at (2,2) and the bodies es, with rule its one spell row.
func darkSightWorld(t *testing.T, second bool, rule SpellRule, es ...Entity) *World {
	t.Helper()
	caster := effectMage(1, 2, 2, 1<<rule.ID)
	caster.Owner = 2
	var script *Script
	if second {
		script = secondGameScript(t, nil, nil, nil)
	}
	w, err := NewStockedSpelledWorld(1, Bounds{Width: 64, Height: 64}, ModeCanonical, Terrain{},
		append([]Entity{caster}, es...), script, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	if playsSecondGame(w) != second {
		t.Fatalf("fixture: second game %v, want %v", playsSecondGame(w), second)
	}
	return w
}

// darkSightHero is the player's Human at (40,40): Mind 25, Reaction 36 and a
// whole-cell sight of 6, the sight word 1648.
func darkSightHero(id EntityID) Entity {
	e := fogWalkHero(40, 40, 1)
	e.ID, e.Owner, e.HP, e.MaxHP = id, SelfSlot, 200, 200
	return e
}

func scanEffectOn(w *World, id EntityID) bool {
	for _, a := range w.attached {
		if a.Target == id && a.Kind == EffectScanRange {
			return true
		}
	}
	return false
}

// landSightCloud lands rule's cloud over (40,40) at power, steps until the
// cloud's effect stands on id, and returns that tick count.
func landSightCloud(t *testing.T, w *World, rule SpellRule, power uint16, id EntityID) int {
	t.Helper()
	if !w.landArea(rule, power, 1, true, 2, 2, 40, 40, nil) {
		t.Fatal("fixture: the cloud did not land")
	}
	for k := 0; k < 256; k++ {
		if scanEffectOn(w, id) {
			return k
		}
		Step(w, nil)
	}
	t.Fatal("the cloud's effect never stood on the body")
	return 0
}

// waitSightCloudOut steps until no scan-range effect stands on id and no
// cloud remains.
func waitSightCloudOut(t *testing.T, w *World, id EntityID) {
	t.Helper()
	for k := 0; k < 20000; k++ {
		if !scanEffectOn(w, id) && len(w.effects) == 0 {
			return
		}
		Step(w, nil)
	}
	t.Fatal("the cloud or its effect never ended")
}

// A Darkness cloud over the player's own unit shrinks the sight its owner's
// fog stamp lights, and the count returns when the effect ends; a Light cloud
// grows it. Both games, at the powers of an early and a late campaign mage.
func TestADarknessCloudShrinksAndLightGrowsThePlayersSight(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			for _, power := range []uint16{0, 20, 60, 110} {
				rule := darkSightRow(second, dark)
				w := darkSightWorld(t, second, rule, darkSightHero(2))
				before := litCount(w.Sight(SelfSlot))
				scanBefore := laEnt(t, w, 2).ScanRange
				landSightCloud(t, w, rule, power, 2)
				under := litCount(w.Sight(SelfSlot))
				hero := laEnt(t, w, 2)
				t.Logf("second %v dark %v power %d: scan %d -> %d, cells %d -> %d",
					second, dark, power, scanBefore, hero.ScanRange, before, under)
				if hero.X != 40 || hero.Y != 40 {
					t.Fatalf("fixture: the hero moved to (%d,%d)", hero.X, hero.Y)
				}
				if dark && under >= before || !dark && under <= before {
					t.Errorf("second %v dark %v power %d: the player's sight lights %d cells under the cloud, %d before",
						second, dark, power, under, before)
				}
				waitSightCloudOut(t, w, 2)
				if after := litCount(w.Sight(SelfSlot)); after != before || laEnt(t, w, 2).ScanRange != scanBefore {
					t.Errorf("second %v dark %v power %d: after the effect the sight lights %d cells at scan %d, want %d at %d",
						second, dark, power, after, laEnt(t, w, 2).ScanRange, before, scanBefore)
				}
			}
		}
	}
}

// A computer unit in a Darkness cloud marches its target sweep at the shorter
// scan range: a cell at its whole sight drops out of the candidate stamp and
// returns when the effect ends. Both games.
func TestAComputerUnitInDarknessSweepsAShorterRange(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		rule := darkSightRow(second, true)
		guard := engFighter(2, 3, 40, 40)
		guard.ScanRange = 6
		w := darkSightWorld(t, second, rule, guard)
		edge, ok := w.cellIndex(46, 40)
		if !ok {
			t.Fatal("fixture: no edge cell")
		}
		gi := indexOfEntity(w.entities, 2)
		sweep := func() ([]byte, int) {
			s := w.groupSight(aiSight, []int{gi})
			return s, litCount(s)
		}
		s, before := sweep()
		if s[edge] == 0 {
			t.Fatal("fixture: the edge cell is outside the sweep before the cloud")
		}
		landSightCloud(t, w, rule, 20, 2)
		s, under := sweep()
		t.Logf("second %v: AI scan %d, sweep cells %d -> %d", second, w.entities[gi].ScanRange, before, under)
		if under >= before || s[edge] != 0 {
			t.Errorf("second %v: under Darkness the sweep lights %d cells (before %d), edge lit %v",
				second, under, before, s[edge] != 0)
		}
		waitSightCloudOut(t, w, 2)
		if _, after := sweep(); after != before {
			t.Errorf("second %v: after the effect the sweep lights %d cells, want %d", second, after, before)
		}
	}
}

// A body standing in a Light or Darkness cloud keeps the cloud's sight from
// the first pulse to the cloud's end at every tick, whatever tick the cloud
// lands on, so no periodic fog refresh falls between two pulses; the effect
// ends at most one pulse period after the cloud (DIV-4834, DIV-4835).
func TestALightOrDarknessCloudHoldsTheSightBetweenPulses(t *testing.T) {
	t.Parallel()
	for _, second := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			rule := darkSightRow(second, dark)
			for _, power := range []uint16{20, 110} {
				for phase := 0; phase < 32; phase++ {
					w := darkSightWorld(t, second, rule, darkSightHero(2))
					before := litCount(w.Sight(SelfSlot))
					for k := 0; k < phase; k++ {
						Step(w, nil)
					}
					if !w.landArea(rule, power, 1, true, 2, 2, 40, 40, nil) {
						t.Fatal("fixture: the cloud did not land")
					}
					pulsed, gaps, tail := false, 0, 0
					for k := 0; k < 4000; k++ {
						Step(w, nil)
						on := scanEffectOn(w, 2)
						if len(w.effects) == 0 {
							if !on {
								break
							}
							tail++
							continue
						}
						pulsed = pulsed || on
						if pulsed && !on {
							gaps++
						}
					}
					if !pulsed {
						t.Fatalf("second %v dark %v power %d phase %d: the effect never stood after a step", second, dark, power, phase)
					}
					if gaps != 0 || tail > cloudPulseTicks {
						t.Errorf("second %v dark %v power %d phase %d: the effect was absent %d ticks inside the cloud and stood %d after it",
							second, dark, power, phase, gaps, tail)
					}
					if after := litCount(w.Sight(SelfSlot)); after != before {
						t.Errorf("second %v dark %v power %d phase %d: after the cloud the sight lights %d cells, want %d",
							second, dark, power, phase, after, before)
					}
				}
			}
		}
	}
}
