package sim

// Instant opcodes 21, 24 and 29 (0165): the mission script's own casts and the
// area effects they leave standing on cells.
//
// Every fixture here is a world and a hand-built script, on scriptitem_test.go's
// own precedent — the three arms take cells, a spell id, a power and a unit
// reference, all plain state in this tree, so nothing here needs a map.
//
// THE SPELL TABLE IS THE FIXTURE'S OWN and its rows are the shipped table's
// SHAPES rather than its values: an area row with an area duration, a point row
// that damages, a point row that heals, and a point row that is neither — the
// last being the shape all six spells the shipped instant-24 nodes name are in
// (spec SC-1).

import (
	"reflect"
	"testing"
)

// The four rows every fixture here draws on, and the entity ids.
const (
	scAreaSpell   uint16 = 19 // an area row, like every spell instant 21 casts
	scDamageSpell uint16 = 13 // a point row that damages
	scHealSpell   uint16 = 6  // a point row that heals
	scInertSpell  uint16 = 20 // a point row this build has no arm for

	scCaster EntityID = 1
	scTarget EntityID = 2
)

// scSpells is the fixture table. The area row's duration column is 15, which is
// what the shipped Wall of Earth, Wall of Fire, Freezing Cloud and Poison Cloud
// all carry.
func scSpells() []SpellRule {
	return []SpellRule{
		{ID: scHealSpell, School: 2, MaxRange: 6, DamageMin: 8, DamageMax: 16,
			TargetsUnit: true, Restorative: true},
		{ID: scDamageSpell, School: 3, MaxRange: 8, DamageMin: 5, DamageMax: 15,
			TargetsUnit: true, Damaging: true},
		// Distribution 4 is the shipped Wall of Earth row's own column: it is
		// what selects the wall paint, where the spell id used to.
		{ID: scAreaSpell, School: 4, MaxRange: 6, Area: true, Distribution: 4, AreaDuration: 15},
		{ID: scInertSpell, School: 4, MaxRange: 5, TargetsUnit: true},
	}
}

// scWorld is a two-entity world running s over the fixture spell table.
func scWorld(t *testing.T, s *Script) *World {
	t.Helper()
	w, err := NewSpelledWorld(1, Bounds{Width: 80, Height: 80}, ModeCanonical, nil,
		[]Entity{
			{ID: scCaster, X: 1, Y: 1, HP: 40, MaxHP: 40},
			{ID: scTarget, X: 40, Y: 41, HP: 40, MaxHP: 40},
		}, s, scSpells())
	if err != nil {
		t.Fatalf("NewSpelledWorld: %v", err)
	}
	return w
}

// cellNode is one instant-21 record: cast at a cell.
func cellNode(fromX, fromY, toX, toY int32, spell uint16, power int32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantCastAtCell,
		Args: [scriptParams]int32{fromX, fromY, toX, toY, int32(spell), power}}
}

// unitNode is one instant-24 record: cast at a unit. The target is a REFERENCE,
// so the spell and the power pack at 2 and 3 rather than at 4 and 5.
func unitNode(fromX, fromY int32, spell uint16, power int32, target EntityID) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantCastAtUnit, Unit: target, HasUnit: true,
		Args: [scriptParams]int32{fromX, fromY, int32(spell), power}}
}

// timeNode is one instant-29 record: set a remaining lifetime.
func timeNode(x, y, spell, duration int32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantCellEffectAge,
		Args: [scriptParams]int32{x, y, spell, duration}}
}

// scPass runs the script pass and stops on the tick it fired, BEFORE the pending
// cast has been resolved: the pass is on phase 6 and the resolve is at the head
// of the next tick, so seven ticks reach one and not the other. It is what every
// case measuring the pending record itself uses.
func scPass(t *testing.T, w *World) {
	t.Helper()
	scriptTicks(w, scriptPassPhase+1, nil)
	if !w.ScriptLatched(0) {
		t.Fatal("the trigger did not fire; every case here measures what happens when it does")
	}
	if w.Tick() != scriptPassPhase+1 {
		t.Fatalf("the world is at tick %d, want %d — this helper must stop before the resolve",
			w.Tick(), scriptPassPhase+1)
	}
}

