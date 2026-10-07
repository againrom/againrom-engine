package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

type helpScrollbarStyleWitness struct {
	dir      string
	frames   []*image.RGBA
	body     []byte
	manifest map[string]any
}

func prepareHelpScrollbarStyleWitness(t *testing.T, f *FrontEnd) *helpScrollbarStyleWitness {
	t.Helper()
	output := os.Getenv("AGAINROM_HELP_SCROLL_STYLE_WITNESS_DIR")
	root, err := filepath.EvalSymlinks(f.Archives.Root)
	if err != nil || !filepath.IsAbs(root) || !filepath.IsAbs(output) {
		t.Fatal("absolute asset root and AGAINROM_HELP_SCROLL_STYLE_WITNESS_DIR required", err)
	}
	ancestor, suffix := filepath.Clean(output), ""
	for {
		resolved, err := filepath.EvalSymlinks(ancestor)
		if err == nil {
			output = filepath.Join(resolved, suffix)
			break
		}
		if !os.IsNotExist(err) || filepath.Dir(ancestor) == ancestor {
			t.Fatal("resolve private help output", err)
		}
		suffix = filepath.Join(filepath.Base(ancestor), suffix)
		ancestor = filepath.Dir(ancestor)
	}
	rel, err := filepath.Rel(root, output)
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("help style output is inside the installed asset root")
	}
	rootHash := fmt.Sprintf("%x", sha256.Sum256([]byte(filepath.ToSlash(root))))
	dir := filepath.Join(output, rootHash, t.Name())
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	state, err := os.MkdirTemp(dir, "state-")
	if err != nil {
		t.Fatal(err)
	}
	f.Options = OptionsStore{Path: filepath.Join(state, "options.txt")}
	f.SoundPlayer, f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer, f.SpeechPlayer = nil, nil, nil, nil, nil
	const path = graphicsPrefix + "interface/scrlbars.256"
	raw, err := f.Archives.Containers.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != "8451c93d1a3bd772ab0e012bb4e3606cbbb7cb950184ad9ad40f631034fffdfe" {
		t.Fatal("installed help ornament source qualification", err)
	}
	frames, err := decodeCursor256Frames(path, raw)
	if err != nil || len(frames) != 26 {
		t.Fatal("installed help ornament frames", err)
	}
	for _, index := range []int{16, 18, 19, 20} {
		if frames[index] == nil || frames[index].Bounds() != image.Rect(0, 0, 24, 24) {
			t.Fatal("installed help ornament dimensions", index)
		}
	}
	body, err := f.Archives.Containers.ReadFile(HelpTextPath)
	if err != nil || len(body) == 0 {
		t.Fatal("independent installed help source body", err)
	}
	w := &helpScrollbarStyleWitness{dir: dir, frames: frames, body: body, manifest: map[string]any{
		"asset_root_sha256": rootHash,
		"scroll_sha256":     fmt.Sprintf("%x", sha256.Sum256(raw)),
		"help_body_sha256":  fmt.Sprintf("%x", sha256.Sum256(body)),
	}}
	t.Cleanup(func() {
		body, err := json.MarshalIndent(w.manifest, "", "  ")
		if err != nil {
			t.Error("marshal private help manifest", err)
			return
		}
		if err := os.WriteFile(filepath.Join(dir, "manifest.json"), body, 0600); err != nil {
			t.Error("write private help manifest", err)
		}
	})
	return w
}

