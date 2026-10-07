package sim

import "testing"

func csRule() SpellRule {
	return SpellRule{ID: 25, ManaCost: 2, TargetsUnit: true, MaxRange: 4}
}

// csItemWorld: caster (1,1), target 2 and bones corpse 3 both at (2,2).
func csItemWorld(t *testing.T, targetDomain Domain) *World {
	t.Helper()
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	target := spEnt(2, 2, 2)
	target.Domain, target.TokenSize = targetDomain, 1
	corpse := spEnt(3, 2, 2)
	corpse.Reaction, corpse.Mind, corpse.Spirit = 41, 37, 29
	corpse.ToHit, corpse.Defence = 71, 73
	w := hlGhostWorld(t, 30, []SpellRule{csRule()}, hlGhostTemplate(), caster, target, corpse)
	w.entities[2].HP, w.entities[2].Decay = -10, DecayBones
	return w
}

// The arm finds the corpse by the target's cell; the target is unchanged.
func TestItemReleaseOfControlSpiritRaisesFromTheCorpseAtTheTargetCell(t *testing.T) {
	w := csItemWorld(t, DomainAir)
	targetBefore := w.entities[1]
	if !w.weaponSpellApply(0, 1, csRule(), 0, nil) {
		t.Fatal("weaponSpellApply refused Control Spirit, want it queued")
	}
	if len(w.entities) != 3 {
		t.Fatalf("the release changed the entity list at once: %d entities", len(w.entities))
	}
	Step(w, nil)
	if len(w.entities) != 3 {
		t.Fatalf("entities = %d, want caster, target and one raised actor", len(w.entities))
	}
	if got := w.entities[1]; got.ID != 2 || got.HP != targetBefore.HP || got.SpellFX != targetBefore.SpellFX || got.Domain != targetBefore.Domain {
		t.Errorf("target object changed: %+v", got)
	}
	g := w.entities[2]
	if g.ID == 3 || g.Class != hlGhostTemplate().Class || g.Owner != 7 || g.X != 2 || g.Y != 2 {
		t.Fatalf("raised actor = %+v, want the owner-7 template actor in the corpse cell", g)
	}
	if g.ToHit != 71 || g.Defence != 73 || g.Reaction != 21 || g.Mind != 37 || g.Spirit != 29 {
		t.Errorf("corpse stores = to-hit %d defence %d reaction %d mind %d spirit %d", g.ToHit, g.Defence, g.Reaction, g.Mind, g.Spirit)
	}
}

// A ground target in the corpse's cell does not block the raise.
func TestItemReleaseOfControlSpiritWithAGroundTargetInTheCellStillRaises(t *testing.T) {
	w := csItemWorld(t, DomainGround)
	if !w.weaponSpellApply(0, 1, csRule(), 0, nil) {
		t.Fatal("weaponSpellApply refused Control Spirit")
	}
	Step(w, nil)
	if len(w.entities) != 3 || w.entities[2].Class != hlGhostTemplate().Class || w.entities[1].ID != 2 {
		t.Fatalf("entities = %+v, want caster, target and the raised actor", w.entities)
	}
}

// A book cast raises despite a ground actor in the corpse cell.
func TestBookCastOfControlSpiritRaisesWithAGroundActorInTheCorpseCell(t *testing.T) {
	caster := effectMage(1, 1, 1, 1<<25)
	caster.Owner = 7
	corpse := spEnt(2, 2, 2)
	other := spEnt(3, 2, 2)
	other.TokenSize = 1
	w := hlGhostWorld(t, 33, []SpellRule{csRule()}, hlGhostTemplate(), caster, corpse, other)
	w.entities[1].HP, w.entities[1].Decay = -10, DecayBones
	spRunCast(w, Cast(1, 2, 25))
	if len(w.entities) != 3 || w.entities[2].Class != hlGhostTemplate().Class || w.entities[2].Owner != 7 {
		t.Fatalf("entities = %+v, want caster, the ground actor and a raised ghost", w.entities)
	}
}

// No corpse in the target's cell: nothing is queued.
func TestItemReleaseOfControlSpiritWithoutACorpseAtTheCellDoesNothing(t *testing.T) {
	w := csItemWorld(t, DomainAir)
	w.entities[2].X = 5
	if w.weaponSpellApply(0, 1, csRule(), 0, nil) {
		t.Fatal("applied with no corpse at the target cell")
	}
}

