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
	catalog               []CutsceneEntry
	seen                  map[string]bool
	record                func(string) error
	words                 CutsceneLibraryWords
	selection, top, press int
	message               string
	credits               func() CreditsView
	roll                  CreditsView
	scroll                []*image.RGBA
	back                  Screen
	at                    time.Time // last roll step; zero before the first
	steps                 int       // pixels the roll has advanced
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
var movieList = image.Rect(170, 134, 464, 334)
var movieOK = image.Rect(172, 376, 296, 400)
var movieCancel = image.Rect(348, 376, 472, 400)
var movieUp = image.Rect(466, 134, 490, 158)
var movieDown = image.Rect(466, 310, 490, 334)
var movieTrack = image.Rect(466, 158, 490, 310)

const movieVisibleRows = 10

func (a *App) openCutsceneLibrary() {
	a.media.selection, a.media.top, a.media.press = 0, 0, -1
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
	n := len(a.seenCutscenes())
	a.media.selection = min(max(0, index), max(0, n-1))
	if a.media.selection < a.media.top {
		a.media.top = a.media.selection
	}
	if a.media.selection >= a.media.top+movieVisibleRows {
		a.media.top = a.media.selection - movieVisibleRows + 1
	}
}

func (a *App) replayMovie() {
	entries := a.seenCutscenes()
	if len(entries) == 0 {
		return
	}
	a.media.message = ""
	if !a.PlayCutsceneSequence(numberedCutscenes(entries[a.media.selection].Directory)) {
		a.media.message = a.mediaMessages().Unavailable
	}
}

func movieHit(p image.Point, top, count int) int {
	for i, r := range []image.Rectangle{movieOK, movieCancel, movieUp, movieDown, movieTrack} {
		if p.In(r) {
			return 100 + i
		}
	}
	if p.In(movieList) {
		i := top + (p.Y-movieList.Min.Y)/20
		if i < count {
			return i
		}
	}
	return -1
}

func (a *App) stepMedia(in appInput, now time.Time) {
	if in.Unfocused {
		a.media.press = -1
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
	if in.Up {
		a.selectMovie(a.media.selection - 1)
	}
	if in.Down {
		a.selectMovie(a.media.selection + 1)
	}
	if in.Home {
		a.selectMovie(0)
	}
	if in.End {
		a.selectMovie(len(a.seenCutscenes()) - 1)
	}
	if in.WheelY > 0 {
		a.selectMovie(a.media.selection - 3)
	}
	if in.WheelY < 0 {
		a.selectMovie(a.media.selection + 3)
	}
	if in.Enter {
		a.replayMovie()
		return
	}
	p, ok := a.windowToNativeFrame(in.CursorX, in.CursorY)
	hit := -1
	if ok {
		hit = movieHit(p, a.media.top, len(a.seenCutscenes()))
	}
	if in.PrimaryPressed {
		a.media.press = hit
	}
	if in.PrimaryReleased {
		pressed := a.media.press
		a.media.press = -1
		if hit < 0 || hit != pressed {
			return
		}
		a.playUISound(UISoundCommonControl)
		switch hit {
		case 100:
			a.replayMovie()
		case 101:
			a.flow.toMenu()
		case 102:
			a.selectMovie(a.media.selection - 1)
		case 103:
			a.selectMovie(a.media.selection + 1)
		case 104:
			a.selectMovie((p.Y - movieTrack.Min.Y) * max(0, len(a.seenCutscenes())-1) / movieTrack.Dy())
		default:
			a.selectMovie(hit)
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
	for i := 0; i < movieVisibleRows; i++ {
		r := image.Rect(movieList.Min.X, movieList.Min.Y+i*20, movieList.Max.X, movieList.Min.Y+(i+1)*20)
		drawMovieBox(dst, r, a.media.top+i == a.media.selection && len(entries) > 0)
		index := a.media.top + i
		if index < len(entries) {
			c := townShellText
			if index == a.media.selection {
				c = color.RGBA{218, 183, 71, 255}
			}
			clip := dst.SubImage(r.Inset(2)).(*image.RGBA)
			font.Draw(clip, entries[index].Title, r.Min.X+4, r.Min.Y+2, c)
		}
	}
	if len(entries) == 0 {
		font.Draw(dst, a.flow.menuDisplayText(a.mediaMessages().Empty), movieList.Min.X+4, 350, townShellText)
	}
	for i, r := range []image.Rectangle{movieOK, movieCancel, movieUp, movieDown} {
		drawMovieBox(dst, r, a.media.press == 100+i)
		label := []string{a.media.words.OK, a.media.words.Cancel, "^", "v"}[i]
		drawTownShellText(dst, font, label, r, townShellText)
	}
	drawMovieBox(dst, movieTrack, false)
	y := movieTrack.Min.Y
	if len(entries) > 1 {
		y += a.media.selection * (movieTrack.Dy() - 24) / (len(entries) - 1)
	}
	if len(a.media.scroll) >= 26 {
		for y := movieTrack.Min.Y; y < movieTrack.Max.Y; y += 24 {
			copyNativeOver(dst, a.media.scroll[19], image.Pt(movieTrack.Min.X, y), movieTrack)
		}
		copyNativeOver(dst, a.media.scroll[18], movieUp.Min, dst.Bounds())
		copyNativeOver(dst, a.media.scroll[20], movieDown.Min, dst.Bounds())
		copyNativeOver(dst, a.media.scroll[16], image.Pt(movieTrack.Min.X, y), dst.Bounds())
	} else {
		drawMovieBox(dst, image.Rect(468, y, 488, y+24), true)
	}
	if a.media.message != "" {
		font.Draw(dst, a.flow.menuDisplayText(a.media.message), movieList.Min.X, 350, townShellText)
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
		for n := 0; n < len(a.seenCutscenes()) && a.media.selection != i; n++ {
			key := "down"
			if a.media.selection > i {
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
