package sim

import "testing"

// hrWorld is a mage of owner mageOwner with Heal at (2, 2), reach 1, a wounded
// ally of the same owner beside it, and a hostile archer of the other owner
// seven cells south, reach 8, ordered onto the mage: it strikes the mage from
// outside the mage's own reach.
func hrWorld(t *testing.T, mageOwner uint32, mageHP int32, auto uint16) *World {
	t.Helper()
	other := uint32(3) - mageOwner
	mage := acCaster(1, 2, 2, 50, 1<<6, auto)
	mage.Owner, mage.Reach, mage.HP = mageOwner, 1, mageHP
	ally := spEnt(2, 3, 2)
	ally.Owner, ally.HP = mageOwner, 40
	archer := spEnt(3, 2, 9)
	archer.Owner, archer.Reach = other, 8
	archer.DamageBase, archer.AlwaysHits, archer.ScanRange = 3, true, 10
	archer.HP, archer.MaxHP = 30000, 30000
	archer.HasAttackTarget, archer.AttackTarget = true, 1
	return hlWorld(t, 71, acEnemies(t), []SpellRule{hlHeal()}, mage, ally, archer)
}

// hrRun steps w n ticks, keeping the archer on the mage, and returns the
// mage's cast events in order.
func hrRun(t *testing.T, w *World, n int) []CastEvent {
	t.Helper()
	var casts []CastEvent
	ai := indexOfEntity(w.entities, 3)
	for k := 0; k < n; k++ {
		if ai >= 0 && w.entities[ai].Alive() && k%8 == 0 && !w.entities[ai].HasAttackTarget {
			w.orderAttack(ai, 1)
		}
		for _, c := range StepObserved(w, nil) {
			if c.Caster == 1 {
				casts = append(casts, c)
			}
		}
	}
	return casts
}

func TestAPlayerMageStruckFromOutsideItsReachHealsItselfFirst(t *testing.T) {
	t.Parallel()

	w := hrWorld(t, SelfSlot, 60, 0)
	casts := hrRun(t, w, 400)
	if len(casts) == 0 {
		t.Fatal("the wounded mage struck from seven cells, outside its reach of 1, never healed")
	}
	if casts[0].Target != 1 || casts[0].Spell != 6 {
		t.Errorf("the first cast went to %d with spell %d, want Heal on the wounded mage itself", casts[0].Target, casts[0].Spell)
	}
}

func TestAPlayerMageStruckFromOutsideItsReachHealsTheWoundedAlly(t *testing.T) {
	t.Parallel()

	w := hrWorld(t, SelfSlot, 100, 0)
	hrRun(t, w, 400)
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("the wounded ally holds %d health of 40: the mage struck from outside its reach never healed it", got)
	}
}

// hrMelee is a mage of the player with Heal at (2, 2), reach 1, a full or
// wounded ally beside it, and a hostile standing in the mage's reach at
// (2, 3). A striking hostile sees and strikes the mage; a passive one has no
// sight and no damage.
func hrMelee(t *testing.T, auto uint16, allyHP int32, striking bool) *World {
	t.Helper()
	mage := acCaster(1, 2, 2, 50, 1<<6, auto)
	mage.Owner, mage.Reach, mage.DamageBase, mage.AlwaysHits = SelfSlot, 1, 5, true
	ally := spEnt(2, 3, 2)
	ally.Owner, ally.HP = SelfSlot, allyHP
	foe := spEnt(3, 2, 3)
	foe.Owner, foe.Reach = 2, 1
	foe.HP, foe.MaxHP = 30000, 30000
	if striking {
		foe.DamageBase, foe.AlwaysHits, foe.ScanRange = 3, true, 10
	}
	return hlWorld(t, 71, acEnemies(t), []SpellRule{hlHeal()}, mage, ally, foe)
}

func hrHeals(casts []CastEvent) int {
	n := 0
	for _, c := range casts {
		if c.Spell == 6 {
			n++
		}
	}
	return n
}

func TestAHostileWithinThePlayerMagesReachStillBlocksTheIdleHeal(t *testing.T) {
	t.Parallel()

	w := hrMelee(t, 0, 40, true)
	if got := hrHeals(hrRun(t, w, 400)); got != 0 {
		t.Errorf("the unarmed mage healed %d times with a hostile in its reach of 1", got)
	}
}

func TestAnArmedHealOutranksTheAttackWhenAPatientExists(t *testing.T) {
	t.Parallel()

	w := hrMelee(t, 6, 40, true)
	casts := hrRun(t, w, 400)
	if hrHeals(casts) == 0 {
		t.Fatal("the mage with Heal armed never healed while a hostile in its reach struck it")
	}
}

