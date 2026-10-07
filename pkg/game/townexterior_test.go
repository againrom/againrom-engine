package game

import (
	"bytes"
	"image"
	"image/color"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"github.com/hajimehoshi/ebiten/v2"
)

type exteriorVoice struct {
	playing bool
	stops   int
}

func (v *exteriorVoice) Playing() bool { return v.playing }
func (v *exteriorVoice) Stop()         { v.playing = false; v.stops++ }

type exteriorRecorder struct {
	samples []audio.Sample
	places  []audio.Placement
	voices  []*exteriorVoice
	ones    int
}

func (r *exteriorRecorder) Play(audio.Sample, audio.Placement) { r.ones++ }
func (r *exteriorRecorder) RequestSample(s audio.Sample, request audio.Request) audio.Voice {
	if request.Source == "school-training" || request.Source == "fixed-interface" {
		r.Play(s, request.Placement)
		return nil
	}
	return r.StartVoice(s, request.Placement)
}
func (r *exteriorRecorder) StartVoice(s audio.Sample, p audio.Placement) audio.Voice {
	v := &exteriorVoice{playing: true}
	r.samples = append(r.samples, s)
	r.places = append(r.places, p)
	r.voices = append(r.voices, v)
	return v
}

var exteriorSoundPaths = []string{"town/shop/enter.wav", "town/school/point.wav", "town/point.wav", "town/gateup.wav", "town/gatedn.wav", "town/guard1.wav", "town/guard2.wav", "town/flag.wav", "town/flugel.wav"}

func exteriorTestArt(t *testing.T) *ui.TownSquareArt {
	t.Helper()
	a, err := LoadTownSquareArt(townSquareSource())
	if err != nil {
		t.Fatal(err)
	}
	m := image.NewPaletted(image.Rect(0, 0, 640, 480), make(color.Palette, 256))
	for i, c := range []byte{0x80, 0x90, 0xc0, 0xa0, 0xb0} {
		m.SetColorIndex(10+i*10, 100, c)
	}
	a.Mask = m
	frames := func(n int) []image.Image {
		out := make([]image.Image, n)
		for i := range out {
			pic := image.NewRGBA(image.Rect(0, 0, 3, 3))
			for x := 0; x < 3; x++ {
				for y := 0; y < 3; y++ {
					pic.SetRGBA(x, y, color.RGBA{R: byte(i + 1), G: 80, A: 255})
				}
			}
			out[i] = pic
		}
		return out
	}
	a.Exterior = &ui.TownExteriorArt{Shop: frames(30), Tavern: frames(10), Fighter: frames(11), Mage: frames(11), Guard: frames(8), Door: frames(9), Sign: frames(10), Fluger: frames(8)}
	return a
}

func exteriorApp(t *testing.T, f *FrontEnd) (*ui.App, *townScreen) {
	t.Helper()
	a := f.App("1120-town-exterior")
	a.Layout(640, 480)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.ags", Label: "Town"}} }, func(string) (ui.MapOpener, bool, error) { f.townUI.resetForNewGame(); return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("load route: %v", a.Screen())
	}
	f.townUI.CloseTip()
	return a, f.townUI
}

func exteriorFixture(t *testing.T) (*FrontEnd, *ui.App, *townScreen, *time.Time, *int, *exteriorRecorder) {
	t.Helper()
	f := shellFrontEnd()
	f.TownSquareArt = resolved(exteriorTestArt(t), nil)
	f.Town.announceMission(f.Town.currentMain())
	now, roll := time.Unix(100, 0), 0
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = func(n int) int {
		if n != 100 {
			t.Fatal(n)
		}
		return roll
	}
	r := &exteriorRecorder{}
	f.SoundPlayer = r
	f.SoundBank = &SoundBank{named: map[string]soundCacheEntry{}, cache: map[int]soundCacheEntry{}}
	for i, p := range exteriorSoundPaths {
		f.SoundBank.named[p] = soundCacheEntry{sample: audio.Sample{Rate: audio.DeviceRate, PCM: []int16{int16(i + 1)}}, ok: true}
	}
	a, s := exteriorApp(t, f)
	return f, a, s, &now, &roll, r
}

func exteriorPointer(t *testing.T, a *ui.App, x int) {
	t.Helper()
	if err := a.HeadlessPointer("hover", x, 100); err != nil {
		t.Fatal(err)
	}
}

