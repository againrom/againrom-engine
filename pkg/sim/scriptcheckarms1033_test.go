package sim

// Check opcodes 4, 16 and 21, and instant opcode 26 — the four arms 1033
// implements. Register behaviour first, then a trigger reading the register,
// on scriptcheckarms_test.go's own precedent (1029): a register no trigger
// reads is invisible, so every arm here is witnessed twice.
//
// Fixtures reuse caSubject, caTarget, caBounds, caTicks, caRegister,
// caConstant, mustScript, scriptTicks, pair, acts and constCheck from
// scriptcheckarms_test.go and script_test.go. Nothing here reads a map.

import "testing"

// ------------------------------------------------------------- check 4

// healthNode is the compiled check-4 record under test, spelt once. gate is
// the node's own first plain parameter, tested against scriptCheckHealthGate.
func healthNode(unit EntityID, gate int32) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckHealth, Register: caRegister, Unit: unit, HasUnit: true,
		Args: [scriptParams]int32{gate}}
}

// TestTheHealthArmAnswersTheUnitsCurrentHealthOnTheGate is B1's register half.
func TestTheHealthArmAnswersTheUnitsCurrentHealthOnTheGate(t *testing.T) {
	t.Parallel()

	w := caStockedWorld(t, mustScript(t, []ScriptCheck{healthNode(caSubject, 6)}, nil, nil), nil)
	scriptTicks(w, caTicks, nil)
	if got, want := w.ScriptRegister(caRegister), int32(5); got != want {
		t.Errorf("register %d is %d, want the subject's own HP, %d", caRegister, got, want)
	}
}

// TestTheHealthArmAnswersCurrentHealthAndNotMaximumHealth separates the two
// health fields, which the shared fixture cannot: caStockedWorld's subject
// carries HP == MaxHP == 5, so a build reading MaxHP here answers correctly by
// coincidence and every other test in this file agrees with it. The entity
// below is damaged, so the two fields differ and only one answer is right.
//
// The arm reads current health (`TRIG-CHECK-051`), and this is the test that
// says so: with MaxHP set to a value the register must NOT take, an
// implementation reading maximum health fails here and nowhere else in the
// package.
func TestTheHealthArmAnswersCurrentHealthAndNotMaximumHealth(t *testing.T) {
	t.Parallel()

	const current, maximum int32 = 3, 9
	w, err := NewStructuredWorld(1, caBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: caSubject, X: 1, Y: 1, HP: current, MaxHP: maximum, MapUnitID: caSubjectMapID}},
		mustScript(t, []ScriptCheck{healthNode(caSubject, 6)}, nil, nil),
		Relations{}, nil, nil, nil, GhostTemplate{}, nil)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	scriptTicks(w, caTicks, nil)
	if got := w.ScriptRegister(caRegister); got != current {
		t.Errorf("register %d is %d, want the subject's CURRENT health %d; its maximum is %d, and "+
			"an arm reading the maximum answers %d here",
			caRegister, got, current, maximum, maximum)
	}
}

// TestTheHealthArmWritesNothingOffTheGate is TRIG-CHECK-051's own word: off
// the literal the arm writes nothing, which is not the same as writing zero.
//
// A CHECK MAY NOT SHARE A REGISTER WITH ANOTHER CHECK (NewScript refuses two
// checks owning one register), so the sentinel here is planted by an INSTANT
// — ScriptInstantSetVariable, register[p0] = p1, the same arm
// TestTheRegisterFileIsOneArrayForBothIdSpaces uses to show a check CAN
// overwrite what an instant wrote. Here the health node owns the SAME
// register and the gate is off, so across a second pass the sentinel must
// survive untouched: an arm writing zero and an arm writing nothing produce
// the same answer only by coincidence of the sentinel chosen, so the sentinel
// is neither 0 nor the subject's own HP.
func TestTheHealthArmWritesNothingOffTheGate(t *testing.T) {
	t.Parallel()

	const sentinel int32 = 42
	s := mustScript(t,
		[]ScriptCheck{healthNode(caSubject, 999), constCheck(2, 1), constCheck(3, 1)},
		[]ScriptInstant{{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{caRegister, sentinel}}},
		[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(2, 3, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
	w := caStockedWorld(t, s, nil)

	scriptTicks(w, 7, nil)
	if got := w.ScriptRegister(caRegister); got != sentinel {
		t.Fatalf("after the first pass register %d is %d, want the sentinel %d the instant just planted",
			caRegister, got, sentinel)
	}
	scriptTicks(w, 16, nil)
	if got := w.ScriptRegister(caRegister); got != sentinel {
		t.Errorf("after the second pass register %d is %d, want the untouched sentinel %d — check 4 "+
			"must write nothing off its own gate, even on a register it owns", caRegister, got, sentinel)
	}
}

// TestATriggerReadingTheHealthArmFiresOnlyOnTheGatedValue is B1's second half.
func TestATriggerReadingTheHealthArmFiresOnlyOnTheGatedValue(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		gate int32
		want uint32
	}{
		{"on the gate, HP matches the constant", 6, 1},
		{"off the gate, the register is never written", 999, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t,
				[]ScriptCheck{healthNode(caSubject, tc.gate), constCheck(caConstant, 5)},
				[]ScriptInstant{{Op: ScriptInstantWin}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)},
					Instants: acts(0), Once: true}})
			if got := s.InertTriggers(); len(got) != 0 {
				t.Fatalf("InertTriggers() is %v, want none — check 4 is implemented", got)
			}
			w := caStockedWorld(t, s, nil)
			scriptTicks(w, caTicks, nil)
			won, lost := w.ScriptCounters()
			if won != tc.want || lost != 0 {
				t.Errorf("the counters are won %d lost %d, want won %d lost 0", won, lost, tc.want)
			}
		})
	}
}

