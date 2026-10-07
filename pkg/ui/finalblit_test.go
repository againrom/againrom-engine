package ui

import (
	"bytes"
	"image/color"
	"io"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/video"
)

var queuedTwoFrameMovie = []byte{
	'A', 'R', 'V', '2', 2, 0, 0, 0, 1, 0, 0, 0, 2, 0, 0, 0, 34, 86, 0, 0, 2, 16, 0, 0,
	255, 0, 0, 255, 0, 255, 0, 255, 4, 0, 0, 0, 11, 12, 13, 14,
	0, 0, 255, 255, 255, 255, 255, 255, 4, 0, 0, 0, 21, 22, 23, 24,
}

type countedMovieInput struct {
	io.ReadCloser
	once   sync.Once
	closes atomic.Int32
}

func (r *countedMovieInput) Close() error {
	var err error
	r.once.Do(func() { r.closes.Add(1); err = r.ReadCloser.Close() })
	return err
}

type movieQueueRecorder struct {
	device *cutsceneAudioDevice
	queue  *cutsceneAudioQueue
	starts int
	stops  int
	chunks [][]byte
}

func (r *movieQueueRecorder) Start(rate, channels int) {
	r.starts++
	r.device.Start(rate, channels)
	r.queue = r.device.s.q
}
func (r *movieQueueRecorder) Push(p []byte) {
	r.chunks = append(r.chunks, append([]byte(nil), p...))
	r.device.Push(p)
}
func (r *movieQueueRecorder) Stop()                        { r.stops++; r.device.Stop() }
func (r *movieQueueRecorder) SetSettings(s audio.Settings) { r.device.SetSettings(s) }

func delayedMovieInput(t *testing.T) (*countedMovieInput, func(), func()) {
	t.Helper()
	r, w := io.Pipe()
	last, eof := make(chan struct{}), make(chan struct{})
	var lastOnce, eofOnce sync.Once
	releaseLast := func() { lastOnce.Do(func() { close(last) }) }
	releaseEOF := func() { eofOnce.Do(func() { close(eof) }) }
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer w.Close()
		if _, err := w.Write(queuedTwoFrameMovie[:40]); err != nil {
			return
		}
		<-last
		if _, err := w.Write(queuedTwoFrameMovie[40:]); err != nil {
			return
		}
		<-eof
	}()
	input := &countedMovieInput{ReadCloser: r}
	t.Cleanup(func() {
		input.Close()
		releaseLast()
		releaseEOF()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Error("movie writer survived cleanup")
		}
	})
	return input, releaseLast, releaseEOF
}

func advanceMovieTo(t *testing.T, a *App, n uint32) {
	t.Helper()
	waitFor(t, "requested movie frame", func() bool {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
		return a.CutsceneFrameNumber() == n
	})
}

func assertMoviePicture(t *testing.T, a *App, left, right color.RGBA) []byte {
	t.Helper()
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("movie composition", note, err)
	}
	if pix.RGBAAt(159, 240) != left || pix.RGBAAt(480, 240) != right {
		t.Fatal("movie lost independently authored final pixels")
	}
	return append([]byte(nil), pix.Pix...)
}

