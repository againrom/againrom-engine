package game

import (
	"bytes"
	"image"
	"image/color"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/town"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

// The tavern scene as the ROM1 description states it: the draw range, the
// state order of the tender, the frame counts and the sound slots.
const (
	tavernRandomRange    = 32768
	tavernDrink          = 0
	tavernBreath         = 1
	tavernCandleLoop     = 9
	tavernCauldronLoop   = 20
	tavernCauldronLoaded = 21
	tavernBreathLoaded   = 24
	tavernDrinkLoaded    = 40
)

var tavernSounds = []string{"drink", "glotok", "steam", "water", "chair", "breath", "enter"}

func tavernSoundKey(slot string) string {
	for _, r := range rom1Town.Rooms {
		if r.Scene == nil {
			continue
		}
		for _, s := range r.Scene.Sounds.Slots {
			if r.Name == "tavern" && s.Name == slot {
				return s.Key
			}
		}
	}
	return ""
}

func tavernTenderState(name string) int {
	if name == "drink" {
		return tavernDrink + 1
	}
	return tavernBreath + 1
}

func tavernLoop(s *townScreen, name string) *town.Loop {
	return s.tavernPage().Actor(name).(*town.Loop)
}

func tavernTender(s *townScreen) *town.Alternating {
	return s.tavernPage().Actor("tender").(*town.Alternating)
}

func TestTavernTenderReplacesStaticBodyThroughBlackShadows(t *testing.T) {
	src := townTavernSource()
	for _, path := range []string{"tender/breath/br0001.bmp", "tender/drink/dr0001.bmp", "candle/t0000.bmp"} {
		src[townTavernArtPrefix+path] = synthBMP(32, 12, color.RGBA{A: 255})
	}
	art, err := LoadTownTavernArt(src)
	if err != nil {
		t.Fatal(err)
	}
	for _, tender := range []string{"breath", "drink"} {
		s := bareTownScreen(t)
		s.draws = fakeTownDraws{}
		s.in.TownTavernArt = resolved(art, nil)
		p := s.tavernPage()
		p.Enter()
		tavernTender(s).ShownState = tavernTenderState(tender)
		view := ui.TownSurfaceView{Kind: ui.TownSurfaceTavern, TavernArt: art, Scene: roomScene{p}}
		frame := ui.ComposeTownSurface(view)
		if got := frame.RGBAAt(250, 155); got != (color.RGBA{A: 255}) {
			t.Fatalf("static tender shows through the replacement shadow: %v", got)
		}
		if got := frame.RGBAAt(180, 50); got != (color.RGBA{B: 0x21, A: 255}) {
			t.Fatalf("candle key erased the room background: %v", got)
		}
	}
}

type tavernInteriorVoice struct {
	playing bool
	stops   int
}

func (v *tavernInteriorVoice) Playing() bool { return v.playing }
func (v *tavernInteriorVoice) Stop()         { v.playing = false; v.stops++ }

type tavernInteriorRecorder struct {
	samples []audio.Sample
	voices  []*tavernInteriorVoice
	ones    int
}

func (r *tavernInteriorRecorder) Play(audio.Sample, audio.Placement) { r.ones++ }
func (r *tavernInteriorRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	return r.StartVoice(s, request.Placement)
}
func (r *tavernInteriorRecorder) StartVoice(s audio.Sample, _ audio.Placement) audio.Voice {
	v := &tavernInteriorVoice{playing: true}
	r.samples = append(r.samples, s)
	r.voices = append(r.voices, v)
	return v
}

func (r *tavernInteriorRecorder) count(name string) int {
	slot := 0
	for i, n := range tavernSounds {
		if n == name {
			slot = i
		}
	}
	want, n := int16(slot+1), 0
	for _, sample := range r.samples {
		if len(sample.PCM) == 1 && sample.PCM[0] == want {
			n++
		}
	}
	return n
}

func tavernInteriorApp(t *testing.T, f *FrontEnd) *ui.App {
	t.Helper()
	a := f.App("1121-tavern-interior")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) {
		f.townUI.resetForNewGame()
		return nil, true, nil
	})
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("load route = %v", a.Screen())
	}
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if f.townUI.room != roomTavern {
		t.Fatalf("door route left room %v", f.townUI.room)
	}
	if err := a.HeadlessStep(); err != nil { // installs the focus/room lifecycle
		t.Fatal(err)
	}
	f.townUI.CloseTip()
	return a
}

