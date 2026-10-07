package ui

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

func unicodeLoadApp(t *testing.T, font *text.Font, loaded *string) *App {
	t.Helper()
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), font, func(r rune) (byte, bool) {
		return textinput.EncodeRune(r, font.Selector)
	})
	a.SetSaveSeams(nil, func() []SaveEntry {
		return []SaveEntry{{Name: "Тест.ags", Label: "Тест.ags", Note: "Сохранение"}}
	}, func(name string) (MapOpener, bool, error) {
		*loaded = name
		return nil, false, errors.New("fixture stops after exact LOAD token")
	})
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenLoad {
		t.Fatalf("LOAD opened %s", a.Screen())
	}
	return a
}

func literalUnicodeLoadRow(font *text.Font) string {
	// The RU row uses the existing input codec before the renderer's selector
	// conversion. EN preserves its unsupported-character fallback.
	name := []byte{'?', '?', '?', '?'}
	if font.Selector == text.SelectorConverting {
		name = []byte{0x92, 0xa5, 0xe1, 0xe2}
	}
	return string(name) + ".ags"
}

func assertUnicodeLoadRow(t *testing.T, font *text.Font, pix *image.RGBA) {
	t.Helper()
	expect := image.NewRGBA(pix.Bounds())
	draw.Draw(expect, expect.Bounds(), pix, pix.Bounds().Min, draw.Src)
	// Compare the glyph-bearing row against independently encoded bytes.
	r := image.Rect(122, 152, 502, 171)
	background := image.NewRGBA(pix.Bounds())
	drawTownShellBox(background, loadPanel, false)
	drawMovieBox(background, r, true)
	draw.Draw(expect, r, background, r.Min, draw.Src)
	font.Draw(expect.SubImage(r).(*image.RGBA), literalUnicodeLoadRow(font), 125, 154, loadSelectedText)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			if pix.RGBAAt(x, y) != expect.RGBAAt(x, y) {
				t.Fatalf("Unicode LOAD glyph differs at %d,%d", x, y)
			}
		}
	}

}

func TestLoadDrawsUnicodeInsideFramedListAndKeepsExactLoadToken(t *testing.T) {
	for _, selector := range []int{0, text.SelectorConverting} {
		font := chargenTestFont()
		font.Selector = selector
		for i := range font.Glyphs {
			for j := range font.Glyphs[i].Pixels {
				font.Glyphs[i].Pixels[j].Level = uint8(1 + i%15)
			}
		}
		loaded := ""
		a := unicodeLoadApp(t, font, &loaded)
		pix, err := a.composeLoadScreen()
		if err != nil {
			t.Fatal(err)
		}
		assertUnicodeLoadRow(t, font, pix)
		if got := a.HeadlessRows(); len(got) != 1 || got[0].Text != "Тест.ags" {
			t.Fatalf("LOAD model lost UTF-8 label: %v", got)
		}
		if err := a.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
		if loaded != "Тест.ags" {
			t.Fatalf("LOAD changed disk token to %q", loaded)
		}
	}
}

func TestComposeScreenSelectsLoadComposer(t *testing.T) {
	loaded := ""
	a := unicodeLoadApp(t, chargenTestFont(), &loaded)
	pix, err := a.composeScreen()
	if err != nil {
		t.Fatal(err)
	}
	want, err := a.composeLoadScreen()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(pix.Pix, want.Pix) {
		t.Fatal("LOAD dispatch selected another composer")
	}
	if _, note, err := a.HeadlessFrame(); err != nil || note != "" {
		t.Fatalf("headless LOAD: %q / %v", note, err)
	}
}

func TestLoadTextClipsAtMeasuredPixelsAndFailureWinsNote(t *testing.T) {
	loaded := ""
	a := unicodeLoadApp(t, chargenTestFont(), &loaded)
	label := strings.Repeat("Тест ", 60)
	clipped := a.fitLoadText(label, 90)
	width, _ := a.flow.menuFont.Measure(a.flow.menuDisplayText(clipped))
	if width > 90 || !strings.HasSuffix(clipped, "...") {
		t.Fatalf("pixel fit = %d %q", width, clipped)
	}
	if a.loadMessage() != "Сохранение" {
		t.Fatal("LOAD lost selected note")
	}
	a.flow.msg = "read failed"
	if a.loadMessage() != "read failed" {
		t.Fatal("selected note hid failure")
	}
}