func TestAResolvedTemporaryCasterReportsItsSourceAndDestinationOnce(t *testing.T) {
	t.Parallel()

	w := scWorld(t, nil)
	w.casts = []scriptCast{{FromX: 7, FromY: 8, ToX: 30, ToY: 31,
		Spell: uint8(scAreaSpell), Power: 20}}

	report := StepReported(w, nil)
	want := ScriptCastEvent{Spell: scAreaSpell, FromX: 7, FromY: 8, ToX: 30, ToY: 31}
	if len(report.ScriptCasts) != 1 || report.ScriptCasts[0] != want {
		t.Fatalf("script cast events = %+v, want [%+v]", report.ScriptCasts, want)
	}
	wantRaw := uint16((15 << 4) + (20<<4)/10)
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != wantRaw {
		t.Fatalf("reported script cloud = %+v, want raw counter %d", got, wantRaw)
	}
	if got := StepReported(w, nil).ScriptCasts; len(got) != 0 {
		t.Fatalf("consumed temporary caster repeated as %+v", got)
	}
}

func TestARefusedTemporaryCasterReportsNoEvent(t *testing.T) {
	t.Parallel()

	w := scWorld(t, nil)
	w.casts = []scriptCast{{FromX: 7, FromY: 8, ToX: 30, ToY: 31,
		Spell: uint8(scInertSpell), Power: 20}}
	if got := StepReported(w, nil).ScriptCasts; len(got) != 0 {
		t.Fatalf("refused temporary cast reported %+v", got)
	}
}

// ---------------------------------------------------------------- AC-1

// TestInstantTwentyOneAppendsOneCastCarryingItsOwnBytes is AC-1: the record is
// the node's own six parameters at the widths the constructor stores them at,
// and an authored power of 0 is stored as the substituted 99.
func TestInstantTwentyOneAppendsOneCastCarryingItsOwnBytes(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{cellNode(42, 33, 38, 33, scAreaSpell, 0)}, 0))
	scPass(t, w)

	want := []ScriptCast{{FromX: 42, FromY: 33, ToX: 38, ToY: 33,
		Spell: scAreaSpell, Power: scriptCastDefaultPower}}
	if got := w.ScriptCasts(); !reflect.DeepEqual(got, want) {
		t.Errorf("the world holds %+v, want %+v", got, want)
	}
}

// TestAnAuthoredPowerIsCarriedAndOnlyZeroIsSubstituted is AC-1's second clause:
// the 99 is a substitution for zero and not a default, so a node authoring 1
// stores 1. Both shapes appear in the shipped campaign — every instant-21 node
// authors 0 and instant-24 nodes author 0, 1, 99 and 100.
func TestAnAuthoredPowerIsCarriedAndOnlyZeroIsSubstituted(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{
		cellNode(1, 1, 5, 5, scAreaSpell, 1),
		cellNode(1, 1, 6, 6, scAreaSpell, 100),
	}, 0, 1))
	scPass(t, w)

	got := w.ScriptCasts()
	if len(got) != 2 {
		t.Fatalf("the world holds %d pending cast(s), want 2", len(got))
	}
	if got[0].Power != 1 || got[1].Power != 100 {
		t.Errorf("the two powers are %d and %d, want 1 and 100", got[0].Power, got[1].Power)
	}
}

// ---------------------------------------------------------------- AC-2

// TestInstantTwentyFourNamesItsTargetAndAnUnresolvedReferenceCastsNothing is
// AC-2, both clauses in one case: the resolved node appends a cast naming the
// entity, and the node whose reference did not resolve appends nothing at all.
func TestInstantTwentyFourNamesItsTargetAndAnUnresolvedReferenceCastsNothing(t *testing.T) {
	t.Parallel()

	unbound := ScriptInstant{Op: ScriptInstantCastAtUnit,
		Args: [scriptParams]int32{20, 15, int32(scInertSpell), 99}}
	w := scWorld(t, fireOnce(t, []ScriptInstant{
		unitNode(20, 15, scInertSpell, 99, scTarget),
		unbound,
	}, 0, 1))
	scPass(t, w)

	want := []ScriptCast{{FromX: 20, FromY: 15, Spell: scInertSpell, Power: 99,
		Target: scTarget, AtUnit: true}}
	if got := w.ScriptCasts(); !reflect.DeepEqual(got, want) {
		t.Errorf("the world holds %+v, want exactly one cast %+v", got, want)
	}
}

