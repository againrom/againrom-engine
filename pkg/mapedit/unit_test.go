package mapedit_test

// The unit setters: the count, the byte-exact read, and the three mutations —
// place, move, delete. Two of those change the image's length, which makes this
// the first file whose edits have more than one part and the first whose parts
// move relative to each other. Every region diff is taken through
// fixture_test.go's comparator, and every image an edit is judged against is
// either one this file cloned before the step or one it assembled itself from the
// container contract.

import (
	"bytes"
	"fmt"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// The values these cases write. The placed record's index is one the fixture
// itself never emits, so its 70 bytes differ from every record already in
// the map.
const (
	unitMoveX = 0x0044ee00
	unitMoveY = 0x0055ff00

	unitPlacedIndex = 7
)

// unitCoordImage is the eight bytes a move must leave at a record's front: x then
// y, each little-endian, written out here rather than encoded the way the setter
// does.
func unitCoordImage(x, y uint32) []byte {
	img := make([]byte, 0, 8)
	for _, v := range []uint32{x, y} {
		img = append(img, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	}
	return img
}

func reopensIdentically(t *testing.T, what string, data []byte) {
	t.Helper()
	doc, err := alm.OpenDocument(data)
	if err != nil {
		t.Fatalf("%s: the edited bytes are not an acceptable map: %v", what, err)
	}
	if got := doc.Write(); !bytes.Equal(got, data) {
		t.Errorf("%s: re-opening the edited bytes and writing them back did not reproduce them (%d vs %d bytes)",
			what, len(got), len(data))
	}
	if _, err := alm.Open(data); err != nil {
		t.Errorf("%s: the view of the edited bytes failed: %v", what, err)
	}
}

// unitLengthChangeRegions is the declared target region of a place or a delete,
// in each image: the added or removed bytes, the type-6 payloadSize word and
// #type6 at type-0 +0x24. Each image is walked for its own, because a
// length-changing edit relocates every byte above the insertion — under the
// type-6-first order that includes the count word the edit itself rewrites.
func unitLengthChangeRegions(t *testing.T, before, after []byte, at, beforeN, afterN int) (bRegions, aRegions []region) {
	t.Helper()
	fb, fa := walkFrame(t, before), walkFrame(t, after)
	recordAt := fb.byType(t, 6).payloadOff + at*unitRecordSize
	bRegions = []region{
		{recordAt, beforeN},
		fb.payloadSizeWord(t, 6),
		fb.payload(t, 0, metaCount6, 4),
	}
	aRegions = []region{
		{recordAt, afterN},
		fa.payloadSizeWord(t, 6),
		fa.payload(t, 0, metaCount6, 4),
	}
	return bRegions, aRegions
}

// unitWords reads the two count words a length-changing edit must keep in step:
// #type6 and the type-6 record's own payloadSize.
func unitWords(t *testing.T, data []byte) (count, payloadSize uint32) {
	t.Helper()
	fs := walkFrame(t, data)
	return u32At(data, fs.payload(t, 0, metaCount6, 4).off), u32At(data, fs.payloadSizeWord(t, 6).off)
}

// ---------------------------------------------------------------------------
// The count and the byte-exact read
// ---------------------------------------------------------------------------

func TestUnitCountAndRecordReadTheFixtureExactly(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(v.order, v.nUnits))

			if got := ed.UnitCount(); got != v.nUnits {
				t.Fatalf("UnitCount = %d, want %d", got, v.nUnits)
			}
			for i := 0; i < v.nUnits; i++ {
				want := fixUnitRecord(i)
				got, err := ed.UnitRecord(i)
				if err != nil {
					t.Fatalf("UnitRecord(%d): %v", i, err)
				}
				if !bytes.Equal(got, want) {
					t.Errorf("UnitRecord(%d) =\n % x\nwant\n % x", i, got, want)
				}
			}

			if v.nUnits == 0 {
				return
			}

			// The record handed out is the caller's alone, in both
			// directions: disturbing it changes neither the model nor the
			// next read, and two reads are two buffers.
			first, err := ed.UnitRecord(0)
			if err != nil {
				t.Fatalf("UnitRecord(0): %v", err)
			}
			second, err := ed.UnitRecord(0)
			if err != nil {
				t.Fatalf("UnitRecord(0): %v", err)
			}
			if &first[0] == &second[0] {
				t.Error("two UnitRecord calls returned the same backing array")
			}
			image := ed.Bytes()
			for i := range first {
				first[i] ^= 0xff
			}
			if !bytes.Equal(second, fixUnitRecord(0)) {
				t.Error("mutating one returned record changed another")
			}
			if got := ed.Bytes(); !bytes.Equal(got, image) {
				t.Error("mutating a returned record changed the model's bytes")
			}
			third, err := ed.UnitRecord(0)
			if err != nil {
				t.Fatalf("UnitRecord(0): %v", err)
			}
			if !bytes.Equal(third, fixUnitRecord(0)) {
				t.Error("mutating a returned record changed a later read")
			}
		})
	}
}

