package mapload_test

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func nativeCarryParty() []mapload.PartyMember {
	p := heroWith(nil)
	p[0].ID = "native-history"
	p[0].Profile = data.Profile{Fighter: true, HealthColumn: true}
	return p
}

func TestNativeCarryHistoryUsesActualActorAndPreservesAbsence(t *testing.T) {
	partial := sim.NativeActorBasis{
		BasePresent: true, BaseKnown: 1 | 1<<23,
		ModifierPresent: true, ModifierKnown: 1<<40 | 1<<41,
		BodyPresent: true, BodyKnown: false, Body: 173,
	}
	partial.Base[0], partial.Base[22], partial.Base[23] = 17, 199, 23
	partial.Modifier[40], partial.Modifier[41], partial.Modifier[58] = 251, 255, 197
	knownZero := (sim.NativeActorBasis{}).WithBase([24]byte{}).WithModifier([64]byte{}).WithBody(0)
	for _, tc := range []struct {
		name  string
		basis sim.NativeActorBasis
		class sim.NativeClass
	}{
		{"partial", partial, sim.NativeClass{Present: true, Fighter: false}},
		{"known-zero", knownZero, sim.NativeClass{Present: true, Fighter: true}},
		{"explicit-absence", sim.NativeActorBasis{}, sim.NativeClass{}},
	} {
		for _, route := range []string{"party", "joiner"} {
			t.Run(tc.name+"/"+route, func(t *testing.T) {
				m, party := carryMap(t), nativeCarryParty()
				w, st := mustStart(t, m, party)
				done := endedWith(t, m, w, st.IDs[0], func(e *sim.Entity) {
					e.NativeBasis, e.NativeClass = tc.basis, tc.class
				}, nil, [sim.EquipSlots]uint16{})
				var captured []mapload.PartyMember
				if route == "party" {
					captured = mapload.CarryParty(party, done, st.IDs)
				} else {
					var ids []sim.EntityID
					captured, ids = mapload.CarryRosterIDs(nil, done, nil,
						map[sim.EntityID]mapload.PartyMember{st.IDs[0]: party[0]})
					if len(ids) != 1 || ids[0] != st.IDs[0] {
						t.Fatalf("actual roster joiner was not captured: %v", ids)
					}
				}
				if len(captured) != 1 || captured[0].Carry == nil {
					t.Fatalf("actual living member has no Carry: %v", captured)
				}
				captured = mapload.OwnParty(mapload.CloneParty(captured))
				next, nextStart := mustStart(t, m, captured)
				got := entityOf(t, next, nextStart.IDs[0])
				if got.NativeBasis != tc.basis {
					t.Errorf("next actual spawn changed native history: got %+v want %+v", got.NativeBasis, tc.basis)
				}
				if got.NativeClass != tc.class {
					t.Errorf("next actual spawn changed native class: got %+v want %+v", got.NativeClass, tc.class)
				}
				if got.ActorLoad.Source.Class != 0 {
					t.Error("native carry was promoted to a source-backed actor")
				}
				if original := entityOf(t, done, st.IDs[0]); original.NativeBasis != tc.basis || original.NativeClass != tc.class {
					t.Error("capture changed the completed World")
				}
			})
		}
	}
}

func TestFirstMaterializedCarryHasNoActorHistory(t *testing.T) {
	m, party := carryMap(t), nativeCarryParty()
	fresh, freshStart := mustStart(t, m, party)
	want := entityOf(t, fresh, freshStart.IDs[0])
	party[0] = mapload.MaterializePartyCarry(party[0], nil)
	if party[0].Carry == nil {
		t.Fatal("fixture has no first holdings Carry")
	}
	materialized, st := mustStart(t, m, mapload.OwnParty(mapload.CloneParty(party)))
	got := entityOf(t, materialized, st.IDs[0])
	if got.NativeBasis != want.NativeBasis || got.NativeClass != want.NativeClass {
		t.Fatal("first holdings materialization invented observed actor history", got.NativeBasis, got.NativeClass)
	}
}

