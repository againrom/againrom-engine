package main

import (
	"crypto/sha256"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/witness"
)

// frameRingHash hashes the pixels of every framed control and of the log box
// along their two outer rings, where a frame's border and the first row of
// its fill lie and no caption is drawn.
func frameRingHash(a *app) string {
	img := a.render()
	h := sha256.New()
	ring := func(r image.Rectangle) {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				if x < r.Min.X+2 || y < r.Min.Y+2 || x >= r.Max.X-2 || y >= r.Max.Y-2 {
					o := img.PixOffset(x, y)
					h.Write(img.Pix[o : o+4])
				}
			}
		}
	}
	box, _ := a.outputBox()
	ring(box)
	for _, it := range a.layout() {
		switch it.kind {
		case iButton, iToggle, iField:
			fmt.Fprintf(h, "%v|", it.rect)
			ring(it.rect)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// TestStarterFramesKeepTheirPixels pins the border and fill pixels of every
// framed control in both languages, both views and with a field focused.
func TestStarterFramesKeepTheirPixels(t *testing.T) {
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		for _, view := range []viewKind{viewMain, viewSettings} {
			for _, focus := range []fieldID{fNone, fAddBase, fVolume, fSaves} {
				a := rigWithLanguage(t, tc.language).app()
				a.setView(view)
				a.focus = focus
				key := fmt.Sprintf("%s/%d/%d", tc.lang, view, focus)
				got := frameRingHash(a)
				if starterFrameHashes[key] != got {
					t.Errorf("%s: ring hash %s, pinned %s", key, got, starterFrameHashes[key])
				}
			}
		}
	}
}

// modListStates are the mod list's three states: every enabled mod accepted
// and ordered, a refusal on an enabled row, and a mod that is not enabled and
// does not fit the base.
var modListStates = []struct {
	name    string
	enabled []string
	mods    map[string]string
}{
	{"ordered", []string{"c-first", "a-addon", "b-core"}, map[string]string{
		"a-addon": manifest("a-addon", `requires = ["b-core>=1.0"]`),
		"b-core":  manifest("b-core"),
		"c-first": manifest("c-first", `applies-to = ["rom1"]`),
	}},
	{"refused", []string{"c-first", "a-addon", "b-core", "clash"}, map[string]string{
		"a-addon": manifest("a-addon", `requires = ["b-core>=1.0"]`),
		"b-core":  manifest("b-core"),
		"c-first": manifest("c-first", `applies-to = ["rom1"]`),
		"clash":   manifest("clash", `conflicts = ["b-core"]`),
	}},
	{"misfit", []string{"a-addon", "b-core"}, map[string]string{
		"a-addon": manifest("a-addon", `requires = ["b-core>=1.0"]`),
		"b-core":  manifest("b-core"),
		"second":  manifest("second", `applies-to = ["rom2-en", "rom2-ru"]`),
	}},
}

// holds reports whether img has a pixel of colour c inside r.
func holds(img *image.RGBA, r image.Rectangle, c color.RGBA) bool {
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if img.RGBAAt(x, y) == c {
				return true
			}
		}
	}
	return false
}

// TestStarterModListFrames draws the main view with the mod list in each state
// in both languages. A numbered accepted row is drawn in the text colour, a
// refused row in the refusal colour with its caption. With
// AGAINROM_STARTER_MOD_FRAMES set each frame is written there as a PNG.
func TestStarterModListFrames(t *testing.T) {
	dir := os.Getenv("AGAINROM_STARTER_MOD_FRAMES")
	for _, tc := range []struct{ language, lang string }{{"english", "en"}, {"russian", "ru"}} {
		for _, st := range modListStates {
			r := rigWithLanguage(t, tc.language)
			for id, m := range st.mods {
				writeModFolder(t, filepath.Join(r.dir, "mods"), id, m)
			}
			a := r.app()
			a.s.Enabled = st.enabled
			img := a.render()
			rows := a.modRows()
			k, refused := 0, 0
			for _, it := range a.layout() {
				if it.kind != iModRow {
					continue
				}
				row := rows[k]
				k++
				if row.enabled && !strings.HasPrefix(it.text, fmt.Sprintf("%d. ", row.order)) {
					t.Errorf("%s/%s: row %q is not numbered %d", tc.lang, st.name, it.text, row.order)
				}
				bad, plain := holds(img, it.rect, colBad), holds(img, it.rect, colText)
				if row.refusal != "" {
					refused++
					if !bad || plain || !strings.Contains(it.text, row.refusal) {
						t.Errorf("%s/%s: refused row %q: refusal colour %v, text colour %v", tc.lang, st.name, it.text, bad, plain)
					}
				} else if bad || !plain {
					t.Errorf("%s/%s: accepted row %q: refusal colour %v, text colour %v", tc.lang, st.name, it.text, bad, plain)
				}
			}
			if want := map[string]int{"ordered": 0, "refused": 1, "misfit": 1}[st.name]; refused != want || k != len(st.mods) {
				t.Errorf("%s/%s: %d of %d rows refused, want %d of %d", tc.lang, st.name, refused, k, want, len(st.mods))
			}
			if dir != "" {
				witness.PNG(t, filepath.Join(dir, fmt.Sprintf("starter-mods-%s-%s.png", st.name, tc.lang)), img)
			}
		}
	}
}

// starterFrameHashes were recorded before the frames became specs.
var starterFrameHashes = map[string]string{
	"en/0/0": "6efe93b4b498d0f2",
	"en/0/1": "e54151fd48054503",
	"en/0/3": "6efe93b4b498d0f2",
	"en/0/4": "6efe93b4b498d0f2",
	"en/1/0": "4c625383ee70e1f8",
	"en/1/1": "4c625383ee70e1f8",
	"en/1/3": "9d835f55bfa1c3bd",
	"en/1/4": "2479d5f46a5879f0",
	"ru/0/0": "6efe93b4b498d0f2",
	"ru/0/1": "e54151fd48054503",
	"ru/0/3": "6efe93b4b498d0f2",
	"ru/0/4": "6efe93b4b498d0f2",
	"ru/1/0": "4c625383ee70e1f8",
	"ru/1/1": "4c625383ee70e1f8",
	"ru/1/3": "9d835f55bfa1c3bd",
	"ru/1/4": "2479d5f46a5879f0",
}
