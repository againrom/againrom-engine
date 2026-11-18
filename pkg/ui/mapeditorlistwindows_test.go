package ui

import (
	"fmt"
	"image"
	"slices"
	"strings"
	"testing"
	"time"

	"againrom/pkg/render/text"
	"againrom/pkg/render/textfit"
	"againrom/pkg/words"
)

var listWindowScreen = image.Pt(1920, 1080)

func checkListWindow(t *testing.T, e *MapEditor, id editorPopupID, s popupSpec) popupLayout {
	t.Helper()
	checkWindowProperties(t, e, s, listWindowScreen)
	l := layoutPopup(s, e.font, e.bodyFont())
	for i, r := range l.Buttons {
		if e.chromeHit(center(r)) != id || e.bodyFont().Advance(s.Buttons[i]) > r.Dx() {
			t.Fatalf("button %d at %v does not fit or receive presses", i, r)
		}
	}
	if l.HasList {
		if !l.List.Rect.In(l.Rect) || !l.List.Bar().In(l.Rect) {
			t.Fatalf("list at %v or bar at %v leaves %v", l.List.Rect, l.List.Bar(), l.Rect)
		}
		top, shown := s.List.Visible()
		for i := 0; i < shown; i++ {
			r := l.List.Row(i)
			if !r.In(l.Rect) || e.chromeHit(center(r)) != id {
				t.Fatalf("row %d at %v does not fit or receive presses", top+i, r)
			}
			if part, row := l.hit(center(r)); part != popupListRow || row != i {
				t.Fatalf("row %d hit as %v %d", i, part, row)
			}
		}
	}
	return l
}

func TestEditorListWindowsLayoutAndActions(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			t.Run("records", func(t *testing.T) {
				e := modesWordsEditor(t, "full", lang, listWindowScreen)
				e.showRecords(0)
				s := mustListSpec(t, e)
				l := checkListWindow(t, e, popList, s)
				want := e.rows()[1]
				chromeClick(e, center(l.List.Row(1)))
				if e.selected != want || e.objectHidden {
					t.Fatalf("record row selected %d, want %d with its window open", e.selected, want)
				}
			})
			t.Run("scenario", func(t *testing.T) {
				e := modesWordsEditor(t, "full", lang, listWindowScreen)
				e.barAct("scripts")
				s := mustListSpec(t, e)
				l := checkListWindow(t, e, popList, s)
				if s.TitleHelp != e.word("editor.scripts.hint") {
					t.Fatalf("Scenario help %q", s.TitleHelp)
				}
				_, resized := edgeDrag(t, e, bottomRight, image.Pt(40, 50))
				e.barAct("structures")
				e.barAct("scripts")
				l = checkListWindow(t, e, popList, mustListSpec(t, e))
				if l.Rect != resized {
					t.Fatalf("Scenario returned at %v, want %v", l.Rect, resized)
				}
				chromeClick(e, center(l.Buttons[0]))
				if e.pick != pickDrop {
					t.Fatal("Drop zones did not start picking")
				}
			})
			t.Run("traps and spells", func(t *testing.T) {
				e := newPlacementLayoutEditor(t, lang)
				for i := 0; i < 32; i++ {
					e.doc.Records = append(e.doc.Records, InspectionRecord{Kind: KindEnchantment, Index: i, Label: "Trap", Spatial: true})
				}
				e.showRecords(editorCellSpellsFilter)
				s := mustListSpec(t, e)
				l := checkListWindow(t, e, popList, s)
				if l.Grid.Empty() || l.ListRows >= s.List.Len() || s.TitleHelp != e.word("editor.cast.palette.help") {
					t.Fatal("spell grid, paged trap list or help is missing")
				}
				chromeClick(e, center(l.GridCells[0]))
				if s := mustListSpec(t, e); !e.addPicking() || !strings.Contains(strings.Join(s.Notes, " "), fmt.Sprintf(e.word("editor.place.placing"), "Choice 1")) {
					t.Fatalf("armed spell state %q", s.Notes)
				}
			})
			t.Run("map picker", func(t *testing.T) {
				e := modesWordsEditor(t, "full", lang, listWindowScreen)
				for i := 0; i < 30; i++ {
					name := fmt.Sprintf("Extra map %02d", i)
					e.maps = append(e.maps, EditorMap{Key: name, Label: name})
				}
				e.enterCatalog()
				s := mustListSpec(t, e)
				l := checkListWindow(t, e, popList, s)
				p := center(l.List.Row(1))
				e.Pointer(Input{CursorX: p.X, CursorY: p.Y, WheelY: -40}, time.Time{})
				paged := mustListSpec(t, e)
				after := checkListWindow(t, e, popList, paged)
				top, shown := paged.List.Visible()
				if top == 0 || shown >= paged.List.Len() || l.Title != after.Title || !slices.Equal(l.Buttons, after.Buttons) {
					t.Fatalf("page starts at %d, title %v, buttons %v", top, after.Title, after.Buttons)
				}
				_, maps := e.catalogRows()
				want := e.maps[maps[top]].Key
				chromeClick(e, center(after.List.Row(0)))
				chromeClick(e, center(after.Buttons[0]))
				if e.doc.Source != want {
					t.Fatalf("Open loaded %q, want %q", e.doc.Source, want)
				}
			})
			t.Run("groups", func(t *testing.T) {
				e := modesWordsEditor(t, "full", lang, listWindowScreen)
				e.showRecords(editorGroupsFilter)
				l := checkListWindow(t, e, popList, mustListSpec(t, e))
				if l.TitleHelp.Empty() || len(l.Notes) == 0 || len(l.Buttons) != 2 {
					t.Fatal("Groups lacks help, the unit count or the two edit actions")
				}
			})
			t.Run("selection", func(t *testing.T) {
				e := modesWordsEditor(t, "full", lang, listWindowScreen)
				e.setSelection([]RecordRef{{KindUnit, 0}, {KindUnit, 1}})
				s, _, ok := e.objectSpec()
				if !ok {
					t.Fatal("Selection window did not open")
				}
				checkListWindow(t, e, popObject, s)
			})
		})
	}
}

