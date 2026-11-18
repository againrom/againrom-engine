package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
)

// countedOverview is an overview whose count lines open their pages, with
// the game's check folded under its heading.
var countedOverview = ModOverviewPage{Sections: []ModOverviewSection{
	{Heading: "What the mod changes", Lines: []string{"Terrain: 1", "Texts: 1", "Missions: 1"}, Pages: []string{ModTerrainScope, ModTextsScope, ModMissionsPage}},
	{Heading: "Check details", Lines: []string{"1 mod(s) active on rom1-en"}, Fold: true},
}}

// pagesEditor is answersEditor with the counted overview and a mission name
// longer than its row.
func pagesEditor(t *testing.T, lang string) (*MapEditor, *modesCalls) {
	t.Helper()
	e, calls := answersEditor(t, lang)
	host := e.doc.Edits.(everyPageHost)
	host.overview = countedOverview
	for i := range host.fields {
		if host.fields[i].Key == "source" {
			host.fields[i].Choices[0].Label = strings.Repeat("The long road over the northern pass ", 8) + "(25)"
		}
	}
	_ = e.Key("package")
	return e, calls
}

// modLayout is W's spec and layout as shown.
func modLayout(t *testing.T, e *MapEditor) (popupSpec, popupLayout) {
	t.Helper()
	s, _, ok := e.modSpec()
	if !ok {
		t.Fatal("no window spec")
	}
	return s, layoutPopup(s, e.font, e.bodyFont())
}

// Each page of W at 1920x1080, in English and in Russian, keeps every part
// inside the screen, shows each label whole or cut with its whole text in
// its tip, and answers hit at each rectangle.
func TestModPagesHaveTheirLayoutPropertiesOnTheTargetScreen(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, _ := pagesEditor(t, lang)
		pages := e.modPages()
		if len(pages) != 8 {
			t.Fatalf("%s: W shows %d pages, want 8: %v", lang, len(pages), pages)
		}
		for k, n := range pages {
			e.turnModPage(k)
			modPageProperties(t, e, image.Rect(0, 0, 1920, 1080), fmt.Sprintf("%s page %d", lang, n))
		}
	}
}

// A press through the input path on each tab shows its page; on an
// overview count line opens the page it counts; on a list row chooses that
// thing; on a Texts language row's Delete reaches the host with that
// language's text.
func TestModPagesAnswerPressesThroughTheInputPath(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		e, calls := pagesEditor(t, lang)
		pages := e.modPages()
		for n := range pages {
			_, l := modLayout(t, e)
			if n >= len(l.Tabs) {
				t.Fatalf("%s: %d tabs for %d pages", lang, len(l.Tabs), len(pages))
			}
			modesClick(e, boxMidpoint(l.Tabs[n]))
			if e.modPage != pages[n] {
				t.Errorf("%s: tab %d shows page %d, want %d", lang, n, e.modPage, pages[n])
			}
		}

		for _, want := range []struct {
			line string
			page int
		}{{"Terrain: 1", modPageTerrain}, {"Texts: 1", modPageTexts}, {"Missions: 1", modPageFields}} {
			turnTo(t, e, modPageOverview)
			s, l := modLayout(t, e)
			pressed := false
			for _, r := range l.Rows {
				if s.Rows[r.Index].Kind == popupLink && s.Rows[r.Index].Tip == want.line {
					modesClick(e, boxMidpoint(r.Label))
					pressed = true
					break
				}
			}
			if !pressed {
				t.Fatalf("%s: the overview has no link %q: %+v", lang, want.line, s.Rows)
			}
			if e.modPage != want.page {
				t.Errorf("%s: the link %q shows page %d, want %d", lang, want.line, e.modPage, want.page)
			}
		}

		turnTo(t, e, modPageItems)
		s, l := modLayout(t, e)
		pressed := false
		for _, r := range l.Rows {
			if row := s.Rows[r.Index]; row.Column == 0 && row.Label == "Weapon Pike" && !r.Label.Empty() {
				modesClick(e, boxMidpoint(r.Label))
				pressed = true
			}
		}
		if !pressed {
			t.Fatalf("%s: the Items list has no row Weapon Pike: %+v", lang, s.Rows)
		}
		s, _ = modLayout(t, e)
		chosen := ""
		for _, r := range s.Rows {
			if r.Column == 1 && len(r.Buttons) > 0 {
				chosen = r.Label
				break
			}
		}
		if chosen != "Weapon Pike" {
			t.Errorf("%s: after a press on its list row the chosen thing is %q", lang, chosen)
		}

		turnTo(t, e, modPageTexts)
		s, l = modLayout(t, e)
		pressed = false
		for _, r := range l.Rows {
			if row := s.Rows[r.Index]; strings.HasPrefix(row.Label, "Russian") && len(r.Buttons) == 1 {
				modesClick(e, boxMidpoint(r.Buttons[0]))
				pressed = true
				break
			}
		}
		if !pressed {
			t.Fatalf("%s: the Russian row of the text has no Delete: %+v", lang, s.Rows)
		}
		if got := strings.Join(calls.lines, "\n"); !strings.Contains(got, "drop.ru:greet") || strings.Contains(got, "drop.en:greet") {
			t.Errorf("%s: Delete on the Russian row ran: %s", lang, got)
		}
	}
}