func TestFinalCutscenePresentationStopsQueuedAudioAndRetainsPicture(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	input, releaseLast, releaseEOF := delayedMovieInput(t)
	a.SetCutscenes(&testCutsceneSource{stream: input})
	device, backend := heldCutsceneDevice(audio.Settings{Master: 100}, 64)
	rec := &movieQueueRecorder{device: device}
	a.SetCutsceneAudio(rec)
	t.Cleanup(a.StopAudio)
	if !a.PlayCutscene("finite") {
		t.Fatal("movie did not open")
	}
	rec.stops = 0
	advanceMovieTo(t, a, 1)
	assertMoviePicture(t, a, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255})
	if rec.stops != 0 || rec.starts != 1 || device.s == nil {
		t.Fatal("penultimate presentation stopped audio")
	}
	for i := 0; i < 3; i++ {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
		assertMoviePicture(t, a, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255})
	}
	if rec.stops != 0 {
		t.Fatal("waiting for a slow final frame stopped audio")
	}
	releaseLast()
	advanceMovieTo(t, a, 2)
	if rec.stops != 0 || len(rec.chunks) != 2 || !bytes.Equal(rec.chunks[1], []byte{21, 22, 23, 24}) {
		t.Fatal("final PCM was suppressed before its picture was presented")
	}
	rec.queue.mu.Lock()
	pending := append([]byte(nil), rec.queue.buf...)
	rec.queue.mu.Unlock()
	if !bytes.Equal(pending, []byte{11, 12, 13, 14, 21, 22, 23, 24}) {
		t.Fatal("production queue lost final PCM before presentation", pending)
	}
	last := a.cutscene.Frame()
	finalPixels := assertMoviePicture(t, a, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255})
	if rec.stops != 1 || device.s != nil {
		t.Fatalf("final blit kept audio queue live: stops=%d attached=%t", rec.stops, device.s != nil)
	}
	if n, err := rec.queue.Read(make([]byte, 16)); n != 0 || err != io.EOF {
		t.Fatal("stopped final queue still delivered PCM", n, err)
	}
	if a.cutscene.Frame() != last || input.closes.Load() != 0 {
		t.Fatal("audio stop disposed the final picture or stream")
	}
	for i := 0; i < 3; i++ {
		if err := a.HeadlessCutsceneStep(""); err != nil {
			t.Fatal(err)
		}
		if got := assertMoviePicture(t, a, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255}); !bytes.Equal(got, finalPixels) {
			t.Fatal("retained final picture changed")
		}
	}
	if rec.stops != 1 || rec.starts != 1 || len(rec.chunks) != 2 {
		t.Fatal("duplicate final Draw/Update replayed or stopped audio twice")
	}
	releaseEOF()
	waitFor(t, "movie EOF", func() bool { a.HeadlessCutsceneStep(""); return a.Screen() == ScreenMenu })
	if rec.stops != 1 || input.closes.Load() != 1 {
		t.Fatal("EOF did not dispose the stream once after final audio stop")
	}
	waitFor(t, "movie backend close", func() bool { calls := backend.calls(); return len(calls) > 0 && calls[len(calls)-1] == "close" })
	calls := backend.calls()
	if len(calls) != 4 || calls[1] != "play" || calls[2] != "pause" || calls[3] != "close" {
		t.Fatal("backend disposal order", calls)
	}
}

func TestFinalCutscenePictureSurvivesUpdatesBeforePresentation(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	input, releaseLast, releaseEOF := delayedMovieInput(t)
	a.SetCutscenes(&testCutsceneSource{stream: input})
	t.Cleanup(a.StopAudio)
	if !a.PlayCutscene("finite") {
		t.Fatal("movie did not open")
	}
	advanceMovieTo(t, a, 1)
	releaseLast()
	advanceMovieTo(t, a, 2)
	releaseEOF()
	for i := 0; i < 10; i++ {
		a.HeadlessCutsceneStep("")
		time.Sleep(time.Millisecond)
	}
	if a.Screen() != ScreenCutscene || input.closes.Load() != 0 {
		t.Fatal("EOF disposed an undrawn final frame")
	}
	assertMoviePicture(t, a, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255})
	waitFor(t, "presented final EOF", func() bool { a.HeadlessCutsceneStep(""); return a.Screen() == ScreenMenu })
}

