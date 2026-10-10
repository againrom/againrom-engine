package game

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

type townAmbientRolls struct {
	values []int
	bounds []int
}

func (r *townAmbientRolls) draw(n int) int {
	r.bounds = append(r.bounds, n)
	if len(r.values) == 0 {
		return 0
	}
	v := r.values[0]
	r.values = r.values[1:]
	return v
}

type townCrowdRecorder struct {
	starts []struct {
		kind   ui.AmbientLoop
		sample audio.Sample
		place  audio.Placement
	}
	stops []ui.AmbientLoop
	voice *townCrowdVoice
}

type townCrowdVoice struct{ playing bool }

func (v *townCrowdVoice) Playing() bool { return v.playing }
func (v *townCrowdVoice) Stop()         { v.playing = false }

func (r *townCrowdRecorder) RequestLoop(kind ui.AmbientLoop, sample audio.Sample, request audio.Request) audio.Voice {
	r.StartLoop(kind, sample, request.Placement)
	r.voice = &townCrowdVoice{playing: true}
	return r.voice
}

func (r *townCrowdRecorder) StartLoop(kind ui.AmbientLoop, sample audio.Sample, place audio.Placement) {
	r.starts = append(r.starts, struct {
		kind   ui.AmbientLoop
		sample audio.Sample
		place  audio.Placement
	}{kind, sample, place})
}
func (*townCrowdRecorder) MoveLoop(ui.AmbientLoop, audio.Placement) {}
func (r *townCrowdRecorder) StopLoop(kind ui.AmbientLoop) {
	r.stops = append(r.stops, kind)
	r.Stop()
}
func (r *townCrowdRecorder) Stop() {
	if r.voice != nil {
		r.voice.Stop()
	}
}
func (*townCrowdRecorder) SetSettings(audio.Settings) {}

func townAmbientFrames(n int, c color.RGBA) []image.Image {
	out := make([]image.Image, n)
	for i := range out {
		pic := image.NewRGBA(image.Rect(0, 0, 2, 2))
		shade := c
		shade.B ^= byte(i)
		for y := 0; y < 2; y++ {
			for x := 0; x < 2; x++ {
				pic.SetRGBA(x, y, shade)
			}
		}
		out[i] = pic
	}
	return out
}

func installTownAmbientTestArt(a *town.Art) {
	for family := 0; family < 9; family++ {
		a.Frames[fmt.Sprintf("birds/%d", family)] = townAmbientFrames(townBirdFrameCount(), color.RGBA{R: byte(20 + family), A: 255})
	}
	a.Frames["stars"] = townAmbientFrames(townStarFrameCount(), color.RGBA{G: 90, A: 255})
}

func townAmbientFixture(t *testing.T, crowdSample bool) (*FrontEnd, *ui.App, *townScreen, *time.Time, *townAmbientRolls, *exteriorRecorder, *townCrowdRecorder) {
	t.Helper()
	f := shellFrontEnd()
	f.TownSquareArt = resolved(exteriorTestArt(t), nil)
	installTownAmbientTestArt(f.TownSquareArt.Value())
	f.Town.announceMission(f.Town.currentMain())
	now := time.Unix(300, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(n int) int {
		if n != 100 {
			t.Fatalf("established exterior draw bound %d", n)
		}
		return 0
	}
	rolls := &townAmbientRolls{values: []int{2, 0, 999, 1, 2, 0}}
	f.TownAmbientRandom = rolls.draw
	birdWait := f.townProcess.WaitLatch("birds")
	birdWait.Ready, birdWait.Wait = true, time.Second
	voice := &exteriorRecorder{}
	loop := &townCrowdRecorder{}
	f.SoundPlayer, f.AmbientPlayer = voice, loop
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{}, cache: map[int]soundCacheEntry{}}
	for i, path := range []string{"town/birds1.wav", "town/birds2.wav", "town/stars.wav"} {
		f.SoundBank.named[path] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(101 + i)}}, ok: true}
	}
	if crowdSample {
		f.SoundBank.named["town/crowd.wav"] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{104}}, ok: true}
	}
	a, s := exteriorApp(t, f)
	return f, a, s, &now, rolls, voice, loop
}