func tavernInteriorFixture(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *time.Time, *int, *tavernInteriorRecorder) {
	t.Helper()
	f := shellFrontEnd()
	art, err := LoadTownTavernArt(townTavernSource())
	if err != nil {
		t.Fatal(err)
	}
	f.TownTavernArt = resolved(art, nil)
	f.Shop = NewShop(0)
	now, roll := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TavernRandom = func(n int) int {
		if n != tavernRandomRange {
			t.Fatalf("random range = %d", n)
		}
		return roll
	}
	r := &tavernInteriorRecorder{}
	f.SoundPlayer = r
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{}, cache: map[int]soundCacheEntry{}}
	for slot, name := range tavernSounds {
		f.SoundBank.named[tavernSoundKey(name)] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(slot + 1)}}, ok: true}
	}
	a := tavernInteriorApp(t, f)
	return f, a, f.townUI, &now, &roll, r
}

// drawTavernApp reads the actual target passed through App.Draw. It does not
// call HeadlessFrame, which would compose a second paint at the same clock.
func drawTavernApp(t *testing.T, a *ui.App, s *townScreen, now *time.Time, elapsed time.Duration) *image.RGBA {
	t.Helper()
	*now = now.Add(elapsed)
	a.Draw(ebiten.NewImage(640, 480))
	// App.Draw has published the exact draw-before-step snapshot into the
	// screen. Compose that read-only view on the CPU; asking ebiten to read the
	// target is forbidden before RunGame starts in unit tests.
	return ui.ComposeTownSurface(s.TownSurface())
}

func TestTavernInteriorActualAppStrictCyclesAndKeeperCaches(t *testing.T) {
	_, a, s, now, _, _ := tavernInteriorFixture(t)

	frame := drawTavernApp(t, a, s, now, 100*time.Millisecond)
	if tavernLoop(s, "candle").Index != 0 || frame.RGBAAt(180, 49).G != 1 {
		t.Fatal("strict >100ms gate admitted equality")
	}
	frame = drawTavernApp(t, a, s, now, time.Millisecond)
	if tavernLoop(s, "candle").Index != 1 || tavernLoop(s, "cauldron").Index != 1 || frame.RGBAAt(180, 49).G != 1 {
		t.Fatal("101ms paint did not draw current then single-step both families")
	}
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(180, 49).G != 2 || frame.RGBAAt(440, 161).G != 2 {
		t.Fatal("post-step cached pictures were not visible on the next paint")
	}
	for i := 0; i < 80; i++ {
		drawTavernApp(t, a, s, now, 101*time.Millisecond)
		if tavernLoop(s, "candle").Index < 0 || tavernLoop(s, "candle").Index >= tavernCandleLoop ||
			tavernLoop(s, "cauldron").Index < 0 || tavernLoop(s, "cauldron").Index >= tavernCauldronLoop {
			t.Fatalf("loaded terminal entry selected: candle %d cauldron %d", tavernLoop(s, "candle").Index, tavernLoop(s, "cauldron").Index)
		}
	}

	// Breath completion resets its logical index and deliberately leaves the
	// cached last picture. Direction is a different field and stays untouched.
	si := tavernTender(s)
	si.State, si.Direction = 2, -1
	si.Index[tavernBreath], si.Cached[tavernBreath] = tavernBreathLoaded-1, tavernBreathLoaded-1
	si.Delay = time.Hour
	s.tavernPage().SetClock("tender", now.Add(-84*time.Millisecond))
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(260, 153).G != tavernBreathLoaded || si.State != 0 || si.Index[tavernBreath] != 0 || si.Cached[tavernBreath] != tavernBreathLoaded-1 || si.Direction != -1 {
		t.Fatalf("breath completion = mode%d index/cache%d/%d direction%d", si.State, si.Index[tavernBreath], si.Cached[tavernBreath], si.Direction)
	}
	si.State = 2
	s.tavernPage().SetClock("tender", *now)
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(260, 153).G != tavernBreathLoaded {
		t.Fatal("next breath did not first draw stale cached br0024")
	}
	drawTavernApp(t, a, s, now, 84*time.Millisecond)
	frame = drawTavernApp(t, a, s, now, 0)
	if si.Index[tavernBreath] != 1 || si.Cached[tavernBreath] != 1 || frame.RGBAAt(260, 153).G != 2 {
		t.Fatal("stale breath cache did not advance to br0002")
	}

	// Drink's top paint is followed by index38, never a second index39.
	si.State, si.Direction = 1, 1
	si.Index[tavernDrink], si.Cached[tavernDrink] = tavernDrinkLoaded-1, tavernDrinkLoaded-1
	si.Delay = time.Hour
	s.tavernPage().SetClock("tender", now.Add(-84*time.Millisecond))
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(260, 153).G != tavernDrinkLoaded || si.Direction != -1 || si.Index[tavernDrink] != tavernDrinkLoaded-2 {
		t.Fatalf("drink turnaround = picture%d direction%d index%d", frame.RGBAAt(260, 153).G, si.Direction, si.Index[tavernDrink])
	}
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(260, 153).G != tavernDrinkLoaded-1 {
		t.Fatal("drink top duplicated instead of drawing index38")
	}

	// A long gap can rearm a descending drink to ascent without resetting its
	// independent index; the same admitted paint then takes one forward step.
	si.State, si.Direction = 1, -1
	si.Index[tavernDrink], si.Cached[tavernDrink] = 20, 20
	si.Delay = 3001 * time.Millisecond
	s.tavernPage().SetClock("tender", now.Add(-3002*time.Millisecond))
	frame = drawTavernApp(t, a, s, now, 0)
	if frame.RGBAAt(260, 153).G != 21 || si.State != 1 || si.Direction != 1 || si.Index[tavernDrink] != 21 {
		t.Fatalf("active rearm = pixel%d mode%d direction%d index%d", frame.RGBAAt(260, 153).G, si.State, si.Direction, si.Index[tavernDrink])
	}
}

