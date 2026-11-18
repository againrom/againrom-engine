package ui

import (
	"bytes"
	"fmt"
	"html"
	"image"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
	"againrom/pkg/words"
)

func TestDescriptionListColumnsFitPageAndTipAtEditorTarget(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, count := range []int{0, 3, 55} {
			t.Run(fmt.Sprintf("%s/%d", lang, count), func(t *testing.T) {
				d := popupDesc{Window: "probe", Title: "editor.variables.title", TitleHelp: "editor.variables.help",
					List: &popupDescList{Each: "uses", Rows: 14, Chosen: "chosen", Columns: []popupDescListColumn{
						{Head: "editor.variables.register", Share: 26, Bind: "register"},
						{Head: "editor.variables.value", Share: 20, Bind: "value"},
						{Head: "editor.variables.how", Share: 24, Bind: "how"},
						{Head: "editor.variables.trigger", Share: 30, Bind: "trigger"},
					}}, Sample: &popupScope{Lists: map[string][]popupScope{"uses": {}}}}
				long := strings.Repeat("Northern gate and bridge ", 30)
				for i := 0; i < count; i++ {
					d.Sample.Lists["uses"] = append(d.Sample.Lists["uses"], popupScope{
						Values: map[string]string{"register": strconv.Itoa(30 + i), "value": "7", "trigger": long},
						Words:  map[string]string{"how": "editor.cmp.4"}})
				}
				if count > 0 {
					d.Sample.Lists["uses"][count-1].Flags = []string{"chosen"}
				} else {
					d.Notes = []popupDescNote{{popupDescButton: popupDescButton{Word: "editor.variables.none"}}}
				}
				f := wordsFont(lang)
				m, err := NewEditorMockup(d.encode(), lang, image.Pt(1920, 1080), f, f, mockupHostEncoder(f))
				if err != nil {
					t.Fatal(err)
				}
				l := m.layout
				if !l.Rect.In(image.Rect(0, 0, 1920, 1080)) || len(l.ListHeads) != 4 {
					t.Fatalf("window %v, heads %v", l.Rect, l.ListHeads)
				}
				for k, r := range l.ListHeads {
					if !r.In(l.Rect) || strings.Join(l.ListHeadLines[k], " ") != m.paint.ListColumns[k].Head {
						t.Errorf("head %d at %v is cut or outside the window", k, r)
					}
				}
				page := m.Page()
				round, err := PageDescription(page)
				if err != nil || !bytes.Equal(round, d.encode()) {
					t.Fatalf("description round trip: %v", err)
				}
				for _, p := range pageBox.FindAllStringSubmatch(string(page), -1) {
					n := func(i int) int { v, _ := strconv.Atoi(p[i]); return v }
					r := image.Rect(n(3), n(4), n(3)+n(5), n(4)+n(6))
					if p[1] != "tip" && !r.In(l.Rect) {
						t.Errorf("page %s %v outside window %v", p[1], r, l.Rect)
					}
				}
				if count == 0 {
					if !bytes.Contains(page, []byte(html.EscapeString(words.For(lang).Text("editor.variables.none")))) {
						t.Fatal("empty list lacks its state note")
					}
					return
				}
				if count == 55 && l.List.Rows >= count {
					t.Fatal("the long list does not page")
				}
				top, shown := m.paint.List.Visible()
				for row := 0; row < min(shown, l.List.Rows); row++ {
					if part, n := l.hit(center(l.List.Cell(row, 0))); part != popupListRow || n != row {
						t.Errorf("row %d hit %v %d", row, part, n)
					}
					for col := 0; col < 4; col++ {
						r := l.List.Cell(row, col)
						box := fmt.Sprintf(`data-part="list-cell" data-row="%d" data-box="%d %d %d %d"`, top+row, r.Min.X, r.Min.Y, r.Dx(), r.Dy())
						if !bytes.Contains(page, []byte(box)) {
							t.Errorf("page lacks cell at %s", box)
						}
					}
				}
				tip, ok := popupTipAt(image.Rect(0, 0, 1920, 1080), f, f, m.paint, l, center(l.List.Cell(count-1-top, 3)), true)
				if !ok || tip.Text != m.encode(long) || !tip.Rect.In(image.Rect(0, 0, 1920, 1080)) || tip.Help {
					t.Fatalf("whole cell tip %+v, present %v", tip, ok)
				}
				if !bytes.Contains(page, []byte(`title="`+html.EscapeString(long)+`"`)) || !bytes.Contains(page, []byte(textfit.Mark)) {
					t.Fatal("page lacks the whole text of the cut cell or the clipping mark")
				}
				if _, ok := popupTipAt(image.Rect(0, 0, 1920, 1080), f, f, m.paint, l, center(l.List.Cell(0, 0)), true); ok {
					t.Fatal("a short cell shows a tip")
				}
			})
		}
	}
}

