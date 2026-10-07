package sim

import "testing"

func visibilityWorld(t *testing.T, terrain Terrain, spell SpellRule, caster Entity, targets ...Entity) *World {
	t.Helper()
	ents := append([]Entity{caster}, targets...)
	w, err := NewStockedSpelledWorld(91, Bounds{Width: 32, Height: 32}, ModeCanonical,
		terrain, ents, nil, acEnemies(t), nil, nil, []SpellRule{spell})
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func heightCell(x, y int, value byte) Terrain {
	h := make([]byte, 32*32)
	h[y*32+x] = value
	return Terrain{Height: h}
}

func TestActorDirectedCastRefusesACurrentlyHiddenTargetWithoutSideEffects(t *testing.T) {
	t.Parallel()

	rule := hlArrow()
	caster := spMage(1, 10, 10, 30, 50, 50, 1<<1)
	caster.Owner, caster.ScanRange, caster.Facing = 1, 1, 64
	target := spEnt(2, 11, 10)
	target.Owner = 2
	w := visibilityWorld(t, heightCell(11, 10, 127), rule, caster, target)

	events := StepObserved(w, []Command{spCast(1, 2, 1)})
	got := w.entities[0]
	if len(events) != 0 || len(w.bookCasts) != 0 || got.Mana != 50 || got.Facing != 64 ||
		w.entities[1].HP != 100 || got.CastWait != 0 {
		t.Fatalf("hidden cast changed state: events=%v pending=%d caster=%+v targetHP=%d",
			events, len(w.bookCasts), got, w.entities[1].HP)
	}
	if why := w.BookSpellRefusal(1, 2, 1); why != "target not currently visible" {
		t.Errorf("refusal = %q, want current visibility", why)
	}
}

func TestVisibilityIsRecheckedBeforeReleaseButNotAfterCommit(t *testing.T) {
	t.Parallel()

	rule := hlArrow()
	caster := spMage(1, 10, 10, 30, 50, 50, 1<<1)
	caster.Owner, caster.ScanRange, caster.AutoSpell = 1, 1, 1
	target := spEnt(2, 11, 10)
	target.Owner = 2
	w := visibilityWorld(t, Terrain{}, rule, caster, target)
	// This probe moves the target explicitly; retaliation must not walk it
	// back into sight after the committed hit.
	for i := range w.groups {
		w.groups[i].order = orderStandGround
	}

	Step(w, []Command{spCast(1, 2, 1)})
	if len(w.bookCasts) != 1 {
		t.Fatal("visible target was not admitted")
	}
	// Still in spell range, but outside the caster's current perception.
	w.entities[1].X, w.entities[1].PostX = 13, 13
	for range castPeriod {
		if events := StepObserved(w, nil); len(events) != 0 {
			t.Fatalf("pre-release sight loss emitted %+v", events)
		}
	}
	if w.entities[0].Mana != 50 || w.entities[1].HP != 100 {
		t.Fatalf("pre-release cancellation paid/applied: mana=%d hp=%d", w.entities[0].Mana, w.entities[1].HP)
	}

	// Restore sight, release one cast, then hide the target again. Application
	// is committed at release and is never rolled back; hidden follow-up auto
	// candidates are filtered until sight returns.
	w.entities[1].X, w.entities[1].PostX = 11, 11
	var released []CastEvent
	for n := 0; n < 32 && len(released) == 0; n++ {
		released = StepObserved(w, nil)
	}
	if len(released) != 1 {
		t.Fatalf("restored visibility released %d casts, want one", len(released))
	}
	hp, mana := w.entities[1].HP, w.entities[0].Mana
	w.entities[1].X, w.entities[1].PostX = 13, 13
	for range 32 {
		if events := StepObserved(w, nil); len(events) != 0 {
			t.Fatalf("hidden follow-up autocast emitted %+v", events)
		}
	}
	if w.entities[1].HP != hp || w.entities[0].Mana != mana {
		t.Fatalf("post-release hiding rolled back or repeated the cast: hp %d/%d mana %d/%d",
			w.entities[1].HP, hp, w.entities[0].Mana, mana)
	}
}

func TestAutocastFiltersHiddenCandidatesBeforeStablePriority(t *testing.T) {
	t.Parallel()

	h := make([]byte, 32*32)
	for x := 11; x <= 13; x++ {
		h[10*32+x] = 127
	}
	caster := spMage(1, 10, 10, 30, 50, 50, 1<<1)
	caster.Owner, caster.ScanRange, caster.AutoSpell = 1, 4, 1
	hiddenNear := spEnt(2, 13, 10)
	hiddenNear.Owner = 2
	visibleFar := spEnt(3, 10, 14)
	visibleFar.Owner = 2
	w := visibilityWorld(t, Terrain{Height: h}, hlArrow(), caster, hiddenNear, visibleFar)

	events := spRunUnbidden(w)
	if len(events) != 1 || events[0].Target != 3 {
		t.Fatalf("autocast events = %+v, want the farther currently visible target 3", events)
	}
	if w.entities[1].HP != 100 || w.entities[2].HP >= 100 {
		t.Fatalf("hidden/visible health = %d/%d, want hidden untouched and visible hit",
			w.entities[1].HP, w.entities[2].HP)
	}
}

func TestPointCastDoesNotBorrowAutomaticHealVisibility(t *testing.T) {
	t.Parallel()

	t.Run("place", func(t *testing.T) {
		rule := SpellRule{ID: 3, ManaCost: 4, School: 1, MaxRange: 6, Area: true,
			Distribution: 2, Radius: 1, AreaDuration: 1, Damaging: true}
		caster := spMage(1, 10, 10, 30, 50, 50, 1<<3)
		caster.ScanRange, caster.Facing = 1, 32
		w := visibilityWorld(t, heightCell(11, 10, 127), rule, caster)
		events := spRunCast(w, Command{Kind: KindCastAt, Entity: 1, X: 11, Y: 10, Spell: 3})
		if len(events) != 1 || w.entities[0].Mana != 46 {
			t.Fatalf("point cast incorrectly used personal sight: events=%v caster=%+v", events, w.entities[0])
		}
	})

	t.Run("heal", func(t *testing.T) {
		caster := spMage(1, 10, 10, 30, 50, 50, 1<<6)
		caster.Owner, caster.ScanRange = 1, 1
		hidden := spEnt(2, 11, 10)
		hidden.Owner, hidden.HP = 1, 20
		visible := spEnt(3, 10, 11)
		visible.Owner, visible.HP = 1, 30
		w := visibilityWorld(t, heightCell(11, 10, 127), hlHeal(), caster, hidden, visible)
		events := spRunUnbidden(w)
		if len(events) != 1 || events[0].Target != 3 || w.entities[1].HP != 20 {
			t.Fatalf("Heal selected through sight: events=%+v hiddenHP=%d", events, w.entities[1].HP)
		}
	})
}

// TestTargetDirectedWeaponSpellUsesVisibilityAtItsRelease asks the perception
// seam where spec.md puts it — at the cast's release — and asks the attack
// ORDER to be free of it.
//
// The order is an attack order and a weapon-spell carrier takes it exactly as a
// plain fighter does, victim written and approach begun. What the carrier does
// not do is land the spell on a target its own sight march cannot reach: the
// release refuses, the cycle turns over, and nothing is spent. Gating the order
// itself made a staff-armed hero ignore a click on any enemy the player can see
// through another unit, while a sword-armed one walked to it.
func TestTargetDirectedWeaponSpellUsesVisibilityAtItsRelease(t *testing.T) {
	t.Parallel()

	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 10, 10, 1, 30, 2, 1)
	caster.Owner, caster.ScanRange, caster.Facing = 1, 1, 64
	target := spEnt(2, 11, 10)
	target.Owner = 2
	w := visibilityWorld(t, heightCell(11, 10, 127), rule, caster, target)

	Step(w, []Command{cbOrder(1, 2)})
	if a := w.entities[0]; !a.HasAttackTarget || a.AttackTarget != 2 {
		t.Fatalf("a weapon-spell carrier refused an attack order on a hidden target: %+v", a)
	}
	for n := 0; n < 24; n++ {
		Step(w, nil)
	}
	if w.entities[1].HP != 100 {
		t.Fatalf("a hidden weapon-spell release landed: targetHP=%d", w.entities[1].HP)
	}

	// The same action releases once its target becomes currently visible.
	// Moving the target changes neither diplomacy nor range.
	w.entities[1].X, w.entities[1].Y, w.entities[1].PostX, w.entities[1].PostY = 10, 11, 10, 11
	Step(w, []Command{cbOrder(1, 2)})
	if a := w.entities[0]; !a.HasAttackTarget || a.AttackTarget != 2 {
		t.Fatalf("the action was lost when its target became visible: %+v", a)
	}
	for n := 0; n < 8 && w.entities[1].HP == 100; n++ {
		Step(w, nil)
	}
	if w.entities[1].HP >= 100 {
		t.Fatal("visible weapon cast never released")
	}
}

