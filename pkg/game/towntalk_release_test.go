package game

import (
	"fmt"
	"image"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

// townTalkScreen is a town screen's talk window contract.
type townTalkScreen interface {
	ui.TownDialogueScreen
	TownDialogueRevision() uint64
	TownDialogueButtonState() ui.DialogueButtonState
}

// townTalkOK presses and releases the open window's OK button through App
// pointer input at 640x480. The press draws the pressed button, checked by
// pressed, and turns nothing; the release turns the page once.
func townTalkOK(t *testing.T, app *ui.App, d townTalkScreen, pressed func()) {
	t.Helper()
	pic, open := d.TownDialogue()
	button, shown := d.TownDialogueButton()
	if !open || !shown || pic == nil {
		t.Fatal("the open talk shows no OK button")
	}
	p := button.Min.Add(button.Max).Div(2).Add(image.Pt((640-pic.Bounds().Dx())/2, (480-pic.Bounds().Dy())/2))
	before := d.TownDialogueRevision()
	for _, edge := range []string{"hover", "press", "release"} {
		if edge == "release" {
			if !d.TownDialogueButtonState().Pressed || d.TownDialogueRevision() != before {
				t.Fatalf("the press left button %+v and revision %d, want pressed at %d", d.TownDialogueButtonState(), d.TownDialogueRevision(), before)
			}
			pressed()
		}
		if err := app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			t.Fatal(err)
		}
		if _, _, err := app.HeadlessFrame(); err != nil {
			t.Fatalf("%s at %v: %v", edge, p, err)
		}
	}
	if got := d.TownDialogueRevision(); got != before+1 {
		t.Fatalf("the release left revision %d, want %d", got, before+1)
	}
}

// TestReleaseTownTalkOKThroughTheWindow turns every page of the installed
// inn conversation of the first mission offer with OK through App. Each
// pressed page is the authored notice of the page with the game's words,
// frame and speaker picture; the last release closes the window into the inn.
func TestReleaseTownTalkOKThroughTheWindow(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f := releaseFront(t)
	app, s := roomExitApp(t, f, 0)
	s.CloseTip()
	var offer TownOffer
	for _, o := range f.Town.Offers(TownTavern) {
		if o.Mission > 0 {
			offer = o
			break
		}
	}
	if offer.Mission == 0 {
		t.Fatal("the first inn holds no mission offer")
	}
	roomExitEnter(t, app, s, "TAVERN", roomTavern)
	if err := app.HeadlessActivate(fmt.Sprintf("NPC %d", offer.NPC)); err != nil {
		t.Fatal(err)
	}
	pages := len(s.townLines())
	if s.room != roomTalk || pages == 0 {
		t.Fatalf("NPC %d opened room %d with %d pages", offer.NPC, s.room, pages)
	}
	for page := 1; page <= pages; page++ {
		if s.room != roomTalk || s.talkPage.part != page {
			t.Fatalf("page %d: room %d page %d", page, s.room, s.talkPage.part)
		}
		townTalkOK(t, app, s, func() {
			l := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(s.mercenaryTalk != 0 || s.dialogue.hasPortrait)
			l.Frame = f.gameMenuArt()
			l = l.WithDialogueButtonState(s.TownDialogueButtonState()).WithDialogueBackdrop(s.talkWindow.Backdrop()).WithFaceWindow(s.dialogue.faceWindow)
			want := ui.RenderNotice(l, f.Font.Value(), s.dialogue.text, s.dialogue.face)
			if got, _ := s.TownDialogue(); !imagesEqual(got, want) {
				t.Fatalf("page %d of %d is not the authored notice of %q", page, pages, s.dialogue.text)
			}
		})
	}
	if s.room != roomTavern {
		t.Fatalf("the last OK left room %d, want the inn", s.room)
	}
	t.Logf("NPC %d mission %d: %d pages, revision %d", offer.NPC, offer.Mission, pages, s.TownDialogueRevision())
}

// TestReleaseSecondTownTalkOKThroughTheWindow turns every page of the
// installed inn talk of speaker 517 with OK through App. Each pressed page is
// the authored notice of its trimmed section part with the game's words and
// frame and the face of the speaker its tag names; the last release closes
// the talk.
func TestReleaseSecondTownTalkOKThroughTheWindow(t *testing.T) {
	f, app := secondGeneratedMage(t, false)
	// Each TownScreen call builds a screen; the App is given this one so the
	// window it drives is the one read here.
	s := f.TownScreen().(*secondCampaignScreen)
	app.SetTown(s)
	for _, target := range []string{"TAVERN", "TALK 517"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	c := f.Town.second
	payload := c.talk.payload
	audience := partyAudience(f.Carried, nil)
	audience.tags = s.tags
	page := 1
	for ; c.talk.payload != nil; page++ {
		r := readDialoguePart(payload, page, audience, nil)
		body, ok := r.text, r.ok
		if page > 64 || !ok || c.talk.part != page {
			t.Fatalf("page %d: part %d, installed part found %v", page, c.talk.part, ok)
		}
		speaker, named := audience.tagRules().speaker(r.tag)
		face, window, found := s.talkFaces.SpeakerFace(speaker)
		if !named || !found || face == nil {
			t.Fatalf("page %d: speaker %d named %v has no face", page, speaker, named)
		}
		townTalkOK(t, app, s, func() {
			l := f.Words.OnLayout(ui.AuthoredDialogueLayout()).WithPortrait(true)
			l.Frame = f.gameMenuArt()
			l = l.WithDialogueButtonState(s.TownDialogueButtonState()).WithDialogueBackdrop(s.talkWindow.Backdrop()).WithFaceWindow(window)
			want := ui.RenderNotice(l, f.Font.Value(), strings.TrimSpace(body), face)
			if got, _ := s.TownDialogue(); !imagesEqual(got, want) {
				t.Fatalf("page %d is not the authored notice of %q", page, body)
			}
		})
	}
	if _, more := dialoguePart(payload, page, audience); more || c.talk.part != 0 || page == 1 {
		t.Fatalf("the talk closed after %d pages with part %d left", page-1, c.talk.part)
	}
	t.Logf("speaker 517: %d pages, revision %d", page-1, s.TownDialogueRevision())
}
