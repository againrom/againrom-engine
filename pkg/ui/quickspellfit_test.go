package ui

import (
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
)

// The marks of a book's slots as the book's 40 pixel cells draw them are written
// magnified to the folder AGAINROM_BOOK_MARK_FRAMES names, else to a temporary
// one, in the font of the install AGAINROM_ASSETS names.
func TestQuickSpellMarkFrame(t *testing.T) {
	dir := os.Getenv("AGAINROM_BOOK_MARK_FRAMES")
	if dir == "" {
		dir = t.TempDir()
	}
	font, game := chargenTestFont(), "test"
	if root := os.Getenv("AGAINROM_ASSETS"); root != "" {
		font, game = saveDialogInstallFont(t, root), filepath.Base(root)
	}
	labels := []string{"F5", "K", "Shift+K", "Ctrl+Shift+F10", "Ctrl+Alt+Numpad7"}
	const cellW, cellH, zoom = 40, 24, 6
	pic := image.NewRGBA(image.Rect(0, 0, cellW*len(labels), cellH*2))
	for x := 0; x < pic.Bounds().Dx(); x++ {
		for y := 0; y < pic.Bounds().Dy(); y++ {
			pic.SetRGBA(x, y, color.RGBA{R: 40, G: 48, B: 44, A: 255})
		}
	}
	for i, label := range labels {
		drawOriginalQuickSpellMark(pic, font, image.Rect(i*cellW, 0, (i+1)*cellW, cellH), label, false)
		drawQuickSpellMark(pic, font, image.Rect(i*cellW, cellH, (i+1)*cellW, 2*cellH), label)
	}
	big := image.NewRGBA(image.Rect(0, 0, pic.Bounds().Dx()*zoom, pic.Bounds().Dy()*zoom))
	for y := 0; y < big.Bounds().Dy(); y++ {
		for x := 0; x < big.Bounds().Dx(); x++ {
			big.SetRGBA(x, y, pic.RGBAAt(x/zoom, y/zoom))
		}
	}
	file := "book-marks-" + game + ".png"
	if err := os.WriteFile(filepath.Join(dir, file), witness.Bytes(t, big), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLongQuickSpellMarkStaysInsideItsCell(t *testing.T) {
	font := chargenTestFont()
	cell := image.Rect(8, 8, 8+28, 8+16)
	for _, label := range []string{"Shift+K", "Ctrl+Shift+F10", "Ctrl+Alt+Numpad7"} {
		for name, draw := range map[string]func(dst *image.RGBA){
			"original": func(dst *image.RGBA) { drawOriginalQuickSpellMark(dst, font, cell, label, false) },
			"plain":    func(dst *image.RGBA) { drawQuickSpellMark(dst, font, cell, label) },
		} {
			dst := image.NewRGBA(image.Rect(0, 0, 160, 40))
			draw(dst)
			drawn := false
			for y := 0; y < 40; y++ {
				for x := 0; x < 160; x++ {
					if dst.RGBAAt(x, y).A == 0 {
						continue
					}
					drawn = true
					if x >= cell.Max.X {
						t.Fatalf("%s mark %q draws at x %d, past its cell edge %d", name, label, x, cell.Max.X)
					}
				}
			}
			if !drawn {
				t.Fatalf("%s mark %q draws nothing", name, label)
			}
		}
	}
}
