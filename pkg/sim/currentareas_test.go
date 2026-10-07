package sim

import (
	"reflect"
	"slices"
	"testing"
)

func currentAreaOrdinaryWorld(t *testing.T, native *World, mutate func(*SavedSpellEffect), installedDuration ...uint16) *World {
	t.Helper()
	rows, err := native.NativeAreaSaveStates()
	if err != nil || len(rows) != 1 {
		t.Fatal("area fixture", rows, err)
	}
	cold := new(World)
	if err := cold.UnmarshalBinary(mustMarshal(t, native)); err != nil {
		t.Fatal(err)
	}
	p := native.CurrentPolicy()
	if len(installedDuration) != 0 {
		for i := range cold.spells {
			if cold.spells[i].ID == 8 {
				cold.spells[i].EffectDuration = installedDuration[0]
			}
		}
		p.KeepSpellDeviations(cold.Spells())
		if err := cold.RestoreCurrentSpellDurations(p.SpellPolicies); err != nil {
			t.Fatal(err)
		}
	}
	r := rows[0]
	v := SavedSpellEffect{Class: "AreaEffect", AE48: [4]byte{1, r.Radius, r.Direction << 5, r.Stage}, AE4C: r.Remaining, AE44: &r.Payload}
	if r.Stopped {
		v.SE40 = 1
	}
	if mutate != nil {
		mutate(&v)
	}
	layer := uint8(255)
	if r.Mode == areaModeCloud {
		layer = 0
	}
	cold.effects, cold.effectOrder = nil, nil
	cold.savedSpellGraph = &SavedSpellGraph{Nodes: []SavedSpellNode{{Value: v, Spell: r.Spell}}, Roots: []uint32{1}}
	cold.savedWorldEffects = &SavedWorldEffects{Areas: []SavedAreaDriver{{ID: 1, Root: 0, Identity: 17, Key: r.Key, Layer: layer, Mode: r.Mode, Spell: r.Spell, Cells: slices.Clone(r.Cells)}}}
	cold.refreshSavedSpellGraph()
	if err := cold.RestoreCurrentAreas([]CurrentAreaRestore{{Node: 1, Policy: r.Policy}}); err != nil {
		t.Fatal(err)
	}
	if len(cold.savedSpellGraph.Nodes) != 0 || len(cold.savedWorldEffects.Areas) != 0 {
		t.Fatal("native ownership left transient history")
	}
	if err := cold.RestoreCurrentContinuation(&p, nil, cold.Actions(), nil); err != nil {
		t.Fatal(err)
	}
	return cold
}

func TestCurrentPoisonDurationPolicyPrecedesAreaDerivation(t *testing.T) {
	rule := SpellRule{ID: 8, Area: true, Radius: 2, EffectKind: EffectHealth,
		EffectMode: EffectContinuous, EffectMagnitude: -2, EffectDuration: 8}
	native := spWorld(t, 77, []SpellRule{rule}, spEnt(1, 10, 10))
	native.effects = []cellEffect{{Key: 0x0a0a, Spell: 8, Mode: areaModeCloud,
		Power: 30, Remaining: 18, Phase: 77, Cells: native.diamondCells(10, 10, 2)}}
	rows, err := native.NativeAreaSaveStates()
	if err != nil || len(rows) != 1 || !rows[0].Policy.DerivedPayload || rows[0].Payload.E40 != 0x0008fffc {
		t.Fatal("source Poison constructor", rows, err)
	}
	cold := currentAreaOrdinaryWorld(t, native, nil, 128)
	if cold.Hash() != native.Hash() || cold.effects[0].Current != nil {
		t.Fatal("cold Poison area became explicit or changed state")
	}
	again := currentAreaOrdinaryWorld(t, cold, nil, 128)
	if again.Hash() != native.Hash() {
		t.Fatal("second Poison cycle changed derived state")
	}
	edited := currentAreaOrdinaryWorld(t, native, func(v *SavedSpellEffect) {
		v.AE44.E40 = 0x0008fffd
	}, 128)
	if edited.effects[0].Current == nil || edited.effects[0].Current.Payload.E40 != 0x0008fffd {
		t.Fatal("edited ordinary AE44 was replaced by derived payload")
	}
	for tick := 0; tick < 18; tick++ {
		native.stepWorldSpellEffects(nil)
		cold.stepWorldSpellEffects(nil)
		again.stepWorldSpellEffects(nil)
		if native.Hash() != cold.Hash() || cold.Hash() != again.Hash() {
			t.Fatal("Poison pulse changed after current policy restore", tick)
		}
	}
}

