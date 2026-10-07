package mapedit_test

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"testing"

	"againrom/pkg/mapedit"
)

// fuzzMaxSteps bounds a script in operations, so a fuzzer-supplied megabyte is a
// long test rather than an unbounded one. Every step re-opens and re-writes the
// whole image, which is what costs.
const fuzzMaxSteps = 256

// errNothingHappened is what an Undo or a Redo with nothing to do reports, so
// those two sit in the same table as the mutations and "accepted" means one
// thing for all of them: the call did something.
var errNothingHappened = errors.New("the call reported that nothing happened")

// fuzzOp is one opcode: the name the coverage test counts by, the number of
// operand bytes it consumes, and the call. arity is fixed per opcode, so a
// script that runs short pads its last operands with zeros instead of shifting
// every later opcode by one byte.
type fuzzOp struct {
	name  string
	arity int
	apply func(ed *mapedit.Editor, a []byte) error
}

// fuzzOps is one entry per mutation this model exposes, plus Undo and Redo.
// Every entry is a kind the coverage test requires the seeds to have had
// accepted at least once.
var fuzzOps = []fuzzOp{
	{"SetTile", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetTile(fuzzCoord(a[0], fixW), fuzzCoord(a[1], fixH), binary.LittleEndian.Uint16(a[2:]))
	}},
	{"SetAltitude", 3, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetAltitude(fuzzCoord(a[0], fixW), fuzzCoord(a[1], fixH), a[2])
	}},
	{"SetOverlay", 3, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetOverlay(fuzzCoord(a[0], fixW), fuzzCoord(a[1], fixH), a[2])
	}},
	{"SetAngle", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetAngle(math.Float32frombits(binary.LittleEndian.Uint32(a)))
	}},
	{"SetWord0C", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetWord0C(binary.LittleEndian.Uint32(a))
	}},
	{"SetWord10", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetWord10(binary.LittleEndian.Uint32(a))
	}},
	{"SetWord14", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetWord14(binary.LittleEndian.Uint32(a))
	}},
	{"SetWord70", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetWord70(binary.LittleEndian.Uint32(a))
	}},
	{"SetWord74", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetWord74(binary.LittleEndian.Uint32(a))
	}},
	{"SetBitmask", 4, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetBitmask(binary.LittleEndian.Uint32(a))
	}},
	{"SetName", 2, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetName(fuzzName(a[0], a[1]))
	}},
	{"SetDescription", 2, func(ed *mapedit.Editor, a []byte) error {
		return ed.SetDescription(fuzzDescription(a[0], a[1]))
	}},
	{"PlaceUnit", 1, func(ed *mapedit.Editor, a []byte) error {
		_, err := ed.PlaceUnit(fuzzRecord(a[0]))
		return err
	}},
	{"MoveUnit", 9, func(ed *mapedit.Editor, a []byte) error {
		return ed.MoveUnit(fuzzIndex(ed, a[0]),
			binary.LittleEndian.Uint32(a[1:]), binary.LittleEndian.Uint32(a[5:]))
	}},
	{"DeleteUnit", 1, func(ed *mapedit.Editor, a []byte) error {
		return ed.DeleteUnit(fuzzIndex(ed, a[0]))
	}},
	{"Undo", 0, func(ed *mapedit.Editor, a []byte) error {
		if ed.Undo() {
			return nil
		}
		return errNothingHappened
	}},
	{"Redo", 0, func(ed *mapedit.Editor, a []byte) error {
		if ed.Redo() {
			return nil
		}
		return errNothingHappened
	}},
}

// ---------------------------------------------------------------------------
// Operands — one script byte to one argument
//
// Each mapping deliberately reaches past the accepted range on both sides. A
// driver that could only produce arguments the model accepts would never reach
// the atomicity path, and every rejection the script provokes is a step whose
// re-open check still has to pass.
// ---------------------------------------------------------------------------