// TestWeaponSpellDoesNotLoadAWindUpTowardAHiddenTarget is the owner's own
// report: a mage sometimes visibly casts Lightning or Rainbow Lightning and
// no damage lands. The mechanism is not selection and not the delivery
// countdown: advanceAttack's top validity gate only drops an order for
// Invisibility (invisibleToActor), never for ordinary terrain sight loss, so
// a weapon-spell carrier's attack order stays admitted and READY loaded a
// full AttackCasting wind-up toward a target it could not currently see —
// releaseWeaponSpell's own release-time actorSeesEntity check then refused
// silently, spending a full charge-and-relax cycle on an application already
// excluded, and the order re-armed the same wind-up on its very next turn.
// This is DIV-1311's fix: an admitted order still stands (HasAttackTarget is
// untouched, on TestTargetDirectedWeaponSpellUsesVisibilityAtItsRelease's
// own ground above) and still waits for sight to return, but it no longer
// spends a cycle finding that out.
func TestWeaponSpellDoesNotLoadAWindUpTowardAHiddenTarget(t *testing.T) {
	t.Parallel()

	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 10, 10, 1, 30, 2, 1)
	caster.Owner, caster.ScanRange, caster.Facing = 1, 1, 64
	target := spEnt(2, 11, 10)
	target.Owner = 2
	w := visibilityWorld(t, heightCell(11, 10, 127), rule, caster, target)

	Step(w, []Command{cbOrder(1, 2)})
	if a := w.entities[0]; !a.HasAttackTarget || a.AttackTarget != 2 {
		t.Fatalf("a weapon-spell carrier refused an attack order on a hidden target: %+v", a)
	}
	for n := 0; n < 16; n++ {
		Step(w, nil)
		if p := w.entities[0].AttackPhase; p != AttackReady {
			t.Fatalf("loaded a wind-up (phase %v) toward a target still hidden", p)
		}
	}
	if w.entities[1].HP != 100 {
		t.Fatal("a hidden weapon-spell release landed")
	}

	// Sight returning still releases, unchanged from the case above.
	w.entities[1].X, w.entities[1].Y, w.entities[1].PostX, w.entities[1].PostY = 10, 11, 10, 11
	for n := 0; n < 8 && w.entities[1].HP == 100; n++ {
		Step(w, nil)
	}
	if w.entities[1].HP >= 100 {
		t.Fatal("visible weapon cast never released once the target came into view")
	}
}

