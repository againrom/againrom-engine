package game

import (
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestReleaseTownSaveKeepsClearedMissionCellAndCampaignObject(t *testing.T) {
	_, original := groundCorpusFile(t, "2027-09-07/game0058.sav", "61746e37c590f5e486a03f1bcbe29a2a19275ad69565af71f95cc8dc7c5ababc")
	leadCell := func(raw []byte) mapload.Cell {
		t.Helper()
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		party, err := file.Party()
		if err != nil {
			t.Fatal(err)
		}
		for _, member := range party {
			if member.Hero {
				return mapload.Cell{X: int32(member.Col()), Y: int32(member.Row())}
			}
		}
		t.Fatal("town SAV has no lead Human")
		return mapload.Cell{}
	}
	mainObject := func(raw []byte) uint32 {
		t.Helper()
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		campaign, present, err := file.Campaign()
		if err != nil || !present {
			t.Fatalf("town SAV campaign: present=%t error=%v", present, err)
		}
		return campaign.Main.MapObject
	}
	if got := leadCell(original); got != (mapload.Cell{X: 16, Y: 12}) {
		t.Fatalf("source lead cell=%v, want (16,12)", got)
	}
	if got := mainObject(original); got != 0 {
		t.Fatalf("source main map object=%d, want 0", got)
	}
	open := func(raw []byte) *FrontEnd {
		t.Helper()
		front := releaseFront(t)
		mission, town, err := front.RestoreOriginal(raw)
		if err != nil || !town || mission != nil {
			t.Fatalf("town LOAD: town=%t mission=%t error=%v", town, mission != nil, err)
		}
		return front
	}
	check := func(front *FrontEnd, stage string) {
		t.Helper()
		state, _, err := front.Snapshot(false)
		if err != nil {
			t.Fatal(err)
		}
		if !state.CampaignState || state.Campaign.Main.MapObject != 0 {
			t.Errorf("%s campaign: state=%t main object=%d, want 0", stage, state.CampaignState, state.Campaign.Main.MapObject)
		}
		for _, member := range state.Party {
			if member.Name == "Danath" {
				if member.Saved == nil || member.Saved.Cell != (mapload.Cell{}) {
					t.Errorf("%s Danath saved cell=%v, want (0,0)", stage, member.Saved)
				}
				return
			}
		}
		t.Fatal("town snapshot has no Danath")
	}
	first := open(original)
	check(first, "original LOAD")
	dir := t.TempDir()
	prepared, err := first.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{
		Directory: dir, Name: "town position", Format: ui.SaveSAV, OnMap: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := prepared.Commit(true); err != nil {
		t.Fatal(err)
	}
	written, err := os.ReadFile(filepath.Join(dir, "town position.sav"))
	if err != nil {
		t.Fatal(err)
	}
	if got := leadCell(written); got != (mapload.Cell{}) {
		t.Errorf("written lead cell=%v, want (0,0)", got)
	}
	if got := mainObject(written); got != 0 {
		t.Errorf("written main map object=%d, want 0", got)
	}
	check(open(written), "cold LOAD")
}
