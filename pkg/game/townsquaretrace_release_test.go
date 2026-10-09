package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// The town square trace is a behaviour record of the whole town front end on
// one install: a scripted pointer path over every square hotspot, each room
// entry and its return, under a synthetic clock and fixed draw sources. Each
// tick records the composed frame's hash, the sound and music requests the
// tick made and the room the screen stands in; each action records the hash
// of the SAV the campaign would write. The recorded lines live in
// testdata/townsquaretrace, one file per install text, and a run must
// reproduce them exactly. AGAINROM_TOWN_TRACE_WRITE=1 rewrites the file.

type traceSounds struct {
	log []string
}

type traceVoice struct {
	log     *traceSounds
	name    string
	playing bool
}

func (v *traceVoice) Playing() bool { return v.playing }
func (v *traceVoice) Stop() {
	if v.playing {
		v.log.log = append(v.log.log, "stop "+v.name)
	}
	v.playing = false
}

func (r *traceSounds) Play(audio.Sample, audio.Placement) { r.log = append(r.log, "play") }

func (r *traceSounds) RequestSample(_ audio.Sample, request audio.Request) audio.Voice {
	name := request.Source + ":" + request.Selector
	r.log = append(r.log, "start "+name)
	return &traceVoice{log: r, name: name, playing: true}
}

func (r *traceSounds) StartVoice(audio.Sample, audio.Placement) audio.Voice {
	r.log = append(r.log, "start voice")
	return &traceVoice{log: r, name: "voice", playing: true}
}

func (r *traceSounds) StartLoop(kind ui.AmbientLoop, _ audio.Sample, _ audio.Placement) {
	r.log = append(r.log, fmt.Sprintf("loop %d", kind))
}

func (r *traceSounds) RequestLoop(kind ui.AmbientLoop, _ audio.Sample, request audio.Request) audio.Voice {
	name := fmt.Sprintf("loop %d %s", kind, request.Selector)
	r.log = append(r.log, "start "+name)
	return &traceVoice{log: r, name: name, playing: true}
}

func (r *traceSounds) MoveLoop(ui.AmbientLoop, audio.Placement) {}
func (r *traceSounds) StopLoop(kind ui.AmbientLoop) {
	r.log = append(r.log, fmt.Sprintf("stoploop %d", kind))
}
func (r *traceSounds) Stop()                      {}
func (r *traceSounds) SetSettings(audio.Settings) {}

func (r *traceSounds) take() string {
	out := strings.Join(r.log, ",")
	r.log = nil
	return out
}

type traceMusicSource struct {
	inner ui.MusicSource
	log   *traceSounds
}

func (s traceMusicSource) Track(name string) (audio.Track, bool) {
	s.log.log = append(s.log.log, "music "+name)
	return s.inner.Track(name)
}

type traceMusicDevice struct{ log *traceSounds }

func (d traceMusicDevice) Start(audio.Track)          { d.log.log = append(d.log.log, "music-start") }
func (d traceMusicDevice) Stop()                      { d.log.log = append(d.log.log, "music-stop") }
func (d traceMusicDevice) Ended() bool                { return false }
func (d traceMusicDevice) SetSettings(audio.Settings) {}

// traceDraw is a fixed 15-bit linear congruential source reduced to [0,n).
func traceDraw(seed uint32) func(n int) int {
	state := seed
	return func(n int) int {
		state = state*1103515245 + 12345
		if n <= 0 {
			return 0
		}
		return int((state>>16)&0x7fff) % n
	}
}

type townTrace struct {
	t     *testing.T
	f     *FrontEnd
	a     *ui.App
	now   *time.Time
	sound *traceSounds
	lines []string
	tick  int
}

func (tr *townTrace) room() string {
	s := tr.f.TownScreen()
	where := []string{fmt.Sprintf("screen%d", tr.a.Screen())}
	if q, ok := s.(interface{ AtTownSquare() bool }); ok && q.AtTownSquare() {
		where = append(where, "square")
	}
	if q, ok := s.(interface{ AtTownShop() bool }); ok && q.AtTownShop() {
		where = append(where, "shop")
	}
	if q, ok := s.(interface{ AtTownSurface() bool }); ok && q.AtTownSurface() {
		where = append(where, "surface")
	}
	if q, ok := s.(interface{ AtWorldMap() bool }); ok && q.AtWorldMap() {
		where = append(where, "map")
	}
	if q, ok := s.(interface{ TownDialogue() (*image.RGBA, bool) }); ok {
		if _, open := q.TownDialogue(); open {
			where = append(where, "dialogue")
		}
	}
	if m, ok := s.(ui.TownMusicScreen); ok {
		scene, mage := m.TownMusic()
		where = append(where, fmt.Sprintf("music%d/%t", scene, mage))
	}
	return strings.Join(where, "+")
}

