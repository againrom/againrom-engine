package ui

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

// EndingView supplies the campaign ending and the independent main-menu hall.
// Hall rows retain their source order. This interface awards no score.
type EndingView struct {
	Hero           string
	Gold           int
	Score          int32
	HasScore       bool
	Recorded       bool
	Credits        []string
	Hall           []EndingHallRow
	HallAvailable  bool
	HallBackground *image.RGBA
	HallFont       *text.Font
}

type EndingHallRow struct {
	Name  string
	Score int32
}

// endingSeams are the installed callbacks of the ending: the campaign ending
// source, the main-menu hall source, the credits roll opener and the campaign
// reset that ends the ending.
type endingSeams struct {
	endingSource func() (EndingView, bool)
	hallSource   func() EndingView
	rollOpener   func() bool
	endingExit   func()
}

type endingWords struct {
	Title, Hall, Credits, Menu, Back, Previous, Next, Gold    string
	MissingHall, MissingCredits, EmptyHall                    string
	Score, UnknownScore, RecordedHall, PendingHall, RetryHall string
}

func (f *flow) endingWords() endingWords {
	return endingWords{Title: f.word("ending.title"), Hall: f.word("ending.hall"), Credits: f.word("ending.credits"),
		Menu: f.word("ending.menu"), Back: f.word("ending.back"), Previous: f.word("ending.previous"),
		Next: f.word("ending.next"), Gold: f.word("ending.gold"), MissingHall: f.word("ending.missing_hall"),
		MissingCredits: f.word("ending.missing_credits"), EmptyHall: f.word("ending.empty_hall"),
		Score: f.word("ending.score"), UnknownScore: f.word("ending.unknown_score"),
		RecordedHall: f.word("ending.recorded_hall"), PendingHall: f.word("ending.pending_hall"),
		RetryHall: f.word("ending.retry_hall")}
}

// SetCampaignEndingExit installs the campaign reset that runs when the hall
// ends the ending.
func (a *App) SetCampaignEndingExit(exit func()) { a.flow.endingSeams.endingExit = exit }

func (a *App) SetCampaignEnding(source func() (EndingView, bool)) {
	a.flow.endingSeams.rollOpener = func() bool {
		if a.media.credits == nil {
			return false
		}
		a.openCredits(ScreenEnding)
		return true
	}
	if source == nil {
		a.flow.endingSeams.endingSource = nil
		return
	}
	a.flow.endingSeams.endingSource = func() (EndingView, bool) {
		view, complete := source()
		if complete {
			credits := strings.Join(view.Credits, "\n")
			view.Credits = nil
			if strings.TrimSpace(credits) != "" {
				view.Credits = a.saveWrapped(credits, 544)
			}
		}
		return view, complete
	}
}

func (a *App) SetHallOfFame(source func() EndingView) { a.flow.endingSeams.hallSource = source }

func (f *flow) showHallOfFame() {
	if f.endingSeams.hallSource == nil {
		return
	}
	f.ending = f.endingSeams.hallSource()
	f.hallFromMenu = true
	f.endingPage, f.endingTop, f.endingFocus, f.endingPress = 2, 0, 0, buttonLatch{}
	f.msg = ""
	f.setScreen(ScreenEnding)
}

func (f *flow) showCampaignEnding() bool {
	if f.endingSeams.endingSource == nil {
		return false
	}
	view, complete := f.endingSeams.endingSource()
	if !complete {
		return false
	}
	f.ending = view
	f.hallFromMenu = false
	f.endingPage, f.endingTop, f.endingFocus, f.endingPress = 2, 0, 0, buttonLatch{}
	f.msg = ""
	// The ending runs the credits and then the hall (FAME-029). Without a
	// credits roll the ending's own text page stands in for it.
	if f.endingSeams.rollOpener != nil && f.endingSeams.rollOpener() {
		return true
	}
	f.endingPage = 1
	f.setScreen(ScreenEnding)
	return true
}

// endingBack leaves the ending's current page. The credits text page leads to
// the hall; the hall's own button leaves the ending; the menu's hall view
// returns to the menu.
func (f *flow) endingBack() {
	switch {
	case f.hallFromMenu:
		f.hallFromMenu = false
		f.toMenu()
	case f.endingPage == 1:
		f.endingToHall()
	default:
		f.endingLeave()
	}
}