// ---------------------------------------------------------------- AC-3

func TestAPendingCastResolvesOnTheNextTickAndLeavesAnAreaEffect(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{cellNode(42, 33, 38, 33, scAreaSpell, 0)}, 0))
	scPass(t, w)

	if len(w.CellEffects()) != 0 {
		t.Fatalf("an area effect stands before the cast was resolved: %+v", w.CellEffects())
	}
	Step(w, nil)

	if got := w.ScriptCasts(); len(got) != 0 {
		t.Errorf("the world still holds %+v; a resolved cast is removed", got)
	}
	life := uint16((15 << 4) + (99<<4)/10)
	want := []CellEffect{{X: 38, Y: 33, Spell: scAreaSpell, Remaining: life,
		Power: 99, Mode: areaModeCloud, Cells: [][2]int32{{37, 31}, {38, 31}, {37, 32}, {38, 32},
			{37, 33}, {38, 33}, {37, 34}, {38, 34}, {37, 35}, {38, 35}}}}
	if got := w.CellEffects(); !reflect.DeepEqual(got, want) {
		t.Errorf("the world holds %+v, want %+v", got, want)
	}
}

func TestAUnitCastOfAnAreaRowLandsAtTheTargetsOwnCell(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, scAreaSpell, 0, scTarget)}, 0))
	scPass(t, w)
	Step(w, nil)

	got := w.CellEffects()
	if len(got) != 1 {
		t.Fatalf("the world holds %d area effect(s), want 1", len(got))
	}
	if got[0].X != 40 || got[0].Y != 41 {
		t.Errorf("the effect stands at (%d, %d), want the target's own cell (40, 41)", got[0].X, got[0].Y)
	}
}

// ---------------------------------------------------------------- AC-4

// TestACellCastOfAPointRowLandsNothing is AC-4. A point effect requires a target
// unit and a destination cell is not one, which is what the original says by
// printing an error and building nothing.
func TestACellCastOfAPointRowLandsNothing(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what  string
		spell uint16
	}{
		{"a damaging point row", scDamageSpell},
		{"a restorative point row", scHealSpell},
		{"a point row this build has no arm for", scInertSpell},
	} {
		t.Run(tc.what, func(t *testing.T) {
			w := scWorld(t, fireOnce(t, []ScriptInstant{cellNode(1, 1, 40, 41, tc.spell, 0)}, 0))
			before := entityCopies(w)
			scPass(t, w)
			Step(w, nil)

			if got := w.CellEffects(); len(got) != 0 {
				t.Errorf("%s placed %+v", tc.what, got)
			}
			if got := entityCopies(w); !reflect.DeepEqual(got, before) {
				t.Errorf("%s changed an entity:\n before %+v\n after  %+v", tc.what, before, got)
			}
		})
	}
}

// entityCopies is the world's entities as values, so a case can say "no entity
// moved" without reaching into the world twice.
func entityCopies(w *World) []Entity { return w.Entities() }

// ---------------------------------------------------------------- AC-5

// TestAUnitCastAppliesTheTwoArmsThisBuildHasAndNothingElse is AC-5, all three
// clauses. The inert case is spec SC-1's own subject and the one the whole
// shipped instant-24 corpus falls into.
func TestAUnitCastAppliesTheTwoArmsThisBuildHasAndNothingElse(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		what  string
		spell uint16
		start int32
		want  string
	}{
		{"a damaging row lowers the target's health", scDamageSpell, 40, "lower"},
		{"a restorative row raises it", scHealSpell, 10, "higher"},
		{"a row that is neither leaves every entity as it was", scInertSpell, 40, "same"},
	} {
		t.Run(tc.what, func(t *testing.T) {
			w := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, tc.spell, 0, scTarget)}, 0))
			ti := indexOfEntity(w.entities, scTarget)
			w.entities[ti].HP = tc.start
			before := entityCopies(w)

			scPass(t, w)
			Step(w, nil)

			hp := w.entities[ti].HP
			switch tc.want {
			case "lower":
				if hp >= tc.start {
					t.Errorf("the target is at %d health, want less than %d", hp, tc.start)
				}
			case "higher":
				if hp <= tc.start {
					t.Errorf("the target is at %d health, want more than %d", hp, tc.start)
				}
			case "same":
				if got := entityCopies(w); !reflect.DeepEqual(got, before) {
					t.Errorf("an entity changed:\n before %+v\n after  %+v", before, got)
				}
			}
		})
	}
}