func mockupHostEncoder(f *text.Font) func(string) string {
	return func(s string) string {
		out, _ := textinput.EncodeText(s, f.Selector, textinput.Substitute)
		return out
	}
}

func TestDescriptionPlainListFitsPagesAndShowsFullRowTip(t *testing.T) {
	bounds := image.Rect(0, 0, 1920, 1080)
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			word := words.For(lang).Text
			short := word("editor.keys.brush")
			long := strings.TrimSpace(strings.Repeat(short+" ", 14))
			d := popupDesc{
				Window: "keys", Title: "editor.keys.title",
				List: &popupDescList{Each: "keys", Bind: "line", Rows: 3, Chosen: "selected", Inert: "heading"},
				Sample: &popupScope{Lists: map[string][]popupScope{"keys": {
					{Words: map[string]string{"line": "editor.keys.title"}, Flags: []string{"heading"}},
					{Words: map[string]string{"line": "editor.keys.fields"}},
					{Values: map[string]string{"line": short}},
				}}},
			}
			f := wordsFont(lang)
			mock := func() *EditorMockup {
				t.Helper()
				m, err := NewEditorMockup(d.encode(), lang, bounds.Size(), f, f, mockupHostEncoder(f))
				if err != nil {
					t.Fatal(err)
				}
				return m
			}
			m := mock()
			if m.plain.List.Rows()[0].Choosable || m.layout.List.Rect.Dx()-2*listTextX < f.Advance(m.encode(short)) {
				t.Fatalf("short list at %v does not fit or allows its heading to be chosen", m.layout.List.Rect)
			}
			d.Sample.Lists["keys"] = append(d.Sample.Lists["keys"],
				popupScope{Values: map[string]string{"line": long}, Flags: []string{"selected"}})
			hover := 3
			d.Preview = &popupPreview{Hover: &popupPreviewHover{Part: "list", Row: hover}}
			m = mock()
			top, shown := m.paint.List.Visible()
			if top != 1 || shown != 3 || !m.layout.Rect.In(bounds) {
				t.Fatalf("visible rows %d/%d, popup %v", top, shown, m.layout.Rect)
			}
			r := m.layout.List.Row(2)
			if part, n := m.layout.hit(center(r)); part != popupListRow || n != 2 {
				t.Fatalf("hovered row hit %v/%d", part, n)
			}
			tip := m.previewTip()
			if tip.Anchor != m.layout.List.rowTextBox(r, m.paint.List.Rows()[hover]) || tip.Text != m.encode(long) || !tip.Rect.In(bounds) {
				t.Fatal("the shared tip lacks the whole row or exceeds the editor bounds")
			}
			painted := 0
			for _, call := range recordWidgets(t, func() { m.Picture() }) {
				if call.kind == widgetTip && call.rect == tip.Rect {
					painted++
				}
			}
			if painted != 1 {
				t.Fatalf("painted %d tips at %v, want one", painted, tip.Rect)
			}
			page := m.Page()
			for _, part := range []string{
				fmt.Sprintf(`data-part="list-row" data-row="3" data-box="%d %d %d %d"`, r.Min.X, r.Min.Y, r.Dx(), r.Dy()),
				fmt.Sprintf(`data-part="tip" data-box="%d %d %d %d"`, tip.Rect.Min.X, tip.Rect.Min.Y, tip.Rect.Dx(), tip.Rect.Dy()),
				`data-part="list-scroll"`, `title="` + html.EscapeString(long) + `"`, textfit.Mark,
			} {
				if !bytes.Contains(page, []byte(part)) {
					t.Errorf("page lacks %s", part)
				}
			}
			round, err := PageDescription(page)
			if err != nil || !bytes.Equal(round, d.encode()) {
				t.Fatalf("description round trip: %v", err)
			}
			if _, ok := popupTipAt(bounds, f, f, m.paint, m.layout, center(m.layout.List.Row(1)), true); ok {
				t.Fatal("a whole row shows a tip")
			}
		})
	}
}