// TestAMapWithNoUnitsRecordHasNoUnitsAndTakesNoUnitEdit is AC-8's unit half —
// the branch declared unreachable when every accepted stream carried all ten
// records.
//
// An absent type-6 record is not an empty type-6 payload, and the contrast is
// asserted here rather than described: on a map whose type-6 record is present
// and empty a place succeeds and is the first unit; on a map with no type-6
// record every unit call is refused, the last of them because adding the record
// is out of this story's scope and not because the roster is full.
//
// The count is the discriminating part. These fixtures carry a #type6 of
// fixUnitsMax with no records behind it — the shape a shipped four-record map
// really has — so a model that read the word and trusted it would report units
// it could not produce, and the first read of one would resolve an offset
// against a record that is not there.
func TestAMapWithNoUnitsRecordHasNoUnitsAndTakesNoUnitEdit(t *testing.T) {
	for _, r := range thinRosters() {
		if r.units {
			continue
		}
		t.Run(r.name, func(t *testing.T) {
			data := r.image()
			ed := newEditor(t, data)

			m, err := ed.Map()
			if err != nil {
				t.Fatalf("the view failed: %v", err)
			}
			if m.Present(6) {
				t.Fatal("the fixture carries a type-6 record, so this case witnesses no absence")
			}
			if m.Meta.Count6 == 0 {
				t.Fatal("#type6 is zero, so a zero unit count here says nothing about the record")
			}
			if len(m.Units) != 0 {
				t.Fatalf("the view has %d units without a type-6 record", len(m.Units))
			}
			if got := ed.UnitCount(); got != 0 {
				t.Errorf("UnitCount = %d with no type-6 record and a #type6 of %d, want 0", got, m.Meta.Count6)
			}

			for _, c := range []struct {
				name string
				call func() error
			}{
				{"UnitRecord(0)", func() error {
					rec, err := ed.UnitRecord(0)
					if err != nil && rec != nil {
						t.Errorf("UnitRecord rejected and still returned a %d-byte record", len(rec))
					}
					return err
				}},
				{"MoveUnit(0)", func() error { return ed.MoveUnit(0, unitMoveX, unitMoveY) }},
				{"DeleteUnit(0)", func() error { return ed.DeleteUnit(0) }},
				{"PlaceUnit", func() error {
					idx, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex))
					if err != nil && idx >= 0 {
						t.Errorf("a rejected place returned index %d, which a caller could mistake for a placed record", idx)
					}
					return err
				}},
			} {
				t.Run(c.name, func(t *testing.T) {
					rejectionChangesNothing(t, ed, c.call)
				})
			}

			// Everything else about the map still edits, and the result is
			// still a map: the refusal is this record's, not the model's.
			if err := ed.SetTile(1, 0, gridNewTile); err != nil {
				t.Errorf("SetTile was refused on the same map: %v", err)
			}
			reopensIdentically(t, "after a grid edit on a roster with no units", ed.Bytes())
		})
	}
}

