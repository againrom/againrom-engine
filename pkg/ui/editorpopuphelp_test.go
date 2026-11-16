package ui

import (
	"image"
	"strings"
	"testing"

	"againrom/pkg/words"
)

// A row's help and the title's help are each a (?) mark the builder lays
// out beside the label or the close button; hovering a mark shows the help
// wrapped to several body lines, not cut, and the label stops before it.
func TestPopupHelpMarkShowsItsHelpWrapped(t *testing.T) {
	body := panelFont()
	help := strings.Repeat("A long explanation of what the field holds. ", 6)
	s := popupSpec{Title: "Window", TitleHelp: help, At: image.Pt(10, 10), Width: 400, Bounds: image.Rect(0, 0, 1280, 720),
		Rows: []popupRow{{Kind: popupEdit, Label: "Name", Help: help}, {Kind: popupEdit, Label: "Plain"}}}
	l := layoutPopup(s, body, body)
	if l.TitleHelp.Empty() || l.TitleHelp.Max.X > l.Close.Min.X {
		t.Fatalf("title mark %v beside close %v", l.TitleHelp, l.Close)
	}
	mark := l.Rows[0].Help
	if mark.Empty() || l.Rows[0].Label.Max.X > mark.Min.X || !l.Rows[1].Help.Empty() {
		t.Fatalf("row marks %v %v, label %v", mark, l.Rows[1].Help, l.Rows[0].Label)
	}
	for _, at := range []image.Point{center(mark), center(l.TitleHelp)} {
		tip, ok := popupTipAt(s.Bounds, body, body, s, l, at, true)
		if !ok || len(tip.Lines) < 2 || strings.Join(tip.Lines, "") != help {
			t.Fatalf("hover at %v: %v %q", at, ok, tip.Lines)
		}
		for _, line := range tip.Lines {
			if body.Advance(line) > tip.Rect.Dx()-8 {
				t.Fatalf("help line %q is cut in %v", line, tip.Rect)
			}
		}
	}
}

// Every page of the mod window has its title help in both languages, and
// the Missions page shows it and its keys behind the title's mark, not as
// lines in the window.
func TestModPagesSayWhatTheyAreForBehindTheTitleMark(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		book := words.For(lang)
		for _, p := range modPageWords {
			if got := book.Text(p.helpWord); got == p.helpWord || strings.TrimSpace(got) == "" {
				t.Errorf("%s: page %s has no help word %s", lang, p.tabWord, p.helpWord)
			}
		}
		e, _ := modMapActionsEditor(t, lang, image.Pt(1280, 800))
		s, _, _ := e.modSpec()
		want := e.editorHostText(e.plainWord("editor.mod.missions.help") + "\n" + e.plainWord("editor.mod.keys"))
		if s.TitleHelp != want {
			t.Errorf("%s: the Missions page's title help is %q", lang, s.TitleHelp)
		}
		for _, note := range s.Notes {
			if strings.Contains(note, e.editorHostText("Ctrl+S")) {
				t.Errorf("%s: the keys stand in the window: %q", lang, note)
			}
		}
	}
}
