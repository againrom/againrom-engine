package game

import (
	"fmt"
	"image"
	"image/color"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func visibleOfferApp(t *testing.T, chapter int) (*FrontEnd, *ui.App, *townScreen) {
	t.Helper()
	f, app, s := dialogueKeysApp(t, chapter)
	for i := range f.Font.Value().Glyphs {
		glyph := &f.Font.Value().Glyphs[i]
		for j := range glyph.Pixels {
			glyph.Pixels[j] = text.Pixel{Level: text.MaxLevel, Painted: true}
		}
	}
	return f, app, s
}

func TestOfferFirstPartAppContinuation(t *testing.T) {
	for _, room := range []struct {
		chapter, mission int
		building         TownBuilding
		door, path       string
		back             townRoom
	}{
		{30, 31, TownShop, "SHOP", "text/shop/npc31m31.txt", roomShop},
		{40, 41, TownSchool, "SCHOOL", "text/training/npc34m41.txt", roomSchool},
	} {
		for _, first := range []string{"", "<part=1 iamfemale>\r\nRejected\r\n"} {
			for _, key := range []string{"enter", "escape", "click"} {
				t.Run(room.door+"/"+first+"/"+key, func(t *testing.T) {
					f, app, s := visibleOfferApp(t, room.chapter)
					f.Archives.Containers = townTextFS(t, []synth.File{{Path: room.path, Data: []byte(first + "<part=2 npc=31>\r\nSecond\r\n")}})
					if err := app.HeadlessActivate(room.door); err != nil {
						t.Fatal(err)
					}
					if got := f.Town.Offers(room.building); len(got) != 0 || !containsMission(f.Town.Available(), room.mission) || f.Town.selectedMission() != room.mission {
						t.Fatal("normal builder return failed to register/remove head", got, f.Town.Available(), f.Town.selectedMission())
					}
					if pic, open := s.TownDialogue(); s.room != roomTalk || !open || pic == nil || s.said != 1 {
						t.Fatal("failed first lookup dropped constructed panel", s.room, s.said, open)
					}
					assertOfferPanel(t, f, s, "Nothing to say", true)
					button := dialogueButtonPoint(t, s)
					for _, edge := range []struct {
						action string
						point  image.Point
					}{
						{"release", button}, {"press", button}, {"release", image.Pt(0, 0)}, {"release", button},
						{"press", image.Pt(0, 0)}, {"release", image.Pt(0, 0)},
					} {
						if err := app.HeadlessPointer(edge.action, edge.point.X, edge.point.Y); err != nil {
							t.Fatal(err)
						}
						if s.room != roomTalk || s.said != 1 {
							t.Fatal("unowned pointer event paged/escaped modal", edge)
						}
					}
					dialogueKeyPress(t, app, s, "space")
					if s.room != roomTalk || s.said != 1 {
						t.Fatal("Space escaped default modal")
					}
					dialogueKeyPress(t, app, s, key)
					if s.room != roomTalk || s.said != 2 || s.dialogue.displayPart != 2 {
						t.Fatal("next command did not attempt part 2", s.room, s.said)
					}
					assertOfferPanel(t, f, s, "Second\r\n", true)
					dialogueKeyPress(t, app, s, key)
					if s.room != room.back || s.said != 3 || s.dialogue.text != "Second\r\n" {
						t.Fatal("failed third lookup did not close once with retained display", s.room, s.said, s.dialogue.text)
					}
					before := s.dialogueRevision
					s.AdvanceTownDialogue()
					if s.dialogueRevision != before || app.HeadlessMessage() != "" {
						t.Fatal("closed panel handled a second close")
					}
				})
			}
		}
	}
}

func assertOfferPanel(t *testing.T, f *FrontEnd, s *townScreen, body string, portrait bool) {
	t.Helper()
	layout := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(portrait)
	layout.Frame = f.gameMenuArt()
	layout = layout.WithDialogueButtonState(s.dialogueButtonState).WithDialogueBackdrop(s.dialoguePolicy)
	want := ui.RenderNotice(layout, f.Font.Value(), body, nil)
	wrong := ui.RenderNotice(layout, f.Font.Value(), "Different words", nil)
	got, shown := s.TownDialogue()
	if !shown || !imagesEqual(got, want) || imagesEqual(got, wrong) {
		t.Fatal("panel lost independently specified displayed body", body, shown)
	}
}

func TestOfferPanelCandidateStoresAndFailedHeader(t *testing.T) {
	for _, vector := range []struct {
		name, payload string
		portrait      bool
		speaker       int
		tips          int
	}{
		{"rejected named candidate", "<part=1 npc=31 iamfemale>\r\nRejected\r\n", true, 31, 0},
		{"rejected unnamed candidate", "<part=1 iamfemale>\r\nRejected\r\n<part=3 npc=31>\r\nOther\r\n", false, 0, 0},
		{"tips zero EOF", "<part=1 npc=31 tips=0>", true, 31, 0},
		{"tips seven EOF", "<part=1 npc=31 tips=7>", true, 31, 7},
		{"tips seven inline without LF", "<part=1 npc=31 tips=7>Not a body", true, 31, 7},
		{"tips not reached after rejection", "<part=1 npc=31 iamfemale tips=7>Not a body", true, 31, 0},
		{"empty readable payload", "", false, 0, 0},
	} {
		t.Run(vector.name, func(t *testing.T) {
			f, app, s := visibleOfferApp(t, 30)
			f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(vector.payload)}})
			if err := app.HeadlessActivate("SHOP"); err != nil {
				t.Fatal(err)
			}
			if s.room != roomTalk || s.dialogue.portrait != vector.portrait || s.dialogue.speaker != vector.speaker || s.dialogue.tips != vector.tips || s.dialogue.displayPart != 0 {
				t.Fatal("failed candidate stores were rolled back or body was accepted", s.room, s.dialogue)
			}
			assertOfferPanel(t, f, s, "Nothing to say", EventHasSpeaker([]byte(vector.payload)))
			dialogueKeyPress(t, app, s, "enter")
			if s.room != roomShop || s.dialogue.closeTips != vector.tips || s.dialogue.text != "Nothing to say" {
				t.Fatal("failed next lookup lost tips-close payload or display", s.room, s.dialogue)
			}
		})
	}
}

