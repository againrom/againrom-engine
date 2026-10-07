package game

import (
	"bytes"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"againrom/pkg/formats/sav"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textsmooth"
	"againrom/pkg/ui"
)

func TestReleaseSaveAcknowledgementIsReadableAndCaptured(t *testing.T) {
	for _, context := range []string{"town", "mission"} {
		t.Run(context, func(t *testing.T) {
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			app := f.App("SAVE acknowledgement")
			t.Cleanup(app.StopAudio)
			app.Layout(1280, 960)
			out := t.TempDir()
			store := SaveStore{Dir: out}
			if context == "town" {
				path := os.Getenv("AGAINROM_CITY_SAV")
				if path == "" {
					t.Skip("AGAINROM_CITY_SAV is not set: the town needs an existing SAV")
				}
				raw, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				input := t.TempDir()
				if err := os.WriteFile(filepath.Join(input, "game0000.sav"), raw, 0600); err != nil {
					t.Fatal(err)
				}
				app.SetSaveSeams(f.SaveSeams(store, OriginalStore{Dir: input}, nil))
				if err := app.HeadlessKey("load"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessActivate("@first"); err != nil || app.Screen() != ui.ScreenTown {
					t.Fatal("App LOAD did not reach the existing town", err, app.Screen())
				}
			} else if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
			f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
			for i := 0; app.HeadlessNoticeOpen() && i < 32; i++ {
				if err := app.HeadlessActivate("notice"); err != nil {
					t.Fatal(err)
				}
			}
			if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenGameMenu {
				t.Fatal("Esc did not open the game menu", err, app.Screen())
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil || app.Screen() != ui.ScreenSave {
				t.Fatal("SAVE did not open its ordinary dialog", err, app.Screen())
			}
			if err := app.HeadlessSaveEdit(out, "Readable checkpoint", ui.SaveSAV); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessSaveAction("save"); err != nil || app.Screen() != ui.ScreenGameMenu {
				t.Fatal("SAVE did not return to the menu", err, app.Screen(), app.HeadlessMessage())
			}
			if app.HeadlessMessage() != f.Words.SaveAcknowledgement {
				t.Fatalf("acknowledgement bytes=% x want installed % x", []byte(app.HeadlessMessage()), []byte(f.Words.SaveAcknowledgement))
			}
			raw, err := os.ReadFile(filepath.Join(out, "Readable checkpoint.sav"))
			if err != nil || !bytes.HasPrefix(raw, []byte("Asg&")) {
				t.Fatal("ordinary SAVE produced no SAV", err)
			}
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil || (doc.World != nil) != (context == "mission") {
				t.Fatal("SAV changed the saved context", err)
			}
			status := image.Rect(8, 452, 632, 468)
			panel := app.GameMenuPanel()
			want := image.NewRGBA(panel.Bounds())
			text.ResetCapture()
			f.Font.Value().Draw(want.SubImage(status).(*image.RGBA), f.Words.SaveAcknowledgement, 8, 452, color.RGBA{255, 255, 255, 255})
			ink := 0
			for y := status.Min.Y; y < status.Max.Y; y++ {
				for x := status.Min.X; x < status.Max.X; x++ {
					if panel.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatal("installed acknowledgement does not match its font raster", x, y)
					}
					if panel.RGBAAt(x, y).A != 0 {
						ink++
					}
				}
			}
			if ink == 0 {
				t.Fatal("SAVE acknowledgement has no readable pixels")
			}
			app.SetTextSmoothing(true)
			canvas := ebiten.NewImage(1280, 960)
			defer canvas.Dispose()
			for range 2 {
				canvas.Clear()
				app.Draw(canvas)
				seen := 0
				for _, fate := range app.TextFates() {
					if fate.Call.Y != 452 {
						continue
					}
					if fate.Fate != textsmooth.Kept && fate.Fate != textsmooth.Blank {
						t.Fatalf("SAVE glyph has fate %s, want kept or blank", fate.Fate)
					}
					seen++
				}
				if seen != len(f.Words.SaveAcknowledgement) {
					t.Fatalf("App captured %d SAVE glyphs, want %d installed bytes", seen, len(f.Words.SaveAcknowledgement))
				}
			}
			if app.HeadlessMessage() != f.Words.SaveAcknowledgement {
				t.Fatal("idle drawing changed the acknowledgement")
			}
			t.Logf("%s: SAV bytes=%d, acknowledgement bytes=%d, ink pixels=%d, two App Draw captures", context, len(raw), len(f.Words.SaveAcknowledgement), ink)
		})
	}
}
