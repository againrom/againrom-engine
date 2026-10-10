package ui

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"slices"
	"time"

	"againrom/pkg/render/frame"
	"againrom/pkg/video"
	"github.com/hajimehoshi/ebiten/v2"
)

// CutsceneSource owns source selection and the movie decoder.
// Refusal means the existing screen stays playable.
type CutsceneSource interface {
	Open(name string) (*video.Player, error)
}

func (a *App) SetCutscenes(source CutsceneSource) { a.cutsceneSource = source }

// SetStartupCutscenes names the movies that play before the menu, in order;
// none plays nothing.
func (a *App) SetStartupCutscenes(names []string) { a.startupCutscenes = slices.Clone(names) }

// SetMissionCutscene identifies the movie family for a completed mission.
// Loading or entering that mission does not request its movie.
func (v *Viewer) SetMissionCutscene(mission int) {
	v.missionCutscene = mission
	v.completionCutscene = ""
	if mission > 0 {
		v.completionCutscene = fmt.Sprintf("m%d", mission)
	}
}

func (v *Viewer) SetCompletionCutscene(directory string) {
	v.missionCutscene = 0
	v.completionCutscene = directory
}

func numberedCutscenes(directory string) []string {
	names := make([]string, 99)
	for n := range names {
		names[n] = fmt.Sprintf("%s/%02d.smk", directory, n+1)
	}
	return names
}

// PlayStartupCutscenes precedes the menu with the installed logos and intro.
// Missing clips remain optional.
func (a *App) PlayStartupCutscenes() bool {
	if a == nil || a.flow.screen != ScreenMenu || len(a.startupCutscenes) == 0 {
		return false
	}
	return a.PlayCutsceneSequence(a.startupCutscenes)
}

func (a *App) startPendingCutscene() {
	if directory := a.flow.completedCutscene; directory != "" {
		a.flow.completedCutscene = ""
		a.PlayCutsceneSequence(numberedCutscenes(directory))
		return
	}
	v := a.flow.viewer
	if a.flow.screen != ScreenMap || v == nil || !v.entryCampaignStart {
		return
	}
	v.entryCampaignStart = false
	a.PlayCutsceneSequence(numberedCutscenes("start"))
}

// PlayCutscene retains the existing destination screen. No input, map tick or
// save operation is dispatched beneath it. Replacement closes the prior
// decoder even when the next source refuses.
func (a *App) PlayCutscene(name string) bool {
	return a.PlayCutsceneSequence([]string{name})
}

// PlayCutsceneSequence scans at most 256 requests, allowing startup prefixes
// and two bounded numbered families in one event. Completion and refusal
// continue; skip cancels the entire scan (VIDEO-030/036).
func (a *App) PlayCutsceneSequence(names []string) bool {
	if a == nil {
		return false
	}
	a.StopCutscene()
	a.cutsceneError = nil
	a.cutsceneDrain = false
	if len(names) > 256 {
		names = names[:256]
	}
	a.cutsceneQueue = append([]string(nil), names...)
	return a.nextCutscene()
}

func (a *App) nextCutscene() bool {
	a.closeCutscene()
	if a.cutsceneSource == nil {
		a.cutsceneQueue = nil
		return false
	}
	for len(a.cutsceneQueue) > 0 {
		name := a.cutsceneQueue[0]
		a.cutsceneQueue = a.cutsceneQueue[1:]
		p, err := a.cutsceneSource.Open(name)
		if err != nil || p == nil {
			if p != nil {
				p.Close()
			}
			if err != nil && !errors.Is(err, video.ErrAbsent) {
				a.cutsceneError = err
			}
			continue
		}
		a.cutscene = p
		a.cutsceneName = name
		a.clearShopDrag()
		if v := a.flow.viewer; v != nil {
			v.StopAmbient()
			v.clearAttackMode()
			v.dragging, v.boxing, v.rightDragging = false, false, false
			v.primaryDown, v.secondaryDown, v.rightPanned = false, false, false
			v.ctrlLatch, v.altLatch, v.shiftLatch = false, false, false
		}
		a.syncMusic()
		return true
	}
	a.syncMusic()
	return false
}

// CutsceneError is diagnostic only; an optional movie never opens an error
// modal over the destination it was meant to precede.
func (a *App) CutsceneError() error { return a.cutsceneError }

// CutsceneName identifies the current request for runtime witnesses.
func (a *App) CutsceneName() string { return a.cutsceneName }

func (a *App) CutsceneFrameNumber() uint32 { return a.cutscene.FrameNumber() }

// CutsceneAudioFormat is the current movie's own rate and channel count, or
// 0,0 before the first frame arrives and for a movie whose decode carries no
// audio track. It is a read-only witness accessor; nothing here starts,
// stops or otherwise drives playback.
func (a *App) CutsceneAudioFormat() (rate uint32, channels uint8) {
	if a == nil {
		return 0, 0
	}
	return a.cutscene.AudioFormat()
}

