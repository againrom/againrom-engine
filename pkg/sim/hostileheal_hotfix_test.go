package sim

import "testing"

// A Heal aimed at a unit whose player the caster's player is hostile to is
// cast like any other: the cost is paid and the marks and event are
// produced, and the target's health does not change (MAGIC-TARGET-017's apply
// arm refuses the hostile pair and the cast around it is unchanged).

func hostileHealWorld(t *testing.T) *World {
	t.Helper()
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	caster.Owner = 1
	caster.AttackCharge, caster.AttackRelax = 4, 2
	enemy := spEnt(2, 3, 3)
	enemy.Owner, enemy.HP = 2, 40
	return hlWorld(t, 5, acEnemies(t), []SpellRule{hlHeal()}, caster, enemy)
}

func TestACommandedHealAtAnEnemyIsCastPaidAndChangesNoHealth(t *testing.T) {
	t.Parallel()

	w := hostileHealWorld(t)
	events := spRunCast(w, spCast(1, 2, 6))

	if got := spAt(t, w, 1).Mana; got != 40 {
		t.Errorf("the caster holds %d mana, want 40: the cast is paid", got)
	}
	if got := spAt(t, w, 2).HP; got != 40 {
		t.Errorf("the enemy holds %d health, want its own untouched 40", got)
	}
	if got := spAt(t, w, 2).SpellFX; got != spellFXLife {
		t.Errorf("the enemy carries a mark of %d ticks, want %d", got, spellFXLife)
	}
	if got := spAt(t, w, 1).SpellFX; got != spellFXLife {
		t.Errorf("the caster carries a mark of %d ticks, want %d", got, spellFXLife)
	}
	if len(events) != 1 || events[0].Spell != 6 || events[0].Target != 2 || events[0].HealthRestored != 0 {
		t.Errorf("cast events %+v, want one Heal event at the enemy restoring nothing", events)
	}
}

func TestAReadHealScrollAtAnEnemyIsConsumedAndChangesNoHealth(t *testing.T) {
	t.Parallel()

	w := hostileHealWorld(t)
	item := ItemInstance{Code: 0xe10, Kind: 4, Price: 50, Effects: []ItemEffect{{Kind: 41, Operand: 6 | 60<<16}}}
	w.carried[0] = []ItemStack{StackItem(item, 1)}
	events := StepObserved(w, []Command{UseScroll(1, 0, 2)})
	for n := 0; n < 60 && len(w.scrollCasts) > 0; n++ {
		events = append(events, StepObserved(w, nil)...)
	}

	if len(w.scrollCasts) != 0 || len(w.carried[0]) != 0 {
		t.Fatalf("the scroll was not consumed by a completed read: casts=%d stacks=%d", len(w.scrollCasts), len(w.carried[0]))
	}
	if got := spAt(t, w, 2).HP; got != 40 {
		t.Errorf("the enemy holds %d health, want its own untouched 40", got)
	}
	if got := spAt(t, w, 1).Mana; got != 50 {
		t.Errorf("the caster holds %d mana, want 50: a scroll costs no mana", got)
	}
	if len(events) != 1 || events[0].Spell != 6 || events[0].HealthRestored != 0 {
		t.Errorf("cast events %+v, want one Heal event restoring nothing", events)
	}
}

func TestAWeaponHealAtAnEnemyChangesNoHealth(t *testing.T) {
	t.Parallel()

	w := hostileHealWorld(t)
	ci, ti := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	w.entities[ci].WeaponSpellLevel = 60
	obs := &castObs{}
	w.releaseWeaponSpell(ci, ti, hlHeal(), obs)

	if got := spAt(t, w, 2).HP; got != 40 {
		t.Errorf("the enemy holds %d health, want its own untouched 40", got)
	}
	if len(obs.casts) != 1 || obs.casts[0].HealthRestored != 0 {
		t.Errorf("observed casts %+v, want one Heal event restoring nothing", obs.casts)
	}
}

func TestAHealAtAnAllyStillRestoresHealth(t *testing.T) {
	t.Parallel()

	w := hostileHealWorld(t)
	ci, ti := indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 2)
	w.entities[ti].Owner = 1
	w.ordinaryEffect(ci, ti, hlHeal(), 60)
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("the ally holds %d health, want more than 40", got)
	}
}
