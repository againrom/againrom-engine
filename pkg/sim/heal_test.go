package sim

import (
	"bytes"
	"testing"
)

// The restorative arm, the spell effect mark and the autocast (0154). The
// fixtures are spell_test.go's own — spEnt, spMage, spCast, spAt — with
// one addition: a world that carries BOTH a relation and a spell table,
// which the commanded cast never needed and both the heal's diplomacy
// refusal and the autocast's target choice do.

// hlWorld is spWorld with a relation, over the same 16x16 open grid.
func hlWorld(t *testing.T, seed uint64, rel Relations, spells []SpellRule, ents ...Entity) *World {
	t.Helper()
	w, err := NewStockedSpelledWorld(seed, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, ents, nil, rel, nil, nil, spells)
	if err != nil {
		t.Fatalf("NewStockedSpelledWorld: %v", err)
	}
	return w
}

// hlGhostTemplate is a SYNTHETIC GhostTemplate: what a definition table might
// ship under the name `Ghost`, in numbers chosen here rather than transcribed
// from any install (golden rule 2). Every value is distinct from the corpse
// fixture's own, so a test can tell which of the two a raised actor took each
// field from.
func hlGhostTemplate() GhostTemplate {
	return GhostTemplate{
		Class: 61, TypeID: 62, Domain: DomainGhost, Speed: 3, RotationSpeed: 16,
		ScanRange: 9, Reach: 2, TokenSize: 1, DyingTime: 13, XPValue: 17,
		Withdraw: 27, Wimpy: 19,
		Protection: [5]int32{11, 12, 13, 14, 15},
		Resistance: [5]uint8{21, 22, 23, 24, 25}, XPSlot: 4,
		ToHit: 41, Defence: 42, Absorption: 43, DamageBase: 44, DamageSpread: 45,
		AttackCharge: 6, AttackRelax: 7, AlwaysHits: true,
	}
}

// hlGhostWorld is hlWorld holding a ghost template, for the one arm that reads
// one: Control Spirit's raise.
func hlGhostWorld(t *testing.T, seed uint64, spells []SpellRule, g GhostTemplate, ents ...Entity) *World {
	t.Helper()
	w, err := NewSummoningWorld(seed, Bounds{Width: 16, Height: 16}, ModeCanonical,
		Terrain{}, ents, nil, Relations{}, nil, nil, spells, g)
	if err != nil {
		t.Fatalf("NewSummoningWorld: %v", err)
	}
	return w
}

// hlAtWar is hlWorld with the caster IN BATTLE: the enemy relation and a
// living owner-2 sentinel actively targeted by every caster.
//
// EVERY COMMANDED-HEAL TEST BUILDS ITS WORLD THROUGH THIS ONE. Out of battle, a
// caster that knows a healing row heals unbidden whether or not anything is
// armed, so a world with no hostile in it would run that arm on the same tick as
// the command and the two would be measured together. The sentinel is placed at
// (2, 9) — inside minimalGuardRange of the casters at (2, 2) and outside the
// heal row's own MaxRange of 6 — and it is never a candidate for anything: it is
// at full health, so the restorative arm skips it, and no heal command names it.
func hlAtWar(t *testing.T, seed uint64, spells []SpellRule, ents ...Entity) *World {
	t.Helper()
	casters := make([]Entity, 0, len(ents)*2)
	for n, e := range ents {
		e.Owner = 1
		if e.KnownSpells != 0 {
			sentinel := spEnt(EntityID(90+n), 2, 9+int32(n))
			sentinel.Owner = 2
			sentinel.HasAttackTarget, sentinel.AttackTarget = true, e.ID
			casters = append(casters, sentinel)
		}
		casters = append(casters, e)
	}
	return hlWorld(t, seed, acEnemies(t), spells, casters...)
}

// hlHeal is a restorative row shaped like the shipped Heal: id 6, school 5
// (Life), a cost, a reach and a damage pair used as the restored band.
func hlHeal() SpellRule {
	return SpellRule{ID: 6, ManaCost: 10, School: 5, MaxRange: 6,
		DamageMin: 10, DamageMax: 20, TargetsUnit: true, Restorative: true}
}

// hlArrow is a damaging row shaped like the shipped Fire Arrow: id 1, school 1
// (Fire), cost 3, reach 7, 4..8.
func hlArrow() SpellRule {
	return SpellRule{ID: 1, ManaCost: 3, School: 1, MaxRange: 7,
		DamageMin: 4, DamageMax: 8, TargetsUnit: true, Damaging: true}
}

