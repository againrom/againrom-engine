package ui

import (
	"image"
	"strings"
	"testing"
	"time"
)

func TestMapTabUsesNaturalWidthWhenBothFit(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, lang)
			e.font = wordsFont(lang)
			e.doc.Name = strings.Repeat("A", 106)
			e.openMaps = []*openMap{{doc: e.doc}, {doc: &InspectionDocument{Name: "Gate"}}}
			run := e.mapTabsRect().Inset(mapTabPad)
			long := e.font.Advance(e.doc.Name) + 2*popupPad + run.Dy()
			total := long + e.font.Advance("Gate") + 2*popupPad + run.Dy()
			if long <= run.Dx()/2 || total > run.Dx() {
				t.Fatalf("fixture needs %d + %d px in %d px", long, total-long, run.Dx())
			}
			_, tabs := e.mapTabStrip()
			if len(tabs.rects) != 2 || !tabs.page[0].Empty() || tabs.captions[0] != e.doc.Name || tabs.captions[1] != "Gate" {
				t.Fatalf("two natural-width tabs need %d of %d px, got %q and paging %v", total, run.Dx(), tabs.captions, tabs.page)
			}
		})
	}
}

func TestSearchResultClippedByHeightShowsFullTip(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	f := wordsFont("en")
	e.SetBodyFont(f)
	label := strings.Repeat("Whitewall ", 2000) + "by the bridge"
	e.doc.Records = []InspectionRecord{{Kind: KindStructure, Label: label}}
	_ = e.Key("search")
	e.TextInput("Whitewall")
	entry, ok := e.openMenuEntry()
	if !ok {
		t.Fatal("no search menu")
	}
	d := e.menuDropdown(entry, true)
	if !d.Open || d.List.Len() != 1 {
		t.Fatal("expected one open search result")
	}
	l := d.listBox(f)
	item := d.List.Rows()[0]
	r := l.rowTextBox(l.Row(0), item)
	if l.wrapPitch == 0 || len(editorTextLines(f, item.Text, r.Dx()))*f.Height()+listTextY <= r.Dy() {
		t.Fatal("search result fits without height clipping")
	}
	requireFullEditorValueTip(t, e, center(r), item.Text)
}

func TestLegendCaptionClippedByHeightShowsFullTip(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	f := wordsFont("en")
	e.SetBodyFont(f)
	label := strings.Repeat("Player of the valley ", 1100) + "at the ford"
	e.doc.Records = []InspectionRecord{{Kind: KindPlayer, Label: label}}
	_ = e.Key("playertint")
	s, ok := e.legendSpec()
	if !ok {
		t.Fatal("no player legend")
	}
	l := layoutPopup(s, e.font, f)
	if len(l.Rows) != 1 || !l.Rect.In(e.mapArea()) {
		t.Fatalf("legend row is not inside the map, legend %v", l.Rect)
	}
	rr := l.Rows[0]
	if len(rr.LabelLines) == 0 || len(strings.Join(rr.LabelLines, "")) >= len(label) {
		t.Fatal("legend caption fits without height clipping")
	}
	requireFullEditorValueTip(t, e, center(rr.Label), label)
}

func requireFullEditorValueTip(t *testing.T, e *MapEditor, at image.Point, full string) {
	t.Helper()
	e.ChromeImage()
	in := Input{CursorX: at.X, CursorY: at.Y}
	now := time.Unix(1, 0)
	e.Pointer(in, now)
	if e.TipShown() != "" {
		t.Fatal("value tip appeared before the hover delay")
	}
	now = now.Add(editorTipDelay * time.Millisecond)
	e.Pointer(in, now)
	tip, ok := e.shownTip()
	if !ok || tip.Text != full || tip.Rows <= 0 || !tip.Rect.In(image.Rectangle{Max: e.window}) {
		t.Fatalf("hover shows %d of %d bytes in %v, rows %d", len(tip.Text), len(full), tip.Rect, tip.Rows)
	}
	in.WheelY = -float64(len(tip.Lines))
	e.Pointer(in, now)
	tip, ok = e.shownTip()
	if !ok || tip.Top+tip.Rows != len(tip.Lines) {
		t.Fatal("scrolling the tip did not reach the last line")
	}
}
