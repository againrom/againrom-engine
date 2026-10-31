package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// The second game's seat of the Scenario bank: a fresh mission takes the
// campaign bank with its mission span cleared, an original mission file
// without an engine document takes the campaign tail's bank, and a LOAD whose
// engine document restored the World's live bank keeps that bank. The seat
// reads the World's loaded-record carrier, not the mission's record.
func TestSecondSeatWorldKeepsTheEngineDocumentsLiveBank(t *testing.T) {
	var campaign, live [1024]int32
	campaign[760], campaign[768] = 5, 10
	live[754], live[760], live[768] = 1, 5, 10
	fresh := campaign
	fresh[760] = 0
	prepare := func(*alm.Map) (*originalFog, error) { return nil, nil }
	for _, c := range []struct {
		name string
		r    missionRequest
		doc  *SnapshotSAVDocument
		kind sim.LoadedRecord
		want [1024]int32
	}{
		{"engine document", missionRequest{prepare: prepare}, &SnapshotSAVDocument{Document: &sav.DocumentData{}}, sim.LoadedDocument, live},
		{"original file", missionRequest{prepare: prepare}, &SnapshotSAVDocument{}, sim.LoadedWithoutDocument, campaign},
		{"fresh mission", missionRequest{}, nil, sim.NoLoadedRecord, fresh},
		{"engine document in the World, none in the record", missionRequest{prepare: prepare}, nil, sim.LoadedDocument, live},
		{"no document in the World, one in the record", missionRequest{prepare: prepare}, &SnapshotSAVDocument{Document: &sav.DocumentData{}}, sim.LoadedWithoutDocument, campaign},
	} {
		t.Run(c.name, func(t *testing.T) {
			town := &Town{second: newSecondCampaign()}
			town.second.bank = campaign
			c.r.town = town
			w := secondGameScriptWorld(t, nil, nil)
			if !w.SetROM2ScenarioState(live) {
				t.Fatal("world has no ROM2 scenario bank")
			}
			w.SetLoadedRecord(c.kind)
			ms := &Mission{World: w, savedDocument: c.doc}
			if err := seatWorldCarriers(entrySeat, carrierSeat{r: c.r, rules: secondCampaignRules{}, ms: ms}); err != nil {
				t.Fatal(err)
			}
			got, _ := w.ROM2ScenarioState()
			for _, i := range []int{754, 760, 768} {
				if got[i] != c.want[i] {
					t.Fatalf("bank[%d] = %d, want %d", i, got[i], c.want[i])
				}
			}
			if town.second.bank != campaign {
				t.Fatal("seat changed the campaign bank")
			}
		})
	}
}
