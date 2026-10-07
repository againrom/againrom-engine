package sim

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"strings"
	"testing"
)

func widenedCellPlanePin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 83
	return widenedCurrentObjectsPin(out)
}

func strippedCellPlanePin(form []byte) []byte {
	out := strippedCurrentObjectsPin(form)
	if len(out) > 0 && out[0] >= 83 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 82
	}
	return out
}

// Form84 adds only its explicit absent-object span to these independently
// transcribed legacy fixtures. Neither adapter calls the production codec.
func widenedCurrentObjectsPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 84
	return widenedCarriedResumePin(out)
}

// Form85 adds only its own explicit empty span, outermost, on the same
// pattern as every span from form76 on.
func widenedCarriedResumePin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 85
	return widenedActionClockPin(out)
}

func strippedCurrentObjectsPin(form []byte) []byte {
	out := strippedCarriedResumePin(form)
	if len(out) > 0 && out[0] >= 84 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 83
	}
	return out
}

// Form85 appends one more empty span outermost, on the same pattern as every
// span from form76 on: this is the new base of the whole stripped*Pin chain,
// on strippedCurrentObjectsPin's own former place as the base (bytes.Clone
// with no further call) before this story.
func strippedCarriedResumePin(form []byte) []byte {
	out := strippedActionClockPin(form)
	if len(out) > 0 && out[0] >= 85 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 84
	}
	return out
}

func cellPlanesFixture(t *testing.T, existingDestination bool) (*World, *SavedCellPlanes) {
	t.Helper()
	w, m, cells, blocks := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	cells[0].Payload[0], cells[0].Payload[1], cells[0].Payload[17] = 9, 0, 0
	if existingDestination {
		cells[1].Payload[0], cells[1].Payload[1] = 7, 0
	} else {
		cells = cells[:1]
	}
	importMotion1115(t, w, m, cells, blocks)
	p := new(SavedCellPlanes)
	p.Costs[0] = 255
	p.Costs[5] = 6
	p.Cost[0x100f], p.Cost[0x1010] = 37, 11
	p.CostKnown[0x100f], p.CostKnown[0x1010] = 1, 1
	p.Static[0x100f], p.Dynamic[0x100f] = 0x20, 0x60
	p.Dynamic[0x1010] = 0x40
	p.Height[0x100f], p.Height[0x1010] = 17, 19
	return w, p
}

func TestSavedCellPlanes1115CreationDetachDeletionAndCurrentBlocks(t *testing.T) {
	w, p := cellPlanesFixture(t, false)
	if err := w.ImportOriginalStructures(nil, nil, []SavedStructureCell{{Cell: 0x100f, BaselineCost: 9}}, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	if got, _ := w.SavedCellPlanes(); got.Cost[0x100f] != 37 || len(w.savedMotion.Cells) != 1 {
		t.Fatal("LOAD recomputed or constructed cells")
	}
	for tick := 0; tick < 5; tick++ {
		var cold World
		if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
			t.Fatal(err)
		}
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("tick%d fresh native continuation differs", tick)
		}
		planes, _ := w.SavedCellPlanes()
		if w.motionCell(0x100f) != nil || w.motionCell(0x1010) == nil {
			t.Fatal("old node was kept or new node missing")
		}
		c := w.motionCell(0x1010)
		want := [52]byte{}
		want[0] = 11
		binary.LittleEndian.PutUint32(want[4:], 0x1004)
		if c.Payload != want {
			t.Fatal("new constructor did not zero52/capture current baseline", c.Payload)
		}
		if planes.Cost[0x100f] != 9 || planes.Static[0x100f] != 0 || planes.Cost[0x1010] != 11 || planes.Static[0x1010] != 0x20 || planes.Dynamic[0x1010] != 0x60 {
			t.Fatal("baseline restore/recompute differs")
		}
		wantOld := byte(0x40)
		if tick == 4 {
			wantOld = 0
		}
		if planes.Dynamic[0x100f] != wantOld {
			t.Fatal("reservation lifetime crossed deletion incorrectly")
		}
		_, sc, present := w.SavedStructures()
		if !present || len(sc) != 1 || sc[0].Cell != 0x1010 || sc[0].BaselineCost != 11 || sc[0].HasStructure {
			t.Fatal("structure-cell population not synchronized", sc)
		}
		_, _, bs, _ := w.SavedActorMotions()
		wantBlocks := 1
		if tick < 4 {
			wantBlocks = 2
		}
		if len(bs) != wantBlocks {
			t.Fatal("obsolete block record survived current plane scan", bs)
		}
		if !strings.Contains(w.motionFor(7).Issue, "speed callback") || strings.Contains(w.motionFor(7).Issue, "deletion") {
			t.Fatal("implemented lifecycle still disclosed as absent")
		}
	}
}

