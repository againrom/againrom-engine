package sim

import (
	"reflect"
	"testing"
)

// ---------------------------------------------------------------- fixtures

// spEnt is one entity for these tests: an id, a cell, a health pool and a
// dwell long enough that nothing here tears a felled one down mid-test — on
// cbEnt's own reason (combat_test.go).
func spEnt(id EntityID, x, y int32) Entity {
	return Entity{ID: id, X: x, Y: y, HP: 100, MaxHP: 100, DyingTime: 200}
}

// spMage is spEnt with a mana pool, a Mind and a book: the four fields a
// cast's economy reads on the caster's side.
func spMage(id EntityID, x, y int32, mind, maxMana, mana int32, known uint32) Entity {
	e := spEnt(id, x, y)
	e.Mind, e.MaxMana, e.Mana, e.KnownSpells, e.ScanRange = mind, maxMana, mana, known, sightRings
	return e
}

// spWorld builds a world over an open 16x16 grid holding spells and ents,
// failing the test rather than returning an error, on cbWorld's own reason
// (combat_test.go).
func spWorld(t *testing.T, seed uint64, spells []SpellRule, ents ...Entity) *World {
	t.Helper()
	w, err := NewSpelledWorld(seed, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, ents, nil, spells)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return w
}

// spCast is one cast command: caster, victim, spell id.
func spCast(caster, victim EntityID, spell uint32) Command {
	return Command{Kind: KindCast, Entity: caster, X: int32(victim), Y: int32(spell)}
}

// spRunCast admits one command and advances through its canonical wind-up,
// returning the release observation. Refusal tests deliberately keep calling
// Step directly: advancing unrelated ticks would no longer measure a refusal
// against the same tick.
func spRunCast(w *World, cmd Command) []CastEvent {
	events := StepObserved(w, []Command{cmd})
	for n := 0; len(events) == 0 && n < 255; n++ {
		if i, ok := w.bookCastIndex(cmd.Entity); ok && w.bookCasts[i].Complete {
			break
		}
		events = StepObserved(w, nil)
		if len(w.bookCasts) == 0 && len(events) == 0 {
			break
		}
	}
	// Most tests in this file ask about one application rather than a retained
	// order. End that fixture after its first release while preserving the
	// recovery the application wrote.
	if i, ok := w.bookCastIndex(cmd.Entity); ok {
		if len(events) != 0 || w.bookCasts[i].Complete {
			if w.bookCasts[i].Phase == bookRelaxing {
				ci := indexOfEntity(w.entities, cmd.Entity)
				w.entities[ci].CastWait = w.bookCasts[i].Remaining
			}
			w.bookCasts = append(w.bookCasts[:i], w.bookCasts[i+1:]...)
		}
	}
	return events
}

func spRunUnbidden(w *World) []CastEvent {
	for n := 0; n < 256; n++ {
		if events := StepObserved(w, nil); len(events) > 0 {
			return events
		}
	}
	return nil
}

// spAt is the world's entry for id, failing when the world holds none, on
// cbAt's own reason (combat_test.go).
func spAt(t *testing.T, w *World, id EntityID) Entity {
	t.Helper()
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		t.Fatalf("world holds no entity %d", id)
	}
	return w.entities[i]
}

// ---------------------------------------------------------------- AC-3

func TestACastSpendsManaOnceAndDamagesTheVictimByAnExactAmount(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	before := w.rng.state
	spRunCast(w, spCast(1, 2, 1))

	if got := spAt(t, w, 1).Mana; got != 15 {
		t.Errorf("caster's mana is %d after the cast, want exactly 15 (20 - the row's own cost of 5)", got)
	}
	gotHP := spAt(t, w, 2).HP
	if gotHP != 86 {
		t.Errorf("victim's health is %d after the cast, want exactly 86 (100 - the roll's own 14)", gotHP)
	}
	if amount := 100 - gotHP; amount < 8 || amount > 16 {
		t.Errorf("the amount taken, %d, falls outside [base, base+spread] = [8, 16]", amount)
	}
	if n := cbDraws(t, before, w.rng.state); n != 2 {
		t.Errorf("the generator drew %d time(s), want exactly 2 (damage and recovery jitter)", n)
	}
}

