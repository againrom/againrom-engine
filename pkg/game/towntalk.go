package game

import (
	"image"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/ui"
)

// talkCursor is the page cursor of a town conversation in either game: the
// text it reads and the part it has reached, counted from 1.
type talkCursor struct {
	payload []byte
	part    int
}

func (c *talkCursor) open(payload []byte) { c.payload, c.part = payload, 1 }

func (c *talkCursor) close() { c.payload, c.part = nil, 0 }

// next is the page turn: the cursor moves to the following part.
func (c *talkCursor) next() { c.part++ }

// read is the current part through the one part reader.
func (c *talkCursor) read(audience EventAudience, candidate func(tag string)) dialogueRead {
	return readDialoguePart(c.payload, c.part, audience, candidate)
}

// townTalkProfile is one game's town talk as data.
type townTalkProfile struct {
	window ui.TownTalkProfile
	// trim cuts the blank space around the part's words.
	trim bool
	// openBeforePart shows the default words before a part reads.
	openBeforePart bool
	// standingFace takes the speaker from every tag naming the part and keeps
	// the face on a part naming nobody; without it the accepted tag does and
	// such a part shows no face.
	standingFace bool
	// mercenaryFaces shows a mercenary candidate or a platoon unit picture.
	mercenaryFaces bool
}

var firstTownTalkProfile = townTalkProfile{
	window:         ui.TownTalkProfile{CountShows: true, NeedsFont: true, ExposesFrame: true, ButtonNeedsArea: true},
	openBeforePart: true, standingFace: true, mercenaryFaces: true,
}

var secondTownTalkProfile = townTalkProfile{trim: true}

// townTalkRules is each host talk rule's profile; an unnamed rule takes the
// first game's.
var townTalkRules = map[string]townTalkProfile{"building": firstTownTalkProfile, "section": secondTownTalkProfile}

// townTalkHost is what the town talk reads of its screen.
type townTalkHost interface {
	campaignTalk() *talkCursor
	talkOpen() bool
	endTalk()
	talkAudience() EventAudience
	talkFace(speaker int) (*image.RGBA, image.Rectangle, bool)
	talkArt() ui.TownTalkArt
	talkProfile() townTalkProfile
}

// townTalkSeat is the town talk a town screen embeds. A seat with no host is
// a closed window.
type townTalkSeat struct {
	talkWindow ui.TownTalk
	talkHost   townTalkHost
	dialogue   townDialogueState
	// talkDiagnostic replaces the words; the next turn ends the talk.
	talkDiagnostic string
}

func (s *townTalkSeat) bindTalk(host townTalkHost) { s.talkHost = host }

// openTalk records the window's opening and reads the first part.
func (s *townTalkSeat) openTalk(host townTalkHost, payload []byte, wasOpen bool) bool {
	s.bindTalk(host)
	s.talkWindow.Open(wasOpen)
	s.dialogue, s.talkDiagnostic = newTownDialogue(payload), ""
	return s.readTalk()
}

// readTalk reads the cursor's part and its speaker's face.
func (s *townTalkSeat) readTalk() bool {
	c := s.talkHost.campaignTalk()
	if c == nil || !s.dialogue.lookup(*c, s.talkHost.talkAudience(), s.talkHost.talkProfile()) {
		return false
	}
	if s.dialogue.portrait {
		s.dialogue.face, s.dialogue.faceWindow, _ = s.talkHost.talkFace(s.dialogue.speaker)
	}
	return true
}

// view is the one producer of the town talk view of both games.
func (s *townTalkSeat) view() ui.TownTalkView {
	if s.talkHost == nil {
		return ui.TownTalkView{}
	}
	p := s.talkHost.talkProfile()
	if !s.talkHost.talkOpen() || !s.dialogue.shown && !p.openBeforePart {
		return ui.TownTalkView{Profile: p.window}
	}
	text := s.dialogue.text
	switch {
	case s.talkDiagnostic != "":
		text = s.talkDiagnostic
	case p.trim:
		text = strings.TrimSpace(text)
	}
	return ui.TownTalkView{Profile: p.window, Open: true,
		Page: ui.TownTalkPage{Text: text, Face: s.dialogue.face, FaceWindow: s.dialogue.faceWindow, Portrait: s.dialogue.hasPortrait},
		Art:  s.talkHost.talkArt()}
}