func TestNativeCarryPotionRecordIsDroppedWithItsRawHistory(t *testing.T) {
	m, party := carryMap(t), nativeCarryParty()
	w, st := mustStart(t, m, party)
	w = endedWith(t, m, w, st.IDs[0], func(e *sim.Entity) {
		e.NativeBasis = sim.NativeActorBasis{ModifierPresent: true, ModifierKnown: uint64(3) << 44}
		binary.LittleEndian.PutUint16(e.NativeBasis.Modifier[44:], 0x1234)
		e.NativeBasis.Modifier[63] = 197
	}, nil, [sim.EquipSlots]uint16{})
	base := entityOf(t, w, st.IDs[0])
	if !w.RestorePotionEffect(st.IDs[0], sim.ActiveEffect{Kind: sim.EffectAbsorption, Mode: sim.EffectDuration, Magnitude: 7, Remaining: 3}) {
		t.Fatal("actual potion attachment refused")
	}
	if e := entityOf(t, w, st.IDs[0]); binary.LittleEndian.Uint16(e.NativeBasis.Modifier[44:]) != 0x123b {
		t.Fatal("attachment fixture did not maintain raw history")
	}
	party = mapload.CarryParty(party, w, st.IDs)
	if party[0].PotionEffect == nil {
		t.Fatal("fixture carried no potion record")
	}
	next, ns := mustStart(t, m, mapload.OwnParty(mapload.CloneParty(party)))
	for tick := 0; tick <= 3; tick++ {
		got := entityOf(t, next, ns.IDs[0])
		if got.NativeBasis != base.NativeBasis || got.Absorption != base.Absorption || len(next.ActiveEffects()) != 0 {
			t.Fatalf("tick=%d raw=%#x effective=%d want raw=%#x effective=%d", tick, binary.LittleEndian.Uint16(got.NativeBasis.Modifier[44:]), got.Absorption, binary.LittleEndian.Uint16(base.NativeBasis.Modifier[44:]), base.Absorption)
		}
		sim.Step(next, nil)
	}
}

func TestNativeCarryCloneOwnsHistory(t *testing.T) {
	p := nativeCarryParty()
	p[0].Carry = &mapload.Carry{NativeHistory: &mapload.NativeCarryHistory{}}
	c := mapload.CloneParty(p)
	c[0].Carry.NativeHistory.Basis = (sim.NativeActorBasis{}).WithBody(27)
	if p[0].Carry.NativeHistory.Basis.HasValues() {
		t.Fatal("clone aliases the input history")
	}
}

func TestNativeCarryTownEquipmentMaintainsOrderedHistory(t *testing.T) {
	p := nativeCarryParty()[0]
	b := [64]byte{18: 0x34, 19: 0x12}
	p.Carry = &mapload.Carry{NativeHistory: &mapload.NativeCarryHistory{Basis: (sim.NativeActorBasis{}).WithModifier(b), Class: sim.NativeClass{Present: true, Fighter: true}}}
	shield := sim.ItemInstance{Code: 0x0201, Kind: 1, Effects: []sim.ItemEffect{{Kind: 12, Operand: 3}}, SourceEquipment: sim.SourceEquipment{Class: sim.SourceShield}}
	before := mapload.CloneParty([]mapload.PartyMember{p})[0]
	p.Carry.EquippedItems[1], p.Carry.Equipped[1] = shield, shield.Code
	if err := mapload.UpdatePartyLoad(before, &p, nil, true, true); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(p.Carry.NativeHistory.Basis.Modifier[18:]) != 0x1237 {
		t.Fatal("town equip lost literal effect history")
	}
	before = mapload.CloneParty([]mapload.PartyMember{p})[0]
	p.Carry.EquippedItems[1], p.Carry.Equipped[1] = sim.ItemInstance{}, 0
	if err := mapload.UpdatePartyLoad(before, &p, nil, true, true); err != nil {
		t.Fatal(err)
	}
	if binary.LittleEndian.Uint16(p.Carry.NativeHistory.Basis.Modifier[18:]) != 0x1234 {
		t.Fatal("town removal lost literal effect history")
	}
}

func TestNativeCarrySiegeSpawnUsesObservedHistory(t *testing.T) {
	p := nativeCarryParty()[0]
	p.MercenaryType, p.Class, p.FigureFace = 1, 1, 1
	want := sim.NativeActorBasis{BasePresent: true, BaseKnown: 1, ModifierPresent: true, ModifierKnown: 1, BodyPresent: true, BodyKnown: true, Body: 47}
	want.Base[0], want.Modifier[0] = 29, 31
	p.Carry = &mapload.Carry{NativeHistory: &mapload.NativeCarryHistory{Basis: want}}
	w, st, err := mapload.StartMission(carryMap(t), initialBasisTable(), mapload.DifficultyNormal, []mapload.PartyMember{p})
	if err != nil {
		t.Fatal(err)
	}
	e := entityOf(t, w, st.IDs[0])
	if e.Humanoid || e.NativeBasis != want || e.NativeClass != (sim.NativeClass{}) {
		t.Fatal("actual siege mint lost captured native history")
	}
}
