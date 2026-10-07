package sim

// The compiled check's two player references (AC-6): what the byte form
// carries for Player, Player2 and their two presence bytes, and that a check
// naming player 0 and one naming no player cross the form as two different
// records rather than one — the same separation every other reference in
// this section already keeps.
//
// The compiled instant's second unit reference (0129 T2, AC-7) joins it
// below: Unit2 and HasUnit2 at the instant record's own tail, on the same
// terms and the same separation, one version higher.

import "testing"

// TestTheCheckRecordCarriesBothPlayerReferences is AC-6 over the check
// record's newest tail: both players a check names, and the two bytes that
// say whether it names either, all cross the form — the pattern
// TestTheCheckRecordCarriesItsGroupAndItsPresence (groupform_test.go) already
// runs for the group beside them, now run for the pair T2 appends.
func TestTheCheckRecordCarriesBothPlayerReferences(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0},
			{Op: 1, Register: 1, Player: 4294967295, HasPlayer: true, Player2: 1, HasPlayer2: true},
			{Op: 1, Register: 2, Player: 3, HasPlayer: true},
			{Op: 1, Register: 3, Player2: 7, HasPlayer2: true},
			{Op: 1, Register: 4},
			{Op: 1, Register: 5, Player: 0, HasPlayer: true, Player2: 0, HasPlayer2: true},
		}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got, want := back.Script().Checks(), s.Checks()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("check %d crosses the form as %+v, want %+v", i, got[i], want[i])
		}
	}
	// The record naming player 0 on both sides and the one naming neither
	// must differ in the bytes, or player zero and no player are one record.
	if got[4].HasPlayer == got[5].HasPlayer || got[4].HasPlayer2 == got[5].HasPlayer2 {
		t.Errorf("a check naming player 0 and a check naming none come back alike: %+v vs %+v",
			got[4], got[5])
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
}

// TestAVersion26FormNamingAPlayerIsRefused is AC-6's other half over this
// story's own field, on TestAVersion22FormIsRefused's pattern (reach_test.go):
// a valid CURRENT form, spoiled at its version byte alone to the version
// this story's player references replaced, is refused — which isolates the
// version check from the form's every other byte and is the reason it lives
// here rather than being folded into TestThePreviousVersionFormIsRefused
// (binary_test.go), whose own two cases are about the version BYTE and not
// about a script that actually carries what the earlier version could not.
func TestAVersion26FormNamingAPlayerIsRefused(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: 1, Register: 0, Player: 5, HasPlayer: true, Player2: 9, HasPlayer2: true},
	}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	spoiled := withByte(form, 0, preScriptPlayerFormVersion)

	var back World
	if err := back.UnmarshalBinary(spoiled); err == nil {
		t.Fatal("a form naming a player at the version before this story existed was accepted")
	}
}

// ---------------------------------------------------------------------------
// The compiled instant's second unit reference (0129 T2, AC-7): the byte
// form carries Unit2 and HasUnit2 the way the check record's own pair
// already does, appended at the instant record's own tail, version 38.
// ---------------------------------------------------------------------------

// TestTheInstantRecordCarriesItsSecondUnitReference is AC-7's round trip: a
// world whose script carries an instant with a second reference encodes and
// decodes to an equal world, and the form's own first byte declares
// formatVersion — TestTheCheckRecordCarriesBothPlayerReferences' own pattern
// above, run for the instant record's newest tail instead of the check
// record's.
func TestTheInstantRecordCarriesItsSecondUnitReference(t *testing.T) {
	t.Parallel()

	s := mustScript(t, nil, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 4294967295, HasUnit2: true},
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true},
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 0, HasUnit2: true},
	}, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	if form[0] != formatVersion {
		t.Fatalf("form[0] = %d, want formatVersion %d", form[0], formatVersion)
	}

	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got, want := back.Script().Instants(), s.Instants()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("instant %d crosses the form as %+v, want %+v", i, got[i], want[i])
		}
	}
	// The instant naming second reference 0 and the one naming none must
	// differ in the bytes, or second-reference zero and no second reference
	// are one record.
	if got[1].HasUnit2 == got[2].HasUnit2 {
		t.Errorf("an instant naming second reference 0 and one naming none come back alike: %+v vs %+v",
			got[1], got[2])
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
}

// TestTwoWorldsDifferingOnlyInAnInstantsSecondReferenceHashDifferently is
// AC-7's discriminating half: Hash derives from MarshalBinary alone
// (hash.go), so a field the encoder left out could round-trip inside the
// struct and still leave two otherwise-identical worlds indistinguishable on
// disk. Two scripts alike but for one instant's Unit2 must not collide.
func TestTwoWorldsDifferingOnlyInAnInstantsSecondReferenceHashDifferently(t *testing.T) {
	t.Parallel()

	s1 := mustScript(t, nil, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true},
	}, nil)
	s2 := mustScript(t, nil, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 3, HasUnit2: true},
	}, nil)
	w1 := scriptWorld(t, s1, []Entity{{ID: 1}})
	w2 := scriptWorld(t, s2, []Entity{{ID: 1}})

	if w1.Hash() == w2.Hash() {
		t.Errorf("worlds differing only in an instant's second reference both hash %#016x", w1.Hash())
	}
}

