package game

import (
	"bytes"
	"image"
	"image/color"
	"slices"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

var speakerPortraitGreen = color.RGBA{G: 0xff, A: 0xff}

// speakerPortraitFaces are the speaker records of the portrait tests: a woman
// fighter whose window opens at her picture's corner, a creature taking its
// class's portrait through the pane's default window, a record with no
// picture, and record 22 with no face record at all.
func speakerPortraitFaces() map[int32]data.NPCFace {
	return map[int32]data.NPCFace{
		51:  {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 8, HasWindow: true},
		103: {Kind: data.NPCPortrait, Class: 64},
		21:  {Kind: data.NPCNoPicture},
	}
}

// speakerPortraitSheets puts the pictures the records name into the editor's
// archives and its unit classes.
func speakerPortraitSheets(x *MapInspector) {
	x.archives.Containers.Overlay("graphics/equipment/ffighter/8.256", func() ([]byte, error) { return invBaseSheet(), nil })
	x.archives.Containers.Overlay("graphics/infowindow/Goblin.bmp", func() ([]byte, error) { return synthBMP(160, 240, speakerPortraitGreen), nil })
	x.units = &terrain.UnitSet{Classes: map[int32]*terrain.UnitClass{64: {Portrait: "Goblin"}}}
}

// gameSpeakerPortrait is speaker id's face as a dialogue in the game resolves
// it, through a resolver of its own over the same records and pictures, cut
// as the dialogue pane cuts it.
func gameSpeakerPortrait(t *testing.T, x *MapInspector, faces map[int32]data.NPCFace, id int) *image.RGBA {
	t.Helper()
	r := speakerResolver{src: x.archives.artSource(), units: x.units, npcFaces: faces,
		portraits: map[string]*image.RGBA{}, figurePics: map[figureCacheKey]*image.RGBA{}}
	face, window, ok := r.SpeakerFace(id)
	if !ok {
		t.Fatalf("the game resolves no face for speaker %d", id)
	}
	return ui.SpeakerPortrait(face, window)
}

func speakerPortraitCatalog(faces map[int32]data.NPCFace) *mapload.Table {
	return &mapload.Table{NPCCatalog: &data.NPCCatalog{Faces: faces, Entries: []data.NPCEntry{
		{ID: 3, Name: "A"}, {ID: 21, Name: "x"}, {ID: 22, Name: "x"}, {ID: 51, Name: "D"}, {ID: 103, Name: "G"},
	}}}
}

// The speaker list carries, for each speaker with a face, the picture the
// game's dialogue shows for that speaker; a speaker with no face, and "No
// speaker", carry none. The list's order, its labels and the written speaker
// are those of the same list without faces.
func TestMessageSpeakerListCarriesEachSpeakersFace(t *testing.T) {
	_, plain := workspaceFixture(t)
	p := plain.Document().Edits.(*mapInspectionEdits)
	p.inspector.table = speakerPortraitCatalog(nil)
	workspacePage(t, plain, "event01.txt", "<part=1,npc=999>\nText")
	bare := p.MessageFields(1)

	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	faces := speakerPortraitFaces()
	s.inspector.table = speakerPortraitCatalog(faces)
	speakerPortraitSheets(s.inspector)
	workspacePage(t, e, "event01.txt", "<part=1,npc=999>\nText")
	fields := s.MessageFields(1)
	speaker := fields[1]

	for _, id := range []int{51, 103} {
		got, want := speaker.Icons[int64(id)], gameSpeakerPortrait(t, s.inspector, speakerFaces(s.inspector.table), id)
		if got == nil || !bytes.Equal(got.Pix, want.Pix) || got.Rect != want.Rect {
			t.Errorf("speaker %d carries %v, want the game's dialogue face", id, got != nil)
		}
	}
	if got := speaker.Icons[103]; got == nil || got.RGBAAt(48, 40) != speakerPortraitGreen || got.RGBAAt(2, 40).A != 0 {
		t.Error("speaker 103's portrait is not its class picture cut at the pane's window")
	}
	for _, id := range []int64{-1, 3, 21, 22, 999} {
		if pic, held := speaker.Icons[id]; held || pic != nil {
			t.Errorf("speaker %d has no face and carries a picture", id)
		}
	}

	if !slices.Equal(speaker.Choices, bare[1].Choices) || speaker.Value != bare[1].Value {
		t.Fatalf("speaker list %v (value %d), without faces %v (value %d)", speaker.Choices, speaker.Value, bare[1].Choices, bare[1].Value)
	}
	var ids []int64
	for _, c := range speaker.Choices {
		ids = append(ids, c.Value)
	}
	if want := []int64{-1, 21, 22, 3, 51, 103, 999}; !slices.Equal(ids, want) {
		t.Fatalf("speaker order %v, want %v", ids, want)
	}
	if _, err := s.SetMessage(1, "Text", 51); err != nil || string(s.workspace.Pages["event01.txt"]) != "<part=1,npc=51>\nText" {
		t.Fatal("markup", err, string(s.workspace.Pages["event01.txt"]))
	}
}

// The person window's NPC list is the speaker list's builder: each record
// with a face carries the same picture the speaker list carries for it.
func TestPersonWindowNPCListCarriesTheSpeakerListsFaces(t *testing.T) {
	_, s, _ := nativeIdentityEditor(t)
	faces := map[int32]data.NPCFace{8: {Kind: data.NPCFigure, Dir: data.FigureDirWomanFighter, Face: 7, HasWindow: true}, 7: {Kind: data.NPCNoPicture}}
	s.inspector.table.NPCCatalog.Faces = faces
	fields := s.Fields("Unit", 0)
	npc := fields[fieldIndex(t, fields, "npc")]
	want := gameSpeakerPortrait(t, s.inspector, speakerFaces(s.inspector.table), 8)
	if got := npc.Icons[8]; got == nil || !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("NPC 8 carries no face, or not the game's dialogue face")
	}
	for _, id := range []int64{-1, 0, 7, 9} {
		if _, held := npc.Icons[id]; held {
			t.Errorf("NPC %d has no face and carries a picture", id)
		}
	}
	speakers := s.MessageFields(ui.NewMessagePage)
	if len(speakers) != 2 || speakers[1].Icons[8] != npc.Icons[8] {
		t.Error("the person window and the speaker list carry different pictures of NPC 8")
	}
}