// TestAnEmptyUnitsRecordIsNotAnAbsentOne is the other side of that contrast,
// stated on its own so neither reading can drift into the other: with the record
// present and its payload empty, a place is accepted and yields index 0.
func TestAnEmptyUnitsRecordIsNotAnAbsentOne(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			data := richFixture(order, 0)
			m, err := alm.Open(data)
			if err != nil {
				t.Fatalf("alm.Open: %v", err)
			}
			if !m.Present(6) {
				t.Fatal("the empty-roster fixture has no type-6 record, so it is the absent case and not this one")
			}

			ed := newEditor(t, data)
			if got := ed.UnitCount(); got != 0 {
				t.Fatalf("UnitCount = %d on an empty type-6 payload, want 0", got)
			}
			idx, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex))
			if err != nil {
				t.Fatalf("PlaceUnit into an empty but present type-6 record: %v", err)
			}
			if idx != 0 {
				t.Errorf("PlaceUnit returned index %d, want 0", idx)
			}
			reopensIdentically(t, "after placing into an empty type-6 record", ed.Bytes())
		})
	}
}

// ---------------------------------------------------------------------------
// AC-2's move half
// ---------------------------------------------------------------------------

// TestMoveWritesTheTwoCoordinateWordsAndCarriesTheRest is AC-2's move half, each
// mutation on a fresh load: the diff is the record's first eight bytes, those
// eight hold the two raw words, the record's other 62 are the fixture's own, and
// the shipped decoder reads the new coordinates back at that index while every
// other unit's stay where they were.
func TestMoveWritesTheTwoCoordinateWordsAndCarriesTheRest(t *testing.T) {
	for _, v := range fixtureVariants() {
		if v.nUnits == 0 {
			continue
		}
		for i := 0; i < v.nUnits; i++ {
			t.Run(fmt.Sprintf("%s/unit %d", v.name, i), func(t *testing.T) {
				before := richFixture(v.order, v.nUnits)
				ed := newEditor(t, before)

				if err := ed.MoveUnit(i, unitMoveX, unitMoveY); err != nil {
					t.Fatalf("MoveUnit(%d): %v", i, err)
				}
				after := ed.Bytes()
				if len(after) != len(before) {
					t.Fatalf("a move changed the image size: %d -> %d", len(before), len(after))
				}

				fsBefore, fsAfter := walkFrame(t, before), walkFrame(t, after)
				target := fsBefore.payload(t, 6, i*unitRecordSize+unitX, 8)
				if got := fsAfter.payload(t, 6, i*unitRecordSize+unitX, 8); got != target {
					t.Fatalf("the coordinate words moved: %+v -> %+v", target, got)
				}

				if ok, why := carriedIdentical(before, after, []region{target}, []region{target}); !ok {
					t.Errorf("a byte outside the two coordinate words changed: %s", why)
				}

				want := unitCoordImage(unitMoveX, unitMoveY)
				if got := fsAfter.at(t, after, target); !bytes.Equal(got, want) {
					t.Errorf("the coordinate words hold % x, want % x", got, want)
				}

				// The other 62 bytes, stated on their own rather than left
				// to the comparator: this is the clause the whole
				// carry-don't-regenerate design exists for.
				record := fsAfter.at(t, after, fsAfter.unit(t, i))
				if got, want := record[8:], fixUnitRecord(i)[8:]; !bytes.Equal(got, want) {
					t.Errorf("the moved record's other 62 bytes are\n % x\nwant\n % x", got, want)
				}

				m, err := ed.Map()
				if err != nil {
					t.Fatalf("the view failed after the move: %v", err)
				}
				if len(m.Units) != v.nUnits {
					t.Fatalf("the view has %d units, want %d", len(m.Units), v.nUnits)
				}
				for j, u := range m.Units {
					wantX, wantY := fixUnitX(j), fixUnitY(j)
					if j == i {
						wantX, wantY = unitMoveX, unitMoveY
					}
					if u.X != wantX || u.Y != wantY {
						t.Errorf("the view's unit %d is at (%#x,%#x), want (%#x,%#x)", j, u.X, u.Y, wantX, wantY)
					}
				}
				reopensIdentically(t, "after a move", after)
			})
		}
	}
}

