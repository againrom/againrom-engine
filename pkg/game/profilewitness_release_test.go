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

	"againrom/pkg/base"
	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The profile witness records what a player meets on each game profile from
// the main menu: a new game up to its first mission, a SAVE there, a LOAD of
// that file in the same session and a LOAD of it from a fresh start. Each step
// records the screen, the composed frame's hash, the World hash when a mission
// runs and the hash of the SAV the session would write. The recorded lines
// live in testdata/profilewitness, one file per install text, and a run must
// reproduce them exactly. AGAINROM_PROFILE_WITNESS_WRITE=1 rewrites the file.

type profileWitness struct {
	t     *testing.T
	f     *FrontEnd
	a     *ui.App
	lines []string
}

func profileWitnessFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	now := time.Unix(5000, 0)
	f.TownAnimationNow = func() time.Time { return now }
	f.TownAnimationRandom = traceDraw(11)
	f.TownAmbientRandom = traceDraw(23)
	f.TavernRandom = traceDraw(37)
	f.ShopRandom = traceDraw(41)
	f.SchoolRandom = traceDraw(53)
	f.randomService().SetStreamSeed(random.TownWildlife, 7)
	f.randomService().SetStreamSeed(random.AmbientBirds, 7)
	f.randomService().SetStreamSeed(random.Music, 3)
	return f
}

func newProfileWitness(t *testing.T, f *FrontEnd, saves string) *profileWitness {
	a := f.App("profile-witness")
	a.Layout(640, 480)
	fixed := func() time.Time { return time.Unix(1700000000, 0).UTC() }
	f.ConfigureSaveSeams(a, SaveStore{Dir: saves}, OriginalStore{Dir: filepath.Join(saves, "original")}, fixed)
	return &profileWitness{t: t, f: f, a: a}
}

func (w *profileWitness) record(name string) {
	pix, _, err := w.a.HeadlessFrame()
	if err != nil && w.a.Screen() == ui.ScreenMap && w.f.live != nil && w.f.live.view != nil {
		pix, err = w.mapFrame()
	}
	frame := "error:" + fmt.Sprint(err)
	if err == nil {
		sum := sha256.Sum256(pix.Pix)
		frame = fmt.Sprintf("%dx%d:%s", pix.Bounds().Dx(), pix.Bounds().Dy(), hex.EncodeToString(sum[:8]))
	}
	world := "none"
	if w.f.live != nil && w.f.live.world != nil {
		world = fmt.Sprintf("%016x/%d", w.f.live.world.Hash(), w.f.live.world.Tick())
	}
	save := "none"
	if snap, label, err := w.f.Snapshot(w.a.Screen() == ui.ScreenMap); err != nil {
		save = "error:" + err.Error()
	} else if payload, err := w.f.ExportCurrentSave(snap, label); err != nil {
		save = "error:" + err.Error()
	} else {
		sum := sha256.Sum256(payload)
		save = hex.EncodeToString(sum[:8])
	}
	state := headlessAppSnapshot(w.f, w.a)
	w.lines = append(w.lines, fmt.Sprintf("%s screen=%s mission=%d place=%s members=%d purse=%d frame=%s world=%s save=%s msg=%q",
		name, state.Screen, state.Mission, state.TownPlace, len(state.Members), state.Purse, frame, world, save, w.a.HeadlessMessage()))
}

// mapFrame composes the map window around the party's first member.
func (w *profileWitness) mapFrame() (*image.RGBA, error) {
	at := image.Pt(0, 0)
	for _, m := range headlessAppSnapshot(w.f, w.a).Members {
		if e, ok := w.f.live.world.Entity(sim.EntityID(m.Entity)); ok && m.Entity != 0 {
			at = image.Pt(int(e.X), int(e.Y))
			break
		}
	}
	pix, _, err := w.f.live.view.HeadlessMapFrame(at, 320, 224)
	return pix, err
}

// closeNotices acknowledges every open mission notice.
func (w *profileWitness) closeNotices(tag string) {
	for i := 0; i < 64; i++ {
		if _, _, open := w.f.LiveNotice(); !open {
			return
		}
		if !w.do(fmt.Sprintf("%s-notice-%d", tag, i), HeadlessStep{Command: "activate", Target: "notice"}) {
			return
		}
	}
}

// do runs one production headless step and records the result; a refused step
// is recorded, not fatal, because the refusal is behaviour too.
func (w *profileWitness) do(name string, step HeadlessStep) bool {
	if err := runHeadlessStep(w.f, w.a, step, map[string]HeadlessState{}); err != nil {
		w.lines = append(w.lines, fmt.Sprintf("%s refused: %v", name, err))
		return false
	}
	w.record(name)
	return true
}

