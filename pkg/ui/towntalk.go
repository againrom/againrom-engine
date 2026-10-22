package ui

import (
	"image"

	"againrom/pkg/render/text"
)

// TownTalkProfile is one game's switches for the town talk window. The window
// itself, its pointer contract and its picture are TownTalk's; a profile only
// turns parts of them off.
type TownTalkProfile struct {
	// CountShows makes Shows count the conversations opened while the window
	// stayed open; off, Shows answers 1 while the window is open.
	CountShows bool
	// NeedsFont draws no picture and answers no frame without a font.
	NeedsFont bool
	// ExposesFrame answers the window's frame art and body to the frame
	// composer; off, the composer gets none.
	ExposesFrame bool
	// ButtonNeedsArea offers the button only when its rectangle has area.
	ButtonNeedsArea bool
}

// TownTalkPage is the page the window shows: its words, the speaker's talk
// picture with the window cut from it, and whether the window takes the
// portrait shape.
type TownTalkPage struct {
	Text       string
	Face       *image.RGBA
	FaceWindow image.Rectangle
	Portrait   bool
}

// TownTalkArt is the game's own resources the window is drawn with.
type TownTalkArt struct {
	Words Words
	Frame *DialogFrame
	Font  *text.Font
}

// TownTalkView is one reading of a game's town talk: its profile, whether a
// conversation is open, its current page and its art.
type TownTalkView struct {
	Profile TownTalkProfile
	Open    bool
	Page    TownTalkPage
	Art     TownTalkArt
}

// TownTalk is the town talk window both games build. It holds the window's
// own state: the revision a redraw keys on, the conversations shown, and the
// button and backdrop the pointer last set. Its zero value is a closed window.
type TownTalk struct {
	revision uint64
	shows    uint64
	button   DialogueButtonState
	backdrop DialogueBackdrop
}

// Open records a conversation opening; wasOpen is whether one was open
// before it.
func (w *TownTalk) Open(wasOpen bool) {
	if !wasOpen {
		w.shows = 0
	}
	w.shows++
	w.revision++
}

// Touch records a change of the window's picture.
func (w *TownTalk) Touch() { w.revision++ }

// Reset drops the shows and the pointer state a new game does not inherit;
// the revision keeps counting.
func (w *TownTalk) Reset() {
	w.shows, w.button, w.backdrop = 0, DialogueButtonState{}, DialogueBackdrop{}
}

// Revision is the count of picture changes.
func (w *TownTalk) Revision() uint64 { return w.revision }

// ButtonState is the button's pointer state.
func (w *TownTalk) ButtonState() DialogueButtonState { return w.button }

// Backdrop is the backdrop policy the window is drawn over.
func (w *TownTalk) Backdrop() DialogueBackdrop { return w.backdrop }

// SetVisual stores the pointer's button state and backdrop policy.
func (w *TownTalk) SetVisual(state DialogueButtonState, policy DialogueBackdrop) {
	w.button, w.backdrop = state, policy
}

// Shows is 0 while v is closed, else the profile's count of shows.
func (w *TownTalk) Shows(v TownTalkView) uint64 {
	switch {
	case !v.Open:
		return 0
	case v.Profile.CountShows:
		return w.shows
	}
	return 1
}

// Layout is the authored dialogue layout with the game's words and frame,
// the page's shape and the pointer's state.
func (w *TownTalk) Layout(v TownTalkView) NoticeLayout {
	l := v.Art.Words.OnLayout(AuthoredDialogueLayout())
	l.Frame = v.Art.Frame
	return l.WithPortrait(v.Page.Portrait).WithDialogueButtonState(w.button).WithDialogueBackdrop(w.backdrop)
}

// Picture is the window drawn for the current page.
func (w *TownTalk) Picture(v TownTalkView) (*image.RGBA, bool) {
	if !v.Open || (v.Profile.NeedsFont && v.Art.Font == nil) {
		return nil, false
	}
	l := w.Layout(v).WithFaceWindow(v.Page.FaceWindow)
	return RenderNotice(l, v.Art.Font, v.Page.Text, v.Page.Face), true
}

// Frame is the window's frame art and its body rectangle.
func (w *TownTalk) Frame(v TownTalkView) (*DialogFrame, image.Rectangle) {
	if !v.Profile.ExposesFrame || !v.Open || (v.Profile.NeedsFont && v.Art.Font == nil) {
		return nil, image.Rectangle{}
	}
	l := w.Layout(v)
	return l.Frame, image.Rectangle{Max: l.Box.Size().Sub(image.Pt(8, 8))}
}

// Button is the OK button's rectangle while the window is open.
func (w *TownTalk) Button(v TownTalkView) (image.Rectangle, bool) {
	if !v.Open {
		return image.Rectangle{}, false
	}
	b := w.Layout(v).Button
	return b, !v.Profile.ButtonNeedsArea || !b.Empty()
}

// Advance is the one answer to Enter, Escape and the OK button: turn is the
// game's page turn, which reports whether it acted, and each acting turn is a
// picture change.
func (w *TownTalk) Advance(turn func() bool) TownAction {
	if turn() {
		w.revision++
	}
	return TownAction{}
}