func TestDescriptionReadsLegacyListHoverAsPreview(t *testing.T) {
	for _, source := range []string{
		`{"window":"list","rows":[],"hover":{"row":0,"column":2}}`,
		`{"window":"list","rows":[],"preview":{"hoverRow":0,"hoverList":true,"hoverColumn":2}}`,
	} {
		d, err := parsePopupDesc([]byte(source))
		if p := d.Preview; err != nil || p == nil || p.Hover == nil || *p.Hover != (popupPreviewHover{Part: "list", Column: 2}) {
			t.Fatalf("list preview %+v, error %v", p, err)
		}
	}
}

func shortColumnMockup(t *testing.T, lang string) (*EditorMockup, [][4]string) {
	t.Helper()
	book := words.For(lang)
	name, spare := "Count", "Spare"
	if lang == "ru" {
		name, spare = "Счет", "Резерв"
	}
	trigger := fmt.Sprintf(book.Text("editor.trigger.title"), 2) + "  " + name
	cells := [][4]string{
		{"30", "7", book.Text("editor.cmp.4"), trigger},
		{"31", "1", book.Text("editor.action.op8"), trigger},
		{"32  " + spare, "", "", ""},
	}
	d := popupDesc{Window: "probe", Title: "editor.variables.title", TitleHelp: "editor.variables.help",
		List: &popupDescList{Each: "uses", Columns: []popupDescListColumn{
			{Head: "editor.variables.register", Share: 26, Bind: "register"},
			{Head: "editor.variables.value", Share: 20, Bind: "value"},
			{Head: "editor.variables.how", Share: 24, Bind: "how"},
			{Head: "editor.variables.trigger", Share: 30, Bind: "trigger"},
		}}, Sample: &popupScope{Lists: map[string][]popupScope{"uses": {}}}}
	for _, row := range cells {
		d.Sample.Lists["uses"] = append(d.Sample.Lists["uses"], popupScope{Values: map[string]string{
			"register": row[0], "value": row[1], "how": row[2], "trigger": row[3],
		}})
	}
	f := wordsFont(lang)
	m, err := NewEditorMockup(d.encode(), lang, image.Pt(1920, 1080), f, f, mockupHostEncoder(f))
	if err != nil {
		t.Fatal(err)
	}
	return m, cells
}

func TestDescriptionListColumnsPaintEachCellInsideItsColumn(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			m, rows := shortColumnMockup(t, lang)
			calls := text.Record(func() { m.Picture() })
			for row, cells := range rows {
				for k, full := range cells {
					box := m.layout.List.Cell(row, k)
					want := m.encode(full)
					var got []text.DrawCall
					for _, c := range calls {
						if c.Y == box.Min.Y+listTextY && c.X >= box.Min.X && c.X < box.Max.X {
							got = append(got, c)
						}
					}
					if len(got) != len(want) {
						t.Errorf("row %d column %d painted %d glyphs, want %d for %q", row, k, len(got), len(want), full)
						continue
					}
					for i, c := range got {
						glyph := image.Rect(c.X, c.Y, c.X+c.Glyph.Width, c.Y+c.Glyph.Height)
						if c.X != box.Min.X+m.body.Advance(want[:i]) || c.Glyph != m.body.GlyphFor(want[i]) || !glyph.In(box) {
							t.Errorf("row %d column %d glyph %d at %d,%d leaves %v or selects the wrong glyph", row, k, i, c.X, c.Y, box)
						}
					}
				}
			}
		})
	}
}

