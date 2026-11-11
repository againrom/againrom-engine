package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// darknessFogWorld is a map screen of either game over fogTestBounds: the
// player's mage at (19,22) with no sight of its own and the player's Human at
// (22,22), Mind 25, Reaction 36 and a whole-cell sight of 6. The mage knows
// the Darkness (dark) or Light cloud row of that game, whose id it returns.
func darknessFogWorld(t *testing.T, second, dark bool) (*mapWorld, uint16) {
	t.Helper()
	id := uint16(12)
	switch {
	case dark && second:
		id = 14
	case dark:
		id = 17
	case second:
		id = 15
	}
	// The shipped durations: area column 20 and effect counter 16 in the
	// first game (MAGIC-398), 15 and the raw 1 in the second (R2-ENGINE-024,
	// R2-ASSET-178).
	rows := []sim.SpellRule{{ID: id, Area: true, Distribution: 3, Radius: 1, AreaDuration: 20, MaxRange: 10,
		ManaCost: 1, TargetsUnit: true, EffectKind: sim.EffectScanRange, EffectMode: sim.EffectDuration, EffectDuration: 16}}
	var script *sim.Script
	if second {
		sim.AssignSecondGameArms(rows)
		rows[0].AreaDuration, rows[0].EffectDuration = 15, 1
		s, err := sim.NewROM2Script(nil, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		script = s
	}
	mage := sim.Entity{ID: 1, X: 19, Y: 22, HP: 100, MaxHP: 100, Owner: sim.SelfSlot, TokenSize: 1, Mind: 50,
		Mana: 500, MaxMana: 500, KnownSpells: 1 << id, AttackCharge: 4, AttackRelax: 2, Book: sim.Spellbook{State: sim.BookPresent}}
	mage.Book.Slots[id-1] = sim.BookSpell{Range: 10, ManaCost: 1}
	hero := sim.Entity{ID: 2, X: 22, Y: 22, HP: 200, MaxHP: 200, Owner: sim.SelfSlot, TokenSize: 1,
		Humanoid: true, Mind: 25, Reaction: 36, ScanRange: 6}
	w, err := sim.NewStockedSpelledWorld(1, fogTestBounds, sim.ModeCanonical, sim.Terrain{},
		[]sim.Entity{mage, hero}, script, sim.Relations{}, nil, nil, rows)
	if err != nil {
		t.Fatal(err)
	}
	mw := newMapWorld(w, nil, nil, fogTestViewer(t))
	if second {
		edition := base.GameROM2.Edition()
		if mw.mission == nil {
			mw.mission = &missionNotices{}
		}
		mw.mission.table = &mapload.Table{Edition: &edition}
	}
	if mw.files().fogStampOnMove == second {
		t.Fatalf("fixture: second game %v plays the other game's fog", second)
	}
	return mw, id
}

func darknessFogHero(mw *mapWorld) (sim.Entity, bool) {
	var hero sim.Entity
	for _, e := range mw.world.EntityView() {
		if e.ID == 2 {
			hero = e
		}
	}
	for _, a := range mw.world.ActiveEffects() {
		if a.Target == 2 && a.Kind == sim.EffectScanRange {
			return hero, true
		}
	}
	return hero, false
}

// The visible layer of the player's fog is smaller around a Human standing in
// a Darkness cloud at every periodic refresh from the first pulse to the
// cloud's end, the Human not moving, whatever tick the cast is given on, and
// is whole again at the first refresh after the effect ends; Light makes it
// larger. Both games' map screens.
func TestTheVisibleFogFollowsADarknessOrLightCloudWithoutAMove(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, dark := range []bool{true, false} {
			for phase := 0; phase < 32; phase++ {
				mw, id := darknessFogWorld(t, second, dark)
				visible := func() int {
					n := 0
					for _, v := range mw.fog.visible {
						if v != 0 {
							n++
						}
					}
					return n
				}
				before := visible()
				for k := 0; k < phase; k++ {
					mw.tick()
				}
				mw.pending = append(mw.pending, sim.CastAt(1, sim.SpellID(id), sim.CellPoint{X: 22, Y: 22}))
				pulsed, refreshes, wrong := false, 0, 0
				for k := 0; k < 4000; k++ {
					mw.tick()
					_, on := darknessFogHero(mw)
					pulsed = pulsed || on
					clouds := len(mw.world.CellEffects())
					if pulsed && clouds == 0 && !on {
						break
					}
					if !pulsed || clouds == 0 || mw.world.Tick()%fogPeriod != 0 {
						continue
					}
					refreshes++
					if n := visible(); dark && n >= before || !dark && n <= before {
						wrong++
					}
				}
				hero, _ := darknessFogHero(mw)
				if hero.X != 22 || hero.Y != 22 {
					t.Fatalf("second %v dark %v phase %d: the Human moved to (%d,%d)", second, dark, phase, hero.X, hero.Y)
				}
				if !pulsed || refreshes == 0 || wrong != 0 {
					t.Errorf("second %v dark %v phase %d: effect stood %v; %d of %d refreshes inside the cloud kept %d cells' extent",
						second, dark, phase, pulsed, wrong, refreshes, before)
				}
				for mw.world.Tick()%fogPeriod != 0 {
					mw.tick()
				}
				if after := visible(); after != before {
					t.Errorf("second %v dark %v phase %d: after the effect the visible layer holds %d cells, want %d",
						second, dark, phase, after, before)
				}
			}
		}
	}
}
