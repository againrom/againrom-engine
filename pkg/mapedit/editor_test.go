package mapedit_test

// The model's core: what New accepts, what a fresh load exposes, whose buffers
// are whose, and the history.
//
// Every undo and redo case compares against a buffer this file cloned before
// the step it is judging, never against a value recomputed from the model or
// from the fixture. That is deliberate: a history implemented as whole-buffer
// snapshots would satisfy a recomputed expectation by construction, so the case
// would stop telling a correct applier from a wrong one — which is the only
// thing it exists to do.

import (
	"bytes"
	"math"
	"reflect"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The formatVersion whose record-header-skipping variant alm rejects.
const formatVersion1000 = 1000

func newEditor(t *testing.T, data []byte) *mapedit.Editor {
	t.Helper()
	ed, err := mapedit.New(data)
	if err != nil {
		t.Fatalf("mapedit.New rejected the fixture: %v", err)
	}
	if ed == nil {
		t.Fatal("mapedit.New returned a nil model with a nil error")
	}
	return ed
}

// sameView reports whether two views say the same thing.
//
// It is not reflect.DeepEqual of the two maps, and the reason is the fixture's
// signalling NaN: the angle and the per-map constants are raw bit patterns the
// reader stores without interpreting, and NaN never equals itself as a float.
// Comparing them structurally would make every view comparison in this package
// false for a map whose angle is exactly the value the fixture chose to prove
// survives. So the float fields are compared as bits and blanked, and the rest
// — every slice, string and word — is compared structurally.
func sameView(a, b *alm.Map) bool {
	if a == nil || b == nil {
		return a == b
	}
	if math.Float32bits(a.Angle) != math.Float32bits(b.Angle) ||
		a.MetaRecordWord != b.MetaRecordWord {
		return false
	}
	if len(a.Records) != len(b.Records) {
		return false
	}
	for i := range a.Records {
		x, y := a.Records[i], b.Records[i]
		if x.TypeID != y.TypeID || x.PayloadSize != y.PayloadSize ||
			x.Word10 != y.Word10 {
			return false
		}
	}

	x, y := *a, *b
	x.Angle, y.Angle = 0, 0
	x.MetaRecordWord, y.MetaRecordWord = 0, 0
	x.Records, y.Records = nil, nil
	return reflect.DeepEqual(x, y)
}

func TestNewAcceptsExactlyWhatOpenDocumentAccepts(t *testing.T) {
	good := richFixture(orderType0First, fixUnitsMax)
	spoil := func(f func(b []byte)) []byte {
		b := clone(good)
		f(b)
		return b
	}

	cases := []struct {
		name string
		data []byte
		// thin marks a stream alm accepts whose roster is not the ten records
		// this model was originally written against.
		thin bool
	}{
		{name: "the fixture itself", data: clone(good)},
		{name: "nil", data: nil},
		{name: "one byte short of the file header", data: clone(good)[:fileHeaderSize-1]},
		{name: "a bad magic", data: spoil(func(b []byte) { putU32(b, fhMagic, 0xdeadbeef) })},
		{name: "a file hdrLen that is not 20", data: spoil(func(b []byte) { putU32(b, fhHdrLen, 24) })},
		{name: "the header-skipping formatVersion", data: spoil(func(b []byte) { putU32(b, fhFormatVersion, formatVersion1000) })},
		{name: "a payload that overruns EOF", data: spoil(func(b []byte) {
			putU32(b, walkFrame(t, b).payloadSizeWord(t, 7).off, 1<<20)
		})},
		{name: "a trailing byte the records do not cover", data: append(clone(good), 0x00)},
		{name: "a W the grids do not match", data: spoil(func(b []byte) {
			putU32(b, walkFrame(t, b).payload(t, 0, metaW, 4).off, fixW+1)
		})},
		{name: "a #type6 the type-6 payload does not match", data: spoil(func(b []byte) {
			putU32(b, walkFrame(t, b).payload(t, 0, metaCount6, 4).off, fixUnitsMax+1)
		})},

		// This fixture's physical order opens 0, 1, 2, so a count of three
		// leaves a well-formed three-record map with the rest of the file
		// trailing it, and a count of nine drops the tenth record the same way.
		{name: "a recordCount of three", data: spoil(func(b []byte) { putU32(b, fhRecordCount, 3) }), thin: true},
		{name: "a recordCount of nine", data: spoil(func(b []byte) { putU32(b, fhRecordCount, 9) }), thin: true},
	}
	for _, r := range thinRosters() {
		cases = append(cases, struct {
			name string
			data []byte
			thin bool
		}{name: r.name, data: r.image(), thin: true})
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, docErr := alm.OpenDocument(c.data)
			ed, err := mapedit.New(c.data)

			switch {
			case err == nil && docErr != nil:
				t.Fatalf("New accepted a stream alm.OpenDocument rejected (%v): this model must never be looser than the acceptance it delegates to", docErr)
			case err != nil && docErr == nil && c.thin:
				t.Fatalf("New refused a roster alm accepts: %v", err)
			case (err != nil) != (docErr != nil):
				t.Fatalf("New error = %v but alm.OpenDocument error = %v: the two accept sets differ", err, docErr)
			}
			if c.thin && docErr != nil {
				t.Fatalf("alm.OpenDocument rejected %v, so this case no longer witnesses the roster it names", docErr)
			}

			if err != nil && ed != nil {
				t.Error("New returned a model beside its error")
			}
			if err == nil && ed == nil {
				t.Error("New returned a nil model with a nil error")
			}
		})
	}
}

