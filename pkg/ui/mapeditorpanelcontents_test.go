package ui

import (
	"image"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/textfit"
	"againrom/pkg/words"
)

func TestNaturalMapPanelFitsBesideTheDock(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			p := &mapPanelEdits{facts: playable()}
			p.facts.Description = strings.TrimSpace(strings.Repeat("Hold the northern bridge. ", 32))
			e, _ := mapPanelEditor(t, p, nil)
			e.SetWords(words.For(lang))
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.SetPanelProfile(nil, nil, nil)
			e.Layout(1920, 1080)
			if !e.panelOpen(panelLeft) || e.bodyFont().Advance(p.facts.Description) <= e.window.X {
				t.Fatal("the fixture needs the objects dock and a value wider than the window")
			}
			s, _ := e.listSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			if room := e.mapArea(); l.Rect.Min.X < room.Min.X || l.Rect.Max.X > room.Max.X {
				t.Fatalf("natural Map panel %v covers a side dock beside %v", l.Rect, room)
			}
			for _, r := range l.Rows {
				row := s.Rows[r.Index]
				if e.bodyFont().Advance(row.Label) > r.Label.Dx()-2 {
					t.Fatalf("label %q is clipped", row.Label)
				}
				if row.Label != e.word("editor.panel.description") {
					continue
				}
				shown := editorClip(e.bodyFont(), row.Value, r.Control.Dx()-2)
				tip, ok := popupTipAt(e.popupBounds(), e.font, e.bodyFont(), s, l, center(r.Control), true)
				if !strings.HasSuffix(shown, textfit.Mark) || !ok || !strings.Contains(tip.Text, p.facts.Description) {
					t.Fatal("the fitted description lost its mark or its full tip")
				}
				modesClick(e, center(r.Control))
				if e.prompt.kind != promptWorkspace || e.prompt.scope != "metadata" || e.prompt.field != 1 {
					t.Fatal("the description link did not open its field")
				}
				return
			}
			t.Fatal("the description row is absent")
		})
	}
}

func TestDockedSaveOutcomeStaysVisibleAtMinimum(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := panelsEditor(t, 1920, 1080, lang)
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.SetPanelProfile([]byte(`{"bottom":{"minimized":false,"width":1}}`), nil, nil)
			rows := make([]InspectionFinding, 4)
			for i := range rows {
				rows[i] = InspectionFinding{Error: true, Text: strings.Repeat("bridge guard ", 30)}
			}
			refusal := &ReportRefusal{Findings: rows}
			check := func() {
				t.Helper()
				s, _ := e.listSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				w := l.window(s)
				if l.Rect != e.sidePanelRect(panelBottom) || w.Title.Text != e.word("editor.side.report") || l.ListRows != popupMinRows {
					t.Fatalf("save report frame %v, title %q, findings %d", l.Rect, w.Title.Text, l.ListRows)
				}
				outcome := strings.Join(w.Notes.Lines, "")
				if !strings.Contains(collapseGaps(outcome), collapseGaps(e.report.note)) {
					t.Errorf("save outcome %q is absent from the visible body", e.report.note)
				}
				if !strings.Contains(w.Title.Help, "bridge guard") || strings.Contains(w.Title.Help, e.report.note) {
					t.Error("report help must contain the folded finding details")
				}
				for _, r := range l.Notes {
					if !r.In(l.Rect) || r.Min.Y < l.List.Rect.Max.Y {
						t.Errorf("report body line %v is outside the space below the findings", r)
					}
				}
			}
			confirmed := false
			e.askSaveWithErrors(refusal, func() error {
				confirmed = true
				return nil
			})
			check()
			if err := e.Key("enter"); err != nil || !confirmed {
				t.Fatalf("Enter did not confirm the save: %v", err)
			}
			e.refuseSave(refusal)
			check()
		})
	}
}

func TestShortReportReservesOnlyItsFindings(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := panelsEditor(t, 1920, 1080, lang)
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.SetPanelProfile([]byte(`{"bottom":{"minimized":false,"width":1}}`), nil, nil)
			height := func(n int) int {
				rows := make([]InspectionFinding, n)
				e.openReport(rows, e.checkReportTitle(rows), nil)
				s, _ := e.listSpec()
				l := layoutPopup(s, e.font, e.bodyFont())
				if l.ListRows != n || l.List.Rows != n || l.Rect != e.sidePanelRect(panelBottom) {
					t.Errorf("%d findings use %d list rows in layout %v", n, l.List.Rows, l.Rect)
				}
				at := center(l.splitterRect())
				lower := at.Add(image.Pt(0, 40))
				modesPress(e, at, true)
				modesPress(e, lower, true)
				modesPress(e, lower, false)
				if got := e.sidePanelRect(panelBottom); got != l.Rect {
					t.Errorf("the splitter shrank the dock for %d findings from %v to %v", n, l.Rect, got)
				}
				return l.Rect.Dy()
			}
			one, two, three := height(1), height(2), height(3)
			if !(one < two && two < three) {
				t.Errorf("restored heights for one, two and three findings are %d, %d and %d", one, two, three)
			}
		})
	}
}