// ------------------------------------------------------------ check 16

// itemDistNode is the compiled check-16 record under test, spelt once.
func itemDistNode(unit EntityID, code uint16, hasItem bool, x, y int32) ScriptCheck {
	n := ScriptCheck{Op: ScriptCheckItemDistance, Register: caRegister, Unit: unit, HasUnit: true,
		Args: [scriptParams]int32{x, y}}
	if hasItem {
		n.Item, n.HasItem = code, true
	}
	return n
}

func TestTheItemDistanceArmMeasuresFromTheUnitNotTheItem(t *testing.T) {
	t.Parallel()

	// caSubject sits at (1,1) in caStockedWorld. Chebyshev distance to (11,1)
	// is 10.
	w := caStockedWorld(t, mustScript(t, []ScriptCheck{itemDistNode(caSubject, caCure, true, 11, 1)}, nil, nil),
		[]Stock{{ID: caSubject, Items: []uint16{caCure}}})
	scriptTicks(w, caTicks, nil)
	if got, want := w.ScriptRegister(caRegister), int32(10); got != want {
		t.Errorf("register %d is %d, want the unit-to-point Chebyshev distance %d", caRegister, got, want)
	}
}

// TestTheItemDistanceArmAnswersTheSentinelWhenTheCodeIsNotFound is
// TRIG-CHECK-052's own 0xff not-found sentinel.
func TestTheItemDistanceArmAnswersTheSentinelWhenTheCodeIsNotFound(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		node  ScriptCheck
		stock []Stock
	}{
		{"the container holds a different code", itemDistNode(caSubject, caCure, true, 11, 1),
			[]Stock{{ID: caSubject, Items: []uint16{caOther}}}},
		{"the container is empty", itemDistNode(caSubject, caCure, true, 11, 1), nil},
		{"the node names no item", itemDistNode(caSubject, caCure, false, 11, 1),
			[]Stock{{ID: caSubject, Items: []uint16{caCure}}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := caStockedWorld(t, mustScript(t, []ScriptCheck{tc.node}, nil, nil), tc.stock)
			scriptTicks(w, caTicks, nil)
			if got := w.ScriptRegister(caRegister); got != 0xff {
				t.Errorf("register %d is %d, want the not-found sentinel 0xff", caRegister, got)
			}
		})
	}
}

// TestTheItemDistanceArmReadsItsOwnParametersEightBitsWide is the read width
// TRIG-CHECK-052 names at two instructions and TRIG-CHECK-054 classes as an
// engine width: the node's own X and Y are byte loads, so a parameter above
// 255 is measured as its low byte.
//
// The unit's position is NOT masked — it comes from the unit's own position
// object, not from the record — so the two operands of the same subtraction
// are read at different widths, and this test is the only place that shows it.
// No shipped node reaches the truncation: the widest authored X/Y in the whole
// corpus are 116 and 130 (`TRIG-CHECK-054`), both inside 8 bits on both roots.
func TestTheItemDistanceArmReadsItsOwnParametersEightBitsWide(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		x, y int32
		want int32
	}{
		// caSubject sits at (1,1). Each parameter is chosen so that masking
		// it and masking only the RESULT give different answers: 0x200
		// truncates to 0, for a Chebyshev distance of 1, where an unmasked
		// parameter gives 511 and scriptByte turns that into 255. A value
		// such as 0x10b would NOT discriminate — 266 masks back to 10, the
		// same answer the truncated parameter gives — so the fixture would
		// pass against an arm that masks nothing.
		{"an X above 255 measures as its low byte", 0x200, 1, 1},
		{"a Y above 255 measures as its low byte", 1, 0x100, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := caStockedWorld(t,
				mustScript(t, []ScriptCheck{itemDistNode(caSubject, caCure, true, tc.x, tc.y)}, nil, nil),
				[]Stock{{ID: caSubject, Items: []uint16{caCure}}})
			scriptTicks(w, caTicks, nil)
			if got := w.ScriptRegister(caRegister); got != tc.want {
				t.Errorf("register %d is %d, want %d — the arm's own X/Y are 8-bit reads",
					caRegister, got, tc.want)
			}
		})
	}
}

