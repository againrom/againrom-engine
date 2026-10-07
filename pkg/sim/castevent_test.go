package sim

import (
	"bytes"
	"reflect"
	"testing"
)

// The cast observation, the weapon-borne mark and the heal autocast. The
// fixtures are heal_test.go's and autocast_test.go's.

// ceStaff is a mage whose WEAPON carries a spell: the four fields a release
// reads, plus an attack order already loaded against victim.
func ceStaff(id EntityID, x, y int32, spell uint16, victim EntityID) Entity {
	e := spMage(id, x, y, 30, 50, 50, 0)
	e.WeaponSpell, e.WeaponSpellLevel = spell, 0
	e.Reach = 1
	e.HasAttackTarget, e.AttackTarget = true, victim
	e.AttackPhase, e.AttackCountdown, e.AttackCharge = AttackCasting, 1, 4
	return e
}

func TestAnObservedStepReportsTheCasterTheTargetAndTheCellsOfAnAppliedCast(t *testing.T) {
	t.Parallel()

	caster := spMage(1, 2, 2, 30, 50, 50, 1<<1)
	enemy := spEnt(2, 5, 4)
	enemy.Owner = 2
	caster.Owner = 1
	w := hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

	events := spRunCast(w, spCast(1, 2, 1))

	if len(events) != 1 {
		t.Fatalf("the step reported %d casts, want exactly the one that landed", len(events))
	}
	got := events[0]
	// Facing 96 is south-east (facingStep 32 × direction 3), which is where a
	// caster at (2,2) turns to face a target at (5,4) before the cast lands.
	want := CastEvent{Caster: 1, Target: 2, Spell: 1, School: 1, Owner: 1, TargetOwner: 2,
		FromX: 2, FromY: 2, ToX: 5, ToY: 4, Weapon: false, Facing: 96}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("the observation is\n %+v\nwant\n %+v", got, want)
	}
}

func TestARefusedCastIsObservedAsNothing(t *testing.T) {
	t.Parallel()

	// Two mana against a cost of three: the cast is refused at the cost.
	caster := spMage(1, 2, 2, 30, 50, 2, 1<<1)
	caster.Owner = 1
	enemy := spEnt(2, 5, 4)
	enemy.Owner = 2
	w := hlWorld(t, 7, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)

	if events := StepObserved(w, []Command{spCast(1, 2, 1)}); len(events) != 0 {
		t.Errorf("a refused cast was observed as %+v", events)
	}
}

func TestObservingAStepChangesNoByteOfTheWorld(t *testing.T) {
	t.Parallel()

	build := func() *World {
		caster := acCaster(1, 2, 2, 50, 1<<1, 1)
		caster.Owner = 1
		enemy := spEnt(2, 4, 4)
		enemy.Owner = 2
		return hlWorld(t, 23, acEnemies(t), []SpellRule{hlArrow()}, caster, enemy)
	}

	plain, observed := build(), build()
	for range 20 {
		Step(plain, nil)
		StepObserved(observed, nil)
		if !bytes.Equal(hlBytes(t, plain), hlBytes(t, observed)) {
			t.Fatal("an observed step and an unobserved step produced different worlds")
		}
	}
}

func TestAWeaponBorneReleaseMarksBothActorsAndIsObserved(t *testing.T) {
	t.Parallel()

	staff := ceStaff(1, 2, 2, 1, 2)
	staff.Owner = 1
	victim := spEnt(2, 3, 2)
	victim.Owner = 2
	w := hlWorld(t, 29, acEnemies(t), []SpellRule{hlArrow()}, staff, victim)

	// The countdown is one, so this step is the release.
	events := StepObserved(w, nil)

	if got := spAt(t, w, 2).HP; got >= 100 {
		t.Fatalf("the victim holds %d health — no release landed and this test would pass for the wrong reason", got)
	}
	if got := spAt(t, w, 1).SpellFX; got != spellFXLife {
		t.Errorf("the caster carries a mark of %d ticks, want %d", got, spellFXLife)
	}
	if got := spAt(t, w, 2).SpellFX; got != spellFXLife {
		t.Errorf("the victim carries a mark of %d ticks, want %d", got, spellFXLife)
	}
	if got := spAt(t, w, 2).SpellFXSpell; got != 1 {
		t.Errorf("the victim's mark names spell %d, want the released row's own 1", got)
	}
	if len(events) != 1 || !events[0].Weapon {
		t.Fatalf("the step observed %+v, want one weapon-borne cast", events)
	}
	if got := events[0]; got.Caster != 1 || got.Target != 2 || got.Spell != 1 {
		t.Errorf("the observation is %+v, want caster 1, target 2, spell 1", got)
	}
}

func TestOutOfBattleACasterThatKnowsAHealHealsWithNothingArmed(t *testing.T) {
	t.Parallel()

	// AutoSpell is 0 throughout: nothing is armed.
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	caster.Owner = 1
	hurt := spEnt(2, 3, 3)
	hurt.Owner, hurt.HP = 1, 40
	w := hlWorld(t, 31, acEnemies(t), []SpellRule{hlHeal()}, caster, hurt)

	spRunUnbidden(w)

	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("the ally holds %d health and held 40 — an unarmed caster heals out of battle", got)
	}
	if got := spAt(t, w, 1).AutoSpell; got != 0 {
		t.Errorf("the caster's autocast is %d, want 0 — the arm sets nothing", got)
	}
	if got := spAt(t, w, 1).Mana; got != 40 {
		t.Errorf("the caster holds %d mana, want 40 — the heal was paid for once", got)
	}
}

