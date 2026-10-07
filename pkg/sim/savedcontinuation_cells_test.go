package sim

import (
	"bytes"
	"encoding/binary"
	"slices"
	"strings"
	"testing"
)

// This supplies explicit baseline owners for synthetic retained-cloud cells.
// Neither a residual layer record nor a current blocked plane proves a baseline.
func retainedCloudCellOwners1162(t *testing.T, w *World) {
	t.Helper()
	cells := make([]SavedActorCell, len(w.savedCellRecords))
	p := &SavedCellPlanes{}
	p.Costs[0], p.Costs[5] = 255, 8
	for i, r := range w.savedCellRecords {
		c := &cells[i]
		c.Cell, c.Payload[0], c.Payload[2] = r.Cell, 8, r.LayerCount
		for layer, identity := range r.SpellEffects {
			binary.LittleEndian.PutUint32(c.Payload[20+4*layer:], identity)
		}
		p.Static[r.Cell], p.Dynamic[r.Cell] = 0x20, 0x20
	}
	if err := w.ImportOriginalActorMotions(nil, cells, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalCellPlanes(p); err != nil {
		t.Fatal(err)
	}
	for _, c := range cells {
		if issue := w.recomputeSavedCell(c.Cell); issue != "" {
			t.Fatal(issue)
		}
	}
	w.refreshSavedPlaneBlocks()
}

func retainedWallCells1162(t *testing.T, armed bool) *World {
	t.Helper()
	w := mustWorld(t, 1162, Bounds{32, 32}, nil)
	w.SetSavedCellRecords([]SavedCellRecord{
		{Cell: 0x1010, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 17}},
		{Cell: 0x1011, LayerCount: 2, SpellEffects: [6]uint32{0, 99, 0, 17}},
		{Cell: 0x1012, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 17}}, // outside radius1
		{Cell: 0x1110, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 17}},
		{Cell: 0x1111, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 0, 88}}, // another wall
	})
	retainedCloudCellOwners1162(t, w)
	c := w.motionCell(0x1011)
	c.Payload[0], c.Payload[1], c.Payload[3], c.Payload[50], c.Payload[51] = 9, 2, 0xab, 0x12, 0x34
	c.Ground, c.Air = SavedActorSlot{Key: 0x1234}, SavedActorSlot{Key: 0x5678}
	binary.LittleEndian.PutUint32(c.Payload[4:], c.Ground.Key)
	binary.LittleEndian.PutUint32(c.Payload[8:], c.Air.Key)
	binary.LittleEndian.PutUint32(c.Payload[16:], 0x9876)
	r := &w.savedCellRecords[1]
	r.Ground.Key, r.Air.Key, r.Sack = c.Ground.Key, c.Air.Key, 0x9876
	r.Residue0, r.Residue1 = 0xab, [2]byte{0x12, 0x34}
	w.savedCellPlanes.Static[c.Cell] |= 0x10
	if issue := w.recomputeSavedCell(c.Cell); issue != "" {
		t.Fatal(issue)
	}
	w.motionCell(0x1110).Payload[1] = 1 // terrain itself blocks after wall removal
	w.recomputeSavedCell(0x1110)
	w.refreshSavedPlaneBlocks()
	w.SetSavedSpellEffects([]SavedSpellEffect{{Class: "AreaEffect", AE48: [4]byte{1, 1}, AE4C: 1, AE44: &SavedEffect{Class: "Effect"}}})
	if armed {
		if err := w.ImportOriginalWorldEffectDrivers(retainedWallDriver1162()); err != nil {
			t.Fatal(err)
		}
	}
	return w
}

func retainedWallDriver1162() *SavedWorldEffects {
	return &SavedWorldEffects{Areas: []SavedAreaDriver{{ID: 1, Root: 0, Identity: 17, Key: 0x1010, Layer: 3, Mode: AreaModeCloud, Spell: 19, Cells: []uint16{0x1010, 0x1011, 0x1012, 0x1110}}}}
}