// endingToHall refreshes the view, which retries an unrecorded result, and
// shows the hall page.
func (f *flow) endingToHall() {
	if f.endingSeams.endingSource != nil {
		if view, complete := f.endingSeams.endingSource(); complete {
			f.ending = view
		}
	}
	f.endingPage, f.endingTop, f.endingFocus, f.endingPress = 2, 0, 0, buttonLatch{}
}

// endingLeave ends the campaign: the terminal route resets it and returns to
// the main menu (FAME-029, SAV-971).
func (f *flow) endingLeave() {
	if f.endingSeams.endingExit != nil {
		f.endingSeams.endingExit()
	}
	f.toMenu()
}

func (f *flow) endingRows() ([]string, int) {
	if f.endingPage == 1 {
		return f.ending.Credits, 20
	}
	rows := make([]string, len(f.ending.Hall))
	for i, row := range f.ending.Hall {
		rows[i] = row.Name
	}
	return rows, 10
}

func (f *flow) endingButtons() []string {
	w := f.endingWords()
	rows, count := f.endingRows()
	if len(rows) <= count {
		return []string{w.Back}
	}
	return []string{w.Back, w.Previous, w.Next}
}

// hallEndRect is the hall's button rectangle: a left-button release inside it
// ends the hall, whether or not the press began there (FAME-029).
var hallEndRect = image.Rect(0x230, 0x1a0, 0x25c, 0x1c0)

// hallHasArtButton reports the hall layout whose only control is the art's
// own button.
func (f *flow) hallHasArtButton() bool {
	return f.endingPage == 2 && len(f.ending.Hall) <= 10 && f.ending.HallBackground != nil
}

func (f *flow) endingButtonRect(index int) image.Rectangle {
	if f.hallHasArtButton() {
		return hallEndRect
	}
	return image.Rect(28+index*148, 430, 168+index*148, 462)
}

func (f *flow) endingActivate(index int) {
	f.endingPress.Clear()
	rows, count := f.endingRows()
	switch index {
	case 0:
		f.endingBack()
	case 1:
		f.endingTop = max(0, f.endingTop-count)
	case 2:
		f.endingTop = min(max(0, len(rows)-count), f.endingTop+count)
	}
}

func (a *App) stepEnding(in appInput) {
	f := a.flow
	if in.Unfocused {
		f.endingPress.Clear()
		return
	}
	buttons := f.endingButtons()
	if in.Down || in.PaneMode {
		f.endingFocus = (f.endingFocus + 1) % len(buttons)
	}
	if in.Up {
		f.endingFocus = (f.endingFocus + len(buttons) - 1) % len(buttons)
	}
	if in.WheelY != 0 {
		rows, count := f.endingRows()
		f.endingTop = min(max(0, len(rows)-count), max(0, f.endingTop-int(in.WheelY)*3))
	}
	if in.Enter {
		a.activateEnding(f.endingFocus)
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	hit := -1
	if ok {
		for i := range buttons {
			if p.In(f.endingButtonRect(i)) {
				hit = i
				break
			}
		}
	}
	if in.PrimaryPressed {
		f.endingPress.Press(hit, hit >= 0)
		if hit >= 0 {
			f.endingFocus = hit
		}
		// A press on the Hall of Fame's OK, button 0 of page 2, requests the
		// hall's own ok.wav unless it plays (VIDEO-SFX-060).
		if hit == 0 && f.endingPage == 2 {
			a.hallSounds.RequestFor("hall-of-fame", a.soundPlayer, a.namedSounds(), ChargenSoundOK)
		}
	}
	if in.PrimaryReleased {
		if ok && f.hallHasArtButton() && p.In(hallEndRect) {
			f.endingPress.Clear()
			a.playUISound(UISoundCommonControl)
			a.activateEnding(0)
			return
		}
		_, activate := f.endingPress.Release(hit, hit >= 0)
		if activate {
			a.playUISound(UISoundCommonControl)
			a.activateEnding(hit)
		}
	}
}

func (a *App) activateEnding(index int) {
	a.flow.endingActivate(index)
}

func (a *App) endingPaint() *savePaint {
	f, w := a.flow, a.flow.endingWords()
	p := &savePaint{pix: image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))}
	draw.Draw(p.pix, p.pix.Bounds(), image.NewUniform(pickerBackground), image.Point{}, draw.Src)
	ink := gameMenuText
	if f.endingPage == 2 && f.ending.HallBackground != nil {
		draw.Draw(p.pix, p.pix.Bounds(), f.ending.HallBackground, image.Point{}, draw.Src)
		ink = color.RGBA{R: 40, G: 22, B: 8, A: 255}
	} else {
		drawFrame(p.pix, panelFrame(image.Rect(20, 20, 620, 416), pickerBackground, saveFocusColor))
		title := w.Title
		if f.endingPage == 1 {
			title = w.Credits
		}
		if f.endingPage == 2 {
			title = w.Hall
		}
		p.label(title, 46, 42, saveFocusColor)
	}
	label := func(s string, x, y, width int) { p.label(a.fitEndingText(s, width), x, y, ink) }
	rows, count := f.endingRows()
	top := min(f.endingTop, max(0, len(rows)-count))
	x, y, pitch, width := 48, 86, 15, 544
	if f.endingPage == 2 {
		x, y, pitch, width = 145, 145, 24, 342
	}
	for i := 0; i < count && top+i < len(rows); i++ {
		if f.endingPage == 2 {
			row := f.ending.Hall[top+i]
			label(fmt.Sprintf("%d.", top+i+1), 145, y+i*pitch, 32)
			label(row.Name, 176, y+i*pitch, 216)
			score := hallScoreText(row.Score)
			label(score, 484-a.endingTextWidth(score), y+i*pitch, 88)
		} else {
			label(rows[top+i], x, y+i*pitch, width)
		}
	}
	if len(rows) == 0 {
		note := w.MissingCredits
		if f.endingPage == 2 {
			note = w.MissingHall
			if f.ending.HallAvailable {
				note = w.EmptyHall
			}
		}
		label(note, x, y, width)
	}
	if f.endingPage == 2 && !f.hallFromMenu && f.ending.HasScore && !f.ending.Recorded {
		label(w.PendingHall, 145, 380, 342)
	}
	for i, name := range f.endingButtons() {
		if f.endingPage == 2 && len(f.ending.Hall) <= 10 && f.ending.HallBackground != nil {
			continue
		}
		r := f.endingButtonRect(i)
		border := gameMenuText
		if i == f.endingFocus {
			border = saveFocusColor
		}
		drawFrame(p.pix, panelFrame(r, pickerBackground, border))
		p.label(a.fitEndingText(name, r.Dx()-12), r.Min.X+6, r.Min.Y+8, gameMenuText)
	}
	return p
}

