package ui

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/render/text"
)

type CutsceneEntry struct{ Directory, Title string }
type CutsceneLibraryWords struct{ Title, OK, Cancel string }
type CreditsView struct {
	Lines []string // installed text bytes, including empty lines
	Logos map[string]*image.RGBA
}

type mediaMessages struct{ Empty, Unavailable, HistoryFailed string }

//go:embed media_ru.json
var mediaRussianJSON []byte
var mediaRussian = func() mediaMessages {
	var w mediaMessages
	if err := json.Unmarshal(mediaRussianJSON, &w); err != nil {
		panic(err)
	}
	return w
}()

func (a *App) mediaMessages() mediaMessages {
	if a.flow.menuFont != nil && a.flow.menuFont.Selector == text.SelectorConverting {
		return mediaRussian
	}
	return mediaMessages{"No cutscenes encountered yet.", "Movie unavailable.", "Could not save viewed movies."}
}

type mediaUI struct {
	catalog []CutsceneEntry
	seen    map[string]bool
	record  func(string) error
	words   CutsceneLibraryWords
	list    *Picker
	press   buttonLatch
	bar     scrollBarInput
	message string
	credits func() CreditsView
	roll    CreditsView
	scroll  []*image.RGBA
	back    Screen
	at      time.Time // last roll step; zero before the first
	steps   int       // pixels the roll has advanced
}

func (a *App) SetCutsceneLibrary(entries []CutsceneEntry, seen []string, record func(string) error, words CutsceneLibraryWords) {
	a.media.catalog = append([]CutsceneEntry(nil), entries...)
	a.media.seen = make(map[string]bool)
	for _, key := range seen {
		a.media.seen[key] = true
	}
	a.media.record, a.media.words = record, words
}

func (a *App) SetCredits(source func() CreditsView) { a.media.credits = source }

func (a *App) SetCutsceneScrollArt(frames []*image.RGBA) {
	a.media.scroll = frames
	a.flow.helpScroll = &helpScrollArt{frames: frames}
	if a.flow.viewer != nil {
		a.flow.viewer.setHelpScrollArt(frames)
	}
}

// SetHoverArt installs interface/Ball.bmp, the hover box's corner picture
// (MENU-128). Nil leaves the corners unpainted.
func (a *App) SetHoverArt(ball image.Image) { a.tooltip.ball = ball }

func (a *App) seenCutscenes() []CutsceneEntry {
	var entries []CutsceneEntry
	for _, entry := range a.media.catalog {
		if a.media.seen[entry.Directory] {
			entries = append(entries, entry)
		}
	}
	return entries
}

