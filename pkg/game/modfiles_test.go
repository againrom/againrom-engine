package game

import (
	"bytes"
	"image"
	"image/color"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/vfs"
)

func modFilesOver(t *testing.T, order mod.Precedence) *modFiles {
	t.Helper()
	fs, err := vfs.Open(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &modFiles{fs: fs, order: order, owners: map[string]modFileEntry{}}
}

func servedText(t *testing.T, b *modFiles, address string) string {
	t.Helper()
	got, err := b.fs.ReadFile(address)
	if err != nil {
		t.Fatal(err)
	}
	return string(got)
}

// One path served by two mods goes to the mod that loads after the other,
// whichever is served first, and names the override; without an order the
// second is refused naming both mods. One mod serving a path twice keeps
// its later file, and a path folds as the archives fold it.
func TestModFilesDecideOnePathInModOrder(t *testing.T) {
	order := mod.Precedence{"later": {"earlier"}}
	for _, first := range []string{"earlier", "later"} {
		b := modFilesOver(t, order)
		second := map[string]string{"earlier": "later", "later": "earlier"}[first]
		if err := b.serveBytes(first, "files/common/main/text/a.txt", "main/text/a.txt", []byte(first)); err != nil {
			t.Fatal(err)
		}
		if err := b.serveBytes(second, "files/en/main/text/a.txt", "MAIN/Text/A.txt", []byte(second)); err != nil {
			t.Fatal(err)
		}
		if got := servedText(t, b, "main/text/a.txt"); got != "later" {
			t.Errorf("%s first: the path serves %q", first, got)
		}
		if len(b.overrides) != 1 || !strings.Contains(b.overrides[0], `mod "later"`) || !strings.Contains(b.overrides[0], `overrides mod "earlier"`) {
			t.Errorf("%s first: overrides %q", first, b.overrides)
		}
	}
	b := modFilesOver(t, nil)
	if err := b.serveBytes("a", "files/common/main/text/a.txt", "main/text/a.txt", []byte("a")); err != nil {
		t.Fatal(err)
	}
	err := b.serveBytes("b", "files/common/main/text/a.txt", "main/text/a.txt", []byte("b"))
	if err == nil || !strings.Contains(err.Error(), `mod "b": files/common/main/text/a.txt: install file main/text/a.txt is served by mod "a"`) ||
		!strings.Contains(err.Error(), "neither loads after the other") {
		t.Fatalf("unordered: %v", err)
	}
	if err := b.serveBytes("a", "data/missions.toml", "main/text/a.txt", []byte("a2")); err != nil || servedText(t, b, "main/text/a.txt") != "a2" {
		t.Fatalf("one mod twice: %v", err)
	}
	if err := b.serveBytes("", "", "main/text/a.txt", []byte("composed")); err != nil || servedText(t, b, "main/text/a.txt") != "composed" {
		t.Fatalf("a composed file: %v", err)
	}
}

func solidPicture(w, h int, c color.RGBA) *image.RGBA {
	pic := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(pic.Pix); i += 4 {
		pic.Pix[i], pic.Pix[i+1], pic.Pix[i+2], pic.Pix[i+3] = c.R, c.G, c.B, 0xff
	}
	return pic
}

// A picture takes the place of the install's when the install's decoder for
// that file reads it and it has the install's size.
func TestModPictureMustDecodeAsTheInstallsAtItsSize(t *testing.T) {
	installed := synth.BMP24(solidPicture(4, 3, color.RGBA{1, 2, 3, 255}))
	mask := synth.BMP8(image.NewPaletted(image.Rect(0, 0, 4, 3), color.Palette{color.Black, color.White}))
	if err := checkModPicture(installed, synth.BMP24(solidPicture(4, 3, color.RGBA{200, 0, 0, 255}))); err != nil {
		t.Fatal(err)
	}
	if err := checkModPicture(mask, synth.BMP8(image.NewPaletted(image.Rect(0, 0, 4, 3), color.Palette{color.White}))); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		installed, modded []byte
		want              string
	}{
		{installed, synth.BMP24(solidPicture(5, 3, color.RGBA{})), "the picture is 5x3; the install's is 4x3"},
		{installed, mask, "cannot decode this picture as the install's 24-bit picture"},
		{mask, installed, "cannot decode this picture as the install's 8-bit picture"},
		{installed, []byte("PNG"), "cannot decode this picture"},
		{[]byte("not a picture"), installed, "the install's picture at this path is not one the game decodes"},
	} {
		if err := checkModPicture(c.installed, c.modded); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%v, want %q", err, c.want)
		}
	}
}

