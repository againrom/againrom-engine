package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestShortcutLayoutSecondCityProvenanceRecord(t *testing.T) {
	f, _, _, _ := secondBagFront(t, nil)
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	outer, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	engine := engineSaveDocument(t, raw)
	writeQuickSpellLeaf(&engine, sav.CityShortcuts{0, 1, -1, -1})
	engineBytes, err := sav.EncodeDocumentData(engine)
	if err != nil {
		t.Fatal(err)
	}
	outer, err = sav.StoreEngineDocument(outer, engineBytes)
	if err != nil {
		t.Fatal(err)
	}
	patched, err := sav.EncodeDocumentData(outer)
	if err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{InstallResources: f.InstallResources}
	cold.Campaign = resolved(Campaign{}, nil)
	if _, town, err := cold.RestoreOriginal(patched); err != nil || !town {
		t.Fatalf("RestoreOriginal: town=%v err=%v", town, err)
	}
	snap, _, err := cold.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	state, err := originalCityFromSnapshot(snap, cold.campaign())
	if err != nil {
		t.Fatal(err)
	}
	want, err := quickSpellsFromOriginalIndices([]int32{0, 1, -1, -1}, secondCampaignRules{}.shortcutLayout())
	if err != nil {
		t.Fatal(err)
	}
	if state.baselineQuickSpells != want {
		t.Fatalf("second city baseline = %v, want %v", state.baselineQuickSpells, want)
	}
}