func sampleStarts(r *exteriorRecorder, id int16) int {
	n := 0
	for _, sample := range r.samples {
		if len(sample.PCM) == 1 && sample.PCM[0] == id {
			n++
		}
	}
	return n
}

func TestTownAmbientBirdDelayPrefixProgressTerminalAndSound(t *testing.T) {
	f, app, screen, now, rolls, voices, _ := townAmbientFixture(t, true)
	exteriorPaint(t, app, now, time.Second)
	if screen.sqBirds().Active {
		t.Fatal("strict one-second delay admitted equality")
	}
	exteriorPaint(t, app, now, time.Millisecond)
	if !screen.sqBirds().Active || screen.sqBirds().Group != 2 || screen.sqBirds().Count != 1 || screen.sqProgress() != "[0 0 0]" {
		t.Fatalf("first arm = group%d count%d progress%v", screen.sqBirds().Group, screen.sqBirds().Count, screen.sqBirds().Progress)
	}
	frame := screen.sqExteriorFrame()
	if !frame.BirdOverlayVisible || !frame.Birds[0].Visible || frame.Birds[0].Family != 6 || frame.Birds[1].Visible {
		t.Fatalf("prefix projection %+v", frame)
	}
	if sampleStarts(voices, 101) != 1 || sampleStarts(voices, 102) != 0 {
		t.Fatal("one-bird episode did not select Birds1 only")
	}
	exteriorPaint(t, app, now, 66*time.Millisecond)
	if screen.sqProgress() != "[0 0 0]" {
		t.Fatal("strict 67ms hub admitted equality")
	}
	exteriorPaint(t, app, now, time.Millisecond)
	if screen.sqProgress() != "[1 1 1]" {
		t.Fatalf("all physical progress words = %v", screen.sqBirds().Progress)
	}
	exteriorPaint(t, app, now, 10*time.Second)
	if screen.sqProgress() != "[2 2 2]" {
		t.Fatal("long paint gap caught up more than once")
	}
	for screen.sqBirds().Progress[0] < townBirdFrameCount() {
		exteriorPaint(t, app, now, 68*time.Millisecond)
	}
	terminal := screen.sqExteriorFrame()
	if !screen.sqBirds().Active || !screen.sqBirds().Terminal || !terminal.BirdOverlayVisible || terminal.Birds[0].Visible {
		t.Fatalf("terminal overlay-only snapshot %+v / %+v", screen.sqBirds(), terminal)
	}
	// The CPU capture inside exteriorPaint composes again at the same clock.
	// It must not consume the terminal snapshot App.Draw just produced.
	if !screen.sqExteriorFrame().BirdOverlayVisible {
		t.Fatal("same-clock reacquire consumed terminal overlay")
	}
	exteriorPaint(t, app, now, time.Millisecond)
	if screen.sqBirds().Active || screen.sqExteriorFrame().BirdOverlayVisible {
		t.Fatal("paint after terminal did not become inactive")
	}
	voices.voices[len(voices.voices)-1].playing = false
	exteriorPaint(t, app, now, f.townProcess.WaitLatch("birds").Wait+time.Millisecond)
	if !screen.sqBirds().Active || screen.sqBirds().Group != 1 || screen.sqBirds().Count != 3 {
		t.Fatalf("three-bird rearm group%d count%d", screen.sqBirds().Group, screen.sqBirds().Count)
	}
	for i, selected := range screen.sqExteriorFrame().Birds {
		if !selected.Visible || selected.Family != 3+i {
			t.Fatalf("selected bird %d = %+v", i, selected)
		}
	}
	if sampleStarts(voices, 102) != 1 {
		t.Fatal("two/three-bird episode did not select Birds2")
	}
	if want := []int{3, 3, 2000, 3, 3, 2000}; !bytes.Equal(intSliceBytes(rolls.bounds), intSliceBytes(want)) {
		t.Fatalf("ambient random bounds %v want %v", rolls.bounds, want)
	}
}