func TestAVictimExactlyAtMaxRangeIsInReach(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	victim := spEnt(2, 5, 0) // Chebyshev distance exactly 5
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	spRunCast(w, spCast(1, 2, 1))

	if got := spAt(t, w, 1).Mana; got != 15 {
		t.Errorf("caster's mana is %d, want exactly 15 — a victim exactly at MaxRange must still be castable", got)
	}
	if got := spAt(t, w, 2).HP; got != 86 {
		t.Errorf("victim's health is %d, want exactly 86 — a victim exactly at MaxRange must still be castable", got)
	}
}

// ---------------------------------------------------------------- AC-4

// TestAnExplicitCastOneManaShortIsRefused is the manual one-command rule: it
// pays nothing, harms nothing and leaves no retry order behind.
func TestAnExplicitCastOneManaShortIsRefused(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	build := func(t *testing.T) *World {
		t.Helper()
		caster := spMage(1, 0, 0, 60, 50, 4, 1<<1) // one below the row's cost of 5
		victim := spEnt(2, 3, 0)
		return spWorld(t, 9, []SpellRule{spell}, caster, victim)
	}

	ordered := build(t)
	Step(ordered, []Command{spCast(1, 2, 1)})
	if len(ordered.bookCasts) != 0 {
		t.Fatalf("one point short produced retry state %+v", ordered.bookCasts)
	}
	if got := ordered.BookSpellRefusal(1, 2, 1); got != "insufficient mana" {
		t.Fatalf("refusal = %q, want insufficient mana", got)
	}
	if got := spAt(t, ordered, 1).Mana; got != 4 {
		t.Errorf("caster's mana is %d, want its starting 4 unchanged", got)
	}
	if got := spAt(t, ordered, 2).HP; got != 100 {
		t.Errorf("victim's health is %d, want its starting 100 unchanged", got)
	}
}

// ---------------------------------------------------------------- AC-5

// TestANonMageOrAnUnknownSpellCastsNothing is AC-5's two clauses: a caster
// with no mana pool at all casts nothing whatever the cost, and a caster who
// has a pool but never learned the spell casts nothing and pays nothing.
func TestANonMageOrAnUnknownSpellCastsNothing(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	for _, tc := range []struct {
		name   string
		caster Entity
	}{
		{"no mana pool at all", spMage(1, 0, 0, 60, 0, 0, 1<<1)},
		{"a mana pool but the spell was never learned", spMage(1, 0, 0, 60, 50, 20, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			victim := spEnt(2, 3, 0)
			w := spWorld(t, 5, []SpellRule{spell}, tc.caster, victim)

			Step(w, []Command{spCast(1, 2, 1)})

			if got := spAt(t, w, 1).Mana; got != tc.caster.Mana {
				t.Errorf("caster's mana is %d, want its starting %d — the cast must pay nothing",
					got, tc.caster.Mana)
			}
			if got := spAt(t, w, 2).HP; got != 100 {
				t.Errorf("victim's health is %d, want 100 — the cast must land nothing", got)
			}
		})
	}
}

// ---------------------------------------------------------------- AC-6

// TestAVictimOneCellBeyondMaxRangeTakesNothingAndCostsNothing is AC-6, with
// MAGIC-POWER-004's power/30 reach term included.
func TestAVictimOneCellBeyondMaxRangeTakesNothingAndCostsNothing(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, School: 1, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	victim := spEnt(2, 7, 0) // power 30 gives range 6; distance 7 is one past it
	w := spWorld(t, 5, []SpellRule{spell}, caster, victim)

	Step(w, []Command{spCast(1, 2, 1)})

	if got := spAt(t, w, 1).Mana; got != 20 {
		t.Errorf("caster's mana is %d, want its starting 20 — a cast past range must cost nothing", got)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("victim's health is %d, want 100 — a cast past range must land nothing", got)
	}
}

// ---------------------------------------------------------------- AC-7

// TestAKnownSpellNamingNoRowOrANonDamagingOrNonTargetingRowIsANoOp is AC-7's
// three cases.
func TestAKnownSpellNamingNoRowOrANonDamagingOrNonTargetingRowIsANoOp(t *testing.T) {
	dmgUnit := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	notDamaging := SpellRule{ID: 2, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: false}
	notUnit := SpellRule{ID: 3, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: false, Damaging: true}
	table := []SpellRule{dmgUnit, notDamaging, notUnit}
	knows := uint32(1<<1 | 1<<2 | 1<<3 | 1<<9)

	for _, tc := range []struct {
		name    string
		spellID uint32
	}{
		{"a spell id naming no row", 9},
		{"a row that is not damaging", 2},
		{"a row that does not target a unit", 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 0, 0, 60, 50, 20, knows)
			victim := spEnt(2, 3, 0)
			w := spWorld(t, 5, table, caster, victim)

			Step(w, []Command{spCast(1, 2, tc.spellID)})

			if got := spAt(t, w, 1).Mana; got != 20 {
				t.Errorf("caster's mana is %d, want its starting 20", got)
			}
			if got := spAt(t, w, 2).HP; got != 100 {
				t.Errorf("victim's health is %d, want 100", got)
			}
		})
	}
}

