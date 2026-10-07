package sim

import (
	"bytes"
	"testing"
)

// The group word's own byte-form witnesses, and the one check that can still
// fail after the pin and its digest have been made to agree with each other.
//
// A pin and a digest computed from that pin agree by construction: if the
// transcription had gained a byte in the wrong place, both would move together
// and neither would say so. What cannot be faked is the pin read BACKWARDS —
// take this story's bytes, remove exactly the four the record grew by, put the
// version byte back, and require the result to be the digest the tree carried
// before the story. That literal predates the story and nothing here can move
// it.

// preGroupFormVersion and preGroupPinDigest are the version byte and the digest
// the pinned world had before the group word existed. They are pasted from the
// pre-story tree and are the fixed point everything below is measured against.
const (
	preGroupFormVersion byte   = 10
	preGroupPinDigest   uint64 = 0x9cfe358e8b5c1fae
	// The routed fixture's, from routeform_test.go's own pre-story pin.
	preGroupRoutedDigest uint64 = 0x6b56295a3f0a55f8
)

// groupWordAt is the group word's offset inside one entity record and ownerWordAt
// the owner's, both written out by hand from the contract rather than read off
// entityLen, so a record that widens again fails here instead of following the
// constant. recordLen is the record's current width, written out for the same
// reason.
//
// THE OWNER IS HERE BECAUSE A LATER STORY APPENDED IT, and peeling it is now part
// of reaching the pre-group form: the derivations below take this file's fixed
// point back two widenings rather than one, in reverse order, so each removal
// leaves the earlier ones where they are. The check they make is unchanged — the
// pre-group digest is still a literal that predates both stories and that nothing
// in this tree can move.
const (
	groupWordAt = 83
	ownerWordAt = 87
	recordLen   = 91
)

// TestThePinIsThePreStoryPinPlusTheGroupWord is SC-3's forward derivation: this
// story's hand-transcribed pin, minus the group word of every record and with
// the version byte put back, IS the pre-story form.
//
// If the story wrote a byte anywhere but in those three four-byte holes, this
// disagrees — and it is the only check in the package that would, because every
// other pin moved with the change.
func TestThePinIsThePreStoryPinPlusTheGroupWord(t *testing.T) {
	t.Parallel()

	const cells = 20
	const records = 3
	base := 34 + cells

	stripped := strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(pinBytesPreGroups, 20, 3), 20, 3)), cells, records), cells)
	// Backwards, so each removal leaves the earlier records where they are, and
	// the owner before the group, so the group's offset is still the contract's.
	for i := records - 1; i >= 0; i-- {
		o := base + recordLen*i
		stripped = append(stripped[:o+ownerWordAt], stripped[o+recordLen:]...)
		stripped = append(stripped[:o+groupWordAt], stripped[o+ownerWordAt:]...)
	}
	stripped[0] = preGroupFormVersion

	if want := 34 + cells + records*83 + records*4 + scriptStateLen + scriptCountsLen; len(stripped) != want {
		t.Fatalf("the stripped form is %d byte(s), want the pre-story %d", len(stripped), want)
	}
	if got := fnv1a(stripped); got != preGroupPinDigest {
		t.Errorf("the pin with its group words removed hashes %#016x, want the pre-story %#016x — "+
			"this story wrote a byte outside the three it was supposed to add",
			got, preGroupPinDigest)
	}
}

// TestTheRoutedPinIsThePreStoryPinPlusTheGroupWord is the same derivation over
// the second pinned fixture, which carries stored ROUTE CELLS after its records.
// A group word written four bytes wide but in the wrong place would leave the
// first pin's arithmetic intact and this one's broken, because here the records
// are not the last thing in the form.
func TestTheRoutedPinIsThePreStoryPinPlusTheGroupWord(t *testing.T) {
	t.Parallel()

	const cells = 16
	const records = 3
	base := 34 + cells

	stripped := strippedOfThePlanes(strippedOfTheFacing(strippedOfTheRelation(strippedOfTheDecayBlock(strippedOfTheSight(rtfBytesPreGroups, 16, 3), 16, 3)), cells, records), cells)
	for i := records - 1; i >= 0; i-- {
		o := base + recordLen*i
		stripped = append(stripped[:o+ownerWordAt], stripped[o+recordLen:]...)
		stripped = append(stripped[:o+groupWordAt], stripped[o+ownerWordAt:]...)
	}
	stripped[0] = preGroupFormVersion

	if got := fnv1a(stripped); got != preGroupRoutedDigest {
		t.Errorf("the routed pin with its group words removed hashes %#016x, want the pre-story "+
			"%#016x — this story wrote a byte outside the three it was supposed to add",
			got, preGroupRoutedDigest)
	}
}

