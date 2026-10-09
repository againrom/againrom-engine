package game

// The shop's own art loading (1009 items 1 and 2): the merchant's base and
// animation families are read out of MoviesArchive and all four rack
// families out of GraphicsArchive through their literal installed addresses
// (SHOP-MERCHANT-046, SHOP-SHELF-047, SHOP-ANIMATION-081..085).
//
// EVERY FIXTURE IS SYNTHETIC (golden rule 2): a small solid-colour BMP built
// by internal/synth, written under the entry path each address resolves to,
// and opened through the same *vfs.FS the front end reads.

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/vfs"
)

func shopAnimationFixtureFS(t *testing.T, omit string) *vfs.FS {
	t.Helper()
	var graphics, movies []synth.File
	for rack := 0; rack < 4; rack++ {
		for frame := 1; frame <= shopRackFrameCount; frame++ {
			path := fmt.Sprintf("interface/shopanim/%02d/%d.bmp", 4-rack, frame)
			if path != omit {
				graphics = append(graphics, synth.File{Path: path, Data: shopSolidBMP(2, 2, color.RGBA{R: uint8(10 + rack*20 + frame), A: 0xff})})
			}
		}
	}
	for frame := 1; frame <= 29; frame++ {
		path := fmt.Sprintf("shopanim/pose2-3/%d.bmp", frame)
		if path != omit {
			movies = append(movies, synth.File{Path: path, Data: shopSolidBMP(2, 2, color.RGBA{G: uint8(frame), A: 0xff})})
		}
	}
	for _, family := range []string{"yes", "no"} {
		for frame := 2; frame <= 12; frame++ {
			path := fmt.Sprintf("shopanim/%s/%d.bmp", family, frame)
			if path != omit {
				movies = append(movies, synth.File{Path: path, Data: shopSolidBMP(2, 2, color.RGBA{B: uint8(frame), A: 0xff})})
			}
		}
	}
	dir := t.TempDir()
	graphicsPath := filepath.Join(dir, GraphicsArchive)
	moviesPath := filepath.Join(dir, MoviesArchive)
	if err := os.WriteFile(graphicsPath, synth.Archive(graphics), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moviesPath, synth.Archive(movies), 0o644); err != nil {
		t.Fatal(err)
	}
	fsys, err := vfs.Open([]string{graphicsPath, moviesPath}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return fsys
}

func shopSolidBMP(w, h int, c color.RGBA) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: c}, image.Point{}, draw.Src)
	return synth.BMP24(img)
}

// shopArtFixtureFS opens a filesystem over a graphics.res and a movies.res
// laid down in the test's own temporary tree, each holding one distinctly
// coloured picture per address loadShopArt resolves. openContainers
// (statics_test.go) opens one archive only, and the merchant's own picture
// lives in a second one, so this helper opens both together the way
// OpenArchives does.
func shopArtFixtureFS(t *testing.T) *vfs.FS {
	t.Helper()
	graphics := synth.Archive([]synth.File{
		{Path: "interface/shopanim/04/1.bmp", Data: shopSolidBMP(2, 2, color.RGBA{R: 0x10, A: 0xff})},
		{Path: "interface/shopanim/03/1.bmp", Data: shopSolidBMP(2, 2, color.RGBA{R: 0x11, A: 0xff})},
		{Path: "interface/shopanim/02/1.bmp", Data: shopSolidBMP(2, 2, color.RGBA{R: 0x12, A: 0xff})},
		{Path: "interface/shopanim/01/1.bmp", Data: shopSolidBMP(2, 2, color.RGBA{R: 0x13, A: 0xff})},
	})
	movies := synth.Archive([]synth.File{
		{Path: "shopanim/pose2-3/1.bmp", Data: shopSolidBMP(2, 2, color.RGBA{R: 0x20, A: 0xff})},
	})
	dir := t.TempDir()
	graphicsPath := filepath.Join(dir, GraphicsArchive)
	moviesPath := filepath.Join(dir, MoviesArchive)
	if err := os.WriteFile(graphicsPath, graphics, 0o644); err != nil {
		t.Fatalf("write %s: %v", graphicsPath, err)
	}
	if err := os.WriteFile(moviesPath, movies, 0o644); err != nil {
		t.Fatalf("write %s: %v", moviesPath, err)
	}
	fsys, err := vfs.Open([]string{graphicsPath, moviesPath}, nil)
	if err != nil {
		t.Fatalf("vfs.Open: %v", err)
	}
	return fsys
}