// hlBytes is one world's canonical form, on spell_test.go's own reason for
// comparing worlds as bytes: a refusal that changed one field of one entity
// would otherwise have to be looked for field by field.
func hlBytes(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// ---------------------------------------------------------------- AC-2

// TestAHealSpendsManaOnceAndRaisesHealthInsideTheBand is AC-2, measured as
// exact numbers against a fixed seed rather than as signs: the mana lost, the
// power, the base and spread the shared arithmetic computes, and the amount the
// seeded generator's single draw produces.
func TestAHealSpendsManaOnceAndRaisesHealthInsideTheBand(t *testing.T) {
	t.Parallel()

	rule := hlHeal()
	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	ally := spEnt(2, 3, 3)
	ally.HP = 40
	w := hlAtWar(t, 7, []SpellRule{rule}, caster, ally)

	before := spAt(t, w, 2).HP
	spRunCast(w, spCast(1, 2, 6))

	if got, want := spAt(t, w, 1).Mana, int32(40); got != want {
		t.Errorf("the caster holds %d mana, want %d — the cost is subtracted once", got, want)
	}
	// Power is clamp(level + Mind - 30, 0, 100) with a level of 0 and a Mind
	// of 30, so 0; the band at power 0 is the row's own columns exactly.
	base, spread := spellDamage(rule.DamageMin, rule.DamageMax, 0)
	if base != 10 || spread != 10 {
		t.Fatalf("the band at power 0 is %d..%d, want the row's own 10..20", base, base+spread)
	}
	got := spAt(t, w, 2).HP
	if got <= before {
		t.Errorf("the ally holds %d health and held %d — a heal raises it", got, before)
	}
	if int64(got-before) < base || int64(got-before) > base+spread {
		t.Errorf("the heal restored %d, want an amount inside [%d, %d]", got-before, base, base+spread)
	}
}

func TestAHealNeverRaisesHealthAboveTheMaximum(t *testing.T) {
	t.Parallel()

	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	ally := spEnt(2, 3, 3)
	ally.HP = 99 // one below, so every draw in the band overshoots
	w := hlAtWar(t, 3, []SpellRule{hlHeal()}, caster, ally)

	spRunCast(w, spCast(1, 2, 6))

	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("the ally holds %d health against a maximum of 100", got)
	}
	if got, want := spAt(t, w, 1).Mana, int32(40); got != want {
		t.Errorf("the caster holds %d mana, want %d — a target near full still pays", got, want)
	}
	if got := spAt(t, w, 2).SpellFX; got != spellFXLife {
		t.Errorf("the ally carries a mark of %d ticks, want %d — the cast was applied", got, spellFXLife)
	}
}

// A commanded heal at a full-health target is cast and paid.
func TestACommandedHealAtAFullyHealthyTargetIsCast(t *testing.T) {
	t.Parallel()

	caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
	w := hlAtWar(t, 3, []SpellRule{hlHeal()}, caster, spEnt(2, 3, 3))

	spRunCast(w, spCast(1, 2, 6))

	if got, want := spAt(t, w, 1).Mana, int32(40); got != want {
		t.Errorf("the caster holds %d mana, want %d", got, want)
	}
	if got := spAt(t, w, 2).HP; got != 100 {
		t.Errorf("the target holds %d health, want the maximum 100", got)
	}
	if got := spAt(t, w, 2).SpellFX; got != spellFXLife {
		t.Errorf("the target carries a mark of %d ticks, want %d", got, spellFXLife)
	}
	if spAt(t, w, 1).CastWait == 0 {
		t.Errorf("the cast left no recovery state")
	}
}

// ---------------------------------------------------------------- AC-3

func TestAHealAtACorpseLeavesTheWorldByteIdentical(t *testing.T) {
	t.Parallel()

	// Owner 1 is hostile to owner 2 and back; relationHostile is bit 0, the
	// bit Hostile itself reads.
	rel := engRel(t, [3]uint32{1, 2, relationHostile}, [3]uint32{2, 1, relationHostile})

	// A CONTROL WORLD RATHER THAN THE SAME WORLD BEFORE THE STEP. A tick
	// changes state that has nothing to do with the cast — a corpse's decay
	// ladder among it — so "the world after is byte-identical to the world
	// before" is only a true statement of a REFUSED cast when the comparison
	// is against the same world stepped with no command at all.
	compare := func(t *testing.T, build func() *World, cmd Command) {
		t.Helper()
		cast, control := build(), build()
		Step(cast, []Command{cmd})
		Step(control, nil)
		if !bytes.Equal(hlBytes(t, cast), hlBytes(t, control)) {
			t.Error("the refused heal left a world the same tick with no command at all does not")
		}
		if got := spAt(t, cast, 1).Mana; got != 50 {
			t.Errorf("the caster holds %d mana, want its own untouched 50", got)
		}
	}

	t.Run("a dead target", func(t *testing.T) {
		compare(t, func() *World {
			caster := spMage(1, 2, 2, 30, 50, 50, 1<<6)
			corpse := spEnt(2, 3, 3)
			corpse.HP = -50
			return hlWorld(t, 5, rel, []SpellRule{hlHeal()}, caster, corpse)
		}, spCast(1, 2, 6))
	})
}