func TestACastTouchesOnlyTheTwoEntitiesAndTheGenerator(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	build := func(t *testing.T) *World {
		t.Helper()
		caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
		victim := spEnt(2, 3, 0)
		bystander := spEnt(3, 10, 10)
		bystander.TargetX, bystander.TargetY, bystander.HasTarget = 15, 15, true
		bystander.Speed = 5
		bystander.Group = 7
		return spWorld(t, 5, []SpellRule{spell}, caster, victim, bystander)
	}

	quiet := build(t)
	ordered := build(t)
	spRunCast(ordered, spCast(1, 2, 1))
	for quiet.Tick() < ordered.Tick() {
		Step(quiet, nil)
	}

	if qb, ob := spAt(t, quiet, 3), spAt(t, ordered, 3); qb != ob {
		t.Errorf("the bystander is %+v under a quiet tick and %+v under the cast — the cast reached a third entity",
			qb, ob)
	}
	if quiet.registers != ordered.registers {
		t.Error("the cast moved a script register")
	}
	if quiet.latches != ordered.latches {
		t.Error("the cast moved a script latch")
	}
	if !reflect.DeepEqual(quiet.groups, ordered.groups) {
		t.Error("the cast moved a group")
	}
	if !reflect.DeepEqual(quiet.routes, ordered.routes) {
		t.Error("the cast moved a route")
	}
	before := quiet.rng.state // both worlds share a seed, so both start equal
	if before != build(t).rng.state {
		t.Fatal("fixture: quiet and a fresh build do not share a starting generator state")
	}
	if ordered.rng.state == quiet.rng.state {
		t.Error("the generator did not move at all under a cast that rolled damage")
	}
}

func TestARefusedCastDrawsNothingFromTheGenerator(t *testing.T) {
	dmgUnit := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	notDamaging := SpellRule{ID: 2, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: false}
	notUnit := SpellRule{ID: 3, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: false, Damaging: true}
	table := []SpellRule{dmgUnit, notDamaging, notUnit}
	knows := uint32(1<<1 | 1<<2 | 1<<3)

	for _, tc := range []struct {
		name    string
		mutate  func(*Entity)
		victim  EntityID
		spellID uint32
	}{
		{"the caster is dead", func(c *Entity) { c.HP = -5 }, 2, 1},
		{"the caster is not a mage", func(c *Entity) { c.MaxMana = 0 }, 2, 1},
		{"the caster does not know the spell", func(c *Entity) { c.KnownSpells = 0 }, 2, 1},
		{"the spell id names no row", nil, 2, 9},
		{"the row is not damaging", nil, 2, 2},
		{"the row does not target a unit", nil, 2, 3},
		{"the victim is not held by the world", nil, 99, 1},
		{"the victim is the caster itself", nil, 1, 1},
		{"the victim is not alive", nil, 5, 1},
		{"the victim is one cell past its power-scaled range", nil, 4, 1},
		{"the caster cannot afford the cost", func(c *Entity) { c.Mana = 4 }, 2, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 0, 0, 60, 50, 20, knows)
			if tc.mutate != nil {
				tc.mutate(&caster)
			}
			victim := spEnt(2, 3, 0)
			downed := spEnt(5, 3, 0)
			downed.HP = 0 // alive() is false at exactly zero with a health system
			farVictim := spEnt(4, 7, 0)
			w := spWorld(t, 5, table, caster, victim, downed, farVictim)

			before := w.rng.state
			Step(w, []Command{spCast(1, tc.victim, tc.spellID)})
			if n := cbDraws(t, before, w.rng.state); n != 0 {
				t.Errorf("the generator drew %d time(s) on a refused cast, want 0", n)
			}
		})
	}
}

func TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	caster.AttackRelax = 4
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	cast := spCast(1, 2, 1)
	spRunCast(w, cast)

	if got := spAt(t, w, 1).Mana; got != 15 {
		t.Fatalf("caster's mana is %d after one released cast, want 15", got)
	}
	if got := spAt(t, w, 2).HP; got != 86 {
		t.Fatalf("victim's health is %d after the release, want exactly 86 (100 - 14)", got)
	}
	// The wait comes down at the TOP of a tick and a command is applied after
	// it, so a cast made this tick leaves the full period standing.
	if got := spAt(t, w, 1).CastWait; got != 4 {
		t.Fatalf("the caster's wait is %d after the release, want the decoded recovery 4", got)
	}

	// And the same command applies again once the floor has run out.
	for range 4 {
		Step(w, nil)
	}
	spRunCast(w, cast)
	if got := spAt(t, w, 1).Mana; got != 10 {
		t.Fatalf("caster's mana is %d after the floor ran out, want 10", got)
	}
}

// TestTwoCastsInOneCommandSliceArmOneWindUp is the path
// TestTheCadenceFloorHoldsASecondCastOffForCastPeriodTicks used to walk before
// it was rewritten to admit one command through spRunCast (1001 review F4).
// Its failure message went on naming two casts in one slice for a while after
// the slice was gone, and nothing in the suite carried the case.
//
// TWO GUARDS REFUSE THE SECOND COMMAND, REDUNDANTLY, and neither is the
// cadence floor or the released set: actorCastBusy's pending-cast clause
// (actionguard.go), reached through bookSpellRefusal, and queueBookCast's own
// existing-cast check (spell.go). Measured by removing each: with either one
// left standing this test is green, and it reddens only when both are gone. So
// what it pins is the PROPERTY — one slice, one wind-up — and not either
// implementation of it, which is what a test of a doubly guarded path can
// honestly claim.
//
// Nothing is paid by either command, because a cast pays at its release and not
// at its admission.
func TestTwoCastsInOneCommandSliceArmOneWindUp(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	cast := spCast(1, 2, 1)
	Step(w, []Command{cast, cast})

	if got := len(w.bookCasts); got != 1 {
		t.Fatalf("two casts in one slice armed %d wind-up(s), want 1", got)
	}
	if got := spAt(t, w, 1).Mana; got != 20 {
		t.Fatalf("admission spent %d mana; a cast pays at its release", 20-got)
	}
	released := false
	for range 255 {
		if len(StepObserved(w, nil)) != 0 {
			released = true
			break
		}
	}
	if !released {
		t.Fatal("the one admitted wind-up never released")
	}
	if got := spAt(t, w, 1).Mana; got != 15 {
		t.Fatalf("caster's mana is %d after the slice ran out, want 15 — exactly one cast was paid", got)
	}
}

