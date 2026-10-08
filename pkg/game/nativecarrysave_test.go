package game

import (
	"encoding/json"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func nativeCarryMember(t *testing.T) (mapload.PartyMember, *mapload.Table) {
	t.Helper()
	m, table := absentMemberFixture(t)
	m.Carry.LiveLoad, m.OriginalHuman, m.PotionEffect = nil, nil, nil
	b := sim.NativeActorBasis{BasePresent: true, BaseKnown: 3, ModifierPresent: true, ModifierKnown: uint64(3) << 44, BodyPresent: true, BodyKnown: true, Body: 29}
	b.Base[0], b.Base[1], b.Base[23] = 7, 9, 201
	b.Modifier[44], b.Modifier[45], b.Modifier[63] = 17, 3, 203
	m.Carry.NativeHistory = &mapload.NativeCarryHistory{Basis: b, Class: sim.NativeClass{Present: true, Fighter: true}}
	return m, table
}

func TestNativeCarryDetachedOrdinarySavePreservesHistory(t *testing.T) {
	for _, absent := range []bool{false, true} {
		m, table := nativeCarryMember(t)
		if absent {
			m.Carry.NativeHistory = &mapload.NativeCarryHistory{}
		}
		want := *m.Carry.NativeHistory
		for cycle := 0; cycle < 2; cycle++ {
			p, err := captureCurrentPartyTemplate(777, m, m, table)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			var cold currentPartyMember
			if err := json.Unmarshal(raw, &cold); err != nil {
				t.Fatal(err)
			}
			m, err = cold.restoreFromCurrent(emptyTemplateWorld(t), table)
			if err != nil {
				t.Fatal(err)
			}
			if m.Carry == nil || m.Carry.NativeHistory == nil || *m.Carry.NativeHistory != want {
				t.Fatalf("absent=%v cycle=%d ordinary template lost observed native history: %+v", absent, cycle, m.Carry)
			}
		}
	}
}

func TestNativeCarryMissionStateUsesRestoredActor(t *testing.T) {
	m, table := nativeCarryMember(t)
	p, err := captureCurrentPartyTemplate(1, m, m, table)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.bindTemplate(); err != nil {
		t.Fatal(err)
	}
	p.Template = nil
	e := sim.Entity{ID: 1, HP: 100, MaxHP: 100, Humanoid: true, NativeBasis: m.Carry.NativeHistory.Basis, NativeClass: m.Carry.NativeHistory.Class}
	w, err := sim.NewStockedWorld(0, sim.Bounds{Width: 128, Height: 128}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1}})
	if err != nil {
		t.Fatal(err)
	}
	got, err := p.restoreFromCurrent(w, table)
	if err != nil {
		t.Fatal(err)
	}
	if got.Carry == nil || got.Carry.NativeHistory == nil || *got.Carry.NativeHistory != *m.Carry.NativeHistory {
		t.Fatal("current mission discarded final restored native actor history", got.Carry)
	}
}

func TestNativeCarryOrdinaryKnownBytesWinAndUnknownResidueRemains(t *testing.T) {
	m, table := nativeCarryMember(t)
	p, err := captureCurrentPartyTemplate(777, m, m, table)
	if err != nil {
		t.Fatal(err)
	}
	r := &p.Template.Records.Objects[p.Template.Records.Actor-1]
	base, err := savedActorRaw(r, "U114", 24)
	if err != nil {
		t.Fatal(err)
	}
	base[0], base[1], base[23] = 31, 5, 199
	modifier, err := savedActorRaw(r, "UD4", 64)
	if err != nil {
		t.Fatal(err)
	}
	modifier[44], modifier[45], modifier[63] = 33, 7, 198
	mustSetValue(r, "Body", 41)
	got, err := p.restoreFromCurrent(emptyTemplateWorld(t), table)
	if err != nil {
		t.Fatal(err)
	}
	want := *m.Carry.NativeHistory
	want.Basis.Base[0], want.Basis.Base[1] = 31, 5
	want.Basis.Modifier[44], want.Basis.Modifier[45] = 33, 7
	want.Basis.Body = 41
	if got.Carry == nil || got.Carry.NativeHistory == nil || *got.Carry.NativeHistory != want {
		t.Fatal("known ordinary edits or unknown residue lost")
	}
	if _, err := captureCurrentPartyTemplate(777, got, got, table); err != nil {
		t.Fatal(err)
	}
}
