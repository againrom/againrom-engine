package ui

import (
	"crypto/sha256"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/spr16"
	"againrom/pkg/vfs"
)

func TestReleaseMissionPackKeepsTheCompleteInstalled80PixelIcon(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the mission-pack witness needs a lawful install")
	}
	hosts := []string{
		filepath.Join(root, "main.res"),
		filepath.Join(root, "graphics.res"),
		filepath.Join(root, "scenario.res"),
		filepath.Join(root, "world.res"),
		filepath.Join(root, "movies.res"),
	}
	containers, err := vfs.Open(hosts, nil)
	if err != nil {
		t.Fatalf("vfs.Open(%v): %v", hosts, err)
	}
	raw, err := containers.ReadFile("graphics/inventory/0001003.16a")
	if err != nil {
		t.Fatalf("read installed inventory icon: %v", err)
	}
	sprite, err := spr16.DecodeA(raw, true)
	if err != nil {
		t.Fatalf("decode installed inventory icon: %v", err)
	}
	if len(sprite.Frames) == 0 {
		t.Fatal("decoded installed inventory icon has no frames")
	}
	frame := sprite.Frames[0]
	if frame.Width != 80 || frame.Height != 80 {
		t.Fatalf("installed inventory icon frame = %dx%d, want 80x80", frame.Width, frame.Height)
	}
	icon := releaseMissionPackIcon(sprite.Palette, frame)

	bar, cols, ok := packBarRect(image.Pt(MissionFrameW, MissionFrameH))
	if !ok || bar != image.Rect(0, 678, 864, 768) || cols != 9 {
		t.Fatalf("mission pack geometry = (%v,%d,%v), want ((0,678)-(864,768),9,true)", bar, cols, ok)
	}
	subject := InventorySubject{Pack: []*image.RGBA{icon}, PackCount: []uint32{1}}
	plain := renderPackBarStars(subject, 0, cols, bar, nil, make([]uint32, cols), -1)
	if plain.Bounds() != image.Rect(0, 0, MissionFrameW, 90) {
		t.Fatalf("mission pack layer = %v, want full-frame-width (0,0)-(%d,90)", plain.Bounds(), MissionFrameW)
	}
	cell := image.Rect(64, 6, 144, 86)
	painted := 0
	for y := 0; y < 80; y++ {
		for x := 0; x < 80; x++ {
			base := invCellFill
			if x == 0 || y == 0 || x == 79 || y == 79 {
				base = invCellBorder
			}
			want := releaseMissionPackBlend(base, icon.RGBAAt(x, y))
			if got := plain.RGBAAt(cell.Min.X+x, cell.Min.Y+y); got != want {
				t.Fatalf("installed icon pixel (%d,%d) composed as %+v, want %+v", x, y, got, want)
			}
			if frame.Pixels[y*80+x].Painted {
				painted++
			}
		}
	}
	if painted == 0 {
		t.Fatal("installed 80x80 icon has no painted pixels")
	}

	starSubject := subject
	starSubject.PackStars = []bool{true}
	star := renderPackBarStars(starSubject, 0, cols, bar, nil, make([]uint32, cols), -1)
	if changed := releaseMissionPackDiff(plain, star, image.Rect(70, 12, 139, 81)); changed == 0 {
		t.Fatal("installed mission icon has no phase-0 star pixel in the decoded 6..74 footprint")
	}
	t.Logf("installed mission pack icon: painted=%d plain=%x phase0=%x",
		painted, sha256.Sum256(plain.Pix), sha256.Sum256(star.Pix))
}

func releaseMissionPackDiff(a, b *image.RGBA, r image.Rectangle) int {
	r = r.Intersect(a.Bounds()).Intersect(b.Bounds())
	n := 0
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if a.RGBAAt(x, y) != b.RGBAAt(x, y) {
				n++
			}
		}
	}
	return n
}

func releaseMissionPackIcon(palette []spr16.Color, frame spr16.FrameA) *image.RGBA {
	icon := image.NewRGBA(image.Rect(0, 0, frame.Width, frame.Height))
	for i, p := range frame.Pixels {
		if !p.Painted {
			continue
		}
		a := uint32(int(p.Level)+1) * 255 / 16
		var c spr16.Color
		if int(p.Index) < len(palette) {
			c = palette[p.Index]
		}
		o := i * 4
		icon.Pix[o] = uint8(uint32(c.R) * a / 255)
		icon.Pix[o+1] = uint8(uint32(c.G) * a / 255)
		icon.Pix[o+2] = uint8(uint32(c.B) * a / 255)
		icon.Pix[o+3] = uint8(a)
	}
	return icon
}

func releaseMissionPackBlend(dst, src color.RGBA) color.RGBA {
	inv := uint32(255 - src.A)
	return color.RGBA{
		R: uint8(uint32(src.R) + uint32(dst.R)*inv/255),
		G: uint8(uint32(src.G) + uint32(dst.G)*inv/255),
		B: uint8(uint32(src.B) + uint32(dst.B)*inv/255),
		A: uint8(uint32(src.A) + uint32(dst.A)*inv/255),
	}
}