// A zero-recovery predecessor still yields to a manual replacement before its release.
func TestManualCastRestartsZeroRecoveryReleaseBoundary(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 50, 1<<1)
	caster.AttackRelax = 0
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	cast := spCast(1, 2, 1)
	Step(w, []Command{cast})
	if len(w.bookCasts) != 1 {
		t.Fatalf("the first cast armed %d wind-up(s), want 1", len(w.bookCasts))
	}
	// Advance to the tick BEFORE the release, so the step below is the one
	// whose head applies the cast.
	for range 255 {
		if len(w.bookCasts) == 0 || w.bookCasts[0].Remaining <= 1 {
			break
		}
		Step(w, nil)
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != 1 {
		t.Fatalf("the wind-up did not reach its last tick: %+v", w.bookCasts)
	}
	before := spAt(t, w, 1).Mana

	// A manual replacement takes priority even on the predecessor's release tick.
	Step(w, []Command{cast})

	got := spAt(t, w, 1)
	if got.CastWait != 0 {
		t.Fatalf("the caster's recovery is %d after the release; this fixture needs zero, "+
			"or the busy gate rather than the released set is what refuses below", got.CastWait)
	}
	if spent := before - got.Mana; spent != 0 {
		t.Fatalf("cancelled cast spent %d mana", spent)
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != castWindupTicks(got) {
		t.Fatalf("manual replacement did not start its own wind-up: %+v", w.bookCasts)
	}
}

// TestAnAttackOrderIsTakenOnTheTickACastReleased is the released set at its
// other command arm (pkg/sim/step.go, KindAttack): the cast applies on that
// tick and the order arriving in the same slice is taken beside it, so no order
// is lost to the one tick a cast's release occupies.
func TestAnAttackOrderIsTakenOnTheTickACastReleased(t *testing.T) {
	spell := SpellRule{ID: 1, ManaCost: 5, MaxRange: 5, DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
	caster := spMage(1, 0, 0, 60, 50, 50, 1<<1)
	caster.AttackRelax = 0
	victim := spEnt(2, 3, 0)
	w := spWorld(t, 42, []SpellRule{spell}, caster, victim)

	Step(w, []Command{spCast(1, 2, 1)})
	for range 255 {
		if len(w.bookCasts) == 0 || w.bookCasts[0].Remaining <= 1 {
			break
		}
		Step(w, nil)
	}
	if len(w.bookCasts) != 1 || w.bookCasts[0].Remaining != 1 {
		t.Fatalf("the wind-up did not reach its last tick: %+v", w.bookCasts)
	}

	Step(w, []Command{{Kind: KindAttack, Entity: 1, X: 2}})

	if got := spAt(t, w, 1); !got.HasAttackTarget || got.AttackTarget != 2 {
		t.Fatalf("an attack order was lost on the tick the caster's own cast released "+
			"(target %v/%d)", got.HasAttackTarget, got.AttackTarget)
	}
	if got := spAt(t, w, 2); got.HP >= got.MaxHP {
		t.Fatalf("the cast did not apply on its release tick: victim health %d", got.HP)
	}
}

// TestOneExplicitCastCommandAppliesOnce covers both manual target forms. A
// command is one requested application; only the separate AI producer owns a
// retained casting order.
func TestOneExplicitCastCommandAppliesOnce(t *testing.T) {
	for _, tc := range []struct {
		name    string
		rule    SpellRule
		command Command
	}{
		{
			name: "unit",
			rule: SpellRule{ID: 1, ManaCost: 1, School: 1, MaxRange: 8,
				DamageMin: 1, DamageMax: 1, TargetsUnit: true, Damaging: true},
			command: spCast(1, 2, 1),
		},
		{
			name: "Fire Ball cell",
			rule: SpellRule{ID: 2, ManaCost: 1, School: 1, MaxRange: 8,
				DamageMin: 1, DamageMax: 1, Area: true, Radius: 1, Damaging: true},
			command: Command{Kind: KindCastAt, Entity: 1, X: 4, Y: 1, Spell: 2},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			caster := spMage(1, 1, 1, 50, 100, 100, 1<<uint(tc.rule.ID))
			caster.AttackCharge, caster.AttackRelax = 1, 2
			victim := spEnt(2, 4, 1)
			victim.HP, victim.MaxHP = 1<<20, 1<<20
			w := spWorld(t, 0x474, []SpellRule{tc.rule}, caster, victim)
			releases := 0
			commands := []Command{tc.command}
			for tick := 0; tick < 200; tick++ {
				if events := StepObserved(w, commands); len(events) != 0 {
					releases++
				}
				commands = nil
			}
			if releases != 1 || len(w.bookCasts) != 0 || w.entities[0].Mana != 99 {
				t.Fatalf("one command produced releases=%d casts=%+v mana=%d, want 1/none/99",
					releases, w.bookCasts, w.entities[0].Mana)
			}
		})
	}
}

func TestRestoreOneShotPlayerCastsNormalizesOnlyLegacyPlayerOrders(t *testing.T) {
	entities := []Entity{
		{ID: 1, Owner: SelfSlot},
		{ID: 2, Owner: SelfSlot},
		{ID: 3, Owner: SelfSlot},
		{ID: 4, Owner: 2},
		{ID: 5, Owner: SelfSlot},
	}
	w := &World{entities: entities, bookCasts: []bookCast{
		{Caster: 1, Spell: 2, Phase: bookCharging, Remaining: 4, Retained: true},
		{Caster: 2, Spell: 2, Phase: bookRelaxing, Remaining: 7, Complete: true, Retained: true},
		{Caster: 3, Spell: 2, Phase: bookPending, Retained: true},
		{Caster: 4, Spell: 2, Phase: bookCharging, Remaining: 5, Retained: true},
		{Caster: 5, Spell: 2, Phase: bookCharging, Remaining: 6},
	}}
	if got := w.RestoreOneShotPlayerCasts(); got != 3 {
		t.Fatalf("repaired %d legacy player orders, want 3", got)
	}
	if len(w.bookCasts) != 3 || w.bookCasts[0].Caster != 1 || w.bookCasts[0].Retained ||
		w.bookCasts[1].Caster != 4 || !w.bookCasts[1].Retained ||
		w.bookCasts[2].Caster != 5 || w.bookCasts[2].Retained {
		t.Fatalf("normalized casts = %+v", w.bookCasts)
	}
	if w.entities[1].CastWait != 7 {
		t.Fatalf("completed player order left CastWait %d, want remaining recovery 7", w.entities[1].CastWait)
	}
	if second := w.RestoreOneShotPlayerCasts(); second != 0 {
		t.Fatalf("second repair changed %d orders", second)
	}
}

func TestSpellPowerClampsSkillPlusMindMinus30(t *testing.T) {
	for _, tc := range []struct{ level, mind, want int32 }{
		{0, 0, 0},
		{0, 30, 0},
		{0, 29, 0},
		{0, 31, 1},
		{0, 60, 30},
		{0, 130, 100},
		{0, 131, 101},
		{0, 200, 170},
		{100, 200, 255},
		{255, 255, 255},
		{0, 285, 255},
		{0, 500, 255},
		{0, -50, 0},
		{50, 0, 20},
		{100, 0, 70},
		{100, 30, 100},
		{5, -50, 0},
	} {
		if got := spellPower(tc.level, tc.mind); got != tc.want {
			t.Errorf("spellPower(level %d, mind %d) = %d, want %d", tc.level, tc.mind, got, tc.want)
		}
	}
}

func TestSpellDamageAtZeroPowerReproducesTheDamageColumnsExactly(t *testing.T) {
	for _, tc := range []struct{ dmin, dmax int32 }{
		{4, 8}, {7, 13}, {1, 3}, {8, 16}, {10, 14}, {3, 5}, {5, 15}, {5, 25},
	} {
		base, spread := spellDamage(tc.dmin, tc.dmax, 0)
		if base != int64(tc.dmin) {
			t.Errorf("spellDamage(%d,%d,0) base = %d, want %d", tc.dmin, tc.dmax, base, tc.dmin)
		}
		if base+spread != int64(tc.dmax) {
			t.Errorf("spellDamage(%d,%d,0) base+spread = %d, want %d", tc.dmin, tc.dmax, base+spread, tc.dmax)
		}
	}
}

func TestSpellDamageTruncatesTheDivisionExactly(t *testing.T) {
	base, spread := spellDamage(5, 15, 45)
	if base != 12 {
		t.Errorf("spellDamage(5,15,45) base = %d, want exactly 12 (375/30 truncated)", base)
	}
	if spread != 25 {
		t.Errorf("spellDamage(5,15,45) spread = %d, want exactly 25 (1125/30 truncated, less base)", spread)
	}
}

func TestNewSpelledWorldRefusesAMalformedSpellTable(t *testing.T) {
	ent := spEnt(1, 0, 0)
	for _, tc := range []struct {
		name   string
		spells []SpellRule
	}{
		{"a repeated id", []SpellRule{{ID: 1, TargetsUnit: true}, {ID: 1, Damaging: true}}},
		{"id 0", []SpellRule{{ID: 0}}},
		{"a negative mana cost", []SpellRule{{ID: 1, ManaCost: -1}}},
		{"a negative DamageMin", []SpellRule{{ID: 1, DamageMin: -1}}},
		{"a negative DamageMax", []SpellRule{{ID: 1, DamageMax: -1}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewSpelledWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil, []Entity{ent}, nil, tc.spells)
			if err == nil {
				t.Fatal("NewSpelledWorld built a world over a malformed table")
			}
		})
	}
}