// TestATriggerReadingTheItemDistanceArmFiresOnlyAtTheMeasuredDistance is B2's
// second half.
func TestATriggerReadingTheItemDistanceArmFiresOnlyAtTheMeasuredDistance(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		stock []Stock
		want  uint32
	}{
		{"holding the code, at distance 10", []Stock{{ID: caSubject, Items: []uint16{caCure}}}, 1},
		{"not holding the code", nil, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := mustScript(t,
				[]ScriptCheck{itemDistNode(caSubject, caCure, true, 11, 1), constCheck(caConstant, 10)},
				[]ScriptInstant{{Op: ScriptInstantWin}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)},
					Instants: acts(0), Once: true}})
			if got := s.InertTriggers(); len(got) != 0 {
				t.Fatalf("InertTriggers() is %v, want none — check 16 is implemented", got)
			}
			w := caStockedWorld(t, s, tc.stock)
			scriptTicks(w, caTicks, nil)
			won, lost := w.ScriptCounters()
			if won != tc.want || lost != 0 {
				t.Errorf("the counters are won %d lost %d, want won %d lost 0", won, lost, tc.want)
			}
		})
	}
}

// -------------------------------------------------------- check 21 / instant 26

// caStructID is the one structure the fixtures below place, and
// caOtherStructID an id no fixture world holds — a resolved reference the
// world does not carry, distinct from a reference that never resolved at all.
const (
	caStructID      StructureID = 5
	caOtherStructID StructureID = 9
)

// structFieldNode is the compiled check-21 record under test.
func structFieldNode(id StructureID, has bool) ScriptCheck {
	return ScriptCheck{Op: ScriptCheckStructField, Register: caRegister, Structure: id, HasStructure: has}
}

// structFieldSet is the compiled instant-26 record under test.
func structFieldSet(id StructureID, has bool, v int32) ScriptInstant {
	return ScriptInstant{Op: ScriptInstantStructField, Structure: id, HasStructure: has,
		Args: [scriptParams]int32{v}}
}

// caStructuredWorld is a one-entity, one-structure world running s, on
// caStockedWorld's own terms.
func caStructuredWorld(t *testing.T, s *Script, structs []Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(1, caBounds, ModeCanonical, Terrain{},
		[]Entity{{ID: caSubject, X: 1, Y: 1, HP: 5, MaxHP: 5, MapUnitID: caSubjectMapID}},
		s, Relations{}, nil, nil, nil, GhostTemplate{}, structs)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	return w
}

// TestTheStructFieldArmAnswersField42SignExtended is B3's register half.
// 0xffff sign-extends to -1, the same MOVSX treatment the arm's own
// instruction applies and the one place in this story a raw uint16 needs it
// (Entity.HP is already signed).
func TestTheStructFieldArmAnswersField42SignExtended(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name  string
		field uint16
		want  int32
	}{
		{"a small positive value", 7, 7},
		{"the high bit set, sign-extends negative", 0xffff, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := caStructuredWorld(t, mustScript(t, []ScriptCheck{structFieldNode(caStructID, true)}, nil, nil),
				[]Structure{{ID: caStructID, Field42: tc.field}})
			scriptTicks(w, caTicks, nil)
			if got := w.ScriptRegister(caRegister); got != tc.want {
				t.Errorf("register %d is %d, want %d", caRegister, got, tc.want)
			}
		})
	}
}