// ---------------------------------------------------------------- AC-6

// TestAnAreaEffectsLifetimeFallsByOneATickAndIsRemovedAtZero is AC-6's first
// clause: the effect is placed with a known lifetime and every tick takes one
// off it.
func TestAnAreaEffectsLifetimeFallsByOneATickAndIsRemovedAtZero(t *testing.T) {
	t.Parallel()

	w := scWorld(t, mustScript(t, nil, nil, nil))
	if !w.placeCellEffect(cellKey(7, 9), scAreaSpell, 3) {
		t.Fatal("placeCellEffect refused a well-formed effect")
	}
	for want := uint16(2); want >= 1; want-- {
		Step(w, nil)
		got := w.CellEffects()
		if len(got) != 1 || got[0].Remaining != want {
			t.Fatalf("after a tick the world holds %+v, want one effect with %d left", got, want)
		}
	}
	Step(w, nil)
	if got := w.CellEffects(); len(got) != 0 {
		t.Errorf("the world still holds %+v; an effect whose last tick ran is removed", got)
	}
}

// TestAnEffectWrittenToZeroIsRemovedRatherThanWrapped is AC-6's second clause.
// Instant 29's store has no test in front of it, so a duration of zero reaches
// the field; a countdown that subtracted first would turn that into 65535 and
// give an expired effect the longest life in the world.
func TestAnEffectWrittenToZeroIsRemovedRatherThanWrapped(t *testing.T) {
	t.Parallel()

	w := scWorld(t, mustScript(t, nil, nil, nil))
	w.placeCellEffect(cellKey(7, 9), scAreaSpell, 500)
	w.setCellEffectTime(7, 9, int32(scAreaSpell), 0)
	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining != 0 {
		t.Fatalf("the world holds %+v, want one effect at 0 — the write has no test in front of it", got)
	}
	Step(w, nil)
	if got := w.CellEffects(); len(got) != 0 {
		t.Errorf("the world holds %+v, want nothing", got)
	}
}

func TestASeventhEffectOnOneCellIsDropped(t *testing.T) {
	t.Parallel()

	w := scWorld(t, mustScript(t, nil, nil, nil))
	for i := 0; i < cellEffectSlots; i++ {
		if !w.placeCellEffect(cellKey(3, 4), uint16(i+1), 100) {
			t.Fatalf("slot %d was refused", i)
		}
	}
	if w.placeCellEffect(cellKey(3, 4), 99, 100) {
		t.Error("a seventh effect on one cell was accepted")
	}
	if got := len(w.CellEffects()); got != cellEffectSlots {
		t.Errorf("the cell holds %d effect(s), want %d", got, cellEffectSlots)
	}
	// A different cell is unaffected: the cap is per key and not per world.
	if !w.placeCellEffect(cellKey(3, 5), 99, 100) {
		t.Error("an effect on a different cell was refused by the other cell's cap")
	}
}

func TestTheEffectSliceStaysSortedByKey(t *testing.T) {
	t.Parallel()

	w := scWorld(t, mustScript(t, nil, nil, nil))
	for _, c := range [][2]int32{{9, 9}, {1, 1}, {5, 5}, {1, 1}} {
		w.placeCellEffect(cellKey(c[0], c[1]), scAreaSpell, 100)
	}
	got := w.CellEffects()
	for i := 1; i < len(got); i++ {
		if cellKey(got[i].X, got[i].Y) < cellKey(got[i-1].X, got[i-1].Y) {
			t.Fatalf("effect %d is on a lower key than effect %d: %+v", i, i-1, got)
		}
	}
	if len(got) != 4 {
		t.Errorf("the world holds %d effect(s), want 4", len(got))
	}
}

