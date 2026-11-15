package sim

import (
	"fmt"
	"strings"
	"testing"
)

// scrollApproachRow is one tick of the caster's order: the move goal, the
// command group and the position.
func scrollApproachRow(w *World) string {
	e := w.Entities()[0]
	return fmt.Sprintf("goal=%v(%d,%d) group=%d|at=(%d,%d)", e.HasTarget, e.TargetX, e.TargetY, e.Group, e.X, e.Y)
}

// scrollApproachWorld puts a caster at (1, 1) and an enemy at (tx, 1). With a
// scroll the caster carries a damage scroll and knows no spell; with a book it
// knows the same spell and holds the mana.
func scrollApproachWorld(t *testing.T, tx int32, scroll, second bool) *World {
	t.Helper()
	rule := SpellRule{ID: 1, ManaCost: 100, School: 1, MaxRange: 3, DamageMin: 10, DamageMax: 10, TargetsUnit: true, Damaging: true}
	var caster Entity
	if scroll {
		caster = spEnt(1, 1, 1)
	} else {
		caster = spMage(1, 1, 1, 30, 200, 200, 1<<1)
	}
	caster.Owner = 1
	caster.AttackCharge, caster.AttackRelax = 4, 2
	victim := spEnt(2, tx, 1)
	victim.Owner = 2
	victim.HP, victim.MaxHP = 500, 500
	w := hlWorld(t, 1090, acEnemies(t), []SpellRule{rule}, caster, victim)
	if scroll {
		w.carried[0] = []ItemStack{StackItem(ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 1 | 60<<16}}}, 1)}
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	return w
}

// scrollApproachTrace casts and records the caster's row each tick until the
// cast ends (no record and the victim hit) and for 40 ticks after.
func scrollApproachTrace(t *testing.T, w *World, scroll bool) (rows []string, end int) {
	t.Helper()
	cmd := spCast(1, 2, 1)
	if scroll {
		cmd = UseScroll(1, 0, 2)
	}
	end = -1
	for tick := 0; tick < 400; tick++ {
		if tick == 0 {
			Step(w, []Command{cmd})
		} else {
			Step(w, nil)
		}
		rows = append(rows, scrollApproachRow(w))
		active := len(w.ScrollCasts()) != 0 || len(w.bookCasts) != 0
		if end < 0 && !active && w.Entities()[1].HP < 500 {
			end = tick
		}
		if end >= 0 && tick == end+40 {
			return rows, end
		}
	}
	t.Fatalf("the cast never ended; last row %s", rows[len(rows)-1])
	return nil, 0
}

// TestScrollCastWalksIntoReachAsASpellDoesAndStands: a scroll cast at an enemy
// out of reach and at one adjacent leaves the caster the same order a book cast
// leaves it from the cast's end on: standing where it cast, with no goal.
func TestScrollCastWalksIntoReachAsASpellDoesAndStands(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, tx := range []int32{12, 2} {
			name := fmt.Sprintf("second=%v enemy=%d", second, tx)
			sRows, sEnd := scrollApproachTrace(t, scrollApproachWorld(t, tx, true, second), true)
			bRows, bEnd := scrollApproachTrace(t, scrollApproachWorld(t, tx, false, second), false)
			t.Logf("%s scroll: cast ends tick %d, then %s", name, sEnd, sRows[sEnd])
			t.Logf("%s spell:  cast ends tick %d, then %s", name, bEnd, bRows[bEnd])
			// The order matches from the cast's end on; the scroll's reach
			// carries its own power, so the cell the caster stops at may differ,
			// but each caster stands where it cast.
			for i := 0; i <= 40; i++ {
				so, sp, _ := strings.Cut(sRows[sEnd+i], "|")
				bo, _, _ := strings.Cut(bRows[bEnd+i], "|")
				_, sp0, _ := strings.Cut(sRows[sEnd], "|")
				if so != bo {
					t.Errorf("%s: %d ticks after the cast the scroll caster holds %s, the spell caster %s", name, i, so, bo)
					break
				}
				if sp != sp0 {
					t.Errorf("%s: the scroll caster walked on after its cast: %s then %s", name, sp0, sp)
					break
				}
			}
			if tx == 2 {
				_, p0, _ := strings.Cut(sRows[0], "|")
				for i, r := range sRows {
					if _, p, _ := strings.Cut(r, "|"); p != p0 {
						t.Errorf("%s: the caster beside the enemy moved at tick %d: %s", name, i, r)
						break
					}
				}
			}
		}
	}
}