// TestAThinRosterLoadsAndTakesTheEditsItCanTake is AC-8's load half. A map that
// does not carry every record is an ordinary map here: it loads unchanged, its
// view is the reader's own, and the setters that address a record it does have
// work and leave the result re-openable. The setters that address one it does
// not are the other half, witnessed beside the layer and the roster they belong
// to.
func TestAThinRosterLoadsAndTakesTheEditsItCanTake(t *testing.T) {
	for _, r := range thinRosters() {
		t.Run(r.name, func(t *testing.T) {
			data := r.image()
			ed := newEditor(t, data)

			if got := ed.Bytes(); !bytes.Equal(got, data) {
				t.Errorf("the fresh model's bytes differ from the input (%d vs %d bytes)", len(got), len(data))
			}
			want, err := alm.Open(data)
			if err != nil {
				t.Fatalf("alm.Open of the thin fixture: %v", err)
			}
			got, err := ed.Map()
			if err != nil {
				t.Fatalf("the fresh model's view failed: %v", err)
			}
			if !sameView(got, want) {
				t.Error("the fresh model's view differs from alm.Open of the input")
			}

			// Every record the roster DOES carry is still reachable, and the
			// two optional ones are checked here rather than only where their
			// absence is: a model that lost a record it has fails no
			// absence case, because the absence cases are about the other
			// rosters. The twelve-record roster is the one this separates —
			// its type-6 sits past the tenth record.
			wantUnits := 0
			if r.units {
				wantUnits = fixUnitsMax
			}
			if n := ed.UnitCount(); n != wantUnits {
				t.Errorf("UnitCount = %d, want %d for this roster", n, wantUnits)
			}
			if r.units {
				rec, err := ed.UnitRecord(0)
				if err != nil {
					t.Fatalf("UnitRecord(0) on a roster that carries type-6: %v", err)
				}
				if !bytes.Equal(rec, fixUnitRecord(0)) {
					t.Error("the first record read back is not the fixture's own")
				}
			}
			if r.overlay {
				if err := ed.SetOverlay(0, 0, gridNewOverlay); err != nil {
					t.Fatalf("SetOverlay on a roster that carries type-3: %v", err)
				}
				if !ed.Undo() {
					t.Fatal("Undo of that overlay edit reported nothing to undo")
				}
			}

			// type-0, type-1 and type-2 are in every accepted stream, so these
			// three are available on every roster above.
			before := ed.Bytes()
			if err := ed.SetTile(2, 1, gridNewTile); err != nil {
				t.Fatalf("SetTile on a thin roster: %v", err)
			}
			if err := ed.SetAltitude(0, 0, gridNewAltitude); err != nil {
				t.Fatalf("SetAltitude on a thin roster: %v", err)
			}
			if err := ed.SetWord0C(0x5e5e5e5e); err != nil {
				t.Fatalf("SetWord0C on a thin roster: %v", err)
			}
			after := ed.Bytes()
			if bytes.Equal(after, before) {
				t.Fatal("three accepted edits changed no byte, so this case witnesses nothing")
			}
			if len(after) != len(before) {
				t.Fatalf("three fixed-length edits changed the image size: %d -> %d", len(before), len(after))
			}
			reopensIdentically(t, "after editing a thin roster", after)

			for i := 0; i < 3; i++ {
				if !ed.Undo() {
					t.Fatalf("Undo %d reported nothing to undo", i+1)
				}
			}
			if out := ed.Bytes(); !bytes.Equal(out, data) {
				t.Error("undoing every edit did not restore the thin fixture exactly")
			}
		})
	}
}