func TestTavernInteriorDelayEndpointsParityAndArmOrder(t *testing.T) {
	_, a, s, now, roll, _ := tavernInteriorFixture(t)
	if got := tavernTender(s).Delay; got != 3000*time.Millisecond {
		t.Fatalf("minimum delay = %v", got)
	}
	// Equality does not arm. The next millisecond arms breath (even delay),
	// publishes br0001, then admits one >83ms forward step.
	drawTavernApp(t, a, s, now, 3000*time.Millisecond)
	if tavernTender(s).State != 0 {
		t.Fatal("delay equality armed")
	}
	frame := drawTavernApp(t, a, s, now, time.Millisecond)
	if tavernTender(s).State != 2 || tavernTender(s).Index[tavernBreath] != 1 || frame.RGBAAt(260, 153).G != 1 {
		t.Fatalf("even arm = mode%d index%d pixel%d", tavernTender(s).State, tavernTender(s).Index[tavernBreath], frame.RGBAAt(260, 153).G)
	}

	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	*roll = tavernRandomRange - 1
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if got := tavernTender(s).Delay; got != 5047*time.Millisecond {
		t.Fatalf("maximum delay = %v", got)
	}
	frame = drawTavernApp(t, a, s, now, 5048*time.Millisecond)
	if tavernTender(s).State != 1 || tavernTender(s).Direction != 1 || tavernTender(s).Index[tavernDrink] != 1 || frame.RGBAAt(260, 153).G != 1 {
		t.Fatalf("odd arm = mode%d direction%d index%d pixel%d", tavernTender(s).State, tavernTender(s).Direction, tavernTender(s).Index[tavernDrink], frame.RGBAAt(260, 153).G)
	}
}

