package game

import (
	"encoding/binary"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentCityGraphKeysAvoidProducerReservations(t *testing.T) {
	doc := cityGraftFixture(t, false)
	reserved, err := sav.ReserveDocumentKeys(doc, 13)
	if err != nil {
		t.Fatal(err)
	}
	collision := reserved[10]
	spellKey := reserved[11]
	effectKey := reserved[12]
	graph := &cityObjectTopology{
		ItemRecords: map[sim.SavedObjectID]sim.SavedItemObject{
			1: {ID: 1, Token: sim.SavedObjectToken{Identity: collision, Reference: 0xdeadbeef}},
		},
		EffectRecords: map[sim.SavedObjectID]sim.SavedEffectObject{
			2: {ID: 2, Token: sim.SavedObjectToken{Identity: effectKey}},
		},
		SpellRecords: map[sim.SavedObjectID]sim.SavedSpellObject{
			3: {ID: 3, This: spellKey},
		},
	}
	before, err := sav.CloneDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := avoidCurrentCityGraphKeyCollisions(doc, graph); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(doc, before) {
		t.Fatal("collision scan mutated the current document")
	}
	if graph.ItemRecords[1].Token.Identity != collision || graph.EffectRecords[2].Token.Identity != effectKey || graph.SpellRecords[3].This != spellKey {
		t.Fatal("valid detached graph identity was erased")
	}
	if graph.ItemRecords[1].Token.Reference != 0xdeadbeef {
		t.Fatal("collision scan changed an unrelated token field")
	}
	keys, err := reserveCurrentCityDocumentKeys(doc, graph, 11)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range keys {
		if key == collision || key == effectKey || key == spellKey {
			t.Fatalf("producer reservation reused detached graph identity %#x", key)
		}
	}
	var existing uint32
	for i := range doc.Objects {
		for _, name := range []string{"Identity", "This"} {
			if value, lookupErr := savedStructureValue(&doc.Objects[i], name); lookupErr == nil && value != 0 {
				existing = value
				break
			}
		}
		if existing != 0 {
			break
		}
	}
	if existing == 0 {
		t.Fatal("fixture lacks an ordinary identity")
	}
	item := graph.ItemRecords[1]
	item.Token.Identity = existing
	graph.ItemRecords[1] = item
	if err := avoidCurrentCityGraphKeyCollisions(doc, graph); err != nil {
		t.Fatal(err)
	}
	if graph.ItemRecords[1].Token.Identity != 0 {
		t.Fatal("ordinary document identity collision remained in detached graph")
	}
}

func TestCurrentCityGraftSnapshotUsesPartyIdentity(t *testing.T) {
	retained := cityfixture.City(false)
	retained.Objects[1].Unit.Raw154[179] = 0xe5
	p, err := sav.CityFromData(retained)
	if err != nil {
		t.Fatal(err)
	}
	source, err := p.DocumentData()
	if err != nil {
		t.Fatal(err)
	}
	current := cityGraftFixture(t, false)
	companion, leader := cityGraftNamed(t, &current, "Companion"), cityGraftNamed(t, &current, "Leader")
	mustSetValue(&current.Objects[companion-1], "Identity", nativeCityIdentity(0))
	mustSetValue(&current.Objects[leader-1], "Identity", nativeCityIdentity(1))
	mustSetValue(&current.Objects[current.Players[0]-1], "Hero", nativeCityIdentity(1))
	s := Snapshot{Party: []mapload.PartyMember{{ID: "companion"}, {ID: "leader"}}, OriginalCity: &SnapshotOriginalCity{Document: retained}}
	s.OriginalCity.Bindings = []SnapshotCityBinding{
		{PartyID: "leader", Identity: cityGraftValue(t, &source.Objects[cityGraftNamed(t, &source, "Leader")-1], "Identity")},
		{PartyID: "companion", Identity: cityGraftValue(t, &source.Objects[cityGraftNamed(t, &source, "Companion")-1], "Identity")},
	}
	doc, err := graftSnapshotCityResidue(current, s)
	if err != nil {
		t.Fatal(err)
	}
	if cityGraftRaw(t, &doc.Objects[cityGraftNamed(t, &doc, "Companion")-1], "U154")[179] != 0xe5 {
		t.Fatal("snapshot binding order displaced stable party identity")
	}
	s.Party[1].ID = "companion"
	doc, err = graftSnapshotCityResidue(current, s)
	if err != nil {
		t.Fatal("ambiguous optional source blocked current city", err)
	}
	if cityGraftRaw(t, &doc.Objects[cityGraftNamed(t, &doc, "Companion")-1], "U154")[179] == 0xe5 {
		t.Fatal("ambiguous party identity selected retained residue")
	}
}

func TestCurrentCityGraftMapUnitIDUsesCurrentSavedValue(t *testing.T) {
	for _, test := range []struct {
		name    string
		current bool
		id      uint16
		want    uint32
	}{
		{"absent", false, 0, 0x39a63185},
		{"cleared", true, 0, 0x39a60000},
		{"changed", true, 321, 0x39a60141},
	} {
		t.Run(test.name, func(t *testing.T) {
			current := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "T08"}}}
			source := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "T08", Value: 0x39a63185}}}
			mergeCityActorResidue(&current, &source, currentCityActorGraft{CurrentMapUnit: test.current, MapUnitID: test.id})
			if got := cityGraftValue(t, &current, "T08"); got != test.want {
				t.Fatalf("T08=%#x, want %#x", got, test.want)
			}
		})
	}
}

