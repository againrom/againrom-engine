package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"strings"
	"testing"

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
		pix, _, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame})
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

func TestProfileWitnessDifferencesNameEveryDifferingLine(t *testing.T) {
	if got := profileWitnessDifferences("a\nb\n", "a\nb\n"); got != "" {
		t.Fatalf("equal traces reported %q", got)
	}
	got := profileWitnessDifferences("a\nb\nc\nd\n", "a\nB\nc\nD\ne\n")
	for _, part := range []string{
		"3 witness lines differ:",
		"witness line 2 differs:\nwant b\ngot  B",
		"witness line 4 differs:\nwant d\ngot  D",
		"witness line 5 differs:\nwant \ngot  e",
	} {
		if !strings.Contains(got, part) {
			t.Errorf("report lacks %q:\n%s", part, got)
		}
	}
	if strings.Contains(got, "line 1 ") || strings.Contains(got, "line 3 ") {
		t.Errorf("report names an equal line:\n%s", got)
	}
}

func (w *profileWitness) record(name string) {
	pix, _, err := w.a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame})
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

func profileWitnessDifferences(want, got string) string {
	if want == got {
		return ""
	}
	wl, gl := strings.Split(want, "\n"), strings.Split(got, "\n")
	var b strings.Builder
	differing := 0
	for i := 0; i < len(wl) || i < len(gl); i++ {
		var w, g string
		if i < len(wl) {
			w = wl[i]
		}
		if i < len(gl) {
			g = gl[i]
		}
		if w != g {
			differing++
			fmt.Fprintf(&b, "\nwitness line %d differs:\nwant %s\ngot  %s", i+1, w, g)
		}
	}
	return fmt.Sprintf("%d witness lines differ:%s", differing, b.String())
}