func TestAVersion37FormIsRefused(t *testing.T) {
	t.Parallel()

	s := mustScript(t, nil, []ScriptInstant{
		{Op: ScriptInstantGiveAll, Unit: 1, HasUnit: true, Unit2: 2, HasUnit2: true},
	}, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	spoiled := withByte(form, 0, 37)

	var back World
	if err := back.UnmarshalBinary(spoiled); err == nil {
		t.Fatal("a form declaring version 37 was accepted")
	}
}

// TestTheCheckRecordCarriesItsItemReference is 1029 B1 over the check record's
// newest tail: the packed item code a check names and the byte that says
// whether it names one, both across the form, on
// TestTheCheckRecordCarriesBothPlayerReferences's own pattern above.
//
// The four checks span the presence space the same way: a code named, another
// code named, no item named at all, and — the sharpest pair — item code 0
// NAMED against no item, which is the case that would collapse into one record
// if the flag were folded into a reserved code instead of carried beside it.
// The arm answers 0 for a node naming no item, so the two are not equivalent at
// run time either.
func TestTheCheckRecordCarriesItsItemReference(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0},
			{Op: ScriptCheckItemTest, Register: 1, Unit: 1, HasUnit: true, Item: 0x0e1e, HasItem: true},
			{Op: ScriptCheckItemTest, Register: 2, Unit: 1, HasUnit: true, Item: 0xffff, HasItem: true},
			{Op: ScriptCheckItemTest, Register: 3, Unit: 1, HasUnit: true},
			{Op: ScriptCheckItemTest, Register: 4, Unit: 1, HasUnit: true, Item: 0, HasItem: true},
		}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	var back World
	if err := back.UnmarshalBinary(form); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	got, want := back.Script().Checks(), s.Checks()
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("check %d crosses the form as %+v, want %+v", i, got[i], want[i])
		}
	}
	if got[3].HasItem == got[4].HasItem {
		t.Errorf("a check naming item code 0 and a check naming none come back alike: %+v vs %+v",
			got[3], got[4])
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
}

// TestTwoWorldsDifferingOnlyInAChecksItemHashDifferently is B1's discriminating
// half, on TestTwoWorldsDifferingOnlyInAnInstantsSecondReferenceHashDifferently's
// own ground: Hash derives from MarshalBinary alone, so a field the encoder left
// out would round-trip inside the struct and still leave two otherwise-identical
// worlds indistinguishable on disk.
func TestTwoWorldsDifferingOnlyInAChecksItemHashDifferently(t *testing.T) {
	t.Parallel()

	s1 := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckItemTest, Register: 0, Unit: 1, HasUnit: true, Item: 0x0e1e, HasItem: true},
	}, nil, nil)
	s2 := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckItemTest, Register: 0, Unit: 1, HasUnit: true, Item: 0x0e1a, HasItem: true},
	}, nil, nil)
	w1 := scriptWorld(t, s1, []Entity{{ID: 1}})
	w2 := scriptWorld(t, s2, []Entity{{ID: 1}})

	if w1.Hash() == w2.Hash() {
		t.Errorf("worlds differing only in a check's item code both hash %#016x", w1.Hash())
	}
	s3 := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckItemTest, Register: 0, Unit: 1, HasUnit: true, HasItem: false},
	}, nil, nil)
	s4 := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckItemTest, Register: 0, Unit: 1, HasUnit: true, HasItem: true},
	}, nil, nil)
	w3 := scriptWorld(t, s3, []Entity{{ID: 1}})
	w4 := scriptWorld(t, s4, []Entity{{ID: 1}})
	if w3.Hash() == w4.Hash() {
		t.Errorf("worlds differing only in a check's item PRESENCE both hash %#016x", w3.Hash())
	}
}

// TestAVersion56FormCarryingAChecksItemIsRefused is B1's version half, on
// TestAVersion26FormNamingAPlayerIsRefused's own pattern: a valid current form,
// spoiled at its version byte alone to the version this story's item reference
// replaced, is refused. A version-56 check record is three bytes shorter, so a
// reader that accepted the byte would walk every record after the first one at
// the wrong offset.
func TestAVersion56FormCarryingAChecksItemIsRefused(t *testing.T) {
	t.Parallel()

	s := mustScript(t, []ScriptCheck{
		{Op: ScriptCheckItemTest, Register: 0, Unit: 1, HasUnit: true, Item: 0x0e1e, HasItem: true},
	}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1}})

	form := mustMarshal(t, w)
	spoiled := withByte(form, 0, 56)

	var back World
	if err := back.UnmarshalBinary(spoiled); err == nil {
		t.Fatal("a form declaring version 56 was accepted")
	}
}
