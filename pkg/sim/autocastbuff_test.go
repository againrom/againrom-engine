package sim

import "testing"

// The autocast usefulness gate (owner): an unbidden buff is attempted only
// where it would improve the actor it lands on, and a defensive row reaches
// the allies its own range covers.
//
// THE DEFECT: autocast re-cast a standing buff on every tick the recovery
// allowed. Each re-cast paid training experience, so a mage standing next to
// nothing levelled its school for free.

// acbShield is a point buff shaped like the shipped Shield row: id 18,
// school 4, MaxRange 0 — a range of zero is what makes it a self-cast, and
// nothing in the production path names the id.
func acbShield() SpellRule {
	return SpellRule{ID: 18, ManaCost: 3, School: 4, MaxRange: 0, TargetsUnit: true,
		Defensive: true, SpellDuration: 15, EffectKind: EffectAbsorption,
		EffectMode: EffectDuration, EffectMagnitude: 5}
}

// acbProtection is a point buff shaped like the shipped Protection from Fire
// row: id 5, school 1, MaxRange 5 — far enough to reach an ally.
func acbProtection() SpellRule {
	return SpellRule{ID: 5, ManaCost: 3, School: 1, MaxRange: 5, TargetsUnit: true,
		Defensive: true, SpellDuration: 30, EffectKind: EffectProtectionFire,
		EffectMode: EffectDuration, EffectMagnitude: 30}
}

// acbBuffCaster is a mage with a buff row armed and known. Mind 30 and no skill
// leave power at zero, so the numbers below are the row's own arithmetic at the
// bottom of the ladder: Shield lands 0/10+3 = 3 for 15*16 = 240 ticks.
func acbBuffCaster(id EntityID, x, y int32, spell uint16) Entity {
	e := spMage(id, x, y, 30, 50, 50, 1<<spell)
	e.AutoSpell = spell
	return e
}

// TestAStandingBuffIsNotRecastWhileItHolds is the defect itself: one cast, and
// then nothing for as long as the effect stands.
//
// 120 ticks is half the 240 the cast grants, so every tick after the first cast
// is one the old rule would have spent: it aimed the row at the caster
// unconditionally, and the only thing spacing the casts was the recovery.
func TestAStandingBuffIsNotRecastWhileItHolds(t *testing.T) {
	t.Parallel()

	w := spWorld(t, 17, []SpellRule{acbShield()}, acbBuffCaster(1, 2, 2, 18))
	for range 120 {
		Step(w, nil)
	}

	if got := spAt(t, w, 1).Mana; got != 47 {
		t.Errorf("the caster holds %d mana after 120 idle ticks, want 47 — one cast of a 3-mana row", got)
	}
	if got := len(w.attached); got != 1 {
		t.Fatalf("the world holds %d attached effect(s), want the one Shield record", got)
	}
	if got := w.attached[0].Magnitude; got != 3 {
		t.Errorf("the standing record's magnitude is %d, want 3", got)
	}
	if got := spAt(t, w, 1).Absorption; got != 3 {
		t.Errorf("the caster's absorption is %d, want 3 — a re-cast stacks nothing, it replaces", got)
	}
	if got := spAt(t, w, 1).CastWait; got != 0 {
		t.Errorf("the caster's wait is %d, want 0 — an attempt that finds nothing to improve costs no recovery", got)
	}
}

// TestABuffIsRecastOnceATenthOfItsDurationIsLeft states the owner's second
// condition at its own boundary. 240 is the duration a fresh cast grants at
// power 0, so the threshold is 24 ticks remaining.
func TestABuffIsRecastOnceATenthOfItsDurationIsLeft(t *testing.T) {
	t.Parallel()

	rule := acbShield()
	for _, tc := range []struct {
		name      string
		remaining uint16
		want      bool
	}{
		{"a tenth of the duration left", 24, true},
		{"one tick more than a tenth", 25, false},
		{"the last tick", 1, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := spWorld(t, 17, []SpellRule{rule}, acbBuffCaster(1, 2, 2, 18))
			if !w.attachEffect(1, 1, rule, EffectAbsorption, 3, 240, EffectDuration) {
				t.Fatal("attachEffect refused the fixture's own standing record")
			}
			w.attached[0].Remaining = tc.remaining

			if got := w.autoCastImproves(0, rule, 0); got != tc.want {
				t.Errorf("autoCastImproves = %v with %d of 240 ticks left, want %v",
					got, tc.remaining, tc.want)
			}
		})
	}
}

// TestABetterCastReplacesAWeakerStandingBuff states the owner's third
// condition: the skill grew between the two casts, so the row would land more
// than the record standing on the actor.
//
// The magnitudes are the row's own arithmetic, written out rather than derived
// from the production expression: Shield lands power/10 + 3, which is 3 at
// power 0 and 5 at power 20.
func TestABetterCastReplacesAWeakerStandingBuff(t *testing.T) {
	t.Parallel()

	rule := acbShield()
	w := spWorld(t, 17, []SpellRule{rule}, acbBuffCaster(1, 2, 2, 18))
	if !w.attachEffect(1, 1, rule, EffectAbsorption, 3, 240, EffectDuration) {
		t.Fatal("attachEffect refused the fixture's own standing record")
	}
	w.attached[0].Remaining = 240

	if w.autoCastImproves(0, rule, 0) {
		t.Error("autoCastImproves = true at the power the standing record was cast at, want false")
	}
	if !w.autoCastImproves(0, rule, 20) {
		t.Error("autoCastImproves = false at power 20, where the row lands 5 against the standing 3")
	}
}

