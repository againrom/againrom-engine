package game

import (
	"encoding/json"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func savePointCity(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentTrainingCity(t)
	f.Archives = &Archives{Containers: worldMapFixtureFS(t)}
	f.worldMapCache = lazy[*worldMapAssets]{}
	return f
}

func TestSavePointPendingReturnRejectsEveryProducer(t *testing.T) {
	f := savePointCity(t)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	pending := s
	pending.WorldMapReturn = &SnapshotMapReturn{Mission: 30, Shown: 1}
	if _, err := f.ExportCurrentSave(pending, label); err == nil {
		t.Fatal("producer serialized pending return as a third save point")
	}
	town := f.TownScreen().(*townScreen)
	town.beginWorldMapReturn(30)
	if town.worldMap == nil || town.worldMap.returnMission == 0 {
		t.Fatal("pending fixture did not reach homeward travel")
	}
	party, gold := mapload.CloneParty(f.Carried), f.Town.Gold()
	if _, _, err := f.Snapshot(false); err == nil {
		t.Fatal("Snapshot accepted pending return")
	}
	if _, _, err := f.Snapshot(true); err == nil {
		t.Fatal("mission request bypassed pending return")
	}
	store := SaveStore{Dir: t.TempDir()}
	save, _, _ := f.SaveSeams(store, OriginalStore{}, nil)
	if _, err := save(false); err == nil {
		t.Fatal("automatic SAVE accepted pending return")
	}
	if _, err := f.SaveDialogSeams(store, OriginalStore{}).Prepare(ui.SaveRequest{Directory: store.Dir, Name: "pending", Format: ui.SaveSAV}); err == nil {
		t.Fatal("named SAVE accepted pending return")
	}
	if !reflect.DeepEqual(f.Carried, party) || f.Town.Gold() != gold || town.worldMap.returnMission != 30 {
		t.Fatal("refused SAVE mutated current return")
	}
	town.arriveWorldMap()
	if _, err := save(false); err != nil {
		t.Fatal("resolved city SAVE failed", err)
	}
}

func TestSavePointLegacyPendingSAVResolvesOnceAndKeepsNextAction(t *testing.T) {
	f := savePointCity(t)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	control := savePointCity(t)
	if _, _, err := control.RestoreOriginal(raw); err != nil {
		t.Fatal(err)
	}
	controlScreen := control.TownScreen().(*townScreen)
	controlScreen.beginWorldMapReturn(30)
	controlScreen.arriveWorldMap()
	want, _, err := control.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw = savePointLegacySAV(t, raw, 30)
	cold := savePointCity(t)
	if _, city, err := cold.RestoreOriginal(raw); err != nil || !city {
		t.Fatal("legacy pending LOAD", city, err)
	}
	screen := cold.TownScreen().(*townScreen)
	if !screen.AtTownSquare() || screen.worldMap != nil && screen.worldMap.returnMission != 0 {
		t.Fatal("legacy pending LOAD restored a third save point")
	}
	after, label, err := cold.Snapshot(false)
	if err != nil || after.WorldMapReturn != nil {
		t.Fatal("resolved city snapshot", after.WorldMapReturn, err)
	}
	if !reflect.DeepEqual(after.Party, want.Party) || after.Gold != want.Gold || !reflect.DeepEqual(after.Campaign, want.Campaign) || !reflect.DeepEqual(after.Documents, want.Documents) {
		t.Fatal("legacy return changed party, purse, journal or campaign")
	}
	for range 3 {
		screen.WorldMapTick()
	}
	if cold.Town.Gold() != s.Gold {
		t.Fatal("resolved return paid twice")
	}
	raw, err = cold.ExportCurrentSave(after, label)
	if err != nil {
		t.Fatal("resolved legacy SAVE", err)
	}
	resumed := savePointCity(t)
	if _, city, err := resumed.RestoreOriginal(raw); err != nil || !city {
		t.Fatal("resolved city cold LOAD", city, err)
	}
	screen = resumed.TownScreen().(*townScreen)
	if !screen.AtTownSquare() || !screen.CanSave() || resumed.Town.Gold() != want.Gold {
		t.Fatal("second LOAD resumed travel or changed purse")
	}
	hero := resumed.Carried[0]
	price := memberSchoolPrice(hero, 1)
	screen.room, screen.schoolCell, screen.shopMember = roomSchool, 0, 0
	screen.TownSurfaceClick(ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}, false)
	if resumed.Town.Gold() != s.Gold-price {
		t.Fatal("next school action after pending LOAD failed")
	}
}

func savePointLegacySAV(t *testing.T, raw []byte, mission int) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	a.WorldMapReturn = &SnapshotMapReturn{Mission: mission, Shown: 1}
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}
