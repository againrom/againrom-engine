package sim

import "testing"

// The predicates as they stood before the perception query, kept verbatim as
// the oracle each purpose row must reproduce.
func todayActorSeesEntity(w *World, i, target int) bool {
	if i < 0 || i >= len(w.entities) || target < 0 || target >= len(w.entities) {
		return false
	}
	if w.entities[i].ID == w.entities[target].ID {
		return true
	}
	observer, candidate := w.entities[i], w.entities[target]
	if !w.sightShows(w.actorSight(i), cellOf(&candidate)) {
		return false
	}
	return !w.hasAttachedSpell(candidate.ID, w.armSpellID(15)) ||
		cellOf(&observer).chebyshevTo(cellOf(&candidate)) <= int64(observer.SeeInvisible)
}

func todayCastSeesEntity(w *World, i, target int) bool {
	if i < 0 || i >= len(w.entities) || target < 0 || target >= len(w.entities) || i == target ||
		!w.ownOrAllied(&w.entities[i], &w.entities[target]) {
		return todayActorSeesEntity(w, i, target)
	}
	return w.sightShows(w.actorSight(i), cellOf(&w.entities[target]))
}

func todayInvisibleToActor(w *World, i, target int) bool {
	return target >= 0 && target < len(w.entities) &&
		w.hasAttachedSpell(w.entities[target].ID, w.armSpellID(15)) && !todayActorSeesEntity(w, i, target)
}

func todayGroupDetectsInvisible(w *World, members []int, candidate cell) bool {
	for _, i := range members {
		if i >= 0 && i < len(w.entities) &&
			cellOf(&w.entities[i]).chebyshevTo(candidate) <= int64(w.entities[i].SeeInvisible) {
			return true
		}
	}
	return false
}

func todayComputerMageGroupSees(w *World, ci, vi int) bool {
	caster, target := w.entities[ci], w.entities[vi]
	if w.SecondGame() || w.humanParticipantUnit(caster) || !isMage(caster) {
		return false
	}
	members := w.groupLivingMembers(caster.Owner, effectiveGroup(&caster))
	stamp := w.groupSight(aiSight, members)
	w.stampAttackNotices(stamp, members, false)
	return w.sightShows(stamp, cellOf(&target)) &&
		(!w.hasAttachedSpell(target.ID, w.armSpellID(15)) || todayGroupDetectsInvisible(w, members, cellOf(&target)))
}

func todayInvisibleTo(w *World, target EntityID, owner uint32) bool {
	ti := indexOfEntity(w.entities, target)
	if ti < 0 || !w.hasAttachedSpell(target, w.armSpellID(15)) {
		return false
	}
	at := cellOf(&w.entities[ti])
	for i := range w.entities {
		e := w.entities[i]
		if e.Owner != owner {
			continue
		}
		if e.ID == target {
			return false
		}
		if cellOf(&e).chebyshevTo(at) <= int64(e.SeeInvisible) {
			return false
		}
	}
	return true
}

// perceptionWorld: observer 0, target 1 at x,2, member 2, player 2's mage 3.
func perceptionWorld(t *testing.T, second, hidden bool, see uint8, x int32, owner uint32, rel byte) *World {
	t.Helper()
	observer := effectMage(1, 2, 2, 1<<1)
	observer.SeeInvisible = see
	target := spEnt(2, x, 2)
	target.Owner = owner
	member := spEnt(3, 2, 5)
	member.Owner, member.SeeInvisible, member.ScanRange = SelfSlot, 1, sightRings
	mage := spMage(4, 9, 9, 30, 100, 100, 1<<1)
	mage.Owner, mage.TokenSize = 2, 1
	invisibility := SpellRule{ID: 15}
	spells := []SpellRule{{ID: 1, ManaCost: 3, School: 1, MaxRange: 5, TargetsUnit: true, DamageMin: 10, DamageMax: 10, Damaging: true}}
	if second {
		invisibility = secondRow(SpellRule{ID: 12, TargetsUnit: true, SpellDuration: 10})
		spells = []SpellRule{invisibility}
	}
	w := hlWorld(t, 101, castSightRelation(t, rel), spells, observer, target, member, mage)
	if second {
		w.rom2 = &rom2ScriptState{}
	}
	if hidden && !w.attachEffect(2, 2, invisibility, EffectInvisible, 1, 30, EffectDuration) {
		t.Fatal("invisibility did not attach")
	}
	return w
}