func (w *helpScrollbarStyleWitness) capture(t *testing.T, f *FrontEnd, name string) *image.RGBA {
	t.Helper()
	layout, body, ok := f.live.view.HelpPanel()
	if !ok || !bytes.Equal([]byte(body), w.body) || layout.Box != image.Rect(76, 60, 564, 420) || layout.Scrollbar != image.Rect(422, 56, 446, 267) {
		t.Fatal("ordinary F1 body or geometry changed")
	}
	pic := helpFrame(t, f)
	file, err := os.Create(filepath.Join(w.dir, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	encodeErr, closeErr := png.Encode(file, pic), file.Close()
	if encodeErr != nil || closeErr != nil {
		t.Fatal(encodeErr, closeErr)
	}
	w.manifest[name+"_pixels_sha256"] = fmt.Sprintf("%x", sha256.Sum256(pic.Pix))
	return pic
}

func (w *helpScrollbarStyleWitness) check(t *testing.T, pic *image.RGBA, first, total int) map[int]int {
	t.Helper()
	height := max(163*12/total, 24)
	thumbY := 80 + (163-height)*first/(total-12)
	counts := map[int]int{}
	for y := 56; y < 267; y++ {
		for x := 422; x < 446; x++ {
			index, sy := 19, (y-80)%24
			switch {
			case y < 80:
				index, sy = 18, y-56
			case y >= 243:
				index, sy = 20, y-243
			case y >= thumbY && y < thumbY+height:
				index, sy = 16, (y-thumbY)*24/height
			}
			want := w.frames[index].RGBAAt(x-422, sy)
			if want.A != 255 {
				continue
			}
			if got := pic.RGBAAt(76+x, 60+y); got != want {
				t.Fatalf("help frame%d source%d,%d at%d,%d =%v want%v", index, x-422, sy, x, y, got, want)
			}
			counts[index]++
		}
	}
	for _, index := range []int{16, 18, 19, 20} {
		if counts[index] < 100 {
			t.Fatal("unqualified installed help source pixels", index, counts)
		}
	}
	return counts
}

func (w *helpScrollbarStyleWitness) run(t *testing.T, f *FrontEnd, a *ui.App) {
	t.Helper()
	a.Layout(1024, 768)
	helpKeys(t, a, "escape", "f1")
	v := f.live.view
	lines, visible := v.HelpLines()
	wantLines := map[string]int{"en": 35, "ru": 48}[filepath.Base(f.Archives.Root)]
	if lines != wantLines || visible != 12 {
		t.Fatal("installed help line qualification", lines, visible, wantLines)
	}
	tick, hash := f.live.world.Tick(), f.live.world.Hash()
	initial := w.capture(t, f, "help-initial")
	t.Run("installed-initial-bar", func(t *testing.T) { w.manifest["initial_frames"] = w.check(t, initial, 0, lines) })
	height := max(163*12/lines, 24)
	travel := 163 - height
	x, y, err := a.HeadlessHelpBarPoint("thumb")
	if err != nil {
		t.Fatal(err)
	}
	for _, point := range []struct {
		edge string
		y    int
	}{{"press", y}, {"move", y + travel/2}} {
		if err := a.HeadlessPointer(point.edge, x, point.y); err != nil {
			t.Fatal(err)
		}
	}
	first := (travel/2*(lines-12) + travel/2) / travel
	if got, _ := v.HelpScroll(); got != first {
		t.Fatalf("ordinary held midpoint is%d want%d", got, first)
	}
	mid := w.capture(t, f, "help-held-midpoint")
	t.Run("installed-proportional-thumb", func(t *testing.T) { w.manifest["midpoint_frames"] = w.check(t, mid, first, lines) })
	a.SetCutsceneScrollArt(nil)
	fallback := w.capture(t, f, "help-missing-art")
	bar := image.Rect(76+422, 60+56, 76+446, 60+267)
	different := 0
	for yy := 0; yy < 480; yy++ {
		for xx := 0; xx < 640; xx++ {
			if !image.Pt(xx, yy).In(bar) && mid.RGBAAt(xx, yy) != fallback.RGBAAt(xx, yy) {
				t.Fatalf("help ornament changed text/OK/outside bar at%d,%d", xx, yy)
			}
			if image.Pt(xx, yy).In(bar) && mid.RGBAAt(xx, yy) != fallback.RGBAAt(xx, yy) {
				different++
			}
		}
	}
	if different == 0 || fallback.RGBAAt(500, 116) != (color.RGBA{41, 68, 57, 255}) {
		t.Fatal("missing-art loss control did not restore the bevel painter")
	}
	w.manifest["bar_pixels_changed_without_art"] = different
	a.SetCutsceneScrollArt(w.frames)
	restored := w.capture(t, f, "help-restored-art")
	if got, _ := v.HelpScroll(); got != first || !bytes.Equal(mid.Pix, restored.Pix) {
		t.Fatal("art replacement reset help scroll or failed to restore paint")
	}
	if err := a.HeadlessPointer("move", x, y+travel); err != nil {
		t.Fatal(err)
	}
	if got, last := v.HelpScroll(); got != lines-12 || got != last {
		t.Fatal("ordinary held endpoint changed", got, last)
	}
	bottom := w.capture(t, f, "help-held-bottom")
	t.Run("installed-bottom-bar", func(t *testing.T) { w.manifest["bottom_frames"] = w.check(t, bottom, lines-12, lines) })
	if err := a.HeadlessPointer("release", 10, 10); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("move", x, y); err != nil {
		t.Fatal(err)
	}
	helpSteps(t, a, 3)
	if got, _ := v.HelpScroll(); got != lines-12 || !v.HelpOpen() || f.live.world.Tick() != tick || f.live.world.Hash() != hash {
		t.Fatal("help input or following idle frames changed World or escaped gesture ownership")
	}
	w.manifest["world_hash_tick_unchanged"] = true
	w.manifest["held_midpoint_before_release"] = first
	w.manifest["thumb_height"] = height
	w.manifest["all_pixels_outside_bar_unchanged"] = true
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	helpSteps(t, a, 8)
	if f.live.world.Tick() == tick {
		t.Fatal("missing-freeze loss control: closed help did not release World ticks")
	}
}