func TestNewSpelledWorldKeepsTheCallersOwnOrder(t *testing.T) {
	in := []SpellRule{
		{ID: 9, TargetsUnit: true, Damaging: true},
		{ID: 2, TargetsUnit: true, Damaging: true},
		{ID: 5, TargetsUnit: true, Damaging: true},
	}
	w, err := NewSpelledWorld(1, Bounds{Width: 4, Height: 4}, ModeCanonical, nil,
		[]Entity{spEnt(1, 0, 0)}, nil, in)
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	if !reflect.DeepEqual(w.spells, in) {
		t.Errorf("the world's table is %+v, want the caller's own order %+v", w.spells, in)
	}
}

func TestNewSpelledWorldWithNoTableCastsNothing(t *testing.T) {
	caster := spMage(1, 0, 0, 60, 50, 20, 1<<1)
	victim := spEnt(2, 3, 0)
	w, err := NewWorld(5, Bounds{Width: 16, Height: 16}, ModeCanonical, nil, []Entity{caster, victim})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}

	Step(w, []Command{spCast(1, 2, 1)})

	if got := spAt(t, w, 1).Mana; got != 20 {
		t.Errorf("caster's mana is %d, want its starting 20 — a world with no table casts nothing", got)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("victim's health is %d, want 100 — a world with no table casts nothing", got)
	}
}