func TestTavernInteriorConditionalSoundStatusAndInterruption(t *testing.T) {
	_, a, s, now, _, r := tavernInteriorFixture(t)
	if r.count("enter") != 1 || r.ones != 0 {
		t.Fatalf("entry requests = %d retained, %d one-shot", r.count("enter"), r.ones)
	}
	drawTavernApp(t, a, s, now, 0)
	drawTavernApp(t, a, s, now, 0)
	if r.count("water") != 1 {
		t.Fatal("repeat water ignored retained playing status")
	}
	s.tavernPage().Voice("water").(*tavernInteriorVoice).playing = false
	drawTavernApp(t, a, s, now, 0)
	if r.count("water") != 2 {
		t.Fatal("completed repeat water did not retry")
	}
	drawTavernApp(t, a, s, now, 10*time.Second)
	if r.count("steam") != 0 {
		t.Fatal("steam admitted equality")
	}
	drawTavernApp(t, a, s, now, time.Millisecond)
	if r.count("steam") != 1 {
		t.Fatal("steam missing above strict threshold")
	}
	if r.count("chair") != 1 || r.count("breath") != 1 {
		t.Fatalf("breath arm requests = chair%d breath%d", r.count("chair"), r.count("breath"))
	}

	si := tavernTender(s)
	si.State, si.Direction = 1, -1
	si.Index[tavernDrink], si.Cached[tavernDrink] = 0, 0
	si.Delay = time.Hour
	s.tavernPage().SetClock("tender", now.Add(-84*time.Millisecond))
	drawTavernApp(t, a, s, now, 0)
	if r.count("glotok") != 1 || si.State != 0 || si.Direction != 0 {
		t.Fatal("reverse completion did not request glotok and clear mode/direction")
	}
	si.State, si.Direction = 1, 1
	si.Index[tavernDrink], si.Cached[tavernDrink] = 30, 30
	si.Delay = time.Hour
	s.tavernPage().SetClock("tender", *now)
	drawTavernApp(t, a, s, now, 0)
	drawTavernApp(t, a, s, now, 0)
	if r.count("drink") != 1 {
		t.Fatal("index30 request ignored status")
	}
	s.tavernPage().Voice("drink").(*tavernInteriorVoice).playing = false
	drawTavernApp(t, a, s, now, 0)
	if r.count("drink") != 2 {
		t.Fatal("index30 request did not retry after status zero")
	}

	// Focus pause stops every retained voice and rebases every clock. Time
	// spent outside focus cannot complete or catch up an episode.
	before := si.Index[tavernDrink]
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if s.tavernPage().Active() {
		t.Fatal("focus loss left controller active")
	}
	for _, slot := range tavernSounds {
		if s.tavernPage().Voice(slot) != nil {
			t.Fatalf("focus loss retained voice %s", slot)
		}
	}
	drawTavernApp(t, a, s, now, time.Hour)
	if si.Index[tavernDrink] != before {
		t.Fatal("unfocused paint advanced")
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	drawTavernApp(t, a, s, now, 83*time.Millisecond)
	if si.Index[tavernDrink] != before {
		t.Fatal("resume admitted 83ms equality")
	}
	drawTavernApp(t, a, s, now, time.Millisecond)
	if si.Index[tavernDrink] == before {
		t.Fatal("resume did not advance after rebased strict gate")
	}

	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomSquare || s.tavernPage().Active() {
		t.Fatal("Back did not interrupt tavern")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("town exit = %v, want game menu", a.Screen())
	}
}

func TestTavernInteriorInteractionsReentryLoadAndMissingFamilies(t *testing.T) {
	f, a, s, now, _, r := tavernInteriorFixture(t)
	drawTavernApp(t, a, s, now, 101*time.Millisecond)
	phase := tavernLoop(s, "candle").Index
	if err := a.HeadlessActivate("Mercenary 3"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if s.tavernSelection.kind != tavernCandidateMercenary || tavernLoop(s, "candle").Index != phase {
		t.Fatal("selected miniature update reset or update-stepped interior")
	}
	if err := a.HeadlessActivate(f.Words.TavernHire); err != nil {
		t.Fatal(err)
	}
	if !f.Town.MercenaryHired(3) || tavernLoop(s, "candle").Index != phase {
		t.Fatal("Hire changed interior controller")
	}
	if err := a.HeadlessActivate(f.Words.TavernFire); err != nil {
		t.Fatal(err)
	}
	if f.Town.MercenaryHired(3) || tavernLoop(s, "candle").Index != phase {
		t.Fatal("Fire changed interior controller")
	}
	if err := a.HeadlessActivate(f.Words.TavernTalk); err != nil {
		t.Fatal(err)
	}
	if s.room != roomTalk || !s.inTavernInterior() {
		t.Fatal("Talk did not retain tavern background")
	}
	drawTavernApp(t, a, s, now, 101*time.Millisecond)
	if tavernLoop(s, "candle").Index == phase {
		t.Fatal("Talk background stopped painting")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.room != roomTavern || !s.tavernPage().Ready() {
		t.Fatal("conversation Back reset controller")
	}
	if err := a.HeadlessActivate("Sleep"); err != nil {
		t.Fatal(err)
	}
	retained := tavernLoop(s, "candle").Index
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if retained == 0 || tavernLoop(s, "candle").Index != 0 || tavernTender(s).State != 0 || tavernTender(s).Direction != 0 {
		t.Fatalf("cross-visit reset = retained%d tender%+v", retained, *tavernTender(s))
	}
	if r.count("enter") != 2 {
		t.Fatal("reentry did not request enter independently")
	}

	// Actual App LOAD installs a different game through resetForNewGame.
	drawTavernApp(t, a, s, now, 101*time.Millisecond)
	if err := a.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown || s.room != roomSquare || s.tavernPage().Ready() {
		t.Fatal("native App load retained tavern presentation")
	}

	// One malformed family cannot remove its siblings or the room route.
	src := townTavernSource()
	delete(src, townTavernArtPrefix+"candle/t0005.bmp")
	degraded, err := LoadTownTavernArt(src)
	if err == nil || degraded == nil || len(degraded.Scene["candle"]) != 0 || len(degraded.Scene["cauldron"]) != tavernCauldronLoaded {
		t.Fatalf("degraded loader = %#v / %v", degraded, err)
	}
	f.TownTavernArt = resolved(degraded, nil)
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	drawTavernApp(t, a, s, now, 101*time.Millisecond)
	if tavernLoop(s, "candle").Index != 0 || tavernLoop(s, "cauldron").Index != 1 {
		t.Fatal("missing candle stopped cauldron or invented a partial cycle")
	}

	// A Player without VoicePlayer and a missing named sample are both local
	// silence. Neither falls back to unlimited one-shot Play calls.
	plain := &legacySoundRecorder{}
	f.SoundPlayer = plain
	delete(f.SoundBank.named, tavernSoundKey("steam"))
	drawTavernApp(t, a, s, now, 11*time.Second)
	if len(plain.samples) != 0 {
		t.Fatal("missing retained capability used one-shot substitute")
	}
}

func TestTavernInteriorMissingSelectedFamilyAndSoundStayLocal(t *testing.T) {
	f, a, s, now, roll, recorder := tavernInteriorFixture(t)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}

	// An odd entry selects Drink. Remove one installed member so the loader
	// discards that family whole; the failed arm chooses a fresh delay and the
	// following even delay can still reach the complete Breath sibling.
	src := townTavernSource()
	delete(src, townTavernArtPrefix+"tender/drink/dr0020.bmp")
	degraded, err := LoadTownTavernArt(src)
	if err == nil || degraded == nil || len(degraded.Scene["drink"]) != 0 || len(degraded.Scene["breath"]) != tavernBreathLoaded {
		t.Fatalf("drink-local degradation = %#v / %v", degraded, err)
	}
	f.TownTavernArt = resolved(degraded, nil)
	*roll = 16 // 3001ms, odd Drink arm.
	if err := a.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	*roll = 0 // fallback delay is 3000ms, even Breath arm.
	drawTavernApp(t, a, s, now, 3002*time.Millisecond)
	if tavernTender(s).State != 0 || tavernTender(s).Delay != 3000*time.Millisecond {
		t.Fatalf("missing drink trapped mode/delay: %+v", *tavernTender(s))
	}
	frame := drawTavernApp(t, a, s, now, 3001*time.Millisecond)
	if tavernTender(s).State != 2 || frame.RGBAAt(260, 153).G != 1 {
		t.Fatal("missing drink prevented complete breath sibling")
	}

	// A missing installed sound leaf remains local even with retained-voice
	// capability: Steam stays silent while Water and the room keep working.
	delete(f.SoundBank.named, tavernSoundKey("steam"))
	drawTavernApp(t, a, s, now, 11*time.Second)
	if recorder.count("steam") != 0 || recorder.count("water") == 0 || s.room != roomTavern {
		t.Fatalf("sound-local degradation = steam%d water%d room%d",
			recorder.count("steam"), recorder.count("water"), s.room)
	}
}

func TestTavernInteriorPresentationLeavesNativeBytesAndHashIdentical(t *testing.T) {
	f, a, s, now, roll, _ := tavernInteriorFixture(t)
	mw, _ := readoutWorld(t, sim.Entity{ID: 1, X: 4, Y: 5})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	native, err := EncodeSave(before, "same")
	if err != nil {
		t.Fatal(err)
	}
	*roll = 17
	for i := 0; i < 120; i++ {
		drawTavernApp(t, a, s, now, 101*time.Millisecond)
	}
	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	nativeAfter, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(native, nativeAfter) {
		t.Fatal("presentation changed native save bytes")
	}
	if mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("presentation changed sim hash or binary state")
	}
	s.resetForNewGame()
	if s.tavernPage().Ready() {
		t.Fatal("new game retained tavern controller")
	}
}