// ---------------------------------------------------------------------------
// AC-3 — place and delete
// ---------------------------------------------------------------------------

// TestPlaceProducesTheImageTheTestBuiltItself is the strongest form the place
// case takes: the whole edited image must equal one fixture_test.go assembled
// from the container contract — the record spliced in, the payloadSize word
// corrected, the count word incremented and nothing else touched. A model that
// got any part of the partition wrong differs from it somewhere.
func TestPlaceProducesTheImageTheTestBuiltItself(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			before := richFixture(v.order, v.nUnits)
			want, _, _ := growByOneUnit(t, before)

			ed := newEditor(t, before)
			idx, err := ed.PlaceUnit(fixUnitRecord(fixUnitsMax))
			if err != nil {
				t.Fatalf("PlaceUnit: %v", err)
			}
			if idx != v.nUnits {
				t.Errorf("PlaceUnit returned index %d, want %d", idx, v.nUnits)
			}
			if got := ed.Bytes(); !bytes.Equal(got, want) {
				t.Errorf("the placed image differs from the one this test grew itself (%d vs %d bytes)",
					len(got), len(want))
			}
		})
	}
}

func TestPlaceAppendsVerbatimAndCarriesEveryOtherByte(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			before := richFixture(v.order, v.nUnits)
			ed := newEditor(t, before)

			record := fixUnitRecord(unitPlacedIndex)
			placed := clone(record)

			idx, err := ed.PlaceUnit(record)
			if err != nil {
				t.Fatalf("PlaceUnit: %v", err)
			}
			after := ed.Bytes()

			if idx != v.nUnits {
				t.Fatalf("PlaceUnit returned index %d, want %d", idx, v.nUnits)
			}
			if got := ed.UnitCount(); got != v.nUnits+1 {
				t.Errorf("UnitCount = %d, want %d", got, v.nUnits+1)
			}
			if got, want := len(after), len(before)+unitRecordSize; got != want {
				t.Fatalf("the image is %d bytes, want %d", got, want)
			}

			countBefore, sizeBefore := unitWords(t, before)
			countAfter, sizeAfter := unitWords(t, after)
			if countAfter != countBefore+1 {
				t.Errorf("#type6 = %d, want %d", countAfter, countBefore+1)
			}
			if sizeAfter != sizeBefore+unitRecordSize {
				t.Errorf("the type-6 payloadSize = %d, want %d", sizeAfter, sizeBefore+unitRecordSize)
			}

			bRegions, aRegions := unitLengthChangeRegions(t, before, after, v.nUnits, 0, unitRecordSize)
			if ok, why := carriedIdentical(before, after, bRegions, aRegions); !ok {
				t.Errorf("a byte outside the inserted record and the two count words changed: %s", why)
			}

			// The record's own bytes, from both sides: through the model's
			// read and out of the image at the offset the walk computes.
			got, err := ed.UnitRecord(idx)
			if err != nil {
				t.Fatalf("UnitRecord(%d) after placing it: %v", idx, err)
			}
			if !bytes.Equal(got, placed) {
				t.Errorf("the placed record reads back as\n % x\nwant\n % x", got, placed)
			}
			fsAfter := walkFrame(t, after)
			if inImage := fsAfter.at(t, after, fsAfter.unit(t, idx)); !bytes.Equal(inImage, placed) {
				t.Errorf("the image holds\n % x\nat index %d, want\n % x", inImage, idx, placed)
			}
			for i := 0; i < v.nUnits; i++ {
				old, err := ed.UnitRecord(i)
				if err != nil {
					t.Fatalf("UnitRecord(%d): %v", i, err)
				}
				if !bytes.Equal(old, fixUnitRecord(i)) {
					t.Errorf("placing a record disturbed record %d", i)
				}
			}

			// The caller's slice is not the model's: mutating it afterwards
			// changes nothing that was placed.
			for i := range record {
				record[i] ^= 0xff
			}
			if got := ed.Bytes(); !bytes.Equal(got, after) {
				t.Error("mutating the caller's record slice after PlaceUnit changed the model's bytes")
			}

			m, err := ed.Map()
			if err != nil {
				t.Fatalf("the view failed after the place: %v", err)
			}
			if len(m.Units) != v.nUnits+1 {
				t.Fatalf("the view has %d units, want %d", len(m.Units), v.nUnits+1)
			}
			if u := m.Units[idx]; u.X != fixUnitX(unitPlacedIndex) || u.Y != fixUnitY(unitPlacedIndex) {
				t.Errorf("the view's new unit is at (%#x,%#x), want (%#x,%#x)",
					u.X, u.Y, fixUnitX(unitPlacedIndex), fixUnitY(unitPlacedIndex))
			}
			reopensIdentically(t, "after a place", after)
		})
	}
}