func TestEditorListButtonsUseNaturalWindowWidth(t *testing.T) {
	e, _ := newModesEditor(t, "full")
	e.Layout(listWindowScreen.X, listWindowScreen.Y)
	e.showRecords(editorScriptsFilter)
	s := mustListSpec(t, e)
	assertWidth := func(s popupSpec) {
		t.Helper()
		want := naturalWidth(&s, e.font, e.bodyFont(), 0)
		if got := layoutPopup(s, e.font, e.bodyFont()).Rect.Dx(); want >= 400 || s.Width != want || got != want {
			t.Errorf("%q asks for %d, laid out %d, natural %d", s.Title, s.Width, got, want)
		}
	}
	assertWidth(s)
	e.cancelPopup(popList)
	e.setSelection([]RecordRef{{KindUnit, 0}, {KindUnit, 1}})
	s, _, ok := e.objectSpec()
	if !ok {
		t.Fatal("Selection window did not open")
	}
	assertWidth(s)
}

func TestWindowedReportKeepsTitleAndShowsOutcomeWithTotals(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			host := &askingEdits{reportModEdits: reportModEdits{reportEdits{rows: reportRows, dirty: true}}}
			e := reportEditor(t, host)
			e.font = wordsFont(lang)
			e.SetWords(words.For(lang))
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.Layout(listWindowScreen.X, listWindowScreen.Y)
			assertReport := func(outcome string) {
				t.Helper()
				e.closeObject()
				s := mustListSpec(t, e)
				l := checkListWindow(t, e, popList, s)
				totals := fmt.Sprintf(e.word("editor.panel.reportvalue"), 3, 1)
				if s.Title != e.word("editor.side.report") || !slices.Contains(s.Notes, e.word(outcome)) ||
					!slices.Contains(s.Notes, totals) || l.Notes[0].Min.Y < l.List.Rect.Max.Y {
					t.Fatalf("report title %q and notes %q", s.Title, s.Notes)
				}
			}
			_ = e.Key("report")
			e.panels.state[panelBottom].Restored = false
			assertReport("editor.report.checked")
			chromeClick(e, center(layoutPopup(mustListSpec(t, e), e.font, e.bodyFont()).List.Row(1)))
			if e.ReportSelected() != 1 || !slices.Contains(mustListSpec(t, e).Notes, reportRows[1].Text) {
				t.Fatal("the clicked finding and its text were not selected")
			}
			e.refuseSave(&ReportRefusal{Findings: reportRows})
			assertReport("editor.report.refused")
			_ = e.Key("saveas")
			assertReport("editor.report.refused")
			if e.report.confirm == nil || len(host.written) != 0 {
				t.Fatal("save did not wait for confirmation")
			}
			if err := e.Key("enter"); err != nil || len(host.written) != 1 {
				t.Fatalf("confirmation wrote %v, error %v", host.written, err)
			}
			assertReport("editor.report.saved")
		})
	}
}

func TestEditorKeysFitTheFullSaveLineAndShareTheRowTip(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			target := strings.TrimSpace(strings.Repeat("long-mod-name ", 16))
			e := modSaveEditor(t, &modSaveEdits{target: target})
			e.font = wordsFont(lang)
			e.SetWords(words.For(lang))
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.Layout(listWindowScreen.X, listWindowScreen.Y)
			_ = e.Key("help")
			want := e.editorHostText(fmt.Sprintf(e.plainWord("editor.keys.modsave"), target))
			s := mustListSpec(t, e)
			last := s.List.Len() - 1
			if got := s.List.Rows()[last].Text; got != want {
				t.Fatalf("save line %q, want %q", got, want)
			}
			l := checkListWindow(t, e, popList, s)
			_, resized := edgeDrag(t, e, rightEdge, image.Pt(448-l.Rect.Dx(), 0))
			if resized.Dx() != 448 {
				t.Fatalf("resized list width %d", resized.Dx())
			}
			e.selectFinding(last)
			s = mustListSpec(t, e)
			l = checkListWindow(t, e, popList, s)
			top, _ := s.List.Visible()
			r := l.List.Row(last - top)
			if got := s.ListText(last, r.Dx()-2*listTextX); got == want || !strings.HasSuffix(got, textfit.Mark) {
				t.Fatalf("clipped save line %q", got)
			}
			e.tip.windows = []tipWindow{{s, l}}
			p := center(r)
			move := Input{CursorX: p.X, CursorY: p.Y}
			now := time.Unix(100, 0)
			e.Pointer(move, now)
			e.Pointer(move, now.Add((editorTipDelay-1)*time.Millisecond))
			if tip := e.TipShown(); tip != "" {
				t.Fatalf("early tip %q", tip)
			}
			e.Pointer(move, now.Add(editorTipDelay*time.Millisecond))
			if tip, ok := e.shownTip(); !ok || tip.Text != want || !tip.Rect.In(s.Bounds) {
				t.Fatalf("save tip %+v", tip)
			}
		})
	}
}