// TestWeaponSpellDivertsWhenSightIsLostDuringTheWindUp covers the companion
// case: sight is present when the wind-up loads and is lost before its count
// reaches zero. Without the fix the wind-up still completed and
// releaseWeaponSpell's own check silently refused, spending the full
// recovery; the fix folds this into the wind-up's own FR-3a re-ask (0139) and
// diverts straight back to READY, exactly as a weapon that stopped carrying a
// spell mid-charge already does.
func TestWeaponSpellDivertsWhenSightIsLostDuringTheWindUp(t *testing.T) {
	t.Parallel()

	rule := wpnRule(6, 6, 5)
	caster := wpnCaster(1, 10, 10, 1, 30, 6, 1)
	caster.Owner, caster.ScanRange, caster.Facing = 1, 3, 64
	target := spEnt(2, 11, 10)
	target.Owner = 2
	w := visibilityWorld(t, Terrain{}, rule, caster, target)

	Step(w, []Command{cbOrder(1, 2)})
	if w.entities[0].AttackPhase != AttackCasting {
		t.Fatalf("wind-up did not load with sight available: %+v", w.entities[0])
	}
	// Hide the target for the rest of the charge (out of the caster's scan
	// range, still within the row's own 5-cell MaxRange).
	w.entities[1].X, w.entities[1].PostX = 14, 14
	w.entities[1].Y, w.entities[1].PostY = 14, 14
	for n := 0; n < 8 && w.entities[0].AttackPhase != AttackReady; n++ {
		Step(w, nil)
	}
	if w.entities[0].AttackPhase != AttackReady || w.entities[0].AttackCountdown != 0 {
		t.Fatalf("did not divert cleanly to READY: %+v", w.entities[0])
	}
	if w.entities[1].HP != 100 {
		t.Fatal("a sight-lost-mid-charge release still landed")
	}
}
