package game

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
)

// The second game's seat of the Scenario bank: a fresh mission takes the
// campaign bank with its mission span cleared, an original mission file
// without an engine document takes the campaign tail's bank, and a LOAD whose
// engine document restored the World's live bank keeps that bank.
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
		want [1024]int32
	}{
		{"engine document", missionRequest{prepare: prepare}, &SnapshotSAVDocument{Document: &sav.DocumentData{}}, live},
		{"original file", missionRequest{prepare: prepare}, &SnapshotSAVDocument{}, campaign},
		{"fresh mission", missionRequest{}, nil, fresh},
	} {
		t.Run(c.name, func(t *testing.T) {
			town := &Town{second: newSecondCampaign()}
			town.second.bank = campaign
			c.r.town = town
			w := secondGameScriptWorld(t, nil, nil)
			if !w.SetROM2ScenarioState(live) {
				t.Fatal("world has no ROM2 scenario bank")
			}
			ms := &Mission{World: w, savedDocument: c.doc}
			if err := (secondCampaignRules{}).seatWorld(c.r, ms); err != nil {
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
