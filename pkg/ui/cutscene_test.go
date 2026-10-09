package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"io"
	"testing"
	"time"

	"againrom/pkg/random"
	"againrom/pkg/video"
)

type testCutsceneSource struct {
	stream io.ReadCloser
	err    error
	names  []string
}

func (s *testCutsceneSource) Open(name string) (*video.Player, error) {
	s.names = append(s.names, name)
	if s.err != nil {
		return nil, s.err
	}
	return video.NewPlayer(s.stream), nil
}

func syntheticMovie() []byte {
	// Two horizontal pixels, two frames, authored independently of encoders.
	return []byte{'A', 'R', 'V', '2', 2, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		255, 0, 0, 255, 0, 255, 0, 255, 0, 0, 255, 255, 255, 255, 255, 255}
}

func TestCutsceneComposesAuthoredColoursAndReturnsAfterEOF(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetCutscenes(&testCutsceneSource{stream: io.NopCloser(bytes.NewReader(syntheticMovie()))})
	if !a.PlayCutscene("fixture") || a.Screen() != ScreenCutscene {
		t.Fatal("not showing movie")
	}
	first, second := false, false
	deadline := time.Now().Add(time.Second)
	for a.Screen() == ScreenCutscene && time.Now().Before(deadline) {
		a.step(appInput{}, time.Now())
		if a.Screen() != ScreenCutscene {
			break
		}
		pix, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatalf("composition %s %v", note, err)
		}
		if pix.RGBAAt(0, 0) != (color.RGBA{0, 0, 0, 255}) {
			t.Fatal("letterbox is not opaque black")
		}
		left, right := pix.RGBAAt(159, 240), pix.RGBAAt(480, 240)
		if left == (color.RGBA{255, 0, 0, 255}) && right == (color.RGBA{0, 255, 0, 255}) {
			first = true
		}
		if left == (color.RGBA{0, 0, 255, 255}) && right == (color.RGBA{255, 255, 255, 255}) {
			second = true
		}
		time.Sleep(time.Millisecond)
	}
	if !first || !second || a.Screen() != ScreenMenu || a.CutsceneError() != nil {
		t.Fatalf("frames %v/%v screen %v error %v", first, second, a.Screen(), a.CutsceneError())
	}
}

func TestCutscenePointerReleaseAndIdleUseTheSameClock(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetCutscenes(&testCutsceneSource{stream: io.NopCloser(bytes.NewReader(syntheticMovie()))})
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	before := a.headlessClock
	if !a.PlayCutscene("fixture") {
		t.Fatal("movie did not start")
	}
	defer a.StopAudio()
	if err := a.HeadlessPointer("release", 0, 0); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessCutsceneStep(""); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenCutscene || a.CutsceneError() != nil || a.headlessClock != before {
		t.Fatal("mixed input clocks stalled the movie or advanced the input clock", a.CutsceneError())
	}
}

func TestCutsceneSkipConsumesPressAndReleaseAndNeverDispatchesMapTick(t *testing.T) {
	for _, key := range []appInput{{AnyKey: true}, {Escape: true}, {Enter: true}, {Panels: true}, {PrimaryPressed: true}, {SecondaryPressed: true}} {
		a := newTestApp(t, appRows(1), okLoader(t))
		if err := a.OpenMission(okOpener(t)); err != nil {
			t.Fatal(err)
		}
		ticks := 0
		a.flow.tick = func() { ticks++ }
		r, w := io.Pipe()
		a.SetCutscenes(&testCutsceneSource{stream: r})
		a.PlayCutscene("fixture")
		a.step(appInput{}, time.Now())
		if a.step(key, time.Now()) || ticks != 0 || a.Screen() != ScreenMap {
			t.Fatalf("skip leaked: %v %d %v", key, ticks, a.Screen())
		}
		if key.PrimaryPressed && !a.suppressPrimaryRelease {
			t.Fatal("release was not retained")
		}
		if key.SecondaryPressed && !a.suppressSecondaryRelease {
			t.Fatal("secondary release was not retained")
		}
		_ = w.Close()
	}
}

