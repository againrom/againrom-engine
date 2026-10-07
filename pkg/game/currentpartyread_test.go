package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentPartyReadsOrdinaryBindingAndPlayerSuffix(t *testing.T) {
	doc, binding, _ := actorProjectionFixture(t, "Human")
	r := &doc.Objects[binding.ObjectIndex-1]
	name := string([]byte{0xc4, 0xe0, 0xed, 0xe0, 0xf1})
	for i := range r.Texts {
		if r.Texts[i].Name == "Name" {
			r.Texts[i].Value = name
		}
	}
	for i := range r.Values {
		switch r.Values[i].Name {
		case "Health":
			r.Values[i].Value = 7
		case "U4B":
			r.Values[i].Value = 138
		case "Stage":
			r.Values[i].Value = 2
		}
	}
	firstPlayer := doc.Objects[doc.Players[0]-1]
	raw, err := json.Marshal(firstPlayer)
	if err != nil {
		t.Fatal(err)
	}
	var secondPlayer sav.DocumentRecordData
	if err := json.Unmarshal(raw, &secondPlayer); err != nil {
		t.Fatal(err)
	}
	for i := range secondPlayer.Values {
		switch secondPlayer.Values[i].Name {
		case "This":
			secondPlayer.Values[i].Value = 0x12345678
		case "F58":
			secondPlayer.Values[i].Value = 37
		}
	}
	doc.Objects = append(doc.Objects, secondPlayer)
	doc.Players = append(doc.Players, uint16(len(doc.Objects)), doc.Players[0])
	before, _ := json.Marshal(doc)
	got, err := sav.ReadDocumentCharacters(doc, []uint16{binding.ObjectIndex, binding.ObjectIndex})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || !reflect.DeepEqual(got[0], got[1]) {
		t.Fatal("explicit alias membership changed", got)
	}
	c := got[0].Character
	if c.ArchiveIndex != binding.ObjectIndex || c.Key != 0 || c.Name != name || c.Stats[sav.StatHealth] != 7 || c.Stage != 2 || got[0].Face != 138 {
		t.Fatalf("ordinary actor fields were not read exactly: %+v", got[0])
	}
	if !c.Basis.Human.HasOwner || c.Basis.Human.ManaReservePercent != 37 {
		t.Fatal("last Player suffix lost to Token owner or repeated root", c.Basis.Human)
	}
	got[0].Character.Basis.Human.ManaReservePercent = 99
	if got[1].Character.Basis.Human.ManaReservePercent != 37 {
		t.Fatal("detached projections share mutable storage")
	}
	after, _ := json.Marshal(doc)
	if string(before) != string(after) {
		t.Fatal("ordinary read changed the document")
	}
	for _, id := range []uint16{0, uint16(len(doc.Objects) + 1), doc.Players[0]} {
		if rows, err := sav.ReadDocumentCharacters(doc, []uint16{binding.ObjectIndex, id}); err == nil || rows != nil {
			t.Fatal("invalid binding returned a partial party", id, err)
		}
	}
}