func requireRecordAboveList(t *testing.T, e *MapEditor) {
	t.Helper()
	_, object := objectLayout(t, e)
	list := listLayout(t, e)
	text.Record(func() { e.ChromeImage() })
	windows := e.ChromeWindows()
	index := func(r image.Rectangle) int {
		return slices.IndexFunc(windows, func(w ChromeWindow) bool { return w.Rect == r })
	}
	objectAt, listAt := index(object.Rect), index(list.Rect)
	if objectAt < 0 || listAt < 0 {
		t.Fatalf("list %v and record %v must both be drawn", list.Rect, object.Rect)
	}
	overlap := object.Rect.Intersect(list.Rect)
	if overlap.Empty() {
		t.Fatal("the fixture must overlap the list and record windows")
	}
	if objectAt <= listAt {
		t.Errorf("the list covers %d pixels of the record window", overlap.Dx()*overlap.Dy())
	}
	p := center(object.Title)
	if part, _ := object.hit(p); part != popupTitle {
		t.Fatalf("record title point hit part %v", part)
	}
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y, PrimaryDown: true}, time.Time{})
	if e.drag != popObject {
		t.Errorf("record title press dragged window %v, want the record window", e.drag)
	}
	e.Pointer(Input{CursorX: p.X, CursorY: p.Y}, time.Time{})
}

func TestScenarioRowOpensRecordAboveItsList(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			e := scriptsListEditor(t)
			e.font = wordsFont(lang)
			e.SetWords(words.For(lang))
			e.SetHostTextEncoder(mockupHostEncoder(e.font))
			e.Layout(listWindowScreen.X, listWindowScreen.Y)
			if err := e.Key("scripts"); err != nil {
				t.Fatal(err)
			}
			_, rows, _ := e.scenarioRows()
			row := slices.IndexFunc(rows, func(r treeRow) bool { return r.member && r.record == -2 })
			box, ok := e.FindControl(ControlQuery{Role: ControlListRow, N: row})
			if row < 0 || !ok {
				t.Fatal("no variable row in Scenario")
			}
			chromeClick(e, center(box))
			if e.Selected() != -2 {
				t.Fatalf("Scenario selected %d, want the variable", e.Selected())
			}
			requireRecordAboveList(t, e)
		})
	}
}

func TestFloatingReportRowOpensRecordAboveItsList(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			for _, grow := range []image.Point{{}, {300, 200}} {
				e := reportEditor(t, &reportEdits{rows: reportRows})
				e.font = wordsFont(lang)
				e.SetWords(words.For(lang))
				e.SetHostTextEncoder(mockupHostEncoder(e.font))
				e.Layout(listWindowScreen.X, listWindowScreen.Y)
				if err := e.Key("report"); err != nil {
					t.Fatal(err)
				}
				e.panels.state[panelBottom].Restored = false
				e.closeObject()
				if grow != (image.Point{}) {
					edgeDrag(t, e, bottomRight, grow)
				}
				box, ok := e.FindControl(ControlQuery{Role: ControlListRow, N: 1})
				if !ok {
					t.Fatal("no finding row in the report")
				}
				chromeClick(e, center(box))
				if e.ReportSelected() != 1 {
					t.Fatalf("Report selected row %d, want 1", e.ReportSelected())
				}
				requireRecordAboveList(t, e)
				if grow == (image.Point{}) {
					list := listLayout(t, e)
					_, record := objectLayout(t, e)
					p := center(list.Title)
					if p.In(record.Rect) || record.Rect.Max.X-2 <= list.Rect.Max.X {
						t.Fatal("the natural windows must each have an exposed edge")
					}
					chromeClick(e, p)
					if r, _ := e.PopupRect(); r != list.Rect {
						t.Fatalf("pressing the Report raised %v, want %v", r, list.Rect)
					}
					chromeClick(e, image.Pt(record.Rect.Max.X-2, center(record.Rect).Y))
					if r, _ := e.PopupRect(); r != record.Rect {
						t.Fatalf("pressing the record's visible edge raised %v, want %v", r, record.Rect)
					}
				}
			}
		})
	}
}