func TestTheStructFieldArmMeasuresNothingForAnUnresolvedReference(t *testing.T) {
	t.Parallel()

	const sentinel int32 = 42
	for _, tc := range []struct {
		name string
		node ScriptCheck
	}{
		{"the reference never resolved at compile time", structFieldNode(caStructID, false)},
		{"the reference resolved to a structure this world does not hold",
			structFieldNode(caOtherStructID, true)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// The sentinel is planted by an instant, on
			// TestTheHealthArmWritesNothingOffTheGate's own ground: two checks
			// may not share a register, and tc.node already owns caRegister.
			s := mustScript(t,
				[]ScriptCheck{tc.node, constCheck(2, 1), constCheck(3, 1)},
				[]ScriptInstant{{Op: ScriptInstantSetVariable, Args: [scriptParams]int32{caRegister, sentinel}}},
				[]ScriptTrigger{{Pairs: [3]ScriptPair{pair(2, 3, ScriptCmpEQ)}, Instants: acts(0), Once: true}})
			w := caStructuredWorld(t, s, []Structure{{ID: caStructID, Field42: 7}})

			scriptTicks(w, 7, nil)
			if got := w.ScriptRegister(caRegister); got != sentinel {
				t.Fatalf("after the first pass register %d is %d, want the sentinel %d the instant "+
					"just planted", caRegister, got, sentinel)
			}
			scriptTicks(w, 16, nil)
			if got := w.ScriptRegister(caRegister); got != sentinel {
				t.Errorf("after the second pass register %d is %d, want the untouched sentinel %d",
					caRegister, got, sentinel)
			}
		})
	}
}

// TestTheStructFieldSetterWritesField42Unconditionally is instant 26's own
// half: the low 16 bits of the node's own first plain parameter replace
// Field42, and every OTHER structure in the world is untouched.
func TestTheStructFieldSetterWritesField42Unconditionally(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{structFieldSet(caStructID, true, 0x1234)}, 0)
	w := caStructuredWorld(t, s, []Structure{{ID: caStructID, Field42: 0}, {ID: caOtherStructID, Field42: 99}})
	runPass(t, w)

	got := w.Structures()
	if len(got) != 2 {
		t.Fatalf("Structures() has %d entries, want 2", len(got))
	}
	for _, st := range got {
		switch st.ID {
		case caStructID:
			if st.Field42 != 0x1234 {
				t.Errorf("structure %d Field42 is %#x, want %#x", st.ID, st.Field42, 0x1234)
			}
		case caOtherStructID:
			if st.Field42 != 99 {
				t.Errorf("structure %d Field42 is %d, want the untouched 99", st.ID, st.Field42)
			}
		default:
			t.Errorf("unexpected structure id %d", st.ID)
		}
	}
}

// TestTheStructFieldSetterWritesNothingForAnUnresolvedReference is instant
// 26's own mirror of the check's safety net.
func TestTheStructFieldSetterWritesNothingForAnUnresolvedReference(t *testing.T) {
	t.Parallel()

	s := fireOnce(t, []ScriptInstant{structFieldSet(caStructID, false, 0x1234)}, 0)
	w := caStructuredWorld(t, s, []Structure{{ID: caStructID, Field42: 7}})
	runPass(t, w)

	got := w.Structures()
	if len(got) != 1 || got[0].Field42 != 7 {
		t.Errorf("Structures() is %+v, want Field42 untouched at 7", got)
	}
}

// TestTheGetterSeesWhatTheSetterWrote is B3's own getter/setter pairing,
// witnessed within one script rather than assumed from the two arms' separate
// tests above: one trigger fires the setter on the first pass alone, unconditionally
// (registers 2/3 compare a constant to itself, fireOnce's own idiom); a second
// trigger reads check 21's register and can only fire once that register has
// been recomputed against the setter's write, which scriptPass does in check
// order BEFORE any trigger's instants run in that same pass — so the win must
// land on the second pass, never the first.
func TestTheGetterSeesWhatTheSetterWrote(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{structFieldNode(caStructID, true), constCheck(caConstant, 0x1234),
			constCheck(2, 1), constCheck(3, 1)},
		[]ScriptInstant{structFieldSet(caStructID, true, 0x1234), {Op: ScriptInstantWin}},
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{pair(2, 3, ScriptCmpEQ)}, Instants: acts(0), Once: true, Latch: 0},
			{Pairs: [3]ScriptPair{pair(caRegister, caConstant, ScriptCmpEQ)}, Instants: acts(1),
				Once: true, Latch: 1},
		})
	if got := s.InertTriggers(); len(got) != 0 {
		t.Fatalf("InertTriggers() is %v, want none — check 21 and instant 26 are both implemented", got)
	}
	w := caStructuredWorld(t, s, []Structure{{ID: caStructID, Field42: 0}})

	scriptTicks(w, 7, nil)
	if won, lost := w.ScriptCounters(); won != 0 || lost != 0 {
		t.Fatalf("after the first pass the counters are won %d lost %d, want 0 and 0 — the setter's "+
			"own write cannot reach the getter until the NEXT pass recomputes it", won, lost)
	}

	scriptTicks(w, 16, nil)
	won, lost := w.ScriptCounters()
	if won != 1 || lost != 0 {
		t.Errorf("after the second pass the counters are won %d lost %d, want won 1 lost 0 — the "+
			"setter's write must reach the getter by now", won, lost)
	}
}