func TestCurrentAreaOrdinaryPayloadContinuesPulseExpiryAndRing(t *testing.T) {
	for _, spell := range []uint16{3, 8, 9} {
		rule := SpellRule{ID: spell, Area: true, Radius: 2, Damaging: true, DamageMin: 4, DamageMax: 8, School: 1}
		if spell == 8 {
			rule.Damaging, rule.EffectKind, rule.EffectMode, rule.EffectMagnitude, rule.EffectDuration = false, EffectHealth, EffectContinuous, -4, 128
		}
		w := spWorld(t, 77, []SpellRule{rule}, spEnt(1, 10, 10))
		e := cellEffect{Key: 0x0a0a, Spell: spell, Mode: areaModeCloud, Power: 30, Remaining: 18, Phase: 77, Cells: w.diamondCells(10, 10, 2)}
		if spell == 9 {
			e.Mode, e.Phase, e.Remaining = areaModeRing, 2, 14
			e.Cells = canonicalCells(w.ringStageCells(e, 0))
		}
		w.effects = []cellEffect{e}
		cold := currentAreaOrdinaryWorld(t, w, nil)
		if cold.Hash() != w.Hash() {
			t.Fatal("ordinary area changed native constructor presence")
		}
		cold = currentAreaOrdinaryWorld(t, cold, nil)
		if cold.Hash() != w.Hash() {
			t.Fatal("second area cycle changed current state")
		}
		for tick := 0; tick < 21; tick++ {
			// The occupant enters and leaves while the retained timer runs.
			for _, world := range []*World{w, cold} {
				world.entities[0].X = int32(10 + tick%3)
				world.stepWorldSpellEffects(nil)
			}
			if !reflect.DeepEqual(w.Entities(), cold.Entities()) || !reflect.DeepEqual(w.ActiveEffects(), cold.ActiveEffects()) || w.RandomState() != cold.RandomState() {
				t.Fatalf("spell %d pulse %d differs", spell, tick)
			}
			want, err := w.NativeAreaSaveStates()
			if err != nil {
				t.Fatal(err)
			}
			got, err := cold.NativeAreaSaveStates()
			if err != nil || !reflect.DeepEqual(want, got) {
				t.Fatalf("spell %d timer/stage %d differs: %v", spell, tick, err)
			}
		}
		if len(w.effects) != 0 {
			t.Fatal("expiry was not exercised")
		}
	}
}

func TestCurrentAreaKeepsEditedOrdinaryPayloadAndUnrelatedHistory(t *testing.T) {
	w := spWorld(t, 7, []SpellRule{{ID: 3, Area: true, Radius: 2, Damaging: true, DamageMin: 4, DamageMax: 4, School: 1}}, spEnt(1, 10, 10))
	w.effects = []cellEffect{{Key: 0x0a0a, Spell: 3, Mode: areaModeCloud, Remaining: 17, Cells: []uint16{0x0a0a}}}
	cold := currentAreaOrdinaryWorld(t, w, func(v *SavedSpellEffect) { v.AE44.DirectDamage[19] = 19 })
	w.stepWorldSpellEffects(nil)
	cold.stepWorldSpellEffects(nil)
	if w.entities[0].HP != 96 || cold.entities[0].HP != 81 {
		t.Fatal("policy overrode edited ordinary damage", w.entities[0].HP, cold.entities[0].HP)
	}
	cold = transportGraphWorld(t, 3)
	g := cold.SavedSpellGraph()
	area := SavedSpellNode{Value: SavedSpellEffect{Class: "AreaEffect", AE48: [4]byte{1, 0, 0, 1}, AE4C: 2, AE44: &SavedEffect{Class: "Effect_DirectDamage", E0C: 9}}, Spell: 9}
	g.Nodes = append(g.Nodes, area, SavedSpellNode{Value: SavedSpellEffect{Class: "AreaEffect", AE44: area.Value.AE44}, Spell: 9, Retired: true})
	g.Roots = []uint32{1, 3, 1}
	cold.savedSpellGraph = g
	cold.savedWorldEffects.Areas = []SavedAreaDriver{{ID: 3, Root: 1, Identity: 17, Key: 0x0505, Layer: 255, Mode: areaModeRing, Spell: 9}, {ID: 4, Root: -1, Identity: 18, Key: 0x0505, Layer: 255, Mode: areaModeRing, Spell: 9}}
	cold.refreshSavedSpellGraph()
	row := CurrentAreaRestore{Node: 3, Policy: CurrentAreaPolicy{RingLife: 16}}
	before := cold.Hash()
	if err := cold.RestoreCurrentAreas([]CurrentAreaRestore{row, row}); err == nil || cold.Hash() != before {
		t.Fatal("bad conversion was not atomic", err)
	}
	if err := cold.RestoreCurrentAreas([]CurrentAreaRestore{row}); err != nil {
		t.Fatal(err)
	}
	g = cold.SavedSpellGraph()
	if len(g.Nodes) != 3 || !g.Nodes[2].Retired || !slices.Equal(g.Roots, []uint32{1, 1}) || g.Nodes[0].Primary != 2 || len(cold.savedWorldEffects.Areas) != 1 || cold.savedWorldEffects.Areas[0].ID != 3 {
		t.Fatal("conversion erased existing history or changed shared transport", g, cold.savedWorldEffects)
	}
	if !slices.Equal(cold.CurrentWorldEffectOrder(), []WorldEffectRef{{EffectSavedGraph, 1}, {EffectNativeArea, 0}, {EffectSavedGraph, 1}}) {
		t.Fatal("ownership transfer changed list order")
	}
	requireSpellGraphBinary(t, cold)
}

func TestCurrentAreaPayloadAbsenceYieldsToOrdinaryZeroAndRadius(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*SavedSpellEffect)
	}{
		{"zero damage", func(v *SavedSpellEffect) { v.AE44.DirectDamage[19] = 0 }},
		{"radius", func(v *SavedSpellEffect) { v.AE48[1] = 0 }},
		{"stopped", func(v *SavedSpellEffect) { v.SE40 = 1 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := spWorld(t, 7, []SpellRule{{ID: 3, Area: true, Radius: 2, Damaging: true, DamageMin: 4, DamageMax: 4, School: 1}}, spEnt(1, 10, 10))
			w.effects = []cellEffect{{Key: 0x0a0a, Spell: 3, Mode: areaModeCloud, Remaining: 17, Cells: []uint16{0x0a0a}}}
			cold := currentAreaOrdinaryWorld(t, w, tc.edit)
			if cold.effects[0].Current == nil {
				t.Fatal("absence policy discarded an ordinary edit")
			}
			next := currentAreaOrdinaryWorld(t, cold, nil)
			if next.Hash() != cold.Hash() {
				t.Fatal("edited ordinary payload changed on second cycle")
			}
		})
	}
}
