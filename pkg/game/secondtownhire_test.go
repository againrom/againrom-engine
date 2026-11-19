package game

import (
	"bytes"
	"encoding/binary"
	"maps"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// kaargHireTownFile is an original Kaarg town file holding identity 22 unnamed.
func kaargHireTownFile() []byte {
	tail := secondSavedTail(0, [2]uint32{2, placeTown})
	binary.LittleEndian.PutUint32(tail[12+4*768:], 30)
	party := []secondOriginalHero{secondFixtureHero(), identityMissionRecord()}
	return labelledContainer(sav.SecondGame.Magic, "kaarghired", secondSavedPartyOf(0, 44, party, false, 0, nil),
		append(shortcutsStore(), tail...))
}

type loadedCompanion struct {
	ID, Name     string
	CompanionNPC int
	Class        int32
	Hero         data.Hero
}

func companionOf(t *testing.T, name string, f *FrontEnd, id int) loadedCompanion {
	t.Helper()
	party := f.NextParty()
	i := slices.IndexFunc(party, func(m mapload.PartyMember) bool { return m.CompanionNPC == id })
	if i < 0 {
		t.Fatalf("%s: the party %v holds no identity %d", name, carriedIdentities(party, f.Table), id)
	}
	m := party[i]
	return loadedCompanion{ID: m.ID, Name: m.Name, CompanionNPC: m.CompanionNPC, Class: m.Class, Hero: m.Hero}
}

// The LOAD names the companion by his line; the resave lists him again.
func TestSecondTownFileNamesTheLoadedCompanion(t *testing.T) {
	cold := coldIdentityTown(t, kaargHireTownFile())
	if got := carriedIdentities(cold.Carried, cold.Table); !slices.Equal(got, []int{21, 22}) {
		t.Fatalf("the LOAD carries identities %v, want 21 and 22", got)
	}
	want := companionOf(t, "the LOAD", cold, 22)
	if want.ID != "npc:22" || want.Name != "Igles" {
		t.Fatalf("the LOAD lists the companion as %q named %q, want npc:22 named Igles", want.ID, want.Name)
	}
	saved := exportTown(t, cold)
	ids := slices.Sorted(maps.Keys(humanPacks(t, saved)))
	if !slices.Equal(ids, []uint32{21, 22}) {
		t.Fatalf("the resave holds Humans of identities %v, want 21 and 22", ids)
	}
	if names := humanNames(t, saved); names[22] != "" || names[21] != "Gildarius" {
		t.Fatalf("the resave stores names %v, want the hero's and none for the companion", names)
	}
	for _, file := range []struct {
		name string
		raw  []byte
	}{{"the resave", saved}, {"its records alone", withoutEngineDocument(t, saved)}} {
		if got := companionOf(t, file.name, coldIdentityTown(t, file.raw), 22); !reflect.DeepEqual(got, want) {
			t.Fatalf("%s lists\n%+v\nwant\n%+v", file.name, got, want)
		}
	}
}

func humanNames(t *testing.T, raw []byte) map[uint32]string {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint32]string{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Human" {
			continue
		}
		id, _ := savedStructureValue(r, "U148")
		for _, text := range r.Texts {
			if text.Name == "Name" {
				out[id&0xffff] = text.Value
			}
		}
	}
	return out
}

// A joined companion is stored unnamed and loads named.
func TestSecondTownSaveStoresTheCompanionUnnamed(t *testing.T) {
	raw := identityTownSave(t)
	if names := humanNames(t, raw); names[22] != "" || names[21] != "Aldor" {
		t.Fatalf("the SAVE stores names %v, want the hero's and none for the companion", names)
	}
	for _, file := range [][]byte{raw, withoutEngineDocument(t, raw)} {
		if got := companionOf(t, "the LOAD", coldIdentityTown(t, file), 22); got.Name != "Igles" {
			t.Fatalf("the LOAD names the companion %q, want Igles", got.Name)
		}
	}
}

// A town file without a companion resaves the same bytes.
func TestSecondTownFileWithoutACompanionRoundTrips(t *testing.T) {
	hero := secondFixtureHero()
	first := exportTown(t, coldIdentityTown(t, secondFixtureTown("town", hero, 44)))
	again := exportTown(t, coldIdentityTown(t, first))
	if !bytes.Equal(first, again) {
		t.Fatal("the second round writes other bytes")
	}
	cold := coldIdentityTown(t, withoutEngineDocument(t, again))
	if party := cold.NextParty(); len(party) != 1 || party[0].Name != hero.name {
		t.Fatalf("the records alone LOAD %d members, want the hero %q", len(party), hero.name)
	}
}