func TestCutsceneMissingAndStallAreFailOpenAndMusicReturns(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	device := &recordingMusicDevice{}
	a.SetMusic(&recordingMusicSource{}, device, random.NewStream(1074))
	a.SetCutscenes(&testCutsceneSource{err: errors.New("missing")})
	if a.PlayCutscene("missing") || a.Screen() != ScreenMenu || a.CutsceneError() == nil {
		t.Fatal("missing media blocked menu")
	}
	r, w := io.Pipe()
	a.SetCutscenes(&testCutsceneSource{stream: r})
	a.PlayCutscene("stall")
	if scene, _, _ := a.musicScene(); scene != MusicSilent {
		t.Fatal("music continues under movie")
	}
	now := time.Unix(10, 0)
	a.step(appInput{Unfocused: true, Escape: true}, now)
	if a.Screen() != ScreenCutscene {
		t.Fatal("unfocused key skipped")
	}
	a.step(appInput{}, now.Add(video.StallLimit))
	if a.Screen() != ScreenMenu || a.CutsceneError() == nil {
		t.Fatal("stall blocked menu")
	}
	if scene, _, _ := a.musicScene(); scene != MusicMenu {
		t.Fatal("music not restored")
	}
	_ = w.Close()
}

func TestCutsceneAudioTeardownDoesNotRestartMusic(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	device := &recordingMusicDevice{}
	a.SetMusic(&recordingMusicSource{}, device, random.NewStream(1074))
	r, w := io.Pipe()
	a.SetCutscenes(&testCutsceneSource{stream: r})
	a.PlayCutscene("fixture")
	a.StopAudio()
	if device.current.StereoPCM != nil || a.cutscene != nil {
		t.Fatal("teardown retains movie/music")
	}
	_ = w.Close()
}

type scanCutsceneSource struct {
	names []string
	fail  map[string]error
}

func (s *scanCutsceneSource) Open(name string) (*video.Player, error) {
	s.names = append(s.names, name)
	if err := s.fail[name]; err != nil {
		return nil, err
	}
	return video.NewPlayer(io.NopCloser(bytes.NewReader(syntheticMovie()))), nil
}