// The merchant's own picture comes out of MoviesArchive, at the address
// SHOP-MERCHANT-046 gives, and not out of graphics.res.
func TestLoadShopArtReadsTheMerchantOutOfMoviesArchive(t *testing.T) {
	art := loadShopArt(shopArtFixtureFS(t))
	merchant := art.Scene["merchant"]
	if len(merchant) != 1 || merchant[0] == nil {
		t.Fatal("merchant is missing with movies.res present")
	}
	if got := shopPixel(merchant[0]); got != (color.RGBA{R: 0x20, A: 0xff}) {
		t.Errorf("merchant pixel = %+v, want the movies.res fixture's own colour", got)
	}
}

// The four rack series' first frames come out of graphics.res, folder `4-i`
// for hit/draw index i (SHOP-SHELF-047), and each folder is distinct.
func TestLoadShopArtReadsTheShelfAnimationsFolderFourMinusI(t *testing.T) {
	art := loadShopArt(shopArtFixtureFS(t))
	want := [4]color.RGBA{{R: 0x10, A: 0xff}, {R: 0x11, A: 0xff}, {R: 0x12, A: 0xff}, {R: 0x13, A: 0xff}}
	for i, w := range want {
		frames := art.Scene[fmt.Sprintf("rack%d", i)]
		if len(frames) == 0 || frames[0] == nil {
			t.Fatalf("rack %d first frame is missing with graphics.res present", i)
		}
		if got := shopPixel(frames[0]); got != w {
			t.Errorf("rack %d pixel = %+v, want folder %d's own colour %+v", i, got, 4-i, w)
		}
	}
}

func TestLoadShopArtLeavesTheSceneEmptyWithNoArchive(t *testing.T) {
	if art := loadShopArt(nil); len(art.Scene) != 0 {
		t.Errorf("scene = %d entries with no archive", len(art.Scene))
	}
}

func TestLoadShopArtCachesEveryAcceptedAnimationFamily(t *testing.T) {
	art := loadShopArt(shopAnimationFixtureFS(t, ""))
	for i := 0; i < 4; i++ {
		frames := art.Scene[fmt.Sprintf("rack%d", i)]
		if len(frames) != shopRackFrameCount {
			t.Fatalf("rack %d cache = %d frames", i, len(frames))
		}
		if got := shopPixel(frames[10]).R; got != uint8(21+i*20) {
			t.Fatalf("rack %d file11 marker = %d", i, got)
		}
	}
	idle, yes, no := art.Scene["idle"], art.Scene["yes"], art.Scene["no"]
	if len(idle) != shopMerchantIdleCount || len(yes) != shopMerchantReactCount || len(no) != shopMerchantReactCount {
		t.Fatalf("merchant family counts = idle%d yes%d no%d", len(idle), len(yes), len(no))
	}
	if got := shopPixel(idle[0]).G; got != 2 {
		t.Fatalf("idle first marker = %d, want Pose file2", got)
	}
	if got := shopPixel(yes[10]).B; got != 12 {
		t.Fatalf("Yes last marker = %d, want file12", got)
	}
}

// A rack series keeps its loaded members and leaves a missing one empty; a
// merchant series with a missing member is dropped alone.
func TestLoadShopArtDropsOnlyTheIncompleteAnimationFamily(t *testing.T) {
	art := loadShopArt(shopAnimationFixtureFS(t, "interface/shopanim/03/7.bmp"))
	rack := art.Scene["rack1"]
	if len(rack) != shopRackFrameCount || rack[0] == nil || rack[6] != nil {
		t.Fatalf("incomplete rack = %d frames, first %v, missing member %v", len(rack), rack[0] != nil, rack[6] == nil)
	}
	if len(art.Scene["rack0"]) != shopRackFrameCount || len(art.Scene["rack2"]) != shopRackFrameCount || len(art.Scene["idle"]) != shopMerchantIdleCount {
		t.Fatal("missing rack member removed a complete sibling")
	}

	art = loadShopArt(shopAnimationFixtureFS(t, "shopanim/yes/6.bmp"))
	if art.Scene["yes"] != nil || len(art.Scene["no"]) != shopMerchantReactCount || len(art.Scene["idle"]) != shopMerchantIdleCount {
		t.Fatal("missing Yes member removed another merchant family")
	}
}

func shopPixel(pic image.Image) color.RGBA {
	b := pic.Bounds()
	return color.RGBAModel.Convert(pic.At(b.Min.X, b.Min.Y)).(color.RGBA)
}
