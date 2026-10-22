package game

import (
	"crypto/sha256"
	"os"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/ui"
)

func TestReleaseGeneratedMissionSAVViewBounds(t *testing.T) {
	for _, open := range []bool{true, false} {
		t.Run(map[bool]string{true: "open", false: "closed"}[open], func(t *testing.T) {
			generatedMissionViewBounds(t, open)
		})
	}
}

func generatedMissionViewBounds(t *testing.T, open bool) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	a := f.App("mission save view bounds")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.NewGameOpener(10, ui.ChargenResult{Name: "Camera", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})); err != nil {
		t.Fatal(err)
	}
	m := f.live.mission.state.Map
	edge := f.live.view.SaveApplication()
	edge.ViewX, edge.ViewY = float64(m.Width-16), float64(m.Height-26)
	edge.SpellBookOpen = open
	if err := f.live.view.RestoreSaveApplication(edge); err != nil {
		t.Fatal(err)
	}
	live := f.live.view.SaveApplication()
	if live.ViewX != edge.ViewX || live.ViewY != edge.ViewY {
		t.Fatalf("edge camera (%v,%v), want (%v,%v)", live.ViewX, live.ViewY, edge.ViewX, edge.ViewY)
	}
	before := f.live.world.Hash()
	path := generatedMissionSave(t, f, a)
	wire, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	x, err := readOriginalStateLeaf(&doc, "/View/X", 2)
	if err != nil {
		t.Fatal(err)
	}
	y, err := readOriginalStateLeaf(&doc, "/View/Y", 2)
	if err != nil {
		t.Fatal(err)
	}
	rows := 22
	if open {
		rows = 19
	}
	if x.Int32 != int32(m.Width-8-27) || y.Int32 != int32(m.Height-8-rows) {
		t.Fatalf("written SAV view (%d,%d), map %dx%d, live (%v,%v)", x.Int32, y.Int32, m.Width, m.Height, live.ViewX, live.ViewY)
	}
	after := f.live.view.SaveApplication()
	if after.ViewX != live.ViewX || after.ViewY != live.ViewY || f.live.world.Hash() != before {
		t.Fatal("SAVE changed live camera or World")
	}
	t.Logf("root %s; map %dx%d; camera (%v,%v); written View (%d,%d); SAV SHA256 %x; unchanged World %x", f.Archives.Root, m.Width, m.Height, live.ViewX, live.ViewY, x.Int32, y.Int32, sha256.Sum256(wire), before)
}
