package game

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestInstanceWeightLegacyCitySalesKeepImportPolicy(t *testing.T) {
	f := city1102Front(t, false)
	city1102Take(t, f, 3)
	f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	s, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic predecessor input; the frozen installed witness is separate.
	// The old snapshot had no city topology, ordered stock or live load carrier.
	s.OriginalCity.Version, s.CityObjects = 3, nil
	legacyCityFixtureIdentities(&s)
	for i := range s.Party {
		clearLegacyItemWeights(&s.Party[i])
		s.Party[i].Carry.LiveLoad, s.Party[i].Carry.OrderedStacks = nil, nil
	}
	for i := range s.OriginalCity.Bindings {
		p := &s.OriginalCity.Bindings[i].Baseline
		clearLegacyItemWeights(p)
		p.Carry.LiveLoad, p.Carry.OrderedStacks = nil, nil
	}
	if _, town, err := f.Restore(s); err != nil || !town {
		t.Fatal("legacy city sale LOAD", town, err)
	}
	if !reflect.DeepEqual(f.Carried, s.Party) || f.Town.cityObjects != nil {
		t.Fatal("legacy LOAD rewrote current party values or invented old topology")
	}
	loaded, _, err := f.Snapshot(false)
	if err != nil || !reflect.DeepEqual(loaded.OriginalCity, s.OriginalCity) {
		t.Fatal("legacy history/baseline changed during LOAD", err)
	}
	roundtrip := func(current *FrontEnd, count, gold int) *FrontEnd {
		t.Helper()
		before, _, err := current.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := current.ExportOriginalSave(before, "legacy current sale")
		if err != nil {
			t.Fatal(err)
		}
		after, _, err := current.Snapshot(false)
		if err != nil || !reflect.DeepEqual(before, after) {
			t.Fatal("SAVE changed current legacy input", err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil || a == nil || len(a.Party) != len(current.Carried) {
			t.Fatal("SAV lacks the current city population", err)
		}
		cold := &FrontEnd{InstallResources: current.InstallResources, RuntimeServices: current.RuntimeServices}
		if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("cold current city SAV", town, err)
		}
		if current.Town.Gold() != gold || cold.Town.Gold() != gold {
			t.Fatal("current purse changed across SAV", current.Town.Gold(), cold.Town.Gold(), gold)
		}
		var heroObject uint16
		for _, p := range a.Party {
			member := trainingPartyMember(t, current, string(p.ID))
			back := trainingPartyMember(t, cold, string(p.ID))
			var object uint16
			for _, binding := range a.Bindings {
				if binding.ID == p.Entity && !binding.Structure && !binding.Missing {
					object = binding.Object
				}
			}
			if string(p.ID) == "hero" {
				heroObject = object
			}
			chars, err := sav.ReadDocumentCharacters(doc, []uint16{object})
			if err != nil || len(chars) != 1 {
				t.Fatal("ordinary current Human binding", object, err)
			}
			c := chars[0].Character
			if member.OriginalHuman == nil || back.OriginalHuman == nil {
				t.Fatal("legacy Human tail carrier disappeared", p.ID)
			}
			wantHuman, gotHuman := member.OriginalHuman.State, back.OriginalHuman.State
			if wantHuman.Attack.Tail != gotHuman.Attack.Tail || wantHuman.Base.Tail != gotHuman.Base.Tail ||
				wantHuman.Modifier.Attack.Tail != gotHuman.Modifier.Attack.Tail {
				t.Fatal("legacy current Human lost opaque tails", p.ID, wantHuman.Attack.Tail, gotHuman.Attack.Tail,
					wantHuman.Base.Tail, gotHuman.Base.Tail, wantHuman.Modifier.Attack.Tail, gotHuman.Modifier.Attack.Tail)
			}
			d, hp, mp := mapload.PartyDisplayWithTable(member, current.Table)
			for i, value := range []int32{d.Body, d.Reaction, d.Mind, d.Spirit, d.Speed} {
				if c.Stats[i] != uint16(value) {
					t.Fatal("ordinary Human differs from current card", p.ID, i, c.Stats[i], value)
				}
			}
			if c.Stats[7] != uint16(d.Capacity) || c.Stats[8] != uint16(hp) || c.Stats[9] != uint16(d.HealthMax) ||
				c.Stats[11] != uint16(mp) || c.Stats[12] != uint16(d.ManaMax) || c.Experience != uint32(d.Experience) {
				t.Fatal("ordinary Human pools/capacity/experience differ from current values", p.ID, c.Stats, c.Experience, d, hp, mp)
			}
			for i, xp := range member.Carry.SkillXP {
				if c.SkillXP[i] != uint32(xp) || back.Carry.SkillXP[i] != xp {
					t.Fatal("current earned skill XP changed", p.ID, i)
				}
			}
			beforeItems := mapload.MemberCarriedItems(member, current.Table)
			afterItems := mapload.MemberCarriedItems(back, cold.Table)
			if len(beforeItems) != len(afterItems) {
				t.Fatal("cold SAV changed current pack quantity", string(p.ID))
			}
			for i, item := range beforeItems {
				if item.WeightPresent != afterItems[i].WeightPresent || item.Weight != afterItems[i].Weight {
					t.Fatalf("cold SAV changed %s pack[%d] weight: %d/present=%t -> %d/present=%t", p.ID, i,
						item.Weight, item.WeightPresent, afterItems[i].Weight, afterItems[i].WeightPresent)
				}
			}
			got, gotHP, gotMP := mapload.PartyDisplayWithTable(back, cold.Table)
			if !reflect.DeepEqual(d, got) || hp != gotHP || mp != gotMP || member.Hero != back.Hero || member.Book != back.Book ||
				member.KnownSpells != back.KnownSpells || !reflect.DeepEqual(mapload.MemberItemEquipment(member, current.Table), mapload.MemberItemEquipment(back, cold.Table)) ||
				!reflect.DeepEqual(beforeItems, afterItems) || back.Carry.LiveLoad != nil {
				currentItemFieldDiagnostics(t, "Card", reflect.ValueOf(d), reflect.ValueOf(got))
				currentItemFieldDiagnostics(t, "Member", reflect.ValueOf(member), reflect.ValueOf(back))
				t.Fatal("cold SAV changed current card, books, items or load absence", string(p.ID), hp, gotHP, mp, gotMP)
			}
		}
		items := mapload.MemberCarriedItems(trainingPartyMember(t, cold, "hero"), nil)
		if len(items) != count {
			t.Fatal("current sale quantity changed", len(items), count)
		}
		for _, item := range items {
			if item.Code != 0x0e0d || item.Price != 5 || item.WeightPresent || item.Weight != 0 {
				t.Fatal("undefined-weight Item changed through SAV", item)
			}
		}
		if count == 2 {
			refs, present := savedObjectRefs(&doc.Objects[heroObject-1], "Inventory")
			if !present || len(refs) != 1 {
				t.Fatal("equal legacy flat units are not one stacked occurrence", refs, present)
			}
			leaf, _, err := sav.NativeActions(doc.State)
			if err != nil {
				t.Fatal(err)
			}
			savedObjectSetValue(&doc.Objects[refs[0]-1], "F4A", 9)
			changed, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			editedDoc, err := sav.DecodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			unchanged, _, err := sav.NativeActions(editedDoc.State)
			if err != nil || !bytes.Equal(leaf, unchanged) {
				t.Fatal("ordinary weight edit changed absence policy", err)
			}
			edited := &FrontEnd{InstallResources: current.InstallResources, RuntimeServices: current.RuntimeServices}
			if _, town, err := edited.RestoreOriginal(changed); err != nil || !town {
				t.Fatal("ordinary weight edit LOAD", town, err)
			}
			want := mapload.MemberCarriedItems(trainingPartyMember(t, cold, "hero"), nil)
			for i := range want {
				want[i].Weight, want[i].WeightPresent = 9, true
			}
			got := mapload.MemberCarriedItems(trainingPartyMember(t, edited, "hero"), nil)
			if !reflect.DeepEqual(want, got) {
				t.Fatal("ordinary weight failed to replace only its own absence", want, got)
			}
		}
		return cold
	}
	f = roundtrip(f, 2, 3008)
	city1102Take(t, f, 1)
	if len(f.Shop.table) != 1 || f.Shop.table[0].Count != 1 || f.Shop.table[0].WeightPresent || f.Shop.table[0].Weight != 0 {
		t.Fatal("next sale did not stage one undefined-weight unit", f.Shop.table)
	}
	result := f.TownScreen().(*townScreen).ShopClick(ui.ShopControl{Kind: ui.ShopControlButton, Index: 2})
	if !strings.Contains(result.Msg, "he pays 3;") {
		t.Fatal("next sale changed the price", result.Msg)
	}
	roundtrip(f, 1, 3011)
}