// TestFreshModelExposesTheInputAndItsView is AC-1: before any mutation the
// bytes are the input and the view is the input's own view.
func TestFreshModelExposesTheInputAndItsView(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			data := richFixture(v.order, v.nUnits)
			ed := newEditor(t, data)

			if got := ed.Bytes(); !bytes.Equal(got, data) {
				t.Errorf("the fresh model's bytes differ from the input (%d bytes vs %d)", len(got), len(data))
			}

			want, err := alm.Open(data)
			if err != nil {
				t.Fatalf("alm.Open of the fixture: %v", err)
			}
			got, err := ed.Map()
			if err != nil {
				t.Fatalf("the fresh model's view failed: %v", err)
			}
			if !sameView(got, want) {
				t.Error("the fresh model's view differs from alm.Open of the input")
			}

			if ed.CanUndo() || ed.CanRedo() {
				t.Errorf("a fresh model reports CanUndo=%v CanRedo=%v, want both false", ed.CanUndo(), ed.CanRedo())
			}
		})
	}
}

// TestTheModelSharesNoBufferWithItsCallerOrWithItself is SC-1's independence
// half: nothing the model hands out is the model, and nothing the caller keeps
// is either.
func TestTheModelSharesNoBufferWithItsCallerOrWithItself(t *testing.T) {
	data := richFixture(orderType0First, fixUnitsMax)
	input := clone(data)
	ed := newEditor(t, input)

	// The caller's slice, mutated behind the model's back after the load.
	for i := range input {
		input[i] ^= 0xff
	}
	if got := ed.Bytes(); !bytes.Equal(got, data) {
		t.Error("mutating the caller's slice after New changed the model's bytes")
	}

	// Two calls, two buffers, and neither is the model's.
	b1, b2 := ed.Bytes(), ed.Bytes()
	if &b1[0] == &b2[0] {
		t.Error("two Bytes calls returned the same backing array")
	}
	for i := range b1 {
		b1[i] ^= 0xff
	}
	if !bytes.Equal(b2, data) {
		t.Error("mutating one returned buffer changed another")
	}
	if got := ed.Bytes(); !bytes.Equal(got, data) {
		t.Error("mutating a returned buffer changed the model's bytes")
	}

	// A buffer taken before an edit still holds the bytes as they were.
	before := ed.Bytes()
	if err := ed.SetAltitude(0, 0, gridNewAltitude); err != nil {
		t.Fatalf("SetAltitude: %v", err)
	}
	if !bytes.Equal(before, data) {
		t.Error("an edit reached into a buffer handed out before it")
	}

	// The view is a fresh value too: two calls are two maps, and disturbing
	// one disturbs neither the model nor the next one.
	edited := ed.Bytes()
	m1, err := ed.Map()
	if err != nil {
		t.Fatalf("the view failed after the edit: %v", err)
	}
	m2, err := ed.Map()
	if err != nil {
		t.Fatalf("the second view failed: %v", err)
	}
	if m1 == m2 {
		t.Error("two Map calls returned the same map")
	}
	m1.Tiles[0] ^= 0xffff
	m1.Altitudes[0] ^= 0xff
	m1.Meta.Slots[0] ^= 0xff
	if got := ed.Bytes(); !bytes.Equal(got, edited) {
		t.Error("mutating a returned view changed the model's bytes")
	}
	m3, err := ed.Map()
	if err != nil {
		t.Fatalf("the third view failed: %v", err)
	}
	if !sameView(m3, m2) {
		t.Error("mutating a returned view changed a later view")
	}
}