func (tr *townTrace) frame() string {
	pix, _, err := tr.a.HeadlessFrame()
	if err != nil {
		tr.t.Fatalf("tick %d frame: %v", tr.tick, err)
	}
	sum := sha256.Sum256(pix.Pix)
	return fmt.Sprintf("%dx%d:%s", pix.Bounds().Dx(), pix.Bounds().Dy(), hex.EncodeToString(sum[:8]))
}

func (tr *townTrace) record(kind string) {
	tr.lines = append(tr.lines, fmt.Sprintf("%04d %s %s frame=%s msg=%q snd=[%s]",
		tr.tick, kind, tr.room(), tr.frame(), tr.a.HeadlessMessage(), tr.sound.take()))
	tr.tick++
}

// hover is one update with the pointer at p followed by one paint, dt after
// the previous tick on the synthetic clock.
func (tr *townTrace) hover(p image.Point, dt time.Duration, n int) {
	for i := 0; i < n; i++ {
		*tr.now = tr.now.Add(dt)
		if err := tr.a.HeadlessPointer("hover", p.X, p.Y); err != nil {
			tr.t.Fatal(err)
		}
		tr.record(fmt.Sprintf("hover%d,%d", p.X, p.Y))
	}
}

func (tr *townTrace) click(p image.Point) {
	*tr.now = tr.now.Add(17 * time.Millisecond)
	if err := tr.a.HeadlessPointer("press", p.X, p.Y); err != nil {
		tr.t.Fatal(err)
	}
	if err := tr.a.HeadlessPointer("release", p.X, p.Y); err != nil {
		tr.t.Fatal(err)
	}
	tr.action(fmt.Sprintf("click%d,%d", p.X, p.Y))
}

func (tr *townTrace) key(name string) {
	*tr.now = tr.now.Add(17 * time.Millisecond)
	if err := tr.a.HeadlessKey(name); err != nil {
		tr.t.Fatal(err)
	}
	tr.action("key-" + name)
}

// action records a tick and the hash of the SAV the campaign would write.
func (tr *townTrace) action(name string) {
	save := "none"
	snap, label, err := tr.f.Snapshot(false)
	if err == nil {
		var payload []byte
		payload, err = tr.f.ExportCurrentSave(snap, label)
		sum := sha256.Sum256(payload)
		save = hex.EncodeToString(sum[:8])
	}
	if err != nil {
		save = "error:" + err.Error()
	}
	tr.record(name + " save=" + save)
}

// expect fails the trace unless the screen stands where the script expects.
func (tr *townTrace) expect(where string) {
	tr.t.Helper()
	got := tr.room()
	for _, part := range strings.Split(where, "+") {
		if !strings.Contains("+"+got+"+", "+"+part+"+") {
			tr.t.Fatalf("tick %d stands at %s, want %s", tr.tick, got, where)
		}
	}
}

// closeDialogue presses Enter until the town dialogue is gone.
func (tr *townTrace) closeDialogue() {
	for i := 0; i < 40; i++ {
		s, ok := tr.f.TownScreen().(interface{ TownDialogue() (*image.RGBA, bool) })
		if !ok {
			return
		}
		if _, open := s.TownDialogue(); !open {
			return
		}
		tr.key("enter")
	}
	tr.t.Fatal("town dialogue did not close")
}

// traceMaskPoints reads the installed square mask with the shared 8-bit BMP
// decoder and answers, for each region byte, two pixels of that byte at a
// third and two thirds of its raster run, and one pixel off every region.
func traceMaskPoints(t *testing.T, f *FrontEnd) (map[byte][2]image.Point, image.Point) {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile("graphics/interface/town/townmask.bmp")
	if err != nil {
		t.Fatal(err)
	}
	mask, err := terrain.DecodeBMP8(raw)
	if err != nil {
		t.Fatal(err)
	}
	runs := map[byte][]image.Point{}
	off := image.Pt(-1, -1)
	for y := 0; y < mask.Bounds().Dy(); y++ {
		for x := 0; x < mask.Bounds().Dx(); x++ {
			c := mask.ColorIndexAt(x, y)
			switch c {
			case 0x80, 0x90, 0xa0, 0xb0, 0xc0:
				runs[c] = append(runs[c], image.Pt(x, y))
			case 0:
				if off.X < 0 && x > 20 && y > 20 {
					off = image.Pt(x, y)
				}
			}
		}
	}
	out := map[byte][2]image.Point{}
	for _, c := range []byte{0x80, 0x90, 0xa0, 0xb0, 0xc0} {
		r := runs[c]
		if len(r) < 3 {
			t.Fatalf("mask byte %#x has %d pixels", c, len(r))
		}
		out[c] = [2]image.Point{r[len(r)/3], r[2*len(r)/3]}
	}
	if off.X < 0 {
		t.Fatal("mask has no background pixel")
	}
	return out, off
}

// traceKey names the install by its town tip text, which differs between the
// English and Russian editions.
func traceKey(t *testing.T, f *FrontEnd) string {
	t.Helper()
	raw, err := f.Archives.Containers.ReadFile(TownTipPath)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:6])
}