func TestCurrentCityGraftLegacyHireClearedMapUnitSurvivesSaveLoad(t *testing.T) {
	f, newFront := cityRosterHiredFixture(t, []int{14}, []int{3})
	doc, err := sav.DecodeDocumentData(hiredActorWithoutSupplement(t, currentTownSave(t, f)))
	if err != nil {
		t.Fatal(err)
	}
	words := []uint32{0x19041904, 0x39a63185, 0xabcd0000}
	count := 0
	for i := range doc.Objects {
		actor := &doc.Objects[i]
		if actor.Class != "Human" || uint8(cityGraftValue(t, actor, "U148")) != 14 {
			continue
		}
		if count >= len(words) {
			t.Fatal("extra hired actor")
		}
		mustSetValue(actor, "T08", words[count])
		count++
	}
	if count != len(words) {
		t.Fatal("missing hired actors", count)
	}
	legacy, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	f = cityProjectionLoad(t, legacy, newFront)
	for cycle := 0; cycle < 2; cycle++ {
		count = 0
		for _, member := range f.Carried {
			if member.MercenaryType != 14 {
				continue
			}
			count++
			if member.Saved == nil || member.Saved.MapUnitID != 0 {
				t.Fatalf("cycle %d hire %s retains previous map placement: %+v", cycle, member.ID, member.Saved)
			}
		}
		if count != len(words) {
			t.Fatal("LOAD changed hired population", count)
		}
		raw := currentTownSave(t, f)
		written, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		var got []uint32
		for i := range written.Objects {
			actor := &written.Objects[i]
			if actor.Class == "Human" && uint8(cityGraftValue(t, actor, "U148")) == 14 {
				got = append(got, cityGraftValue(t, actor, "T08"))
			}
		}
		want := []uint32{0x19040000, 0x39a60000, 0xabcd0000}
		if !slices.Equal(got, want) {
			t.Fatalf("cycle %d ordinary T08=%#x, want %#x", cycle, got, want)
		}
		cold := cityProjectionLoad(t, raw, newFront)
		for _, member := range cold.Carried {
			if member.MercenaryType == 14 && (member.Saved == nil || member.Saved.MapUnitID != 0) {
				t.Fatalf("cycle %d SAVE/LOAD restored previous map placement: %+v", cycle, member.Saved)
			}
		}
		f = cold
	}
}

func cityGraftFixture(t *testing.T, retained bool) sav.DocumentData {
	t.Helper()
	data := cityfixture.City(false)
	if retained {
		data.Objects[1].Unit.SpellbookFlag = 1
		data.Objects[1].Unit.SpellbookCount = 1
	} else {
		data.Objects[1].Unit.ContainerFlag = 0
	}
	p, err := sav.CityFromData(data)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := p.DocumentData()
	if err != nil {
		t.Fatal(err)
	}
	return doc
}

func cityGraftNamed(t *testing.T, doc *sav.DocumentData, name string) uint16 {
	t.Helper()
	for i, r := range doc.Objects {
		for _, field := range r.Texts {
			if field.Name == "Name" && field.Value == name {
				return uint16(i + 1)
			}
		}
	}
	t.Fatalf("missing test object %q", name)
	return 0
}

func cityGraftRaw(t *testing.T, r *sav.DocumentRecordData, name string) []byte {
	t.Helper()
	for _, field := range r.Raw {
		if field.Name == name {
			return field.Bytes
		}
	}
	t.Fatalf("missing raw %s", name)
	return nil
}

func cityGraftRefs(t *testing.T, r *sav.DocumentRecordData, name string) []uint16 {
	t.Helper()
	for _, field := range r.RefSlots {
		if field.Name == name {
			return field.Objects
		}
	}
	t.Fatalf("missing refs %s", name)
	return nil
}

