package sim

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"testing"
)

// These outer adapters are independent of the producer/decoder. They preserve
// every earlier literal fixture and remove only the newly specified envelope.
func widenedSavedMotionPin(old []byte) []byte {
	out := append(bytes.Clone(old), 0, 0, 0, 0)
	out[0] = 82
	return widenedCellPlanePin(out)
}

func strippedSavedMotionPin(form []byte) []byte {
	out := strippedCellPlanePin(form)
	if len(out) > 0 && out[0] >= 82 {
		span := int(binary.LittleEndian.Uint32(out[len(out)-4:]))
		out = out[:len(out)-4-span]
		out[0] = 81
	}
	return out
}

func motionFixture1115(t *testing.T, fineX, fineY uint8, stepX, stepY int8, direction, elapsed, total uint16) (*World, SavedActorMotion, []SavedActorCell, []SavedActorBlock) {
	t.Helper()
	w := mustWorld(t, 1115, Bounds{32, 32}, []Entity{{ID: 7, Owner: 1, X: 15, Y: 16, HP: 100, MaxHP: 100, Speed: 1000, Facing: 64}})
	o := SavedActorOrder{Entity: 7, State: 0xb}
	o.Raw[8], o.Raw[9], o.Raw[10], o.Raw[11] = 1, 3, 20, 16
	g := SavedGroup{ID: 1, Selector: 1, Members: []SavedGroupMember{{Archive: 1, Entity: 7, Bound: true}}}
	g.AI[0x20], g.AI[0x45] = 4, 1
	if err := w.ImportSavedGroups([]SavedGroup{g}, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	m := SavedActorMotion{Entity: 7, Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: fineX, FineY: fineY, Residue: 0x9234, TerrainKey: 0x31415926}, StaticRoute: []uint16{0x1010, 0x1011}, DynamicRoute: []uint16{0x1010}}
	m.ActorAction = 77
	m.Mover[0], m.Mover[1], m.Mover[10], m.Mover[33] = 64, 64, 100, 0xcd
	for _, at := range []int{6, 0x74, 0x76, 0x80, 0xa6} {
		binary.LittleEndian.PutUint16(m.Mover[at:], 0x1010)
	}
	binary.LittleEndian.PutUint16(m.Mover[0x70:], 0x100f)
	binary.LittleEndian.PutUint16(m.Mover[0xa8:], 32)
	binary.LittleEndian.PutUint16(m.Mover[0xaa:], total)
	binary.LittleEndian.PutUint16(m.Mover[0xac:], elapsed)
	binary.LittleEndian.PutUint16(m.Mover[0xae:], direction)
	m.Mover[0xb0], m.Mover[0xb1] = byte(stepX), byte(stepY)
	cells := []SavedActorCell{{Cell: 0x100f, Ground: SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}}, {Cell: 0x1010}}
	binary.LittleEndian.PutUint32(cells[0].Payload[4:], 0x1004)
	cells[0].Payload[17], cells[1].Payload[17] = 0x92, 0x93
	blocks := []SavedActorBlock{{Cell: 0x100f, Dyn: 0x43, Static: 7}, {Cell: 0x1010, Dyn: 0x45, Static: 9}}
	return w, m, cells, blocks
}

func importMotion1115(t *testing.T, w *World, m SavedActorMotion, cells []SavedActorCell, blocks []SavedActorBlock) {
	t.Helper()
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{m}, cells, blocks); err != nil {
		t.Fatal(err)
	}
}

