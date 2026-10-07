package game

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/winicon"
)

// iconPicture is an opaque picture of size by size pixels whose colours depend on
// seed, so two pictures of one size differ.
func iconPicture(size, seed int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			k := (x + 2*y + seed) % 13
			img.SetNRGBA(x, y, color.NRGBA{R: byte(k*19 + seed), G: byte(k*7 + 3*seed), B: byte(250 - k*11), A: 255})
		}
	}
	return img
}

// iconExecutable is a synthetic executable holding one icon group whose images
// are given as size and depth pairs, in the order listed.
func iconExecutable(pairs [][2]int) (exe []byte, pictures []*image.NRGBA) {
	var entries []synth.IconEntry
	for i, p := range pairs {
		size, depth := p[0], p[1]
		pic := iconPicture(size, i+1)
		pictures = append(pictures, pic)
		entries = append(entries, synth.IconEntry{ID: uint16(i + 1), Width: size, Height: size, Depth: depth, Data: synth.IconDIB(pic, depth)})
	}
	res := []synth.PEResource{{Type: 14, ID: 1, Language: 0x409, Data: synth.IconGroup(entries)}}
	res = append(res, synth.IconResources(entries, 0x409)...)
	return synth.PE(res), pictures
}

func writeInstallFile(t *testing.T, dir, name string, data []byte) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

// The window's icon is one image per stored size, the deepest of each, smallest
// first, wherever the executable lists them and however the install spells its
// name.
func TestWindowIconTakesTheDeepestImageOfEachSize(t *testing.T) {
	// Sizes 32, 16 and 48 at several depths in a mixed order, and 64 stored once
	// at 4 bits. wantFrom holds, for each size smallest first, the index in pairs
	// of the deepest image of that size.
	pairs := [][2]int{{32, 8}, {32, 4}, {48, 24}, {16, 4}, {64, 4}, {16, 8}, {48, 8}}
	wantFrom := []int{5, 0, 2, 4}
	wantSizes := []int{16, 32, 48, 64}
	for _, name := range []string{"rom.exe", "ROM.EXE", "Rom.Exe"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			exe, pictures := iconExecutable(pairs)
			writeInstallFile(t, dir, name, exe)
			writeInstallFile(t, dir, "other.exe", []byte("not read"))

			images, err := WindowIcon(dir)
			if err != nil {
				t.Fatal(err)
			}
			var sizes []int
			for _, img := range images {
				sizes = append(sizes, img.Bounds().Dx())
			}
			if !reflect.DeepEqual(sizes, wantSizes) {
				t.Fatalf("sizes %v, want %v", sizes, wantSizes)
			}
			for i, img := range images {
				nrgba, ok := img.(*image.NRGBA)
				if !ok {
					t.Fatalf("image %d is a %T, want *image.NRGBA", i, img)
				}
				if !bytes.Equal(nrgba.Pix, pictures[wantFrom[i]].Pix) {
					t.Errorf("the %d pixel image is not the picture of the deepest stored image", wantSizes[i])
				}
			}
		})
	}
}

// Where an install cannot give an icon the error says why, and names the file
// where the file is the cause.
func TestWindowIconReportsWhatIsMissing(t *testing.T) {
	good, _ := iconExecutable([][2]int{{16, 8}})
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string) string
		want  string
	}{
		{"no executable", func(t *testing.T, dir string) string { return dir }, "rom.exe is not in"},
		{"a folder of that name", func(t *testing.T, dir string) string {
			if err := os.Mkdir(filepath.Join(dir, "rom.exe"), 0o700); err != nil {
				t.Fatal(err)
			}
			return dir
		}, "rom.exe is not in"},
		{"a root that is not there", func(t *testing.T, dir string) string { return filepath.Join(dir, "absent") }, "absent"},
		{"not an executable", func(t *testing.T, dir string) string {
			writeInstallFile(t, dir, "ROM.EXE", bytes.Repeat([]byte("MZ but nothing more "), 10))
			return dir
		}, "ROM.EXE"},
		{"no icon group", func(t *testing.T, dir string) string {
			writeInstallFile(t, dir, "rom.exe", synth.PE(nil))
			return dir
		}, "no icon group"},
		{"cut off inside the icons", func(t *testing.T, dir string) string {
			writeInstallFile(t, dir, "rom.exe", good[:len(good)-40])
			return dir
		}, "rom.exe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			root := c.setup(t, t.TempDir())
			images, err := WindowIcon(root)
			if err == nil {
				t.Fatalf("got %d images and no error", len(images))
			}
			if images != nil {
				t.Errorf("an error came with %d images", len(images))
			}
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("error %q lacks %q", err, c.want)
			}
		})
	}
}

// deepestPerSize keeps the first image of the greatest depth of each size, so a
// tie is settled by the executable's own order, and orders sizes by width, then
// height. Images are told apart by identity: all are blank.
func TestDeepestPerSizeSettlesTiesByOrderAndSortsBySize(t *testing.T) {
	pic := func(w, h int) *image.NRGBA { return image.NewNRGBA(image.Rect(0, 0, w, h)) }
	tie1, tie2 := pic(32, 32), pic(32, 32)
	shallowTall, deepTall := pic(16, 32), pic(16, 32)
	square, small := pic(16, 16), pic(8, 8)
	got := deepestPerSize([]winicon.Image{
		{Depth: 8, Pix: tie1}, {Depth: 8, Pix: tie2}, {Depth: 4, Pix: shallowTall},
		{Depth: 4, Pix: square}, {Depth: 32, Pix: small}, {Depth: 8, Pix: deepTall},
	})
	want := []image.Image{small, square, deepTall, tie1}
	if len(got) != len(want) {
		t.Fatalf("%d images, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("position %d holds a %v image, want another: 8x8, 16x16, the 8 bit 16x32, the first 32x32", i, got[i].Bounds())
		}
	}
}