// turnTownTalk is the one page turn of both games; past the last part or a
// diagnostic it ends the talk.
func (s *townTalkSeat) turnTownTalk() bool {
	if s.talkHost == nil {
		return false
	}
	c := s.talkHost.campaignTalk()
	if c == nil {
		return false
	}
	c.next()
	if s.talkDiagnostic == "" && s.readTalk() {
		return true
	}
	s.dialogue.closeTips = s.dialogue.tips
	s.talkDiagnostic = ""
	s.talkHost.endTalk()
	return true
}

func (s *townTalkSeat) TownDialogueRevision() uint64 { return s.talkWindow.Revision() }

func (s *townTalkSeat) TownDialogueButtonState() ui.DialogueButtonState {
	return s.talkWindow.ButtonState()
}

func (s *townTalkSeat) TownDialogueVisual(state ui.DialogueButtonState, policy ui.DialogueBackdrop) {
	s.talkWindow.SetVisual(state, policy)
}

func (s *townTalkSeat) TownDialogueShows() uint64 { return s.talkWindow.Shows(s.view()) }

func (s *townTalkSeat) TownDialogue() (*image.RGBA, bool) {
	return s.talkWindow.Picture(s.view())
}

func (s *townTalkSeat) TownDialogueFrame() (*ui.DialogFrame, image.Rectangle) {
	return s.talkWindow.Frame(s.view())
}

func (s *townTalkSeat) TownDialogueButton() (image.Rectangle, bool) {
	return s.talkWindow.Button(s.view())
}

// AdvanceTownDialogue is the one answer to Enter, Escape and a click on the
// button: the page turn, which on the last page ends the conversation.
func (s *townTalkSeat) AdvanceTownDialogue() ui.TownAction {
	return s.talkWindow.Advance(s.turnTownTalk)
}

func (t *townScreen) campaignTalk() *talkCursor { return t.campaignTown().talkCursor() }
func (t *townScreen) talkOpen() bool            { return t.campaignTown().talking() }
func (t *townScreen) endTalk()                  { t.campaignTown().endTalk() }

func (t *townScreen) talkProfile() townTalkProfile {
	if p, ok := townTalkRules[t.hostRules().Talk]; ok {
		return p
	}
	return firstTownTalkProfile
}

func (t *townScreen) talkAudience() EventAudience {
	audience := partyAudience(t.sess.Carried, nil)
	audience.tags = t.tags
	return audience
}

func (t *townScreen) talkArt() ui.TownTalkArt {
	var frame *ui.DialogFrame
	if t.art != nil {
		frame = t.art.menuFrame()
	}
	return ui.TownTalkArt{Words: t.in.Words, Frame: frame, Font: t.in.Font.Value()}
}

// talkFace is the town talk's speaker face in both games: the mission
// notice's speaker resolver, or under mercenary faces a candidate or platoon
// picture in the record's window.
func (t *townScreen) talkFace(speaker int) (*image.RGBA, image.Rectangle, bool) {
	if t.talkProfile().mercenaryFaces {
		rec := t.in.NPCFaces[int32(speaker)]
		var win image.Rectangle
		if rec.HasWindow {
			win = ui.NoticeFaceWindow(rec.WindowX, rec.WindowY)
		}
		switch {
		case t.mercenaryTalk != 0:
			pic, ok := t.mercenaryTalkPicture(t.mercenaryTalk)
			return pic, win, ok
		case rec.Tokens.Has(data.NPCTokenPlatoon):
			pic, ok := t.platoonTalkPicture(speaker)
			return pic, win, ok
		}
	}
	return t.resolver.SpeakerFace(speaker)
}