func TestSavedCellPlanes1115ExistingPayloadAndLiveTailOwnDeletion(t *testing.T) {
	for _, keep := range []bool{false, true} {
		w, p := cellPlanesFixture(t, true)
		w.motionCell(0x100f).Payload[0x2c] = 5 // stale imported operation
		if err := w.ImportOriginalCellPlanes(p); err != nil {
			t.Fatal(err)
		}
		tail := [6]byte{}
		if keep {
			tail = [6]byte{5, 20, 1, 2, 3, 4}
		}
		w.writeCellTail(0x100f, tail)
		Step(w, nil)
		if (w.motionCell(0x100f) != nil) != keep {
			t.Fatal("deletion used stale imported trigger")
		}
		if c := w.motionCell(0x1010); c.Payload[0] != 7 || c.Payload[17] != 0x93 {
			t.Fatal("existing node was reconstructed from current planes")
		}
		if w.savedCellPlanes.Cost[0x1010] != 7 {
			t.Fatal("existing cell did not recompute from captured baseline")
		}
	}
}

func TestSavedCellPlanes1115RecomputeExactLayersBuildingsAndBitFour(t *testing.T) {
	w, p := cellPlanesFixture(t, true)
	st := Structure{ID: 9, Col: 16, Row: 16, Width: 2, Height: 1, Attach: 3}
	src := SavedStructure{ID: 9, Class: SavedBuilding, SourceKey: 0xbeef, ArchiveIndex: 9, Blocking: 1}
	src.Position[0], src.Position[1] = 16, 16
	src.Base52[14], src.Base52[15] = 2, 1
	binary.LittleEndian.PutUint32(src.Base52[18:], 1)
	if err := w.ImportOriginalStructures([]Structure{st}, []SavedStructure{src}, nil, bytes.Clone(w.grid)); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	c := w.motionCell(0x1010)
	c.Payload = [52]byte{}
	c.Payload[0], c.Payload[1], c.Payload[2] = 19, 0, 6
	c.Ground = SavedActorSlot{Key: 0x1234}
	c.Air = SavedActorSlot{Key: 0x5678}
	binary.LittleEndian.PutUint32(c.Payload[4:], 0x1234)
	binary.LittleEndian.PutUint32(c.Payload[8:], 0x5678)
	binary.LittleEndian.PutUint32(c.Payload[12:], 0xbeef)
	w.savedCellPlanes.Static[c.Cell] = 0x10
	if issue := w.recomputeSavedCell(c.Cell); issue != "" {
		t.Fatal(issue)
	}
	if p := w.savedCellPlanes; p.Cost[c.Cell] != 19 || p.Static[c.Cell] != 0x35 || p.Dynamic[c.Cell] != 0xf5 {
		t.Fatal("blocking footprint or actor bits differ")
	}
	// Move the source Building anchor left: offset1's clear bit opens this
	// exact saved key; footprint geometry is not used to choose the Building.
	w.structures[0].Col = 15
	w.savedStructures[0].Position[0] = 15
	for at := 20; at <= 40; at += 4 {
		binary.LittleEndian.PutUint32(c.Payload[at:], uint32(at+1))
	}
	if issue := w.recomputeSavedCell(c.Cell); issue != "" {
		t.Fatal(issue)
	}
	if p := w.savedCellPlanes; p.Cost[c.Cell] != 0 || p.Static[c.Cell] != 0x35 || p.Dynamic[c.Cell] != 0xf5 {
		t.Fatal("six byte shifts/fourth blocking layer/bit4 differ")
	}
	clear(c.Payload[20:44])
	c.Payload[2] = 6 // count is not a layer pointer
	if issue := w.recomputeSavedCell(c.Cell); issue != "" {
		t.Fatal(issue)
	}
	if p := w.savedCellPlanes; p.Cost[c.Cell] != 6 || p.Static[c.Cell] != 0x30 || p.Dynamic[c.Cell] != 0xf0 {
		t.Fatal("clear Building bit should open/use CostCracked")
	}
	if w.costAt(cell{16, 16}) != 6 || w.heightAt(cell{16, 16}) != 19 || !w.terrainOpen(DomainGround, 16, 16) || !w.terrainOpen(DomainGhost, 16, 16) {
		t.Fatal("current planes do not drive routing/rate readers")
	}
	w.refreshSavedPlaneBlocks()
	_ = mustMarshal(t, w)
}

