package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func TestSavedAreaPayloadMatchesCurrentApplication(t *testing.T) {
	rules := []SpellRule{
		{ID: 3, Area: true, Damaging: true, School: 1, DamageMin: 4, DamageMax: 8},
		{ID: 4, Area: true, Damaging: true, School: 1, DamageMin: 4, DamageMax: 8},
		{ID: 7, Area: true, EffectKind: EffectSpeed, EffectMode: EffectDuration, EffectDuration: 16},
		{ID: 8, Area: true, School: 2, EffectKind: EffectHealth, EffectMode: EffectContinuous, EffectMagnitude: -4, EffectDuration: 128},
		{ID: 9, Area: true, Damaging: true, School: 2, DamageMin: 4, DamageMax: 8},
		{ID: 12, Area: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 16},
		{ID: 17, Area: true, EffectKind: EffectScanRange, EffectMode: EffectDuration, EffectDuration: 16},
		{ID: 19, Area: true},
		{ID: 21, Area: true, Damaging: true, School: 4, DamageMin: 4, DamageMax: 8},
		{ID: 12, Area: true, EffectKind: EffectScanRange},
	}
	for _, rule := range rules {
		for _, power := range []uint16{0, 30, 100} {
			t.Run(fmt.Sprintf("%d/mode%d/power%d", rule.ID, rule.EffectMode, power), func(t *testing.T) {
				makeWorld := func() *World {
					e := spEnt(1, 10, 10)
					e.Speed, e.ScanRange = 10, 8
					return spWorld(t, 77, []SpellRule{rule}, e)
				}
				native, loaded := makeWorld(), makeWorld()
				e := cellEffect{Key: 0x0a0a, Spell: rule.ID, Mode: areaModeCloud, Power: power, Remaining: 32, Cells: []uint16{0x0a0a}}
				if rule.ID == 4 || rule.ID == 9 || rule.ID == 21 {
					e.Mode = areaModeRing
				}
				native.effects = []cellEffect{e}
				rows, err := native.NativeAreaSaveStates()
				if err != nil {
					t.Fatal(err)
				}
				native.applyAreaCells(e, rule, []uint16{0x0a0a})
				loaded.applySavedAreaPayload(SavedAreaDriver{Key: e.Key, Spell: e.Spell, Mode: e.Mode}, SavedSpellEffect{AE44: &rows[0].Payload}, []uint16{0x0a0a})
				n, l := reflect.ValueOf(native.Entities()[0]), reflect.ValueOf(loaded.Entities()[0])
				for i := 0; i < n.NumField(); i++ {
					if n.Field(i).CanInterface() && !reflect.DeepEqual(n.Field(i).Interface(), l.Field(i).Interface()) {
						t.Errorf("actor field %s native=%v loaded=%v", n.Type().Field(i).Name, n.Field(i).Interface(), l.Field(i).Interface())
					}
				}
				if !reflect.DeepEqual(native.ActiveEffects(), loaded.ActiveEffects()) || native.RandomState() != loaded.RandomState() {
					t.Errorf("attachments native=%+v loaded=%+v; RNG native=%d loaded=%d", native.ActiveEffects(), loaded.ActiveEffects(), native.RandomState(), loaded.RandomState())
				}
			})
		}
	}
}

func TestAttachedContinuousModeBitsResume(t *testing.T) {
	for _, mode := range []EffectMode{2, 3, 6, 7} {
		t.Run(fmt.Sprint(mode), func(t *testing.T) {
			w := originalAttachmentWorld(t)
			w.entities[0].Protection[1] = 0
			e := ActiveEffect{Target: 1, Spell: 8, Kind: EffectHealth, Mode: mode, Magnitude: -4, Remaining: 8}
			if err := w.ImportOriginalAttachedEffects([]ActiveEffect{e}); err != nil {
				t.Fatal(err)
			}
			before := w.entities[0].HP
			w.stepAttachedEffects()
			if w.entities[0].HP != before-4 {
				t.Errorf("continuous bit set: HP after pulse=%d want=%d, remaining=%d", w.entities[0].HP, before-4, w.attached[0].Remaining)
			}
			w.attached[0].Remaining = 1
			before = w.entities[0].HP
			w.stepAttachedEffects()
			if w.entities[0].HP != before {
				t.Errorf("continuous expiry reversed damage: HP=%d want=%d", w.entities[0].HP, before)
			}
		})
	}
}
