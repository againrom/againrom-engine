package game

import (
	"fmt"
	"image"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/formats/bmp"
	"againrom/pkg/ui"
)

// chrgenReleaseMembers are the sixteen chrgen members of sfx.res that
// VIDEO-SFX-058..060 name, spelled literally: levels, hero, OK, statistic,
// then the fighter and mage skill members in stored skill slot order.
var chrgenReleaseMembers = []string{
	"chrgen/level1.wav", "chrgen/level2.wav", "chrgen/level3.wav",
	"chrgen/char.wav", "chrgen/ok.wav", "chrgen/+_-.wav",
	"chrgen/skill/fsword.wav", "chrgen/skill/faxe.wav", "chrgen/skill/fclub.wav", "chrgen/skill/fpike.wav", "chrgen/skill/fbow.wav",
	"chrgen/skill/mfire.wav", "chrgen/skill/mwater.wav", "chrgen/skill/mair.wav", "chrgen/skill/mearth.wav", "chrgen/skill/mastral.wav",
}

// TestReleaseChargenSoundsThroughAppInput drives the installed menu,
// pre-create and detailed pages through App pointer input and asserts each
// press's audio request: the member, decoded independently from the
// install's sfx.res, centred and as a voice. Every device is a silent double,
// so nothing reaches the sound card; the master volume is 0 besides.
func TestReleaseChargenSoundsThroughAppInput(t *testing.T) {
	f := releaseFront(t)

	// The oracle: the install's own chrgen population and each member's
	// waveform, read outside the production named-sample resolver.
	sounds, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, e := range sounds.Entries() {
		if rest, ok := strings.CutPrefix(e.Address, "sfx/chrgen/"); ok {
			listed = append(listed, "chrgen/"+rest)
		}
	}
	want := append([]string{}, chrgenReleaseMembers...)
	sort.Strings(want)
	if !reflect.DeepEqual(listed, want) {
		t.Fatalf("installed chrgen members %q, want the claims' sixteen %q", listed, want)
	}
	decoded := map[string]audio.Sample{}
	for _, m := range chrgenReleaseMembers {
		raw, err := sounds.ReadFile("sfx/" + m)
		if err != nil {
			t.Fatal(err)
		}
		s, err := audio.DecodeWAV(raw, audio.DeviceRate)
		if err != nil || len(s.PCM) == 0 {
			t.Fatalf("%s: %v", m, err)
		}
		for other, o := range decoded {
			if reflect.DeepEqual(s, o) {
				t.Fatalf("%s decodes identical to %s; the oracle cannot tell them apart", m, other)
			}
		}
		decoded[m] = s
	}
	member := func(s audio.Sample) string {
		for m, d := range decoded {
			if reflect.DeepEqual(s, d) {
				return m
			}
		}
		return fmt.Sprintf("unmatched sample of %d frames", len(s.PCM))
	}

	rec := &exteriorRecorder{}
	f.Sound = SoundOptions{Enabled: true, Volume: 0}
	f.SoundPlayer, f.SoundBank = rec, OpenSounds(f.Archives.Root)
	f.SpeechPlayer, f.MusicPlayer, f.AmbientPlayer = &menuSettingsPlayer{}, &menuSettingsPlayer{}, &menuSettingsPlayer{}
	f.CutsceneAudioPlayer = fakeCutsceneAudioPlayer{}
	app := f.App("installed chargen sounds")
	defer app.StopAudio()
	app.SetCutscenes(nil)
	app.Layout(640, 480)

	requested := func() []string {
		out := []string{}
		for _, s := range rec.samples {
			out = append(out, member(s))
		}
		return out
	}
	var log []string
	expect := func(step string, add ...string) {
		t.Helper()
		log = append(log, add...)
		if got := requested(); !reflect.DeepEqual(got, log) {
			t.Fatalf("%s: requested %q, want %q", step, got, log)
		}
	}
	click := func(p image.Point) {
		t.Helper()
		for _, action := range []string{"press", "release"} {
			if err := app.HeadlessPointer(action, p.X, p.Y); err != nil {
				t.Fatal(err)
			}
		}
	}
	state := func() ui.HeadlessChargen {
		t.Helper()
		s, ok := app.HeadlessChargenState()
		if !ok {
			t.Fatalf("generator not showing: screen %v", app.Screen())
		}
		return s
	}

	// Main menu: the New Game press requests the menu's ok.wav; the
	// generator's open requests nothing.
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenPicker {
		if err := app.HeadlessActivate("@first"); err != nil {
			t.Fatal(err)
		}
	}
	if s := state(); s.Stage != ui.ChargenStagePreCreate {
		t.Fatalf("new game opened stage %q", s.Stage)
	}
	expect("menu New Game press", "chrgen/ok.wav")

	// The installed generator with its tip closed, so no press lands on the
	// panel. Its pages are this install's own art and masks.
	c := ui.NewChargen(f.ChargenSetup())
	c.CloseTip()
	if err := app.OpenChargen(c, func(res ui.ChargenResult) (ui.MapOpener, error) { return f.NewGameOpener(10, res), nil }); err != nil {
		t.Fatal(err)
	}
	if state().Name == "" {
		if err := app.HeadlessType("Hero", false); err != nil {
			t.Fatal(err)
		}
	}
	prePoint := func(owner string) image.Point {
		t.Helper()
		for y := 0; y < 480; y++ {
			for x := 0; x < 640; x++ {
				// Back rebuilds the tip popup (TOWN-518); a press there is the popup's.
				if got, ok := ui.PreCreateControlAt(c, image.Pt(x, y)); ok && got == owner && !c.TipPanel().Covers(image.Pt(x, y)) {
					return image.Pt(x, y)
				}
			}
		}
		t.Fatalf("no installed pre-create pixel is owned by %q", owner)
		return image.Point{}
	}
	for level := 1; level <= 3; level++ {
		click(prePoint(fmt.Sprintf("difficulty %d", level)))
		if got := state().Difficulty; got != level {
			t.Fatalf("difficulty %d press selected %d", level, got)
		}
		expect(fmt.Sprintf("difficulty %d press", level), fmt.Sprintf("chrgen/level%d.wav", level))
	}

	// Skills of both classes: a mage hero first, then back for a fighter.
	col := map[bool]string{false: "fighter", true: "mag"}
	for _, hero := range []int{3, 0} {
		mage := hero == 3
		for h := 0; h < 4; h++ {
			click(prePoint(fmt.Sprintf("choice %d", (hero+1+h)%4)))
			expect(fmt.Sprintf("hero %d press", (hero+1+h)%4), "chrgen/char.wav")
		}
		voices := len(rec.voices)
		click(prePoint("forward"))
		if s := state(); s.Stage != ui.ChargenStageDetailed {
			t.Fatalf("OK press left stage %q", s.Stage)
		}
		expect("OK press", "chrgen/ok.wav")
		for _, v := range rec.voices[voices-4:] {
			if v.playing {
				t.Fatal("the pre-create close left a page member playing")
			}
		}

		// Skill presses: every distinct index of the class's own installed
		// column mask, pressed at its first pixel. A press that selects a
		// skill requests that skill's member for this class.
		raw, err := f.Archives.Containers.ReadFile("graphics/interface/chrgen/" + col[mage] + "/mask.bmp")
		if err != nil {
			t.Fatal(err)
		}
		mask, err := bmp.DecodePaletted(raw)
		if err != nil {
			t.Fatal(err)
		}
		first := map[uint8]image.Point{}
		var codes []int
		for y := 0; y < 480; y++ {
			for x := 0; x < 320; x++ {
				code := mask.ColorIndexAt(x, y)
				if _, ok := first[code]; !ok {
					first[code] = image.Pt(160+x, y) // the column's decoded rect starts at x=160
					codes = append(codes, int(code))
				}
			}
		}
		sort.Ints(codes)
		skills := map[int]bool{}
		for _, code := range codes {
			before := len(rec.samples)
			click(first[uint8(code)])
			if len(rec.samples) == before {
				continue
			}
			chosen := -1
			for _, ctl := range state().Controls {
				if ctl.Kind == ui.ChargenControlSkill && ctl.Chosen {
					chosen = ctl.Focus
				}
			}
			slot := 6 + chosen
			if mage {
				slot += 5
			}
			if chosen < 0 || skills[chosen] {
				t.Fatalf("mask index %d requested %q with skill %d chosen", code, requested()[before:], chosen)
			}
			skills[chosen] = true
			expect(fmt.Sprintf("%s skill %d press", col[mage], chosen), chrgenReleaseMembers[slot])
		}
		if len(skills) != 5 {
			t.Fatalf("%s column presses reached %d skills, want 5", col[mage], len(skills))
		}
		if !mage {
			break
		}
		click(image.Pt(552, 160)) // the detailed page's Back button (MENU-139)
		if s := state(); s.Stage != ui.ChargenStagePreCreate {
			t.Fatalf("detailed Back left stage %q", s.Stage)
		}
		expect("detailed Back")
	}

	// Statistic steps at the page's fixed boxes, each one applied step.
	stat := func(i int) int {
		for _, ctl := range state().Controls {
			if ctl.Kind == ui.ChargenControlStatUp && ctl.Focus == 6+2*i {
				return ctl.Value
			}
		}
		t.Fatalf("no statistic %d", i)
		return 0
	}
	// The class's start spends the pool, so a decrease comes first: the
	// increase after it spends exactly what the decrease returned.
	v0 := stat(0)
	if v0 < 17 {
		t.Fatalf("statistic 0 starts at %d, too near the floor for two decreases", v0)
	}
	click(image.Pt(142, 64)) // statistic 0 down
	click(image.Pt(117, 64)) // statistic 0 up
	click(image.Pt(142, 64))
	if got := stat(0); got != v0-1 {
		t.Fatalf("statistic 0 %d after down, up, down; want %d", got, v0-1)
	}
	expect("statistic presses", "chrgen/+_-.wav", "chrgen/+_-.wav", "chrgen/+_-.wav")

	// A held statistic decrease: one request for the press, then one on the
	// 10th engine tick and every 4th tick after it (DIV-1493).
	v1 := stat(1)
	if v1 < 20 {
		t.Fatalf("statistic 1 starts at %d, too near the floor for five decreases", v1)
	}
	held := image.Pt(142, 96) // statistic 1 down
	if err := app.HeadlessPointer("press", held.X, held.Y); err != nil {
		t.Fatal(err)
	}
	expect("held press", "chrgen/+_-.wav")
	var ticks []int
	for tick := 1; tick <= 22; tick++ {
		before := len(rec.samples)
		if err := app.HeadlessPointer("move", held.X, held.Y); err != nil {
			t.Fatal(err)
		}
		if len(rec.samples) > before {
			ticks = append(ticks, tick)
		}
	}
	if err := app.HeadlessPointer("release", held.X, held.Y); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(ticks, []int{10, 14, 18, 22}) || stat(1) != v1-5 {
		t.Fatalf("held repeat on ticks %v, statistic 1 %d -> %d; want [10 14 18 22] and five steps", ticks, v1, stat(1))
	}
	expect("held repeat", "chrgen/+_-.wav", "chrgen/+_-.wav", "chrgen/+_-.wav", "chrgen/+_-.wav")

	centred := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	for i, p := range rec.places {
		if p != centred {
			t.Errorf("request %d (%s) placement %+v, want centred", i, requested()[i], p)
		}
	}
	covered := map[string]bool{}
	for _, m := range requested() {
		covered[m] = true
	}
	if len(covered) != len(chrgenReleaseMembers) {
		t.Errorf("the route requested %d of the sixteen members", len(covered))
	}
	t.Logf("%d chrgen requests over all sixteen installed members; held repeat on ticks %v", len(rec.samples), ticks)
}