// rejectionChangesNothing drives one call that must be rejected and requires
// the bytes, the view and both history reports to be exactly what they were.
// It takes the model in whatever history state the caller has put it in, since
// "unchanged" has to mean unchanged from that state and not merely empty.
func rejectionChangesNothing(t *testing.T, ed *mapedit.Editor, call func() error) {
	t.Helper()

	before := ed.Bytes()
	viewBefore, errBefore := ed.Map()
	canUndo, canRedo := ed.CanUndo(), ed.CanRedo()

	if err := call(); err == nil {
		t.Fatal("the call was accepted, so this case witnesses no rejection at all")
	}

	if got := ed.Bytes(); !bytes.Equal(got, before) {
		t.Error("a rejected mutation changed the bytes")
	}
	viewAfter, errAfter := ed.Map()
	if (errBefore == nil) != (errAfter == nil) {
		t.Errorf("a rejected mutation changed whether the view opens: %v -> %v", errBefore, errAfter)
	}
	if !sameView(viewAfter, viewBefore) {
		t.Error("a rejected mutation changed the view")
	}
	if ed.CanUndo() != canUndo || ed.CanRedo() != canRedo {
		t.Errorf("a rejected mutation changed the history: CanUndo %v -> %v, CanRedo %v -> %v",
			canUndo, ed.CanUndo(), canRedo, ed.CanRedo())
	}
}

// TestUndoAndRedoOnAnEmptyHistoryAreNoOps is AC-6, run twice over so a call
// that reported nothing cannot have moved the cursor anyway.
func TestUndoAndRedoOnAnEmptyHistoryAreNoOps(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			data := richFixture(order, fixUnitsMax)
			ed := newEditor(t, data)
			pristine := ed.Bytes()

			for i := 0; i < 2; i++ {
				if ed.Undo() {
					t.Error("Undo on an empty history reported that something happened")
				}
				if ed.Redo() {
					t.Error("Redo with nothing undone reported that something happened")
				}
			}
			if ed.CanUndo() || ed.CanRedo() {
				t.Errorf("after the no-ops: CanUndo=%v CanRedo=%v, want both false", ed.CanUndo(), ed.CanRedo())
			}
			if got := ed.Bytes(); !bytes.Equal(got, pristine) {
				t.Error("a no-op Undo or Redo changed the bytes")
			}
			if _, err := ed.Map(); err != nil {
				t.Errorf("the view failed after the no-ops: %v", err)
			}
		})
	}
}

// TestUndoRestoresAndRedoReappliesTheExactBytes is AC-5's first sequence: M,
// Undo, Redo, against the two buffers the test cloned itself.
func TestUndoRestoresAndRedoReappliesTheExactBytes(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(order, fixUnitsMax))

			before := ed.Bytes()
			if err := ed.SetTile(2, 1, gridNewTile); err != nil {
				t.Fatalf("SetTile: %v", err)
			}
			after := ed.Bytes()
			if bytes.Equal(after, before) {
				t.Fatal("the mutation changed no byte, so this case witnesses nothing")
			}
			if !ed.CanUndo() || ed.CanRedo() {
				t.Errorf("after the mutation: CanUndo=%v CanRedo=%v, want true and false", ed.CanUndo(), ed.CanRedo())
			}

			if !ed.Undo() {
				t.Fatal("Undo after a mutation reported nothing to undo")
			}
			if got := ed.Bytes(); !bytes.Equal(got, before) {
				t.Error("Undo did not restore the pre-mutation bytes exactly")
			}
			if ed.CanUndo() || !ed.CanRedo() {
				t.Errorf("after the Undo: CanUndo=%v CanRedo=%v, want false and true", ed.CanUndo(), ed.CanRedo())
			}

			if !ed.Redo() {
				t.Fatal("Redo after an Undo reported nothing to redo")
			}
			if got := ed.Bytes(); !bytes.Equal(got, after) {
				t.Error("Redo did not reproduce the post-mutation bytes exactly")
			}
			if !ed.CanUndo() || ed.CanRedo() {
				t.Errorf("after the Redo: CanUndo=%v CanRedo=%v, want true and false", ed.CanUndo(), ed.CanRedo())
			}
		})
	}
}

