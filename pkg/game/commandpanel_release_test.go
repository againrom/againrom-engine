package game

import (
	"image"
	"strings"
	"testing"
)

// TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall is docs/1028-
// command-panel contract B1's own install-gated witness, composed WITH the
// story rather than as a follow-up. It reads ONE root, releaseFront's own
// contract; pipeline/check-release-tests.sh runs this whole package's suite
// once per root (en, then ru), which is how every other TestRelease* test in
// this package reaches both installs (G1).
//
// WHY IT EXISTS. `readChargenBMP` and `LoadInstallWords` are exercised
// against synthetic fixtures elsewhere in this package (commandpanelart_test.go,
// installtext_test.go); neither fixture can carry the shipped install's own
// byte geometry or its own main.txt content, so a defect in either — the
// wrong archive path, the wrong main.txt slot, a mismatched accelerator
// letter — could pass every synthetic test in this repository and still
// draw or state something wrong on a lawful install. This test checks two
// things no synthetic fixture can check:
//
//  1. The four panel bodies are 160x80 and distinct, while both shipped
//     closing seams are 16x80 and resolved beside their own ground.
//  2. Each of the eight resolved main.txt labels (`MENU-COMBAT-019`) ends
//     with the bracketed accelerator letter THIS BUILD actually binds for
//     that cell (app.go's own Attack/Move/Guard/Defend/Cast/Book/Doll keys),
//     on the INSTALL's own bytes rather than on the contract's transcribed
//     premise. A cell reassigned to the wrong main.txt slot, or a key bound
//     to a letter the install's own accelerator does not name, fails here.
func TestReleaseCommandPanelArtAndLabelsComposeFromTheRealInstall(t *testing.T) {
	f := releaseFront(t)
	if f.CommandPanelArt.Err() != nil || f.CommandPanelArt.Value() == nil {
		t.Fatalf("production command panel art did not resolve: %v", f.CommandPanelArt.Err())
	}
	art := f.CommandPanelArt
	want := image.Rect(0, 0, 160, 80)
	for name, pic := range map[string]image.Image{
		"Heads": art.Value().Heads, "Active": art.Value().Active,
		"Disabled": art.Value().Disabled, "Selected": art.Value().Selected,
	} {
		if pic == nil {
			t.Fatalf("%s did not resolve", name)
		}
		if pic.Bounds() != want {
			t.Errorf("%s bounds = %v, want %v (MENU-COMBAT-018)", name, pic.Bounds(), want)
		}
	}
	seamWant := image.Rect(0, 0, 16, 80)
	for name, pic := range map[string]image.Image{
		"HeadsSeam": art.Value().HeadsSeam, "ActiveSeam": art.Value().ActiveSeam,
	} {
		if pic == nil {
			t.Fatalf("%s did not resolve", name)
		}
		if pic.Bounds() != seamWant {
			t.Errorf("%s bounds = %v, want %v", name, pic.Bounds(), seamWant)
		}
	}
	if imagesEqual(art.Value().Heads, art.Value().Active) {
		t.Error("Heads and Active are pixel-identical over their whole 160x80 area; the panel's inactive and active grounds must be two distinct shipped bitmaps")
	}
	if imagesEqual(art.Value().Active, art.Value().Disabled) {
		t.Error("Active and Disabled are pixel-identical; the panel's active ground and its disabled-cell overlay must be two distinct shipped bitmaps")
	}
	if imagesEqual(art.Value().Disabled, art.Value().Selected) {
		t.Error("Disabled and Selected are pixel-identical; the panel's disabled-cell and selected-cell overlays must be two distinct shipped bitmaps")
	}

	// THE EIGHT LABELS AGAINST THE EIGHT KEYS THIS BUILD ACTUALLY BINDS
	// (app.go's own readAppInput), in the panel's own cell order
	// (commandpanel.go's commandCellAttack..commandCellRetreat).
	cellLetters := [8]string{"<A>", "<M>", "<G>", "<D>", "<C>", "<S>", "<T>", "<R>"}
	words := f.Words
	for cell, want := range cellLetters {
		got := words.Command[cell]
		if got == "" {
			t.Errorf("cell %d: no label resolved from main.txt", cell)
			continue
		}
		if !strings.HasSuffix(got, want) {
			tail := got
			if len(tail) > 4 {
				tail = tail[len(tail)-4:]
			}
			t.Errorf("cell %d: label ends %q, want the accelerator %q this build binds for it (raw label %q)",
				cell, tail, want, got)
		}
	}
}

// imagesEqual reports whether a and b agree at EVERY pixel of a shared
// bounds. The panel's four bitmaps share a common frame drawn at their outer
// edge (`MENU-COMBAT-018`), so a corner or edge sample alone cannot tell two
// of them apart; only a full-area compare can. Two files swapped by a wrong
// path constant would agree at every pixel, because they would be the SAME
// file read twice.
func imagesEqual(a, b image.Image) bool {
	if a == nil || b == nil {
		return false
	}
	ba, bb := a.Bounds(), b.Bounds()
	if ba.Size() != bb.Size() {
		return false
	}
	w, h := ba.Dx(), ba.Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if a.At(ba.Min.X+x, ba.Min.Y+y) != b.At(bb.Min.X+x, bb.Min.Y+y) {
				return false
			}
		}
	}
	return true
}
