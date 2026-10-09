package game

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/ui"
)

func screenshotFixture() *image.RGBA {
	frame := image.NewRGBA(image.Rect(0, 0, 13, 7))
	for y := 0; y < 7; y++ {
		for x := 0; x < 13; x++ {
			frame.SetRGBA(x, y, color.RGBA{R: byte(x * 17), G: byte(y * 31), B: byte(x + y), A: 255})
		}
	}
	return frame
}

func screenshotWantPNG(t *testing.T, path string, want image.Image) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	got, err := png.Decode(file)
	if err != nil || got.Bounds() != want.Bounds() {
		t.Fatal("PNG dimensions", err)
	}
	for y := want.Bounds().Min.Y; y < want.Bounds().Max.Y; y++ {
		for x := want.Bounds().Min.X; x < want.Bounds().Max.X; x++ {
			if color.RGBAModel.Convert(got.At(x, y)) != color.RGBAModel.Convert(want.At(x, y)) {
				t.Fatal("PNG changed frame pixel", x, y)
			}
		}
	}
}

func TestScreenshotPublicationPreservesExistingNames(t *testing.T) {
	store := SaveStore{Dir: t.TempDir()}
	seed := []byte("existing screenshot")
	if err := os.WriteFile(filepath.Join(store.Dir, "screen0000.png"), seed, 0600); err != nil {
		t.Fatal(err)
	}
	frame := screenshotFixture()
	for _, want := range []string{"screen0001.png", "screen0002.png"} {
		name, err := writeScreenshot(store, nil, frame)
		if err != nil || name != want {
			t.Fatal("screenshot publication", name, err)
		}
		screenshotWantPNG(t, filepath.Join(store.Dir, name), frame)
	}
	if got, err := os.ReadFile(filepath.Join(store.Dir, "screen0000.png")); err != nil || !bytes.Equal(got, seed) {
		t.Fatal("existing screenshot was replaced", err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 3 {
		t.Fatal("temporary publication file remained", entries, err)
	}
}

func TestScreenshotOutputRequiresExplicitSafeDirectory(t *testing.T) {
	install := profileInstall(t)
	for _, target := range []string{"", install, filepath.Join(install, "new-screenshots")} {
		if _, err := writeScreenshot(SaveStore{Dir: target}, []string{install}, screenshotFixture()); err == nil {
			t.Fatal("unsafe screenshot target accepted", target)
		}
	}
	if _, err := os.Stat(filepath.Join(install, "new-screenshots")); !os.IsNotExist(err) {
		t.Fatal("refused target directory was created", err)
	}
	profile, err := ResolveRuntimeProfile("", filepath.Join(install, "againrom.exe"), t.TempDir(), install)
	if err != nil {
		t.Fatal(err)
	}
	name, err := writeScreenshot(profile.Saves, []string{install}, screenshotFixture())
	if err != nil || name != "screen0000.png" {
		t.Fatal("selected private profile was refused", name, err)
	}
	screenshotWantPNG(t, filepath.Join(profile.Saves.Dir, name), screenshotFixture())
	forged := profile.Saves
	forged.Dir = install
	if _, err := writeScreenshot(forged, []string{install}, screenshotFixture()); err == nil {
		t.Fatal("profile grant escaped its selected directory")
	}
}

func TestScreenshotInvalidFrameCreatesNoOutput(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "not-created")
	if _, err := writeScreenshot(SaveStore{Dir: dir}, nil, nil); err == nil {
		t.Fatal("nil frame accepted")
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("invalid frame created output directory", err)
	}
	buf := screenshotBuffer{}
	if _, err := buf.Write(make([]byte, maxSaveBytes+1)); err == nil || buf.Len() != 0 {
		t.Fatal("PNG output exceeded bounded allocation", err)
	}
}

func TestScreenshotPublicationFailureRecoversPrivateStaging(t *testing.T) {
	files := &faultSaveOps{fail: "publish", removeFailures: 1}
	store := SaveStore{Dir: t.TempDir(), files: files}
	if _, err := writeScreenshot(store, nil, screenshotFixture()); err == nil {
		t.Fatal("publication failure was reported as success")
	}
	if _, err := os.Stat(filepath.Join(store.Dir, "screen0000.png")); !os.IsNotExist(err) {
		t.Fatal("failed publication exposed a final PNG", err)
	}
	files.fail = ""
	name, err := writeScreenshot(store, nil, screenshotFixture())
	if err != nil || name != "screen0000.png" {
		t.Fatal("retry did not recover private staging", name, err)
	}
	entries, err := os.ReadDir(store.Dir)
	if err != nil || len(entries) != 1 || entries[0].Name() != name {
		t.Fatal("failed screenshot left staging behind", entries, err)
	}
}

func TestScreenshotKeepsLaunchDirectoryAfterSaveBrowse(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	a := f.App("screenshot profile")
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	launchDir, browsedDir := t.TempDir(), t.TempDir()
	f.ConfigureSaveSeams(a, SaveStore{Dir: launchDir}, OriginalStore{}, nil)
	if err := a.HeadlessKey("f2"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveEdit(browsedDir, "Checkpoint", ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(browsedDir, "Checkpoint.sav")); err != nil {
		t.Fatal("SAVE did not change its current directory", err)
	}
	if err := a.HeadlessKey("alt-s"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessScreenshotFrame(screenshotFixture()); err != nil {
		t.Fatal(err)
	}
	screenshotWantPNG(t, filepath.Join(launchDir, "screen0000.png"), screenshotFixture())
	if entries, err := filepath.Glob(filepath.Join(browsedDir, "*.png")); err != nil || len(entries) != 0 {
		t.Fatal("screenshot followed SAVE browsing", entries, err)
	}
}

func TestScreenshotRejectedPathLogsAndConsumesCapture(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	a := f.App("screenshot refusal")
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	install := profileInstall(t)
	f.ConfigureSaveSeams(a, SaveStore{Dir: install}, OriginalStore{Dir: install}, nil)
	if err := a.HeadlessKey("alt-s"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessScreenshotFrame(screenshotFixture()); err == nil {
		t.Fatal("screenshot install fence was bypassed")
	}
	lines := f.live.view.MessageLines()
	if len(lines) == 0 || !strings.HasPrefix(lines[len(lines)-1].Text, "Screenshot failed: ") {
		t.Fatal("screenshot failure was not posted to the map log", lines)
	}
	if err := a.HeadlessScreenshotFrame(screenshotFixture()); err != nil || len(f.live.view.MessageLines()) != len(lines) {
		t.Fatal("refused capture was retained for another frame", err)
	}
	if matches, err := filepath.Glob(filepath.Join(install, "*.png")); err != nil || len(matches) != 0 {
		t.Fatal("refused screenshot wrote inside the fixture install", matches, err)
	}
}
