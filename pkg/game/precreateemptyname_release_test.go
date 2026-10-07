package game

import (
	"fmt"
	"image"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// preCreateRig is the installed pre-create page opened by the main menu's NEW
// GAME, its tip panel closed, on a front end whose sound device records every
// request. A request is named by its waveform: ok.wav and char.wav are decoded
// from the install's sfx.res outside the production resolver.
type preCreateRig struct {
	t                    *testing.T
	app                  *ui.App
	c                    *ui.Chargen
	rec                  *exteriorRecorder
	opened               int
	ok, char             audio.Sample
	okAt, heroAt, backAt image.Point
	fieldAt              image.Point
}

func newPreCreateRig(t *testing.T) *preCreateRig {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	sounds, err := OpenContainers(filepath.Join(f.Archives.Root, "sfx.res"))
	if err != nil {
		t.Fatal(err)
	}
	decode := func(member string) audio.Sample {
		t.Helper()
		raw, err := sounds.ReadFile("sfx/chrgen/" + member)
		if err != nil {
			t.Fatal(err)
		}
		s, err := audio.DecodeWAV(raw, audio.DeviceRate)
		if err != nil || len(s.PCM) == 0 {
			t.Fatalf("%s: %v", member, err)
		}
		return s
	}
	r := &preCreateRig{t: t, rec: &exteriorRecorder{}, ok: decode("ok.wav"), char: decode("char.wav")}
	if reflect.DeepEqual(r.ok, r.char) {
		t.Fatal("ok.wav and char.wav decode identical; the oracle cannot tell them apart")
	}
	f.Sound = SoundOptions{Enabled: true, Volume: 0}
	f.SoundPlayer, f.SoundBank = r.rec, OpenSounds(f.Archives.Root)
	f.SpeechPlayer, f.MusicPlayer, f.AmbientPlayer = &menuSettingsPlayer{}, &menuSettingsPlayer{}, &menuSettingsPlayer{}
	f.CutsceneAudioPlayer = fakeCutsceneAudioPlayer{}
	r.app = f.App("pre-create empty name")
	t.Cleanup(r.app.StopAudio)
	r.app.SetCutscenes(nil)
	r.app.Layout(640, 480)
	r.c = preCreateNameOpen(t, f, r.app)
	r.opened = len(r.rec.samples)

	// The tip panel covers the field and a corner of OK until it is closed.
	if tip := r.c.TipPanel(); !tip.Rect.Empty() {
		box := ui.TipPanelCloseRect(tip.Rect)
		r.click(box.Min.Add(box.Size().Div(2)))
		if !r.c.TipPanel().Rect.Empty() {
			t.Fatal("the tip panel's Close left it open")
		}
	}
	r.okAt, r.heroAt, r.backAt = r.owned("forward"), r.owned("choice 3"), r.owned("back")
	r.fieldAt = r.owned("name")
	r.opened = len(r.rec.samples)
	return r
}

// owned is the pixel the page's own hit test gives a control: the owned pixel
// nearest the centroid of all it owns.
func (r *preCreateRig) owned(owner string) image.Point {
	r.t.Helper()
	var pixels []image.Point
	var sum image.Point
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if got, ok := ui.PreCreateControlAt(r.c, image.Pt(x, y)); ok && got == owner {
				pixels = append(pixels, image.Pt(x, y))
				sum = sum.Add(image.Pt(x, y))
			}
		}
	}
	if len(pixels) == 0 {
		r.t.Fatalf("no installed pre-create pixel is owned by %q", owner)
	}
	centre := sum.Div(len(pixels))
	return slices.MinFunc(pixels, func(a, b image.Point) int {
		da, db := a.Sub(centre), b.Sub(centre)
		return (da.X*da.X + da.Y*da.Y) - (db.X*db.X + db.Y*db.Y)
	})
}

func (r *preCreateRig) state() ui.HeadlessChargen {
	r.t.Helper()
	s, ok := r.app.HeadlessChargenState()
	if !ok {
		r.t.Fatalf("the generator is not showing: %s", r.app.Screen())
	}
	return s
}

// requested names each sound request since the page opened.
func (r *preCreateRig) requested() []string {
	out := []string{}
	for _, s := range r.rec.samples[r.opened:] {
		switch {
		case reflect.DeepEqual(s, r.ok):
			out = append(out, "ok.wav")
		case reflect.DeepEqual(s, r.char):
			out = append(out, "char.wav")
		default:
			out = append(out, fmt.Sprintf("a sample of %d frames", len(s.PCM)))
		}
	}
	return out
}

// click is a primary press and release at p, two headless dispatches.
func (r *preCreateRig) click(p image.Point) {
	r.t.Helper()
	for _, edge := range []string{"press", "release"} {
		if err := r.app.HeadlessPointer(edge, p.X, p.Y); err != nil {
			r.t.Fatal(err)
		}
	}
}

// press moves the focus to a control with the page's own arrow keys and presses
// Enter.
func (r *preCreateRig) press(kind string) {
	r.t.Helper()
	s := r.state()
	if err := headlessChargenPress(r.app, &s, kind, ""); err != nil {
		r.t.Fatal(err)
	}
}