// A mod's text reaches a converting second-game install in the Windows code
// page its text files are written in, so the reader's conversion gives the
// font's bytes back; every other install takes the font's bytes.
func TestModTextFileBytesReadBackThroughTheInstallCode(t *testing.T) {
	font := EncodeInstallTextFile([]byte("Восток\r\n"), 1)
	if bytes.Equal(font, []byte("Восток\r\n")) || len(font) != 8 {
		t.Fatalf("font bytes % x", font)
	}
	converting := TextCode{From: 1251, To: 866}
	file := converting.FileBytes(font)
	if bytes.Equal(file, font) || !bytes.Equal(converting.Bytes(file), font) {
		t.Fatalf("file % x reads back % x, want % x", file, converting.Bytes(file), font)
	}
	plain := TextCode{From: 866, To: 866}
	if !bytes.Equal(plain.FileBytes(font), font) {
		t.Fatal("a plain install's file bytes differ from the font's")
	}
}

// A mod that changes no game state does not count against a save's mod
// mark: a save with no mark loads under it alone, a marked save loads under
// the marked mods with it beside them, and a marked save still refuses when
// only it is active.
func TestStateFreeModsDoNotCountAgainstTheModMark(t *testing.T) {
	plain := modTestSAV(t)
	picture := mod.SetEntry{ID: "picture", Version: "1", Digest: "p", StateFree: true}
	pictureOnly := mod.Set{Base: "rom1-en", Mods: []mod.SetEntry{picture}}
	got, _, err := applyModMark(plain, mapload.ModContext{Set: pictureOnly})
	if err != nil || !bytes.Equal(got, plain) {
		t.Fatalf("an unmarked save under a picture mod: %v", err)
	}
	doc, err := sav.DecodeDocumentData(plain)
	if err != nil {
		t.Fatal(err)
	}
	if err := markDocumentForMods(&doc, pictureOnly.Marked(), nil, nil); err != nil {
		t.Fatal(err)
	}
	if after, err := sav.EncodeDocumentData(doc); err != nil || !bytes.Equal(after, plain) {
		t.Fatalf("a picture mod marked the save: %v", err)
	}
	if err := markDocumentForMods(&doc, modTestSet(150), nil, nil); err != nil {
		t.Fatal(err)
	}
	marked, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	both := modTestSet(150)
	both.Mods = append(both.Mods, picture)
	if _, _, err := applyModMark(marked, mapload.ModContext{Set: both}); err != nil {
		t.Fatalf("a marked save under its mod and a picture mod: %v", err)
	}
	if _, _, err := applyModMark(marked, mapload.ModContext{Set: pictureOnly}); err == nil || !strings.Contains(err.Error(), "skill-cap 1.0.0") {
		t.Fatalf("a marked save under a picture mod alone: %v", err)
	}
}

// A save marked under a mod that is state-free now still loads while that mod
// is active, and still refuses when it is not.
func TestMarkNamingAStateFreeModLoadsWhileItIsActive(t *testing.T) {
	plain := modTestSAV(t)
	doc, err := sav.DecodeDocumentData(plain)
	if err != nil {
		t.Fatal(err)
	}
	set := modTestSet(150)
	if err := markDocumentForMods(&doc, set, nil, nil); err != nil {
		t.Fatal(err)
	}
	marked, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	now := mod.Set{Base: set.Base}
	for _, m := range set.Mods {
		m.StateFree = true
		now.Mods = append(now.Mods, m)
	}
	if _, _, err := applyModMark(marked, mapload.ModContext{Set: now}); err != nil {
		t.Fatalf("a save marked under a mod that is state-free now, with it active: %v", err)
	}
	other := mod.Set{Base: set.Base, Mods: []mod.SetEntry{{ID: "elsewhere", Version: "1", Digest: "e", StateFree: true}}}
	if _, _, err := applyModMark(marked, mapload.ModContext{Set: other}); err == nil || !strings.Contains(err.Error(), "skill-cap 1.0.0") {
		t.Fatalf("a marked save with its mod not active: %v", err)
	}
	if _, _, err := applyModMark(marked, mapload.ModContext{Set: mod.Set{Base: set.Base}}); err == nil {
		t.Fatal("a marked save with no mod active loaded")
	}
}
