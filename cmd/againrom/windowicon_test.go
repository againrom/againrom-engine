package main

// The window's icon is handed over on the start-up path. A launch is watched
// through the desktop seam: the icon call and the run call are recorded and no
// window opens. Every icon here is a synthetic picture built by the test.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// desktopLog records what a launch asked of the window system.
type desktopLog struct {
	calls []string
	icons [][]image.Image
}

// watchDesktop swaps the window system for a recorder until the test ends.
func watchDesktop(t *testing.T) *desktopLog {
	t.Helper()
	log := &desktopLog{}
	saved := desktop
	t.Cleanup(func() { desktop = saved })
	desktop.setIcon = func(images []image.Image) {
		log.calls = append(log.calls, "icon")
		log.icons = append(log.icons, images)
	}
	desktop.run = func(app *ui.App) error {
		log.calls = append(log.calls, "run")
		app.StopAudio()
		return nil
	}
	return log
}

// launch runs a normal start-up over root, silent and with its saves in a
// temporary directory, and returns the exit code and the error stream.
func launch(t *testing.T, root string, extra ...string) (int, string) {
	t.Helper()
	args := append([]string{"-assets", root, "-saves", filepath.Join(t.TempDir(), "saves"), "-sound=false", "-movies=false"}, extra...)
	var out, errOut bytes.Buffer
	code := runWithProfilePaths(args, noEnv, &out, &errOut, filepath.Join(root, "againrom.exe"), t.TempDir())
	return code, errOut.String()
}

// iconArt is a size by size picture of opaque colours with its first pixel
// transparent. The k-th colour differs from every other in its red channel.
func iconArt(size, colours int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			k := (x + 3*y) % colours
			img.SetNRGBA(x, y, color.NRGBA{R: byte(k*37 + 11), G: byte(k*91 + 5), B: byte(200 - k*3), A: 255})
		}
	}
	img.SetNRGBA(0, 0, color.NRGBA{})
	return img
}

// writeIconExecutable writes a synthetic executable holding one icon group of
// sixteen-colour and 256-colour images at 32, 48 and 16 pixels, listed the way
// the original lists its own: the shallower image of a size may come first. It
// returns the 256-colour pictures, smallest first.
func writeIconExecutable(t *testing.T, dir, name string) []*image.NRGBA {
	t.Helper()
	type spec struct{ size, depth int }
	specs := []spec{{32, 4}, {32, 8}, {48, 8}, {16, 4}, {48, 4}, {16, 8}}
	deep := map[int]*image.NRGBA{}
	var entries []synth.IconEntry
	for i, s := range specs {
		colours := 1 << s.depth
		if s.depth == 8 {
			colours = 200
		}
		art := iconArt(s.size, colours)
		if s.depth == 8 {
			deep[s.size] = art
		}
		entries = append(entries, synth.IconEntry{ID: uint16(i + 1), Width: s.size, Height: s.size, Depth: s.depth, Data: synth.IconDIB(art, s.depth)})
	}
	res := []synth.PEResource{{Type: 14, ID: 119, Language: 0x419, Data: synth.IconGroup(entries)}}
	res = append(res, synth.IconResources(entries, 0x419)...)
	if err := os.WriteFile(filepath.Join(dir, name), synth.PE(res), 0o644); err != nil {
		t.Fatal(err)
	}
	return []*image.NRGBA{deep[16], deep[32], deep[48]}
}

func sizesOf(images []image.Image) []int {
	var sizes []int
	for _, img := range images {
		sizes = append(sizes, img.Bounds().Dx())
	}
	return sizes
}

// The start-up path gives the window the icon its install's executable holds:
// one picture per size, the deepest colour depth of each, smallest first. Both
// spellings of the file name are found.
func TestStartupHandsTheWindowTheInstalledIcon(t *testing.T) {
	for _, name := range []string{"rom.exe", "ROM.EXE"} {
		t.Run(name, func(t *testing.T) {
			install := missionInstall(t)
			want := writeIconExecutable(t, install, name)
			log := watchDesktop(t)

			code, stderr := launch(t, install)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if !reflect.DeepEqual(log.calls, []string{"icon", "run"}) {
				t.Fatalf("the window system was asked %v, want the icon and then the window", log.calls)
			}
			got := log.icons[0]
			if len(got) != 3 || !reflect.DeepEqual(sizesOf(got), []int{16, 32, 48}) {
				t.Fatalf("the icon call received sizes %v, want 16, 32 and 48", sizesOf(got))
			}
			for i, img := range got {
				nrgba, ok := img.(*image.NRGBA)
				if !ok {
					t.Fatalf("image %d is a %T, want *image.NRGBA", i, img)
				}
				if !bytes.Equal(nrgba.Pix, want[i].Pix) {
					t.Errorf("the %d-pixel image is not the 256-colour picture of that size", want[i].Bounds().Dx())
				}
			}
			if strings.Contains(stderr, "window icon") {
				t.Errorf("a readable icon printed %q", stderr)
			}
		})
	}
}