// erase focuses the name field and removes its text with key 8, one byte per
// key.
func (r *preCreateRig) erase() {
	r.t.Helper()
	s := r.state()
	if err := headlessChargenFocus(r.app, &s, ui.ChargenControlName, ""); err != nil {
		r.t.Fatal(err)
	}
	for range len(s.Name) {
		if err := r.app.HeadlessType("", true); err != nil {
			r.t.Fatal(err)
		}
	}
	if got := r.state().Name; got != "" {
		r.t.Fatalf("key 8 left %q in the field", got)
	}
}

// typeName clicks the field and types name into it.
func (r *preCreateRig) typeName(name string) {
	r.t.Helper()
	r.click(r.fieldAt)
	if err := r.app.HeadlessType(name, false); err != nil {
		r.t.Fatal(err)
	}
	if got := r.state().Name; got != name {
		r.t.Fatalf("typed %q and the field holds %q", name, got)
	}
}

// stays fails unless the pre-create page is still showing with an empty field
// and the sound device has been asked for exactly want since it opened.
func (r *preCreateRig) stays(route string, want ...string) {
	r.t.Helper()
	s := r.state()
	if s.Stage != ui.ChargenStagePreCreate || s.Name != "" || r.app.Screen() != ui.ScreenChargen {
		r.t.Fatalf("%s with an empty name: the %s page, name %q, screen %s; want the pre-create page and an empty field",
			route, s.Stage, s.Name, r.app.Screen())
	}
	if got := r.requested(); !slices.Equal(got, want) {
		r.t.Fatalf("%s with an empty name: requested %q, want %q", route, got, want)
	}
}

// continues fails unless the detailed page is showing and the sound device has
// been asked for exactly want since the pre-create page opened.
func (r *preCreateRig) continues(route string, want ...string) {
	r.t.Helper()
	if s := r.state(); s.Stage != ui.ChargenStageDetailed {
		r.t.Fatalf("%s with a name typed: the %s page, want the detailed page", route, s.Stage)
	}
	if got := r.requested(); !slices.Equal(got, want) {
		r.t.Fatalf("%s with a name typed: requested %q, want %q", route, got, want)
	}
}

// The installed pre-create page with its name field emptied, driven through
// ui.App input. TEXT-CHARGEN-028: the OK control continues only for a
// non-empty name. VIDEO-SFX-058: with an empty name, OK and Enter request
// nothing and send nothing. The hero double-click is the owner's second route
// to the same transition (DIV-1507) and takes the same test. Each such route
// leaves the page as it is, requests nothing beyond the hero's own first
// press, and with a name typed afterwards the same route continues. The amulet
// and Escape do not test the name, so an empty field never holds a player on
// the page.
func TestReleasePreCreateEmptyNameKeepsThePageThroughAppInput(t *testing.T) {
	routes := []struct {
		name  string
		leave func(r *preCreateRig)
		// prior is what one run of the route requests when it does not
		// continue; extra is what it requests when it does.
		prior, extra []string
	}{
		{"OK press", func(r *preCreateRig) { r.click(r.okAt) }, nil, []string{"ok.wav"}},
		{"Enter on OK", func(r *preCreateRig) { r.press(ui.ChargenControlForward) }, nil, []string{"ok.wav"}},
		// The first click of the pair is a hero press and requests char.wav;
		// the second requests nothing and is the continue.
		{"hero double-click", func(r *preCreateRig) { r.click(r.heroAt); r.click(r.heroAt) }, []string{"char.wav"}, []string{"char.wav"}},
	}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			r := newPreCreateRig(t)
			r.erase()
			route.leave(r)
			r.stays(route.name, route.prior...)
			r.typeName("Bob")
			route.leave(r)
			r.continues(route.name, append(slices.Clone(route.prior), route.extra...)...)
		})
	}

	// Back from the detailed page reopens the pre-create page with the typed
	// name; with that name erased the continue routes keep the page again.
	t.Run("after Back", func(t *testing.T) {
		r := newPreCreateRig(t)
		r.erase()
		r.typeName("Bob")
		r.click(r.okAt)
		r.continues("OK press", "ok.wav")
		r.press(ui.ChargenControlBack)
		if s := r.state(); s.Stage != ui.ChargenStagePreCreate || s.Name != "Bob" {
			t.Fatalf("Back: the %s page with the field %q, want the pre-create page keeping Bob", s.Stage, s.Name)
		}
		r.erase()
		r.press(ui.ChargenControlForward)
		r.stays("Enter on OK after Back", "ok.wav")
		r.typeName("Al")
		r.press(ui.ChargenControlForward)
		r.continues("Enter on OK", "ok.wav", "ok.wav")
	})

	for _, route := range []struct {
		name  string
		leave func(r *preCreateRig)
	}{
		{"amulet press", func(r *preCreateRig) { r.click(r.backAt) }},
		{"Escape", func(r *preCreateRig) {
			if err := r.app.HeadlessKey("escape"); err != nil {
				r.t.Fatal(err)
			}
		}},
	} {
		t.Run(route.name+" leaves", func(t *testing.T) {
			r := newPreCreateRig(t)
			r.erase()
			route.leave(r)
			if r.app.Screen() != ui.ScreenMenu {
				t.Fatalf("%s with an empty name left the player on screen %s, want the menu", route.name, r.app.Screen())
			}
		})
	}
}