// TestTwoWorldsDifferingOnlyInAGroupHashDifferently is SC-3's differential and
// the whole of R-2's mitigation.
//
// Every pinned digest in this package moved with this story, and a digest that
// changed for the right reason looks exactly like one that changed for the
// wrong one. This is the assertion that does not care what the digest IS: two
// worlds alike in every other field hash differently iff the group word reaches
// the form at all. A field encoded nowhere would make these two equal.
func TestTwoWorldsDifferingOnlyInAGroupHashDifferently(t *testing.T) {
	t.Parallel()

	build := func(g uint32) *World {
		return mustWorld(t, 7, Bounds{Width: 4, Height: 4}, []Entity{
			{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5, Group: g},
			{ID: 2, X: 2, Y: 2, HP: 5, MaxHP: 5, Group: 9},
		})
	}
	a, b := build(3), build(4)
	if a.Hash() == b.Hash() {
		t.Errorf("two worlds differing only in one entity's group hash alike (%#016x) — "+
			"the field does not reach the byte form", a.Hash())
	}
	if bytes.Equal(mustMarshal(t, a), mustMarshal(t, b)) {
		t.Errorf("two worlds differing only in one entity's group marshal to the same bytes")
	}
	// And the same world twice is still the same world, so the difference above
	// is the group and not the encoder being unstable.
	if build(3).Hash() != a.Hash() {
		t.Errorf("one world built twice hashes differently")
	}
}

// TestADeadUnitKeepsItsGroupThroughTheForm is the decoder's missing refusal,
// asserted as behaviour rather than trusted as an omission.
//
// Four fields of a felled unit ARE refused on the way in — a target, a crossing,
// a group rate term and an attack order — and each is residue of a state the
// tick that left it already spent. Membership is not one of them, and if a fifth
// clause were ever added by pattern-matching the other four, the count of a
// group whose last member has died could no longer be taken.
func TestADeadUnitKeepsItsGroupThroughTheForm(t *testing.T) {
	t.Parallel()

	w := mustWorld(t, 1, Bounds{Width: 4, Height: 4}, []Entity{
		{ID: 1, X: 1, Y: 1, HP: 0, MaxHP: 10, Group: 6},
		{ID: 2, X: 2, Y: 2, HP: -40, MaxHP: 10, Group: 6},
	})
	for _, e := range w.Entities() {
		if e.Alive() {
			t.Fatalf("entity %d is alive; this fixture is about units that are not", e.ID)
		}
		if e.Group != 6 {
			t.Errorf("entity %d carries group %d before the round trip, want 6", e.ID, e.Group)
		}
	}
	var back World
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatalf("a form carrying two dead members of one group was refused: %v", err)
	}
	for _, e := range back.Entities() {
		if e.Group != 6 {
			t.Errorf("entity %d comes back carrying group %d, want 6", e.ID, e.Group)
		}
	}
}

// TestTheCheckRecordCarriesItsGroupAndItsPresence is AC-5 over the script
// section's own widened record: the group a check names, and the byte that says
// whether it names one, both cross the form — and group zero NAMED is not the
// same record as no group at all.
func TestTheCheckRecordCarriesItsGroupAndItsPresence(t *testing.T) {
	t.Parallel()

	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0},
			{Op: 1, Register: 1, Group: 4294967295, HasGroup: true},
			{Op: 1, Register: 2, Group: 0, HasGroup: true},
			{Op: 1, Register: 3},
		}, nil, nil)
	w := scriptWorld(t, s, []Entity{{ID: 1, Group: 0}})

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
	// The two that differ ONLY in the presence flag must differ in the bytes, or
	// group zero and no group are one record.
	if got[2].HasGroup == got[3].HasGroup {
		t.Errorf("a check naming group 0 and a check naming none come back alike")
	}
	if back.Hash() != w.Hash() {
		t.Errorf("the round trip hashes %#016x, want %#016x", back.Hash(), w.Hash())
	}
}