// ---------------------------------------------------------------- AC-7

// TestInstantTwentyNineWritesEveryMatchOnItsOwnCellAndNoOther is AC-7's first
// two clauses: all matches on the keyed cell, no effect anywhere else.
func TestInstantTwentyNineWritesEveryMatchOnItsOwnCellAndNoOther(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{timeNode(38, 33, int32(scAreaSpell), 30000)}, 0))
	w.placeCellEffect(cellKey(38, 33), scAreaSpell, 100)
	w.placeCellEffect(cellKey(38, 33), scAreaSpell, 100)
	w.placeCellEffect(cellKey(38, 33), scDamageSpell, 100)
	w.placeCellEffect(cellKey(39, 33), scAreaSpell, 100)
	scPass(t, w)

	got := w.CellEffects()
	if len(got) != 4 {
		t.Fatalf("the world holds %d effect(s), want 4", len(got))
	}
	// The pass itself takes a tick off every standing effect before the
	// instant runs, so the untouched ones are at 100 minus the seven ticks
	// scPass advances; only the relation between them is asserted.
	matched := 0
	for _, e := range got {
		if e.Remaining == 30000 {
			matched++
			if e.Spell != scAreaSpell || e.X != 38 || e.Y != 33 {
				t.Errorf("effect %+v took the write and should not have", e)
			}
		}
	}
	if matched != 2 {
		t.Errorf("%d effect(s) took the write, want the 2 matching ones", matched)
	}
}

// TestAnIdentifierAboveAByteMatchesNothing is AC-7's third clause. The arm
// zero-extends the effect's own id BYTE and compares it with the WHOLE dword, so
// 256 + 19 matches no effect that a byte-truncated comparison would have matched.
func TestAnIdentifierAboveAByteMatchesNothing(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{timeNode(38, 33, 256+int32(scAreaSpell), 30000)}, 0))
	w.placeCellEffect(cellKey(38, 33), scAreaSpell, 100)
	scPass(t, w)

	if got := w.CellEffects(); len(got) != 1 || got[0].Remaining == 30000 {
		t.Errorf("the world holds %+v; a comparison against the full dword matches nothing here", got)
	}
}

// ---------------------------------------------------------------- AC-8

// TestTheCellKeyIsSixteenBitArithmetic is AC-8. The key is
// `(u16)y << 8` plus `(u16)x` computed in sixteen bits, so an x above 255 CARRIES
// into the y byte instead of being masked off — which an OR would not do.
func TestTheCellKeyIsSixteenBitArithmetic(t *testing.T) {
	t.Parallel()

	if a, b := cellKey(300, 0), cellKey(44, 1); a != b {
		t.Errorf("cellKey(300, 0) is %d and cellKey(44, 1) is %d; the ADD carries into the y byte", a, b)
	}
	if a, b := cellKey(300, 0), cellKey(44, 0); a == b {
		t.Error("cellKey(300, 0) equals cellKey(44, 0); an OR would mask the carry away and this is an ADD")
	}
	// And the key is read back the way it was built, which is the whole of
	// where an effect stands.
	x, y := keyCell(cellKey(130, 94))
	if x != 130 || y != 94 {
		t.Errorf("keyCell(cellKey(130, 94)) is (%d, %d), want (130, 94)", x, y)
	}
}

func TestAScriptCastIsNotReportedAsACastEvent(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, scDamageSpell, 0, scTarget)}, 0))
	scPass(t, w)
	if got := StepObserved(w, nil); len(got) != 0 {
		t.Errorf("the resolve reported %+v, want nothing", got)
	}
	// And the damage still landed, so this is not passing by casting nothing.
	if hp := w.entities[indexOfEntity(w.entities, scTarget)].HP; hp >= 40 {
		t.Errorf("the target is at %d health; the cast did not land and this test measured nothing", hp)
	}
}