// TestAProtectionClampedAwayIsNotWorthRecasting is why the third condition is
// measured as what would LAND and not as the row's nominal magnitude. The actor
// already sits at the 100 clamp, so a second 15 moves nothing; a gate comparing
// nominal magnitudes would read 15 against the standing 0 and re-cast for ever,
// which is the defect in a second form.
func TestAProtectionClampedAwayIsNotWorthRecasting(t *testing.T) {
	t.Parallel()

	rule := acbProtection()
	caster := acbBuffCaster(1, 2, 2, 5)
	caster.Protection[0] = 100
	w := spWorld(t, 17, []SpellRule{rule}, caster)

	if !w.attachEffect(1, 1, rule, EffectProtectionFire, 15, 240, EffectDuration) {
		t.Fatal("attachEffect refused the fixture's own standing record")
	}
	w.attached[0].Remaining = 240
	if got := w.attached[0].Magnitude; got != 0 {
		t.Fatalf("the standing record landed %d on an actor at the clamp, want 0", got)
	}
	if got := spAt(t, w, 1).Protection[0]; got != 100 {
		t.Fatalf("the actor's fire protection is %d, want the clamp's own 100", got)
	}

	if w.autoCastImproves(0, rule, 30) {
		t.Error("autoCastImproves = true where the cast would land nothing, want false")
	}
}

// TestADefensiveBuffReachesAnAllyInsideTheRowsRange is the owner's targeting
// half: a protection is cast on allies, not on the caster alone. The caster is
// at distance zero and takes the first cast; the ally is the next actor the
// gate still calls improvable.
//
// The hostile is the control. It stands nearer than the ally and never carries
// the effect.
func TestADefensiveBuffReachesAnAllyInsideTheRowsRange(t *testing.T) {
	t.Parallel()

	caster := acbBuffCaster(1, 2, 2, 5)
	caster.Owner = 1
	ally := spEnt(2, 4, 4)
	ally.Owner = 1
	enemy := spEnt(3, 3, 3)
	enemy.Owner = 2
	w := hlWorld(t, 17, acEnemies(t), []SpellRule{acbProtection()}, caster, ally, enemy)

	for range 3 {
		spRunUnbidden(w)
	}

	if !w.HasEffectSpell(1, 5) {
		t.Error("the caster carries no protection, want the first cast on itself")
	}
	if !w.HasEffectSpell(2, 5) {
		t.Error("the ally carries no protection, want the second cast to reach it")
	}
	if w.HasEffectSpell(3, 5) {
		t.Error("the hostile carries the protection, want a buff never aimed at an enemy")
	}
}

// TestAZeroRangeBuffNeverLeavesItsCaster is the other half of the same rule:
// which actors a buff reaches is the row's own MaxRange and not a spell id
// written here. Shield ships zero, so an ally standing next to the caster is
// out of range of it.
func TestAZeroRangeBuffNeverLeavesItsCaster(t *testing.T) {
	t.Parallel()

	caster := acbBuffCaster(1, 2, 2, 18)
	ally := spEnt(2, 3, 2)
	w := spWorld(t, 17, []SpellRule{acbShield()}, caster, ally)

	for range 3 {
		spRunUnbidden(w)
	}

	if !w.HasEffectSpell(1, 18) {
		t.Error("the caster carries no shield, want the self-cast a zero range leaves")
	}
	if w.HasEffectSpell(2, 18) {
		t.Error("the adjacent ally carries the shield, want a zero-range row to reach nobody else")
	}
}

// TestTheGateBindsAutocastAndNotACommandedCast is the rule's own boundary. The
// owner asked for the spells that autocast; a player who orders a re-cast by
// hand gets one, standing effect or not.
func TestTheGateBindsAutocastAndNotACommandedCast(t *testing.T) {
	t.Parallel()

	rule := acbShield()
	caster := acbBuffCaster(1, 2, 2, 18)
	caster.AutoSpell = 0
	w := spWorld(t, 17, []SpellRule{rule}, caster)
	if !w.attachEffect(1, 1, rule, EffectAbsorption, 3, 240, EffectDuration) {
		t.Fatal("attachEffect refused the fixture's own standing record")
	}
	w.attached[0].Remaining = 100

	spRunCast(w, spCast(1, 1, 18))

	if got := spAt(t, w, 1).Mana; got != 47 {
		t.Errorf("the caster holds %d mana after a commanded re-cast, want 47 — the gate binds autocast alone", got)
	}
	if got := w.attached[0].Remaining; got != 240 {
		t.Errorf("the standing record has %d ticks left, want the commanded re-cast's fresh 240", got)
	}
}