func (w *profileWitness) saveAndLoad(tag string) {
	w.closeNotices(tag)
	if !w.do(tag+"-save-open", HeadlessStep{Command: "save_open"}) {
		return
	}
	if !w.do(tag+"-save-name", HeadlessStep{Command: "save_edit", Name: "Witness " + tag, Target: string(ui.SaveSAV)}) {
		return
	}
	if !w.do(tag+"-save", HeadlessStep{Command: "save_action", Target: "save"}) {
		return
	}
	if w.do(tag+"-load", HeadlessStep{Command: "load", Target: "Witness " + tag}) && w.a.Screen() == ui.ScreenMap {
		w.do(tag+"-load-run", HeadlessStep{Command: "wait_ticks", Ticks: 48})
	}
}

func TestReleaseProfileWitnessIsUnchanged(t *testing.T) {
	f := profileWitnessFront(t)
	if p := f.Base().Profile; p.Limits.NoCharacterGeneration || p.Edition().Campaign == base.CampaignDestinations {
		t.Skip("the base's new game does not open a mission through generation; TestReleaseSecondGameProfileWitnessIsUnchanged covers the town start")
	}
	runProfileWitness(t, f)
}

func TestReleaseSecondGameProfileWitnessIsUnchanged(t *testing.T) {
	secondGameRoot(t)
	runProfileWitness(t, profileWitnessFront(t))
}

func runProfileWitness(t *testing.T, f *FrontEnd) {
	saves := t.TempDir()
	w := newProfileWitness(t, f, saves)
	w.record("menu")
	if f.Base().Profile.Edition().Campaign == base.CampaignDestinations {
		// A town-start campaign with no generator wired opens its town.
		w.do("new-game", HeadlessStep{Command: "activate", Target: "new game"})
		w.saveAndLoad("town")
		for _, target := range []string{"TAVERN", "TALK 517"} {
			w.do("activate-"+target, HeadlessStep{Command: "activate", Target: target})
		}
		for page := 0; page < 64; page++ {
			if !w.do(fmt.Sprintf("notice-%d", page), HeadlessStep{Command: "activate", Target: "notice"}) {
				break
			}
		}
		for _, target := range []string{"GATES", "mission 10", "ENTER"} {
			w.do("activate-"+target, HeadlessStep{Command: "activate", Target: target})
		}
	} else {
		w.do("new-game", HeadlessStep{Command: "activate", Target: "NEW GAME"})
		w.do("pick", HeadlessStep{Command: "activate", Target: "Mission 10: 10.alm"})
		w.do("generate", HeadlessStep{Command: "create_character", Character: &HeadlessCharacter{
			Name: "Witness", Sex: "Female", Class: "Mage", Skill: "Air",
			Stats: map[string]int{"Mind": 30, "Body": 20, "Reaction": 25, "Spirit": 25},
		}})
	}
	if w.a.Screen() != ui.ScreenMap {
		t.Fatalf("the witness did not reach the first mission: %s\n%s", w.a.Screen(), strings.Join(w.lines, "\n"))
	}
	w.do("run", HeadlessStep{Command: "wait_ticks", Ticks: 64})
	w.saveAndLoad("mission")

	// A fresh start loads the newest save from the main menu.
	fresh := profileWitnessFront(t)
	cold := newProfileWitness(t, fresh, saves)
	cold.record("fresh-menu")
	if cold.do("fresh-load", HeadlessStep{Command: "load", Target: "Witness mission"}) && cold.a.Screen() == ui.ScreenMap {
		cold.do("fresh-load-run", HeadlessStep{Command: "wait_ticks", Ticks: 48})
	}
	w.lines = append(w.lines, cold.lines...)

	got := strings.Join(w.lines, "\n") + "\n"
	path := filepath.Join("testdata", "profilewitness", f.Base().Profile.ID+".txt")
	if os.Getenv("AGAINROM_PROFILE_WITNESS_WRITE") != "" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote %d witness lines to %s", len(w.lines), path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("no recorded witness for this install (%v); record it on unchanged code", err)
	}
	want = bytes.ReplaceAll(want, []byte("\r\n"), []byte("\n"))
	if string(want) != got {
		wl, gl := strings.Split(string(want), "\n"), strings.Split(got, "\n")
		for i := 0; i < len(wl) || i < len(gl); i++ {
			var a, b string
			if i < len(wl) {
				a = wl[i]
			}
			if i < len(gl) {
				b = gl[i]
			}
			if a != b {
				t.Fatalf("witness line %d differs:\nwant %s\ngot  %s", i+1, a, b)
			}
		}
	}
}