// TestTwoMutationsUndoTwiceRedoTwice is AC-5's second sequence. The two
// mutations are on different layers and different cells, so a history that
// reverted the wrong entry — or the right one twice — lands on bytes no
// snapshot in this test matches.
func TestTwoMutationsUndoTwiceRedoTwice(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(order, fixUnitsMax))

			s0 := ed.Bytes()
			if err := ed.SetTile(0, 0, gridNewTile); err != nil {
				t.Fatalf("SetTile: %v", err)
			}
			s1 := ed.Bytes()
			if err := ed.SetOverlay(2, 1, gridNewOverlay); err != nil {
				t.Fatalf("SetOverlay: %v", err)
			}
			s2 := ed.Bytes()
			if bytes.Equal(s0, s1) || bytes.Equal(s1, s2) {
				t.Fatal("the two mutations did not produce three distinct images")
			}

			for _, step := range []struct {
				what string
				do   func() bool
				want []byte
			}{
				{"Undo", ed.Undo, s1},
				{"Undo, Undo", ed.Undo, s0},
				{"Redo", ed.Redo, s1},
				{"Redo, Redo", ed.Redo, s2},
			} {
				if !step.do() {
					t.Fatalf("%s reported nothing to do", step.what)
				}
				if got := ed.Bytes(); !bytes.Equal(got, step.want) {
					t.Errorf("after %s the bytes are not the image the test cloned for that point", step.what)
				}
			}
			if !ed.CanUndo() || ed.CanRedo() {
				t.Errorf("at the end: CanUndo=%v CanRedo=%v, want true and false", ed.CanUndo(), ed.CanRedo())
			}
		})
	}
}

// TestAMutationAfterAnUndoDiscardsTheRedoHistory is AC-5's third sequence. The
// discarded entry is then re-checked from the other side: the new mutation's
// own Undo must return to the image before it, not to the image the discarded
// entry would have produced.
func TestAMutationAfterAnUndoDiscardsTheRedoHistory(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(order, fixUnitsMax))

			s0 := ed.Bytes()
			if err := ed.SetTile(1, 0, gridNewTile); err != nil {
				t.Fatalf("SetTile: %v", err)
			}
			s1 := ed.Bytes()
			if !ed.Undo() {
				t.Fatal("Undo reported nothing to undo")
			}
			if got := ed.Bytes(); !bytes.Equal(got, s0) {
				t.Fatal("Undo did not restore the pre-mutation bytes")
			}

			if err := ed.SetAltitude(2, 1, gridNewAltitude); err != nil {
				t.Fatalf("SetAltitude: %v", err)
			}
			s2 := ed.Bytes()
			if bytes.Equal(s2, s1) {
				t.Fatal("the second mutation produced the first one's image, so the discard is unobservable")
			}

			if ed.CanRedo() {
				t.Error("CanRedo is true after a mutation that should have discarded the redo history")
			}
			if ed.Redo() {
				t.Error("Redo re-applied a mutation that a later mutation had discarded")
			}
			if got := ed.Bytes(); !bytes.Equal(got, s2) {
				t.Error("a no-op Redo changed the bytes")
			}

			if !ed.Undo() {
				t.Fatal("Undo after the replacing mutation reported nothing to undo")
			}
			if got := ed.Bytes(); !bytes.Equal(got, s0) {
				t.Error("undoing the replacing mutation did not restore the image before it")
			}
			if ed.CanUndo() {
				t.Error("CanUndo is true after the only surviving entry was undone")
			}
		})
	}
}

func TestAMutationThatChangesNoByteIsStillRecorded(t *testing.T) {
	ed := newEditor(t, richFixture(orderType6First, fixUnitsMax))

	m, err := ed.Map()
	if err != nil {
		t.Fatalf("the fresh view failed: %v", err)
	}
	same := m.Tiles[1*fixW+2]

	before := ed.Bytes()
	if err := ed.SetTile(2, 1, same); err != nil {
		t.Fatalf("SetTile to the value already there: %v", err)
	}
	if got := ed.Bytes(); !bytes.Equal(got, before) {
		t.Error("setting a cell to the value it already held changed a byte")
	}
	if !ed.CanUndo() {
		t.Fatal("a mutation whose new value equalled the old was not recorded")
	}
	if !ed.Undo() {
		t.Fatal("Undo reported nothing to undo after that mutation")
	}
	if got := ed.Bytes(); !bytes.Equal(got, before) {
		t.Error("undoing it changed a byte")
	}
	if !ed.CanRedo() || !ed.Redo() {
		t.Error("the undone mutation is not redoable")
	}
	if got := ed.Bytes(); !bytes.Equal(got, before) {
		t.Error("redoing it changed a byte")
	}
}
