package main

// The window's icon is handed over on the start-up path. A launch is watched
// through the desktop seam: the icon call and the run call are recorded and no
// window opens. Every icon here is a synthetic picture built by the test.

import (
	"bytes"
	"image"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/appicon"
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

func sizesOf(images []image.Image) []int {
	var sizes []int
	for _, img := range images {
		sizes = append(sizes, img.Bounds().Dx())
	}
	return sizes
}

// The start-up path gives the window the game's embedded icon, whatever the
// install: every size of the icon package, smallest first, and then the window.
func TestStartupHandsTheWindowTheGameIcon(t *testing.T) {
	install := missionInstall(t)
	log := watchDesktop(t)

	code, stderr := launch(t, install)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !reflect.DeepEqual(log.calls, []string{"icon", "run"}) {
		t.Fatalf("the window system was asked %v, want the icon and then the window", log.calls)
	}
	got := log.icons[0]
	if !reflect.DeepEqual(sizesOf(got), appicon.Sizes) {
		t.Fatalf("the icon call received sizes %v, want %v", sizesOf(got), appicon.Sizes)
	}
	want := appicon.Images(appicon.Game)
	for i, img := range got {
		if !bytes.Equal(img.(*image.NRGBA).Pix, want[i].(*image.NRGBA).Pix) {
			t.Errorf("the %d-pixel image is not the game icon", sizesOf(got)[i])
		}
	}
	other := appicon.Images(appicon.MapEditor)[len(want)-1].(*image.NRGBA)
	if bytes.Equal(got[len(got)-1].(*image.NRGBA).Pix, other.Pix) {
		t.Error("the game window got the map editor icon")
	}
	if strings.Contains(stderr, "window icon") {
		t.Errorf("the launch printed %q", stderr)
	}
}

// An install without the original executable still gets the game's icon: the
// icon no longer comes from the install.
func TestStartupIconDoesNotDependOnTheInstalledExecutable(t *testing.T) {
	install := missionInstall(t)
	for _, name := range []string{"rom.exe", "ROM.EXE"} {
		if err := os.Remove(filepath.Join(install, name)); err != nil && !os.IsNotExist(err) {
			t.Fatal(err)
		}
	}
	log := watchDesktop(t)
	if code, stderr := launch(t, install); code != 0 || !reflect.DeepEqual(log.calls, []string{"icon", "run"}) {
		t.Fatalf("exit %d, calls %v, stderr %s", code, log.calls, stderr)
	}
}

// A launch that opens no window asks the window system for nothing: the check
// and headless modes, and a mission number that names none.
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
			log := watchDesktop(t)
			launch(t, install, extra...)
			if len(log.calls) != 0 {
				t.Errorf("the window system was asked %v, want nothing", log.calls)
			}
		})
	}
}

// The game icon reaches the window on each lawful install, EN and RU alike.
// Nothing opens a window, and the saves go to a temporary directory.
func TestReleaseStartupHandsTheWindowTheGameIcon(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs a lawful install")
	}
	log := watchDesktop(t)
	code, stderr := launch(t, root)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !reflect.DeepEqual(log.calls, []string{"icon", "run"}) {
		t.Fatalf("the window system was asked %v, want the icon and then the window", log.calls)
	}
	if !reflect.DeepEqual(sizesOf(log.icons[0]), appicon.Sizes) {
		t.Fatalf("the icon call received sizes %v", sizesOf(log.icons[0]))
	}
}