func TestOfferPanelSamePartRetryAndSubstring(t *testing.T) {
	for _, payload := range []string{
		"<part=1 npc=22 iamfemale>\r\nRejected\r\n<part=1 npc=31>\r\nAccepted\r\n",
		"<part=10 npc=31>\r\nAccepted\r\n<part=1 npc=22>\r\nLater\r\n",
	} {
		f, app, s := visibleOfferApp(t, 30)
		f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(payload)}})
		if err := app.HeadlessActivate("SHOP"); err != nil {
			t.Fatal(err)
		}
		if s.dialogue.speaker != 31 || s.dialogue.displayPart != 1 {
			t.Fatal("same-part scan or substring match selected a different candidate", s.dialogue)
		}
		assertOfferPanel(t, f, s, "Accepted\r\n", true)
	}
}

func TestOfferFailedLookupRetainsDisplayFaceAndVoice(t *testing.T) {
	f, app, s := visibleOfferApp(t, 30)
	payload := "<part=1 npc=31>\r\nStanding\r\n<part=2 npc=34 iamfemale tips=7>\r\nRejected\r\n"
	f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(payload)}})
	f.SpeechBank = responseBank(t, []synth.File{{Path: "shop/npc31m31p1.wav", Data: speechWAV(1234)}, {Path: "shop/npc31m31p2.wav", Data: speechWAV(5678)}})
	r := &tavernInteriorRecorder{}
	f.SpeechPlayer = r
	defer app.StopAudio()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal(err)
	}
	if len(r.voices) != 1 {
		t.Fatal("accepted first part did not deliver one voice", len(r.voices))
	}
	face := image.NewRGBA(image.Rect(0, 0, 3, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 3; x++ {
			face.SetRGBA(x, y, color.RGBA{R: 197, G: 43, B: 139, A: 255})
		}
	}
	s.dialogue.face, s.dialogue.faceWindow = face, image.Rect(1, 2, 3, 4)
	standing, _ := s.TownDialogue()
	s.dialogue.face = nil
	missingFace, _ := s.TownDialogue()
	s.dialogue.face = face
	if imagesEqual(standing, missingFace) {
		t.Fatal("independent retained-face loss control changes no rendered pixels")
	}
	s.said = 2
	if s.lookupTownDialogue() || s.dialogue.text != "Standing" || s.dialogue.face != face || s.dialogue.faceWindow != image.Rect(1, 2, 3, 4) || s.dialogue.displayPart != 1 || s.dialogue.speaker != 34 || !s.dialogue.portrait || s.dialogue.tips != 0 {
		t.Fatal("failed parser changed delivered display or lost candidate metadata", s.dialogue)
	}
	retained, shown := s.TownDialogue()
	if !shown || !imagesEqual(retained, standing) {
		t.Fatal("failed lookup replaced rendered standing text or face")
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if len(r.voices) != 1 || r.voices[0].stops != 0 || !r.voices[0].Playing() {
		t.Fatal("failed lookup replaced/stopped standing voice", len(r.voices), r.voices[0])
	}
	name, ok := s.townSpeechPath()
	if !ok || name != "speech/shop/npc31m31p1.wav" {
		t.Fatal("failed part retargeted delivered speech", name, ok)
	}
	s.said = 1
	dialogueKeyPress(t, app, s, "click")
	if s.room != roomShop || len(r.voices) != 1 || r.voices[0].stops != 1 {
		t.Fatal("close replayed a rejected candidate or failed teardown", s.room, r.voices)
	}
}

func TestOfferTipsStoreSurvivesPagingAndZeroSuppressesClosePayload(t *testing.T) {
	for _, vector := range []struct {
		payload string
		steps   int
		want    int
	}{
		{"<part=1 tips=7>missing LF<part=2>\r\nStanding\r\n", 2, 7},
		{"<part=1 tips=7>\r\nStanding\r\n<part=2 tips=0>missing LF", 1, 0},
	} {
		f, app, s := visibleOfferApp(t, 30)
		f.Archives.Containers = townTextFS(t, []synth.File{{Path: "text/shop/npc31m31.txt", Data: []byte(vector.payload)}})
		if err := app.HeadlessActivate("SHOP"); err != nil {
			t.Fatal(err)
		}
		if s.dialogue.tips != 7 {
			t.Fatal("first candidate did not store tips before body result")
		}
		for i := 0; i < vector.steps; i++ {
			dialogueKeyPress(t, app, s, "enter")
		}
		wantText := "Standing\r\n"
		if vector.want == 0 {
			wantText = "Standing"
		}
		if s.room != roomShop || s.dialogue.closeTips != vector.want || s.dialogue.tips != vector.want || s.dialogue.text != wantText {
			t.Fatal("pager lost current tips or conditional close payload", s.room, s.dialogue)
		}
	}
}

func TestOfferPanelMetadataScanBounds(t *testing.T) {
	for _, value := range []string{"", "+", "-", "2147483648", "-2147483649", "99999999999999999999999999"} {
		d := newTownDialogue([]byte(fmt.Sprintf("<part=1 tips=%s>no LF", value)))
		if d.lookup(1, EventAudience{}) || d.tips != 0 || d.text != "Nothing to say" {
			t.Fatal("invalid integer updated panel or admitted header", value, d)
		}
	}
	for _, value := range []string{"+7;", "-7;", " 7 rest"} {
		d := newTownDialogue([]byte(fmt.Sprintf("<part=1 tips=%s>no LF", value)))
		if d.lookup(1, EventAudience{}) || d.tips == 0 {
			t.Fatal("bounded signed integer store was lost", value, d)
		}
	}
}
