package game

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/ui"
)

func TestMissionFrameInstalledHeights(t *testing.T) {
	dir, before := os.Getenv("AGAINROM_MISSION_FRAMES"), os.Getenv("AGAINROM_MISSION_BEFORE")
	if dir == "" || os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("needs AGAINROM_MISSION_FRAMES and AGAINROM_ASSETS")
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(dir, "options.txt")}
	f.Carried = MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	a := f.App("mission frame")
	a.SetCutscenes(nil)
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		window image.Point
		height int
	}{
		{image.Pt(1024, 768), 768},
		{image.Pt(1920, 1200), 768},
		{image.Pt(1280, 720), 720},
		{image.Pt(1920, 1080), 720},
		{image.Pt(2560, 1440), 720},
	} {
		a.Layout(c.window.X, c.window.Y)
		pix, note, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartFrame, Scaled: true})
		if err != nil {
			t.Fatal(err)
		}
		if pix.Bounds().Size() != c.window {
			t.Fatalf("frame %v, want window %v", pix.Bounds(), c.window)
		}
		png := witness.Bytes(t, pix)
		var saved []byte
		saveErr := f.captureSave(captureTimer, true, func(s Snapshot, _ string) error {
			var err error
			saved, err = f.ExportCurrentSave(s, "frame")
			return err
		})
		if saveErr != nil && !errors.Is(saveErr, errSavingUnavailable) {
			t.Fatal(saveErr)
		}
		saveProof := fmt.Sprintf("%x", sha256.Sum256(saved))
		if saveErr != nil {
			saveProof = saveErr.Error()
		}
		name := fmt.Sprintf("%s-%dx%d", filepath.Base(f.Archives.Root), c.window.X, c.window.Y)
		receipt := []byte(fmt.Sprintf("world %016x\nsav %s\nrgba %x\npng %x\n", f.live.world.Hash(), saveProof, sha256.Sum256(pix.Pix), sha256.Sum256(png)))
		for _, file := range []struct {
			suffix string
			raw    []byte
		}{{".png", png}, {".txt", receipt}} {
			if err := os.WriteFile(filepath.Join(dir, name+file.suffix), file.raw, 0o644); err != nil {
				t.Fatal(err)
			}
			if before != "" && c.height == 768 {
				old, err := os.ReadFile(filepath.Join(before, name+file.suffix))
				if err != nil || !bytes.Equal(old, file.raw) {
					t.Errorf("%s%s changed at retained height 768: %v", name, file.suffix, err)
				}
			}
		}
		t.Logf("%s: frame %v scale %v; %s; %s", name, f.live.view.FrameSize(), f.live.view.Placement().Scale(), note.Note, receipt)
		if got := f.live.view.FrameSize().Y; got != c.height {
			t.Errorf("window %v: mission height %d, want %d", c.window, got, c.height)
		}
		if got := f.live.view.ViewportSize().Y / 32; got != c.height/32 {
			t.Errorf("window %v: map rows %d, want %d", c.window, got, c.height/32)
		}
	}
}