func TestMapDescriptionIsOneLinkedValue(t *testing.T) {
	p := &mapPanelEdits{facts: playable()}
	p.facts.Description = strings.TrimSpace(strings.Repeat("Hold the northern bridge. ", 24))
	e, _ := mapPanelEditor(t, p, nil)
	e.Layout(1920, 1080)
	s, links := e.panelSpec()
	n := 0
	for i, link := range links {
		if link == linkDescription {
			n++
			if n == 1 && s.Rows[i].Value != p.facts.Description {
				t.Fatal("the description row lost part of its value")
			}
		}
	}
	if n != 1 {
		t.Fatalf("description uses %d rows", n)
	}
	clickPanelRow(t, e, panelRow(t, e, e.word("editor.panel.description")))
	if e.prompt.kind != promptWorkspace || e.prompt.scope != "metadata" || e.prompt.field != 1 {
		t.Fatal("the description link did not open its field")
	}
}

func TestReportDockDragKeepsThreeFindings(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := panelsEditor(t, 1920, 1080, lang)
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.SetPanelProfile([]byte(`{"bottom":{"minimized":true,"width":96}}`), nil, e.panels.save)
			strip, ok := e.stripSpec(panelBottom)
			if !ok {
				t.Fatal("the saved report strip did not open")
			}
			modesClick(e, center(layoutPopup(strip, e.font, e.bodyFont()).Close))
			minimum := e.reportDockMinHeight(e.mapArea().Dx())
			if got := e.sidePanelRect(panelBottom).Dy(); got != minimum || e.panels.state[panelBottom].Width != 96 {
				t.Fatalf("restored height = %d, saved height = %d, minimum = %d", got, e.panels.state[panelBottom].Width, minimum)
			}
			e.panels.state[panelBottom] = panelState{Restored: true, Width: 320}
			e.relayout()
			if s, ok := e.reportHintSpec(); !ok || len(s.Rows) != 0 || s.TitleHelp != e.word("editor.side.noreport") {
				t.Error("the empty dock does not put its hint behind title help")
			}
			rows := make([]InspectionFinding, 32)
			for i := range rows {
				rows[i] = InspectionFinding{Kind: "Unit", Index: 1, Text: strings.Repeat("bridge guard ", 25)}
			}
			e.openReport(rows, e.checkReportTitle(rows), nil)
			minimum = e.reportDockMinHeight(e.mapArea().Dx())
			read := func() (popupSpec, popupLayout) {
				s, ok := e.listSpec()
				if !ok {
					t.Fatal("no report")
				}
				return s, layoutPopup(s, e.font, e.bodyFont())
			}
			_, tall := read()
			if tall.ListRows <= popupMinRows || len(tall.Notes) <= 1 {
				t.Fatal("the tall dock should show findings and details")
			}
			at := center(tall.splitterRect())
			drag := func(dy int) {
				modesPress(e, at, true)
				modesPress(e, at.Add(image.Pt(0, dy)), true)
				modesPress(e, at.Add(image.Pt(0, dy)), false)
			}
			drag(280)
			s, l := read()
			panel := e.sidePanelRect(panelBottom)
			if panel.Dy() != minimum || l.Rect != panel || l.ListRows != popupMinRows || len(l.Notes) != 2 || !l.List.Rect.In(panel) || !l.List.Bar().In(panel) {
				t.Fatalf("minimum dock %v, layout %v, findings %d, notes %d", panel, l.Rect, l.ListRows, len(l.Notes))
			}
			if help := l.window(s).Title.Help; !strings.Contains(help, "bridge guard") || l.TitleHelp.Empty() {
				t.Fatal("the findings details did not move into title help")
			}
			if displayed := l.window(s).Notes.Lines[len(l.Notes)-1]; displayed != s.Notes[len(s.Notes)-1] {
				t.Fatal("shrinking the findings lost the visible list counter")
			}
			at = center(l.splitterRect())
			drag(30)
			if e.sidePanelRect(panelBottom) != panel {
				t.Fatal("the dock shrank past the minimum")
			}
			p := center(l.List.Row(1))
			if part, n := l.hit(p); part != popupListRow || n != 1 {
				t.Fatalf("finding hit = %v %d", part, n)
			}
			modesClick(e, p)
			if e.report.sel != 1 || e.selected != 1 {
				t.Fatalf("finding click selected report row %d and record %d", e.report.sel, e.selected)
			}
		})
	}
}

