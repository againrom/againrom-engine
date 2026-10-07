package game

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestReleaseCurrentCityPartyOrdinaryWireWins(t *testing.T) {
	f := currentTown(t, nil, nil)
	for i := range f.Carried {
		if f.Carried[i].Carry == nil {
			continue
		}
		for slot := range f.Carried[i].Carry.EquippedItems {
			f.Carried[i].Carry.EquippedItems[slot].Kind = 0
		}
		f.Carried[i].Carry.SkillXP[5]++
		if load := f.Carried[i].Carry.LiveLoad; load != nil && load.Inventory.Source.Class != 0 {
			load.Inventory.Source.SkillXP[5]++
		}
	}
	want := mapload.CloneParty(f.Carried)
	raw := currentTownSave(t, f)
	for cycle := 0; cycle < 2; cycle++ {
		g := currentTownReload(t, raw)
		if len(g.Carried) != len(want) {
			t.Fatal("city roster changed", cycle, len(g.Carried), len(want))
		}
		if !reflect.DeepEqual(f.Town.heroGrants, g.Town.heroGrants) {
			t.Fatal("city companion consumption changed", cycle, f.Town.heroGrants, g.Town.heroGrants)
		}
		for i, before := range want {
			after := g.Carried[i]
			if before.ID != after.ID || before.Name != after.Name || before.Hero != after.Hero || before.StartingHero != after.StartingHero || before.FigureFace != after.FigureFace {
				t.Fatalf("city member %d identity/base changed: before %+v after %+v", i, before, after)
			}
			if !reflect.DeepEqual(mapload.MemberItemEquipment(before, f.Table), mapload.MemberItemEquipment(after, g.Table)) ||
				!reflect.DeepEqual(mapload.MemberCarriedItems(before, f.Table), mapload.MemberCarriedItems(after, g.Table)) {
				t.Fatalf("city member %d current holdings changed", i)
			}
			if before.Carry != nil && (after.Carry == nil || before.Carry.SkillXP != after.Carry.SkillXP) {
				t.Fatal("city experience was recomputed from displayed levels", cycle, i)
			}
		}
		raw = currentTownSave(t, g)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil || a == nil || len(a.Party) == 0 {
		t.Fatal("no current party binding", err)
	}
	for _, p := range a.Party {
		if p.Member != nil || p.Policy == nil || p.Base == nil || p.City == nil {
			t.Fatal("city party duplicates ordinary member")
		}
	}
	if err := bindCurrentPartyRecords(a, &doc); err != nil {
		t.Fatal(err)
	}
	var index uint16
	for _, binding := range a.Bindings {
		if binding.ID == a.Party[0].Entity && !binding.Structure && !binding.Missing {
			index = binding.Object
		}
	}
	if index == 0 {
		t.Fatal("no ordinary city actor")
	}
	leaf, _, err := sav.NativeActions(doc.State)
	if err != nil {
		t.Fatal(err)
	}
	r := &doc.Objects[index-1]
	name := string([]byte{0xc4, 0xe0, 0xed, 0xe0, 0xf1, ' ', '3'})
	mustSetText(r, "Name", name)
	mustSetValue(r, "Body", 77)
	var nextHero uint32
	for _, binding := range a.Bindings {
		if binding.ID == a.Party[1].Entity && !binding.Structure && !binding.Missing {
			nextHero, err = savedStructureValue(&doc.Objects[binding.Object-1], "Identity")
		}
	}
	if err != nil || nextHero == 0 {
		t.Fatal("hero transfer has no ordinary actor", err)
	}
	for _, player := range doc.Players {
		mustSetValue(&doc.Objects[player-1], "Hero", nextHero)
	}
	modified, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	back, err := sav.DecodeDocumentData(modified)
	if err != nil {
		t.Fatal(err)
	}
	unchanged, _, err := sav.NativeActions(back.State)
	if err != nil || !bytes.Equal(leaf, unchanged) {
		t.Fatal("ordinary edit changed supplement", err)
	}
	wantBody := int32(77)
	if a.Party[0].Base.Native {
		wantBody -= int32(int8(a.Party[0].ordinary.Character.Basis.Human.Fields.Modifier[0]))
	}
	for cycle := 0; cycle < 2; cycle++ {
		g := currentTownReload(t, modified)
		if g.Carried[0].Name != name || g.Carried[0].StartingHero || !g.Carried[1].StartingHero || g.Carried[0].Hero.Body != wantBody {
			t.Fatal("retained city policy overrode current ordinary values", cycle, g.Carried[0].Name, g.Carried[0].StartingHero, g.Carried[0].Hero.Body, wantBody)
		}
		modified = currentTownSave(t, g)
	}
}

func TestReleaseCurrentCityRejectsMalformedPolicyAtomically(t *testing.T) {
	f := currentTown(t, nil, nil)
	raw := currentTownSave(t, f)
	beforeParty, beforeTown, beforeFame := mapload.CloneParty(f.Carried), f.Town, cloneFame(f.fame)
	cases := []struct {
		name string
		edit func(*currentActionData)
	}{
		{"member replay", func(a *currentActionData) { a.Party[0].Member = &mapload.PartyMember{} }},
		{"absent base", func(a *currentActionData) { a.Party[0].Base = nil }},
		{"absent policy", func(a *currentActionData) { a.Party[0].Policy = nil }},
		{"absent city policy", func(a *currentActionData) { a.Party[0].City = nil }},
		{"book mode", func(a *currentActionData) { a.Party[0].City.BookMode = 255 }},
		{"repeated member", func(a *currentActionData) { a.Party = append(a.Party, a.Party[0]) }},
		{"repeated binding", func(a *currentActionData) { a.Bindings = append(a.Bindings, a.Bindings[0]) }},
		{"missing binding", func(a *currentActionData) { a.Bindings = nil }},
		{"null actor", func(a *currentActionData) { a.Bindings[0].Object = 0 }},
		{"extra roster", func(a *currentActionData) { a.Roster = append(a.Roster, a.Party[0]) }},
		{"repeated item", func(a *currentActionData) { a.Ownership = append(a.Ownership, a.Ownership[0]) }},
		{"missing items", func(a *currentActionData) { a.Ownership = nil }},
		{"item replay", func(a *currentActionData) { a.Ownership[0].Item = &sim.SavedItemObject{} }},
		{"load class", func(a *currentActionData) { a.Party[0].City.Load = &currentCityLoadPolicy{SourceClass: 3} }},
		{"load width", func(a *currentActionData) { a.Party[0].City.Load = &currentCityLoadPolicy{SpeedLift: 1 << 32} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil || a == nil || len(a.Party) == 0 || len(a.Ownership) == 0 {
				t.Fatal("missing populated current city", err)
			}
			tc.edit(a)
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			bad, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err := f.RestoreOriginal(bad); err == nil {
				t.Fatal("malformed city policy was accepted")
			}
			if f.Town != beforeTown || !reflect.DeepEqual(f.Carried, beforeParty) || !reflect.DeepEqual(f.fame, beforeFame) {
				t.Fatal("rejected candidate changed the live city")
			}
		})
	}
}
