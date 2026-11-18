package ui

import (
	"bytes"
	"fmt"
	"image"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/render/textfit"
)

func TestChromeDescriptionFitsAtEditorTarget(t *testing.T) {
	size := image.Pt(1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			long := strings.TrimSpace(strings.Repeat("Northern Gate / Северные ворота ", 22))
			maps := []string{long, long, long, long, long, long, long, long}
			hits := make([]chromeHitValue, 43)
			for i := range hits {
				hits[i] = chromeHitValue{Category: "units", Name: fmt.Sprintf("Gate guard %d", i+1)}
			}
			players := []string{"One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight"}
			f := wordsFont(lang)
			query := "Gate"
			for name, sample := range map[string]chromePreview{
				"tabs":   {Maps: maps, Active: 4, Hover: "tab"},
				"file":   {Menu: "file"},
				"tools":  {Menu: "tools", ListRows: 13},
				"view":   {Menu: "view"},
				"window": {Menu: "window"},
				"search": {Query: &query, Hits: hits, SelectedHit: 22},
				"status": {Status: []string{"74,38", long}, Hover: "status"},
				"legend": {Players: players, FocusPlayer: 2},
			} {
				t.Run(name, func(t *testing.T) {
					d := popupDesc{Window: name, Chrome: &sample}
					m, err := NewEditorMockup(d.encode(), lang, size, f, f, mockupHostEncoder(f))
					if err != nil {
						t.Fatal(err)
					}
					c := m.chrome
					page := m.Page()
					round, err := PageDescription(page)
					if err != nil || !bytes.Equal(round, d.encode()) {
						t.Fatalf("description round trip: %v", err)
					}
					frames := map[string]image.Rectangle{
						"map-tabs":     c.editor.mapTabsRect(),
						"status-strip": c.status.Rect,
						"legend":       c.legend.Rect,
					}
					if c.menu.Open {
						frames["dropdown"] = c.menu.listFrame(f)
					}
					published := map[string]image.Rectangle{}
					for _, part := range pageBox.FindAllStringSubmatch(string(page), -1) {
						n := func(i int) int { v, _ := strconv.Atoi(part[i]); return v }
						box := image.Rect(n(3), n(4), n(3)+n(5), n(4)+n(6))
						published[part[1]] = box
						if !box.In(c.bounds) {
							t.Errorf("page %s at %v outside %v", part[1], box, c.bounds)
						}
					}
					for part, frame := range frames {
						if !frame.Empty() && published[part] != frame {
							t.Errorf("HTML %s %v differs from layout %v", part, published[part], frame)
						}
					}
					switch name {
					case "tabs":
						l := c.tabLayout
						if l.first+l.active != sample.Active || l.page[0].Empty() || !strings.HasSuffix(l.captions[l.active], textfit.Mark) {
							t.Fatalf("active tab does not fit a page: %+v", l)
						}
						for i, r := range l.rects {
							if part, n := l.hit(center(r)); part != popupTab || n != i {
								t.Errorf("tab at %v hit %v %d", r, part, n)
							}
						}
					case "status":
						row := c.status.Rows[0]
						value := statusRowText(f, c.statusSpec.Rows[0].Label, row.Label)
						if !strings.HasSuffix(value, textfit.Mark) || row.Help.Empty() {
							t.Fatalf("status has no clipping mark or help: %q", value)
						}
					case "legend":
						rows := c.legendSpec.Rows
						if len(c.legend.Rows) != len(players) || !rows[1].Swatch.focus || !rows[7].Swatch.muted {
							t.Fatalf("legend rows: %+v", rows)
						}
						for _, r := range c.legend.Rows {
							if !r.Rect.In(c.legend.Rect) || !r.Swatch.In(r.Rect) {
								t.Errorf("legend row outside the frame: %+v", r)
							}
						}
					default:
						l := c.menu.listBox(f)
						top, n := c.menu.List.Visible()
						if !c.menu.Open || !c.menu.Bounds(f).In(c.bounds) || n != l.Rows || l.Rect.Min.Y < c.menu.Rect.Max.Y {
							t.Fatalf("list placement: %+v", l)
						}
						if (name == "tools" || name == "search") && n >= c.menu.List.Len() {
							t.Fatal("long list does not page")
						}
						if name == "search" && (top == 0 || sample.SelectedHit < top || sample.SelectedHit >= top+n) {
							t.Fatalf("selected hit %d outside page %d+%d", sample.SelectedHit, top, n)
						}
						for i := range n {
							p := center(l.Row(i))
							if index, ok := l.RowAt(p); !ok || index != i || !c.menu.Covers(f, p) {
								t.Errorf("row at %v has no hit", p)
							}
						}
					}
					if sample.Hover != "" && (c.tip.Rect.Empty() || c.tip.Help || !strings.Contains(c.tip.Text, m.encode(long))) {
						t.Fatalf("tip lacks the complete clipped value: %+v", c.tip)
					}
				})
			}
		})
	}
}

func TestFittedDropdownOpensAboveOrPagesInsideWindow(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	for _, count := range []int{5, 85} {
		picker := NewPicker(make([]PickerRow, count))
		anchor := image.Rect(1800, 990, 1880, 1015)
		d := (dropdown{Rect: anchor, Open: true, List: picker}).fitList(bounds, nil, 2500, count)
		l := d.listBox(nil)
		if !d.Open || !d.Bounds(nil).In(bounds) || l.Rect.Max.Y > anchor.Min.Y || l.Rect.Min.X == anchor.Min.X {
			t.Fatalf("fitted list %v for %d rows", l.Rect, count)
		}
		_, shown := picker.Visible()
		if shown != l.Rows || (count == 5 && shown != count) || (count == 85 && shown >= count) {
			t.Fatalf("list shows %d for %d rows, geometry %d", shown, count, l.Rows)
		}
	}
}
