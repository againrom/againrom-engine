package game

import (
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestConfiguredFutureMainRemainsPlayableAfterAcceptance(t *testing.T) {
	for _, building := range []TownBuilding{TownShop, TownTavern} {
		t.Run(map[TownBuilding]string{TownShop: "shop", TownTavern: "tavern"}[building], func(t *testing.T) {
			chapter := Chapter{Mission: 30, Inn: []int{30}, InnNPC: []int{22}}
			if building == TownShop {
				chapter.Shop = []int{40}
			} else {
				chapter.Inn, chapter.InnNPC = []int{40}, []int{22}
			}
			camp := Campaign{
				Main: []int{30, 40, 50}, Offered: []int{30, 40},
				Chapters: map[int]Chapter{30: chapter, 40: {Mission: 40}, 50: {Mission: 50}},
			}
			town := NewTown(camp)
			town.Arrive()
			if town.Chapter() != 30 || len(town.Available()) != 0 {
				t.Fatal("fresh chapter30 fixture", town.Chapter(), town.Available())
			}
			if mission, accepted := town.Take(building, 0); mission != 40 || !accepted {
				t.Fatal("configured main40 acceptance", mission, accepted)
			}
			town.registerOfferedMission(40)
			if !slices.Contains(town.Available(), 40) {
				t.Fatalf("accepted configured main40 was consumed but cannot pass the gate: main=%d available=%v remaining offers=%v", town.Chapter(), town.Available(), town.Offers(building))
			}
		})
	}
}

func configuredGateCampaign(building TownBuilding) Campaign {
	ch := Chapter{Mission: 30, Inn: []int{30}, InnNPC: []int{22}, School: []int{31, 39}}
	if building == TownShop {
		ch.Shop = []int{40}
	} else {
		ch.Inn = []int{40}
	}
	return Campaign{
		Main: []int{30, 40, 50}, Side: []int{31, 39, 41}, Offered: []int{30, 31, 39, 40, 41},
		Chapters: map[int]Chapter{30: ch, 31: {Mission: 31, Payment: 7}, 39: {Mission: 39},
			40: {Mission: 40, Payment: 23, School: []int{41}}, 41: {Mission: 41}, 50: {Mission: 50}},
	}
}

func configuredGateProjection(t *testing.T, town *Town) sav.CampaignProjection {
	t.Helper()
	if town.progress != nil {
		return town.progress.projection()
	}
	var s Snapshot
	snapshotTown(town, &s)
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(town.camp, nil)}}
	p, err := nativeCampaignProjection(f, s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestConfiguredMainRecordAcceptanceAndCompatibility(t *testing.T) {
	for _, building := range []TownBuilding{TownShop, TownTavern} {
		for _, restored := range []bool{false, true} {
			t.Run(map[TownBuilding]string{TownShop: "shop", TownTavern: "tavern"}[building]+map[bool]string{false: "/fresh", true: "/restored"}[restored], func(t *testing.T) {
				c := configuredGateCampaign(building)
				town := NewTown(c)
				town.Arrive()
				if restored {
					p, err := campaignProgressFromSAV(c, configuredGateProjection(t, town))
					if err != nil {
						t.Fatal("unaccepted future main candidate cannot LOAD", err)
					}
					town = newTownFromCampaignProgress(c, p)
				}
				town.gold = 100
				town.announceMission(30)
				town.announceMission(31)
				town.currentRecords().record(39).age = 1
				if m, ok := town.Take(building, 0); m != 40 || !ok || !town.registerOfferedMission(m) {
					t.Fatal("future main handover", m, ok)
				}
				r := town.currentRecords()
				if town.currentMain() != 40 || !r.main.announced || town.gateMission() != 40 || town.selectedMission() != 40 || slices.Contains(town.Available(), 30) {
					t.Fatal("future main has no exclusive current record/latch/selection", town.currentMain(), r, town.Available())
				}
				if child := r.record(31); child == nil || child.age != 1 || !child.announced || r.record(39) != nil || r.record(41).age != 0 {
					t.Fatal("main-load retention/expiry changed", r)
				}
				if !town.taken[offerRef{30, building, 0}] || len(town.taken) != 1 || town.Gold() != 100 {
					t.Fatal("handover lost provenance or paid completion", town.taken, town.Gold())
				}
				before := r.snapshot()
				if !town.registerOfferedMission(40) || !reflect.DeepEqual(before, town.currentRecords().snapshot()) || len(town.taken) != 1 || town.Gold() != 100 {
					t.Fatal("repeated registration reloads/consumes/pays")
				}
				var s Snapshot
				snapshotTown(town, &s)
				raw, err := EncodeSave(s, "configured main")
				if err != nil {
					t.Fatal(err)
				}
				decoded, _, err := DecodeSave(raw)
				if err != nil {
					t.Fatal(err)
				}
				back := restoreTown(c, decoded)
				if !reflect.DeepEqual(before, back.currentRecords().snapshot()) || back.gateMission() != 40 || back.Gold() != 100 || !back.taken[offerRef{30, building, 0}] {
					t.Fatal("additive continuation loses record or provenance")
				}
				projection := configuredGateProjection(t, back)
				decodedCampaign := autoGetCampaign(t, originalSaveWithCampaign(t, 0, projection))
				p, err := campaignProgressFromSAV(c, decodedCampaign)
				if err != nil {
					t.Fatal("emitted campaign cannot LOAD", err)
				}
				cold := newTownFromCampaignProgress(c, p)
				if cold.gateMission() != 40 || p.main.payment != 23 || p.record(31).age != 1 {
					t.Fatal("SAV record continuation changed", p)
				}
				if _, ok := back.Won(40); !ok || back.Gold() != 123 || back.currentRecords().record(31) != nil {
					t.Fatal("completion reward or next main age pass", back.Gold(), back.currentRecords())
				}
				if _, ok := back.Won(40); ok || back.Gold() != 123 {
					t.Fatal("completion paid twice")
				}
			})
		}
	}
}

func TestConfiguredMainRecordInvalidAndHistoricalControls(t *testing.T) {
	c := configuredGateCampaign(TownShop)
	for _, invalid := range []int{20, 60, 0} {
		ch := c.Chapters[30]
		ch.Shop = []int{invalid}
		c.Chapters[30] = ch
		town := NewTown(c)
		town.Arrive()
		before := town.records.snapshot()
		if _, ok := town.Take(TownShop, 0); ok || town.registerOfferedMission(invalid) || len(town.taken) != 0 || len(town.Available()) != 0 || !reflect.DeepEqual(before, town.records.snapshot()) {
			t.Fatal("invalid mission consumed or changed campaign", invalid)
		}
	}
	c = configuredGateCampaign(TownShop)
	legacy := restoreTown(c, Snapshot{Open: true, MainMission: 30, SelectedMission: 40, Available: []int{31, 40}, Taken: []SnapshotOffer{{Chapter: 30, Building: int(TownShop), Index: 0}}})
	if legacy.currentMain() != 40 || legacy.gateMission() != 40 || legacy.currentRecords().record(31).age != 1 || !legacy.taken[offerRef{30, TownShop, 0}] {
		t.Fatal("historical accepted future main lost", legacy.currentRecords(), legacy.Available())
	}
}