func TestStalledCutsceneStopsAudioWithoutFinalPresentation(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	input, _, _ := delayedMovieInput(t)
	a.SetCutscenes(&testCutsceneSource{stream: input})
	device, backend := heldCutsceneDevice(audio.Settings{Master: 100}, 64)
	rec := &movieQueueRecorder{device: device}
	a.SetCutsceneAudio(rec)
	t.Cleanup(a.StopAudio)
	if !a.PlayCutscene("finite") {
		t.Fatal("movie did not open")
	}
	rec.stops = 0
	advanceMovieTo(t, a, 1)
	assertMoviePicture(t, a, color.RGBA{255, 0, 0, 255}, color.RGBA{0, 255, 0, 255})
	a.step(appInput{}, time.Now().Add(video.StallLimit))
	if a.Screen() != ScreenMenu || a.CutsceneError() == nil || rec.stops != 1 || device.s != nil || input.closes.Load() != 1 {
		t.Fatal("stalled final frame retained its stream or audio queue")
	}
	waitFor(t, "stalled backend close", func() bool { calls := backend.calls(); return len(calls) > 0 && calls[len(calls)-1] == "close" })
}

type queuedMovieSequence struct {
	inputs []*countedMovieInput
}

func (s *queuedMovieSequence) Open(string) (*video.Player, error) {
	r := &countedMovieInput{ReadCloser: io.NopCloser(bytes.NewReader(queuedTwoFrameMovie))}
	s.inputs = append(s.inputs, r)
	return video.NewPlayer(r), nil
}

func TestFinalCutscenePresentationRoutesToTheNextMovieOnce(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	source := &queuedMovieSequence{}
	a.SetCutscenes(source)
	device, _ := heldCutsceneDevice(audio.Settings{Master: 100}, 64)
	var backends []*heldPlayer
	device.newPlayer = func(src io.Reader) (cutscenePlayer, error) {
		p := &heldPlayer{src: &heldFillSource{q: src.(*cutsceneAudioQueue), fill: 64}, fill: 64, playing: make(chan struct{})}
		backends = append(backends, p)
		return p, nil
	}
	rec := &movieQueueRecorder{device: device}
	a.SetCutsceneAudio(rec)
	t.Cleanup(a.StopAudio)
	if !a.PlayCutsceneSequence([]string{"first", "second"}) {
		t.Fatal("sequence did not open")
	}
	rec.stops = 0
	for i, name := range []string{"first", "second"} {
		if a.CutsceneName() != name {
			t.Fatal("wrong sequence member", a.CutsceneName())
		}
		advanceMovieTo(t, a, 2)
		if rec.stops != i || rec.starts != i+1 {
			t.Fatal("sequence cut audio before final presentation")
		}
		assertMoviePicture(t, a, color.RGBA{0, 0, 255, 255}, color.RGBA{255, 255, 255, 255})
		if rec.stops != i+1 || source.inputs[i].closes.Load() != 0 {
			t.Fatal("final presentation disposed movie or missed its audio stop")
		}
		waitFor(t, "sequence continuation", func() bool { a.HeadlessCutsceneStep(""); return a.CutsceneName() != name })
		if source.inputs[i].closes.Load() != 1 || rec.stops != i+1 {
			t.Fatal("sequence EOF double-stopped audio or failed stream disposal")
		}
	}
	if len(source.inputs) != 2 || a.Screen() != ScreenMenu || rec.starts != 2 || rec.stops != 2 {
		t.Fatal("sequence did not return once to its destination")
	}
	for _, backend := range backends {
		waitFor(t, "sequence backend close", func() bool { calls := backend.calls(); return len(calls) > 0 && calls[len(calls)-1] == "close" })
	}
}

func TestStoppedCutsceneQueueRejectsALateInFlightPush(t *testing.T) {
	d, backend := heldCutsceneDevice(audio.Settings{Master: 100}, 64)
	d.Start(audio.DeviceRate, 2)
	q := d.s.q
	d.Push([]byte{1, 2, 3, 4})
	d.Stop()
	q.push([]byte{5, 6, 7, 8})
	if n, err := q.Read(make([]byte, 16)); n != 0 || err != io.EOF {
		t.Fatal("detached queue admitted a late PCM push", n, err)
	}
	q.mu.Lock()
	retained := len(q.buf)
	q.mu.Unlock()
	if retained != 0 {
		t.Fatal("detached queue retained discarded PCM")
	}
	waitFor(t, "late-push backend close", func() bool { calls := backend.calls(); return len(calls) > 0 && calls[len(calls)-1] == "close" })
}
