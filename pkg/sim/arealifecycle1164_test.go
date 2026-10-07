package sim

import (
	"slices"
	"testing"
)

func areaLifecycleWorld(t *testing.T, actors ...Entity) *World {
	t.Helper()
	rules := []SpellRule{
		{ID: 3, Area: true, Distribution: 4, Radius: 2, AreaDuration: 2, Damaging: true, DamageMin: 4, DamageMax: 8, School: 1},
		{ID: 7, Area: true, Distribution: 3, Radius: 2, AreaDuration: 2},
		{ID: 8, Area: true, Distribution: 3, Radius: 2, AreaDuration: 2, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -4, EffectDuration: 128, School: 2},
		{ID: 12, Area: true, Distribution: 3, Radius: 2, AreaDuration: 2, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectMagnitude: 1, EffectDuration: 16},
	}
	w, err := NewStockedSpelledWorld(1164, Bounds{32, 32}, ModeCanonical, Terrain{}, actors, nil, Relations{}, nil, nil, rules)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func landAreaForTest(t *testing.T, w *World, spell uint32, x, y int32, caster EntityID, power uint16) {
	t.Helper()
	rule, ok := w.findSpell(spell)
	if !ok || !w.landArea(rule, power, caster, indexOfEntity(w.entities, caster) >= 0, x, y+1, x, y, nil) {
		t.Fatal("area did not land", spell, x, y)
	}
}

func cold1164(t *testing.T, w *World) *World {
	t.Helper()
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() {
		t.Fatal("native cut lost current state")
	}
	return &cold
}

func TestAreas1164IndependentOwnersNoRevealAndThirdRepaint(t *testing.T) {
	w := areaLifecycleWorld(t)
	landAreaForTest(t, w, 3, 20, 20, 99, 0)
	landAreaForTest(t, w, 3, 20, 20, 99, 0)
	if len(w.effects) != 2 || len(w.effects[0].Cells) != 0 || len(w.effects[1].Cells) != 10 || w.FireWallCount(20, 20) != 1 {
		t.Fatal("overwrite lost clock or retained old paint")
	}
	// Instant29 addresses a covered non-anchor cell; only its current owner is retimed.
	w.setCellEffectTime(19, 20, 3, 1)
	if w.effects[0].Remaining != 32 || w.effects[1].Remaining != 1 {
		t.Fatal("retime did not follow current pointer")
	}
	Step(w, nil)
	back := cold1164(t, w)
	for _, world := range []*World{w, back} {
		Step(world, nil)
	}
	if len(w.effects) != 1 || w.effects[0].Remaining != 30 || len(w.effects[0].Cells) != 0 || w.FireWallCount(20, 20) != 0 {
		t.Fatal("cleanup revealed an older owner")
	}
	for _, world := range []*World{w, back} {
		landAreaForTest(t, world, 3, 20, 20, 99, 0)
	}
	if w.Hash() != back.Hash() {
		t.Fatal("third repaint depends on receiver history")
	}
	for tick := 0; tick < 34; tick++ {
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() {
			t.Fatal("continued clocks diverged", tick)
		}
	}
	if len(w.effects) != 0 || w.FireWallCount(20, 20) != 0 {
		t.Fatal("independent clocks did not finish")
	}
}

func TestAreas1164SquarePresenceOwnPayloadAndCausalOrder(t *testing.T) {
	target := spEnt(1, 22, 19)
	target.HP, target.MaxHP = 1000, 1000
	a, b := spEnt(2, 1, 1), spEnt(3, 2, 1)
	w := areaLifecycleWorld(t, target, a, b)
	// First anchor sorts after the second: source2 must still apply first.
	landAreaForTest(t, w, 8, 20, 21, 2, 30)
	if containsKey(w.effects[0].Cells, cellKey(22, 19)) {
		t.Fatal("corner unexpectedly belongs to first paint")
	}
	landAreaForTest(t, w, 8, 21, 20, 3, 0)
	// Explicit current-cell fixture: second paint supplies the square corner
	// outside the first diamond. Its own scan also contains that cell.
	w.effects[1].Cells = canonicalCells(append(w.effects[1].Cells, cellKey(22, 19)))
	w.effects[0].Remaining, w.effects[1].Remaining = 17, 17
	back := cold1164(t, w)
	for _, world := range []*World{w, back} {
		Step(world, nil)
	}
	if len(w.attached) != 1 || w.attached[0].Caster != 2 || w.attached[0].Magnitude != -8 || w.attached[0].Remaining != 127 || w.entities[0].HP != 984 {
		t.Fatal("square borrow, first source/payload, or area-before-actor phase", w.attached, w.entities[0].HP)
	}
	if w.Hash() != back.Hash() {
		t.Fatal("causal ordering changed on native load")
	}
	// Same-spell gate is necessary; own square alone cannot hit an unpainted cell.
	for i := range w.effects {
		w.effects[i].Cells = nil
		w.effects[i].Remaining = 17
	}
	w.attached = nil
	prior := w.entities[0].HP
	Step(w, nil)
	if w.entities[0].HP != prior || len(w.attached) != 0 {
		t.Fatal("unpainted square pulsed")
	}
}

func TestAreas1164FinalZeroAndFootprintVisits(t *testing.T) {
	for _, spell := range []uint32{3, 8} {
		t.Run(string(rune('0'+spell)), func(t *testing.T) {
			target := spEnt(1, 20, 19)
			target.HP, target.MaxHP, target.TokenSize = 1000, 1000, 2
			w := areaLifecycleWorld(t, target)
			landAreaForTest(t, w, spell, 20, 20, 99, 0)
			w.effects[0].Remaining = 1
			back := cold1164(t, w)
			seed := w.rng.state
			Step(w, nil)
			Step(back, nil)
			if len(w.effects) != 1 || w.effects[0].Remaining != 0 || w.entities[0].HP == 1000 || w.Hash() != back.Hash() {
				t.Fatal("final zero pulse did not survive native cut")
			}
			if spell == 8 && (w.entities[0].HP != 992 || len(w.attached) != 1 || w.attached[0].Remaining != 127) {
				t.Fatal("per-cell refresh stacked magnitude, or same-tick actor pulse absent", w.entities[0].HP, w.attached)
			}
			if spell == 3 && w.rng.state != seed+uint64(target.TokenSize*target.TokenSize)*gamma {
				t.Fatal("Fire must make four direct-damage calls, without footprint division/dedup", w.rng.state-seed)
			}
			Step(w, nil)
			Step(back, nil)
			if len(w.effects) != 0 || w.Hash() != back.Hash() {
				t.Fatal("zero cleanup boundary")
			}
		})
	}
}

func TestAreas1164FirePoisonConflictsPreserveAttachmentsAndClocks(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		w := areaLifecycleWorld(t, spEnt(1, 20, 20))
		poison, _ := w.findSpell(8)
		w.attachEffect(1, 99, poison, EffectHealth, -4, 128, EffectContinuous)
		first, second := uint32(8), uint32(3)
		if reverse {
			first, second = second, first
		}
		landAreaForTest(t, w, first, 20, 20, 99, 0)
		landAreaForTest(t, w, second, 20, 20, 99, 0)
		if len(w.effects) != 2 || len(w.attached) != 1 || w.areaLayerPresent(cellKey(20, 20), 8) || !w.areaLayerPresent(cellKey(20, 20), 3) {
			t.Fatal("Fire conflict revoked independent clock/attachment", reverse)
		}
		back := cold1164(t, w)
		Step(w, nil)
		Step(back, nil)
		if w.Hash() != back.Hash() || w.attached[0].Remaining != 127 {
			t.Fatal("conflict lost attachment continuation")
		}
	}
}