func TestCutsceneNumberedScanContinuesMissFailureCompletionButSkipStops(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	s := &scanCutsceneSource{fail: map[string]error{"m10/01.smk": video.ErrAbsent, "m10/02.smk": errors.New("decode open")}}
	a.SetCutscenes(s)
	a.PlayCutsceneSequence([]string{"m10/01.smk", "m10/02.smk", "m10/03.smk", "m10/04.smk", "m10/05.smk"})
	deadline := time.Now().Add(time.Second)
	for len(s.names) < 4 && time.Now().Before(deadline) {
		a.step(appInput{}, time.Now())
		if a.Screen() == ScreenCutscene {
			if _, _, err := a.HeadlessFrame(); err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(time.Millisecond)
	}
	if len(s.names) != 4 {
		t.Fatalf("natural completion did not continue: %v", s.names)
	}
	a.step(appInput{Close: true}, time.Now())
	if len(s.names) != 4 || a.Screen() != ScreenMenu {
		t.Fatalf("skip continued: %v %v", s.names, a.Screen())
	}
}

func TestCutsceneCompletedMissionUsesBoundedNumberedScanOnce(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	s := &scanCutsceneSource{fail: make(map[string]error)}
	for n := 1; n <= 99; n++ {
		s.fail[fmt.Sprintf("m10/%02d.smk", n)] = video.ErrAbsent
	}
	a.SetCutscenes(s)
	v, tick, order, cadence, affect, advance, attack, grab, stance, march, err := okOpener(t)()
	if err != nil {
		t.Fatal(err)
	}
	v.SetMissionCutscene(10)
	a.flow.enter(v, tick, order, cadence, affect, advance, attack, grab, stance, march)
	a.startPendingCutscene()
	if len(s.names) != 0 {
		t.Fatal("fresh mission requested its completion movie")
	}
	a.flow.advance = func(...NoticeAction) (NoticeDest, string, MapOpener) { return NoticeToMapList, "", nil }
	a.flow.takeNoticeAction(NoticeVictory)
	a.startPendingCutscene()
	a.startPendingCutscene()
	if len(s.names) != 99 || s.names[0] != "m10/01.smk" || s.names[98] != "m10/99.smk" || a.Screen() != ScreenPicker {
		t.Fatalf("scan %v screen %v", s.names, a.Screen())
	}
}

func TestCutsceneDrainsHeldKeyboardAndMouseBeforeMapInput(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	if err := a.OpenMission(okOpener(t)); err != nil {
		t.Fatal(err)
	}
	ticks := 0
	a.flow.tick = func() { ticks++ }
	r, w := io.Pipe()
	defer w.Close()
	a.SetCutscenes(&testCutsceneSource{stream: r})
	a.PlayCutscene("fixture")
	a.step(appInput{AnyKey: true, AnyHeld: true}, time.Now())
	for n := 0; n < 3; n++ {
		a.step(appInput{AnyHeld: true, AttackHeld: true, Viewer: Input{PrimaryDown: true, PanRight: true}}, time.Now())
	}
	if ticks != 0 || !a.cutsceneDrain {
		t.Fatal("held skip gesture reached the map")
	}
	a.step(appInput{PrimaryReleased: true}, time.Now())
	if a.cutsceneDrain || ticks != 0 {
		t.Fatal("release did not drain")
	}
}

func TestCutsceneF2F3SkipPrecedesSaveLoadAndDropsOldDrags(t *testing.T) {
	for _, key := range []string{"f2", "f3"} {
		a := newTestApp(t, appRows(1), okLoader(t))
		if err := a.OpenMission(okOpener(t)); err != nil {
			t.Fatal(err)
		}
		v := a.flow.viewer
		v.SetGameMenuContext(func() GameMenuContext { return GameMenuContext{Campaign: true} })
		v.dragging, v.boxing, v.rightDragging = true, true, true
		v.primaryDown, v.secondaryDown, v.rightPanned = true, true, true
		spy := &spySaveSeams{}
		spy.install(a)
		r, w := io.Pipe()
		a.SetCutscenes(&testCutsceneSource{stream: r})
		a.PlayCutscene("fixture")
		if v.dragging || v.boxing || v.rightDragging || v.primaryDown || v.secondaryDown || v.rightPanned {
			t.Fatal("overlay retained a map gesture")
		}
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if a.Screen() != ScreenMap || len(spy.saved) != 0 || len(spy.loaded) != 0 {
			t.Fatalf("%s dispatched beneath overlay", key)
		}
		w.Close()
	}
}

func TestCutsceneFitMatchesLiteralColoredRectangles(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetCutscenes(&testCutsceneSource{stream: io.NopCloser(bytes.NewReader(syntheticMovie()))})
	a.PlayCutscene("fixture")
	defer a.StopCutscene()
	deadline := time.Now().Add(time.Second)
	for a.CutsceneFrameNumber() == 0 && time.Now().Before(deadline) {
		a.step(appInput{}, time.Now())
		time.Sleep(time.Millisecond)
	}
	if a.CutsceneFrameNumber() != 1 {
		t.Fatal("missing literal fixture frame")
	}
	got, _, err := a.HeadlessFrame()
	if err != nil {
		t.Fatal(err)
	}
	want := image.NewRGBA(image.Rect(0, 0, 640, 480))
	draw.Draw(want, want.Bounds(), image.NewUniform(color.RGBA{0, 0, 0, 255}), image.Point{}, draw.Src)
	draw.Draw(want, image.Rect(0, 80, 320, 400), image.NewUniform(color.RGBA{255, 0, 0, 255}), image.Point{}, draw.Src)
	draw.Draw(want, image.Rect(320, 80, 640, 400), image.NewUniform(color.RGBA{0, 255, 0, 255}), image.Point{}, draw.Src)
	if !bytes.Equal(got.Pix, want.Pix) {
		t.Fatal("composition differs from independent literal rectangles")
	}
}
