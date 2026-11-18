package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/textfit"
)

func TestMapTabCloseTargetsItsOwnMap(t *testing.T) {
	e, _, _ := menuEditor(t, "en")
	active := e.doc
	active.Source = "active.alm"
	clean, _, _ := menuEditor(t, "en")
	clean.doc.Source = "clean.alm"
	dirty, _, _ := menuEditor(t, "en")
	dirty.doc.Source, dirty.doc.Edits = "dirty.alm", &reportEdits{dirty: true}
	e.openMaps = []*openMap{{doc: active}, {doc: clean.doc}, {doc: dirty.doc}}
	closeTab := func(n int) {
		t.Helper()
		r, ok := e.FindControl(ControlQuery{Role: ControlMapTab, N: n, Flag: true})
		if !ok {
			t.Fatalf("map %d has no close mark", n)
		}
		chromeClick(e, center(r))
	}
	closeTab(1)
	if e.doc != active || e.OpenMapCount() != 2 || e.openMaps[1].doc != dirty.doc || e.prompt.open() {
		t.Fatal("closing the clean background map changed the shown map or closed another map")
	}
	closeTab(1)
	if e.prompt.kind != promptLeave || !strings.Contains(strings.Join(e.PromptNotes(), ""), "dirty.alm") {
		t.Fatalf("dirty background map left prompt %d, %q", e.prompt.kind, e.PromptNotes())
	}
	_ = e.Key("escape")
	if e.OpenMapCount() != 2 {
		t.Fatal("cancel removed the dirty map")
	}
	e.ShowOpenMap(0)
	closeTab(1)
	s, ok := e.promptSpec()
	if !ok {
		t.Fatal("no unsaved-changes question on the second press")
	}
	chromeClick(e, center(layoutPopup(s, e.font, e.bodyFont()).Buttons[leaveGo]))
	if e.OpenMapCount() != 1 || e.doc != active {
		t.Fatal("discard did not close only the dirty map")
	}
	if _, ok := e.FindControl(ControlQuery{Role: ControlMapTab, Flag: true}); ok {
		t.Fatal("the last map still has a close mark")
	}
}

func TestEditorToolsMenuPagesAfterThirteenRows(t *testing.T) {
	e, _, _ := terrainEditor(t)
	e.Layout(1920, 1080)
	chromeClick(e, barPoint(t, e, "tools"))
	b, ok := e.openMenuEntry()
	if !ok {
		t.Fatal("Tools did not open")
	}
	d := e.menuDropdown(b, true)
	last, ok := e.MenuRow("tools", "randomheights")
	if !ok || d.Rows != 13 || d.List.Len() <= d.Rows {
		t.Fatalf("Tools shows %d of %d rows", d.Rows, d.List.Len())
	}
	query := ControlQuery{Role: ControlMenuRow, N: last}
	if _, ok := e.FindControl(query); ok {
		t.Fatal("the last action appears on the first page")
	}
	p := center(d.listBox(e.bodyFont()).Row(0))
	for range d.List.Len() {
		e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, time.Time{})
	}
	r, ok := e.FindControl(query)
	if !ok {
		t.Fatal("the last action could not be reached by scrolling")
	}
	chromeClick(e, center(r))
	if e.prompt.kind != promptSeed || e.prompt.scope != "randomheights" {
		t.Fatalf("last row opened prompt %d for %q", e.prompt.kind, e.prompt.scope)
	}
}

