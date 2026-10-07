package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAreaSaveCurrentPayloadAndIndependentOwners(t *testing.T) {
	w := areaLifecycleWorld(t)
	landAreaForTest(t, w, 3, 20, 20, 99, 30)
	landAreaForTest(t, w, 3, 20, 20, 99, 0)
	landAreaForTest(t, w, 8, 10, 10, 99, 30)
	hash := w.Hash()
	rows, err := w.NativeAreaSaveStates()
	if err != nil || len(rows) != 3 || w.Hash() != hash {
		t.Fatal("projection mutated or dropped an area", rows, err)
	}
	if len(rows[0].Cells) != 0 || len(rows[1].Cells) != 10 || rows[0].Remaining != 80 || rows[1].Remaining != 32 {
		t.Fatal("overwritten independent clocks", rows)
	}
	for i, want := range [][3]byte{{8, 8, 1}, {4, 4, 1}} {
		p := rows[i].Payload
		if p.Class != "Effect_DirectDamage" || p.DirectDamage[19] != want[0] || p.DirectDamage[20] != want[1] || p.DirectDamage[21] != want[2] {
			t.Fatal("frozen base/spread/school", p)
		}
	}
	if p := rows[2].Payload; p.Class != "Effect" || p.E3C != 6 || p.E3D != 2 || p.E40 != 0x0080fff8 {
		t.Fatal("Poison signed magnitude and remaining lifetime", p)
	}
	rows[1].Cells[0]++
	if w.Hash() != hash {
		t.Fatal("projection aliases current cell ownership")
	}
}

func TestAreaSaveRingCutsContinueWithoutReplay(t *testing.T) {
	for _, spell := range []uint16{4, 9, 21} {
		stages := map[uint16]int{4: 2, 9: 6, 21: 32}[spell]
		for age := 0; age < 3*(stages-1); age++ {
			t.Run(fmt.Sprintf("%d/%d", spell, age), func(t *testing.T) {
				rule := SpellRule{ID: spell, Area: true, Distribution: 5, Damaging: true, School: 1, DamageMin: 4, DamageMax: 8}
				makeWorld := func() *World {
					w, err := NewStockedSpelledWorld(17, Bounds{32, 32}, ModeCanonical, Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{rule})
					if err != nil {
						t.Fatal(err)
					}
					return w
				}
				w := makeWorld()
				w.effects = []cellEffect{{Key: 20 | 20<<8, Spell: spell, Remaining: uint16(3*(stages-1) + 1 - age), Mode: areaModeRing, Phase: uint8(age), Direction: 3}}
				rows, err := w.NativeAreaSaveStates()
				if err != nil || len(rows) != 1 {
					t.Fatal(rows, err)
				}
				r := rows[0]
				if r.Stage != uint8(age/3+1) || r.Remaining != uint16(2-age%3) || r.Direction != 3 {
					t.Fatal("next stage/countdown", r)
				}
				cold := makeWorld()
				cold.savedSpellEffects = []SavedSpellEffect{{Class: "AreaEffect", AE48: [4]byte{1, 0, 96, r.Stage}, AE4C: r.Remaining, AE44: &r.Payload}}
				cold.savedWorldEffects = &SavedWorldEffects{Areas: []SavedAreaDriver{{ID: 1, Root: 0, Identity: 17, Key: r.Key, Layer: 255, Mode: areaModeRing, Spell: spell}}}
				for tick := age + 1; tick <= 3*(stages-1); tick++ {
					w.decayCellEffects(nil)
					cold.stepSavedWorldEffects(nil)
					if (len(w.effects) == 0) != (len(cold.savedSpellEffects) == 0) {
						t.Fatal("retirement cut", tick)
					}
					if len(w.effects) > 0 && cold.savedSpellEffects[0].AE48[3] != w.effects[0].Phase/3+1 {
						t.Fatal("stage repeated or skipped", tick)
					}
				}
			})
		}
	}
}

func TestSavedAreaDamageUsesFrozenBlock(t *testing.T) {
	w := areaLifecycleWorld(t, Entity{ID: 1, X: 20, Y: 20, HP: 100, MaxHP: 100, Protection: [5]int32{25}})
	p := &SavedEffect{Class: "Effect_DirectDamage", E0C: 3}
	p.DirectDamage[19], p.DirectDamage[21] = 4, 1
	before := append([]SpellRule(nil), w.spells...)
	w.applySavedAreaPayload(SavedAreaDriver{Key: 20 | 20<<8, Mode: areaModeCloud, Spell: 3}, SavedSpellEffect{AE44: p}, []uint16{20 | 20<<8})
	if w.entities[0].HP != 97 || !reflect.DeepEqual(before, w.spells) {
		t.Fatal("frozen block damage or rule mutation", w.entities[0].HP)
	}
	p.DirectDamage[0] = 1
	if SavedAreaPayloadSupported(p) {
		t.Fatal("unbound physical operand admitted")
	}
}

func TestSacrificeAreaSavesConstructedDamage(t *testing.T) {
	rule := SpellRule{ID: 4, Area: true, Distribution: 5, School: 1}
	w, err := NewStockedSpelledWorld(17, Bounds{32, 32}, ModeCanonical, Terrain{}, nil, nil, Relations{}, nil, nil, []SpellRule{rule})
	if err != nil {
		t.Fatal(err)
	}
	w.effects = []cellEffect{{Key: 0x1414, Spell: 4, Mode: areaModeRing, Power: 100, DamageMin: 255, DamageMax: 510}}
	rows, err := w.NativeAreaSaveStates()
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	p := rows[0].Payload
	if p.Class != "Effect_DirectDamage" || p.DirectDamage[19] != 255 || p.DirectDamage[20] != 255 || p.DirectDamage[21] != 1 {
		t.Fatal("current sacrifice damage was replaced by its empty table row", p)
	}
}

func TestSavedAreaDamageReachesStructureReferences(t *testing.T) {
	w := structureAreaWorld(t, []Structure{{ID: 1, Col: 10, Row: 10, Width: 2, Height: 1, Attach: 3, Field42: 100, MaxHealth: 100}})
	w.spells = []SpellRule{{ID: 9, Area: true}}
	p := &SavedEffect{Class: "Effect_DirectDamage", E0C: 9}
	p.DirectDamage[19], p.DirectDamage[20], p.DirectDamage[21] = 10, 1, 1
	w.applySavedAreaPayload(SavedAreaDriver{Key: 0x0a0a, Mode: areaModeRing, Spell: 9}, SavedSpellEffect{AE44: p}, []uint16{0x0a0a, 0x0a0b})
	if w.structures[0].Field42 != 89 {
		t.Fatal("retained ring did not apply damage per structure reference", w.structures[0].Field42)
	}
}