// Only successfully presented movies unlock their installed catalog group.
// Logos and failed opens cannot unlock an entry. A multipart group replays its
// complete numbered sequence in the selected archive family.
func (a *App) encounterCutscene(name string) {
	directory, _, ok := strings.Cut(strings.ToLower(strings.ReplaceAll(name, `\`, "/")), "/")
	if !ok {
		return
	}
	for _, entry := range a.media.catalog {
		if entry.Directory != directory || a.media.seen[directory] {
			continue
		}
		a.media.seen[directory] = true
		if a.media.record != nil {
			if err := a.media.record(directory); err != nil {
				a.media.message = a.mediaMessages().HistoryFailed
			}
		}
		return
	}
}

var moviePanel = image.Rect(120, 62, 520, 424)
var movieListArg = image.Rect(170, 134, 466, 334)
var movieOK = image.Rect(172, 376, 296, 400)
var movieCancel = image.Rect(348, 376, 472, 400)

const movieVisibleRows = 10

// The library's two buttons, as button latch ids.
const (
	movieOKButton = iota
	movieCancelButton
)

// movieListBox is the library's shared list.
func (a *App) movieListBox() listBox {
	return newListBox(movieListArg, movieVisibleRows, a.flow.menuFont)
}

// movieList is the shared list over the seen entries, rebuilt when the seen
// set changes and keeping the selected index.
func (a *App) movieList() *Picker {
	entries := a.seenCutscenes()
	if a.media.list == nil || a.media.list.Len() != len(entries) {
		sel := 0
		if a.media.list != nil {
			sel = max(a.media.list.Selection(), 0)
		}
		rows := make([]PickerRow, len(entries))
		for i, e := range entries {
			rows[i] = PickerRow{Text: e.Title, Choosable: true}
		}
		a.media.list = NewPicker(rows).SetWindow(movieVisibleRows)
		a.media.list.Select(clampIndex(sel, len(rows)))
	}
	return a.media.list
}

func (a *App) openCutsceneLibrary() {
	a.media.list, a.media.press, a.media.bar = nil, buttonLatch{}, scrollBarInput{}
	a.flow.setScreen(ScreenCutsceneLibrary)
	if a.media.words.Title == "" {
		a.media.words.Title = "View Cutscenes"
	}
	if a.media.words.OK == "" {
		a.media.words.OK = "OK"
	}
	if a.media.words.Cancel == "" {
		a.media.words.Cancel = "Cancel"
	}
	// Opening the viewer is observational: it never marks catalog entries.
}

func (a *App) selectMovie(index int) {
	l := a.movieList()
	l.Select(clampIndex(index, l.Len()))
}

func (a *App) replayMovie() {
	entries := a.seenCutscenes()
	if len(entries) == 0 {
		return
	}
	a.media.message = ""
	if !a.PlayCutsceneSequence(numberedCutscenes(entries[max(a.movieList().Selection(), 0)].Directory)) {
		a.media.message = a.mediaMessages().Unavailable
	}
}

// movieButtonAt is the library button under p.
func movieButtonAt(p image.Point) (int, bool) {
	switch {
	case p.In(movieOK):
		return movieOKButton, true
	case p.In(movieCancel):
		return movieCancelButton, true
	}
	return 0, false
}

func (a *App) stepMedia(in appInput, now time.Time) {
	if in.Unfocused {
		a.media.press.clear()
		a.media.bar.reset()
		a.media.at = time.Time{}
		return
	}
	if a.flow.screen == ScreenCredits {
		// Any key-down or a left-button press ends the roll; the right
		// button does not (FAME-031).
		if in.Escape || in.AnyKey || in.Enter || in.PrimaryPressed {
			a.closeCredits()
			return
		}
		// One pixel per step, and a step only when creditsStepGap has passed
		// since the last one; the first step waits for nothing (FAME-030). A
		// tick that follows a pause only restarts the gap.
		switch {
		case a.media.at.IsZero() && a.media.steps > 0:
			a.media.at = now
		case a.media.at.IsZero() || now.Sub(a.media.at) >= creditsStepGap:
			a.media.steps++
			a.media.at = now
		}
		// The roll ends after creditsStart plus line count times line pitch
		// steps (FAME-030).
		if n := len(a.media.roll.Lines); n > 0 && a.media.steps >= creditsStart+n*a.creditsPitch() {
			a.closeCredits()
		}
		return
	}
	if in.Escape || in.SecondaryPressed {
		a.flow.toMenu()
		return
	}
	list := a.movieList()
	listKey(list, in.Up, in.Down, in.PageUp, in.PageDown)
	if in.Home {
		a.selectMovie(0)
	}
	if in.End {
		a.selectMovie(list.Len() - 1)
	}
	if in.WheelY > 0 {
		list.Move(-3)
	}
	if in.WheelY < 0 {
		list.Move(3)
	}
	if in.Enter {
		a.replayMovie()
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	box := a.movieListBox()
	if req, pos := a.media.bar.step(listBar(box, list), p, ok, in); req != barNone {
		listBarRequest(list, req, pos)
		return
	}
	if a.media.bar.active() {
		return
	}
	if in.PrimaryPressed {
		a.media.press.press(movieButtonAt(p))
		if row, hit := box.RowAt(p); ok && hit {
			top, _ := list.Visible()
			a.selectMovie(top + row)
		}
	}
	if in.PrimaryReleased {
		at, inside := movieButtonAt(p)
		button, activated := a.media.press.release(at, ok && inside)
		if !activated {
			return
		}
		a.playUISound(UISoundCommonControl)
		switch button {
		case movieOKButton:
			a.replayMovie()
		case movieCancelButton:
			a.flow.toMenu()
		}
	}
}

func (a *App) composeCutsceneLibrary() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	if a.assets != nil {
		draw.Draw(dst, dst.Bounds(), a.assets.Compose(a.sel.State()), image.Point{}, draw.Src)
	}
	drawMenuPanel(dst, moviePanel, a.flow.menuArt)
	font := a.flow.menuFont
	if font == nil {
		return dst
	}
	drawTownShellText(dst, font, a.media.words.Title, image.Rect(148, 88, 492, 116), townShellText)
	entries := a.seenCutscenes()
	list := a.movieList()
	pointer, pointerOK := a.pointerFrame()
	drawListBox(dst, font, a.media.scroll, a.movieListBox(), list, func(row, _ int) string { return entries[row].Title }, pointer, pointerOK)
	if len(entries) == 0 {
		font.Draw(dst, a.flow.menuDisplayText(a.mediaMessages().Empty), movieListArg.Min.X+4, 350, townShellText)
	}
	for i, r := range []image.Rectangle{movieOK, movieCancel} {
		inside := pointerOK && pointer.In(r)
		drawPushButton(dst, font, pushButton{Rect: r, Label: []string{a.media.words.OK, a.media.words.Cancel}[i],
			Hover: inside, Pressed: a.media.press.pressed(i), Inside: inside})
	}
	if a.media.message != "" {
		font.Draw(dst, a.flow.menuDisplayText(a.media.message), movieListArg.Min.X, 350, townShellText)
	}
	return dst
}

func drawMovieBox(dst *image.RGBA, r image.Rectangle, selected bool) {
	c := color.RGBA{0, 0, 0, 45}
	if selected {
		c = color.RGBA{0, 7, 6, 220}
	}
	draw.Draw(dst, r, &image.Uniform{C: c}, image.Point{}, draw.Over)
	outline(dst, r, color.RGBA{57, 77, 65, 255})
}

func (a *App) openCredits(back Screen) {
	a.media.roll = CreditsView{}
	if a.media.credits != nil {
		a.media.roll = a.media.credits()
	}
	a.media.back, a.media.at, a.media.steps = back, time.Time{}, 0
	a.flow.setScreen(ScreenCredits)
}

func (a *App) closeCredits() {
	a.flow.setScreen(a.media.back)
	a.media.roll = CreditsView{}
	// A key/click which leaves the roll belongs entirely to that screen.
	a.cutsceneDrain = true
	a.suppressPrimaryRelease, a.suppressSecondaryRelease = true, true
}

// creditsStart is the roll's start offset in pixels and creditsStepGap the
// shortest time between two one-pixel steps (FAME-030).
const (
	creditsStart   = 480
	creditsStepGap = 23 * time.Millisecond
)

// creditsScroll is the number of one-pixel steps the roll has taken.
func (a *App) creditsScroll() int { return a.media.steps }

// creditsPitch is the installed font's own line height (15 in both
// installs); 15 stands in when no font is installed.
func (a *App) creditsPitch() int {
	if h := a.flow.menuFont.Height(); h > 0 {
		return h
	}
	return 15
}

func (a *App) composeCredits() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	draw.Draw(dst, dst.Bounds(), image.Black, image.Point{}, draw.Src)
	font := a.flow.menuFont
	if font == nil {
		return dst
	}
	if len(a.media.roll.Lines) == 0 {
		drawTownShellText(dst, font, a.flow.menuDisplayText(a.flow.endingWords().MissingCredits), image.Rect(20, 220, 620, 260), townShellText)
		return dst
	}
	for i, line := range a.media.roll.Lines {
		y := creditsStart - a.creditsScroll() + i*a.creditsPitch()
		if logo := a.media.roll.Logos[strings.ToLower(strings.TrimSpace(line))]; logo != nil {
			copyNative(dst, logo, image.Pt((frame.W-logo.Bounds().Dx())/2, y), dst.Bounds())
		} else if y > -a.creditsPitch() && y < frame.H && line != "" {
			drawTownShellText(dst, font, line, image.Rect(8, y, frame.W-8, y+a.creditsPitch()), color.RGBA{255, 255, 255, 255})
		}
	}
	return dst
}

func (a *App) headlessCutscene(target string) error {
	for i, entry := range a.seenCutscenes() {
		if !strings.EqualFold(target, entry.Title) && !(target == "@first" && i == 0) {
			continue
		}
		for n := 0; n < len(a.seenCutscenes()) && a.movieList().Selection() != i; n++ {
			key := "down"
			if a.movieList().Selection() > i {
				key = "up"
			}
			if err := a.HeadlessKey(key); err != nil {
				return err
			}
		}
		return a.HeadlessKey("enter")
	}
	return fmt.Errorf("cutscenes has no encountered movie %q", target)
}
