package witness

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sample() *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i := range img.Pix {
		img.Pix[i] = uint8(i * 7)
	}
	img.SetNRGBA(1, 1, color.NRGBA{R: 9, G: 8, B: 7, A: 255})
	return img
}

func TestPNGWritesTheImageAndNamesTheTest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shot.png")
	img := sample()
	PNG(t, path, img)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := png.Decode(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("the file is not a PNG: %v", err)
	}
	if got.Bounds() != img.Bounds() {
		t.Fatalf("bounds %v, want %v", got.Bounds(), img.Bounds())
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 3; x++ {
			wr, wg, wb, wa := img.At(x, y).RGBA()
			gr, gg, gb, ga := got.At(x, y).RGBA()
			if wr != gr || wg != gg || wb != gb || wa != ga {
				t.Fatalf("pixel %d,%d differs", x, y)
			}
		}
	}
	if !bytes.Contains(raw, []byte("test\x00TestPNGWritesTheImageAndNamesTheTest")) {
		t.Error("the test's name is not in the file")
	}
	i := bytes.Index(raw, []byte("source\x00"))
	if i < 0 || !strings.Contains(string(raw[i+7:i+7+60]), "internal/witness/witness_test.go:") {
		t.Error("the caller's source is not in the file")
	}
}

func TestBytesCarriesNoText(t *testing.T) {
	raw := Bytes(t, sample())
	if bytes.Contains(raw, []byte("tEXt")) {
		t.Error("Bytes added a text chunk")
	}
	if _, err := Decode(bytes.NewReader(raw)); err != nil {
		t.Fatal(err)
	}
	cfg, err := DecodeConfig(bytes.NewReader(raw))
	if err != nil || cfg.Width != 3 || cfg.Height != 2 {
		t.Fatalf("config %v err %v", cfg, err)
	}
}