func loadScrollTestFrames() []*image.RGBA {
	frames := make([]*image.RGBA, 26)
	for _, index := range []int{16, 18, 19, 20} {
		pic := image.NewRGBA(image.Rect(0, 0, 24, 24))
		for y := 0; y < 24; y++ {
			for x := 0; x < 24; x++ {
				pic.SetRGBA(x, y, color.RGBA{uint8(30 + index*7), uint8(20 + x*5), uint8(30 + y*7), 255})
			}
		}
		frames[index] = pic
	}
	return frames
}

func loadScrollTestApp(t *testing.T, frames []*image.RGBA, loaded *[]string) *App {
	t.Helper()
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), chargenTestFont(), nil)
	rows := make([]SaveEntry, 27)
	for i := range rows {
		rows[i] = SaveEntry{Name: fmt.Sprintf("slot-%02d.sav", i), Label: fmt.Sprintf("row-%02d", i), Note: fmt.Sprintf("note-%02d", i)}
	}
	a.SetSaveSeams(nil, func() []SaveEntry { return rows }, func(name string) (MapOpener, bool, error) {
		*loaded = append(*loaded, name)
		return nil, false, errors.New("synthetic LOAD refusal")
	})
	a.SetCutsceneScrollArt(frames)
	if err := a.HeadlessKey("load"); err != nil {
		t.Fatal(err)
	}
	if a.Screen() != ScreenLoad {
		t.Fatalf("LOAD opened %s", a.Screen())
	}
	return a
}

func loadScrollTestFrame(t *testing.T, a *App) *image.RGBA {
	t.Helper()
	pix, note, err := a.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatalf("LOAD frame: %q / %v", note, err)
	}
	if pix.Bounds() != image.Rect(0, 0, 640, 480) {
		t.Fatalf("LOAD frame bounds = %v", pix.Bounds())
	}
	return pix
}

func assertLoadScrollSourcePixels(t *testing.T, pix, fallback *image.RGBA, frames []*image.RGBA, thumbY int) {
	t.Helper()
	strip := image.Rect(504, 152, 528, 342)
	counts := map[int]int{}
	for y := 0; y < 480; y++ {
		for x := 0; x < 640; x++ {
			if !image.Pt(x, y).In(strip) {
				if pix.RGBAAt(x, y) != fallback.RGBAAt(x, y) {
					t.Fatalf("LOAD skin changed pixels outside strip at %d,%d", x, y)
				}
				continue
			}
			index, sy := 19, (y-176)%24
			switch {
			case y < 176:
				index, sy = 18, y-152
			case y >= 318:
				index, sy = 20, y-318
			case y >= thumbY && y < thumbY+24:
				index, sy = 16, y-thumbY
			}
			want := frames[index].RGBAAt(x-504, sy)
			if got := pix.RGBAAt(x, y); got != want {
				t.Fatalf("LOAD frame %d source %d,%d at %d,%d = %v, want %v", index, x-504, sy, x, y, got, want)
			}
			counts[index]++
		}
	}
	for _, index := range []int{16, 18, 19, 20} {
		want := 576
		if index == 19 {
			want = 2832
		}
		if counts[index] != want {
			t.Fatalf("LOAD frame %d checked %d pixels, want %d", index, counts[index], want)
		}
	}
}

