package sim

import (
	"fmt"
	"strings"
	"testing"
)

// autoChoiceSightWorld: a player's wounded mage of sight 5 at (20, 8) with
// Fire Arrow and Heal, the player's scout and detector, an allied player 3,
// a neutral player 4, enemies of player 2 seen by the caster, the scout, the
// ally or nobody, invisible enemies the caster lights or the detector
// detects, an invisible ally and a wounded own unit on a cell no one lights.
func autoChoiceSightWorld(t *testing.T, second bool) *World {
	t.Helper()
	fire, heal := castDistanceRow(second, false), hlHeal()
	heal.Second = second
	hide := SpellRule{ID: 15, Arm: InvisibilitySpellArm(), School: 3, ManaCost: 5, TargetsUnit: true,
		Defensive: true, Second: second}
	unit := func(id EntityID, x, y int32, owner uint32, sight, detect uint8, hp int32) Entity {
		e := spEnt(id, x, y)
		e.Owner, e.ScanRange, e.SeeInvisible, e.TokenSize, e.HP = owner, sight, detect, 1, hp
		return e
	}
	caster := spMage(1, 20, 8, 30, 100, 100, 1<<fire.ID|1<<heal.ID)
	caster.Owner, caster.ScanRange, caster.Speed, caster.HP, caster.TokenSize = SelfSlot, 5, 60, 60, 1
	caster.AttackCharge, caster.AttackRelax = 8, 4
	ents := []Entity{
		caster,
		unit(2, 17, 8, 2, 3, 0, 100),
		unit(3, 14, 8, 2, 3, 0, 100),
		unit(4, 12, 12, SelfSlot, 8, 0, 100),
		unit(5, 30, 8, 3, 4, 0, 50),
		unit(6, 27, 8, 2, 3, 0, 100),
		unit(7, 2, 1, SelfSlot, 0, 0, 40),
		unit(8, 16, 3, SelfSlot, 3, 3, 100),
		unit(9, 14, 4, 2, 3, 0, 100),
		unit(10, 23, 8, 3, 0, 0, 50),
		unit(11, 38, 1, 2, 3, 0, 100),
		unit(12, 22, 6, 4, 0, 0, 70),
	}
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile},
		[3]uint32{SelfSlot, 3, relationLocked}, [3]uint32{3, SelfSlot, relationLocked},
		[3]uint32{3, 2, relationHostile}, [3]uint32{2, 3, relationHostile})
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 40, Height: 16}, ModeCanonical,
		Terrain{}, ents, nil, rel, nil, nil, []SpellRule{fire, heal, hide})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	for i := range w.groups {
		if w.groups[i].owner != SelfSlot {
			w.groups[i].order = orderStandGround
		}
	}
	for _, id := range []EntityID{2, 9, 10} {
		if !w.attachEffect(id, id, hide, EffectInvisible, 1, 30, EffectDuration) || !w.invisible(int(id-1)) {
			t.Fatal("invisibility did not attach")
		}
	}
	return w
}

// autoChoiceSightAnswers is what the automatic choice of the caster at index
// 0 takes in w: the sight of each actor, the Fire Arrow pick at its range and
// at any range, the Heal pick, the support candidates, the cast approach's
// side sight, the caster's own cast sight, and the hash after 60 steps of the
// automatic cast.
func autoChoiceSightAnswers(w *World) string {
	bits := func(f func(i int) bool) string {
		var b strings.Builder
		for i := range w.entities {
			if f(i) {
				b.WriteByte('1')
			} else {
				b.WriteByte('0')
			}
		}
		return b.String()
	}
	var sight autoChoiceSight
	seen := bits(func(i int) bool { return w.autoChoiceSees(0, i, &sight) })
	fire, _ := w.findSpell(1)
	heal, _ := w.findSpell(6)
	near, _ := w.autoCastTargetWithin(0, fire, 7, 1)
	far, _ := w.autoCastTargetWithin(0, fire, 64, 1)
	healed, _ := w.autoCastTargetWithin(0, heal, 64, 1)
	support := w.autoCastCandidates(0, AutoCastOwn|AutoCastAllies|AutoCastNeutral)
	side := bits(func(i int) bool { return w.ownerSees(SelfSlot, w.entities[i].ID) })
	own := bits(func(i int) bool { return w.castSeesEntity(0, i) })
	w.entities[0].AutoSpell = 1
	for range 60 {
		Step(w, nil)
	}
	return fmt.Sprintf("sees %s near %d far %d heal %d support %v side %s own %s hash %016x",
		seen, near, far, healed, support, side, own, w.Hash())
}

// TestTheAutomaticChoiceSeesAsEachGameDid pins what each game's automatic
// choice takes: the first game the cells its caster lights, the second what
// its side and allies perceive (owner, DIV-4872).
func TestTheAutomaticChoiceSeesAsEachGameDid(t *testing.T) {
	t.Parallel()

	games := map[bool]bool{}
	for _, tc := range []struct {
		name   string
		second bool
		want   string
	}{
		{name: "first game", want: "sees 110000000101 near 2 far 2 heal 1 support [0 9 11] " +
			"side 101100111101 own 100000000101 hash 672efb045a66956b"},
		{name: "second game", second: true, want: "sees 101111111101 near 3 far 3 heal 1 support [0 3 4 6 7 9 11] " +
			"side 101100111101 own 100000000101 hash a09257093dae208f"},
	} {
		games[tc.second] = true
		t.Run(tc.name, func(t *testing.T) {
			if got := autoChoiceSightAnswers(autoChoiceSightWorld(t, tc.second)); got != tc.want {
				t.Fatalf("got  %s\nwant %s", got, tc.want)
			}
		})
	}
	if !games[false] || !games[true] {
		t.Fatalf("games pinned %v, want both", games)
	}
}