func TestCurrentPartyUsesFinalHoldingsBooksAndPools(t *testing.T) {
	doc, binding, original := actorProjectionFixture(t, "Human")
	for i := range doc.Objects[binding.ObjectIndex-1].Values {
		v := &doc.Objects[binding.ObjectIndex-1].Values[i]
		if v.Name == "Body" {
			v.Value = 17
		}
	}
	e := original.Entities()[0]
	e.ActorLoad.Source.Stats[sav.StatBody] = 17
	e.ID = 0
	e.Class, e.HP, e.MaxHP, e.Mana, e.MaxMana = 23, 7, 119, 5, 91
	e.SkillXP = [6]int32{17, 23, 31, 47, 59, 61}
	e.KnownSpells = 4
	e.Book = sim.Spellbook{State: sim.BookPresent}
	item := sim.PlainItem(0xe03)
	item.ObjectID = 91
	stock := sim.Stock{ID: 0, ItemInstances: []sim.ItemInstance{item}}
	stock.EquippedItems[1] = item
	w, err := sim.NewStockedWorld(23, original.Bounds(), sim.ModeCanonical, sim.Terrain{}, []sim.Entity{e}, nil, sim.Relations{}, nil, []sim.Stock{stock})
	if err != nil {
		t.Fatal(err)
	}
	stale := mapload.PartyMember{ID: "stable", Name: "stale", Hero: data.Hero{Body: 42},
		Carry: &mapload.Carry{SkillXP: [6]int32{999}}, Saved: &mapload.Saved{HP: 999},
		KnownSpells: 1, Book: sim.Spellbook{State: sim.BookAbsent}, Worn: [sim.EquipSlots]uint16{0x101},
		Carried: []uint16{0xe02}, SpellbookRestored: true}
	stale.StartingHero = true
	stale.PotionEffect = &sim.ActiveEffect{Target: 0}
	stale.OriginalHuman = mapload.BindOriginalHuman(stale, data.HumanState{Body: 999})
	stale.RetireOriginalHuman()
	p := captureCurrentParty(0, stale)
	a := &currentActionData{Bindings: []currentActionBinding{{ID: 0, Object: binding.ObjectIndex}},
		Party: []currentPartyMember{p}, Roster: []currentPartyMember{p}}
	if err := bindCurrentPartyRecords(a, &doc); err != nil {
		t.Fatal(err)
	}
	if err := stripCurrentPartyValues(a, w); err != nil {
		t.Fatal(err)
	}
	for _, rows := range [][]currentPartyMember{a.Party, a.Roster} {
		if rows[0].Name != nil || rows[0].Member != nil || rows[0].Policy == nil || !rows[0].Policy.Carry || !rows[0].Policy.Saved {
			t.Fatal("current values duplicated in continuation")
		}
		if rows[0].Base == nil || rows[0].Human == nil || !rows[0].Human.Retired {
			t.Fatal("base or Human values duplicated instead of mode/operands")
		}
	}
	ms := &Mission{World: w}
	if err := restoreCurrentPartyMembers(ms, a, nil); err != nil {
		t.Fatal(err)
	}
	got := ms.Party[0]
	if got.ID != "stable" || got.Name == "stale" || got.Hero.Body != 17 || got.Class != e.Class || got.Saved.HP != 7 || got.Saved.MaxHP != 119 || got.Carry.SkillXP != e.SkillXP {
		t.Fatal("current values or independent base operands changed", got)
	}
	if got.KnownSpells != e.KnownSpells || got.Book != e.Book || !got.SpellbookPresent || !got.SpellbookRestored {
		t.Fatal("current book or teaching mode lost", got)
	}
	if got.StartingHero || got.PotionEffect != nil || got.Profile.Fighter != a.Party[0].ordinary.Character.Basis.Human.Fighter {
		t.Fatal("ordinary primary/profile or expired potion lost to retained member")
	}
	if got.OriginalHuman == nil || !got.OriginalHuman.Retired || got.OriginalHuman.State.Body != 17 || got.OriginalHuman.Equipment[1].ObjectID != 91 || got.OriginalHuman.PartyID != "stable" {
		t.Fatal("Human state or guards did not use ordinary/current values", got.OriginalHuman)
	}
	legacy := p
	legacy.ordinary = a.Party[0].ordinary
	old, err := legacy.restoreFromCurrent(w, nil)
	if err != nil || !reflect.DeepEqual(old, got) {
		t.Fatal("legacy full-member input retained a second ordinary authority", err)
	}
	if len(got.CarriedItems) != 1 || got.CarriedItems[0].ObjectID != 91 || got.Carried[0] != item.Code || got.WornItems[1].ObjectID != 91 || got.Worn[0] != 0 || got.Worn[1] != item.Code {
		t.Fatal("current holdings or aliases lost", got)
	}
	if !reflect.DeepEqual(got, ms.Start.Roster[0]) || !reflect.DeepEqual(ms.Start.IDs, []sim.EntityID{0}) {
		t.Fatal("party and roster binding differ")
	}
	for _, corrupt := range []func(*currentPartyMember){
		func(p *currentPartyMember) { p.Member = &stale },
		func(p *currentPartyMember) { p.Policy = nil },
		func(p *currentPartyMember) { p.Base = nil },
		func(p *currentPartyMember) { p.Weapon = nil },
		func(p *currentPartyMember) { p.ordinary = nil },
		func(p *currentPartyMember) { p.Entity = 9999 },
	} {
		bad := a.Party[0]
		corrupt(&bad)
		before := mapload.CloneParty(ms.Party)
		if err := restoreCurrentPartyMembers(ms, &currentActionData{Party: []currentPartyMember{a.Party[0], bad}}, nil); err == nil || !reflect.DeepEqual(before, ms.Party) {
			t.Fatal("invalid policy partially adopted member state", err)
		}
	}
	ms.Party[0].CarriedItems[0].Code++
	unchanged, _ := w.CarriedItems(0)
	if unchanged[0].Code != item.Code || ms.Start.Roster[0].CarriedItems[0].Code != item.Code {
		t.Fatal("restored views share mutable containers")
	}
	for _, invalid := range []currentActionData{
		{Party: []currentPartyMember{p}},
		{Bindings: a.Bindings, Party: []currentPartyMember{p, p}},
		{Bindings: []currentActionBinding{a.Bindings[0], a.Bindings[0]}, Party: []currentPartyMember{p}},
	} {
		if err := bindCurrentPartyRecords(&invalid, &doc); err == nil {
			t.Fatal("invalid membership binding accepted")
		}
	}
}
