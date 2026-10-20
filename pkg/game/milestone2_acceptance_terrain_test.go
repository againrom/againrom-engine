package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

// Raw counts and values come from terrainCellsRaw, whose only decoder input
// is the first terrain structure's location. The expected population does
// not shrink if the decoder drops a record or misreports a payload offset.
// The compared planes are the carrier the LOAD builds with the one plane
// builder (the document's Blocks over the installed constructor planes), the
// carrier every first-game World holds from mission start.
func TestMilestone2TerrainBlockPlane(t *testing.T) {
	sessionCorpusAudit(t)
	filesChecked, recordsChecked, differences := 0, 0, 0
	widths := map[int]int{}
	var refused []milestone2ResumeRefusal
	milestone2CarrierCorpus(t, currentCarrierTerrainBlockPlane, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if !mf.present {
			return
		}
		r, err := milestone2TerrainFile(t, mf, fe)
		if err != nil {
			refused = append(refused, milestone2ResumeRefusal{mf.rel, err})
			return
		}
		filesChecked++
		recordsChecked += r.records
		widths[r.width]++
		for _, diff := range r.differences {
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

type milestone2CarrierFileResult struct {
	records, width int
	differences    []string
}

// milestone2TerrainFile compares one world-half file's raw Blocks with the
// planes its LOAD builds; an error is a LOAD refusal.
func milestone2TerrainFile(t *testing.T, mf milestone2File, fe *FrontEnd) (milestone2CarrierFileResult, error) {
	t.Helper()
	wants, _, err := terrainCellsRaw(mf.body, mf.f.World.BlocksOff)
	if err != nil {
		t.Fatal(err)
	}
	ms, _, err := loadOriginalMission(fe, mf.raw)
	if err != nil {
		return milestone2CarrierFileResult{}, err
	}
	planes, _ := ms.World.SavedCellPlanes()
	doc, err := sav.DecodeDocumentData(mf.raw)
	if err != nil {
		t.Fatal(err)
	}
	wants = terminalTerrainAcceptance(wants, doc)
	return milestone2CarrierFileResult{records: len(wants), width: ms.Map.Width, differences: terrainAcceptanceDifferences(wants, planes)}, nil
}