func TestRetainedCloud1162RetirementOwnsPayloadPlanesAndRouting(t *testing.T) {
	w := retainedWallCells1162(t, true)
	before := slices.Clone(w.savedMotion.Cells)
	if w.terrainOpen(DomainGround, 16, 16) || w.savedCellPlanes.Static[0x1010] != 0x25 {
		t.Fatal("controlled loaded wall must block before expiry")
	}
	Step(w, nil)
	if len(w.savedSpellEffects) != 1 || w.savedSpellEffects[0].AE4C != 0 || w.terrainOpen(DomainGround, 16, 16) {
		t.Fatal("zero cut must retain the root and its wall")
	}
	Step(w, nil)
	check := func(world *World) {
		t.Helper()
		if len(world.savedSpellEffects) != 0 || world.savedWorldEffects != nil {
			t.Fatal("retired wall carrier remained in the world")
		}
		for _, original := range before {
			want := original
			if original.Cell == 0x1010 || original.Cell == 0x1011 || original.Cell == 0x1110 {
				clear(want.Payload[32:36])
				want.Payload[2]--
			}
			current := world.motionCell(original.Cell)
			if original.Cell == 0x1010 || original.Cell == 0x1110 {
				if current != nil {
					t.Fatalf("empty cloud node %04x was not unlinked", original.Cell)
				}
				for _, r := range world.savedCellRecords {
					if r.Cell == original.Cell {
						t.Fatal("retired empty residual remains")
					}
				}
				continue
			}
			if current == nil || *current != want {
				t.Fatalf("unrelated payload changed at%04x", original.Cell)
			}
			for _, r := range world.savedCellRecords {
				if r.Cell == original.Cell && (r.SpellEffects[3] != binary.LittleEndian.Uint32(want.Payload[32:]) || r.LayerCount != want.Payload[2]) {
					t.Fatal("residual differs")
				}
			}
		}
		for _, want := range []struct {
			key             uint16
			cost, stat, dyn byte
		}{
			{0x1010, 8, 0, 0x20}, {0x1011, 36, 0x32, 0xf2},
			{0x1012, 32, 0x25, 0x25}, {0x1110, 8, 1, 0x21}, {0x1111, 32, 0x25, 0x25},
		} {
			p := world.savedCellPlanes
			if p.Cost[want.key] != want.cost || p.Static[want.key] != want.stat || p.Dynamic[want.key] != want.dyn {
				t.Fatalf("cell%04x planes got %02x/%02x/%02x want %02x/%02x/%02x", want.key, p.Cost[want.key], p.Static[want.key], p.Dynamic[want.key], want.cost, want.stat, want.dyn)
			}
		}
		if !world.terrainOpen(DomainGround, 16, 16) || !world.terrainOpen(DomainGhost, 16, 16) || world.terrainOpen(DomainGround, 16, 17) || world.terrainOpen(DomainGround, 17, 17) || world.terrainOpen(DomainGround, 18, 16) {
			t.Fatal("routing lost the distinction between cleared wall, terrain, another identity and outside radius")
		}
	}
	check(w)
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	check(&cold)
	Step(w, nil)
	Step(&cold, nil)
	check(&cold)
	if w.Hash() != cold.Hash() {
		t.Fatal("next step after retired-cut LOAD differs")
	}
}

func TestRetainedCloud1162MissingOwnersAndCorruptionAreAtomic(t *testing.T) {
	for _, tc := range []struct {
		name, issue string
		corrupt     func(*World)
	}{
		{"planes", "plane authority", func(w *World) { w.savedCellPlanes = nil }},
		{"motion", "current Cell", func(w *World) { w.savedMotion = nil }},
		{"node", "current Cell", func(w *World) { w.savedMotion.Cells = w.savedMotion.Cells[1:] }},
		{"residual", "retained world-effect", func(w *World) { w.savedCellRecords = w.savedCellRecords[1:] }},
		{"layer", "layer payload", func(w *World) { w.motionCell(0x1010).Payload[32]++ }},
		{"count", "layer count", func(w *World) { w.motionCell(0x1010).Payload[2]++ }},
		{"building", "unresolved Building", func(w *World) { binary.LittleEndian.PutUint32(w.motionCell(0x1011).Payload[12:], 0xbeef) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := retainedWallCells1162(t, false)
			tc.corrupt(w)
			before := w.Hash()
			err := w.ImportOriginalWorldEffectDrivers(retainedWallDriver1162())
			if err == nil || !strings.Contains(err.Error(), tc.issue) || w.Hash() != before {
				t.Fatalf("bad owner admission was not explicit/atomic: %v", err)
			}
			// Encoding a deliberately invalid in-memory driver is a hostile
			// wire control, not an ordinary SAVE. Unmarshal must be atomic too.
			w.savedWorldEffects = retainedWallDriver1162()
			bad := w.encode()
			if _, err := w.MarshalBinary(); err == nil {
				t.Fatal("invalid current owners were saveable")
			}
			receiver := retainedWallCells1162(t, true)
			old := receiver.Hash()
			if err := receiver.UnmarshalBinary(bad); err == nil || receiver.Hash() != old {
				t.Fatal("invalid native owners admitted or receiver changed", err)
			}
			// A late missing owner must not clear one cell before discovering
			// another is unusable, nor retire the root independently of cells.
			prior := bytes.Clone(w.encode())
			w.retireSavedArea(0)
			if !bytes.Equal(prior, w.encode()) {
				t.Fatal("invalid current cleanup published a partial retirement")
			}
		})
	}
}