func mustMarshal(t *testing.T, w *World) []byte {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	return b
}

// TestTheSackArmIsStillLoudBesideTheGroupArm is AC-6 and SC-6. Opcode 14 — the
// sack at a cell — landed in this story (0103); the function keeps this name
// only because docs/0070-group-population/verification.md and
// docs/0071-unit-owner/verification.md cite it by name, not because the arm
// under test is still the sack arm — the sentinel moved to opcode 8 there, then
// to opcode 9, and is now scriptCheckSentinelOp (script_test.go), which 1029
// made the suite's one declaration of "a check this build does not evaluate".
//
// The group arm arriving must not quiet the arms that have not. The sentinel
// arm is deliberately unimplemented, so the only thing it could answer is a
// constant, and a constant answer is exactly what the inert rule exists to
// stop. So it is still reported, its readers are still inert, and a trigger
// reading only group checks beside it is live and fires.
//
// The authored operand is the zero the unimplemented arm's register would sit
// at, so a build that let it through would win this mission on the first pass.
func TestTheSackArmIsStillLoudBesideTheGroupArm(t *testing.T) {
	t.Parallel()

	const notAnArm = scriptCheckSentinelOp
	if scriptCheckSupported(notAnArm) {
		t.Fatalf("opcode %d is implemented; this test needs a check this build still does not evaluate", notAnArm)
	}
	if !scriptCheckSupported(ScriptCheckGroupCount) {
		t.Fatal("the group arm is not implemented")
	}
	s := mustScript(t,
		[]ScriptCheck{
			{Op: ScriptCheckConstant, Register: 0},
			{Op: notAnArm, Register: 1, Args: [scriptParams]int32{38, 64}},
			{Op: ScriptCheckGroupCount, Register: 2, Group: 3, HasGroup: true},
		},
		[]ScriptInstant{{Op: ScriptInstantWin}},
		[]ScriptTrigger{
			{Pairs: [3]ScriptPair{{Left: 1, Right: 0, Cmp: ScriptCmpEQ, Used: true}},
				Instants: [4]int32{0, ScriptNone, ScriptNone, ScriptNone},
				Once:     true, Latch: 0},
			{Pairs: [3]ScriptPair{{Left: 2, Right: 0, Cmp: ScriptCmpGT, Used: true}},
				Instants: [4]int32{ScriptNone, ScriptNone, ScriptNone, ScriptNone},
				Once:     true, Latch: 1}})

	if gaps := s.Unsupported(); len(gaps) != 1 || gaps[0].Op != notAnArm {
		t.Errorf("the report names %+v, want only the sack arm — opcode 1 is implemented now", gaps)
	}
	if inert := s.InertTriggers(); len(inert) != 1 || inert[0] != 0 {
		t.Errorf("triggers %v are inert, want only the one reading the sack arm", inert)
	}

	w := scriptWorld(t, s, []Entity{{ID: 1, HP: 4, MaxHP: 4, Group: 3}})
	scriptTicks(w, 64, nil)
	if w.Outcome() != OutcomeUndecided {
		t.Errorf("the mission came to %d; an unimplemented arm must not decide one", w.Outcome())
	}
	if w.ScriptLatched(0) {
		t.Errorf("an inert trigger touched its latch")
	}
	// And the group check beside it measured, and its own trigger ran.
	if got := w.ScriptRegister(2); got != 1 {
		t.Errorf("the group check measured %d, want 1", got)
	}
	if !w.ScriptLatched(1) {
		t.Errorf("the trigger reading only the group check did not fire")
	}
}