func TestPerceptionQueryReproducesEachPredicate(t *testing.T) {
	groups := [][]int{nil, {0}, {2}, {0, 2}, {0, 1, 2}, {3}, {-1, 9}}
	cases := 0
	answers := map[string][2]int{}
	tally := func(kind string, v bool) {
		a := answers[kind]
		if v {
			a[1]++
		} else {
			a[0]++
		}
		answers[kind] = a
	}
	for _, second := range []bool{false, true} {
		for _, hidden := range []bool{false, true} {
			for see := uint8(0); see <= 4; see++ {
				for x := int32(3); x <= 7; x++ {
					for _, owner := range []uint32{SelfSlot, 2} {
						for _, rel := range []byte{0, relationLocked, relationHostile, relationLocked | relationHostile} {
							w := perceptionWorld(t, second, hidden, see, x, owner, rel)
							n := len(w.entities)
							for i := -1; i <= n; i++ {
								for tg := -1; tg <= n; tg++ {
									if got, want := w.actorSeesEntity(i, tg), todayActorSeesEntity(w, i, tg); got != want {
										t.Fatalf("attack %v %v see=%d x=%d owner=%d rel=%d %d->%d: %v, today %v", second, hidden, see, x, owner, rel, i, tg, got, want)
									}
									if got, want := w.castSeesEntity(i, tg), todayCastSeesEntity(w, i, tg); got != want {
										t.Fatalf("cast %v %v see=%d x=%d owner=%d rel=%d %d->%d: %v, today %v", second, hidden, see, x, owner, rel, i, tg, got, want)
									}
									if got, want := w.invisibleToActor(i, tg), todayInvisibleToActor(w, i, tg); got != want {
										t.Fatalf("continue %v %v see=%d x=%d owner=%d rel=%d %d->%d: %v, today %v", second, hidden, see, x, owner, rel, i, tg, got, want)
									}
									tally("attack hidden", hidden && tg == 1 && todayActorSeesEntity(w, i, tg))
									tally("cast hidden", hidden && tg == 1 && todayCastSeesEntity(w, i, tg))
									tally("continue", todayInvisibleToActor(w, i, tg))
									cases += 3
								}
							}
							for i := 0; i < n; i++ {
								for tg := 0; tg < n; tg++ {
									if got, want := w.computerMageGroupSees(i, tg), todayComputerMageGroupSees(w, i, tg); got != want {
										t.Fatalf("mage group %v %v see=%d x=%d %d->%d: %v, today %v", second, hidden, see, x, i, tg, got, want)
									}
									for _, members := range groups {
										invisible := w.hasAttachedSpell(w.entities[tg].ID, w.armSpellID(15))
										want := !invisible || todayGroupDetectsInvisible(w, members, cellOf(&w.entities[tg]))
										if got := w.perceives(perception{purpose: perceiveGroup, members: members, target: tg}); got != want {
											t.Fatalf("group %v %v see=%d x=%d %v->%d: %v, today %v", second, hidden, see, x, members, tg, got, want)
										}
										tally("group hidden", invisible && want)
									}
									tally("mage group", todayComputerMageGroupSees(w, i, tg))
									cases += 1 + len(groups)
								}
							}
							for id := EntityID(0); id <= 5; id++ {
								for _, slot := range []uint32{0, SelfSlot, 2, 3} {
									if got, want := w.InvisibleTo(id, slot), todayInvisibleTo(w, id, slot); got != want {
										t.Fatalf("participant %v %v see=%d x=%d id=%d slot=%d: %v, today %v", second, hidden, see, x, id, slot, got, want)
									}
									tally("participant", todayInvisibleTo(w, id, slot))
									cases++
								}
							}
						}
					}
				}
			}
		}
	}
	for kind, a := range answers {
		if a[0] == 0 || a[1] == 0 {
			t.Errorf("%s answered only one way: %v", kind, a)
		}
	}
	t.Logf("%d perception answers equal today's; false/true per kind %v", cases, answers)
}

func TestPerceptionQuerySeeInvisibleRadiusEdge(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, see := range []uint8{1, 2} {
			w := perceptionWorld(t, second, true, see, 4, 2, relationHostile)
			want := see == 2
			if w.actorSeesEntity(0, 1) != want || w.castSeesEntity(0, 1) != want || w.invisibleToActor(0, 1) == want ||
				w.perceives(perception{purpose: perceiveGroup, members: []int{0}, target: 1}) != want {
				t.Fatalf("second=%v see=%d: the radius edge moved", second, see)
			}
			if !w.actorSees(0, cellOf(&w.entities[1])) {
				t.Fatalf("second=%v: terrain sight lost the target cell", second)
			}
		}
	}
}