// fuzzCoord maps a script byte onto -1..n for a dimension of n cells: -1 and n
// lie outside the grid and everything between lies inside it.
func fuzzCoord(b byte, n int) int { return int(b)%(n+2) - 1 }

// fuzzIndex maps a script byte onto -1..UnitCount(), so it produces every valid
// file-order index and one invalid one on each side. The count is read from the
// model because the script cannot know it: a place or a delete earlier in the
// same script has already changed it.
func fuzzIndex(ed *mapedit.Editor, b byte) int { return int(b)%(ed.UnitCount()+2) - 1 }

// fuzzName builds an ASCII name of 0..67 letters — the field holds 63 and a
// terminator, so the long ones are rejected — and, when the seed's high bit is
// set, appends a rune outside ASCII, which is the other rejection this field has.
func fuzzName(n, seed byte) string {
	out := make([]byte, 0, 68)
	for i := 0; i < int(n)%68; i++ {
		out = append(out, byte('A'+(int(seed)+i)%26))
	}
	if seed&0x80 != 0 {
		return string(out) + "\u00c4"
	}
	return string(out)
}

// fuzzDescription is fuzzName's Windows-1251 counterpart: 0..67 Cyrillic capitals,
// each one byte in that code page, plus — when the seed's high bit is set — U+2603,
// which the code page has no byte for at all.
func fuzzDescription(n, seed byte) string {
	out := make([]rune, 0, 68)
	for i := 0; i < int(n)%68; i++ {
		out = append(out, rune(0x0410+(int(seed)+i)%32))
	}
	if seed&0x80 != 0 {
		out = append(out, '\u2603')
	}
	return string(out)
}

// fuzzRecord builds the 70 bytes of a placed unit as a per-seed ramp in a buffer
// this driver allocates for the call and hands over. Nothing about the content
// matters to the model, which copies the record verbatim, and nothing about it
// matters to the reader, which takes six raw fields out of it and validates none
// of them — so the ramp's only job is to be a buffer the driver owns.
func fuzzRecord(seed byte) []byte {
	rec := make([]byte, unitRecordSize)
	for i := range rec {
		rec[i] = byte(int(seed) + 3*i)
	}
	return rec
}

// ---------------------------------------------------------------------------
// The driver
// ---------------------------------------------------------------------------

// runScript applies one script and holds AC-7 over it, returning the variant it
// ran on and how many times each opcode was accepted. It reports the counts
// rather than judging them: under-coverage is not a defect of an input, and the
// target must not fail for it.
func runScript(t *testing.T, script []byte) (string, map[string]int) {
	t.Helper()

	variants := fixtureVariants()
	v := variants[0]
	body := script
	if len(script) > 0 {
		v = variants[int(script[0])%len(variants)]
		body = script[1:]
	}

	// The model is handed a copy, so pristine — the image every claim below is
	// judged against — is a buffer the model never saw.
	pristine := richFixture(v.order, v.nUnits)
	ed := newEditor(t, clone(pristine))

	accepted := make(map[string]int, len(fuzzOps))
	steps, cursor := 0, 0
	for cursor < len(body) && steps < fuzzMaxSteps {
		op := fuzzOps[int(body[cursor])%len(fuzzOps)]
		cursor++
		args := make([]byte, op.arity)
		cursor += copy(args, body[cursor:])

		if err := op.apply(ed, args); err == nil {
			accepted[op.name]++
		}
		reopensIdentically(t, fmt.Sprintf("%s: after step %d (%s)", v.name, steps, op.name), ed.Bytes())
		steps++
	}

	for undone := 0; ed.Undo(); undone++ {
		if undone > fuzzMaxSteps {
			t.Fatalf("%s: Undo still reported work after %d reverts of a %d-step script", v.name, undone, steps)
		}
	}
	if ed.CanUndo() {
		t.Errorf("%s: CanUndo is true after Undo reported nothing left to undo", v.name)
	}
	if got := ed.Bytes(); !bytes.Equal(got, pristine) {
		t.Errorf("%s: undoing a %d-step script to the empty history did not restore the input (%d bytes vs %d)",
			v.name, steps, len(got), len(pristine))
	}
	return v.name, accepted
}