func TestSavedMotion1115LiteralCrossingEachSaveAndIndependentOccupancy(t *testing.T) {
	w, m, cells, blocks := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	binary.LittleEndian.PutUint16(m.Mover[0x72:], 0x2468)
	for j := 0; j < 8; j++ {
		m.Mover[0x82+j] = byte(0xe0 + j)
	}
	w.tick = 123 // restored timestamp is not an already-executed candidate
	importMotion1115(t, w, m, cells, blocks)
	for tick, wantFine := range []uint8{0, 32, 64, 96, 128} {
		form := mustMarshal(t, w)
		var cold World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("tick%d cold continuation differs", tick+1)
		}
		got, cs, bs, present := w.SavedActorMotions()
		e := w.Entities()[0]
		wantAction := uint32(1)
		if tick == 4 {
			wantAction = 0
		}
		if !present || e.X != 16 || e.Y != 16 || e.Transit != 0 || e.Stride.Present || got[0].Position.FineX != wantFine || got[0].Position.FineY != 128 || !got[0].Current || got[0].Issue != "original boundary speed callback is not executed" || got[0].ActorAction != wantAction {
			t.Fatalf("tick%d fine displacement: entity=%+v motion=%+v", tick+1, e, got[0])
		}
		if got[0].Position.Residue != 0x9234 || got[0].Position.TerrainKey != 0x31415926 || got[0].Mover[33] != 0xcd || !reflect.DeepEqual(got[0].StaticRoute, m.StaticRoute) {
			t.Fatal("unrelated position/mover/static route changed")
		}
		// Literal source order from canonical SAV-CELLFAIL-583/LEAVE-584:
		// entry X/Y/fractionX/fractionY, then successful old detach's same
		// fields. These boundary caches do not follow later in-cell steps.
		if !bytes.Equal(got[0].Mover[0x82:0x8a], []byte{16, 16, 0, 128, 15, 16, 224, 128}) || binary.LittleEndian.Uint16(got[0].Mover[0x72:]) != 0x2468 {
			t.Fatal("cache source order differs or unimplemented callback result invented")
		}
		if cs[0].Ground.Key != 0 || cs[1].Ground != (SavedActorSlot{Key: 0x1004, Entity: 7, Bound: true}) || cs[0].Payload[17] != 0x92 || cs[1].Payload[17] != 0x93 {
			t.Fatal("independent slots/payload not migrated")
		}
		if occupants, _ := w.cellLayerOccupants(15, 16); len(occupants) != 0 {
			t.Fatal("old reservation was mistaken for actor slot", occupants)
		}
		if occupants, _ := w.cellLayerOccupants(16, 16); len(occupants) != 1 || occupants[0] != 0 {
			t.Fatal("new actor slot not used by cell layers", occupants)
		}
		scratch := newRouteScratch(w)
		oldAt, _ := w.cellIndex(15, 16)
		destAt, _ := w.cellIndex(16, 16)
		wantOld := int32(1)
		if tick == 4 {
			wantOld = 0
		}
		if scratch.at(0, oldAt) != wantOld || scratch.at(0, destAt) != 1 {
			t.Fatalf("tick%d occupancy old%d new%d", tick+1, scratch.at(0, oldAt), scratch.at(0, destAt))
		}
		wantDyn := byte(0x43)
		if tick == 4 {
			wantDyn = 3
		}
		if bs[0].Dyn != wantDyn || bs[1].Dyn != 0x45 || bs[0].Static != 7 || bs[1].Static != 9 {
			t.Fatal("reservation/slot bits differ", bs)
		}
		fx, fy, ok := w.ActorFinePosition(7)
		if !ok || fx != wantFine || fy != 128 || w.ActorMotionActive(7) != (tick != 4) {
			t.Fatal("UI seam not current")
		}
		if tick == 4 {
			for _, at := range []int{0xa8, 0xaa, 0xac, 0x80, 0xa6} {
				if binary.LittleEndian.Uint16(got[0].Mover[at:]) != 0 {
					t.Fatalf("arrival left operand%02x", at)
				}
			}
			if len(got[0].DynamicRoute) != 0 || w.savedOrder(7).Raw[9] != 0 {
				t.Fatal("arrival failed route/progress cleanup")
			}
		} else if binary.LittleEndian.Uint16(got[0].Mover[0xa6:]) != 0x100f {
			t.Fatal("boundary failed to retain old packed anchor")
		}
	}
}