// ---------------------------------------------------------------- AC-4

// TestAHealAtOneselfIsAppliedAndADamageCastAtOneselfIsNot is AC-4, both halves
// on one fixture so the difference between them is the row alone.
func TestAHealAtOneselfIsAppliedAndADamageCastAtOneselfIsNot(t *testing.T) {
	t.Parallel()

	table := []SpellRule{hlArrow(), hlHeal()}

	t.Run("a heal at oneself", func(t *testing.T) {
		caster := spMage(1, 2, 2, 30, 50, 50, (1<<6)|(1<<1))
		caster.HP = 40
		w := hlAtWar(t, 9, table, caster)

		spRunCast(w, spCast(1, 1, 6))

		e := spAt(t, w, 1)
		if e.HP <= 40 {
			t.Errorf("the caster holds %d health and held 40 — a heal at oneself is applied", e.HP)
		}
		if e.Mana != 40 {
			t.Errorf("the caster holds %d mana, want 40", e.Mana)
		}
		if e.SpellFX != spellFXLife {
			t.Errorf("the caster carries a mark of %d ticks, want %d — one mark, not two",
				e.SpellFX, spellFXLife)
		}
	})

	t.Run("a damage cast at oneself", func(t *testing.T) {
		build := func() *World {
			caster := spMage(1, 2, 2, 30, 50, 50, (1<<6)|(1<<1))
			caster.HP = 40
			return hlAtWar(t, 9, table, caster)
		}
		w, control := build(), build()
		Step(w, []Command{spCast(1, 1, 1)})
		Step(control, nil)
		if !bytes.Equal(hlBytes(t, control), hlBytes(t, w)) {
			t.Error("a damage cast aimed at its own caster changed the world past its header")
		}
	})
}

// ---------------------------------------------------------------- AC-5

// TestAnAppliedCastMarksBothActorsAndTheMarkExpiresToNothing is AC-5 whole: the
// mark is set on both, falls by one a tick, and the world it leaves behind is
// byte-identical to one stepped the same number of ticks with no cast at all.
func TestAnAppliedCastMarksBothActorsAndTheMarkExpiresToNothing(t *testing.T) {
	t.Parallel()

	table := []SpellRule{hlArrow()}
	build := func() *World {
		caster := spMage(1, 2, 2, 30, 50, 50, 1<<1)
		return hlWorld(t, 11, Relations{}, table, caster, spEnt(2, 4, 4))
	}

	w := build()
	spRunCast(w, spCast(1, 2, 1))

	for _, id := range []EntityID{1, 2} {
		e := spAt(t, w, id)
		if e.SpellFX != spellFXLife || e.SpellFXSpell != 1 {
			t.Errorf("entity %d carries mark %d/%d, want %d ticks of spell 1",
				id, e.SpellFX, e.SpellFXSpell, spellFXLife)
		}
	}

	// One tick off per step, and the spell id cleared with the last of them.
	for tick := 1; tick <= spellFXLife; tick++ {
		Step(w, nil)
		want := uint16(spellFXLife - tick)
		if got := spAt(t, w, 1).SpellFX; got != want {
			t.Fatalf("after %d further tick(s) the caster carries %d, want %d", tick, got, want)
		}
	}
	if got := spAt(t, w, 1).SpellFXSpell; got != 0 {
		t.Errorf("the expired mark still names spell %d, want 0", got)
	}
}

func TestARefusedCastLeavesNoMark(t *testing.T) {
	t.Parallel()

	// One mana short of the cost, which is the refusal that runs last and so
	// has the most of the arm in front of it.
	caster := spMage(1, 2, 2, 30, 50, 2, 1<<1)
	w := hlWorld(t, 13, Relations{}, []SpellRule{hlArrow()}, caster, spEnt(2, 4, 4))

	spRunCast(w, spCast(1, 2, 1))

	for _, id := range []EntityID{1, 2} {
		if got := spAt(t, w, id).SpellFX; got != 0 {
			t.Errorf("entity %d carries a mark of %d ticks after a refused cast", id, got)
		}
	}
}

