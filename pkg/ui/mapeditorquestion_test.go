package ui

import (
	"go/ast"
	"go/token"
	"image"
	"strconv"
	"strings"
	"testing"
)

// Every question window at 1920x1080 holds the layout properties: inside
// the screen, whole labels, each element hit at its place, its notes,
// help and list rows inside, every note line whole inside its box past the
// draw inset; a click on the close mark and one on the last button close it
// through the editor's input.
func TestQuestionWindowsHoldTheLayoutProperties(t *testing.T) {
	size := image.Pt(1920, 1080)
	eachQuestion(t, func(t *testing.T, e *MapEditor, name string, s popupSpec) {
		checkWindowProperties(t, e, s, size)
		body := e.bodyFont()
		l := layoutPopup(s, e.font, body)
		for _, b := range append([]image.Rectangle{l.TitleHelp}, l.Notes...) {
			if !b.Empty() && !b.In(l.Rect) {
				t.Errorf("%v leaves the window %v", b, l.Rect)
			}
		}
		for i, r := range l.Notes {
			if w := body.Advance(s.Notes[i]) + 2*popupTextInset; w > r.Dx() {
				t.Errorf("note line %q is %d wide in a %d wide box", s.Notes[i], w, r.Dx())
			}
		}
		if l.HasList {
			for i := range l.List.Rows {
				r := l.List.Row(i)
				if part, row := l.hit(r.Min.Add(r.Size().Div(2))); !r.In(l.Rect) || part != popupListRow || row != i {
					t.Errorf("list row %d at %v gives part %d row %d", i, r, part, row)
				}
			}
			r := l.List.Row(3)
			modesClick(e, r.Min.Add(r.Size().Div(2)))
			if e.prompt.group.list.Selection() != 3 {
				t.Errorf("a click on list row 3 chose %d", e.prompt.group.list.Selection())
			}
		}
		modesClick(e, l.Close.Min.Add(l.Close.Size().Div(2)))
		if e.prompt.open() {
			t.Fatal("a click on the close mark left the window open")
		}
		for _, w := range questionWindows() {
			if w.name == name {
				w.open(e)
			}
		}
		s, _ = e.promptSpec()
		l = layoutPopup(s, e.font, body)
		last := l.Buttons[len(l.Buttons)-1]
		modesClick(e, last.Min.Add(last.Size().Div(2)))
		if e.prompt.open() {
			t.Fatal("a click on the last button left the window open")
		}
	})
}

// questionBuilders are the question windows' builders, by file.
var questionBuilders = map[string][]string{
	"mapeditorquestion.go":   {"questionSpec", "spec"},
	"mapeditorleave.go":      {"leaveSpec"},
	"mapeditormappanel.go":   {"noticeSpec", "noticeNotes", "newModeSpec"},
	"mapeditorscriptform.go": {"scriptFormSpec"},
	"mapeditorareaedit.go":   {"areaTriggerSpec", "areaSharedSpec"},
	"mapeditorgroups.go":     {"toGroupSpec", "groupWarnSpec"},
	"editortestparty.go":     {"testPartySpec"},
	"mapeditorplayers.go":    {"playerDeleteSpec"},
}

// The question builders hold no text of their own.
func TestQuestionBuildersHoldNoEnglishLiteral(t *testing.T) {
	checkNoEnglishLiteral(t, questionBuilders)
}

// joinsWord reports whether lit is a literal of more than spaces and other
// holds a call of the editor's word.
func joinsWord(lit, other ast.Expr) bool {
	b, ok := lit.(*ast.BasicLit)
	if !ok || b.Kind != token.STRING {
		return false
	}
	if s, err := strconv.Unquote(b.Value); err != nil || strings.TrimSpace(s) == "" {
		return false
	}
	word := false
	ast.Inspect(other, func(n ast.Node) bool {
		if c, ok := n.(*ast.CallExpr); ok {
			if sel, ok := c.Fun.(*ast.SelectorExpr); ok && (sel.Sel.Name == "word" || sel.Sel.Name == "plainWord") {
				word = true
			}
		}
		return !word
	})
	return word
}