func TestEditorChromeFrameworkFitsAndHitsAtEditorTarget(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := menuEditor(t, lang)
			e.SetBodyFont(wordsFont(lang))
			e.font = wordsFont(lang)
			bounds := image.Rect(0, 0, 1920, 1080)
			inside := func(r image.Rectangle) {
				t.Helper()
				if !r.In(bounds) {
					t.Errorf("%v outside %v", r, bounds)
				}
			}
			hover := func(p image.Point, want string, immediate bool) {
				t.Helper()
				now := time.Now()
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, now)
				if !immediate {
					if e.TipShown() != "" {
						t.Fatal("value tip appeared before its delay")
					}
					e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, now.Add(editorTipDelay*time.Millisecond))
				}
				tip, ok := e.shownTip()
				if !ok || tip.Text != want || tip.Help != immediate {
					t.Fatalf("tip %v help=%v text length %d, want help=%v length %d", ok, tip.Help, len(tip.Text), immediate, len(want))
				}
				inside(tip.Rect)
			}
			active := e.doc
			active.Name = strings.Repeat("Northern gate ", 24)
			e.openMaps = []*openMap{{doc: active}}
			for i := 1; i < maxOpenMaps; i++ {
				d := *active
				d.Name = fmt.Sprintf("%d %s", i, active.Name)
				e.openMaps = append(e.openMaps, &openMap{doc: &d, selected: -1})
			}
			_, tabs := e.mapTabStrip()
			inside(e.mapTabsRect())
			for i, r := range tabs.rects {
				inside(r)
				inside(tabs.closes[i])
				p := center(tabs.closes[i])
				if part, n := tabs.hit(p); part != popupClose || n != i || e.chromeHit(p) != popMapTabs {
					t.Fatalf("close mark %d hit %v/%d", i, part, n)
				}
			}
			for _, r := range tabs.page {
				inside(r)
			}
			p := center(tabs.rects[tabs.active])
			if full := tabs.full[tabs.active]; full != active.Name || !strings.HasSuffix(tabs.captions[tabs.active], textfit.Mark) {
				t.Fatal("the long active tab does not retain its full caption")
			}
			hover(p, active.Name, false)
			hover(center(tabs.closes[tabs.active]), e.word("editor.tabs.close"), false)
			hover(center(tabs.page[1]), e.word("editor.tabs.next"), false)
			next := tabs.pageTarget(1)
			chromeClick(e, center(tabs.page[1]))
			_, tabs = e.mapTabStrip()
			if e.shownMap != next || tabs.first+tabs.active != next {
				t.Fatal("next page did not bring its active tab into view")
			}
			hover(center(tabs.page[0]), e.word("editor.tabs.previous"), false)
			chromeClick(e, center(tabs.page[0]))
			_, tabs = e.mapTabStrip()
			n := e.OpenMapCount()
			chromeClick(e, center(tabs.closes[tabs.active]))
			if e.OpenMapCount() != n-1 {
				t.Fatal("a visible close mark did not go through Pointer")
			}

			e.doc.Source = strings.Repeat("long map source ", 24)
			s := e.statusSpec()
			status := layoutPopup(s, e.font, e.bodyFont())
			row := status.Rows[0]
			inside(status.Rect)
			inside(row.Label)
			inside(row.Help)
			line, width, whole, room := e.StatusFit()
			if width > room || whole <= room || !strings.HasSuffix(line, textfit.Mark) {
				t.Fatalf("status fit %d of %d, full width %d", width, room, whole)
			}
			hover(center(row.Label), e.StatusLine(), false)
			hover(center(row.Help), strings.Join(e.helpLines(), "\n"), true)

			dropdownHitAt := func(n int) image.Point {
				t.Helper()
				b, ok := e.openMenuEntry()
				if !ok {
					t.Fatal("no open menu")
				}
				d := e.menuDropdown(b, true)
				frame := d.listFrame(e.font)
				inside(frame)
				painted := image.Rectangle{}
				widgetRecord = func(c widgetCall) {
					if c.kind == widgetFrame && c.rect.Overlaps(d.listBox(e.font).Rect) {
						painted = c.rect
					}
				}
				e.ChromeImage()
				widgetRecord = nil
				if painted != frame || e.chromeHit(frame.Min) != popMenu || d.Covers(e.font, image.Pt(frame.Min.X, frame.Min.Y-1)) {
					t.Fatalf("frame %v, painted %v, rim at %v", frame, painted, e.chromeHit(frame.Min))
				}
				r, ok := e.FindControl(ControlQuery{Role: ControlMenuRow, N: n})
				if !ok {
					t.Fatalf("menu row %d outside its viewport", n)
				}
				inside(r)
				p := center(r)
				if part, i := d.hit(e.font, p); part != dropdownRow || i != n {
					t.Fatalf("menu row %d hit %v/%d", n, part, i)
				}
				return p
			}
			chromeClick(e, barPoint(t, e, "view"))
			chromeClick(e, dropdownHitAt(menuKeyRow(t, e, "view", "playertint")))
			if !e.view.playerColours {
				t.Fatal("View row failed to toggle the legend")
			}

			_ = e.Key("search")
			e.TextInput("hall")
			rows := e.searchHits()
			if len(rows) != 1 {
				t.Fatalf("search found %d hits", len(rows))
			}
			chromeClick(e, dropdownHitAt(0))
			if e.searchTyping() || e.selected != rows[0].row.record {
				t.Fatal("search row did not select the tree member")
			}

			e.selected = -1
			e.doc.Records = nil
			for i := range editorPlayerTints {
				label := strings.Repeat(fmt.Sprintf("Player %d ", i+1), 80)
				e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindPlayer, Index: i, Label: label})
			}
			s, ok := e.legendSpec()
			if !ok {
				t.Fatal("no player legend")
			}
			legend := layoutPopup(s, e.font, e.bodyFont())
			inside(legend.Rect)
			if !legend.Rect.In(e.mapArea()) || legend.RowBar.Empty() || len(legend.Rows) == 0 {
				t.Fatalf("legend %v does not page within %v", legend.Rect, e.mapArea())
			}
			for _, rr := range legend.Rows {
				if !rr.Rect.In(legend.Rect) || !rr.Swatch.In(rr.Rect) || len(rr.LabelLines) < 2 {
					t.Fatalf("legend row %d does not wrap inside its frame", rr.Index)
				}
			}
			p = center(legend.Rows[0].Rect)
			if part, i := legend.hit(p); part != popupRowLine || i != legend.Rows[0].Index || e.chromeHit(p) != popLegend {
				t.Fatalf("legend row hit %v/%d", part, i)
			}
			e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -1}, time.Time{})
			s, _ = e.legendSpec()
			if after := layoutPopup(s, e.font, e.bodyFont()); after.Rows[0].Index <= legend.Rows[0].Index {
				t.Fatal("wheel over the legend row did not page the legend")
			}
		})
	}
}