func TestSavedCellPlanes1115NativeConsistencyIsAtomic(t *testing.T) {
	w, p := cellPlanesFixture(t, true)
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	valid := mustMarshal(t, w)
	start := len(valid) - entityIDFloorLen - spellDeliverySpanLen - 40 - 327691
	gridAt := 34 + 16*32 + 15
	blocksAt := start - 4 - 2*4
	for _, tc := range []struct {
		name string
		edit func([]byte)
	}{
		{"raw-static-vs-grid", func(b []byte) { b[start+65536+0x100f] ^= 1 }},
		{"grid-vs-raw-static", func(b []byte) { b[gridAt] ^= 1 }},
		{"raw-dynamic-vs-block", func(b []byte) { b[start+2*65536+0x100f] ^= 0x40 }},
		{"block-vs-raw-plane", func(b []byte) { b[blocksAt+2] ^= 0x40 }},
		{"class-zero-cost", func(b []byte) { b[start+5*65536] = 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(valid)
			tc.edit(bad)
			if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
				t.Fatal("conflicting native authority was adopted", err)
			}
		})
	}
	// The native magic-wall overlay is separately owned, not an original raw
	// static bit; a legitimate overlay may differ from the translated plane.
	overlay := bytes.Clone(valid)
	overlay[gridAt] |= 4
	var cold World
	if err := cold.UnmarshalBinary(overlay); err != nil || !bytes.Equal(overlay, mustMarshal(t, &cold)) {
		t.Fatal("independent native magic-wall overlay rejected", err)
	}
}

func TestSavedCellPlanes1115ExplicitTailNodesAndAtomicImport(t *testing.T) {
	w, p := cellPlanesFixture(t, true)
	tail := [6]byte{9, 31, 7, 8, 6, 5}
	w.writeCellTail(0x1111, tail)
	w.writeCellTail(0x1212, tail)
	p.CostKnown[0x1111], p.Cost[0x1111], p.Static[0x1111] = 1, 13, 0x10
	before := mustMarshal(t, w)
	if err := w.ImportOriginalCellPlanes(p); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("late unknown tail-only baseline partly adopted")
	}
	p.CostKnown[0x1212], p.Cost[0x1212] = 1, 21
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	c := w.motionCell(0x1111)
	var want [52]byte
	want[0], want[1] = 13, 0x10
	copy(want[0x2c:0x32], tail[:])
	if c == nil || c.Payload != want || w.savedCellPlanes.Static[0x1111] != 0x30 || w.savedCellPlanes.Dynamic[0x1111] != 0x30 {
		t.Fatal("explicit tail owner missing or baseline captured after record bit")
	}
	// A later native tail write creates its node immediately, not on SAVE.
	w.savedCellPlanes.CostKnown[0x1313], w.savedCellPlanes.Cost[0x1313] = 1, 23
	w.writeCellTail(0x1313, tail)
	if c := w.motionCell(0x1313); c == nil || c.Payload[0] != 23 || !bytes.Equal(c.Payload[0x2c:0x32], tail[:]) {
		t.Fatal("later explicit tail owner did not create a node")
	}
	_ = mustMarshal(t, w)
}

