package game

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

func tavernTalkFigureSheet(c color.RGBA) []byte {
	pixels := make([]synth.Pixel256, data.PortraitW*data.PortraitH)
	for i := range pixels {
		pixels[i] = synth.Pixel256{Index: 1, Opaque: true}
	}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, c},
		Frames:  []synth.Frame256{{Width: data.PortraitW, Height: data.PortraitH, Pixels: pixels}},
	})
}

func tavernTalkArchives(t *testing.T) *Archives {
	t.Helper()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, MainArchive)
	graphicsPath := filepath.Join(dir, GraphicsArchive)
	mainBytes := synth.Archive([]synth.File{{
		Path: "text/inn/mercenary/npc03.txt",
		Data: []byte("<part=1,npc=22>\r\ncandidate words"),
	}, {Path: "text/inn/npc/npc09m30.txt", Data: []byte("<part=1,npc=9>\r\nstory words")}})
	graphicsBytes := synth.Archive([]synth.File{
		{Path: data.ItemFigureBasePath(data.FigureDirManFighter, 1),
			Data: tavernTalkFigureSheet(color.RGBA{R: 0xff})},
		{Path: data.ItemFigureBasePath(data.FigureDirWomanFighter, 2),
			Data: tavernTalkFigureSheet(color.RGBA{B: 0xff})},
	})
	if err := os.WriteFile(mainPath, mainBytes, 0o600); err != nil {
		t.Fatalf("write %s: %v", mainPath, err)
	}
	if err := os.WriteFile(graphicsPath, graphicsBytes, 0o600); err != nil {
		t.Fatalf("write %s: %v", graphicsPath, err)
	}
	fsys, err := vfs.Open([]string{mainPath, graphicsPath}, nil)
	if err != nil {
		t.Fatalf("open tavern Talk fixture: %v", err)
	}
	return &Archives{Containers: fsys}
}

// The candidate row is a blue female figure while the party speaker fallback
// is a red male figure. The expected modal is composed directly from the
// candidate's explicit figure address, independently of the town resolver.
func TestMercenaryTalkUsesTheSelectedCandidateFigure(t *testing.T) {
	f := shellFrontEnd()
	params := humansParams(30, 0, 2)
	params[16] = 3
	params[18] = 1
	humans := dbCollection{{}, {name: "NPC03_1", params: params}}
	f.Humans, f.Table.Humans = humans, humans
	f.Archives = tavernTalkArchives(t)
	f.Font = resolved(missionFont(), nil)
	f.NPCFaces = map[int32]data.NPCFace{22: {}}
	f.Carried[0].FigureDir = string(data.FigureDirManFighter)
	f.Carried[0].FigureFace = 1
	s := f.townUI
	s.room = roomTavern
	s.composeShopFaces()
	s.openMercenaryDialogue(3)

	got, ok := s.TownDialogue()
	if !ok || got == nil {
		t.Fatal("candidate dialogue did not compose")
	}
	layout, _, ok := s.townDialogueLayout()
	if !ok {
		t.Fatal("candidate dialogue layout is unavailable")
	}
	expected, _ := composeInventorySubject(f.Archives.Containers, 3, data.Equipment{},
		data.FigureDirWomanFighter, 2)
	player, _ := composeInventorySubject(f.Archives.Containers, 1, data.Equipment{},
		data.FigureDirManFighter, 1)
	want := ui.RenderNotice(layout, f.Font.Value(), "candidate words", expected.Figure)
	wrong := ui.RenderNotice(layout, f.Font.Value(), "candidate words", player.Figure)
	if !imagesEqual(got, want) {
		t.Fatal("candidate dialogue did not use the selected female figure")
	}
	if imagesEqual(got, wrong) {
		t.Fatal("candidate dialogue still used the party/player speaker fallback")
	}
}
