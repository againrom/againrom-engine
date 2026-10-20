package game

import "againrom/pkg/formats/sav"

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

// missionEndTravel is the travel arm of a mission end (SAV-1329) for a
// restored campaign whose AutoGetMission is not -1 and not above the current
// main mission. It latches the main record's announce flag, selects that
// record and answers its mission; it completes nothing. A higher value and -1
// answer false and take the ordinary completion, as does the win of the
// campaign's terminal mission, whose ending precedes the arm.
func (t *Town) missionEndTravel(terminal bool) (int, bool) {
	if terminal || t == nil || t.progress == nil {
		return 0, false
	}
	p := t.progress
	request := int64(int32(p.autoGet))
	if request == -1 || request > int64(p.main.mission) {
		return 0, false
	}
	p.main.announced = true
	p.selected = p.main.mission
	t.refreshAvailable()
	return p.main.mission, true
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