func TestDescriptionListFewRowsKeepTheirNaturalColumnWidth(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, count := range []int{0, 3, 55} {
			for _, wideHead := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/rows=%d/wide-head=%v", lang, count, wideHead), func(t *testing.T) {
					m, sourceRows := shortColumnMockup(t, lang)
					d, body := m.desc, m.body
					items := d.Sample.Lists["uses"]
					d.Sample.Lists["uses"] = nil
					d.List.Rows = 14
					var rows [][4]string
					for i := 0; i < count; i++ {
						item, row := items[i%len(items)], sourceRows[i%len(sourceRows)]
						if wideHead {
							row = [4]string{"30", "7", "1", "2"}
							item.Values = map[string]string{"register": "30", "value": "7", "how": "1", "trigger": "2"}
						}
						d.Sample.Lists["uses"] = append(d.Sample.Lists["uses"], item)
						rows = append(rows, row)
					}
					heading := body
					if wideHead {
						heading = wordsFont(lang)
						heading.Spacing += 4
						head := m.encode(words.For(lang).Text("editor.variables.how"))
						g := heading.GlyphFor(head[len(head)-1])
						g.Width += 48
						g.Pixels = make([]text.Pixel, g.Width*g.Height)
						g.Pixels[g.Width-1] = text.Pixel{Level: text.MaxLevel, Painted: true}
						headWidth, _ := heading.Measure(head)
						if headWidth <= heading.Advance(head) {
							t.Fatal("the heading fixture has no glyph overhang")
						}
						for _, row := range rows {
							cellWidth, _ := body.Measure(m.encode(row[2]))
							if headWidth <= cellWidth {
								t.Fatal("the heading is not wider than every cell")
							}
						}
					}
					var err error
					m, err = NewEditorMockup(d.encode(), lang, image.Pt(1920, 1080), heading, body, mockupHostEncoder(body))
					if err != nil {
						t.Fatal(err)
					}
					calls := text.Record(func() { m.Picture() })
					shares := [4]int{26, 20, 24, 30}
					var need [4]int
					for k, key := range []string{"editor.variables.register", "editor.variables.value", "editor.variables.how", "editor.variables.trigger"} {
						head := m.encode(words.For(lang).Text(key))
						headWidth, _ := heading.Measure(head)
						need[k] = headWidth + listColumnGap
						if got := m.layout.List.Cell(0, k).Dx(); got < headWidth {
							t.Errorf("column %d leaves %d pixels for its heading, need %d plus the column padding", k, got, headWidth)
						}
						r := m.layout.ListHeads[k]
						var painted []text.DrawCall
						for _, c := range calls {
							if c.Clip == r && c.Color == editorPopupInk.Title {
								painted = append(painted, c)
							}
						}
						if len(painted) != len(head) || len(m.layout.ListHeadLines[k]) != 1 {
							t.Errorf("column %d painted %d heading glyphs in %d lines, need %d in one line", k, len(painted), len(m.layout.ListHeadLines[k]), len(head))
						} else {
							for i, c := range painted {
								glyph := image.Rect(c.X, c.Y, c.X+c.Glyph.Width, c.Y+c.Glyph.Height)
								if c.Glyph != heading.GlyphFor(head[i]) || c.X != r.Min.X+heading.Advance(head[:i]) || !glyph.In(r) {
									t.Errorf("column %d heading glyph %d has the wrong font, spacing or bounds %v inside %v", k, i, glyph, r)
									break
								}
							}
						}
						for _, row := range rows {
							cellWidth, _ := body.Measure(m.encode(row[k]))
							need[k] = max(need[k], cellWidth+listColumnGap)
						}
					}
					// The smallest shared row whose four shares hold the independent cells.
					content := 0
					for k := range need {
						content = max(content, (need[k]*100+shares[k]-1)/shares[k])
					}
					var natural [4]int
					at := 0
					for k, share := range shares {
						natural[k] = (at+share)*content/100 - at*content/100
						at += share
					}
					frame := 2*popupPad + widgetSpriteSize + 2*listTextX
					if got, limit := m.layout.Rect.Dx(), natural[0]+natural[1]+natural[2]+natural[3]+frame; got > limit {
						t.Fatalf("list width %d exceeds natural columns %v plus frame %d", got, natural, frame)
					}
					for row := 0; row < min(len(rows), m.layout.List.Rows); row++ {
						for k, full := range rows[row] {
							cellWidth, _ := body.Measure(m.encode(full))
							if cellWidth > m.layout.List.Cell(row, k).Dx() {
								t.Errorf("short row %d column %d does not fit", row, k)
							}
						}
					}
				})
			}
		}
	}
}
func TestPopupListCellTipUsesTheSharedDelay(t *testing.T) {
	f := wordsFont("en")
	long := strings.Repeat("A trigger with a long name ", 15)
	s := popupSpec{Title: "Window", Width: 400, At: image.Pt(200, 200), Bounds: image.Rect(0, 0, 1920, 1080),
		List: NewPicker([]PickerRow{{Text: long + "\t7", Choosable: true}}).SetWindow(1), ListRows: 1,
		ListColumns: []listColumn{{Head: "Trigger", Share: 1}, {Head: "Value", Share: 1}}}
	l := layoutPopup(s, f, f)
	e := NewMapEditor(f, nil, nil)
	e.SetBodyFont(f)
	e.Layout(1920, 1080)
	e.tip.windows = []tipWindow{{s, l}}
	p := center(l.List.Cell(0, 0))
	e.pointer, e.pointerOK = p, true
	t0 := time.Unix(100, 0)
	e.observeTip(p, t0, false)
	if e.TipShown() != "" {
		t.Fatal("cell tip shows without the delay")
	}
	e.observeTip(p, t0.Add(editorTipDelay*time.Millisecond), false)
	if e.TipShown() != long {
		t.Fatalf("cell tip after the delay %q", e.TipShown())
	}
	if tip, ok := popupTipAt(s.Bounds, f, f, s, l, l.List.Bar().Min, true); ok {
		t.Fatalf("scrollbar shows cell tip %+v", tip)
	}
}