func TestSavedMotion1115EntryCachePrecedesArrivalFractionSnap(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 0, 1)
	importMotion1115(t, w, m, cs, bs)
	Step(w, nil)
	got := w.motionFor(7)
	if got.Position.Cell != 0x1010 || got.Position.FineX != 128 || got.ActorAction != 0 || got.Active || !bytes.Equal(got.Mover[0x82:0x8a], []byte{16, 16, 0, 128, 15, 16, 224, 128}) {
		t.Fatal("entry cache incorrectly used the later centered Position", got)
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil || cold.Hash() != w.Hash() {
		t.Fatal("arrival cache save/load differs", err)
	}
}

func TestSavedMotion1115EmptyCellDeletionIsAnExplicitCoverageGap(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	cs[0].Payload[17] = 0 // remove the fixture's independent nonactor retainer
	cs[0].Payload[3], cs[0].Payload[50], cs[0].Payload[51] = 0x7b, 0x8c, 0x9d
	importMotion1115(t, w, m, cs, bs)
	for range 5 {
		Step(w, nil)
	}
	got := w.motionFor(7)
	if got.Issue != "original empty cell deletion and baseline restoration are not executed" || got.Position.Cell != 0x1010 || got.Position.FineX != 128 || got.Active || got.ActorAction != 0 {
		t.Fatal("deletion gap silently projected or stopped known crossing", got)
	}
	if len(w.savedMotion.Cells) != 2 {
		t.Fatal("unimplemented deletion was enacted")
	}
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal("known coverage gap blocked native preservation", err)
	}
}

func TestSavedMotion1115DeletionPredicateIgnoresResidueButUsesEveryRetainer(t *testing.T) {
	p := [52]byte{}
	p[0], p[1], p[3], p[0x2d], p[0x2e], p[0x2f], p[0x30], p[0x31], p[0x32], p[0x33] = 1, 2, 3, 4, 5, 6, 7, 8, 9, 10
	if !motionCellDeletionEligible(p) {
		t.Fatal("baseline/residue/unusedtail falsely retains a node")
	}
	for _, at := range []int{2, 4, 7, 8, 11, 12, 15, 16, 19, 0x2c} {
		q := p
		q[at] = 1
		if motionCellDeletionEligible(q) {
			t.Fatalf("retainer+%02x ignored", at)
		}
	}
}

func TestSavedMotion1115SignedAndSpecialAxisMath(t *testing.T) {
	for _, tc := range []struct {
		name   string
		fx, fy uint8
		sx, sy int8
		dir    uint16
		cell   uint16
		wx, wy uint8
	}{
		{"samecell", 128, 128, 32, 0, 2, 0x100f, 160, 128},
		{"negative", 16, 128, -32, 0, 6, 0x100e, 240, 128},
		{"ne-zero", 224, 32, 32, -32, 1, 0x100f, 255, 0},
		{"sw-zero", 32, 224, -32, 32, 5, 0x110e, 255, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, m, cs, bs := motionFixture1115(t, tc.fx, tc.fy, tc.sx, tc.sy, tc.dir, 1, 20)
			// A centered initial Position with pending operands is deliberately
			// unsupported. Use a noncenter fraction for the in-cell witness.
			if tc.name == "samecell" {
				m.Position.FineY = 127
				tc.wy = 127
			}
			if tc.cell != 0x100f && tc.cell != 0x1010 {
				cs = append(cs, SavedActorCell{Cell: tc.cell})
				bs = append(bs, SavedActorBlock{Cell: tc.cell})
			}
			importMotion1115(t, w, m, cs, bs)
			Step(w, nil)
			got := w.motionFor(7)
			if got.Position.Cell != tc.cell || got.Position.FineX != tc.wx || got.Position.FineY != tc.wy {
				t.Fatalf("want%04x/%d,%d got%+v", tc.cell, tc.wx, tc.wy, got.Position)
			}
		})
	}
}

