package sim

import "testing"

// TRIG-OFFMAP-041, AI-LIST-003: idle autocast skips an off-map foe.
func TestIdleAutocastIgnoresAnOffMapFoe(t *testing.T) {
	t.Parallel()

	build := func(offMap bool) *World {
		caster := acCaster(1, 2, 2, 50, 1<<1, 1)
		caster.AttackCharge, caster.AttackRelax = 12, 4
		caster.Owner = 1
		hidden := spEnt(2, 3, 3)
		hidden.Owner = 2
		far := spEnt(3, 6, 6)
		far.Owner = 2
		w := hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, hidden, far)
		if offMap && !w.takeOffMap(indexOfEntity(w.entities, 2)) {
			t.Fatal("the fixture foe did not leave the map")
		}
		return w
	}

	control := build(false)
	if id, ok := control.autoCastTarget(0, hlArrow()); !ok || id != 2 {
		t.Fatalf("control target %d/%v, want the nearer foe 2", id, ok)
	}
	w := build(true)
	if id, ok := w.autoCastTarget(0, hlArrow()); !ok || id != 3 {
		t.Fatalf("target %d/%v with foe 2 off the map, want the on-map foe 3", id, ok)
	}

	alone := build(true)
	alone.entities[indexOfEntity(alone.entities, 3)].OffMap = true
	for range 40 {
		Step(alone, nil)
	}
	if got := spAt(t, alone, 1).Mana; got != 50 {
		t.Fatalf("the caster holds %d mana with every foe off the map, want its own untouched 50", got)
	}
	if len(alone.deliveries) != 0 {
		t.Fatalf("%d deliveries in flight at off-map foes, want none", len(alone.deliveries))
	}
}

func TestSupportAutocastCandidatesSkipOffMapFriends(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, 1<<6, 6)
	caster.Owner = 1
	friend := spEnt(2, 3, 3)
	friend.Owner = 1
	friend.HP = 20
	w := hlWorld(t, 29, Relations{}, []SpellRule{hlHeal()}, caster, friend)
	if got := w.autoCastCandidates(0, AutoCastOwn); len(got) != 2 {
		t.Fatalf("control candidates %v, want the caster and the friend", got)
	}
	if !w.takeOffMap(indexOfEntity(w.entities, 2)) {
		t.Fatal("the fixture friend did not leave the map")
	}
	if got := w.autoCastCandidates(0, AutoCastOwn); len(got) != 1 || got[0] != 0 {
		t.Fatalf("candidates %v with the friend off the map, want the caster alone", got)
	}
	if id, ok := w.autoCastTarget(0, hlHeal()); ok {
		t.Fatalf("idle heal chose off-map friend %d", id)
	}
}
