package game

import (
	"bytes"
	"image"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// On the installed map whose fifth dialogue the owner reported, the speakers
// "Main hero" (21) and "Traitor mage" (22) have faces, and 22's is the face
// the game gives a !MySex mage when the player's first hero is a man.
func TestReleaseDialogueHeroSpeakersHaveTheFaceOfAMaleHero(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	e := x.Editor()
	e.Layout(1280, 800)
	if err := e.Open("scenario/130.alm"); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	fields := s.MessageFields(5)
	if len(fields) < 2 {
		t.Fatal("the map holds no dialogue 5")
	}
	speaker := fields[1]
	r := speakerResolver{src: x.archives.artSource(), units: x.units, npcFaces: speakerFaces(x.table),
		portraits: map[string]*image.RGBA{}, figurePics: map[figureCacheKey]*image.RGBA{},
		cast: speakerCast{playerDir: data.FigureDirManFighter, hasPlayer: true}}
	for _, id := range []int64{21, 22} {
		if speaker.Icons[id] == nil {
			t.Errorf("speaker %d has no face", id)
			continue
		}
		face, window, ok := r.SpeakerFace(int(id))
		if !ok {
			t.Fatalf("the game gives no face for speaker %d with a man as the first hero", id)
		}
		if want := ui.SpeakerPortrait(face, window); !bytes.Equal(speaker.Icons[id].Pix, want.Pix) {
			t.Errorf("speaker %d has not the face the game gives with a man as the first hero", id)
		}
		if speaker.IconNotes[id] == "" {
			t.Errorf("speaker %d has no note on its portrait", id)
		}
	}
	if speaker.IconNotes[57] != "" {
		t.Error("speaker 57 has a note on its portrait")
	}
}