func TestSavedCellPlanes1115DeletedTailResidueDoesNotResurrect(t *testing.T) {
	w, p := cellPlanesFixture(t, false)
	w.writeCellTail(0x100f, [6]byte{0, 31, 7, 8, 6, 5})
	p.Static[0x100f] |= 0x10
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	Step(w, nil)
	if w.motionCell(0x100f) != nil || len(w.CellTails()) != 0 || w.savedCellPlanes.Cost[0x100f] != 9 || w.savedCellPlanes.Static[0x100f] != 0x10 {
		t.Fatal("deleted node retained a ghost tail or lost baseline/bit4")
	}
	if issue := w.createSavedCell(0x100f); issue != "" {
		t.Fatal(issue)
	}
	if c := w.motionCell(0x100f); c.Payload[1] != 0x10 || !bytes.Equal(c.Payload[0x2c:0x32], make([]byte, 6)) {
		t.Fatal("re-entry resurrected deleted trigger residue")
	}
	w.recomputeSavedCell(0x100f)
	w.refreshSavedPlaneBlocks()
	_ = mustMarshal(t, w)
}

func TestSavedCellPlanes1115FirstNativeTailNodeAndConflictingTailFault(t *testing.T) {
	w := mustWorld(t, 1115, Bounds{32, 32}, nil)
	tail := [6]byte{0x73, 0x59, 0x31, 0x4d, 0x29, 0x61}
	w.writeCellTail(0x1111, tail)
	p := new(SavedCellPlanes)
	p.Costs[0] = 255
	before := mustMarshal(t, w)
	if err := w.ImportOriginalCellPlanes(p); err == nil || !bytes.Equal(before, mustMarshal(t, w)) || w.savedMotion != nil {
		t.Fatal("unknown first-node baseline was partly adopted")
	}
	p.CostKnown[0x1111], p.Cost[0x1111] = 1, 17
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	if w.savedMotion == nil || len(w.savedMotion.Motions) != 0 || len(w.savedMotion.Cells) != 1 || w.motionCell(0x1111).Payload[0] != 17 {
		t.Fatal("explicit first tail did not create only its typed Cell node")
	}
	valid := mustMarshal(t, w)
	first, last := bytes.Index(valid, tail[:]), bytes.LastIndex(valid, tail[:])
	if first < 0 || first == last || bytes.Count(valid, tail[:]) != 2 {
		t.Fatal("distinct native-tail/raw-cell literal witness missing")
	}
	for _, at := range []int{first + 1, last + 1} {
		bad := bytes.Clone(valid)
		bad[at] ^= 1
		if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
			t.Fatal("conflicting trigger bytes were adopted", err)
		}
	}
	var cold World
	if err := cold.UnmarshalBinary(valid); err != nil || !bytes.Equal(valid, mustMarshal(t, &cold)) {
		t.Fatal("first-node state failed native roundtrip", err)
	}
}

func TestSavedCellPlanes1115UnknownCostAndUnresolvedBuildingRefuseBeforeMutation(t *testing.T) {
	for _, building := range []bool{false, true} {
		w, p := cellPlanesFixture(t, building)
		if building {
			binary.LittleEndian.PutUint32(w.motionCell(0x1010).Payload[12:], 0xdeadbeef)
		} else {
			p.CostKnown[0x1010] = 0
		}
		if err := w.ImportOriginalCellPlanes(p); err != nil {
			t.Fatal(err)
		}
		before, _ := w.SavedCellPlanes()
		cells := append([]SavedActorCell(nil), w.savedMotion.Cells...)
		Step(w, nil)
		after, _ := w.SavedCellPlanes()
		if *before != *after || !reflect.DeepEqual(cells, w.savedMotion.Cells) {
			t.Fatal("unsupported recomputation partially changed planes/nodes")
		}
		if w.motionFor(7).Issue == "" || w.ActorMotionActive(7) {
			t.Fatal("missing authority silently accepted")
		}
		_ = mustMarshal(t, w)
	}
}

func TestSavedCellPlanes1115FlatBlockScanBoundaries(t *testing.T) {
	w, p := cellPlanesFixture(t, true)
	for _, k := range []int{0x806, 0x807, 0x8ff, 0x900, 0xeded, 0xedee} {
		p.Static[k] = byte(k)
		p.Dynamic[k] = 16
	}
	p.Dynamic[0x100f], p.Dynamic[0x1010] = 15, 0
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	_, _, bs, _ := w.SavedActorMotions()
	want := []SavedActorBlock{{Cell: 0x807, Dyn: 16, Static: 7}, {Cell: 0x8ff, Dyn: 16, Static: 255}, {Cell: 0x900, Dyn: 16}, {Cell: 0xeded, Dyn: 16, Static: 237}}
	if !reflect.DeepEqual(bs, want) {
		t.Fatal("scan became a rectangle, used static threshold, or retained drops", bs)
	}
}