func TestAnArmedHealWithNoPatientLeavesTheMageToItsAttack(t *testing.T) {
	t.Parallel()

	armed, unarmed := hrMelee(t, 6, 100, false), hrMelee(t, 0, 100, false)
	// A player's mage of reach below 2 starts no strike on its own
	// (AI-GUARD-007); this is a mage with a staff's.
	armed.entities[indexOfEntity(armed.entities, 1)].Reach = 2
	unarmed.entities[indexOfEntity(unarmed.entities, 1)].Reach = 2
	for k := 0; k < 400; k++ {
		Step(armed, nil)
		Step(unarmed, nil)
		a, u := spAt(t, armed, 3), spAt(t, unarmed, 3)
		if a.HP != u.HP {
			t.Fatalf("tick %d: the hostile holds %d health beside the armed mage and %d beside the unarmed one", k, a.HP, u.HP)
		}
	}
	if got := spAt(t, armed, 3).HP; got >= 30000 {
		t.Errorf("the hostile in reach holds %d health: the mage never struck it", got)
	}
}

func TestAnAIOwnedMageStruckFromOutsideItsReachKeepsTheOldCondition(t *testing.T) {
	t.Parallel()

	w := hrWorld(t, 2, 60, 0)
	ci := indexOfEntity(w.entities, 1)
	for k := 0; k < 200; k++ {
		if ai := indexOfEntity(w.entities, 3); k%8 == 0 && !w.entities[ai].HasAttackTarget {
			w.orderAttack(ai, 1)
		}
		Step(w, nil)
		if ai := indexOfEntity(w.entities, 3); w.entities[ai].HasAttackTarget && w.entities[ai].AttackTarget == 1 {
			if !w.inCombat(ci) {
				t.Fatalf("tick %d: the AI owner's mage is out of combat while a hostile holds it as victim", k)
			}
		}
	}
}

// TestTheFirstGamesAutomaticCastingLevelTakesTheReachCondition is the level
// route of the human player's mage in the first game (owner): an enemy in its
// reach of 1 stops support even while the mage strikes it, which the second
// game's attack-range gate would allow; an enemy striking from outside its
// reach does not stop support.
func TestTheFirstGamesAutomaticCastingLevelTakesTheReachCondition(t *testing.T) {
	near := acsWorld(t, acEnemies(t), AutoCastMinimum|AutoCastOwn, acsMask(6), 0, acsUnit(2, SelfSlot, 4, 40), acsUnit(3, 2, 3, 100))
	e := &near.entities[0]
	e.Reach, e.AttackTarget, e.HasAttackTarget = 1, 3, true
	if near.supportAllowed(0) {
		t.Error("an enemy in the mage's reach left the level's support open")
	}
	if !near.supportWhileAttacking(0) {
		t.Fatal("fixture: the second game's gate would refuse this melee mage too, so it controls nothing")
	}
	far := acsUnit(3, 2, 7, 100)
	far.Reach, far.HasAttackTarget, far.AttackTarget = 8, true, 1
	w := acsWorld(t, acEnemies(t), AutoCastMinimum|AutoCastOwn, acsMask(6), 0, acsUnit(2, SelfSlot, 4, 40), far)
	w.entities[0].Reach = 1
	if !w.supportAllowed(0) {
		t.Error("an enemy striking from outside the mage's reach closed the level's support")
	}
}

// TestTheHealArmConditionReadsReachNotEngagement is the predicate alone on a
// player's mage of reach 1 that holds no victim: a hostile in its reach is
// combat; one two cells away holding it as victim is not; the same hostile is
// combat for an AI owner's mage. A mage that sees nothing is in combat only
// while the victim it holds stands in its reach.
func TestTheHealArmConditionReadsReachNotEngagement(t *testing.T) {
	for _, tc := range []struct {
		name    string
		owner   uint32
		y       int32
		holding bool
		blind   bool
		want    bool
	}{
		{"hostile in reach", SelfSlot, 3, false, false, true},
		{"hostile outside reach striking it", SelfSlot, 4, true, false, false},
		{"AI owner's mage, hostile outside reach striking it", 2, 4, true, false, true},
		{"blind mage holding a victim in reach", SelfSlot, 3, false, true, true},
		{"blind mage holding a victim outside reach", SelfSlot, 4, false, true, false},
	} {
		mage := acCaster(1, 2, 2, 50, 1<<6, 0)
		mage.Owner, mage.Reach = tc.owner, 1
		if tc.blind {
			mage.ScanRange, mage.HasAttackTarget, mage.AttackTarget = 0, true, 3
		}
		foe := spEnt(3, 2, tc.y)
		foe.Owner, foe.Reach = 3-tc.owner, 2
		if tc.holding {
			foe.HasAttackTarget, foe.AttackTarget = true, 1
		}
		w := hlWorld(t, 71, acEnemies(t), []SpellRule{hlHeal()}, mage, foe)
		if got := w.inCombat(indexOfEntity(w.entities, 1)); got != tc.want {
			t.Errorf("%s: inCombat %v, want %v", tc.name, got, tc.want)
		}
	}
}