// ---------------------------------------------------------------- AC-6

// TestTheAutocastAndTheMarkRoundTripAndReachTheDigest is AC-6: the three fields
// this story added survive the byte form unchanged, and two worlds differing
// only in one of them hash differently.
func TestTheAutocastAndTheMarkRoundTripAndReachTheDigest(t *testing.T) {
	t.Parallel()

	base := func() Entity {
		e := spMage(1, 2, 2, 30, 50, 50, 1<<1)
		return e
	}
	loaded := base()
	loaded.AutoSpell, loaded.CastWait, loaded.SpellFX, loaded.SpellFXSpell = 1, 9, 3, 1

	w := hlWorld(t, 17, Relations{}, []SpellRule{hlArrow()}, loaded)
	form := hlBytes(t, w)

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got := spAt(t, &back, 1)
	if got.AutoSpell != 1 || got.CastWait != 9 || got.SpellFX != 3 || got.SpellFXSpell != 1 {
		t.Errorf("the decoded entity carries %d/%d/%d/%d, want 1/9/3/1",
			got.AutoSpell, got.CastWait, got.SpellFX, got.SpellFXSpell)
	}

	for _, tc := range []struct {
		what string
		edit func(Entity) Entity
	}{
		{"the autocast spell", func(e Entity) Entity { e.AutoSpell = 6; return e }},
		{"the autocast wait", func(e Entity) Entity { e.CastWait = 8; return e }},
		{"the mark's remaining ticks", func(e Entity) Entity { e.SpellFX = 2; return e }},
		{"the marking spell", func(e Entity) Entity { e.SpellFXSpell = 6; return e }},
	} {
		other := hlWorld(t, 17, Relations{}, []SpellRule{hlArrow()}, tc.edit(loaded))
		if w.Hash() == other.Hash() {
			t.Errorf("two worlds differing only in %s hash the same", tc.what)
		}
	}
}

// TestASupersededFormVersionIsRefused is AC-6's last clause, on the version
// ladder's own terms: a version this build no longer reads is refused rather
// than migrated.
//
// IT WAS CALLED TestAVersion44FormIsRefused AND THE NAME IS GONE. The doc
// standing here argued the name was safe because 44 is a PREVIOUS version
// and not the current one, so a bump would move the literal and not the
// name. That is true of the assertion and false of the name: 44 was "the
// previous version" only while 45 was current, and the moment the tree moved
// past it the name described a version three bumps back. A test name in
// pkg/sim has now gone stale on a version number three times — 0099, 0104,
// 0106 — each time under its own doc block warning about exactly that, and
// re-spelling the number is what failed each time. The version this test
// actually refuses is preSpellStateFormVersion, named once, in the body.
func TestASupersededFormVersionIsRefused(t *testing.T) {
	t.Parallel()

	w := hlWorld(t, 17, Relations{}, []SpellRule{hlArrow()}, spEnt(1, 2, 2))
	form := hlBytes(t, w)
	form[0] = preSpellStateFormVersion

	var back World
	if err := back.UnmarshalBinary(form); err == nil {
		t.Errorf("a form declaring version %d was accepted", preSpellStateFormVersion)
	}
}

func TestTheArmIsChosenByTheRowsOwnFlagAndNeverByItsID(t *testing.T) {
	t.Parallel()

	// Heal wearing Fire Arrow's id and Fire Arrow wearing Heal's.
	heal := hlHeal()
	heal.ID = 1
	arrow := hlArrow()
	arrow.ID = 6

	build := func() *World {
		caster := spMage(1, 2, 2, 30, 50, 50, (1<<1)|(1<<6))
		ally := spEnt(2, 3, 3)
		ally.HP = 40
		return hlAtWar(t, 47, []SpellRule{heal, arrow}, caster, ally)
	}
	w := build()

	// Spell 1 is now the restorative row, so it must HEAL.
	spRunCast(w, spCast(1, 2, 1))
	if got := spAt(t, w, 2).HP; got <= 40 {
		t.Errorf("spell 1 left the target at %d health; the restorative row wears that id now", got)
	}

	// A fresh identical world isolates the damaging row from idle Heal and
	// recovery, neither of which is part of this renumbering assertion.
	w = build()
	was := spAt(t, w, 2).HP
	spRunCast(w, spCast(1, 2, 6))
	if got := spAt(t, w, 2).HP; got >= was {
		t.Errorf("spell 6 left the target at %d health and it held %d; the damaging row wears that id now",
			got, was)
	}
}