func TestAreas1164MoreThanSixAndMalformedCurrentOwnership(t *testing.T) {
	w := areaLifecycleWorld(t)
	for i := 0; i < 8; i++ {
		landAreaForTest(t, w, 8, 20, 20, 99, 0)
	}
	if len(w.effects) != 8 {
		t.Fatal("six layers were treated as six objects")
	}
	back := cold1164(t, w)
	w.effects[0].Cells = slices.Clone(w.effects[7].Cells)
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("duplicate current owner was saveable")
	}
	prior := back.Hash()
	if err := back.UnmarshalBinary(w.encode()); err == nil || back.Hash() != prior {
		t.Fatal("hostile duplicate owner admitted or changed receiver")
	}
}

func TestAreas1164RestoredBeforeFreshAndUnifiedCurrentCells(t *testing.T) {
	target := spEnt(1, 20, 20)
	target.HP, target.MaxHP = 1000, 1000
	w := areaLifecycleWorld(t, target, spEnt(2, 1, 1))
	w.SetSavedSpellEffects([]SavedSpellEffect{{Class: "AreaEffect", AE48: [4]byte{77, 2}, AE4C: 17, AE44: &SavedEffect{Class: "Effect", E0C: 8, E3C: 6, E3D: 2, E40: 0x0080fffc}}})
	w.SetSavedCellRecords([]SavedCellRecord{{Cell: 0x1414, LayerCount: 1, SpellEffects: [6]uint32{0, 0, 17}}})
	retainedCloudCellOwners1162(t, w)
	// Explicit constructor authority for the surrounding fresh paint.
	for y := 18; y <= 22; y++ {
		for x := 18; x <= 22; x++ {
			key := y*256 + x
			w.savedCellPlanes.Cost[key], w.savedCellPlanes.CostKnown[key] = 8, 1
		}
	}
	if err := w.ImportOriginalWorldEffectDrivers(&SavedWorldEffects{Areas: []SavedAreaDriver{{ID: 1, Root: 0, Identity: 17, Key: 0x1414, Layer: 2, Mode: AreaModeCloud, Spell: 8, Cells: []uint16{0x1414}}}}); err != nil {
		t.Fatal(err)
	}
	landAreaForTest(t, w, 8, 20, 20, 2, 30)
	w.effects[0].Remaining = 17
	if len(w.savedWorldEffects.Areas[0].Cells) != 0 || w.savedSpellEffects[0].AE4C != 17 || w.motionCell(0x1414).Payload[2] != 0 || w.savedCellPlanes.Cost[0x1414] != 32 {
		t.Fatal("overwrite deleted old clock, invented raw pointer, or lost native layer cost")
	}
	back := cold1164(t, w)
	for _, world := range []*World{w, back} {
		Step(world, nil)
	}
	if len(w.attached) != 1 || w.attached[0].HasCaster || w.attached[0].Magnitude != -4 || w.attached[0].Remaining != 127 || w.entities[0].HP != 992 {
		t.Fatal("fresh native area overtook original retained area", w.attached, w.entities[0].HP)
	}
	if w.Hash() != back.Hash() {
		t.Fatal("cross-container order changed at native LOAD")
	}
	for range 17 {
		Step(w, nil)
		Step(back, nil)
	} // both final-zero pulses, then cleanup
	if len(w.effects) != 0 || len(w.savedSpellEffects) != 0 || w.Hash() != back.Hash() {
		t.Fatal("mixed current owners failed to retire")
	}
	if w.motionCell(0x1414) != nil || w.savedCellPlanes.Static[0x1414] != 0 || w.savedCellPlanes.Dynamic[0x1414] != 0x20 {
		t.Fatal("empty native cloud cell tail differs")
	}
}

func TestAreas1164PulseVisitsXBeforeY(t *testing.T) {
	w := areaLifecycleWorld(t)
	w.effects = []cellEffect{{Spell: 3, Mode: AreaModeCloud, Cells: []uint16{0x1314, 0x1413, 0x1414, 0x1514, 0x1415}}}
	// The native input list is deliberately irrelevant to the pulse order.
	want := []uint16{0x1413, 0x1314, 0x1414, 0x1514, 0x1415}
	if got := w.cloudPulseCells(0x1414, 3, 2); !slices.Equal(got, want) {
		t.Fatal("cloud scan is not x-outer/y-inner", got)
	}
}

func areaHeaderDigest1164(current []byte, previous uint64) uint64 {
	old := strippedAttackNoticePin(current)
	old[0] = 91
	if fnv1a(old) != previous {
		panic("form91 frozen predecessor digest changed")
	}
	old[0] = 92
	return fnv1a(widenedAttackNoticePin(old))
}
