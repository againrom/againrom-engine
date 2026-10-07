package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
)

func TestMissionCurrentMapPointSurvivesTwoSAVCycles(t *testing.T) {
	for _, first := range []bool{false, true} {
		t.Run(fmt.Sprint(first), func(t *testing.T) {
			f, app := itemMutationOpen(t, 1, false)
			if f.Town == nil || f.Town.progress == nil {
				t.Fatal("fixture has no current campaign")
			}
			f.Town.progress.firstMapPoint = first
			for cycle := 0; cycle < 2; cycle++ {
				s := snapshotCurrentObjects(t, f)
				if !s.CampaignState || s.Campaign.FirstMapPoint != first {
					t.Fatal("current campaign lost map-point value", cycle)
				}
				raw, err := f.ExportCurrentSave(s, "current map point")
				if err != nil {
					t.Fatal(err)
				}
				doc, err := sav.DecodeDocumentData(raw)
				if err != nil {
					t.Fatal(err)
				}
				var want uint32
				if first {
					want = 1
				}
				if doc.Campaign.Scalars[4] != want {
					t.Fatalf("cycle %d: ordinary map-point word %d, want %d", cycle, doc.Campaign.Scalars[4], want)
				}
				f, app = itemMutationCheckpoint(t, f, app, 1)
				f.live.tick()
			}
		})
	}
}
