package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func TestAutoHealing1191CommandPercentagesAndSourceFloor(t *testing.T) {
	w := &World{entities: []Entity{
		{Owner: SelfSlot, MaxMana: 101, ActorLoad: ActorLoad{Source: SourceActor{Class: 1, HasOwner: true}}},
		{Owner: SelfSlot + 1, MaxMana: 71, ActorLoad: ActorLoad{Source: SourceActor{Class: 1, HasOwner: true, ManaFloor: 43, ManaReservePercent: 61}}},
	}}
	foreign := w.entities[1]
	for _, tc := range []struct {
		value   int32
		percent uint32
		floor   uint16
	}{
		{0, 100, 101}, {1, 50, 50}, {2, 0, 0}, {3, 3, 3}, {37, 37, 37}, {100, 100, 101}, {-1, 100, 101}, {101, 100, 101},
	} {
		w.applyPlayerParameter(SelfSlot, PlayerParameterAutoHealing, tc.value)
		p, present := w.AutoHealing(SelfSlot)
		s := w.entities[0].ActorLoad.Source
		if !present || p != tc.percent || s.ManaReservePercent != tc.percent || s.ManaFloor != tc.floor || !reflect.DeepEqual(w.entities[1], foreign) {
			t.Fatalf("mode%d: policy %d/%v, source %+v", tc.value, p, present, s)
		}
	}
	w.entities[0].MaxMana = -3
	if !w.SetAutoHealing(SelfSlot, 1) || int16(w.entities[0].ActorLoad.Source.ManaFloor) != -1 {
		t.Fatal("signed maximum/truncation changed")
	}
	w.entities[0].ActorLoad.Source.ManaFloor = 99
	if !w.SetAutoHealing(SelfSlot, -1) || int16(w.entities[0].ActorLoad.Source.ManaFloor) != -1 {
		t.Fatal("invalid value did not retain percentage and rederive floor")
	}
	if w.SetAutoHealing(relationSlots, 0) || w.ImportAutoHealing(relationSlots, 50) {
		t.Fatal("accepted player outside roster")
	}
}

func TestAutoHealing1191MalformedPolicyTablesAreAtomic(t *testing.T) {
	w := hlWorld(t, 17, acEnemies(t), []SpellRule{hlHeal()}, acCaster(1, 2, 2, 51, 1<<6, 0))
	w.ImportAutoHealing(0, 95)
	w.ImportAutoHealing(49, 0xffffffff)
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	start := len(raw) - 17 // two five-byte rows, count, span and magic
	before := w.Hash()
	for _, tc := range []struct {
		name   string
		change func([]byte)
	}{
		{"magic", func(b []byte) { b[len(b)-1] = 0 }},
		{"span", func(b []byte) { binary.LittleEndian.PutUint16(b[len(b)-6:], 65535) }},
		{"zero count", func(b []byte) { b[start] = 0 }},
		{"wide count", func(b []byte) { b[start] = 51 }},
		{"duplicate slot", func(b []byte) { b[start+6] = b[start+1] }},
		{"wide slot", func(b []byte) { b[start+6] = 50 }},
		{"reverse order", func(b []byte) { b[start+1], b[start+6] = 49, 0 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(raw)
			tc.change(bad)
			if err := w.UnmarshalBinary(bad); err == nil || w.Hash() != before {
				t.Fatal("malformed policy admitted or changed receiver", err)
			}
		})
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != before {
		t.Fatal("raw unsigned policy or boundary slots lost", err)
	}
}

func TestAutoHealing1191StrictThresholdAndExplicitAutocast(t *testing.T) {
	for _, tc := range []struct {
		mode, mana int32
		auto       uint16
		allowed    bool
	}{
		{0, 100, 0, false}, {1, 50, 0, false}, {1, 51, 0, true}, {2, 0, 0, false}, {2, 1, 0, true}, {0, 51, 6, true},
	} {
		caster := acCaster(1, 2, 2, tc.mana, 1<<6, tc.auto)
		caster.Owner, caster.MaxMana = SelfSlot, 100
		w := hlWorld(t, 17, acEnemies(t), []SpellRule{hlHeal()}, caster)
		if !w.SetAutoHealing(SelfSlot, tc.mode) {
			t.Fatal("setting refused")
		}
		if got := len(w.autoCastOrder(0)) != 0; got != tc.allowed {
			t.Fatalf("mode%d mana%d armed%d: allowed=%v", tc.mode, tc.mana, tc.auto, got)
		}
	}
	caster := acCaster(1, 2, 2, 40, 1<<6, 0)
	caster.Owner, caster.MaxMana = SelfSlot, 100
	caster.ActorLoad.Source = SourceActor{Class: 1, HasSpellbook: true, ManaFloor: 39}
	w := &World{entities: []Entity{caster}}
	if !w.ImportAutoHealing(SelfSlot, 95) || !w.globalHealAllowed(0) || w.entities[0].ActorLoad.Source.ManaFloor != 39 {
		t.Fatal("import recalculated a saved floor")
	}
	w.entities[0].Mana = 39
	if w.globalHealAllowed(0) {
		t.Fatal("saved-floor equality passed")
	}
	w.entities[0].Mana = 40
	w.entities[0].ActorLoad.Source.HasSpellbook = false
	if w.globalHealAllowed(0) {
		t.Fatal("source actor without a spellbook passed")
	}
}

func TestAutoHealing1191NativeRoundTripAndAbsentDefault(t *testing.T) {
	caster := acCaster(1, 2, 2, 51, 1<<6, 0)
	caster.Owner, caster.MaxMana = SelfSlot, 100
	hurt := spEnt(2, 3, 3)
	hurt.Owner, hurt.HP = SelfSlot, 30
	w := hlWorld(t, 17, acEnemies(t), []SpellRule{hlHeal()}, caster, hurt)
	old, err := w.MarshalBinary()
	if err != nil || old[0] != 95 {
		t.Fatal("absent policy changed legacy form", err)
	}
	if !w.SetAutoHealing(SelfSlot, 1) {
		t.Fatal("setting refused")
	}
	raw, err := w.MarshalBinary()
	if err != nil || raw[0] != 96 || bytes.Equal(raw, old) {
		t.Fatal("policy missing from native form", err)
	}
	var back World
	if err := CheckSaveForm(raw); err != nil || back.UnmarshalBinary(raw) != nil || back.Hash() != w.Hash() {
		t.Fatal("cold continuation lost policy", err)
	}
	for tick := 0; tick < 40; tick++ {
		Step(w, nil)
		Step(&back, nil)
		if w.Hash() != back.Hash() {
			t.Fatalf("continuation differs at tick%d", tick)
		}
	}
	if spAt(t, w, 2).HP <= 30 || spAt(t, w, 1).Mana >= 51 {
		t.Fatal("strict threshold did not lead to a real healing cast")
	}
	before := back.Hash()
	for _, cut := range []int{0, 1, len(raw) - 1, len(raw) - 5, len(raw) - 6, len(raw) - 12} {
		if err := back.UnmarshalBinary(raw[:cut]); err == nil || back.Hash() != before {
			t.Fatalf("truncated form accepted or mutated receiver at%d", cut)
		}
	}
	if err := back.UnmarshalBinary(old); err != nil {
		t.Fatal(err)
	}
	if _, present := back.AutoHealing(SelfSlot); present {
		t.Fatal("absent old policy inherited the receiver's setting")
	}
	if again, _ := back.MarshalBinary(); !bytes.Equal(again, old) {
		t.Fatal("old form no longer round-trips exactly")
	}
}
