package sim

// The STRUCTURE SECTION's own byte form (1033 B3): round trip through the
// public MarshalBinary/UnmarshalBinary pair, and decodeStructures's own
// refusals, exercised directly since nothing else in this package calls it
// with a malformed buffer.

import (
	"bytes"
	"testing"
)

func TestStructureBlockingMaskSurvivesNativeFormAndTicks(t *testing.T) {
	st := Structure{ID: 7, Col: 10, Row: 11, Width: 3, Height: 2, Attach: 0b111111, Blocking: 0b100001}
	w := sbWorld(t, []Structure{st})
	first, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if first[0] != structureBlockingFormVersion {
		t.Fatalf("unequal masks lost optional form: version %d", first[0])
	}
	var cold World
	if err := cold.UnmarshalBinary(first); err != nil {
		t.Fatal(err)
	}
	if got := cold.Structures(); len(got) != 1 || got[0] != st {
		t.Fatalf("cold mask: %+v want %+v", got, st)
	}
	constructed := sbWorld(t, []Structure{st})
	constructed.ghost = GhostTemplate{Class: 7, Speed: 256}
	if err := constructed.UnmarshalBinary(first); err != nil {
		t.Fatal(err)
	}
	if constructed.ghost.Class != 7 || constructed.ghost.Speed != 256 {
		t.Fatal("loaded mask form lost install-derived constructor data")
	}
	for range 32 {
		Step(w, nil)
		Step(&cold, nil)
	}
	second, err := cold.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	written, err := w.MarshalBinary()
	if err != nil || !bytes.Equal(second, written) {
		t.Fatalf("post-tick native forms diverged: %v", err)
	}
	old := sbWorld(t, []Structure{{ID: 7, Col: 10, Row: 11, Width: 1, Height: 1, Attach: 1, Blocking: 1}})
	legacy, err := old.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if legacy[0] != formatVersion {
		t.Fatalf("equal masks changed the base form: version %d", legacy[0])
	}
	var restored World
	if err := restored.UnmarshalBinary(legacy); err != nil || restored.Structures()[0].Blocking != 1 {
		t.Fatalf("legacy mask default: %+v, %v", restored.Structures(), err)
	}
	fresh := restored.Structures()
	fresh[0].Blocking = 0
	if !restored.RestoreStructureBlocking(fresh) || restored.Structures()[0].Blocking != 0 {
		t.Fatalf("matching mission mask did not repair old native form: %+v", restored.Structures())
	}
}

// sbWorld is a world holding structs, on caStructuredWorld's own terms but
// self-contained so this file does not depend on scriptcheckarms1033_test.go.
func sbWorld(t *testing.T, structs []Structure) *World {
	t.Helper()
	w, err := NewStructuredWorld(1, Bounds{Width: 40, Height: 40}, ModeCanonical, Terrain{},
		[]Entity{{ID: 1, X: 1, Y: 1, HP: 5, MaxHP: 5}}, nil, Relations{}, nil, nil, nil,
		GhostTemplate{}, structs)
	if err != nil {
		t.Fatalf("NewStructuredWorld: %v", err)
	}
	return w
}

// TestTheStructureSectionRoundTripsThroughTheByteForm is B3's persistence
// half: several structures, distinct nonzero Field42 values, exact fidelity
// after MarshalBinary/UnmarshalBinary.
func TestTheStructureSectionRoundTripsThroughTheByteForm(t *testing.T) {
	t.Parallel()

	want := []Structure{
		{ID: 3, Field42: 0x1234, MaxHealth: 0x2000, Col: -7, Row: 12, Width: 1, Height: 1, Attach: 1},
		{ID: 7, Field42: 0xffff, MaxHealth: 30000, Col: 22, Row: -3, Width: 2, Height: 3, Attach: 0xa5a5a5a5},
		{ID: 40, Field42: 0},
	}
	w := sbWorld(t, want)

	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var got World
	if err := got.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}

	gs := got.Structures()
	if len(gs) != len(want) {
		t.Fatalf("Structures() has %d entries after round trip, want %d", len(gs), len(want))
	}
	for i, w := range want {
		if gs[i] != w {
			t.Errorf("structure %d is %+v after round trip, want %+v", i, gs[i], w)
		}
	}
}

