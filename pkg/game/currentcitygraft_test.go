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
	retained.Objects[1].Unit.RawA6[22] = 0xe5
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
	if cityGraftRaw(t, &doc.Objects[cityGraftNamed(t, &doc, "Companion")-1], "UA6")[22] != 0xe5 {
		t.Fatal("snapshot binding order displaced stable party identity")
	}
	s.Party[1].ID = "companion"
	doc, err = graftSnapshotCityResidue(current, s)
	if err != nil {
		t.Fatal("ambiguous optional source blocked current city", err)
	}
	if cityGraftRaw(t, &doc.Objects[cityGraftNamed(t, &doc, "Companion")-1], "UA6")[22] == 0xe5 {
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
		{"cleared", true, 0, 0x39a60000},
		{"changed", true, 321, 0x39a60141},
	} {
		t.Run(test.name, func(t *testing.T) {
			current := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "T08", Value: uint32(test.id)}}}
			source := sav.DocumentRecordData{Class: "Human", Values: []sav.DocumentValueData{{Name: "T08", Value: 0x39a63185}}}
			mergeCityActorResidue(&current, &source, currentCityActorGraft{}, unknownRecordSpans())
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

// Every loaded value is poisoned; the grafted document may differ from the
// constructed one only inside an unknown-meaning span.
func TestCurrentCityGraftTakesOnlyUnknownSpans(t *testing.T) {
	source, current := cityGraftFixture(t, true), cityGraftFixture(t, false)
	s1, s2 := cityGraftNamed(t, &source, "Companion"), cityGraftNamed(t, &source, "Leader")
	c1, c2 := cityGraftNamed(t, &current, "Companion"), cityGraftNamed(t, &current, "Leader")
	for i := range source.Objects {
		r := &source.Objects[i]
		for j := range r.Values {
			if r.Values[j].Name != "Identity" && r.Values[j].Name != "This" {
				r.Values[j].Value ^= 0x5a5a5a5a
			}
		}
		for j := range r.Raw {
			for k := range r.Raw[j].Bytes {
				r.Raw[j].Bytes[k] ^= 0x5a
			}
		}
	}
	doc, err := graftCurrentCityResidue(current, source,
		[]currentCityActorGraft{{Source: s1, Current: c1, RetainedHumanTails: true}, {Source: s2, Current: c2}},
		[]currentCityPlayerGraft{{Source: source.Players[0], Current: current.Players[0]}})
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Objects) != len(current.Objects) {
		t.Fatal("graft changed the constructed population")
	}
	spans := unknownRecordSpans()
	allowed := func(pattern string, at int) bool {
		for _, span := range spans[pattern] {
			if slices.Contains(span.offsets, at) {
				return true
			}
		}
		return false
	}
	grafted := 0
	for i := range doc.Objects {
		a, b := &doc.Objects[i], &current.Objects[i]
		for j := range a.Values {
			var x, y [4]byte
			binary.LittleEndian.PutUint32(x[:], a.Values[j].Value)
			binary.LittleEndian.PutUint32(y[:], b.Values[j].Value)
			for k := range x {
				if x[k] != y[k] {
					grafted++
					if !allowed(a.Class+".v."+a.Values[j].Name, k) {
						t.Fatalf("%s %s byte %d came from the loaded town", a.Class, a.Values[j].Name, k)
					}
				}
			}
		}
		for j := range a.Raw {
			for k := range a.Raw[j].Bytes {
				if a.Raw[j].Bytes[k] != b.Raw[j].Bytes[k] {
					grafted++
					if !allowed(a.Class+".r."+a.Raw[j].Name, k) {
						t.Fatalf("%s %s byte %d came from the loaded town", a.Class, a.Raw[j].Name, k)
					}
				}
			}
		}
		if !reflect.DeepEqual(a.Texts, b.Texts) || !reflect.DeepEqual(a.RefSlots, b.RefSlots) || !reflect.DeepEqual(a.Inline, b.Inline) {
			t.Fatalf("%s text, edge or inline record came from the loaded town", a.Class)
		}
	}
	if grafted == 0 {
		t.Fatal("no unknown span was grafted")
	}
	tail := cityGraftRaw(t, &doc.Objects[c2-1], "UA6")
	if tail[22] != cityGraftRaw(t, &current.Objects[c2-1], "UA6")[22] {
		t.Fatal("a loaded Human tail replaced the party's own")
	}
}
