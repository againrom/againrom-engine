package ui

import (
	"image"
	"reflect"
	"testing"

	"againrom/pkg/formats/textinput"
	"againrom/pkg/render/text"
)

// lookPage is a save page as the second game builds it: ruled rows, a name
// hint, row hints and a gold heading while the list has the focus.
func lookPage(p *pageSeamsFake) SavePage {
	page := testSavePage(p)
	page.Ruled, page.RowHint = true, true
	page.NameHint = "ENTER A NAME"
	page.NameBox = image.Rect(120, 98, 527, 122)
	page.List = image.Rect(120, 160, 503, 160)
	page.Rows = 7
	page.EmptyInk = loadDisabledText
	page.Texts = []PageText{{Text: "Saved Games", At: image.Pt(120, 134), Focus: true, Listing: true}}
	return page
}

func lookApp(t *testing.T, fake *pageSeamsFake) *App {
	t.Helper()
	font := chargenTestFont()
	a := newTestApp(t, appRows(1), okLoader(t))
	a.SetWords(AuthoredWords(), font, func(r rune) (byte, bool) { return textinput.EncodeRune(r, font.Selector) })
	a.SetTooltipFont(font)
	seams := fake.seams()
	seams.CanDelete = func(dir, name string) bool { return true }
	seams.PrepareDelete = func(dir, name string) (func() error, error) {
		return func() error {
			for i, e := range fake.entries {
				if e.Name == name {
					fake.entries = append(fake.entries[:i], fake.entries[i+1:]...)
					break
				}
			}
			return nil
		}, nil
	}
	a.SetSaveDialogSeams(seams)
	page := lookPage(fake)
	a.SetSavePages(SavePages{Load: firstLoadPage(LoadWindowWords{}), Save: &page})
	openTestSavePage(t, a)
	return a
}

func headingColor(t *testing.T, a *App) (got bool, gold bool) {
	t.Helper()
	calls := text.Record(func() {
		if _, err := a.composePage(a.flow.pages.save.view()); err != nil {
			t.Fatal(err)
		}
	})
	for _, c := range calls {
		if c.Y == 134 && c.X == 120 {
			return true, c.Color == loadSelectedText
		}
	}
	return false, false
}

func TestSavePageHeadingIsGoldWhileTheListHasTheFocus(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0000.sav", Label: "Alpha"}}}
	a := lookApp(t, fake)
	if drawn, gold := headingColor(t, a); !drawn || gold {
		t.Fatalf("heading drawn %v gold %v on a fresh page, want drawn and not gold", drawn, gold)
	}
	v := a.flow.pages.save.view()
	a.stepPage(v, appInput{Down: true})
	if drawn, gold := headingColor(t, a); !drawn || !gold {
		t.Fatalf("heading drawn %v gold %v after a list key, want gold", drawn, gold)
	}
	a.stepPage(v, appInput{Typed: "x"})
	if _, gold := headingColor(t, a); gold {
		t.Fatal("heading stays gold after typing in the name box")
	}
}

func TestSavePageRowDetailAndNameHintAreHoverCaptions(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{{Name: "game0000.sav", Label: "Alpha", Detail: "Alpha  01.02.2003  04:05:06"}}}
	a := lookApp(t, fake)
	a.flow.pages.save.list.Select(1)
	if got := a.flow.pageMessage(a.flow.pages.save.view()); got != "" {
		t.Fatalf("line under the list %q, want none", got)
	}
	a.tooltipPoint = image.Pt(130, 105)
	if got := a.tooltipTarget().lines; !reflect.DeepEqual(got, []string{"ENTER A NAME"}) {
		t.Fatalf("name box caption %q", got)
	}
	box := a.pageListBox(a.flow.pages.save.view())
	a.tooltipPoint = box.Row(1).Min.Add(image.Pt(10, 5))
	if got := a.tooltipTarget().lines; !reflect.DeepEqual(got, []string{"Alpha  01.02.2003  04:05:06"}) {
		t.Fatalf("row caption %q", got)
	}
	a.tooltipPoint = box.Row(0).Min.Add(image.Pt(10, 5))
	if got := a.tooltipTarget().lines; got != nil {
		t.Fatalf("Empty Slot row caption %q, want none", got)
	}
}

func TestSavePageListIsRuledAndTheFirstGamesIsNot(t *testing.T) {
	fake := &pageSeamsFake{}
	a := lookApp(t, fake)
	if box := a.pageListBox(a.flow.pages.save.view()); !box.Ruled || !box.Dark || box.Rows != 7 {
		t.Fatalf("save list ruled %v dark %v rows %d", box.Ruled, box.Dark, box.Rows)
	}
	first := pageView{page: firstLoadPage(LoadWindowWords{})}
	if box := a.pageListBox(first); box.Ruled || box.Dark {
		t.Fatalf("first game's list ruled %v dark %v", box.Ruled, box.Dark)
	}
	if rows := a.HeadlessRows(); len(rows) == 0 || rows[0].Ink != loadDisabledText {
		t.Fatalf("Empty Slot row ink %v", rows)
	}
}

func TestSavePageDeleteLeavesTheEmptySlotChosenAndTheNameEmpty(t *testing.T) {
	fake := &pageSeamsFake{entries: []SaveEntry{
		{Name: "game0001.sav", Label: "Alpha"}, {Name: "game0000.sav", Label: "Beta"}}}
	a := lookApp(t, fake)
	if err := a.HeadlessSaveSelect(2); err != nil {
		t.Fatal(err)
	}
	if got := string(a.flow.pages.save.name); got != "Beta" {
		t.Fatalf("name %q before the delete", got)
	}
	v := a.flow.pages.save.view()
	a.flow.confirmDeletePage(v)
	if !v.ui.confirm {
		t.Fatal("Delete did not ask")
	}
	a.acceptPage(v)
	s := a.flow.pages.save
	if s.list.Selection() != 0 || len(s.name) != 0 {
		t.Fatalf("after Delete and Yes: row %d name %q, want row 0 and an empty name", s.list.Selection(), string(s.name))
	}
	if rows := a.HeadlessRows(); len(rows) != 2 {
		t.Fatalf("rows %d after the delete, want Empty Slot and one save", len(rows))
	}
}