// listDesc is a brush window: a picture row of heads, a grid with a ticked
// cell, a list whose chosen row's picture stands beside it, and a switch
// button.
const listDesc = `{
  "window": "probe",
  "title": "editor.bar.objects",
  "width": 420,
  "labelWidth": 90,
  "rows": [{"kind": "text", "label": "editor.brush.size", "value": "size"}],
  "grid": {"each": "modes", "picture": "icon", "tip": "name", "ticked": "ticked", "side": 34, "rows": 1,
    "rowHeads": ["editor.brush.mode"]},
  "list": {"each": "objects", "bind": "name", "chosen": "chosen",
    "aside": {"picture": "chosen.picture", "width": 40}},
  "switch": {"word": "editor.place.ruined", "on": "ruined"},
  "sample": {
    "values": {"size": "2", "chosen.picture": "record:1"},
    "lists": {
      "modes": [
        {"words": {"name": "editor.brush.mode.step"}},
        {"words": {"name": "editor.brush.mode.level"}, "flags": ["ticked"]}
      ],
      "objects": [
        {"words": {"name": "editor.brush.noobject"}},
        {"values": {"name": "Rock 1"}, "flags": ["chosen"]},
        {"values": {"name": "Rock 2"}}
      ]
    }
  }
}`

func TestDescriptionListStandsItsPictureBesideIt(t *testing.T) {
	m, err := newEditorMockup([]byte(listDesc), "en", image.Pt(800, 600), wordsFont("en"), wordsFont("en"), nil, mockupPerson{picture: redPicture})
	if err != nil {
		t.Fatal(err)
	}
	g := m.plain.Grid
	if g == nil || len(g.Cells) != 2 || g.Cells[0].Ticked || !g.Cells[1].Ticked || len(g.RowHeads) != 1 || g.RowHeads[0] != "Mode" || g.Cols != 2 {
		t.Fatalf("grid %+v", g)
	}
	if m.plain.List == nil || m.plain.List.Len() != 3 || m.plain.List.Selection() != 1 || m.plain.ListRows != 3 {
		t.Fatalf("list rows %d shown %d", m.plain.List.Len(), m.plain.ListRows)
	}
	if !strings.HasSuffix(m.plain.SwitchButton, "Ruined") {
		t.Errorf("switch button %q", m.plain.SwitchButton)
	}
	box := m.layout.Aside
	if box.Empty() || m.plain.ListPicture == nil {
		t.Fatalf("aside %v picture %v", box, m.plain.ListPicture != nil)
	}
	p := m.Picture()
	inside, outside := 0, 0
	for y := p.Rect.Min.Y; y < p.Rect.Max.Y; y++ {
		for x := p.Rect.Min.X; x < p.Rect.Max.X; x++ {
			if p.RGBAAt(x, y) != pictureRed {
				continue
			}
			if image.Pt(x, y).In(box) {
				inside++
			} else {
				outside++
			}
		}
	}
	if inside == 0 || outside != 0 {
		t.Errorf("%d picture pixels beside the list %v, %d outside", inside, box, outside)
	}
	page := m.Page()
	for part, want := range map[string]int{`data-part="list-row"`: 3, `data-part="switch-button"`: 1, ` ticked"`: 1, `>record:1<`: 1} {
		if n := bytes.Count(page, []byte(part)); n != want {
			t.Errorf("the page holds %d %s, not %d", n, part, want)
		}
	}
}
