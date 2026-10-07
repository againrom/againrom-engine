package game

import (
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

func clearTownTestGateLatches(town *Town) {
	town.available = make(map[int]bool)
	r := town.currentRecords()
	r.main.announced = false
	for i := range r.children {
		r.children[i].announced = false
	}
}

func gateRecordTown(t *testing.T, restored bool) *Town {
	t.Helper()
	c := townCampaign(t)
	c.Main = append(c.Main, 50)
	c.Chapters[50] = Chapter{Mission: 50}
	c.Offered = append(c.Offered, 39)
	ch := c.Chapters[30]
	ch.Shop = append(ch.Shop, 39)
	c.Chapters[30] = ch
	town := NewTown(c)
	town.Arrive()
	if restored {
		r := town.records.snapshot()
		p, err := campaignProgressFromSAV(c, sav.CampaignProjection{
			Main: r.Main, Children: r.Children, SelectedMission: 30,
			InnNPC: []uint16{22, 90}, InnMission: []uint16{30, 0}, ShopMission: []uint16{31, 39},
			MercenaryWorking: make([]uint16, 15), MercenaryPristine: make([]uint16, 15), MercenaryHired: make([]bool, 15),
		})
		if err != nil {
			t.Fatal(err)
		}
		town = newTownFromCampaignProgress(c, p)
	}
	return town
}

func TestGateCurrentRecordPriorityRetentionAndExpiry(t *testing.T) {
	for _, restored := range []bool{false, true} {
		t.Run(map[bool]string{false: "fresh", true: "restored"}[restored], func(t *testing.T) {
			town := gateRecordTown(t, restored)
			if len(town.Offers(TownTavern)) == 0 || town.gateMission() != -1 {
				t.Fatal("hearing/listing supplied a latch")
			}
			if mission, ok := town.Take(TownShop, 0); !ok || mission != 31 || town.gateMission() != 31 {
				t.Fatal("side acceptance", mission, ok, town.gateMission())
			}
			if _, ok := town.Take(TownTavern, 0); !ok || town.gateMission() != 30 {
				t.Fatal("main latch has no priority")
			}
			if _, ok := town.Won(30); !ok || town.gateMission() != 31 {
				t.Fatal("main load lost retained accepted child")
			}
			if r := town.currentRecords().record(31); r == nil || !r.announced || r.age != 1 {
				t.Fatal("retained record", r)
			}
			if town.currentRecords().main.announced || town.currentRecords().record(39).age != 1 {
				t.Fatal("new main latch or missing-section child age")
			}
			if _, ok := town.Won(40); !ok || town.currentRecords().record(31) != nil || town.currentRecords().record(39) != nil || town.gateMission() != -1 {
				t.Fatal("age2 records survived the second main load")
			}
		})
	}
}

func TestGateCompletedChildAndMissingSectionCandidate(t *testing.T) {
	for _, restored := range []bool{false, true} {
		town := gateRecordTown(t, restored)
		if _, ok := town.Take(TownShop, 0); !ok {
			t.Fatal("side acceptance")
		}
		if side, ok := town.Won(31); !side || !ok || town.gateMission() != -1 || town.currentRecords().record(31) != nil {
			t.Fatal("completed side remained live")
		}
		if mission, ok := town.Take(TownShop, 1); !ok || mission != 39 || town.gateMission() != 39 {
			t.Fatal("declared candidate without section was lost", mission, ok)
		}
		if town.currentRecords().record(39).payment != 0 {
			t.Fatal("missing section invented payment")
		}
	}
}

func TestGateSnapshotRecordsAndHistoricalDefault(t *testing.T) {
	town := gateRecordTown(t, false)
	town.Take(TownShop, 0)
	town.Won(30)
	var s Snapshot
	snapshotTown(town, &s)
	if err := validateSnapshotCampaignRecords(town.camp, s); err != nil {
		t.Fatal(err)
	}
	raw, err := EncodeSave(s, "gate record fixture")
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(raw)
	if err != nil {
		t.Fatal(err)
	}
	back := restoreTown(town.camp, decoded)
	if !reflect.DeepEqual(town.records, back.records) || back.gateMission() != 31 {
		t.Fatal("detached snapshot lost current records", back.records)
	}
	decoded.CampaignRecords = nil
	legacy := restoreTown(town.camp, decoded)
	if r := legacy.records.record(31); r == nil || r.age != 1 || !r.announced || legacy.gateMission() != 31 {
		t.Fatal("historical accepted previous-chapter default", r)
	}
	legacy.Won(40)
	if legacy.records.record(31) != nil {
		t.Fatal("historical default never expires")
	}
	corrupt := *s.CampaignRecords
	corrupt.Children = append([]sav.CampaignRecord(nil), corrupt.Children...)
	corrupt.Children[0].Age = 2
	s.CampaignRecords = &corrupt
	if err := validateSnapshotCampaignRecords(town.camp, s); err == nil {
		t.Fatal("expired wire record accepted")
	}
}

func TestGateSnapshotConfigurationAndLegacyCompatibility(t *testing.T) {
	c := townCampaign(t)
	t.Run("Arrive creates current records", func(t *testing.T) {
		town := NewTown(c)
		town.Arrive()
		var s Snapshot
		snapshotTown(town, &s)
		back := restoreTown(c, s)
		if town.currentMain() != 30 || !reflect.DeepEqual(town.records, back.records) || back.gateMission() != -1 {
			t.Fatal("Arrive record authority changed", town.records, back.records)
		}
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil)}}
		projection, err := nativeCampaignProjection(f, s)
		if err != nil || projection.Main.Mission != 30 || len(projection.Children) != 1 || projection.Children[0].Mission != 31 || projection.Children[0].Age != 0 || projection.Children[0].Announced {
			t.Fatal("current SAV record authority", projection, err)
		}
		if _, err := campaignProgressFromSAV(c, projection); err != nil {
			t.Fatal("current SAV projection cannot LOAD", err)
		}
	})
	t.Run("stale pre-town records follow resolved main", func(t *testing.T) {
		town := NewTown(c)
		town.open = true
		var s Snapshot
		snapshotTown(town, &s)
		if s.MainMission != 10 || s.CampaignRecords.Main.Mission != 10 {
			t.Fatal("fixture did not preserve stale prologue identity")
		}
		back := restoreTown(c, s)
		if back.currentMain() != 30 || back.records.main.mission != 30 || back.records.record(31) == nil || back.records.record(31).age != 0 || back.gateMission() != -1 {
			t.Fatal("resolved town retained stale records", back.records)
		}
	})
	t.Run("changed campaign keeps configured building rows", func(t *testing.T) {
		var s Snapshot
		snapshotTown(saveTown(t), &s)
		changed := Campaign{Main: []int{30}, Offered: []int{30}, Chapters: map[int]Chapter{30: {Mission: 30, Shop: []int{31}}}}
		if err := validateSnapshotCampaignRecords(changed, s); err != nil {
			t.Fatal("changed campaign rejected structurally valid state", err)
		}
		back := restoreTown(changed, s)
		if back.currentMain() != 30 || back.records.main.mission != 30 || back.records.record(31) == nil || len(back.Offers(TownShop)) != 1 || back.Offers(TownShop)[0].Mission != 31 || back.gold != s.Gold {
			t.Fatal("changed campaign did not use configured rows", back.records)
		}
	})
	t.Run("compatible record survives absent mission definition", func(t *testing.T) {
		town := gateRecordTown(t, false)
		town.Take(TownShop, 0)
		town.Won(30)
		var s Snapshot
		snapshotTown(town, &s)
		changed := town.camp
		changed.Side = []int{41}
		changed.Offered = []int{30, 40, 41}
		changed.Chapters = map[int]Chapter{40: changed.Chapters[40], 50: changed.Chapters[50]}
		if err := validateSnapshotCampaignRecords(changed, s); err != nil {
			t.Fatal("absent definition rejected current record", err)
		}
		back := restoreTown(changed, s)
		if !reflect.DeepEqual(town.records, back.records) || back.gateMission() != 31 {
			t.Fatal("compatible age/latch/payment rebuilt from definitions", back.records)
		}
	})
	t.Run("nil records restore accepted missing-section child", func(t *testing.T) {
		town := gateRecordTown(t, false)
		town.Take(TownShop, 1)
		town.Won(30)
		var s Snapshot
		snapshotTown(town, &s)
		s.CampaignRecords = nil
		back := restoreTown(town.camp, s)
		child := back.records.record(39)
		if child == nil || child.age != 1 || !child.announced || child.payment != 0 || back.gateMission() != 39 {
			t.Fatal("historical missing-section fallback", child)
		}
		back.Won(40)
		if back.records.record(39) != nil || back.gateMission() != -1 {
			t.Fatal("historical fallback did not expire")
		}
	})
}