// TestDeleteRemovesTheRecordAndShiftsTheLaterOnesDown is AC-3's delete half. The
// index deleted runs over the whole roster, so the first, a middle and the last
// record each get their turn, and the survivors are checked by identity — record j
// must be the fixture's j below the deletion and its j+1 above it, which is what
// "shifting later units down" means observably.
func TestDeleteRemovesTheRecordAndShiftsTheLaterOnesDown(t *testing.T) {
	for _, order := range fixtureOrders {
		for i := 0; i < fixUnitsMax; i++ {
			t.Run(fmt.Sprintf("%s/unit %d", order.name, i), func(t *testing.T) {
				before := richFixture(order, fixUnitsMax)
				ed := newEditor(t, before)

				if err := ed.DeleteUnit(i); err != nil {
					t.Fatalf("DeleteUnit(%d): %v", i, err)
				}
				after := ed.Bytes()

				if got := ed.UnitCount(); got != fixUnitsMax-1 {
					t.Errorf("UnitCount = %d, want %d", got, fixUnitsMax-1)
				}
				if got, want := len(after), len(before)-unitRecordSize; got != want {
					t.Fatalf("the image is %d bytes, want %d", got, want)
				}

				countBefore, sizeBefore := unitWords(t, before)
				countAfter, sizeAfter := unitWords(t, after)
				if countAfter != countBefore-1 {
					t.Errorf("#type6 = %d, want %d", countAfter, countBefore-1)
				}
				if sizeAfter != sizeBefore-unitRecordSize {
					t.Errorf("the type-6 payloadSize = %d, want %d", sizeAfter, sizeBefore-unitRecordSize)
				}

				bRegions, aRegions := unitLengthChangeRegions(t, before, after, i, unitRecordSize, 0)
				if ok, why := carriedIdentical(before, after, bRegions, aRegions); !ok {
					t.Errorf("a byte outside the removed record and the two count words changed: %s", why)
				}

				for j := 0; j < fixUnitsMax-1; j++ {
					source := j
					if j >= i {
						source = j + 1
					}
					got, err := ed.UnitRecord(j)
					if err != nil {
						t.Fatalf("UnitRecord(%d): %v", j, err)
					}
					if !bytes.Equal(got, fixUnitRecord(source)) {
						t.Errorf("after deleting %d, record %d is not the fixture's record %d", i, j, source)
					}
				}
				if _, err := ed.UnitRecord(fixUnitsMax - 1); err == nil {
					t.Errorf("index %d still reads after the delete", fixUnitsMax-1)
				}

				m, err := ed.Map()
				if err != nil {
					t.Fatalf("the view failed after the delete: %v", err)
				}
				if len(m.Units) != fixUnitsMax-1 {
					t.Fatalf("the view has %d units, want %d", len(m.Units), fixUnitsMax-1)
				}
				for j, u := range m.Units {
					source := j
					if j >= i {
						source = j + 1
					}
					if u.X != fixUnitX(source) || u.Y != fixUnitY(source) {
						t.Errorf("the view's unit %d is at (%#x,%#x), want the fixture's %d at (%#x,%#x)",
							j, u.X, u.Y, source, fixUnitX(source), fixUnitY(source))
					}
				}
				reopensIdentically(t, "after a delete", after)
			})
		}
	}
}