// HeadlessCutsceneStep drives the same presentation/input dispatch with real
// wall time. Deterministic simulation scenarios keep their separate clock.
func (a *App) HeadlessCutsceneStep(skip string) error {
	in := appInput{}
	switch skip {
	case "":
	case "key":
		in.AnyKey = true
	case "left":
		in.PrimaryPressed = true
	case "right":
		in.SecondaryPressed = true
	case "close":
		in.Close = true
	default:
		return fmt.Errorf("video: unknown skip input %q", skip)
	}
	if a.step(in, time.Now()) {
		return fmt.Errorf("video: input exited application")
	}
	return nil
}

func (a *App) StopCutscene() {
	if a == nil {
		return
	}
	a.cutsceneQueue = nil
	a.closeCutscene()
	a.syncMusic()
}

func (a *App) closeCutscene() {
	// Every skip route and every natural completion or failure reaches this one
	// teardown seam, so stopping the streamed track here -- rather than at each
	// of those call sites -- is what makes "every skip route stops the audio
	// immediately" a property of one place instead of four
	// (key/left/right/close all route through stepCutscene's StopCutscene,
	// below).
	a.stopCutsceneAudio()
	if a.cutscene == nil {
		return
	}
	a.cutscene.Close()
	a.cutscene = nil
	a.cutsceneFinalPresented = false
	a.cutsceneName = ""
	a.cutsceneDrain = true
	if a.cutsceneCanvas != nil {
		a.cutsceneCanvas.Deallocate()
		a.cutsceneCanvas = nil
	}
	// Discard presentation wall time at the map's existing cadence seam.
	// This preserves the selected rung and stop bit, including a prior pause.
	a.flow.syncCadence(true)
}

func (a *App) stepCutscene(in appInput, now time.Time) {
	if in.Close || (!in.Unfocused && (in.AnyKey || in.Escape || in.Enter || in.Panels || in.PrimaryPressed || in.SecondaryPressed)) {
		if in.PrimaryPressed {
			a.suppressPrimaryRelease = true
		}
		if in.SecondaryPressed {
			a.suppressSecondaryRelease = true
		}
		a.StopCutscene()
		return
	}
	if a.cutscene.FinalFrame() && !a.cutsceneFinalPresented {
		return
	}
	if !a.cutscene.Advance(now) {
		if err := a.cutscene.Err(); err != nil {
			a.cutsceneError = err
		}
		a.nextCutscene()
		return
	}
	a.pumpCutsceneAudio()
	if a.cutscene.FrameNumber() > 0 {
		a.encounterCutscene(a.cutsceneName)
	}
}

// composeCutscene is also the headless witness: the presented movie frame
// (already windowed, doubled, faded and panned by pkg/video from its sidecar)
// is fitted aspect-preserving, nearest neighbour, inside a black 640x480 frame.
// A 640x360 frame lands at a 60 row offset unscaled, a 640x480 frame fills it.
func (a *App) composeCutscene() *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, frame.W, frame.H))
	for n := 3; n < len(dst.Pix); n += 4 {
		dst.Pix[n] = 255
	}
	src := a.cutscene.Frame()
	if src == nil || src.Rect.Empty() {
		return dst
	}
	sw, sh := src.Rect.Dx(), src.Rect.Dy()
	w, h := frame.W, sh*frame.W/sw
	if h > frame.H {
		h, w = frame.H, sw*frame.H/sh
	}
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	x0, y0 := (frame.W-w)/2, (frame.H-h)/2
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := src.RGBAAt(src.Rect.Min.X+x*sw/w, src.Rect.Min.Y+y*sh/h)
			c.A = 255
			dst.SetRGBA(x0+x, y0+y, c)
		}
	}
	return dst
}

func (a *App) drawCutscene(screen *ebiten.Image) {
	ebiten.SetCursorMode(ebiten.CursorModeHidden)
	// The ordinary draw path sees this same cache on the first restored frame.
	a.flow.cursor.SetPointerHidden(true)
	screen.Fill(color.Black)
	if a.cutsceneCanvas == nil {
		a.cutsceneCanvas = ebiten.NewImage(frame.W, frame.H)
	}
	a.cutsceneCanvas.WritePixels(a.composeCutscene().Pix)
	if a.place.Valid() {
		ox, oy := a.place.Origin()
		var op ebiten.DrawImageOptions
		op.GeoM.Scale(a.place.Scale(), a.place.Scale())
		op.GeoM.Translate(ox, oy)
		op.Filter = ebiten.FilterNearest
		// The cutscene frame shares a.place with the town composite
		// (app.go), so it shares its scaler (owner direction).
		drawFinalFrame(screen, a.cutsceneCanvas, &op, &cutsceneSharpBuf, a.frameSmoothingOff)
		a.finishCutscenePresentation()
	}
}

func (a *App) finishCutscenePresentation() {
	if a.cutscene != nil && a.cutscene.FinalFrame() && !a.cutsceneFinalPresented {
		a.cutsceneFinalPresented = true
		a.haltCutsceneAudio()
	}
}
