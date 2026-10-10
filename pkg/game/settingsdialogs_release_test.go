package game

import (
	"image"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/vfs"
)

// The Game Options and Sound Options dialogs read these installed rows
// (TEXT-099, TEXT-100); the cutscene and tune tables hold their counts.
func TestReleaseSettingsDialogsReadInstalledRows(t *testing.T) {
	f := releaseFront(t)
	dialogs := LoadTextTable(f.Archives.Containers, DialogsTextPath, f.textCode())
	if dialogs.Lines() != 166 {
		t.Fatalf("dialogs.txt has %d rows, want 166", dialogs.Lines())
	}
	game := []int{0, 1, 78, 150, 156, 164}
	for i := 50; i <= 67; i++ {
		game = append(game, i)
	}
	game = append(game, 159, 160, 161, 162)
	sound := []int{0, 23, 75, 143, 165}
	for i := 7; i <= 21; i++ {
		sound = append(sound, i)
	}
	if len(game) != 28 || len(sound) != 20 {
		t.Fatalf("row lists hold %d and %d rows, want 28 and 20", len(game), len(sound))
	}
	for _, i := range append(game, sound...) {
		if _, ok := dialogs.At(i); !ok {
			t.Errorf("dialogs.txt row %d is absent", i)
		}
	}
	patch, err := vfs.Open([]string{filepath.Join(f.Archives.Root, patchArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	table := LoadTextTable(patch, patchTextPath, f.textCode())
	if table.Lines() != 67 {
		t.Fatalf("patch.txt has %d rows, want 67", table.Lines())
	}
	for i := patchGraphicsRow; i < patchGraphicsRow+3; i++ {
		if _, ok := table.At(i); !ok {
			t.Errorf("patch.txt row %d is absent", i)
		}
	}
	tunes := musicTitles(LoadTextTable(f.Archives.Containers, mainPrefix+"text/tunes.txt", f.textCode()))
	if len(tunes) != 21 {
		t.Fatalf("tunes.txt names %d tracks, want 21", len(tunes))
	}
	for key := range tunes {
		if key != strings.ToLower(key) || !strings.HasSuffix(key, ".wav") {
			t.Errorf("tune key %q is not a lowercase .wav name", key)
		}
	}
	titles := LoadTextTable(f.Archives.Containers, mainPrefix+"text/cutscene.txt", f.textCode())
	paths := LoadTextTable(f.Archives.Containers, mainPrefix+"text/cutpaths.txt", f.textCode())
	for i := 0; i < 14; i++ {
		if _, ok := titles.At(i); !ok {
			t.Errorf("cutscene.txt row %d is absent", i)
		}
		if _, ok := paths.At(i); !ok {
			t.Errorf("cutpaths.txt row %d is absent", i)
		}
	}
	if _, ok := titles.At(14); ok {
		t.Error("cutscene.txt has a fifteenth row")
	}
}

// opaqueBox is the bounding box of every pixel the panel paints.
func opaqueBox(img *image.RGBA) image.Rectangle {
	box := image.Rectangle{}
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.RGBAAt(x, y).A != 0 {
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	return box
}

// The two dialogs share the snapped frame (MENU-077), take the install's own
// captions for Shadows, Dynamic lighting and Object animations (TEXT-099) and,
// in a mission, list the 12 mission tracks with installed titles (VIDEO-076).
func TestReleaseSettingsDialogsFrameCaptionsAndMissionTracks(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	f.LoadOptions()
	a := f.App("settings dialogs")
	defer a.StopAudio()
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpener(20)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	openMissionGameMenu(t, a)
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if box := opaqueBox(a.GameMenuPanel()); box != image.Rect(76, 28, 564, 452) {
		t.Fatalf("Game Options paints %v, want the 488x424 frame at (76,28)", box)
	}
	patch, err := vfs.Open([]string{filepath.Join(f.Archives.Root, patchArchive)}, nil)
	if err != nil {
		t.Fatal(err)
	}
	table := LoadTextTable(patch, patchTextPath, f.textCode())
	rows := a.HeadlessRows()
	for i, row := range []int{5, 6, 7} {
		word, _ := table.At(patchGraphicsRow + i)
		if !strings.HasPrefix(rows[row].Text, drawnMenuLabel(word)) {
			t.Errorf("graphics row %d reads %q, want the installed %q", row, rows[row].Text, drawnMenuLabel(word))
		}
	}
	if err := a.HeadlessGameMenuAction("options-cancel"); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		t.Fatal(err)
	}
	if box := opaqueBox(a.GameMenuPanel()); box != image.Rect(76, 60, 564, 420) {
		t.Fatalf("Sound Options paints %v, want the 488x360 frame at (76,60)", box)
	}
	candidates, _ := a.MusicPlaybackState()
	titles := musicTitles(LoadTextTable(f.Archives.Containers, mainPrefix+"text/tunes.txt", f.textCode()))
	if len(candidates) != 12 || candidates[0] != "B00.wav" || candidates[11] != "B11.wav" {
		t.Fatalf("a mission lists %v, want B00..B11", candidates)
	}
	for _, name := range candidates {
		if titles[strings.ToLower(name)] == "" {
			t.Errorf("track %s has no installed title", name)
		}
	}
}
