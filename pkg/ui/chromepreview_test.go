package ui

import (
	"bytes"
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/textfit"
)

func TestChromePreviewUsesEditorRects(t *testing.T) {
	query := "Gate guards by the southern bridge and west of the old road"
	screen := image.Rect(0, 0, 1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		for _, floating := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/floating=%v", lang, floating), func(t *testing.T) {
				state := chromePreview{Maps: []string{"South Gate", "Old Road"}, Floating: floating, Query: &query}
				desc := popupDesc{Window: "chrome", Chrome: &state}.encode()
				font := wordsFont(lang)
				m, err := NewEditorMockup(desc, lang, screen.Size(), font, font, nil)
				if err != nil {
					t.Fatal(err)
				}
				c, page := m.chrome, m.Page()
				back, err := PageDescription(page)
				if err != nil || !bytes.Equal(back, desc) {
					t.Fatalf("description round trip: %s, %v", back, err)
				}
				checkBox := func(name string, row int, r image.Rectangle) {
					t.Helper()
					if r.Empty() || !r.In(screen) {
						t.Fatalf("%s lies outside the editor at %v", name, r)
					}
					part := fmt.Sprintf(`data-part="%s"`, name)
					if row >= 0 {
						part += fmt.Sprintf(` data-row="%d"`, row)
					}
					part += fmt.Sprintf(` data-box="%d %d %d %d"`, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
					if !bytes.Contains(page, []byte(part)) {
						t.Errorf("HTML does not place %s", part)
					}
				}
				e := c.editor
				l := e.topBarLayout()
				if len(l.Bar.Cells) != len(barKeys) {
					t.Fatalf("top bar placed %d of %d entries", len(l.Bar.Cells), len(barKeys))
				}
				for i, b := range l.Bar.Cells {
					checkBox("bar-entry", i, b.Rect)
					label := b.Caption
					if b.Menu {
						label += " v"
					}
					if font.Advance(label)+8 > b.Rect.Dx() {
						t.Errorf("caption %q does not fit %v", label, b.Rect)
					}
					if part, index := l.hit(center(b.Rect)); part != popupBarCellPart || index != b.Index {
						t.Errorf("bar entry %d hits %v/%d", b.Index, part, index)
					}
				}
				tools := e.toolbarLayout()
				pictures := 0
				for _, b := range toolButtons {
					if b.onToolbar() {
						pictures++
					}
				}
				if len(tools.Bar.Cells) != pictures {
					t.Fatalf("toolbar placed %d of %d pictures", len(tools.Bar.Cells), pictures)
				}
				for _, b := range tools.Bar.Cells {
					checkBox("tool-"+toolButtons[b.Index].key, -1, b.Rect)
					if part, index := tools.hit(center(b.Rect)); part != popupBarCellPart || index != b.Index {
						t.Errorf("tool %d hits %v/%d", b.Index, part, index)
					}
					tip, ok := popupTipAt(screen, font, font, popupSpec{}, tools, center(b.Rect), true)
					if !ok || tip.Text != e.toolWord(toolButtons[b.Index]) || !tip.Rect.In(screen) {
						t.Errorf("tool %d shows tip %q at %v", b.Index, tip.Text, tip.Rect)
					}
				}
				field := c.search.Bar.Cells[0]
				checkBox("search-strip", -1, c.search.Rect)
				checkBox("search-field", -1, field.Rect)
				checkBox("tip", -1, c.tip.Rect)
				if c.search.Rect.Overlaps(e.mapTabsRect()) || c.search.Rect.Overlaps(e.toolbarRect()) {
					t.Fatalf("search strip overlaps the tabs or docked toolbar at %v", c.search.Rect)
				}
				shown, _ := editFieldText(font, *field.Edit, field.Rect)
				if !strings.HasSuffix(shown, textfit.Mark) || font.Advance(shown) > field.Rect.Dx()-2-editTextX || c.tip.Text != query {
					t.Fatalf("clipped search %q, tip %q", shown, c.tip.Text)
				}
			})
		}
	}
}

func TestMapEditorChromeBarControlsUseLayoutHits(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	e.body = wordsFont("en")
	e.Layout(1920, 1080)
	file, _ := e.controlBarRect("file")
	chromeClick(e, center(file))
	if e.menu != "file" {
		t.Fatal("press on the File head did not open its menu")
	}
	_ = e.Key("escape")
	grid, _ := e.controlToolbarRect("grid")
	on := e.view.grid
	chromeClick(e, center(grid))
	if e.view.grid == on {
		t.Fatal("press on the Grid picture did not toggle the grid")
	}
	_ = e.Key(searchKey)
	query := "Gate guards by the southern bridge and west of the old road"
	e.TextInput(query)
	if e.search.text != query {
		t.Fatalf("search accepted %q", e.search.text)
	}
	at := center(e.searchEntry().rect)
	now := time.Unix(10, 0)
	e.Pointer(Input{CursorX: at.X, CursorY: at.Y, PrimaryDown: true}, now)
	pointer := Input{CursorX: at.X, CursorY: at.Y}
	e.Pointer(pointer, now)
	if !e.searchTyping() || e.input.model.Caret >= len([]rune(query)) {
		t.Fatal("press on the search field did not position its caret")
	}
	e.Pointer(pointer, now.Add(time.Duration(editorTipDelay-1)*time.Millisecond))
	if e.TipShown() != "" {
		t.Fatal("search tip appeared before the hover delay")
	}
	e.Pointer(pointer, now.Add(time.Duration(editorTipDelay)*time.Millisecond))
	if got := e.TipShown(); got != query {
		t.Fatalf("settled search shows %q, want the whole query", got)
	}
}