func TestTownAmbientRandomDoesNotShiftSignAndFlugerStream(t *testing.T) {
	f, app, _, now, _, _, _ := townAmbientFixture(t, true)
	var events []string
	f.TownAnimationRandom = func(n int) int {
		events = append(events, fmt.Sprintf("exterior:%d", n))
		return 0
	}
	f.TownAmbientRandom = func(n int) int {
		events = append(events, fmt.Sprintf("bird:%d", n))
		return 0
	}
	exteriorPaint(t, app, now, time.Second)
	if fmt.Sprint(events) != "[exterior:100 exterior:100]" {
		t.Fatalf("pre-arm exterior order %v", events)
	}
	events = nil
	exteriorPaint(t, app, now, time.Millisecond)
	if fmt.Sprint(events) != "[bird:3 bird:3 bird:2000]" {
		t.Fatalf("bird arm perturbed exterior stream: %v", events)
	}
	events = nil
	exteriorPaint(t, app, now, 66*time.Millisecond)
	exteriorPaint(t, app, now, time.Millisecond)
	if fmt.Sprint(events) != "[exterior:100 exterior:100]" {
		t.Fatalf("first admitted hub order %v", events)
	}
}

func intSliceBytes(v []int) []byte {
	out := make([]byte, 0, len(v)*4)
	for _, n := range v {
		out = append(out, byte(n), byte(n>>8), byte(n>>16), byte(n>>24))
	}
	return out
}

func TestTownAmbientStarS00ThroughHideTenTerminalAndStatueRoute(t *testing.T) {
	f, app, screen, now, _, voices, _ := townAmbientFixture(t, true)
	f.townProcess.WaitLatch("birds").Wait = 24 * time.Hour
	if got := screen.sqExteriorFrame().Star; !got.Visible || got.Frame != 0 {
		t.Fatalf("entry star = %+v", got)
	}
	p := image.Pt(50, 100)
	if control, ok := squareControlAt(f, p); !ok || control.Kind != ui.TownSquareControlMenu {
		t.Fatalf("statue control changed: %+v %v", control, ok)
	}
	exteriorPointer(t, app, p.X)
	for want := 1; want <= 8; want++ {
		exteriorPaint(t, app, now, 68*time.Millisecond)
		if got := screen.sqExteriorFrame().Star; !got.Visible || got.Frame != want {
			t.Fatalf("star step %d = %+v", want, got)
		}
	}
	exteriorPaint(t, app, now, 68*time.Millisecond)
	if got := screen.sqExteriorFrame().Star; got.Visible || screen.sqEpisode("star").Enabled || screen.squareView().EndCount("star") != 1 {
		t.Fatalf("first terminal = %+v active%v count%d", got, screen.sqEpisode("star").Enabled, screen.squareView().EndCount("star"))
	}
	for i := 0; i < 9; i++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		exteriorPaint(t, app, now, 68*time.Millisecond)
	}
	if screen.sqEpisode("star").Frame != 0 || screen.sqEpisode("star").Visible || screen.squareView().EndCount("star") != 0 {
		t.Fatalf("tenth terminal current%d visible%v count%d", screen.sqEpisode("star").Frame, screen.sqEpisode("star").Visible, screen.squareView().EndCount("star"))
	}
	// The recorder does not advance sample time. Report the installed request
	// as naturally complete before the later current-zero step retries it.
	if voice, ok := screen.sqVoice("star").(*exteriorVoice); ok {
		voice.playing = false
	}
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, app, now, 68*time.Millisecond)
	if got := screen.sqExteriorFrame().Star; !got.Visible || got.Frame != 1 {
		t.Fatalf("post-reset rearm = %+v", got)
	}
	if sampleStarts(voices, 103) != 2 {
		t.Fatalf("Stars request count %d, want only the two current-zero steps", sampleStarts(voices, 103))
	}
	if err := app.HeadlessPointer("release", p.X, p.Y); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatal("statue animation changed click-to-menu routing")
	}
}

