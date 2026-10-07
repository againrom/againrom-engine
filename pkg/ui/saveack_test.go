package ui

import (
	"bytes"
	"image"
	"image/color"
	"testing"

	"againrom/pkg/render/text"
)

func TestGameMenuCapturesTheInstalledRussianSaveAcknowledgement(t *testing.T) {
	t.Cleanup(text.ResetCapture)
	font := &text.Font{Selector: text.SelectorConverting, Glyphs: make([]text.Glyph, 224)}
	for i := range font.Glyphs {
		font.Glyphs[i] = text.Glyph{Width: 2, Height: 2, Advance: 2,
			Pixels: []text.Pixel{{Level: 15, Painted: true}, {}, {}, {Level: 7, Painted: true}}}
	}
	for _, back := range []Screen{ScreenTown, ScreenMap} {
		t.Run(back.String(), func(t *testing.T) {
			a := newTestApp(t, appRows(0), okLoader(t))
			words := AuthoredWords()
			words.SaveAcknowledgement = string([]byte{0x82, 0xa0, 0xe8, ' ', 0xaf, 0xa5, 0xe0, 0xe1, 0xae, 0xad, 0xa0, 0xa6, ' ', 0xe1, 0xae, 0xe5, 0xe0, 0xa0, 0xad, 0xa5, 0xad})
			a.SetWords(words, font, nil)
			a.flow.openGameMenu(back)
			a.flow.msg = words.SaveAcknowledgement
			text.ResetCapture()
			text.SetCapture(false)
			defer text.StopCapture()
			pic := a.GameMenuPanel()
			var calls []text.DrawCall
			for _, call := range text.Captured() {
				if call.Y == pickerMessageY {
					calls = append(calls, call)
				}
			}
			if len(calls) != len(words.SaveAcknowledgement) {
				t.Fatalf("captured SAVE status glyphs=%d, want %d installed bytes", len(calls), len(words.SaveAcknowledgement))
			}
			want := image.NewRGBA(pic.Bounds())
			font.Draw(want, words.SaveAcknowledgement, pickerLeft, pickerMessageY, color.RGBA{255, 255, 255, 255})
			status := image.Rect(pickerLeft, pickerMessageY, pickerLeft+pickerCols*pickerAdvance, pickerMessageY+pickerLine)
			for y := status.Min.Y; y < status.Max.Y; y++ {
				for x := status.Min.X; x < status.Max.X; x++ {
					if pic.RGBAAt(x, y) != want.RGBAAt(x, y) {
						t.Fatal("installed status does not match the loaded Russian glyph raster", x, y)
					}
				}
			}
			for i, call := range calls {
				if call.Glyph != font.GlyphFor(words.SaveAcknowledgement[i]) || call.Clip != status {
					t.Fatalf("glyph %d was converted or clipped differently: %+v", i, call)
				}
			}
			saved := append([]byte(nil), []byte(a.HeadlessMessage())...)
			a.flow.msg = "save failed: cannot create checkpoint.sav"
			text.ResetCapture()
			text.SetCapture(false)
			failed := a.GameMenuPanel()
			for _, call := range text.Captured() {
				if call.Y == pickerMessageY {
					t.Fatal("UTF-8 diagnostic error was treated as installed text")
				}
			}
			if a.HeadlessMessage() != "save failed: cannot create checkpoint.sav" || !bytes.Equal(saved, []byte(words.SaveAcknowledgement)) {
				t.Fatal("rendering changed the status encoding")
			}
			if failed.RGBAAt(pickerLeft, pickerMessageY).A != 0 {
				t.Fatal("diagnostic fallback was replaced by an installed glyph")
			}
		})
	}
}

func TestTownUnderTheGameMenuDrawsNoSecondSaveLine(t *testing.T) {
	a := newTestApp(t, appRows(0), okLoader(t))
	a.flow.msg = AuthoredWords().SaveAcknowledgement
	a.flow.screen = ScreenTown
	if got := a.townRoomMessage(); got != a.flow.msg {
		t.Fatalf("the town itself shows %q, want %q", got, a.flow.msg)
	}
	a.flow.screen = ScreenGameMenu
	if got := a.townRoomMessage(); got != "" {
		t.Fatalf("the town under the menu also draws %q", got)
	}
}