func TestSavedMotion1115CommandRateAndFacingDoNotReplaceAcceptedStep(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	importMotion1115(t, w, m, cs, bs)
	Step(w, []Command{{Entity: 7, X: 15, Y: 12}})
	if !w.motionFor(7).Active || w.motionFor(7).Issue == "" {
		t.Fatal("replacement did not retain active with explicit route coverage gap")
	}
	w.entities[0].Speed = 1
	if !w.turnToward(0, 0, -1) || w.entities[0].Facing != 64 {
		t.Fatal("cast/turn stole active facing authority")
	}
	for range 4 {
		Step(w, nil)
	}
	if p := w.motionFor(7).Position; p.Cell != 0x1010 || p.FineX != 128 {
		t.Fatal("replacement changed captured axis rate", p)
	}
	Step(w, nil)
	if _, _, ok := w.ActorFinePosition(7); ok {
		t.Fatal("later native turn/step still claims original mover")
	}
	if err := w.savedMotionFault(); err != nil {
		t.Fatal(err)
	}
	_ = mustMarshal(t, w)
}

func TestSavedMotion1115UnsupportedImportAndMissingCellStayNativeSaveable(t *testing.T) {
	for _, cause := range []string{"elapsed", "packed", "progress", "missing-cell", "occupied-cell", "missing-key"} {
		t.Run(cause, func(t *testing.T) {
			w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
			switch cause {
			case "elapsed":
				binary.LittleEndian.PutUint16(m.Mover[0xac:], 8)
			case "packed":
				m.Position.PackedCell++
			case "progress":
				w.savedOrder(7).Raw[9] = 2
			case "missing-cell":
				cs = cs[:1]
			case "occupied-cell":
				cs[1].Ground.Key = 0xfeed
				binary.LittleEndian.PutUint32(cs[1].Payload[4:], 0xfeed)
			case "missing-key":
				cs[0].Ground = SavedActorSlot{}
				clear(cs[0].Payload[4:8])
			}
			importMotion1115(t, w, m, cs, bs)
			Step(w, nil)
			if len(w.SavedActorMotionIssues()) == 0 {
				t.Fatal("unsupported path has no coverage issue")
			}
			var fresh World
			if err := fresh.UnmarshalBinary(mustMarshal(t, w)); err != nil {
				t.Fatal("unsupported state lost native saveability", err)
			}
			if len(w.savedMotion.Cells) != len(cs) {
				t.Fatal("invented new payload")
			}
		})
	}
}

func TestSavedMotion1115IndependentSlotQueryAndDetachedAtomicImport(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	cs = append(cs, SavedActorCell{Cell: 0x1212, Ground: SavedActorSlot{Key: 0xbeef}})
	binary.LittleEndian.PutUint32(cs[2].Payload[4:], 0xbeef)
	bad := append([]SavedActorCell(nil), cs...)
	bad[2].Ground.Entity = 99
	before := mustMarshal(t, w)
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{m}, bad, bs); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("late invalid slot partly adopted")
	}
	importMotion1115(t, w, m, cs, bs)
	cs[0].Payload[17] = 0
	m.StaticRoute[0] = 0
	got, _, _, _ := w.SavedActorMotions()
	got[0].StaticRoute[0] = 0
	if w.motionFor(7).StaticRoute[0] != 0x1010 || w.motionCell(0x100f).Payload[17] != 0x92 {
		t.Fatal("import/read API aliases mutable input")
	}
	scratch := newRouteScratch(w)
	at, _ := w.cellIndex(18, 18)
	occupants, _ := w.cellLayerOccupants(18, 18)
	if scratch.at(0, at) != 1 || len(occupants) != 0 {
		t.Fatal("unresolved slot was ignored or invented an entity")
	}
	if err := w.HeadlessPlace(7, 10, 10); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := w.ActorFinePosition(7); ok {
		t.Fatal("teleport retained stale original fine Position")
	}
	scratch.occupy(w)
	placed, _ := w.Entity(7)
	old, _ := w.cellIndex(15, 16)
	now, _ := w.cellIndex(placed.X, placed.Y)
	if scratch.at(0, old) != 0 || scratch.at(0, now) != 1 {
		t.Fatal("invalidation retained old ghost occupancy")
	}
	_ = mustMarshal(t, w)
}