func cityGraftValue(t *testing.T, r *sav.DocumentRecordData, name string) uint32 {
	t.Helper()
	v, err := savedStructureValue(r, name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCurrentCityGraftCurrentFieldsAndSharedDiary(t *testing.T) {
	source, current := cityGraftFixture(t, true), cityGraftFixture(t, false)
	s1, s2 := cityGraftNamed(t, &source, "Companion"), cityGraftNamed(t, &source, "Leader")
	c1, c2 := cityGraftNamed(t, &current, "Companion"), cityGraftNamed(t, &current, "Leader")
	sp, cp := source.Players[0], current.Players[0]
	oldPlayerKey := cityGraftValue(t, &source.Objects[sp-1], "This")
	diary := mustNewDiaryRecord(2, oldPlayerKey)
	binary.LittleEndian.PutUint32(cityGraftRaw(t, &diary, "Journal"), 0x12345678)
	binary.LittleEndian.PutUint16(cityGraftRaw(t, &diary, "JournalWords")[2:], 0x1234)
	source.Objects = append(source.Objects, diary)
	for _, index := range []uint16{s1, s2} {
		mustSetRefs(&source.Objects[index-1], "Diary", []uint16{uint16(len(source.Objects))})
	}
	for i := range source.Objects[sp-1].Inline {
		if source.Objects[sp-1].Inline[i].Name == "Diary" {
			source.Objects[sp-1].Inline[i].Record = diary
		}
	}
	mustSetValue(&source.Objects[s1-1], "Health", 9)
	mustSetValue(&source.Objects[s1-1], "U4C", 0xa6)
	mustSetValue(&source.Objects[s1-1], "U49", 7)
	mustSetValue(&source.Objects[s1-1], "U64", 0x01000000)
	cityGraftRaw(t, &source.Objects[s1-1], "U154")[179] = 0xe7
	cityGraftRaw(t, &source.Objects[s1-1], "U154")[10] = 9
	binary.LittleEndian.PutUint32(cityGraftRaw(t, &source.Objects[s1-1], "U50"), oldPlayerKey)
	binary.LittleEndian.PutUint32(cityGraftRaw(t, &source.Objects[s1-1], "U158")[0x0c:], cityGraftValue(t, &source.Objects[s2-1], "Identity"))
	cityGraftRaw(t, &source.Objects[sp-1], "Raw10")[7] = 0xc1
	cityGraftRaw(t, &source.Objects[sp-1], "PRaw32")[3] = 0x9a
	cityGraftRaw(t, &source.Objects[sp-1], "PRaw32")[31] = 0
	mustSetValue(&source.Objects[sp-1], "F44", 77)
	mustSetValue(&source.Objects[sp-1], "F58", 81)
	var err error
	source, _, err = sav.ReindexDocumentData(source)
	if err != nil {
		t.Fatal(err)
	}
	s1, s2 = cityGraftNamed(t, &source, "Companion"), cityGraftNamed(t, &source, "Leader")
	mustSetText(&current.Objects[c1-1], "Name", "Current companion")
	mustSetValue(&current.Objects[c1-1], "Health", 111)
	mustSetValue(&current.Objects[c1-1], "U4C", 0)
	cityGraftRaw(t, &current.Objects[c1-1], "U154")[10] = 33
	cityGraftRaw(t, &current.Objects[c1-1], "UA6")[23] = 0x63
	mustSetValue(&current.Objects[cp-1], "Money", 999)
	mustSetValue(&current.Objects[cp-1], "F58", 95)
	cityGraftRaw(t, &current.Objects[cp-1], "PRaw32")[31] = 2
	beforeCurrent, _ := sav.CloneDocumentData(current)
	beforeSource, _ := sav.CloneDocumentData(source)
	doc, err := graftCurrentCityResidue(current, source,
		[]currentCityActorGraft{{Source: s1, Current: c1}, {Source: s2, Current: c2}},
		[]currentCityPlayerGraft{{Source: source.Players[0], Current: cp, CurrentFormation: true, CurrentAutoheal: true}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(source, beforeSource) || !reflect.DeepEqual(current, beforeCurrent) {
		t.Fatal("graft mutated an input")
	}
	for round := 0; round < 2; round++ {
		a := &doc.Objects[cityGraftNamed(t, &doc, "Current companion")-1]
		b := &doc.Objects[cityGraftNamed(t, &doc, "Leader")-1]
		p := &doc.Objects[doc.Players[0]-1]
		if cityGraftValue(t, a, "Health") != 111 || cityGraftValue(t, a, "U4C") != 0xa0 || cityGraftValue(t, a, "HasInventory") != 0 || cityGraftValue(t, a, "HasSpellbook") != 0 {
			t.Fatal("retained fields displaced current values or inactive grammar")
		}
		if cityGraftValue(t, a, "U49") != 7 || cityGraftRaw(t, a, "U154")[179] != 0xe7 || cityGraftRaw(t, a, "U154")[10] != 33 || cityGraftRaw(t, a, "UA6")[23] != 0x63 {
			t.Fatal("mixed current/retained raw mask is wrong")
		}
		if cityGraftValue(t, p, "Money") != 999 || cityGraftValue(t, p, "F44") != 77 || cityGraftValue(t, p, "F58") != 95 || cityGraftRaw(t, p, "Raw10")[7] != 0xc1 || cityGraftRaw(t, p, "PRaw32")[3] != 0x9a || cityGraftRaw(t, p, "PRaw32")[31] != 2 {
			t.Fatal("Player current/residue ownership is wrong")
		}
		da, db := cityGraftRefs(t, a, "Diary")[0], cityGraftRefs(t, b, "Diary")[0]
		if da == 0 || da != db || len(doc.Objects) != 4 {
			t.Fatal("shared Diary was lost or duplicated")
		}
		d := &doc.Objects[da-1]
		if binary.LittleEndian.Uint32(cityGraftRaw(t, d, "Journal")) != 0x12345678 || binary.LittleEndian.Uint16(cityGraftRaw(t, d, "JournalWords")[2:]) != 0x1234 || cityGraftValue(t, d, "D2C") != cityGraftValue(t, p, "This") {
			t.Fatal("Diary arrays or owner reference were lost")
		}
		if binary.LittleEndian.Uint32(cityGraftRaw(t, a, "U158")[0x0c:]) != cityGraftValue(t, b, "Identity") || binary.LittleEndian.Uint32(cityGraftRaw(t, a, "U50")) != oldPlayerKey || cityGraftValue(t, a, "U64") != 0x01000000 {
			t.Fatal("key relocation touched opaque bytes or failed an actual edge")
		}
		for i := range doc.Objects {
			for _, field := range doc.Objects[i].Values {
				if (field.Name == "Identity" || field.Name == "This") && field.Value == 0x01000000 {
					t.Fatal("missing retained reference accidentally bound to a fresh object")
				}
			}
		}
		doc, err = sav.RemintDocumentKeys(doc)
		if err != nil {
			t.Fatal(err)
		}
		wire, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		doc, err = sav.DecodeDocumentData(wire)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestCurrentCityGraftReorderedRemovedActorAndReturn(t *testing.T) {
	source, current := cityGraftFixture(t, true), cityGraftFixture(t, false)
	s1 := cityGraftNamed(t, &source, "Companion")
	c1 := cityGraftNamed(t, &current, "Companion")
	for _, name := range []string{"Stage", "U5C", "U64", "U44", "U40"} {
		mustSetValue(&source.Objects[s1-1], name, 3)
	}
	cityGraftRaw(t, &source.Objects[s1-1], "U154")[179] = 0xee
	removed := cityGraftNamed(t, &source, "Leader")
	cityGraftRaw(t, &source.Objects[removed-1], "U154")[179] = 0xdd
	mustSetRefs(&source.Objects[s1-1], "U68", []uint16{removed})
	p := &current.Objects[current.Players[0]-1]
	for i := range p.Groups {
		for j := range p.Groups[i].RefSlots {
			if p.Groups[i].RefSlots[j].Name == "Actors" {
				slices.Reverse(p.Groups[i].RefSlots[j].Objects)
			}
		}
	}
	var err error
	current, _, err = sav.ReindexDocumentData(current)
	if err != nil {
		t.Fatal(err)
	}
	c1 = cityGraftNamed(t, &current, "Companion")
	doc, err := graftCurrentCityResidue(current, source, []currentCityActorGraft{{Source: s1, Current: c1, Returned: true}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := &doc.Objects[cityGraftNamed(t, &doc, "Companion")-1]
	for _, name := range []string{"Stage", "U5C", "U64", "U44", "U40"} {
		if cityGraftValue(t, a, name) != 0 {
			t.Fatalf("return did not clear %s", name)
		}
	}
	if cityGraftRefs(t, a, "U68")[0] != 0 || cityGraftRaw(t, a, "U154")[179] != 0xee || len(doc.Objects) != len(current.Objects) {
		t.Fatal("return residue or current population was lost")
	}
	newActor := &doc.Objects[cityGraftNamed(t, &doc, "Leader")-1]
	if cityGraftRaw(t, newActor, "U154")[179] != 0 {
		t.Fatal("equal-looking unbound new actor inherited removed actor residue")
	}
}
