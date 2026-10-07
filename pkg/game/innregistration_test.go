package game

import (
	"fmt"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func innRegistrationApp(t *testing.T, restored bool) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f, app, s := dialogueKeysApp(t, 30)
	c := f.Campaign.Value()
	c.Side = []int{31, 41}
	c.Chapters[30] = Chapter{Mission: 30, Inn: []int{31, 41, 31, 0}, InnNPC: []int{22, 23, 24, 90}}
	f.Campaign = resolved(c, nil)
	f.Town = NewTown(c)
	f.Town.Arrive()
	if restored {
		p := campaignProjectionAt(c, 30)
		p.InnMission, p.InnNPC = []uint16{31, 41, 31, 0}, []uint16{22, 23, 24, 90}
		p.Children = []sav.CampaignRecord{{Mission: 31}, {Mission: 41}}
		progress, err := campaignProgressFromSAV(c, p)
		if err != nil {
			t.Fatal(err)
		}
		f.Town = newTownFromCampaignProgress(c, progress)
	}
	f.Archives.Containers = townTextFS(t, []synth.File{
		{Path: "text/inn/npc/npc22m31.txt", Data: []byte("<part=1>\r\nfirst")},
		{Path: "text/inn/npc/npc23m41.txt", Data: []byte("<part=1>\r\nother")},
		{Path: "text/inn/npc/npc24m31.txt", Data: []byte("<part=1>\r\nduplicate")},
		{Path: "text/inn/npc/npc90m30.txt", Data: []byte("<part=1>\r\nsentinel")},
	})
	s.worldMap = &worldMapState{assets: &worldMapAssets{data: &globalMapData{
		Missions: map[int]int{31: 0, 41: 1},
		Objects:  []globalMapObject{{Valid: true, Picture: "marker"}, {Valid: true, Picture: "nothing"}},
	}}}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	return f, app, s
}

func queuedInnMissions(s *townScreen) []int {
	return append([]int(nil), s.innQueue...)
}

func TestInnRegistersHeardIdentitiesInOrder(t *testing.T) {
	for _, restored := range []bool{false, true} {
		for _, vector := range []struct {
			npcs, pending []int
			remaining     []TownOffer
			selected      int
		}{
			{[]int{24}, []int{31}, []TownOffer{{Index: 1, Mission: 41, NPC: 23}, {Index: 2, Mission: 31, NPC: 24}, {Index: 3, NPC: 90}}, 31},
			{[]int{24, 24}, []int{31, 31}, []TownOffer{{Index: 1, Mission: 41, NPC: 23}, {Index: 3, NPC: 90}}, 31},
			{[]int{24, 23, 24}, []int{31, 41, 31}, []TownOffer{{Index: 3, NPC: 90}}, 31},
			{[]int{90, 24, 90}, []int{31}, []TownOffer{{Index: 1, Mission: 41, NPC: 23}, {Index: 2, Mission: 31, NPC: 24}, {Index: 3, NPC: 90}}, 31},
		} {
			t.Run(fmt.Sprintf("restored=%v/pending=%v", restored, vector.npcs), func(t *testing.T) {
				f, app, s := innRegistrationApp(t, restored)
				gold := f.Town.Gold()
				for _, npc := range vector.npcs {
					if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", npc)); err != nil {
						t.Fatal(err)
					}
					if s.room != roomTalk {
						t.Fatal("speaker did not open dialogue", npc, s.room)
					}
					dialogueKeyPress(t, app, s, "click")
					if s.room != roomTavern {
						t.Fatal("dialogue did not close to inn", npc, s.room)
					}
				}
				if got := queuedInnMissions(s); !reflect.DeepEqual(got, vector.pending) {
					t.Errorf("heard identities = %v, want %v", got, vector.pending)
				}
				if len(f.Town.Available()) != 0 || len(f.Town.Offers(TownTavern)) != 4 {
					t.Fatal("registration occurred before exit")
				}
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if got := f.Town.Offers(TownTavern); !reflect.DeepEqual(got, vector.remaining) {
					t.Errorf("remaining paired offers = %+v, want %+v", got, vector.remaining)
				}
				if got := f.Town.selectedMission(); got != vector.selected {
					t.Errorf("selected identity = %d, want %d", got, vector.selected)
				}
				if f.Town.Gold() != gold || len(s.innQueue) != 0 {
					t.Error("exit changed gold or retained pending queue")
				}
				if !s.worldSelectedOnce[31] || s.worldSelectedOnce[41] {
					t.Errorf("marker identities = %v, want picture-bearing 31 only", s.worldSelectedOnce)
				}
				for _, identity := range vector.pending {
					if !containsMission(f.Town.Available(), identity) {
						t.Errorf("identity %d was not announced", identity)
					}
					if restored && !f.Town.progress.record(identity).announced {
						t.Errorf("record %d announce latch remains clear", identity)
					}
				}
			})
		}
	}
}

func TestInnRegistrationRunsAfterLastPairAndKeepsCurrentMain(t *testing.T) {
	f, _, _ := innRegistrationApp(t, true)
	p := f.Town.progress
	p.innMission, p.innNPC = []int{0}, []int{90}
	p.main.addHero = []int{12}
	p.main.shopMin, p.main.shopMax = 2, 7
	p.selected = 41
	p.markers = []campaignProgressMarker{{value: 31, picture: "retained", field0: 9, field1: 12}}
	markers := append([]campaignProgressMarker(nil), p.markers...)
	if !f.Town.registerInnMission(31) || p.selected != 31 || !p.record(31).announced {
		t.Fatal("registration with no pair omitted selection or announcement", p.selected, p.record(31))
	}
	p.record(31).announced = false
	p.selected = 41
	if !f.Town.registerInnMission(31) || p.selected != 31 || !p.record(31).announced {
		t.Fatal("repeated registration with no pair omitted its effects")
	}
	before := p.main
	before.announced = true
	if !f.Town.registerInnMission(30) || p.selected != 30 || !reflect.DeepEqual(p.main, before) {
		t.Fatal("current-main registration reloaded or replaced the current record", p.main, before)
	}
	if !p.firstMapPoint || p.markerSelected || !reflect.DeepEqual(p.markers, markers) || !reflect.DeepEqual(p.innMission, []int{0}) || !reflect.DeepEqual(p.innNPC, []int{90}) {
		t.Fatal("registration changed map-entry policy, retained marker payload, or sentinel pair")
	}
}

func TestInnQueueCallsEveryIdentityInOrder(t *testing.T) {
	for _, pending := range [][]int{{31}, {31, 31}, {31, 41, 31}, {0, 31}} {
		var calls []int
		registerInnMissions(pending, func(identity int) { calls = append(calls, identity) })
		if !reflect.DeepEqual(calls, pending) {
			t.Fatalf("registration calls = %v, want %v", calls, pending)
		}
	}
}
