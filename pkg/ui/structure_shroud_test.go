package ui

import (
	"image"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestStructureCrownSurvivesAnUnseenGroundCell(t *testing.T) {
	v := newStaticsViewer(t, staticsBundle(), true, true)
	plane := make([]byte, cliffW*cliffH)
	plane[cliffW+1] = FogVisible
	v.SetFog(plane, cliffW, cliffH)
	frame := staticsFrame(32, 32, 1)
	// The crown is one full cell above the ground row, inside explored art.
	pieces := []terrain.StructurePlacement{{Cell: image.Pt(1, 2), TopLeft: image.Pt(32, 32), Frame: frame}}
	for _, flat := range []bool{true, false} {
		v.SetFlat(flat)
		got := v.fogGateStructurePlacements(pieces)
		if len(got) != 1 || &got[0] != &pieces[0] {
			t.Fatal("fog discarded the crown before pixel masking")
		}
		if v.fogGateSack(1, 2) {
			t.Fatal("drawing an overhang disclosed its hidden interaction target")
		}
	}
}