// One corpse serves one release in a tick.
func TestTwoItemReleasesShareOneCorpseOnce(t *testing.T) {
	w := csItemWorld(t, DomainAir)
	if !w.weaponSpellApply(0, 1, csRule(), 0, nil) || w.weaponSpellApply(0, 1, csRule(), 0, nil) {
		t.Fatal("want the first release queued and the second refused")
	}
}

// The fighter's rider reaches the arm through a strike.
func TestFighterRiderOfControlSpiritRaisesThroughAStrike(t *testing.T) {
	caster := wpnCaster(1, 1, 1, 25, 30, 1, 0)
	caster.MaxMana, caster.Mana = 0, 0
	caster.Owner, caster.DamageBase, caster.AlwaysHits = 7, 5, true
	target := spEnt(2, 2, 2)
	target.TokenSize, target.Owner = 1, 3
	corpse := spEnt(3, 2, 2)
	w := hlGhostWorld(t, 31, []SpellRule{csRule()}, hlGhostTemplate(), caster, target, corpse)
	w.entities[2].HP, w.entities[2].Decay = -10, DecayBones
	Step(w, []Command{cbOrder(1, 2)})
	for range 30 {
		Step(w, nil)
	}
	raised := false
	for _, e := range w.entities {
		if e.ID == 3 {
			t.Fatal("corpse still present: the rider did not reach the arm")
		}
		raised = raised || e.Class == hlGhostTemplate().Class && e.Owner == 7
	}
	if !raised {
		t.Fatalf("no ghost raised by the rider: %+v", w.entities)
	}
}

// A Scroll of Control Spirit uses the same cell-found corpse.
func TestScrollOfControlSpiritRaisesFromTheCorpseAtTheTargetCell(t *testing.T) {
	caster := spEnt(1, 1, 1)
	caster.Owner, caster.AttackCharge, caster.AttackRelax = 7, 4, 2
	target := spEnt(2, 2, 2)
	target.Domain, target.TokenSize = DomainAir, 1
	corpse := spEnt(3, 2, 2)
	w := hlGhostWorld(t, 32, []SpellRule{csRule()}, hlGhostTemplate(), caster, target, corpse)
	w.entities[2].HP, w.entities[2].Decay = -10, DecayBones
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 25 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	Step(w, []Command{UseScroll(1, 0, 2)})
	if len(w.ScrollCasts()) != 1 {
		t.Fatal("scroll not admitted")
	}
	for tick := 0; tick < 60 && len(w.ScrollCasts()) != 0; tick++ {
		Step(w, nil)
	}
	for _, e := range w.entities {
		if e.ID == 3 {
			t.Fatal("corpse not consumed")
		}
	}
	if len(w.entities) != 3 || w.entities[2].Class != hlGhostTemplate().Class || w.entities[2].Owner != 7 {
		t.Fatalf("entities = %+v, want caster, target and the raised actor", w.entities)
	}
}

// The scroll target's own corpse is consumed.
func TestScrollOfControlSpiritConsumesTheTargetsOwnCorpse(t *testing.T) {
	caster := spEnt(1, 1, 1)
	caster.Owner, caster.AttackCharge, caster.AttackRelax = 7, 4, 2
	first := spEnt(2, 2, 2)
	own := spEnt(3, 2, 2)
	w := hlGhostWorld(t, 34, []SpellRule{csRule()}, hlGhostTemplate(), caster, first, own)
	for _, i := range []int{1, 2} {
		w.entities[i].HP, w.entities[i].Decay = -10, DecayBones
	}
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 25 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	Step(w, []Command{UseScroll(1, 0, 3)})
	for tick := 0; tick < 60 && len(w.ScrollCasts()) != 0; tick++ {
		Step(w, nil)
	}
	var left []EntityID
	for _, e := range w.entities {
		left = append(left, e.ID)
	}
	if indexOfEntity(w.entities, 2) < 0 || indexOfEntity(w.entities, 3) >= 0 || len(left) != 3 {
		t.Fatalf("entities left = %v, want the first corpse kept and the target's own consumed", left)
	}
}
