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
	for rack, folder := range shopRackFolders {
		for frame := 1; frame <= shopRackFrameCount; frame++ {
			path := fmt.Sprintf("interface/shopanim/%s/%d.bmp", folder, frame)
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
	if art.Merchant == nil {
		t.Fatal("Merchant is nil with movies.res present")
	}
	if got := art.Merchant.RGBAAt(0, 0); got != (color.RGBA{R: 0x20, A: 0xff}) {
		t.Errorf("Merchant pixel = %+v, want the movies.res fixture's own colour", got)
	}
}

// The four shelf animations' first frame come out of graphics.res, folder
// `4-i` for hit/draw index i (SHOP-SHELF-047), and each folder is distinct.
func TestLoadShopArtReadsTheShelfAnimationsFolderFourMinusI(t *testing.T) {
	art := loadShopArt(shopArtFixtureFS(t))
	want := [4]color.RGBA{{R: 0x10, A: 0xff}, {R: 0x11, A: 0xff}, {R: 0x12, A: 0xff}, {R: 0x13, A: 0xff}}
	for i, w := range want {
		if art.ShelfAnim[i] == nil {
			t.Fatalf("ShelfAnim[%d] is nil with graphics.res present", i)
		}
		if got := art.ShelfAnim[i].RGBAAt(0, 0); got != w {
			t.Errorf("ShelfAnim[%d] pixel = %+v, want folder %d's own colour %+v", i, got, 4-i, w)
		}
	}
}

func TestLoadShopArtLeavesTheNewFieldsNilWithNoArchive(t *testing.T) {
	art := loadShopArt(nil)
	if art.Merchant != nil {
		t.Error("Merchant is non-nil with no archive")
	}
	for i, pic := range art.ShelfAnim {
		if pic != nil {
			t.Errorf("ShelfAnim[%d] is non-nil with no archive", i)
		}
	}
}

func TestLoadShopArtCachesEveryAcceptedAnimationFamily(t *testing.T) {
	art := loadShopArt(shopAnimationFixtureFS(t, ""))
	for i, frames := range art.RackAnimation {
		if len(frames) != shopRackFrameCount || frames[0] != art.ShelfAnim[i] {
			t.Fatalf("rack %d cache = %d frames, first alias %v", i, len(frames), frames[0] == art.ShelfAnim[i])
		}
		if got := frames[10].RGBAAt(0, 0).R; got != uint8(21+i*20) {
			t.Fatalf("rack %d file11 marker = %d", i, got)
		}
	}
	if len(art.MerchantIdle) != shopMerchantIdleCount || len(art.MerchantYes) != shopMerchantReactCount || len(art.MerchantNo) != shopMerchantReactCount {
		t.Fatalf("merchant family counts = idle%d yes%d no%d", len(art.MerchantIdle), len(art.MerchantYes), len(art.MerchantNo))
	}
	if got := art.MerchantIdle[0].RGBAAt(0, 0).G; got != 2 {
		t.Fatalf("idle first marker = %d, want Pose file2", got)
	}
	if got := art.MerchantYes[10].RGBAAt(0, 0).B; got != 12 {
		t.Fatalf("Yes last marker = %d, want file12", got)
	}
}

func TestLoadShopArtDropsOnlyTheIncompleteAnimationFamily(t *testing.T) {
	art := loadShopArt(shopAnimationFixtureFS(t, "interface/shopanim/03/7.bmp"))
	if art.RackAnimation[1] != nil || art.ShelfAnim[1] == nil {
		t.Fatalf("incomplete rack = series %#v first %v", art.RackAnimation[1], art.ShelfAnim[1] != nil)
	}
	if len(art.RackAnimation[0]) != shopRackFrameCount || len(art.RackAnimation[2]) != shopRackFrameCount || len(art.MerchantIdle) != shopMerchantIdleCount {
		t.Fatal("missing rack member removed a complete sibling")
	}

	art = loadShopArt(shopAnimationFixtureFS(t, "shopanim/yes/6.bmp"))
	if art.MerchantYes != nil || len(art.MerchantNo) != shopMerchantReactCount || len(art.MerchantIdle) != shopMerchantIdleCount {
		t.Fatal("missing Yes member removed another merchant family")
	}
}