func TestPlacingACloneOfAnExistingRecordIsTheSupportedWorkflow(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(order, fixUnitsMax))

			clone0, err := ed.UnitRecord(0)
			if err != nil {
				t.Fatalf("UnitRecord(0): %v", err)
			}
			idx, err := ed.PlaceUnit(clone0)
			if err != nil {
				t.Fatalf("PlaceUnit(a clone of record 0): %v", err)
			}
			if err := ed.MoveUnit(idx, unitMoveX, unitMoveY); err != nil {
				t.Fatalf("MoveUnit(%d): %v", idx, err)
			}

			got, err := ed.UnitRecord(idx)
			if err != nil {
				t.Fatalf("UnitRecord(%d): %v", idx, err)
			}
			want := fixUnitRecord(0)
			copy(want, unitCoordImage(unitMoveX, unitMoveY))
			if !bytes.Equal(got, want) {
				t.Errorf("the moved clone is\n % x\nwant\n % x", got, want)
			}

			original, err := ed.UnitRecord(0)
			if err != nil {
				t.Fatalf("UnitRecord(0): %v", err)
			}
			if !bytes.Equal(original, fixUnitRecord(0)) {
				t.Error("cloning, placing and moving disturbed the record it was cloned from")
			}
			reopensIdentically(t, "after the clone-and-place workflow", ed.Bytes())
		})
	}
}

