//go:build sessioncorpusaudit

package game

import (
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
)

// Raw counts and values come from terrainCellsRaw, whose only decoder input
// is the first terrain structure's location. The expected population does
// not shrink if the decoder drops a record or misreports a payload offset.
func TestMilestone2TerrainBlockPlane(t *testing.T) {
	filesChecked, recordsChecked, differences := 0, 0, 0
	widths := map[int]int{}
	var refused []milestone2ResumeRefusal
	milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		wants, _, err := terrainCellsRaw(mf.body, mf.f.World.BlocksOff)
		if err != nil {
			t.Fatal(err)
		}
		ms, _, err := ResumeOriginalSave(fe.Archives.Containers, mf.raw, fe.Table, mapload.DifficultyNormal, nil, fe.Bodies)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		filesChecked++
		recordsChecked += len(wants)
		widths[ms.Map.Width]++
		planes, _ := ms.World.SavedCellPlanes()
		doc, err := sav.DecodeDocumentData(mf.raw)
		if err != nil {
			t.Fatal(err)
		}
		wants = terminalTerrainAcceptance(wants, doc)
		for _, diff := range terrainAcceptanceDifferences(wants, planes) {
			differences++
			t.Error(diff)
		}
	})
	t.Logf("terrain block plane: %d world-half file(s), %d raw delta record(s); map widths %v; %d refused", filesChecked, recordsChecked, widths, len(refused))
	milestone2LogRefusals(t, "terrain block plane", refused)
	t.Logf("terrain block plane: %d field or invariant differences", differences)
	if filesChecked == 0 || recordsChecked == 0 {
		t.Fatal("no terrain population compared")
	}
}