func TestACasterHealsItselfOutOfBattle(t *testing.T) {
	t.Parallel()

	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	caster.Owner, caster.HP = 1, 50
	w := hlWorld(t, 33, acEnemies(t), []SpellRule{hlHeal()}, caster)

	spRunUnbidden(w)

	if got := spAt(t, w, 1).HP; got <= 50 {
		t.Errorf("the caster holds %d health and held 50 — it heals itself out of battle", got)
	}
}

func TestInBattleAnArmedAttackingRowKeepsPriorityOverAKnownHeal(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, (1<<1)|(1<<6), 1)
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	enemy.HasAttackTarget, enemy.AttackTarget = true, caster.ID
	hurt := spEnt(3, 3, 3)
	hurt.Owner, hurt.HP = 1, 30
	w := hlWorld(t, 35, acEnemies(t), []SpellRule{hlArrow(), hlHeal()}, caster, enemy, hurt)

	spRunUnbidden(w)

	if got := spAt(t, w, 2).HP; got >= 100 {
		t.Errorf("the enemy holds %d health — the armed attacking row is what runs in battle", got)
	}
	if got := spAt(t, w, 3).HP; got != 30 {
		t.Errorf("the wounded ally holds %d health, want its own untouched 30", got)
	}
}

func TestAnArmedHealTakesFirstPriorityInBattle(t *testing.T) {
	t.Parallel()

	caster := acCaster(1, 2, 2, 50, (1<<1)|(1<<6), 6)
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	enemy.HasAttackTarget, enemy.AttackTarget = true, caster.ID
	hurt := spEnt(3, 3, 3)
	hurt.Owner, hurt.HP = 1, 30
	w := hlWorld(t, 37, acEnemies(t), []SpellRule{hlArrow(), hlHeal()}, caster, enemy, hurt)

	spRunUnbidden(w)

	if got := spAt(t, w, 3).HP; got <= 30 {
		t.Errorf("the wounded ally holds %d health and held 30 — an armed heal is first in battle", got)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("the enemy holds %d health, want its own untouched 100 — one cast a tick", got)
	}
}

func TestTheUnarmedOutOfBattleHealKeepsAManaReserveAndAnArmedOneDoesNot(t *testing.T) {
	t.Parallel()

	// A pool of 40 reserves 10. A caster holding 19 cannot pay 10 and stay at
	// or above 10, so the unarmed arm does not attempt it.
	build := func(auto uint16) *World {
		caster := spMage(1, 2, 2, 30, 40, 19, 1<<6)
		caster.Owner, caster.AutoSpell = 1, auto
		hurt := spEnt(2, 3, 3)
		hurt.Owner, hurt.HP = 1, 30
		return hlWorld(t, 39, acEnemies(t), []SpellRule{hlHeal()}, caster, hurt)
	}

	unarmed := build(0)
	Step(unarmed, nil)
	if got := spAt(t, unarmed, 2).HP; got != 30 {
		t.Errorf("the ally holds %d health, want 30 — the reserve refused the unbidden heal", got)
	}
	if got := spAt(t, unarmed, 1).Mana; got != 19 {
		t.Errorf("the caster holds %d mana, want its own untouched 19", got)
	}

	armed := build(6)
	spRunUnbidden(armed)
	if got := spAt(t, armed, 2).HP; got <= 30 {
		t.Errorf("the ally holds %d health and held 30 — an armed heal keeps no reserve", got)
	}
}

// TestAHigherPriorityRowWithNoTargetFallsThroughToTheArmedOne is autoCast's own
// fall-through: a mage whose heal has nobody to aim at still fires what it armed
// on the same tick.
func TestAHigherPriorityRowWithNoTargetFallsThroughToTheArmedOne(t *testing.T) {
	t.Parallel()

	// Every ally is at full health, so the restorative arm has no candidate.
	// The caster is in battle, so the heal is not first priority here either;
	// what this measures is that the tick is not lost to the higher priority.
	caster := acCaster(1, 2, 2, 50, (1<<1)|(1<<6), 1)
	caster.Owner = 1
	enemy := spEnt(2, 4, 4)
	enemy.Owner = 2
	well := spEnt(3, 3, 3)
	well.Owner = 1
	w := hlWorld(t, 41, acEnemies(t), []SpellRule{hlArrow(), hlHeal()}, caster, enemy, well)

	spRunUnbidden(w)

	if got := spAt(t, w, 2).HP; got >= 100 {
		t.Errorf("the enemy holds %d health — the armed row ran when the heal found nobody", got)
	}
}

// TestAnEntityWithNoBookAndNothingArmedCastsNothing is autoCastOrder's own first
// statement, and it is the state every entity in every world built before this
// story is in.
func TestAnEntityWithNoBookAndNothingArmedCastsNothing(t *testing.T) {
	t.Parallel()

	plain := spEnt(1, 2, 2)
	plain.Owner = 1
	hurt := spEnt(2, 3, 3)
	hurt.Owner, hurt.HP = 1, 30
	w := hlWorld(t, 43, acEnemies(t), []SpellRule{hlHeal()}, plain, hurt)

	Step(w, nil)

	if got := spAt(t, w, 2).HP; got != 30 {
		t.Errorf("the wounded ally holds %d health, want 30 — nothing here can cast", got)
	}
}