// ---------------------------------------------------------------------------
// The seeds
// ---------------------------------------------------------------------------

// seedOp is one operation of a seed script, named rather than numbered: the
// opcode byte is looked up, so a seed cannot drift out of step with the table
// by an insertion into it.
type seedOp struct {
	name string
	args []byte
}

func sop(name string, args ...byte) seedOp { return seedOp{name: name, args: args} }

// seedScript encodes a variant selector and a list of operations as the byte
// string the driver reads back. Each operation's operands are padded or trimmed
// to that opcode's own arity, so a miscounted seed cannot shift the operations
// after it.
func seedScript(tb testing.TB, variant int, ops ...seedOp) []byte {
	tb.Helper()
	out := []byte{byte(variant)}
	for _, o := range ops {
		i := -1
		for j, op := range fuzzOps {
			if op.name == o.name {
				i = j
				break
			}
		}
		if i < 0 {
			tb.Fatalf("seed names the operation %q, which the opcode table does not have", o.name)
		}
		args := make([]byte, fuzzOps[i].arity)
		copy(args, o.args)
		out = append(out, byte(i))
		out = append(out, args...)
	}
	return out
}

// everyOperation is a script that has every opcode in the table accepted once,
// in an order that makes each one possible: a unit is placed before one is moved
// or deleted, something is applied before it is undone, and something is undone
// before it is redone. The four variants each get one, so the coverage claim
// holds per fixture rather than only across the corpus — including on the
// variant whose type-6 payload starts out empty.
func everyOperation(tb testing.TB, variant int) []byte {
	return seedScript(tb, variant,
		sop("PlaceUnit", 0x10),
		sop("SetTile", 2, 2, 0xcd, 0xab),
		sop("SetAltitude", 2, 2, 0x5a),
		sop("SetOverlay", 1, 1, 0x6b),
		sop("SetAngle", 0x00, 0x00, 0xc0, 0x3f),
		sop("SetWord0C", 0x0c, 0x00, 0xd0, 0x1f),
		sop("SetWord10", 0x10, 0x00, 0xd0, 0x1f),
		sop("SetWord14", 0x14, 0x00, 0xd0, 0x1f),
		sop("SetWord70", 0x70, 0x00, 0xd0, 0x1f),
		sop("SetWord74", 0x74, 0x00, 0xd0, 0x1f),
		sop("SetBitmask", 0x18, 0x1f, 0x00, 0x00),
		sop("SetName", 5, 0x03),
		sop("SetDescription", 4, 0x02),
		sop("MoveUnit", 1, 0x00, 0xee, 0x44, 0x00, 0x00, 0xff, 0x55, 0x00),
		sop("DeleteUnit", 1),
		sop("Undo"),
		sop("Redo"),
	)
}