// The owner reference groups score digits and leaves a zero score blank.
func hallScoreText(score int32) string {
	if score == 0 {
		return ""
	}
	return GroupDigits(int64(score))
}

func (a *App) fitEndingText(value string, width int) string {
	if a.endingFont() == nil {
		return value
	}
	if a.endingTextWidth(value) <= width {
		return value
	}
	r := []rune(value)
	for len(r) > 0 && a.endingTextWidth(string(r)+"...") > width {
		r = r[:len(r)-1]
	}
	return string(r) + "..."
}

func (a *App) endingFont() *text.Font {
	if a.flow.endingPage == 2 && a.flow.ending.HallFont != nil {
		return a.flow.ending.HallFont
	}
	return a.flow.menuFont
}

func (a *App) endingTextWidth(s string) int {
	if font := a.endingFont(); font != nil {
		width, _ := font.Measure(a.flow.menuDisplayText(s))
		return width
	}
	return len(s) * 8
}

func (a *App) composeEndingScreen() (*image.RGBA, error) {
	font := a.endingFont()
	if font == nil {
		return nil, fmt.Errorf("ending has no installed font")
	}
	p := a.endingPaint()
	for _, label := range p.texts {
		font.Draw(p.pix, a.flow.menuDisplayText(label.text), label.at.X, label.at.Y, label.color)
	}
	return p.pix, nil
}

func (a *App) drawEnding() {
	a.hasMenu = false
	if pix, err := a.composeEndingScreen(); err == nil {
		a.writeCanvas(pix)
		return
	}
	p := a.endingPaint()
	a.writeCanvas(p.pix)
	for _, label := range p.texts {
		a.printCanvas(label.text, label.at.X, label.at.Y)
	}
}

func (a *App) headlessEnding(target string) error {
	for index, label := range a.flow.endingButtons() {
		if !strings.EqualFold(target, label) {
			continue
		}
		for i := 0; i < len(a.flow.endingButtons()) && a.flow.endingFocus != index; i++ {
			if err := a.HeadlessKey("down"); err != nil {
				return err
			}
		}
		return a.HeadlessKey("enter")
	}
	return fmt.Errorf("ending has no control %q", target)
}