func TestMapPanelPagesWithFixedTitleAndFullValueTip(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			p := &mapPanelEdits{facts: playable()}
			p.facts.Description = strings.TrimSpace(strings.Repeat("Hold the northern bridge. ", 24))
			e, _ := mapPanelEditor(t, p, nil)
			e.SetWords(words.For(lang))
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.Layout(1920, 1080)
			e.places[popList].key = e.windowKey(popList, "")
			e.places[popList].size = image.Pt(580, 160)
			s, _ := e.listSpec()
			l := layoutPopup(s, e.font, e.bodyFont())
			bounds := image.Rect(0, 0, 1920, 1080)
			if l.RowBar.Empty() || l.TitleHelp.Empty() || !l.Rect.In(bounds) || !l.RowBar.In(l.Rect) {
				t.Fatal("the map panel has no bounded page and title help")
			}
			found := false
			for _, r := range l.Rows {
				row := s.Rows[r.Index]
				if !r.Rect.In(l.Rect) || e.bodyFont().Advance(row.Label) > r.Label.Dx()-2 {
					t.Fatalf("row %d cuts its label or leaves the panel", r.Index)
				}
				if row.Label == e.word("editor.panel.description") {
					found = true
					tip, ok := popupTipAt(bounds, e.font, e.bodyFont(), s, l, center(r.Control), true)
					if !ok || !strings.Contains(tip.Text, p.facts.Description) || !tip.Rect.In(bounds) {
						t.Fatal("the clipped description lost its full tip")
					}
				}
			}
			if !found {
				t.Fatal("the description is absent from the first page")
			}
			at := center(l.RowsViewport)
			e.Pointer(Input{CursorX: at.X, CursorY: at.Y, WheelY: -1}, time.Time{})
			s, _ = e.listSpec()
			next := layoutPopup(s, e.font, e.bodyFont())
			if next.Title != l.Title || next.TitleHelp != l.TitleHelp || next.Rows[0].Index <= l.Rows[0].Index {
				t.Fatal("paging did not move the rows under the fixed title")
			}
		})
	}
}

func TestObjectsPanelClippedMemberTipAndSelection(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e, _, _ := panelsEditor(t, 1920, 1080, lang)
			e.font = wordsFont(lang)
			e.SetBodyFont(e.font)
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			name := strings.TrimSpace(strings.Repeat("Northern bridge guard ", 8))
			e.doc.Records[0].Label = name
			s, ok := e.treeSpec()
			if !ok {
				t.Fatal("no objects panel")
			}
			l := layoutPopup(s, e.font, e.bodyFont())
			panel := e.sidePanelRect(panelLeft)
			if l.Rect != panel || !l.List.Rect.In(panel) || !l.List.Bar().In(panel) || !l.Rows[0].Rect.In(panel) {
				t.Fatal("the objects contents leave the panel")
			}
			if e.bodyFont().Advance(s.Rows[0].Label) > l.Rows[0].Label.Dx()-2 {
				t.Fatal("the search label is clipped")
			}
			top, shown := s.List.Visible()
			for i, row := range s.List.Rows()[top : top+shown] {
				rowBox := l.List.Row(i)
				if row.Heading && l.List.rowFace(e.bodyFont(), row).Advance(row.Text) > l.List.rowTextBox(rowBox, row).Dx() {
					t.Fatalf("tree heading %q is clipped", row.Text)
				}
				if !strings.Contains(row.Text, name) {
					continue
				}
				at := center(rowBox)
				if part, n := l.hit(at); part != popupListRow || n != i {
					t.Fatalf("object hit = %v %d", part, n)
				}
				e.ChromeImage()
				start := time.Unix(1, 0)
				in := Input{CursorX: at.X, CursorY: at.Y}
				e.Pointer(in, start)
				if e.TipShown() != "" {
					t.Fatal("the name tip appeared before the hover delay")
				}
				e.Pointer(in, start.Add(time.Second))
				if tip, ok := e.shownTip(); !ok || tip.Text != row.Text || !tip.Rect.In(e.popupBounds()) {
					t.Fatal("the clipped member lost its full tip")
				}
				modesClick(e, at)
				if e.selected != 0 {
					t.Fatalf("the tree click selected record %d", e.selected)
				}
				return
			}
			t.Fatal("the named member is missing from the visible tree")
		})
	}
}