func TestSavedMotion1115IndependentWireAndAtomicFaults(t *testing.T) {
	w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
	importMotion1115(t, w, m, cs, bs)
	valid := mustMarshal(t, w)
	// Literal counts + fixed211 + two static and one dynamic words +2cells64
	// +2blocks4 gives365 bytes; no producer size helper participates.
	const span = 12 + 211 + 6 + 128 + 8
	// entityIDFloor (form94) closes the form outside this section entirely.
	end := len(valid) - entityIDFloorLen - spellDeliverySpanLen
	start := end - 44 - span
	rec := start + 12
	firstCell := rec + 211 + 6
	if valid[0] != formatVersion || binary.LittleEndian.Uint32(valid[end-44:]) != span || !bytes.Equal(valid[start:start+12], []byte{1, 0, 0, 0, 2, 0, 0, 0, 2, 0, 0, 0}) || valid[rec+196] != 3 || binary.LittleEndian.Uint16(valid[rec+4:]) != 0x100f || valid[rec+8] != 224 || binary.LittleEndian.Uint32(valid[firstCell+6:]) != 0x1004 || binary.LittleEndian.Uint32(valid[rec+207:]) != 77 {
		t.Fatal("new envelope differs from literal contract")
	}
	for _, tc := range []struct {
		name string
		edit func([]byte)
	}{
		{"span", func(b []byte) { binary.LittleEndian.PutUint32(b[end-44:], math.MaxUint32) }},
		{"motion-count", func(b []byte) { binary.LittleEndian.PutUint32(b[start:], math.MaxUint32) }},
		{"cell-count", func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 65537) }},
		{"block-count", func(b []byte) { binary.LittleEndian.PutUint32(b[start+8:], 65537) }},
		{"flags", func(b []byte) { b[rec+196] = 4 }},
		{"inactive-noncenter", func(b []byte) { b[rec+196] = 1 }},
		{"issue-length", func(b []byte) { binary.LittleEndian.PutUint16(b[rec+197:], 257) }},
		{"route-count", func(b []byte) { binary.LittleEndian.PutUint32(b[rec+199:], math.MaxUint32) }},
		{"missing-entity", func(b []byte) { binary.LittleEndian.PutUint32(b[rec:], 99) }},
		{"position", func(b []byte) { b[rec+4]++ }},
		{"elapsed", func(b []byte) { binary.LittleEndian.PutUint16(b[rec+16+0xac:], 8) }},
		{"direction", func(b []byte) { binary.LittleEndian.PutUint16(b[rec+16+0xae:], 8) }},
		{"late-slot-flag", func(b []byte) { b[firstCell+64+58] = 2 }},
		{"late-unbound-entity", func(b []byte) { binary.LittleEndian.PutUint32(b[firstCell+64+54:], 99) }},
		{"duplicate-cell", func(b []byte) { binary.LittleEndian.PutUint16(b[firstCell+64:], 0x100f) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(valid)
			tc.edit(bad)
			if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
				t.Fatal("invalid native motion partly adopted", err)
			}
		})
	}
	for n := 1; n < span; n++ {
		bad := append(bytes.Clone(valid[:start]), valid[start:start+n]...)
		bad = binary.LittleEndian.AppendUint32(bad, uint32(n))
		bad = append(bad, make([]byte, 24)...)
		if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
			t.Fatalf("truncated motion%d admitted/changed receiver: %v", n, err)
		}
	}
}

func TestSavedMotion1115ExplicitAbsence(t *testing.T) {
	w := mustWorld(t, 1, Bounds{4, 4}, nil)
	before := mustMarshal(t, w)
	if err := w.ImportOriginalActorMotions(nil, nil, nil); err != nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("nil import not absent")
	}
	if err := w.ImportOriginalActorMotions([]SavedActorMotion{}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if w.Hash() == fnv1a(before) {
		t.Fatal("present empty and absent registry hash equally")
	}
}

