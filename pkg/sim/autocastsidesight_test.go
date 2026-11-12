package sim

import "testing"

// sideSightWorld: a player's mage of sight 5 at (20, 8) armed with Fire Arrow
// (range 7), a standing enemy 7 cells west at (13, 8), and, with scout, a
// unit of the mage's player of sight 12 at (12, 12) that sees the enemy.
func sideSightWorld(t *testing.T, second, scout bool) *World {
	t.Helper()
	rule := castDistanceRow(second, false)
	caster := spMage(1, 20, 8, 30, 100, 100, 1<<rule.ID)
	caster.Owner, caster.ScanRange, caster.Speed = SelfSlot, 5, 60
	caster.AttackCharge, caster.AttackRelax = 8, 4
	target := spEnt(2, 13, 8)
	target.Owner, target.HP, target.MaxHP, target.TokenSize = 2, 5000, 5000, 1
	ents := []Entity{caster, target}
	if scout {
		s := spEnt(3, 12, 12)
		s.Owner, s.ScanRange, s.TokenSize = SelfSlot, 12, 1
		ents = append(ents, s)
	}
	w, err := NewStockedSpelledWorld(7, Bounds{Width: 40, Height: 16}, ModeCanonical,
		Terrain{}, ents, nil, acEnemies(t), nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	if second {
		w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
	}
	for i := range w.groups {
		if w.groups[i].owner == 2 {
			w.groups[i].order = orderStandGround
		}
	}
	return w
}

// TestTheSecondGameAutomaticCastTakesWhatItsSideSees: the second game's
// automatic cast chooses an enemy a living unit of the caster's player sees
// and casts from the spell's range; an enemy no unit of the side sees is not
// chosen; an ordered cast asks no sight. The first game keeps the caster's
// own sight (owner, DIV-4872).
func TestTheSecondGameAutomaticCastTakesWhatItsSideSees(t *testing.T) {
	t.Parallel()

	auto := func(w *World) { w.entities[0].AutoSpell = 1 }
	ordered := func(w *World) { Step(w, []Command{spCast(1, 2, 1)}) }
	for _, tc := range []struct {
		name          string
		second, scout bool
		order         func(*World)
		cast          bool
	}{
		{name: "no unit of the side sees the enemy", second: true, order: auto},
		{name: "another unit of the side sees the enemy", second: true, scout: true, order: auto, cast: true},
		{name: "an ordered cast at the unseen enemy", second: true, order: ordered, cast: true},
		{name: "first game: the scout lends no sight", scout: true, order: auto},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := sideSightWorld(t, tc.second, tc.scout)
			if w.ownerSees(SelfSlot, 2) != tc.scout {
				t.Fatalf("the side sees the enemy %v, want %v", !tc.scout, tc.scout)
			}
			x, edge, ok := castDistanceRelease(t, w, tc.order)
			if ok != tc.cast || ok && (x != 20 || edge != 7) {
				t.Fatalf("release %v from x=%d at edge distance %d, want %v from x=20 at 7", ok, x, edge, tc.cast)
			}
		})
	}
}

// TestTheSecondGameSupportCandidatesTakeWhatTheSideSees: the support levels'
// candidate list asks the same sight as the automatic cast: a living ally
// lights its own cell in the second game; an own unit sees itself.
func TestTheSecondGameSupportCandidatesTakeWhatTheSideSees(t *testing.T) {
	t.Parallel()

	build := func(second, scout bool) *World {
		caster := acCaster(1, 2, 2, 50, 1<<6, 6)
		caster.Owner, caster.ScanRange = SelfSlot, 3
		ally := spEnt(2, 12, 2)
		ally.Owner, ally.HP = 2, 20
		own := spEnt(3, 14, 14)
		own.Owner, own.HP, own.ScanRange = SelfSlot, 20, 0
		ents := []Entity{caster, ally, own}
		if scout {
			s := spEnt(4, 11, 4)
			s.Owner, s.ScanRange = SelfSlot, 4
			ents = append(ents, s)
		}
		w := hlWorld(t, 29, castSightRelation(t, relationLocked), []SpellRule{hlHeal()}, ents...)
		if second {
			w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
		}
		return w
	}
	has := func(list []int, i int) bool {
		for _, v := range list {
			if v == i {
				return true
			}
		}
		return false
	}
	for _, tc := range []struct {
		name          string
		second, scout bool
		ally, own     bool
	}{
		{name: "second game, no scout", second: true, ally: true, own: true},
		{name: "second game, a scout sees the ally", second: true, scout: true, ally: true, own: true},
		{name: "first game, a scout sees the ally", scout: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := build(tc.second, tc.scout).autoCastCandidates(0, AutoCastOwn|AutoCastAllies)
			if !has(got, 0) || has(got, 1) != tc.ally || has(got, 2) != tc.own {
				t.Fatalf("candidates %v, want the caster, ally %v, own unit %v", got, tc.ally, tc.own)
			}
		})
	}
}

// TestOwnerSeesAnswersAsEachUnitsCastSight: ownerSees over the side's lit set
// answers what asking castSeesEntity of each living unit answered, invisible,
// allied, dead and far actors included, in both games.
func TestOwnerSeesAnswersAsEachUnitsCastSight(t *testing.T) {
	t.Parallel()

	eachUnit := func(w *World, owner uint32, vi int) bool {
		for j := range w.entities {
			if w.entities[j].Owner == owner && w.entities[j].Alive() && w.castSeesEntity(j, vi) {
				return true
			}
		}
		return false
	}
	unit := func(id EntityID, x, y int32, owner uint32, sight, detect uint8) Entity {
		e := spEnt(id, x, y)
		e.Owner, e.ScanRange, e.SeeInvisible, e.TokenSize = owner, sight, detect, 1
		return e
	}
	ents := []Entity{
		unit(1, 2, 2, SelfSlot, 3, 0),
		unit(2, 10, 10, SelfSlot, 2, 1),
		unit(3, 6, 6, SelfSlot, 4, 0),
		unit(4, 11, 10, 2, 3, 0),
		unit(5, 4, 2, 2, 3, 0),
		unit(6, 5, 5, 2, 3, 0),
		unit(7, 15, 15, 2, 3, 0),
		unit(8, 3, 3, 3, 3, 0),
		unit(9, 14, 2, 3, 2, 0),
	}
	ents[2].HP = 0
	rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, relationHostile},
		[3]uint32{SelfSlot, 3, relationLocked}, [3]uint32{3, SelfSlot, relationLocked})
	for _, second := range []bool{false, true} {
		w := hlWorld(t, 31, rel, nil, ents...)
		if second {
			w.rom2, w.game = &rom2ScriptState{}, newGameProfile(ScriptROM2)
		}
		for _, id := range []EntityID{4, 5, 8} {
			if !w.attachEffect(id, id, SpellRule{ID: 15}, EffectInvisible, 1, 30, EffectDuration) {
				t.Fatal("invisibility did not attach")
			}
		}
		seen := 0
		for _, owner := range []uint32{SelfSlot, 2, 3} {
			for vi := range w.entities {
				want := eachUnit(w, owner, vi)
				if got := w.ownerSees(owner, w.entities[vi].ID); got != want {
					t.Fatalf("second %v: owner %d sees %d: %v, each unit %v", second, owner, w.entities[vi].ID, got, want)
				}
				if want {
					seen++
				}
			}
		}
		if seen == 0 || seen == 3*len(w.entities) {
			t.Fatalf("second %v: %d of %d answers seen, want a mix", second, seen, 3*len(w.entities))
		}
	}
}