func TestAnObservedStepAndAnUnobservedStepAreOneWorld(t *testing.T) {
	t.Parallel()

	build := func() *World {
		w := scWorld(t, fireOnce(t, []ScriptInstant{
			cellNode(1, 1, 38, 33, scAreaSpell, 0),
			unitNode(1, 1, scDamageSpell, 0, scTarget),
		}, 0, 1))
		scPass(t, w)
		return w
	}
	plain, observed := build(), build()
	Step(plain, nil)
	StepObserved(observed, nil)
	if plain.Hash() != observed.Hash() {
		t.Errorf("an observed step hashes %#016x and an unobserved one %#016x", observed.Hash(), plain.Hash())
	}
}

func TestACastThatLandsNothingLeavesTheWorldWhereItWas(t *testing.T) {
	t.Parallel()

	with := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, scInertSpell, 0, scTarget)}, 0))
	// The control fires the same trigger over an instant that reaches no
	// entity at all — register 5 set to the 0 it already holds — so the two
	// worlds differ in the cast and in nothing else about how their scripts run.
	without := scWorld(t, fireOnce(t, []ScriptInstant{
		{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{5, 0}},
	}, 0))
	scriptTicks(with, 40, nil)
	scriptTicks(without, 40, nil)

	if len(with.ScriptCasts()) != 0 {
		t.Fatalf("the resolved cast is still pending: %+v", with.ScriptCasts())
	}
	if !reflect.DeepEqual(with.Entities(), without.Entities()) {
		t.Errorf("the two worlds' entities differ:\n with    %+v\n without %+v",
			with.Entities(), without.Entities())
	}
	if len(with.CellEffects()) != 0 {
		t.Errorf("a point row this build has no arm for placed %+v", with.CellEffects())
	}
}

func TestNothingHereReadsAnOwner(t *testing.T) {
	t.Parallel()

	hp := func(owner uint32) int32 {
		w := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, scDamageSpell, 0, scTarget)}, 0))
		ti := indexOfEntity(w.entities, scTarget)
		w.entities[ti].Owner = owner
		scriptTicks(w, 40, nil)
		return w.entities[ti].HP
	}
	if a, b := hp(0), hp(7); a != b {
		t.Errorf("a target in slot 0 ends at %d health and one in slot 7 at %d", a, b)
	}
}

// ---------------------------------------------------------------- the tables

// TestTheThreeOpcodesAreReportedAsSupported is AC-11's own half inside this
// package: the compile-time table and the runtime dispatch are one answer, so a
// script naming these three arms reports no gap.
func TestTheThreeOpcodesAreReportedAsSupported(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{
		cellNode(1, 1, 2, 2, scAreaSpell, 0),
		unitNode(1, 1, scInertSpell, 0, scTarget),
		timeNode(2, 2, int32(scAreaSpell), 1),
	}, 0, 1, 2)
	if got := s.Unsupported(); len(got) != 0 {
		t.Errorf("the script reports %+v unsupported, want nothing", got)
	}
}

func TestASpellIdNamingNoRowLandsNothing(t *testing.T) {
	t.Parallel()

	w := scWorld(t, fireOnce(t, []ScriptInstant{cellNode(1, 1, 2, 2, 250, 0)}, 0))
	scriptTicks(w, 40, nil)

	if got := w.CellEffects(); len(got) != 0 {
		t.Errorf("a spell no row names placed %+v", got)
	}
	if got := w.ScriptCasts(); len(got) != 0 {
		t.Errorf("the cast is still pending: %+v", got)
	}
}

func TestAUnitCastWhoseTargetIsGoneLandsNothing(t *testing.T) {
	t.Parallel()

	for _, spell := range []uint16{scAreaSpell, scDamageSpell} {
		w := scWorld(t, fireOnce(t, []ScriptInstant{unitNode(1, 1, spell, 0, scTarget)}, 0))
		scPass(t, w)
		w.entities[indexOfEntity(w.entities, scTarget)].HP = minHP
		Step(w, nil)

		if got := w.CellEffects(); len(got) != 0 {
			t.Errorf("spell %d placed %+v at a felled target", spell, got)
		}
		if got := w.ScriptCasts(); len(got) != 0 {
			t.Errorf("spell %d left %+v pending", spell, got)
		}
	}
}