func TestSavedCellPlanes1115FixedWireDetachedAtomic(t *testing.T) {
	w, p := cellPlanesFixture(t, true)
	p.Cost[0x2222], p.Static[0x2222], p.Dynamic[0x2222], p.Height[0x2222], p.CostKnown[0x2222] = 1, 2, 3, 4, 1
	p.Costs[10] = 11
	before := mustMarshal(t, w)
	badp := *p
	badp.CostKnown[65535] = 2
	if err := w.ImportOriginalCellPlanes(&badp); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("late invalid knowledge partly imported")
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	valid := mustMarshal(t, w)
	const span = 327691
	start := len(valid) - entityIDFloorLen - spellDeliverySpanLen - 40 - span
	if valid[0] != formatVersion || binary.LittleEndian.Uint32(valid[len(valid)-entityIDFloorLen-spellDeliverySpanLen-40:]) != span || !bytes.Equal(valid[len(valid)-entityIDFloorLen-spellDeliverySpanLen-36:len(valid)-entityIDFloorLen-spellDeliverySpanLen], make([]byte, 36)) {
		t.Fatal("wrong independent fixed suffix")
	}
	for plane, want := range []byte{1, 2, 3, 4, 1} {
		if valid[start+plane*65536+0x2222] != want {
			t.Fatal("wire plane order differs")
		}
	}
	if valid[len(valid)-entityIDFloorLen-spellDeliverySpanLen-41] != 11 {
		t.Fatal("Cost table not last")
	}
	p.Cost[0x2222] = 99
	got, _ := w.SavedCellPlanes()
	got.Cost[0x2222] = 88
	if w.savedCellPlanes.Cost[0x2222] != 1 {
		t.Fatal("import/getter aliases caller")
	}
	for _, mutate := range []func([]byte) []byte{
		func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-40:], math.MaxUint32)
			return b
		},
		func(b []byte) []byte {
			binary.LittleEndian.PutUint32(b[len(b)-entityIDFloorLen-spellDeliverySpanLen-40:], span-1)
			return b
		},
		func(b []byte) []byte { b[start+4*65536+65535] = 2; return b },
		func(b []byte) []byte { return b[:len(b)-1] },
	} {
		bad := mutate(bytes.Clone(valid))
		if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
			t.Fatal("malformed plane suffix partly adopted", err)
		}
	}
	var cold World
	if err := cold.UnmarshalBinary(valid); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("native plane roundtrip", err)
	}
}

func TestSavedCellPlanes1115Historical82LiteralPinDigests(t *testing.T) {
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, 0x6146553a6b832a65}, {rtfBytes, 0x81b722e3112d3074}} {
		if old := strippedCellPlanePin(tc.form); old[0] != 82 || fnv1a(old) != tc.hash {
			t.Fatal("changed literal predecessor82 bytes")
		}
	}
}

func TestSavedCellPlanes1115Historical83LiteralPinDigests(t *testing.T) {
	for _, tc := range []struct {
		form []byte
		hash uint64
	}{{pinBytes, 0x603e43cc48d5bb6e}, {rtfBytes, 0x7c3db8d18e5c0a77}} {
		old := strippedCurrentObjectsPin(tc.form)
		if old[0] != 83 || fnv1a(old) != tc.hash {
			t.Fatal("changed literal predecessor83 bytes")
		}
		// pinBytes and rtfBytes both close on entityIDFloorLen=10 (see their own
		// var declarations); the widen chain up to widenedAttackNoticePin stops
		// at form93 to keep areaHeaderDigest1164's frozen digest untouched, so
		// form94's own outermost floor wrap is applied here explicitly.
		if tc.form[0] != formatVersion || !bytes.Equal(tc.form, widenedEntityIDFloorPin(widenedCurrentObjectsPin(old), 10)) {
			t.Fatal("current pin differs beyond absent object span/version")
		}
	}
}
