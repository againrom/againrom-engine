package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func releaseLoadScroll(t *testing.T, f *FrontEnd) {
	t.Helper()
	root, err := filepath.Abs(f.Archives.Root)
	if err != nil {
		t.Fatal(err)
	}
	root, err = editorPhysicalDirectory(root)
	if err != nil {
		t.Fatal(err)
	}
	rootHash := sha256.Sum256([]byte(root))
	dir := os.Getenv("AGAINROM_LOAD_SCROLL_ARTIFACTS")
	if !filepath.IsAbs(dir) {
		t.Fatal("absolute AGAINROM_LOAD_SCROLL_ARTIFACTS required")
	}
	rel, err := filepath.Rel(root, filepath.Clean(dir))
	if err == nil && (rel == "." || rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)) {
		t.Fatal("Load scroll output is inside the installed asset root")
	}
	out := filepath.Join(dir, fmt.Sprintf("%s-%x", filepath.Base(root), rootHash[:4]))
	if err := os.MkdirAll(out, 0755); err != nil {
		t.Fatal(err)
	}
	const artPath = graphicsPrefix + "interface/scrlbars.256"
	raw, err := f.Archives.Containers.ReadFile(artPath)
	if err != nil {
		t.Fatal(err)
	}
	frames, err := decodeCursor256Frames(artPath, raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 26 {
		t.Fatalf("installed scroll frames=%d, want26", len(frames))
	}
	for _, index := range []int{16, 18, 19, 20} {
		if frames[index] == nil || frames[index].Bounds().Size() != image.Pt(24, 24) {
			t.Fatalf("installed frame%d bounds", index)
		}
	}
	rows := make([]ui.SaveEntry, 27)
	for i := range rows {
		rows[i] = ui.SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("Saved game %02d", i), Note: fmt.Sprintf("Selected slot %02d", i)}
	}
	loaded, prepared, removed := []string{}, []string{}, 0
	makeApp := func(record bool) *ui.App {
		a := f.App("load thumb witness")
		t.Cleanup(a.StopAudio)
		a.SetCutscenes(nil)
		a.Layout(640, 480)
		a.SetSaveSeams(nil, func() []ui.SaveEntry { return rows }, func(name string) (ui.MapOpener, bool, error) {
			if record {
				loaded = append(loaded, name)
			}
			return nil, false, fmt.Errorf("witness received exact Load token")
		})
		a.SetSaveDelete(func(string) bool { return true }, func(name string) (func() error, error) {
			if record {
				prepared = append(prepared, name)
			}
			return func() error {
				if record {
					removed++
				}
				return nil
			}, nil
		})
		if err := a.HeadlessKey("load"); err != nil {
			t.Fatal(err)
		}
		if a.Screen() != ui.ScreenLoad || len(a.HeadlessRows()) != 27 {
			t.Fatal("ordinary App fixture did not open27 Load rows")
		}
		return a
	}
	a, oracle := makeApp(true), makeApp(false)
	type receipt struct {
		Stage, Input         string
		X, Y, Selection, Top int
		Frame                string
		FrameSHA256          string
		OraclePixels         int
		OpaquePixels         map[int]int
	}
	receipts := []receipt{}
	key := func(a *ui.App, key string, count int) {
		for i := 0; i < count; i++ {
			if err := a.HeadlessKey(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	capture := func(stage, input string, x, y, selection, top int) {
		t.Helper()
		pic, note, err := a.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal(note, err)
		}
		want, note, err := oracle.HeadlessFrame()
		if err != nil || note != "" {
			t.Fatal(note, err)
		}
		checked := 0
		for yy := 152; yy < 375; yy++ {
			for xx := 122; xx < 528; xx++ {
				checked++
				if pic.RGBAAt(xx, yy) != want.RGBAAt(xx, yy) {
					t.Fatalf("%s before release: pixel%d,%d differs from keyboard selection%d/top%d", stage, xx, yy, selection, top)
				}
			}
		}
		thumb := image.Rect(504, 176+selection*118/26, 528, 200+selection*118/26)
		counts := map[int]int{}
		for _, index := range []int{18, 19, 20, 16} {
			r := image.Rect(504, 152, 528, 176)
			if index == 19 {
				r = image.Rect(504, 176, 528, 318)
			}
			if index == 20 {
				r = image.Rect(504, 318, 528, 342)
			}
			if index == 16 {
				r = thumb
			}
			for yy := r.Min.Y; yy < r.Max.Y; yy++ {
				for xx := r.Min.X; xx < r.Max.X; xx++ {
					if index == 19 && image.Pt(xx, yy).In(thumb) {
						continue
					}
					sy := (yy - r.Min.Y) % 24
					c := frames[index].RGBAAt(xx-504, sy)
					if c.A != 255 {
						continue
					}
					counts[index]++
					if got := pic.RGBAAt(xx, yy); got != c {
						t.Fatalf("%s installed frame%d pixel%d,%d=%v want%v", stage, index, xx, yy, got, c)
					}
				}
			}
			if counts[index] == 0 {
				t.Fatalf("%s frame%d vacuous", stage, index)
			}
		}
		if len(loaded) != 0 || len(prepared) != 0 || removed != 0 {
			t.Fatalf("%s scroll activated Load/Delete: %v/%v/%d", stage, loaded, prepared, removed)
		}
		name := stage + ".png"
		file, err := os.Create(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, pic)
		closeErr := file.Close()
		if encodeErr != nil || closeErr != nil {
			t.Fatal(encodeErr, closeErr)
		}
		bytes, err := os.ReadFile(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(bytes)
		receipts = append(receipts, receipt{stage, input, x, y, selection, top, name, fmt.Sprintf("%x", hash), checked, counts})
	}
	pointer := func(action string, x, y int) {
		t.Helper()
		if err := a.HeadlessPointer(action, x, y); err != nil {
			t.Fatal(err)
		}
	}
	capture("initial", "load", 0, 0, 0, 0)
	pointer("press", 516, 188)
	capture("pressed", "press", 516, 188, 0, 0)
	pointer("move", 516, 247)
	key(oracle, "down", 13)
	capture("held-midpoint", "move", 516, 247, 13, 4)
	pointer("move", 516, 306)
	key(oracle, "down", 13)
	capture("held-end", "move", 516, 306, 26, 17)
	pointer("move", 516, 221)
	key(oracle, "up", 19)
	capture("held-reversal", "move", 516, 221, 7, 7)
	pointer("move", 140, 247)
	key(oracle, "down", 6)
	capture("held-outside-bar", "move", 140, 247, 13, 7)
	pointer("release", 140, 272)
	capture("release-on-selected-row", "release", 140, 272, 13, 7)
	pointer("move", 516, 306)
	capture("release-tail", "move", 516, 306, 13, 7)
	pointer("release", 324, 392)
	capture("delete-release-tail", "release", 324, 392, 13, 7)
	pointer("press", 516, 247)
	if err := a.HeadlessFocus(false); err != nil {
		t.Fatal(err)
	}
	pointer("move", 516, 306)
	if err := a.HeadlessFocus(true); err != nil {
		t.Fatal(err)
	}
	pointer("move", 516, 306)
	pointer("release", 200, 392)
	capture("focus-cancelled", "release", 200, 392, 13, 7)
	pointer("press", 516, 247)
	key(a, "escape", 1)
	key(a, "load", 1)
	pointer("move", 516, 306)
	pointer("release", 200, 392)
	key(oracle, "home", 1)
	capture("reopened", "release", 200, 392, 0, 0)
	pointer("press", 516, 188)
	pointer("move", 516, 247)
	pointer("release", 516, 247)
	key(oracle, "down", 13)
	capture("final-selected", "release", 516, 247, 13, 4)
	key(a, "enter", 1)
	if len(loaded) != 1 || loaded[0] != "slot-13.sav" || len(prepared) != 0 || removed != 0 {
		t.Fatalf("exact token/action ownership=%v/%v/%d", loaded, prepared, removed)
	}
	manifest := struct {
		Root             string
		ArtSHA256        string
		Stages           []receipt
		Loaded, Prepared []string
		Removed          int
	}{root, fmt.Sprintf("%x", sha256.Sum256(raw)), receipts, loaded, prepared, removed}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(out, "receipt.json"), append(data, '\n'), 0644); err != nil {
		t.Fatal(err)
	}
	t.Logf("Load thumb: root=%s stages=%d,27 rows, keyboard oracle pixels/stage=%d; held13/top4,26/top17,reversal7/top7; exact token=%s; artifacts=%s", root, len(receipts), receipts[0].OraclePixels, loaded[0], out)
}