func TestFramedLoadScrollSkinKeepsLiteralGeometryAndSelection(t *testing.T) {
	frames := loadScrollTestFrames()
	var loaded []string
	a := loadScrollTestApp(t, nil, &loaded)
	fallback := loadScrollTestFrame(t, a)
	a.SetCutsceneScrollArt(frames)
	initial := loadScrollTestFrame(t, a)
	assertLoadScrollSourcePixels(t, initial, fallback, frames, 176)
	for i := 0; i < 13; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	if top, count := a.flow.loadList.Visible(); top != 4 || count != 10 || a.flow.loadList.Selection() != 13 {
		t.Fatalf("LOAD selection/window = %d/%d/%d", a.flow.loadList.Selection(), top, count)
	}
	a.SetCutsceneScrollArt(nil)
	fallback = loadScrollTestFrame(t, a)
	a.SetCutsceneScrollArt(frames)
	moved := loadScrollTestFrame(t, a)
	assertLoadScrollSourcePixels(t, moved, fallback, frames, 235)
	if got, want := moved.RGBAAt(515, 184), frames[19].RGBAAt(11, 8); got != want || got == initial.RGBAAt(515, 184) {
		t.Fatalf("old LOAD thumb did not restore track: %v, want %v", got, want)
	}
	rows := a.HeadlessRows()
	if len(rows) != 27 || rows[13].Text != "row-13" || !rows[13].Choosable || a.loadMessage() != "note-13" {
		t.Fatalf("LOAD row/note changed: %v / %q", rows, a.loadMessage())
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0] != "slot-13.sav" {
		t.Fatalf("LOAD changed selected disk token: %v", loaded)
	}
}

func TestFramedLoadScrollSkinNeedsOnlyUsableRequiredFrames(t *testing.T) {
	for _, name := range []string{"21 frames", "unrelated nil"} {
		t.Run(name, func(t *testing.T) {
			frames := loadScrollTestFrames()
			if name == "21 frames" {
				frames = frames[:21]
			}
			var loaded []string
			a := loadScrollTestApp(t, nil, &loaded)
			fallback := loadScrollTestFrame(t, a)
			a.SetCutsceneScrollArt(frames)
			assertLoadScrollSourcePixels(t, loadScrollTestFrame(t, a), fallback, frames, 176)
		})
	}
	for _, index := range []int{16, 18, 19, 20} {
		for _, damage := range []string{"missing", "nil", "empty"} {
			t.Run(fmt.Sprintf("frame-%d-%s", index, damage), func(t *testing.T) {
				frames := loadScrollTestFrames()
				switch damage {
				case "missing":
					frames = frames[:index]
				case "nil":
					frames[index] = nil
				case "empty":
					frames[index] = image.NewRGBA(image.Rectangle{})
				}
				var loaded []string
				a := loadScrollTestApp(t, nil, &loaded)
				fallback := loadScrollTestFrame(t, a)
				a.SetCutsceneScrollArt(frames)
				got := loadScrollTestFrame(t, a)
				if !bytes.Equal(got.Pix, fallback.Pix) {
					t.Fatalf("unusable LOAD frame %d (%s) did not retain complete fallback", index, damage)
				}
			})
		}
	}
}

func TestFramedLoadScrollSkinKeepsExactDoubleClickAndReset(t *testing.T) {
	var loaded []string
	a := loadScrollTestApp(t, loadScrollTestFrames(), &loaded)
	for i := 0; i < 13; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	now := time.Unix(600, 0)
	click := func() {
		a.step(appInput{CursorX: 140, CursorY: 329, PrimaryPressed: true}, now)
		a.step(appInput{CursorX: 140, CursorY: 329, PrimaryReleased: true}, now.Add(time.Millisecond))
	}
	click()
	if len(loaded) != 0 || a.flow.loadList.Selection() != 13 {
		t.Fatalf("first LOAD click changed activation/selection: %v / %d", loaded, a.flow.loadList.Selection())
	}
	lastName, lastClick := a.flow.loadUI.lastName, a.flow.loadUI.lastClick
	loadScrollTestFrame(t, a)
	if a.flow.loadUI.lastName != lastName || a.flow.loadUI.lastClick != lastClick {
		t.Fatal("LOAD skin paint changed double-click state")
	}
	now = now.Add(200 * time.Millisecond)
	click()
	if len(loaded) != 1 || loaded[0] != "slot-13.sav" || a.Screen() != ScreenLoad || a.HeadlessMessage() == "" {
		t.Fatalf("LOAD double click changed token/refusal: %v / %s / %q", loaded, a.Screen(), a.HeadlessMessage())
	}
	if a.flow.loadUI.lastName != "" || !a.flow.loadUI.lastClick.IsZero() {
		t.Fatal("refused LOAD kept stale double-click state")
	}
	now = now.Add(50 * time.Millisecond)
	click()
	if len(loaded) != 1 {
		t.Fatalf("stale LOAD click activated again: %v", loaded)
	}
}
