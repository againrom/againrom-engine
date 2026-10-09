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

func assertUnicodeLoadRow(t *testing.T, a *App, font *text.Font, pix *image.RGBA) {
	t.Helper()
	expect := image.NewRGBA(pix.Bounds())
	draw.Draw(expect, expect.Bounds(), pix, pix.Bounds().Min, draw.Src)
	// Compare the glyph-bearing row against independently encoded bytes: the
	// shared list's first row, pitch font height plus 4, selected.
	r := image.Rect(122, 152, 504, 152+font.Height()+4)
	if got := a.loadListBox().Row(0); got != r {
		t.Fatalf("LOAD row 0 = %v, want %v", got, r)
	}
	background := image.NewRGBA(pix.Bounds())
	drawTownShellBox(background, loadPanel, false)
	draw.Draw(background, r, &image.Uniform{C: color.RGBA{0, 7, 6, 220}}, image.Point{}, draw.Over)
	outline(background, r, color.RGBA{57, 77, 65, 255})
	draw.Draw(expect, r, background, r.Min, draw.Src)
	font.Draw(expect.SubImage(r.Inset(1)).(*image.RGBA), literalUnicodeLoadRow(font), 125, 154, loadSelectedText)
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
		assertUnicodeLoadRow(t, a, font, pix)
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

func loadScrollTestFrames() []*image.RGBA { return widgetTestFrames() }

func loadScrollTestApp(t *testing.T, frames []*image.RGBA, loaded *[]string) *App {
	t.Helper()
	a := newTestApp(t, nil, nil)
	a.SetWords(AuthoredWords(), solidFont15(), nil)
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

// TestLoadDrawsThroughTheSharedKit: the Load window draws its list, bar and
// three buttons with the shared builders, the bar bound to the selection
// over the save count (MENU-120), Delete disabled while it cannot act.
func TestLoadDrawsThroughTheSharedKit(t *testing.T) {
	var loaded []string
	a := loadScrollTestApp(t, loadScrollTestFrames(), &loaded)
	for i := 0; i < 13; i++ {
		if err := a.HeadlessKey("down"); err != nil {
			t.Fatal(err)
		}
	}
	calls := recordWidgets(t, func() { loadScrollTestFrame(t, a) })
	var bars, lists, buttons int
	for _, c := range calls {
		switch c.kind {
		case widgetListBox:
			lists++
			if c.rect != image.Rect(122, 152, 504, 344) {
				t.Errorf("list at %v", c.rect)
			}
		case widgetVScrollBar:
			bars++
			b := c.state.(vScrollBar)
			if c.rect != image.Rect(504, 152, 528, 344) || b.Pos != 13 || b.Count != 27 {
				t.Errorf("bar %v at %d of %d", c.rect, b.Pos, b.Count)
			}
		case widgetPushButton:
			b := c.state.(pushButton)
			if c.rect != loadButtonRect(buttons) || b.Disabled != (buttons == loadDeleteButton) {
				t.Errorf("button %d at %v disabled %t", buttons, c.rect, b.Disabled)
			}
			buttons++
		}
	}
	if lists != 1 || bars != 1 || buttons != 3 {
		t.Fatalf("Load drew %d lists, %d bars, %d buttons", lists, bars, buttons)
	}
	if err := a.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0] != "slot-13.sav" {
		t.Fatalf("LOAD changed selected disk token: %v", loaded)
	}
}

// TestLoadBarNeedsEveryClaimedFrame: a scroll bank missing any frame the
// painter selects draws the plain fallback bar, never a partial skin.
func TestLoadBarNeedsEveryClaimedFrame(t *testing.T) {
	for _, index := range []int{0, 7, 10, 18, 19, 20, 21, 22, 23} {
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
				if got := loadScrollTestFrame(t, a); !bytes.Equal(got.Pix, fallback.Pix) {
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
