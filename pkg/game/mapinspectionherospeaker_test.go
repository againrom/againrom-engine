package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

// heroSpeakerFaces are the records of the hero speaker tests: the registry's
// four archetypes and the four hero records, and record 51, a woman fighter
// who needs no hero.
func heroSpeakerFaces(t *testing.T) map[int32]data.NPCFace {
	t.Helper()
	section := func(name, flags string, face int32) synth.RegNode {
		children := []synth.RegNode{{Name: "Flags", Kind: 0, Str: flags}}
		if face != 0 {
			children = append(children, synth.RegNode{Name: "Face", Kind: kindInt, Int: face})
		}
		return synth.RegNode{Name: name, Kind: kindDir, Children: children}
	}
	registry, err := reg.Parse(synth.Reg(kindRoot, []synth.RegNode{
		section("MaleFighter", "", 5), section("MaleMage", "", 3),
		section("FemaleFighter", "", 1), section("FemaleMage", "", 1),
		section("npc21", "Hero,Me,Start", 0),
		section("npc22", "Hero,Mage,!MySex,Start", 0),
		section("npc23", "Hero,!Mage,!MySex,Start", 0),
		section("npc24", "Hero,!MyClass,MySex,Start", 0),
	}))
	if err != nil {
		t.Fatal(err)
	}
	faces := data.LoadNPCFaces(registry)
	faces[51] = data.NPCFace{Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8, HasWindow: true}
	return faces
}

// With no player in the editor, a speaker whose face the game takes from the
// player's hero shows the face the game gives when the first hero is a man:
// the main hero the man fighter's, the traitor mage the woman mage's. A speaker
// that needs no hero keeps his own face.
func TestEditorSpeakersTakeTheFaceOfAMaleHero(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	faces := heroSpeakerFaces(t)
	entries := []data.NPCEntry{{ID: 21, Name: "x"}, {ID: 22, Name: "x"}, {ID: 51, Name: "D"}}
	s.inspector.table = &mapload.Table{NPCCatalog: &data.NPCCatalog{Faces: faces, Entries: entries}}
	speakerPortraitSheets(s.inspector)
	src := s.inspector.archives.Containers
	for _, f := range []struct {
		dir  data.FigureDir
		face int
	}{{data.FigureDirManFighter, 5}, {data.FigureDirWomanMage, 1}} {
		src.Overlay("graphics/"+data.ItemFigureBasePath(f.dir, f.face), func() ([]byte, error) { return invBaseSheet(), nil })
	}
	workspacePage(t, e, "event01.txt", "<part=1,npc=999>\nText")
	got := s.MessageFields(1)[1].Icons
	art := s.inspector.archives.artSource()
	for id, want := range map[int64]figureID{
		21: {Dir: data.FigureDirManFighter, Face: 5, Hero: true},
		22: {Dir: data.FigureDirWomanMage, Face: 1, Hero: true},
	} {
		pic, _ := composeInventoryPortrait(art, 0, data.Equipment{}, want)
		exp := ui.SpeakerPortrait(pic.Figure, image.Rectangle{})
		if got[id] == nil || !bytes.Equal(got[id].Pix, exp.Pix) {
			t.Errorf("speaker %d has no face of %+v", id, want)
		}
	}
	if got[51] == nil {
		t.Error("the speaker who needs no hero lost his face")
	}
}

// The hover line over a speaker's portrait is given for the speakers whose
// face depends on the player's hero and for no other.
func TestHeroFaceNotesNameOnlyTheHeroSpeakers(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	workspacePage(t, e, "event01.txt", "<part=1,npc=999>\nText")
	notes := s.MessageFields(1)[1].IconNotes
	for _, id := range []int64{21, 22, 23, 24} {
		if notes[id] == "" {
			t.Errorf("speaker %d has no note", id)
		}
	}
	for _, id := range []int64{-1, 5, 51, 57} {
		if notes[id] != "" {
			t.Errorf("speaker %d has the note %q", id, notes[id])
		}
	}
}