// TestPlaceThenDeletingItReproducesTheOriginalImage is AC-3's "rises then falls"
// read as strictly as it can be: the two mutations are exact mirrors, so their
// composition is the identity on the bytes — not merely on the counts.
func TestPlaceThenDeletingItReproducesTheOriginalImage(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			original := richFixture(v.order, v.nUnits)
			ed := newEditor(t, original)

			idx, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex))
			if err != nil {
				t.Fatalf("PlaceUnit: %v", err)
			}
			grown := ed.Bytes()
			if bytes.Equal(grown, original) {
				t.Fatal("the place changed no byte, so this case witnesses nothing")
			}

			if err := ed.DeleteUnit(idx); err != nil {
				t.Fatalf("DeleteUnit(%d): %v", idx, err)
			}
			if got := ed.Bytes(); !bytes.Equal(got, original) {
				t.Errorf("placing a record and deleting it again did not reproduce the image (%d vs %d bytes)",
					len(got), len(original))
			}
			if got := ed.UnitCount(); got != v.nUnits {
				t.Errorf("UnitCount = %d after the round trip, want %d", got, v.nUnits)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SC-7 over the length-changing mutations — the applier's ordering rule
// ---------------------------------------------------------------------------

// TestUndoAndRedoOfALengthChangingEdit is the case T3 could not write and left
// disclosed. A place and a delete are three-part edits in which the count words
// and the inserted bytes sit at different offsets, so the order the applier walks
// its parts in becomes observable — but only in one of the two record orders. With
// type-0 first the count words precede the insertion and a wrong order still lands
// on the right offsets; with type-6 first they sit above it, and a forward walk
// that went ascending would write #type6 at a pre-image offset the insertion has
// already moved, while a revert that went descending would do the mirror of that
// on the way back. Both orders run here, and the second is the one that separates
// them.
//
// The chain is three edits — two places and a delete — so the delete is in it on
// the empty roster too, where deleting the only record placed would otherwise
// bring the image straight back to the one it started from and leave the sequence
// with nothing to distinguish. Every comparison is against a buffer this test
// cloned before the step it judges.
func TestUndoAndRedoOfALengthChangingEdit(t *testing.T) {
	for _, v := range fixtureVariants() {
		t.Run(v.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(v.order, v.nUnits))

			shots := [][]byte{ed.Bytes()}
			for _, edit := range []struct {
				what string
				do   func() error
			}{
				{"place A", func() error { _, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex)); return err }},
				{"place B", func() error { _, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex + 1)); return err }},
				{"delete 0", func() error { return ed.DeleteUnit(0) }},
			} {
				if err := edit.do(); err != nil {
					t.Fatalf("%s: %v", edit.what, err)
				}
				shots = append(shots, ed.Bytes())
			}
			for i := range shots {
				for j := i + 1; j < len(shots); j++ {
					if bytes.Equal(shots[i], shots[j]) {
						t.Fatalf("images %d and %d are equal, so the sequence cannot tell those points apart", i, j)
					}
				}
			}

			steps := []struct {
				what string
				do   func() bool
				want []byte
			}{
				{"Undo", ed.Undo, shots[2]},
				{"Undo, Undo", ed.Undo, shots[1]},
				{"Undo, Undo, Undo", ed.Undo, shots[0]},
				{"Redo", ed.Redo, shots[1]},
				{"Redo, Redo", ed.Redo, shots[2]},
				{"Redo, Redo, Redo", ed.Redo, shots[3]},
			}
			for _, step := range steps {
				if !step.do() {
					t.Fatalf("%s reported nothing to do", step.what)
				}
				got := ed.Bytes()
				if !bytes.Equal(got, step.want) {
					t.Fatalf("after %s the bytes are not the image cloned for that point (%d vs %d bytes)",
						step.what, len(got), len(step.want))
				}
				reopensIdentically(t, "after "+step.what, got)
			}
			if !ed.CanUndo() || ed.CanRedo() {
				t.Errorf("at the end: CanUndo=%v CanRedo=%v, want true and false", ed.CanUndo(), ed.CanRedo())
			}
		})
	}
}

// TestALengthChangingMutationAfterAnUndoDiscardsTheRedoHistory pairs the discard
// rule with edits that change the image size, where a stale redo entry would not
// merely restore the wrong bytes but splice at an offset the buffer no longer has.
func TestALengthChangingMutationAfterAnUndoDiscardsTheRedoHistory(t *testing.T) {
	for _, order := range fixtureOrders {
		t.Run(order.name, func(t *testing.T) {
			ed := newEditor(t, richFixture(order, fixUnitsMax))

			s0 := ed.Bytes()
			if _, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex)); err != nil {
				t.Fatalf("PlaceUnit: %v", err)
			}
			s1 := ed.Bytes()
			if !ed.Undo() {
				t.Fatal("Undo reported nothing to undo")
			}
			if got := ed.Bytes(); !bytes.Equal(got, s0) {
				t.Fatal("undoing the place did not restore the pre-place bytes")
			}

			if err := ed.DeleteUnit(fixUnitsMax - 1); err != nil {
				t.Fatalf("DeleteUnit: %v", err)
			}
			s2 := ed.Bytes()
			if bytes.Equal(s2, s1) {
				t.Fatal("the replacing mutation produced the discarded one's image")
			}

			if ed.CanRedo() {
				t.Error("CanRedo is true after a mutation that should have discarded the redo history")
			}
			if ed.Redo() {
				t.Error("Redo re-applied a place a later delete had discarded")
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
			reopensIdentically(t, "after the discard sequence", ed.Bytes())
		})
	}
}

// ---------------------------------------------------------------------------
// SC-6 — the unit rejections
// ---------------------------------------------------------------------------

