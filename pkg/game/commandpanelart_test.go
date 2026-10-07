package game

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// commandPanelSource is a complete, synthetic command panel archive: the four
// decoded bodies and the two closing seams, each a distinct solid fill so a
// swapped read is visible in the pixel value rather than only in the bounds.
func commandPanelSource() chargenSource {
	return chargenSource{
		commandPanelHeadsPath:      synthBMP(160, 80, color.RGBA{R: 0x11, A: 0xff}),
		commandPanelHeadsSeamPath:  synthBMP(16, 80, color.RGBA{R: 0x12, A: 0xff}),
		commandPanelActivePath:     synthBMP(160, 80, color.RGBA{G: 0x22, A: 0xff}),
		commandPanelActiveSeamPath: synthBMP(16, 80, color.RGBA{G: 0x23, A: 0xff}),
		commandPanelDisabledPath:   synthBMP(160, 80, color.RGBA{B: 0x33, A: 0xff}),
		commandPanelSelectedPath:   synthBMP(160, 80, color.RGBA{R: 0x44, G: 0x44, A: 0xff}),
	}
}

func TestLoadCommandPanelArtReadsBodiesAndClosingSeams(t *testing.T) {
	got, err := LoadCommandPanelArt(commandPanelSource())
	if err != nil {
		t.Fatal(err)
	}
	if got.Heads == nil || got.HeadsSeam == nil || got.Active == nil || got.ActiveSeam == nil ||
		got.Disabled == nil || got.Selected == nil {
		t.Fatal("one of the six bitmaps was not retained")
	}
	if w, h := got.Heads.Bounds().Dx(), got.Heads.Bounds().Dy(); w != 160 || h != 80 {
		t.Fatalf("Heads is %dx%d, want 160x80", w, h)
	}
	if w, h := got.Active.Bounds().Dx(), got.Active.Bounds().Dy(); w != 160 || h != 80 {
		t.Fatalf("Active is %dx%d, want 160x80", w, h)
	}
	if w, h := got.HeadsSeam.Bounds().Dx(), got.HeadsSeam.Bounds().Dy(); w != 16 || h != 80 {
		t.Fatalf("HeadsSeam is %dx%d, want 16x80", w, h)
	}
	if w, h := got.ActiveSeam.Bounds().Dx(), got.ActiveSeam.Bounds().Dy(); w != 16 || h != 80 {
		t.Fatalf("ActiveSeam is %dx%d, want 16x80", w, h)
	}
	if w, h := got.Disabled.Bounds().Dx(), got.Disabled.Bounds().Dy(); w != 160 || h != 80 {
		t.Fatalf("Disabled is %dx%d, want 160x80", w, h)
	}
	if w, h := got.Selected.Bounds().Dx(), got.Selected.Bounds().Dy(); w != 160 || h != 80 {
		t.Fatalf("Selected is %dx%d, want 160x80", w, h)
	}
	// The four are distinct pictures and not one bitmap read four times: each
	// fixture's own top-left pixel is unique, so a load that mixed up two
	// addresses would land the wrong colour on the wrong field.
	at := func(pic image.Image) color.RGBA {
		return color.RGBAModel.Convert(pic.At(0, 0)).(color.RGBA)
	}
	if c := at(got.Heads); c.R != 0x11 {
		t.Error("Heads does not carry its own fixture's pixel")
	}
	if c := at(got.Active); c.G != 0x22 {
		t.Error("Active does not carry its own fixture's pixel")
	}
	if c := at(got.HeadsSeam); c.R != 0x12 {
		t.Error("HeadsSeam does not carry its own fixture's pixel")
	}
	if c := at(got.ActiveSeam); c.G != 0x23 {
		t.Error("ActiveSeam does not carry its own fixture's pixel")
	}
	if c := at(got.Disabled); c.B != 0x33 {
		t.Error("Disabled does not carry its own fixture's pixel")
	}
	if c := at(got.Selected); c.R != 0x44 {
		t.Error("Selected does not carry its own fixture's pixel")
	}
}

// ALL SIX OR NONE: a missing node fails the whole load, named by its own
// address, matching LoadAttackPointer's own contract.
func TestLoadCommandPanelArtMissingNodeIsAddressBearing(t *testing.T) {
	for _, addr := range []string{
		commandPanelHeadsPath,
		commandPanelHeadsSeamPath,
		commandPanelActivePath,
		commandPanelActiveSeamPath,
		commandPanelDisabledPath,
		commandPanelSelectedPath,
	} {
		src := commandPanelSource()
		delete(src, addr)
		if _, err := LoadCommandPanelArt(src); err == nil || !strings.Contains(err.Error(), addr) {
			t.Errorf("missing %s: error = %v, want an error naming %s", addr, err, addr)
		}
	}
}

func TestLoadCommandPanelArtNoSource(t *testing.T) {
	if _, err := LoadCommandPanelArt(nil); err == nil {
		t.Error("a nil source loaded a picture")
	}
}