func TestTownAmbientCrowdStartsOnceCleansAndRetriesWithoutVisualState(t *testing.T) {
	f, app, screen, now, _, _, crowd := townAmbientFixture(t, true)
	if len(crowd.starts) != 1 || crowd.starts[0].kind != ui.AmbientTownCrowd || crowd.starts[0].sample.PCM[0] != 104 {
		t.Fatalf("entry crowd starts = %+v", crowd.starts)
	}
	before := screen.sqExteriorFrame()
	for i := 0; i < 4; i++ {
		exteriorPaint(t, app, now, 68*time.Millisecond)
	}
	if len(crowd.starts) != 1 || before.Birds != screen.sqExteriorFrame().Birds || before.Star != screen.sqExteriorFrame().Star {
		t.Fatal("crowd restarted or created a visual frame owner")
	}
	if err := app.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if len(crowd.stops) != 1 || crowd.stops[0] != ui.AmbientTownCrowd {
		t.Fatal("focus loss did not clean crowd")
	}
	if err := app.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, app, now, time.Millisecond)
	if len(crowd.starts) != 2 {
		t.Fatal("focus resume did not restart crowd once")
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if len(crowd.stops) != 2 {
		t.Fatal("menu did not clean crowd")
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, app, now, time.Millisecond)
	if len(crowd.starts) != 3 {
		t.Fatal("menu resume did not restart crowd")
	}
	screen.resetForNewGame()
	if len(crowd.stops) != 3 || screen.squareView().Ready() || screen.sqBirds().Active || screen.sqEpisode("star").Frame != 0 {
		t.Fatal("new-game reset retained crowd/exterior state")
	}

	_, retryApp, retryScreen, retryNow, _, _, retry := townAmbientFixture(t, false)
	if len(retry.starts) != 0 {
		t.Fatal("missing crowd sample started a loop")
	}
	retryScreen.in.SoundBank.named["town/crowd.wav"] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{204}}, ok: true}
	exteriorPaint(t, retryApp, retryNow, time.Millisecond)
	if len(retry.starts) != 0 {
		t.Fatal("crowd performed a paint retry after entry")
	}
	if err := retryApp.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := retryApp.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	if len(retry.starts) != 1 || retry.starts[0].sample.PCM[0] != 204 {
		t.Fatal("crowd did not request on resumed entry")
	}
	_ = f
}

func TestTownAmbientOptionalSoundCapabilityAndSampleRetry(t *testing.T) {
	f, _, screen, _, _, _, _ := townAmbientFixture(t, false)
	plain := &legacySoundRecorder{}
	f.SoundPlayer = plain
	delete(f.SoundBank.named, "town/birds1.wav")
	host := townSquareHost{screen}
	if v := host.PlaySound("town-exterior", "town/birds1.wav"); len(plain.samples) != 0 || v != nil {
		t.Fatal("unretained player became an unlimited one-shot substitute")
	}
	recorder := &exteriorRecorder{}
	f.SoundPlayer = recorder
	host.PlaySound("town-exterior", "town/birds1.wav")
	if len(recorder.samples) != 0 {
		t.Fatal("missing sample started a voice")
	}
	f.SoundBank.named["town/birds1.wav"] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{301}}, ok: true}
	host.PlaySound("town-exterior", "town/birds1.wav")
	if sampleStarts(recorder, 301) != 1 {
		t.Fatal("newly available retained player/sample was not retried")
	}
}

func TestTownAmbientPresentationDoesNotChangeNativeOrSimulationState(t *testing.T) {
	f, app, _, now, _, _, _ := townAmbientFixture(t, true)
	mw, _ := readoutWorld(t, sim.Entity{ID: 1123, X: 8, Y: 9})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	before, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeBefore, err := EncodeSave(before, label)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 80; i++ {
		exteriorPointer(t, app, []int{0, 50}[i%2])
		exteriorPaint(t, app, now, 68*time.Millisecond)
	}
	after, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, label)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(nativeBefore, nativeAfter) {
		t.Fatal("town ambience changed native save bytes")
	}
	if mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("town ambience changed simulation form or hash")
	}
}
