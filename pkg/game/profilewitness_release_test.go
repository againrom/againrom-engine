package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/random"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The profile witness records new game, mission SAVE and same-session/cold
// LOAD. ROM1 SAVE hashes exclude only the temporary-directory field interior.
// AGAINROM_PROFILE_WITNESS_WRITE=1 rewrites the per-install recorded trace.

type profileWitness struct {
	t                 *testing.T
	f                 *FrontEnd
	a                 *ui.App
	lines             []string
	maskSaveDirectory bool
}

func profileWitnessFrameHash(pix *image.RGBA, screen ui.Screen) [32]byte {
	if screen != ui.ScreenSave {
		return sha256.Sum256(pix.Pix)
	}
	normalized := *pix
	normalized.Pix = append([]byte(nil), pix.Pix...)
	draw.Draw(&normalized, image.Rect(107, 53, 435, 78), image.Transparent, image.Point{}, draw.Src)
	return sha256.Sum256(normalized.Pix)
}

func TestProfileWitnessSaveDirectoryMaskPreservesOtherPixels(t *testing.T) {
	render := func(directory string) *image.RGBA {
		t.Helper()
		a := ui.NewApp("profile witness", nil, nil, nil)
		a.Layout(640, 480)
		a.SetWords(ui.AuthoredWords(), loadNoticeFont(), nil)
		a.SetSaveDialogSeams(ui.SaveDialogSeams{
			Directory: directory,
			List: func(path string) (ui.SaveDirectory, error) {
				return ui.SaveDirectory{Path: path}, nil
			},
			Prepare: func(ui.SaveRequest) (ui.PreparedSave, error) {
				return ui.PreparedSave{}, fmt.Errorf("frame fixture cannot write")
			},
		})
		v, err := ui.NewViewer("profile witness", terrain.Grid{Width: 1, Height: 1, Tiles: []uint16{0}}, &terrain.Tileset{})
		if err != nil {
			t.Fatal(err)
		}
		advance := func(...ui.NoticeAction) (ui.NoticeDest, string, ui.MapOpener) {
			return ui.NoticeStay, "", nil
		}
		if err := a.OpenMission(func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
			return v, nil, nil, nil, nil, advance, nil, nil, nil, nil, nil
		}); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := a.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		state, open := a.HeadlessSaveState()
		if !open || state.Request.Directory != directory || state.Focus != "name" {
			t.Fatalf("SAVE state = %+v, open=%t", state, open)
		}
		pix, _, err := a.HeadlessFrame()
		if err != nil {
			t.Fatal(err)
		}
		return pix
	}
	left := render(`C:\FirstTemporaryRoot\profile-saves`)
	right := render(`D:\AnotherTemporaryRoot\profile-saves`)
	leftBytes, rightBytes := append([]byte(nil), left.Pix...), append([]byte(nil), right.Pix...)
	if sha256.Sum256(left.Pix) == sha256.Sum256(right.Pix) {
		t.Fatal("different SAVE directory spellings produced identical raw frames")
	}
	masked := profileWitnessFrameHash(left, ui.ScreenSave)
	if masked != profileWitnessFrameHash(right, ui.ScreenSave) {
		t.Fatal("SAVE directory spelling changed the normalized frame")
	}
	if !bytes.Equal(left.Pix, leftBytes) || !bytes.Equal(right.Pix, rightBytes) {
		t.Fatal("normalizing a frame changed its original pixels")
	}
	for _, at := range []image.Point{
		image.Pt(107, 53), image.Pt(434, 77),
		image.Pt(106, 60), image.Pt(435, 60), image.Pt(110, 52), image.Pt(110, 78),
		image.Pt(38, 23), image.Pt(110, 272), image.Pt(270, 440),
	} {
		changed := *left
		changed.Pix = append([]byte(nil), left.Pix...)
		changed.Pix[changed.PixOffset(at.X, at.Y)] ^= 0xff
		inside := at.X >= 107 && at.X < 435 && at.Y >= 53 && at.Y < 78
		if equal := masked == profileWitnessFrameHash(&changed, ui.ScreenSave); equal != inside {
			t.Errorf("pixel %v normalized equality=%t, want %t", at, equal, inside)
		}
		if profileWitnessFrameHash(&changed, ui.ScreenMenu) != sha256.Sum256(changed.Pix) {
			t.Errorf("non-SAVE frame at %v was normalized", at)
		}
	}
	if profileWitnessFrameHash(left, ui.ScreenMenu) == profileWitnessFrameHash(right, ui.ScreenMenu) {
		t.Fatal("non-SAVE directory pixels stopped affecting the hash")
	}
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

func newProfileWitness(t *testing.T, f *FrontEnd, saves string, maskSaveDirectory bool) *profileWitness {
	a := f.App("profile-witness")
	a.Layout(640, 480)
	fixed := func() time.Time { return time.Unix(1700000000, 0).UTC() }
	f.ConfigureSaveSeams(a, SaveStore{Dir: saves}, OriginalStore{Dir: filepath.Join(saves, "original")}, fixed)
	return &profileWitness{t: t, f: f, a: a, maskSaveDirectory: maskSaveDirectory}
}

func (w *profileWitness) record(name string) {
	pix, _, err := w.a.HeadlessFrame()
	if err != nil && w.a.Screen() == ui.ScreenMap && w.f.live != nil && w.f.live.view != nil {
		pix, err = w.mapFrame()
	}
	frame := "error:" + fmt.Sprint(err)
	if err == nil {
		sum := sha256.Sum256(pix.Pix)
		if w.maskSaveDirectory {
			sum = profileWitnessFrameHash(pix, w.a.Screen())
		}
		frame = fmt.Sprintf("%dx%d:%s", pix.Bounds().Dx(), pix.Bounds().Dy(), hex.EncodeToString(sum[:8]))
		if w.maskSaveDirectory && w.a.Screen() == ui.ScreenSave {
			frame += ":directory-masked"
		}
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
	runProfileWitness(t, f, true)
}

func TestReleaseSecondGameProfileWitnessIsUnchanged(t *testing.T) {
	secondGameRoot(t)
	runProfileWitness(t, profileWitnessFront(t), false)
}

func runProfileWitness(t *testing.T, f *FrontEnd, maskSaveDirectory bool) {
	saves := t.TempDir()
	w := newProfileWitness(t, f, saves, maskSaveDirectory)
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
		for _, target := range []string{"EXIT", "GATES", "mission 10", "ENTER"} {
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
	cold := newProfileWitness(t, fresh, saves, maskSaveDirectory)
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
