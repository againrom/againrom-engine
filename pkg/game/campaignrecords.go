package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
)

type campaignRecords struct {
	main     campaignProgressRecord
	children []campaignProgressRecord
}

type SnapshotCampaignRecords struct {
	Main     sav.CampaignRecord
	Children []sav.CampaignRecord
}

func newCampaignRecords(c Campaign, main int) campaignRecords {
	r := campaignRecords{main: progressRecordFromCampaign(c, main)}
	for _, mission := range candidateSides(c.Chapters[main]) {
		r.children = append(r.children, progressRecordFromCampaign(c, mission))
	}
	return r
}

func (r *campaignRecords) loadMain(c Campaign, main int, announced bool) {
	kept := r.children[:0]
	seen := make(map[int]bool)
	for _, child := range r.children {
		child.age++
		if child.age < 2 {
			kept = append(kept, child)
			seen[child.mission] = true
		}
	}
	r.children = kept
	r.main = progressRecordFromCampaign(c, main)
	r.main.announced = announced
	for _, mission := range candidateSides(c.Chapters[main]) {
		if !seen[mission] {
			r.children = append(r.children, progressRecordFromCampaign(c, mission))
			seen[mission] = true
		}
	}
}

func (r *campaignRecords) removeChild(mission int) bool {
	for i := range r.children {
		if r.children[i].mission == mission {
			copy(r.children[i:], r.children[i+1:])
			r.children = r.children[:len(r.children)-1]
			return true
		}
	}
	return false
}

func (t *Town) currentRecords() *campaignRecords {
	if t == nil {
		return nil
	}
	if t.progress != nil {
		return &t.progress.campaignRecords
	}
	return &t.records
}

func (r *campaignRecords) accepts(c Campaign, mission int) bool {
	return r != nil && (r.record(mission) != nil ||
		mission > r.main.mission && mission%mainMissionStride == 0 && containsMission(c.Main, mission))
}

func (t *Town) prepareOfferedRecord(mission int) bool {
	r := t.currentRecords()
	if r == nil || !r.accepts(t.camp, mission) {
		return false
	}
	if r.record(mission) == nil {
		if t.progress != nil {
			t.progress.loadMain(t.camp, mission, false)
		} else {
			t.main, t.selected = mission, mission
			t.records.loadMain(t.camp, mission, false)
		}
		t.refreshAvailable()
	}
	return true
}

func (t *Town) announceMission(mission int) {
	if record := t.currentRecords().record(mission); record != nil && !t.Done(mission) {
		record.announced = true
		t.available[mission] = true
	}
}

func (t *Town) refreshAvailable() {
	t.available = make(map[int]bool)
	r := t.currentRecords()
	for _, record := range append([]campaignProgressRecord{r.main}, r.children...) {
		if record.announced && !t.Done(record.mission) {
			t.available[record.mission] = true
		}
	}
}

func (t *Town) gateMission() int {
	r := t.currentRecords()
	if r == nil {
		return -1
	}
	if r.main.announced && !t.Done(r.main.mission) {
		return r.main.mission
	}
	for _, child := range r.children {
		if child.announced && !t.Done(child.mission) {
			return child.mission
		}
	}
	return -1
}

func (r *campaignRecords) snapshot() *SnapshotCampaignRecords {
	out := &SnapshotCampaignRecords{Main: r.main.savRecord(false)}
	for _, child := range r.children {
		out.Children = append(out.Children, child.savRecord(true))
	}
	return out
}

func validateSnapshotCampaignRecords(_ Campaign, s Snapshot) error {
	if s.CampaignState || s.CampaignRecords == nil {
		return nil
	}
	r := s.CampaignRecords
	if r.Main.Mission == 0 || int(r.Main.Mission) != s.MainMission {
		return fmt.Errorf("saved campaign records have invalid main mission %d", r.Main.Mission)
	}
	seen := map[uint32]bool{r.Main.Mission: true}
	for _, child := range r.Children {
		if child.Mission == 0 || seen[child.Mission] || child.Age >= 2 {
			return fmt.Errorf("saved campaign child %d is zero, duplicate or expired", child.Mission)
		}
		seen[child.Mission] = true
	}
	return nil
}

func (t *Town) restoreCampaignRecords(s Snapshot) {
	if s.CampaignRecords != nil && int(s.CampaignRecords.Main.Mission) == t.main {
		t.records.main = progressRecordFromSAV(s.CampaignRecords.Main)
		t.records.children = nil
		for _, child := range s.CampaignRecords.Children {
			t.records.children = append(t.records.children, progressRecordFromSAV(child))
		}
	} else {
		t.records = newCampaignRecords(t.camp, t.main)
		for _, mission := range s.Available {
			age := t.main/mainMissionStride - mission/mainMissionStride
			if t.records.record(mission) == nil && mission > 0 && mission%mainMissionStride != 0 && age >= 0 && age < 2 {
				r := progressRecordFromCampaign(t.camp, mission)
				r.age = age
				t.records.children = append(t.records.children, r)
			}
			t.prepareOfferedRecord(mission)
			if record := t.records.record(mission); record != nil {
				record.announced = true
			}
		}
	}
	for _, mission := range s.Won {
		t.records.removeChild(mission)
	}
	t.refreshAvailable()
}
