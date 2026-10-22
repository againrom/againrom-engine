package game

import (
	"image"

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

// townTalkSource is what a town screen answers its talk window with: the page
// it shows and its page turn.
type townTalkSource interface {
	talkView() ui.TownTalkView
	turnTownTalk() bool
}

// townTalkSeat is the town talk window a town screen embeds, and the methods
// pkg/ui reads it through. A seat with no source is a closed window.
type townTalkSeat struct {
	talkWindow ui.TownTalk
	talkSource townTalkSource
}

// openTalk binds the screen and records a conversation opening.
func (s *townTalkSeat) openTalk(source townTalkSource, wasOpen bool) {
	s.talkSource = source
	s.talkWindow.Open(wasOpen)
}

func (s *townTalkSeat) view() ui.TownTalkView {
	if s.talkSource == nil {
		return ui.TownTalkView{}
	}
	return s.talkSource.talkView()
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
// button: the screen's page turn, which on the last page closes the
// conversation.
func (s *townTalkSeat) AdvanceTownDialogue() ui.TownAction {
	return s.talkWindow.Advance(func() bool {
		return s.talkSource != nil && s.talkSource.turnTownTalk()
	})
}