// TestUnitSetterRejections is AC-4's unit half: an out-of-range index for each of
// the three index-taking calls, and a record one byte short and one byte long for
// the place. Each runs against three history states, the undone one being the
// state that can see a setter apply before it validates, and over a map with no
// units as well as one with several — on an empty roster every index is out of
// range, which is the case a bounds check written as "> n" gets wrong.
func TestUnitSetterRejections(t *testing.T) {
	badIndexCalls := []struct {
		name string
		call func(t *testing.T, ed *mapedit.Editor, i int) error
	}{
		{"UnitRecord", func(t *testing.T, ed *mapedit.Editor, i int) error {
			r, err := ed.UnitRecord(i)
			if err != nil && r != nil {
				t.Errorf("UnitRecord(%d) rejected the index and still returned a %d-byte record", i, len(r))
			}
			return err
		}},
		{"MoveUnit", func(_ *testing.T, ed *mapedit.Editor, i int) error {
			return ed.MoveUnit(i, unitMoveX, unitMoveY)
		}},
		{"DeleteUnit", func(_ *testing.T, ed *mapedit.Editor, i int) error { return ed.DeleteUnit(i) }},
	}
	badRecords := []struct {
		name   string
		record []byte
	}{
		{"nil", nil},
		{"empty", []byte{}},
		{"69 bytes", fixUnitRecord(0)[:unitRecordSize-1]},
		{"71 bytes", append(fixUnitRecord(0), 0x5a)},
	}

	// Each state declares the roster size it leaves behind, so the boundary
	// index below is the test's own arithmetic and not the model's answer read
	// back to it.
	states := []struct {
		name  string
		count func(nUnits int) int
		setUp func(t *testing.T, ed *mapedit.Editor)
	}{
		{"a fresh model", func(n int) int { return n }, func(*testing.T, *mapedit.Editor) {}},
		{"one accepted place", func(n int) int { return n + 1 }, func(t *testing.T, ed *mapedit.Editor) {
			if _, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex)); err != nil {
				t.Fatalf("the set-up place failed: %v", err)
			}
		}},
		{"one place, undone", func(n int) int { return n }, func(t *testing.T, ed *mapedit.Editor) {
			if _, err := ed.PlaceUnit(fixUnitRecord(unitPlacedIndex)); err != nil {
				t.Fatalf("the set-up place failed: %v", err)
			}
			if !ed.Undo() {
				t.Fatal("the set-up Undo reported nothing to undo")
			}
		}},
	}

	for _, nUnits := range []int{0, fixUnitsMax} {
		for _, state := range states {
			// The first out-of-range index is the roster size itself, which
			// is the boundary a check written as "> n" gets wrong; on an
			// empty roster that boundary is zero.
			n := state.count(nUnits)
			outside := []int{-1, -unitRecordSize, n, n + 1, 1 << 20}

			for _, c := range badIndexCalls {
				for _, i := range outside {
					t.Run(fmt.Sprintf("%d units/%s/%s(%d)", nUnits, state.name, c.name, i), func(t *testing.T) {
						ed := newEditor(t, richFixture(orderType6First, nUnits))
						state.setUp(t, ed)
						if got := ed.UnitCount(); got != n {
							t.Fatalf("the set-up left %d units, want %d", got, n)
						}
						rejectionChangesNothing(t, ed, func() error { return c.call(t, ed, i) })
					})
				}
			}
			for _, r := range badRecords {
				t.Run(fmt.Sprintf("%d units/%s/PlaceUnit(%s)", nUnits, state.name, r.name), func(t *testing.T) {
					ed := newEditor(t, richFixture(orderType6First, nUnits))
					state.setUp(t, ed)
					rejectionChangesNothing(t, ed, func() error {
						idx, err := ed.PlaceUnit(r.record)
						if err != nil && idx >= 0 {
							t.Errorf("a rejected place returned index %d, which a caller could mistake for a placed record", idx)
						}
						return err
					})
				})
			}
		}
	}
}