func TestSavedMotion1115TypedSlotIdentityCannotBeReboundToExistingActor(t *testing.T) {
	makeWorld := func(withSource bool) (*World, []SavedActorMotion, []SavedActorCell) {
		w := mustWorld(t, 1115, Bounds{32, 32}, []Entity{{ID: 0, X: 15, Y: 16, HP: 10, MaxHP: 10}, {ID: 8, X: 17, Y: 16, HP: 10, MaxHP: 10}})
		motions := []SavedActorMotion{{Entity: 0, Position: SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 128, FineY: 128}}, {Entity: 8, Position: SavedActorPosition{Cell: 0x1011, PackedCell: 0x1011, FineX: 128, FineY: 128}}}
		cells := []SavedActorCell{{Cell: 0x100f, Ground: SavedActorSlot{Key: 0x1004, Entity: 0, Bound: true}}, {Cell: 0x1011, Ground: SavedActorSlot{Key: 0x1006, Entity: 8, Bound: true}}}
		for i := range cells {
			binary.LittleEndian.PutUint32(cells[i].Payload[4:], cells[i].Ground.Key)
			if withSource {
				w.entities[i].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: uint16(i + 1), Identity: cells[i].Ground.Key}
				w.entities[i].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
			}
		}
		return w, motions, cells
	}
	for _, source := range []bool{false, true} {
		w, motions, cells := makeWorld(source)
		if err := w.ImportOriginalActorMotions(motions, cells, nil); err != nil {
			t.Fatal(err)
		}
		valid := mustMarshal(t, w)
		// Independently specified two fixed211 motions, no lists/issues.
		// entityIDFloor (form94) closes the form outside this section entirely.
		firstCell := len(valid) - entityIDFloorLen - spellDeliverySpanLen - 44 - (12 + 2*211 + 2*64) + 12 + 2*211
		for _, mutate := range []func([]byte){
			func(b []byte) { binary.LittleEndian.PutUint32(b[firstCell+54:], 8) },
			func(b []byte) { binary.LittleEndian.PutUint32(b[firstCell+64+6:], 0x1004) },
			func(b []byte) { binary.LittleEndian.PutUint32(b[firstCell+64+54:], 0) },
		} {
			bad := bytes.Clone(valid)
			mutate(bad)
			if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(valid, mustMarshal(t, w)) {
				t.Fatal("checksum-valid existing-actor rebinding admitted/partly adopted")
			}
		}
	}
	// A swap could remain bijective; exact nonzero SourceBinding still refuses it.
	w, motions, cells := makeWorld(true)
	cells[0].Ground.Entity, cells[1].Ground.Entity = 8, 0
	before := mustMarshal(t, w)
	if err := w.ImportOriginalActorMotions(motions, cells, nil); err == nil || !bytes.Equal(before, mustMarshal(t, w)) {
		t.Fatal("bijective but wrong original identity import admitted")
	}
}

func TestSavedMotion1115PendingOriginalTurnIsNotSilentCurrentContinuation(t *testing.T) {
	for _, centered := range []bool{false, true} {
		w, m, cs, bs := motionFixture1115(t, 224, 128, 32, 0, 2, 3, 8)
		m.Mover[1], m.Mover[0xa0] = 192, 1
		if centered {
			m.Position.FineX = 128
			clear(m.Mover[0xaa:0xae])
		}
		importMotion1115(t, w, m, cs, bs)
		if centered {
			if w.motionFor(7).Issue != "" || !w.ActorMotionActive(7) {
				t.Fatal("centered pending turn should retain body ownership")
			}
		} else {
			for range 5 {
				Step(w, nil)
			}
			got := w.motionFor(7)
			if got.Position.FineX != 128 || got.Active || got.Issue == "" || got.Mover[1] != 192 || got.Mover[0xa0] != 1 {
				t.Fatal("pending turn silently executed or dropped")
			}
		}
		_ = mustMarshal(t, w)
	}
}