func exteriorAdvanceWithoutBlit(t *testing.T, a *ui.App, now *time.Time, dt time.Duration) {
	t.Helper()
	*now = now.Add(dt)
	if _, note, err := a.HeadlessFrame(); err != nil || note != "" {
		t.Fatalf("frame %q: %v", note, err)
	}
}

func exteriorPaint(t *testing.T, a *ui.App, now *time.Time, dt time.Duration) *image.RGBA {
	t.Helper()
	*now = now.Add(dt)
	// Real Draw dispatch advances the controller. CPU capture calls that same
	// composer at the unchanged clock and must not advance it a second time.
	a.Draw(ebiten.NewImage(640, 480))
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("frame %q: %v", note, err)
	}
	return pix
}

func TestTownExteriorAppDeliveredCyclesStrictPaintAndStationaryPointer(t *testing.T) {
	_, a, s, now, roll, _ := exteriorFixture(t)
	exteriorPointer(t, a, 10)
	if s.exterior.frame.Tavern != 0 || !s.exterior.tavern || s.exterior.selector != 2 {
		t.Fatal("delivery did not arm without stepping")
	}
	exteriorPaint(t, a, now, 67*time.Millisecond)
	if s.exterior.frame.Tavern != 0 {
		t.Fatal("67ms admitted hub")
	}
	pix := exteriorPaint(t, a, now, time.Millisecond)
	if s.exterior.frame.Tavern != 1 || pix.RGBAAt(124, 312).R != 2 {
		t.Fatal("68ms missing real Draw frame")
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if s.exterior.selector != 2 || s.exterior.frame.Tavern != 1 {
		t.Fatal("idle lost town pointer or Update stepped paint")
	}
	if err := a.HeadlessKey("up"); err != nil {
		t.Fatal(err)
	}
	if s.exterior.selector != 2 {
		t.Fatal("key lost pointer")
	}
	exteriorPointer(t, a, 0)
	for i := 2; i <= 10; i++ {
		exteriorPaint(t, a, now, time.Second)
		if s.exterior.frame.Tavern != i%10 {
			t.Fatalf("one step after long gap: %d", s.exterior.frame.Tavern)
		}
	}
	if s.exterior.tavern {
		t.Fatal("cycle did not clear")
	}
	exteriorPaint(t, a, now, time.Second)
	if s.exterior.frame.Tavern != 0 {
		t.Fatal("paint rearmed delivered-only tavern")
	}
	exteriorPointer(t, a, 10)
	for i := 0; i < 11; i++ {
		if err := a.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	if s.exterior.frame.Tavern != 1 {
		t.Fatal("stationary delivered updates did not rearm completed cycle")
	}
	*roll = 95
	exteriorPointer(t, a, 20)
	if s.exterior.shop {
		t.Fatal("shop armed at95")
	}
	*roll = 96
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	if !s.exterior.shop {
		t.Fatal("stationary shop update did not retry at96")
	}
	*roll = 0
	exteriorPointer(t, a, 0)
	for i := 1; i <= 30; i++ {
		exteriorPaint(t, a, now, 68*time.Millisecond)
		if s.exterior.frame.Shop != i%30 {
			t.Fatalf("shop cycle %d", i)
		}
	}
	if s.exterior.shop {
		t.Fatal("shop cycle retained bit")
	}
}

func TestTownExteriorAppSchoolSharedBitAndIndependentAmbience(t *testing.T) {
	_, a, s, now, roll, _ := exteriorFixture(t)
	*roll = 95
	exteriorPointer(t, a, 30)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Fighter != 0 || s.exterior.frame.Mage != 0 || s.exterior.frame.Sign != 1 || s.exterior.frame.Fluger != 0 {
		t.Fatalf("threshold boundary: %+v", s.exterior.frame)
	}
	*roll = 96
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Fighter != 1 || s.exterior.frame.Mage != 1 {
		t.Fatal("school start")
	}
	exteriorPointer(t, a, 0)
	*roll = 0
	for i := 2; i <= 10; i++ {
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Fighter != 10 || s.exterior.fighterStep != 0 {
		t.Fatal("leave reversed school or endpoint did not wait")
	}
	*roll = 96
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Fighter != 9 || s.exterior.fighterStep != -1 {
		t.Fatal("upper endpoint did not reverse")
	}
	*roll = 0
	for i := 8; i >= 0; i-- {
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.school {
		t.Fatal("school return did not clear shared bit")
	}
	// Independently transcribed TOWN-402 shared-bit boundary. The mutation
	// seeds a reachable helper boundary; all advancement remains App.Draw.
	s.exterior.frame.Fighter, s.exterior.fighterStep = 0, -1
	s.exterior.frame.Mage, s.exterior.mageStep = 5, 1
	exteriorPointer(t, a, 30)
	exteriorPointer(t, a, 0)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.school || s.exterior.frame.Mage != 6 {
		t.Fatal("fighter clear skipped mage in same hub")
	}
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Mage != 6 {
		t.Fatal("cleared bit did not pause both")
	}
	*roll = 98
	exteriorPaint(t, a, now, 68*time.Millisecond)
	*roll = 0
	if s.exterior.frame.Fluger != 1 || !s.exterior.fluger {
		t.Fatal("fluger not independently armed on blank")
	}
	for i := 0; i < 10; i++ {
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	if s.exterior.fluger || s.exterior.sign || s.exterior.frame.Fluger != 0 || s.exterior.frame.Sign != 0 {
		t.Fatal("ambient cycles did not end")
	}
	if s.exterior.frame.Mage != 6 {
		t.Fatal("ambient tick advanced unarmed school")
	}
}

func TestTownExteriorAppGateReverseGuardUnavailableAndSoundCancel(t *testing.T) {
	f, a, s, now, _, r := exteriorFixture(t)
	exteriorPointer(t, a, 20)
	shop := s.exterior.voices[exteriorShop].(*exteriorVoice)
	exteriorPointer(t, a, 30)
	if shop.stops != 1 {
		t.Fatal("school did not cancel shop")
	}
	school := s.exterior.voices[exteriorSchool].(*exteriorVoice)
	exteriorPointer(t, a, 10)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if school.stops != 1 {
		t.Fatal("tavern preincrement did not cancel school")
	}
	exteriorPointer(t, a, 40)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Door != 7 {
		t.Fatal("gate did not open")
	}
	up := s.exterior.voices[exteriorGate].(*exteriorVoice)
	exteriorPointer(t, a, 0)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Door != 8 || up.stops != 1 {
		t.Fatal("gate did not reverse/cancel")
	}
	exteriorPointer(t, a, 40)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Door != 7 {
		t.Fatal("gate reentry reset progress")
	}
	for i := 0; i < 8; i++ {
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	if s.exterior.frame.Door != 0 || s.exterior.frame.Guard != 7 || s.exterior.guardStep != 0 || s.exterior.voices[exteriorGuard] != nil {
		t.Fatal("endpoints/overshoot release")
	}
	clearTownTestGateLatches(f.Town)
	exteriorPointer(t, a, 40)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Door != 8 || !s.exterior.gateLatch || s.exterior.frame.Guard != 6 {
		t.Fatal("unavailable gate/guard branch")
	}
	for i := 0; i < 7; i++ {
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	if s.exterior.frame.Guard != 0 || s.exterior.guardStep != 0 || s.exterior.voices[exteriorGuard] != nil {
		t.Fatal("negative overshoot")
	}
	// All retained channel starts use centered placement; no one-shot fallback.
	for _, p := range r.places {
		if p != (audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}) {
			t.Fatal(p)
		}
	}
	if r.ones != 0 {
		t.Fatal("exterior used unrestricted one-shots")
	}
}

func TestTownExteriorAppMenuFocusRoomReentryAndLoadReset(t *testing.T) {
	_, a, s, now, _, _ := exteriorFixture(t)
	exteriorPointer(t, a, 10)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if s.exterior.active || s.exterior.selector != -1 {
		t.Fatal("menu kept stale hover active")
	}
	*now = now.Add(time.Hour)
	a.Draw(ebiten.NewImage(640, 480))
	if s.exterior.frame.Tavern != 1 {
		t.Fatal("menu paint advanced square")
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, a, now, 0)
	if s.exterior.frame.Tavern != 2 {
		t.Fatal("resume did not admit one due process-timer hub")
	}
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, a, now, time.Hour)
	if s.exterior.active || s.exterior.frame.Tavern != 2 {
		t.Fatal("focus pause failed")
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if s.exterior.frame.Tavern != 3 || s.exterior.selector != 2 {
		t.Fatal("focus resume lost pointer/cycle")
	}
	for _, x := range []int{10, 20, 30, 40} {
		if err := a.HeadlessPointer("release", x, 100); err != nil {
			t.Fatal(err)
		}
		if s.room == roomSquare {
			t.Fatalf("door at%d did not enter", x)
		}
		for i := 0; s.room != roomSquare && i < 64; i++ {
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
		}
		if s.room != roomSquare || s.exterior.frame.Tavern != 0 || s.exterior.frame.Door != 8 {
			t.Fatal("Back/reentry did not reset")
		}
	}
	exteriorPointer(t, a, 10)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if err := a.HeadlessKey("f3"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown || s.exterior.frame.Tavern != 0 || s.exterior.frame.Door != 8 || s.exterior.tavern {
		t.Fatal("App load retained old entrance cycle")
	}
}

func TestTownExteriorPresentationLeavesNativeBytesIdentical(t *testing.T) {
	f, a, s, now, roll, _ := exteriorFixture(t)
	mw, _ := readoutWorld(t, sim.Entity{ID: 1, X: 4, Y: 5})
	f.live = mw
	worldBytes, worldHash := marshalWorld(t, mw.world), mw.world.Hash()
	before, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeSave(before, "same")
	if err != nil {
		t.Fatal(err)
	}
	*roll = 99
	for i := 0; i < 80; i++ {
		exteriorPointer(t, a, []int{10, 20, 30, 40, 0}[i%5])
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	after, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := EncodeSave(after, "same")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, b2) {
		t.Fatal("presentation changed native save bytes")
	}
	if mw.world.Hash() != worldHash || !bytes.Equal(worldBytes, marshalWorld(t, mw.world)) {
		t.Fatal("presentation changed sim hash or binary state")
	}
	s.resetForNewGame()
	if !reflect.DeepEqual(s.exterior, townExteriorAnimation{}) {
		t.Fatal("new game retained exterior")
	}
}

func TestTownExteriorMissingMotionAndSoundCapability(t *testing.T) {
	f, a, s, now, _, _ := exteriorFixture(t)
	f.TownSquareArt.Value().Exterior = &ui.TownExteriorArt{}
	f.SoundPlayer = &legacySoundRecorder{}
	for _, x := range []int{10, 20, 30, 40, 0} {
		exteriorPointer(t, a, x)
		exteriorPaint(t, a, now, 68*time.Millisecond)
	}
	if len(f.SoundPlayer.(*legacySoundRecorder).samples) != 0 {
		t.Fatal("one-shot substitute")
	}
	f.TownSquareArt = lazy[*ui.TownSquareArt]{}
	exteriorPointer(t, a, 10)
	a.Draw(ebiten.NewImage(640, 480))
	if s.exterior.selector != -1 {
		t.Fatal("missing mask retained selector")
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if s.room == roomSquare {
		t.Fatal("missing art lost row fallback")
	}
}

func TestTownExteriorSoundStatusRetriesAndGuardSameHubRelease(t *testing.T) {
	_, a, s, now, _, r := exteriorFixture(t)
	count := func(id int16) int {
		n := 0
		for _, sample := range r.samples {
			if len(sample.PCM) == 1 && sample.PCM[0] == id {
				n++
			}
		}
		return n
	}
	exteriorPointer(t, a, 20)
	voice := s.exterior.voices[exteriorShop].(*exteriorVoice)
	exteriorPointer(t, a, 0)
	exteriorPointer(t, a, 20)
	if count(1) != 1 || voice.stops != 0 {
		t.Fatal("playing shop voice restarted after leave")
	}
	voice.playing = false // backend reports natural completion, not guessed time
	exteriorPointer(t, a, 0)
	exteriorPointer(t, a, 20)
	if count(1) != 2 || voice.stops != 1 {
		t.Fatal("finished voice did not release/retry")
	}
	voice = s.exterior.voices[exteriorShop].(*exteriorVoice)
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	if voice.stops != 1 {
		t.Fatal("focus did not release retained voice")
	}
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	s.exterior.frame.Guard = 7
	s.exterior.guardLatch = false
	exteriorPointer(t, a, 0)
	exteriorPaint(t, a, now, 68*time.Millisecond)
	if count(7) != 1 || r.voices[len(r.voices)-1].stops != 1 || s.exterior.voices[exteriorGuard] != nil {
		t.Fatal("terminal guard request was not released in its same hub")
	}
}