// An install whose icon cannot be read still starts: the window opens, no icon
// is handed over, and the reason is reported once.
func TestStartupWithoutAReadableIconStillOpensTheWindow(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(t *testing.T, dir string)
		want   string
	}{
		{"no executable", func(t *testing.T, dir string) {}, "rom.exe is not in"},
		{"not an executable", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "rom.exe"), []byte(strings.Repeat("not an executable ", 20)), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "not a Windows executable"},
		{"an executable without icons", func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "rom.exe"), synth.PE(nil), 0o644); err != nil {
				t.Fatal(err)
			}
		}, "no icon group"},
		{"an executable cut off inside its icons", func(t *testing.T, dir string) {
			writeIconExecutable(t, dir, "rom.exe")
			path := filepath.Join(dir, "rom.exe")
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, data[:len(data)-len(data)/8], 0o644); err != nil {
				t.Fatal(err)
			}
		}, "rom.exe"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			install := missionInstall(t)
			c.mutate(t, install)
			log := watchDesktop(t)

			code, stderr := launch(t, install)
			if code != 0 {
				t.Fatalf("exit %d: %s", code, stderr)
			}
			if !reflect.DeepEqual(log.calls, []string{"run"}) {
				t.Fatalf("the window system was asked %v, want the window alone", log.calls)
			}
			if n := strings.Count(stderr, "window icon unavailable"); n != 1 || !strings.Contains(stderr, c.want) {
				t.Errorf("stderr %q does not report the icon once with %q", stderr, c.want)
			}
		})
	}
}

// A launch that opens no window reads no icon and asks the window system for
// nothing: the check and headless modes, and a mission number that names none.
func TestNoIconIsHandedWhereNoWindowOpens(t *testing.T) {
	scenario := filepath.Join(t.TempDir(), "menu.json")
	if err := os.WriteFile(scenario, []byte(`{"version":1,"steps":[{"command":"assert_state","state":{"screen":"menu"}}]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, extra := range map[string][]string{
		"check":                  {"-check"},
		"headless":               {"-headless", scenario},
		"a mission that is none": {"-mission", "0"},
	} {
		t.Run(name, func(t *testing.T) {
			install := missionInstall(t)
			writeIconExecutable(t, install, "rom.exe")
			log := watchDesktop(t)
			launch(t, install, extra...)
			if len(log.calls) != 0 {
				t.Errorf("the window system was asked %v, want nothing", log.calls)
			}
		})
	}
}

// The installed icon reaches the window on each lawful install: three images,
// 16, 32 and 48 pixels a side, each the 256-colour image of its size. The
// fingerprints are SHA-256 of the image's RGBA bytes, measured with an
// independent decoder. Nothing opens a window, and the saves go to a temporary
// directory.
func TestReleaseStartupHandsTheWindowTheInstalledIcon(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the installed executable's icon needs a lawful install")
	}
	log := watchDesktop(t)
	code, stderr := launch(t, root)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !reflect.DeepEqual(log.calls, []string{"icon", "run"}) {
		t.Fatalf("the window system was asked %v, want the icon and then the window", log.calls)
	}
	want := []struct {
		size   int
		sha256 string
	}{
		{16, "face43024b5b5724bccf6f36b3aba5a6624f1bcd08ea3a8a7ff38ca1a189ecbb"},
		{32, "5f9b6f80730751fbc68120b6b944c8b909d940bd8ad18a4a6ce6e1acebfaf9ad"},
		{48, "60d63672e4dc91bb6ab19ff442051491ca6e5ed5880d532083449efb5733bbd6"},
	}
	got := log.icons[0]
	if len(got) != len(want) {
		t.Fatalf("the icon call received %d images of sizes %v, want %d", len(got), sizesOf(got), len(want))
	}
	for i, w := range want {
		nrgba, ok := got[i].(*image.NRGBA)
		if !ok || nrgba.Bounds() != image.Rect(0, 0, w.size, w.size) {
			t.Errorf("image %d is %T %v, want a %d pixel square *image.NRGBA", i, got[i], got[i].Bounds(), w.size)
			continue
		}
		sum := sha256.Sum256(nrgba.Pix)
		if fp := hex.EncodeToString(sum[:]); fp != w.sha256 {
			t.Errorf("the %d pixel image has fingerprint %s, want %s", w.size, fp, w.sha256)
		}
	}
	t.Logf("the window-icon call received %d images: %d, %d and %d pixels a side", len(got), got[0].Bounds().Dx(), got[1].Bounds().Dx(), got[2].Bounds().Dx())
}
