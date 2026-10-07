package game

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// stockSpeakerLayerSheet is a weapon layer that paints a green block inside the
// dialogue's default 72x96 window of a 160x240 canvas, so a dialogue drawing a
// figure that wears it differs from one drawing the bare sheet.
func stockSpeakerLayerSheet() []byte {
	pixels := make([]synth.Pixel256, data.PortraitW*data.PortraitH)
	for y := 20; y < 30; y++ {
		for x := 40; x < 80; x++ {
			pixels[y*data.PortraitW+x] = synth.Pixel256{Index: 1, Opaque: true}
		}
	}
	return synth.Sheet256(synth.Sheet256Options{
		Palette: []color.RGBA{{}, {G: 0xff}},
		Frames:  []synth.Frame256{{Width: data.PortraitW, Height: data.PortraitH, Pixels: pixels}},
	})
}

// stockSpeakerArchives carries the two base sheets a tavern speaker can draw
// (a blue woman fighter, face 2; a green man fighter, face 27), the party
// member's red man fighter, the stock unit's weapon layer and one inn
// conversation whose only speaker is npc90.
func stockSpeakerArchives(t *testing.T, layerPath string) *Archives {
	t.Helper()
	dir := t.TempDir()
	mainPath := filepath.Join(dir, MainArchive)
	graphicsPath := filepath.Join(dir, GraphicsArchive)
	mainBytes := synth.Archive([]synth.File{{
		Path: "text/inn/npc/npc90m30.txt",
		Data: []byte("<part=1,npc=90>\r\nstock words"),
	}})
	graphicsBytes := synth.Archive([]synth.File{
		{Path: data.ItemFigureBasePath(data.FigureDirManFighter, 1), Data: tavernTalkFigureSheet(color.RGBA{R: 0xff})},
		{Path: data.ItemFigureBasePath(data.FigureDirWomanFighter, 2), Data: tavernTalkFigureSheet(color.RGBA{B: 0xff})},
		{Path: data.ItemFigureBasePath(data.FigureDirManFighter, 27), Data: tavernTalkFigureSheet(color.RGBA{G: 0x80})},
		{Path: layerPath, Data: stockSpeakerLayerSheet()},
	})
	if err := os.WriteFile(mainPath, mainBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(graphicsPath, graphicsBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{mainPath, graphicsPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Archives{Containers: fsys}
}

// TestTavernDialogueDressesTheStockUnitItsSpeakerAnswersFor: the tavern's client
// actors are the stock mercenary units, so a Human record one of them passes on
// sex, class and face draws that unit wearing its own set (DLG-SPEAKER-023,
// TAVERN-TALKPIC-016). A Human record no unit passes, and every speaker of the
// shop or the school, stays the record's bare sheet (DLG-SPEAKER-022).
func TestTavernDialogueDressesTheStockUnitItsSpeakerAnswersFor(t *testing.T) {
	f := shellFrontEnd()
	params := humansParams(30, 0, 2)
	params[16], params[18] = 3, 1
	cells := make([]string, 10)
	cells[0] = "Short Sword"
	humans := dbCollection{{}, {name: "NPC10_1", params: params, strings: cells}}
	weapons := dbCollection{{}, {name: "Short Sword", params: chargenWeaponParams(data.SkillBlade)}}
	f.Humans, f.Table.Humans = humans, humans
	f.Table.Weapons = weapons
	f.Table.Shapes, f.Table.Materials = emptyScale{}, emptyScale{}
	f.Font = resolved(missionFont(), nil)
	f.Carried[0].FigureDir = string(data.FigureDirManFighter)
	f.Carried[0].FigureFace = 1
	s := f.townUI

	squad, ok := s.buildMercenarySquad(10, 1)
	if !ok || len(squad) != 1 {
		t.Fatal("the stock unit's template did not build")
	}
	worn := mapload.EquipmentFromParty(squad[0])
	occupied, _ := worn.Occupied(1)
	code, _ := worn.Code(1)
	if !occupied {
		t.Fatal("the stock unit wears no weapon, so the fixture cannot tell it from a bare sheet")
	}
	f.Archives = stockSpeakerArchives(t, data.ItemFigureLayerPath(data.FigureDirWomanFighter, code))
	human := data.NPCTokens(data.NPCTokenHuman) | data.NPCTokens(data.NPCTokenNotMage) | data.NPCTokens(data.NPCTokenFace)
	f.NPCFaces = map[int32]data.NPCFace{
		90: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 2,
			Tokens: human | data.NPCTokens(data.NPCTokenFemale)},
		64: {Kind: data.NPCFigure, Dir: data.FigureDirManFighter, Face: 27,
			Tokens: human | data.NPCTokens(data.NPCTokenNotFemale)},
	}
	src := f.Archives.Containers
	unit := figureID{Dir: data.FigureDirWomanFighter, Face: 2}
	dressed, _ := composeUnitFigure(src, worn, unit)
	bare, _ := composeUnitFigure(src, data.Equipment{}, unit)
	if dressed == nil || bare == nil || imagesEqual(dressed, bare) {
		t.Fatal("the fixture's weapon layer changes nothing")
	}

	s.room = roomTavern
	s.composeShopFaces()
	if !s.openTownDialogue(TownTavern, TownOffer{NPC: 90, Mission: 30}, 90) {
		t.Fatal("the tavern conversation did not open")
	}
	got, ok := s.TownDialogue()
	layout, _, layoutOK := s.townDialogueLayout()
	if !ok || got == nil || !layoutOK {
		t.Fatal("the tavern conversation did not compose")
	}
	if !imagesEqual(got, ui.RenderNotice(layout, f.Font.Value(), "stock words", dressed)) {
		t.Error("the tavern dialogue does not draw the stock unit wearing its own set")
	}
	if imagesEqual(got, ui.RenderNotice(layout, f.Font.Value(), "stock words", bare)) {
		t.Error("the tavern dialogue draws the stock unit's speaker bare")
	}

	other := figureID{Dir: data.FigureDirManFighter, Face: 27}
	otherBare, _ := composeUnitFigure(src, data.Equipment{}, other)
	if pic, _, ok := s.resolver.SpeakerFace(64); !ok || !imagesEqual(pic, otherBare) {
		t.Error("a Human record no stock unit passes did not stay the record's bare sheet")
	}

	for _, room := range []townRoom{roomShop, roomSchool} {
		s.room = room
		s.composeShopFaces()
		if len(s.resolver.cast.actors) != 0 {
			t.Errorf("room %v resolves speakers over %d actors, want none", room, len(s.resolver.cast.actors))
		}
		if pic, _, ok := s.resolver.SpeakerFace(90); !ok || !imagesEqual(pic, bare) {
			t.Errorf("room %v does not draw npc90 bare", room)
		}
	}
}
