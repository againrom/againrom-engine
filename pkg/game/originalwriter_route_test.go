package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

type fakeOriginalCityDocument struct {
	roster  []sav.CityCharacter
	payload []byte
	data    sav.CityData
	err     error
	updates []sav.CityUpdate
}

func (document *fakeOriginalCityDocument) Roster() []sav.CityCharacter {
	return append([]sav.CityCharacter(nil), document.roster...)
}

func (document *fakeOriginalCityDocument) Data() sav.CityData { return document.data }

func (document *fakeOriginalCityDocument) Marshal(update sav.CityUpdate) ([]byte, error) {
	copy := update
	copy.Label = append([]byte(nil), update.Label...)
	copy.Characters = append([]sav.CityCharacterUpdate(nil), update.Characters...)
	if update.Campaign != nil {
		campaign := *update.Campaign
		copy.Campaign = &campaign
	}
	document.updates = append(document.updates, copy)
	if document.err != nil {
		return nil, document.err
	}
	return append([]byte(nil), document.payload...), nil
}

func originalCityRouteFixture(t *testing.T) (*FrontEnd, *fakeOriginalCityDocument) {
	t.Helper()
	raw, newFront := spellbookCitySource(t)
	f := newFront()
	if _, town, err := f.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("route fixture LOAD", town, err)
	}
	retained := f.originalCity.document
	data, ok := retained.(interface{ Data() sav.CityData })
	if !ok {
		t.Fatal("route fixture has no ordinary city document")
	}
	document := &fakeOriginalCityDocument{roster: retained.Roster(), data: data.Data(), payload: []byte("legacy writer must not be called")}
	f.originalCity.document = document
	f.worldMapCache = resolved(&worldMapAssets{data: &globalMapData{Missions: map[int]int{10: 0, 20: 1}, Objects: []globalMapObject{{Valid: true, Picture: "town"}, {Valid: true, Picture: "mission"}}}}, nil)
	f.Town.gold = 4321
	f.fame = SnapshotFame{Known: true, Time: 77}
	f.originalCity.captureSession(&f.CampaignSession)
	return f, document
}

func TestSaveSeamsAuthorsCurrentCityInConsecutiveSlots(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{Dir: t.TempDir()}, func() time.Time { return time.Unix(123, 0) })
	var first []byte
	for i, want := range []string{"game0000.sav", "game0001.sav"} {
		name, err := save(false)
		if err != nil || name != want {
			t.Fatal("current SAVE slot", name, want, err)
		}
		raw, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		if len(doc.Players) != 1 {
			t.Fatal("current city player population", doc.Players)
		}
		if got, err := savedStructureValue(&doc.Objects[doc.Players[0]-1], "Money"); err != nil || got != 4321 {
			t.Fatal("current purse not written", got, err)
		}
		if i == 0 {
			first = raw
		} else if !bytes.Equal(first, raw) {
			t.Fatal("same captured state produced different SAV bytes")
		}
		cold := &FrontEnd{InstallResources: f.InstallResources}
		if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
			t.Fatal("current city cold LOAD", town, err)
		}
		if cold.Town.Gold() != f.Town.Gold() || len(cold.Carried) != len(f.Carried) {
			t.Fatal("current city population or purse changed")
		}
		for at := range f.Carried {
			if cold.Carried[at].Name != f.Carried[at].Name || cold.Carried[at].KnownSpells != f.Carried[at].KnownSpells {
				t.Fatal("current character changed", at)
			}
		}
	}
	if len(document.updates) != 0 {
		t.Fatal("SAVE selected the retained document's writer")
	}
}

func TestSaveSeamsWritesChangedPartyToSAV(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	added := mapload.CloneParty(f.Carried[1:2])[0]
	added.ID, added.Name = "player:joined", "Joined"
	added.StartingHero, added.PlayerCharacter = false, false
	f.Carried = append(f.Carried, added)
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("changed party SAVE", name, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("changed party cold LOAD", town, err)
	}
	if len(cold.Carried) != 3 || cold.Carried[2].Name != "Joined" {
		t.Fatal("SAV lost joined member", cold.Carried)
	}
	if len(document.updates) != 0 {
		t.Fatal("changed roster selected retained writer")
	}
}

func TestSaveSeamsWritesSAVAfterSelectedMissionSessionState(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	f.Town.progress.markerSelected = true
	f.TownScreen().(*townScreen).worldSelectedOnce = map[int]bool{20: true}
	f.Offered = 20
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil || !IsOriginal(name) {
		t.Fatal("selected mission SAVE", name, err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("selected mission cold LOAD", town, err)
	}
	if cold.Offered != 20 {
		t.Fatal("current offered mission changed", cold.Offered)
	}
	if len(document.updates) != 0 {
		t.Fatal("selected mission used retained writer")
	}
}

func TestSaveSeamsUsesOneProducerAcrossProvenance(t *testing.T) {
	for _, kind := range []string{"native city", "unavailable city provenance", "mission"} {
		t.Run(kind, func(t *testing.T) {
			f, document := originalCityRouteFixture(t)
			onMap := kind == "mission"
			if onMap {
				f = currentPoolFixtureFront(t, 91, 92)
				if err := f.App("current native route").OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			} else if kind == "native city" {
				f.originalCity = nil
			} else {
				f.originalCity = &originalCitySaveState{unavailable: errors.New("unsupported old city document")}
			}
			dir := t.TempDir()
			save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
			name, err := save(onMap)
			if err != nil || !IsOriginal(name) {
				t.Fatal("current SAVE", name, err)
			}
			raw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				t.Fatal(err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			if onMap && (doc.World == nil || doc.Head.Mission != 10) {
				t.Fatal("current mission absent from SAV")
			}
			if len(document.updates) != 0 {
				t.Fatal("route used retained writer")
			}
		})
	}
}

func TestSaveSeamsRejectsInvalidCurrentStateWithoutPublishing(t *testing.T) {
	f, document := originalCityRouteFixture(t)
	document.err = errors.New("obsolete writer must not choose the route")
	f.Town.gold = -1
	before := mapload.CloneParty(f.Carried)
	dir := t.TempDir()
	save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	name, err := save(false)
	if name != "" || err == nil {
		t.Fatal("invalid current purse accepted", name, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 0 {
		t.Fatal("failed SAVE published a file", entries, err)
	}
	if !reflect.DeepEqual(before, f.Carried) || len(document.updates) != 0 {
		t.Fatal("failed SAVE changed state or chose another writer")
	}
}