func TestReleaseTownSquareTraceIsUnchanged(t *testing.T) {
	f := releaseFront(t)
	now := time.Unix(5000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = traceDraw(11)
	f.TownAmbientRandom = traceDraw(23)
	f.TavernRandom = traceDraw(37)
	f.ShopRandom = traceDraw(41)
	f.SchoolRandom = traceDraw(53)
	f.AmbientSeed = 7
	f.MusicSeed = 3
	f.PersistenceContext.tipsOff = false
	sounds := &traceSounds{}
	f.SoundPlayer, f.SpeechPlayer, f.AmbientPlayer = sounds, sounds, sounds
	f.SoundBank = OpenSounds(f.Archives.Root)
	f.Carried = f.NextParty()
	f.arriveInTown()
	points, off := traceMaskPoints(t, f)

	a := f.App("town-square-trace")
	a.Layout(640, 480)
	a.SetMusic(traceMusicSource{inner: f.MusicBank, log: sounds}, traceMusicDevice{log: sounds}, f.MusicSeed)
	a.SetSaveSeams(nil, func() []ui.SaveEntry { return []ui.SaveEntry{{Name: "town.sav", Label: "Town"}} },
		func(string) (ui.MapOpener, bool, error) { return nil, true, nil })
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessActivate("@first"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ui.ScreenTown {
		t.Fatalf("trace did not reach the town: %v", a.Screen())
	}
	tr := &townTrace{t: t, f: f, a: a, now: &now, sound: sounds}
	tr.action("enter-town")

	tr.hover(off, 34*time.Millisecond, 20)
	for _, c := range []byte{0x80, 0x90, 0xa0, 0xb0, 0xc0} {
		tr.hover(points[c][0], 34*time.Millisecond, 25)
		tr.hover(points[c][1], 50*time.Millisecond, 25)
		tr.hover(off, 68*time.Millisecond, 12)
	}
	tr.hover(off, 53*time.Millisecond, 300)

	// The gates with no gate mission open the guard's line over the square.
	if f.Town.gateMission() != -1 {
		t.Fatal("a fresh town already holds a gate mission")
	}
	tr.hover(points[0xa0][0], 34*time.Millisecond, 10)
	tr.click(points[0xa0][0])
	tr.expect("square+dialogue")
	tr.hover(points[0xa0][0], 34*time.Millisecond, 10)
	tr.closeDialogue()
	tr.hover(points[0xa0][0], 34*time.Millisecond, 20)
	// The statue opens the game menu.
	tr.hover(points[0xb0][0], 34*time.Millisecond, 5)
	tr.click(points[0xb0][0])
	if a.Screen() != ui.ScreenGameMenu {
		t.Fatalf("statue opened screen %v", a.Screen())
	}
	tr.hover(points[0xb0][0], 34*time.Millisecond, 5)
	tr.key("escape")
	tr.expect("square")
	tr.hover(off, 34*time.Millisecond, 10)
	// Every building in turn, each one returned from with Escape.
	for _, door := range []struct {
		code byte
		room string
	}{{0x80, "surface"}, {0x90, "shop"}, {0xc0, "surface"}} {
		tr.hover(points[door.code][0], 34*time.Millisecond, 3)
		tr.click(points[door.code][0])
		tr.expect(door.room)
		tr.closeDialogue()
		tr.hover(points[door.code][0], 41*time.Millisecond, 40)
		tr.key("escape")
		tr.closeDialogue()
		tr.expect("square")
		tr.hover(off, 41*time.Millisecond, 20)
	}
	// With a gate mission the gates open onto the world map.
	if f.Town.gateMission() == -1 {
		f.Town.announceMission(f.Town.currentMain())
		tr.action("announce")
	}
	tr.hover(points[0xa0][1], 34*time.Millisecond, 30)
	tr.click(points[0xa0][1])
	tr.expect("map")
	tr.hover(points[0xa0][1], 70*time.Millisecond, 20)
	tr.key("escape")
	tr.expect("square")
	tr.hover(off, 34*time.Millisecond, 40)
	tr.hover(points[0xa0][1], 34*time.Millisecond, 20)

	got := strings.Join(tr.lines, "\n") + "\n"
	path := filepath.Join("testdata", "townsquaretrace", traceKey(t, f)+".txt")
	if os.Getenv("AGAINROM_TOWN_TRACE_WRITE") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d trace lines to %s", len(tr.lines), path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no recorded trace for this install (%v); record it on unchanged code", err)
	}
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if string(want) == got {
		t.Logf("town square trace: %d ticks identical to %s", len(tr.lines), path)
		return
	}
	wl := strings.Split(string(want), "\n")
	gl := strings.Split(got, "\n")
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			t.Fatalf("trace differs at line %d:\nwant %s\ngot  %s", i+1, w, g)
		}
	}
}