// fuzzSeeds is the corpus, declared here in code: this package keeps nothing
// under testdata/, so the seeds a plain `go test` runs are the seeds this file
// says they are.
func fuzzSeeds(tb testing.TB) [][]byte {
	tb.Helper()
	seeds := [][]byte{
		everyOperation(tb, 0),
		everyOperation(tb, 1),
		everyOperation(tb, 2),
		everyOperation(tb, 3),

		// Undo and redo across a length change, then a mutation that discards
		// the redo history and a redo that must find nothing.
		seedScript(tb, 3,
			sop("PlaceUnit", 0x20),
			sop("PlaceUnit", 0x30),
			sop("SetTile", 1, 1, 0x11, 0x22),
			sop("Undo"), sop("Undo"), sop("Undo"), sop("Undo"),
			sop("Redo"), sop("Redo"),
			sop("SetOverlay", 2, 1, 0x77),
			sop("Redo"),
		),

		// A length change undone and then replaced. The mutation after the undo
		// discards an entry that had removed bytes from the image, so a log the
		// applier failed to truncate is re-reverted on the way back to the empty
		// history and takes 70 bytes with it. Nothing shorter reaches this: a
		// re-reverted fixed-length part writes the same bytes twice and is
		// indistinguishable from writing them once.
		seedScript(tb, 2,
			sop("PlaceUnit", 0x40),
			sop("Undo"),
			sop("SetTile", 2, 1, 0x33, 0x44),
		),
		seedScript(tb, 1,
			sop("DeleteUnit", 1),
			sop("Undo"),
			sop("SetAltitude", 1, 2, 0x55),
		),

		// Deleting past the end of a populated map: the last delete has no index
		// left to take.
		seedScript(tb, 1,
			sop("DeleteUnit", 1), sop("DeleteUnit", 1), sop("DeleteUnit", 1), sop("DeleteUnit", 1),
		),

		// Every rejection surface in one script, on a map with no units at all.
		seedScript(tb, 0,
			sop("SetTile", 0, 0, 0x01, 0x02), // x = -1
			sop("SetAltitude", 4, 3, 0x03),   // x = W, y = H
			sop("MoveUnit", 0),               // no units to move
			sop("DeleteUnit", 0),
			sop("SetName", 200, 0x01),         // over 63 encoded bytes
			sop("SetDescription", 0x02, 0xff), // a rune the code page has no byte for
			sop("Undo"),                       // nothing was ever accepted
			sop("Redo"),
		),

		// Raw byte strings with no operation structure at all: the opcode and
		// operand mapping has to be total, and these are what a fuzzer's own
		// mutations of the corpus look like.
		{},
		{0x02},
		{0x01, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
			0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15},
		bytes.Repeat([]byte{0xff}, 48),
		bytes.Repeat([]byte{0x00}, 48),
	}
	return seeds
}

// ---------------------------------------------------------------------------
// SC-8
// ---------------------------------------------------------------------------

// FuzzEditScript is AC-7. Under plain `go test` it runs the declared seeds; under
// -fuzz it runs whatever the fuzzer builds out of them, and the round-trip claim
// is the same either way. It never fails for under-coverage: the counts runScript
// returns are discarded here on purpose, because an input that exercises two
// opcodes has found nothing wrong.
func FuzzEditScript(f *testing.F) {
	for _, s := range fuzzSeeds(f) {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, script []byte) {
		runScript(t, script)
	})
}

// TestFuzzSeedsAcceptEveryMutationKind is the other half of SC-8, and the reason
// the coverage assertion is out here: a driver whose operand mapping had drifted
// into rejecting every call would satisfy the target on every input it ever saw,
// since a rejected mutation leaves an image that re-opens perfectly. This test is
// what makes that go red — and it counts kinds by name, so an opcode that stopped
// being reachable is a named failure rather than a coverage percentage.
func TestFuzzSeedsAcceptEveryMutationKind(t *testing.T) {
	total := make(map[string]int, len(fuzzOps))
	perVariant := make(map[string]map[string]int)

	for i, s := range fuzzSeeds(t) {
		t.Run(fmt.Sprintf("seed %d", i), func(t *testing.T) {
			variant, accepted := runScript(t, s)
			if perVariant[variant] == nil {
				perVariant[variant] = make(map[string]int, len(fuzzOps))
			}
			for name, n := range accepted {
				total[name] += n
				perVariant[variant][name] += n
			}
		})
	}

	for _, op := range fuzzOps {
		if total[op.name] == 0 {
			t.Errorf("no seed had %s accepted even once: the driver cannot reach that mutation", op.name)
		}
	}

	if len(perVariant) != len(fixtureVariants()) {
		t.Errorf("the seeds ran %d of the %d fixture variants", len(perVariant), len(fixtureVariants()))
	}
	for variant, counts := range perVariant {
		for _, op := range fuzzOps {
			if counts[op.name] == 0 {
				t.Errorf("on %s, no seed had %s accepted even once", variant, op.name)
			}
		}
	}
}
