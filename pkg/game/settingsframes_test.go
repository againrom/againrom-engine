package game

import (
	"bytes"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/ui"
)

// The settings pages over a mission at the small window and the large. With
// AGAINROM_SETTINGS_FRAMES naming a folder and AGAINROM_ASSETS an install, the
// frames are written there; with AGAINROM_SETTINGS_BEFORE naming a folder of
// earlier frames, each frame's PNG must equal its earlier one byte for byte.
func TestSettingsPagesFramesInEachLanguage(t *testing.T) {
	dir, before := os.Getenv("AGAINROM_SETTINGS_FRAMES"), os.Getenv("AGAINROM_SETTINGS_BEFORE")
	if dir == "" || os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("needs AGAINROM_SETTINGS_FRAMES and AGAINROM_ASSETS")
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	game := filepath.Base(os.Getenv("AGAINROM_ASSETS"))
	party := f.ChargenParty(ui.ChargenResult{Name: "Settings", Choices: []int{1, 0, 3}, Stats: []int{31, 27, 24, 29}})
	a := f.App("settings frames")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(20, party)); err != nil {
		t.Fatal(err)
	}
	a.Layout(640, 480)
	openMissionGameMenu(t, a)
	action := func(names ...string) func() error {
		return func() error {
			for _, n := range names {
				if err := a.HeadlessGameMenuAction(n); err != nil {
					return fmt.Errorf("%s: %w", n, err)
				}
			}
			return nil
		}
	}
	// keys-moved takes the patrol's key for the attack, then waits on the
	// attack's row for its next key.
	moved := func() error {
		if err := action("keys")(); err != nil {
			return err
		}
		if err := a.HeadlessRebindKey("attack", "P"); err != nil {
			return err
		}
		return action("key-list")()
	}
	unmove := func() error {
		if err := a.HeadlessKey("escape"); err != nil {
			return err
		}
		return action("key-defaults", "page-return")()
	}
	// keys-wait-* waits on the longest action name of each language.
	waiting := func(name string) func() error {
		return func() error {
			if err := action("keys")(); err != nil {
				return err
			}
			return a.HeadlessWaitKey(name)
		}
	}
	unwait := func() error {
		if err := a.HeadlessKey("escape"); err != nil {
			return err
		}
		return action("page-return")()
	}
	pages := []struct {
		name        string
		open, leave func() error
	}{
		{"game-options", action("game-options"), action("options-cancel")},
		{"sound-options", action("sound-options"), action("page-return")},
		{"menu", action(), action()},
		{"keys", action("keys"), action("page-return")},
		{"keys-moved", moved, unmove},
		{"keys-wait-pause-text", waiting("pause-text"), unwait},
		{"keys-wait-unpaced", waiting("unpaced"), unwait},
	}
	for _, size := range []image.Point{{X: 640, Y: 480}, {X: 1280, Y: 800}} {
		a.LayoutF(float64(size.X), float64(size.Y))
		for _, p := range pages {
			if err := p.open(); err != nil {
				t.Fatal(p.name, err)
			}
			for _, part := range []struct {
				name string
				req  ui.HeadlessRequest
			}{{"window", ui.HeadlessRequest{Part: ui.HeadlessPartFrame, Scaled: true}}, {"panel", ui.HeadlessRequest{Part: ui.HeadlessPartGameMenuPanel}}} {
				pix, _, err := a.HeadlessPicture(part.req)
				if err != nil {
					t.Fatal(err)
				}
				name := fmt.Sprintf("%s-%s-%dx%d-%s.png", p.name, game, size.X, size.Y, part.name)
				b := witness.Bytes(t, pix)
				if err := os.WriteFile(filepath.Join(dir, name), b, 0o644); err != nil {
					t.Fatal(err)
				}
				if before != "" {
					earlier, err := os.ReadFile(filepath.Join(before, name))
					if err == nil && !bytes.Equal(earlier, b) {
						t.Error(name, "differs from its earlier frame")
					}
				}
			}
			if err := p.leave(); err != nil {
				t.Fatal(p.name, err)
			}
			openMissionGameMenu(t, a)
		}
	}
}