func TestDecodeStructuresRefusesHalfAFootprint(t *testing.T) {
	buf := make([]byte, structureCountLen+structureRecordLen)
	buf[0] = 1
	buf[structureCountLen+16] = 1
	if _, _, err := decodeStructures(buf); err == nil {
		t.Fatal("decodeStructures accepted width 1 and height 0")
	}
}

// TestAnEmptyStructureListRoundTripsToNil is the zero-structure case every
// world built before this story has: NewStructuredWorld's own ladder
// predecessors all name none.
func TestAnEmptyStructureListRoundTripsToNil(t *testing.T) {
	t.Parallel()

	w := sbWorld(t, nil)
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	var got World
	if err := got.UnmarshalBinary(data); err != nil {
		t.Fatalf("UnmarshalBinary: %v", err)
	}
	if gs := got.Structures(); len(gs) != 0 {
		t.Errorf("Structures() is %+v after round trip, want none", gs)
	}
}

// TestStructureSectionLenMatchesWhatEncodeWrites is structureSectionLen's own
// word, checked against encodeStructures's actual output rather than assumed
// from the constants.
func TestStructureSectionLenMatchesWhatEncodeWrites(t *testing.T) {
	t.Parallel()

	w := sbWorld(t, []Structure{{ID: 1, Field42: 1}, {ID: 2, Field42: 2}, {ID: 3, Field42: 3}})
	buf := make([]byte, w.structureSectionLen()+8)
	for i := range buf {
		buf[i] = 0xcc
	}
	end := w.encodeStructures(buf, 4)
	if got, want := end-4, w.structureSectionLen(); got != want {
		t.Errorf("encodeStructures wrote %d byte(s), structureSectionLen() says %d", got, want)
	}
	if buf[0] != 0xcc || buf[1] != 0xcc || buf[2] != 0xcc || buf[3] != 0xcc {
		t.Errorf("encodeStructures wrote before its own offset: %v", buf[:4])
	}
	for i := end; i < len(buf); i++ {
		if buf[i] != 0xcc {
			t.Errorf("encodeStructures wrote past the offset it returned, at byte %d", i)
			break
		}
	}
}

// TestDecodeStructuresRefusesATruncatedCount is the count field's own bound:
// fewer than structureCountLen bytes.
func TestDecodeStructuresRefusesATruncatedCount(t *testing.T) {
	t.Parallel()

	if _, _, err := decodeStructures([]byte{1, 2, 3}); err == nil {
		t.Error("decodeStructures accepted a 3-byte buffer, want a refusal — the count alone needs 4")
	}
}

// TestDecodeStructuresRefusesADeclaredCountTheBufferCannotHold is the
// declared-count bound checked BEFORE any record is allocated (structure.go's
// own doc comment, decodeGroups' and decodeCasting's precedent).
func TestDecodeStructuresRefusesADeclaredCountTheBufferCannotHold(t *testing.T) {
	t.Parallel()

	buf := make([]byte, structureCountLen)
	// Declares 100 records; the buffer holds bytes for none.
	buf[0], buf[1], buf[2], buf[3] = 100, 0, 0, 0
	if _, _, err := decodeStructures(buf); err == nil {
		t.Error("decodeStructures accepted a declared count of 100 over an empty record buffer, want a refusal")
	}
}

// TestDecodeStructuresRefusesNonAscendingOrDuplicateIDs is the entity
// section's own rule, applied here: the order is canonical, so a form
// carrying either encodes differently from the world it decodes to.
func TestDecodeStructuresRefusesNonAscendingOrDuplicateIDs(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		ids  []uint32
	}{
		{"descending", []uint32{5, 3}},
		{"duplicate", []uint32{5, 5}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			buf := make([]byte, structureCountLen+structureRecordLen*len(tc.ids))
			buf[0] = byte(len(tc.ids))
			for i, id := range tc.ids {
				o := structureCountLen + structureRecordLen*i
				buf[o] = byte(id)
				// Field42 left zero; only the id ordering is under test.
			}
			if _, _, err := decodeStructures(buf); err == nil {
				t.Errorf("decodeStructures accepted %s ids %v, want a refusal", tc.name, tc.ids)
			}
		})
	}
}
