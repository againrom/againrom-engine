package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"image"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode"

	"againrom/pkg/pathdialog"
)

// promptsHost is the views fixture's host with the save window's switch and
// a sack's recipe items, whose pictures its item looks give.
type promptsHost struct {
	*viewsHost
	inside bool
}

func (h *promptsHost) TextsInside() bool                         { return h.inside }
func (h *promptsHost) SetTextsInside(on bool) error              { h.inside = on; return nil }
func (h *promptsHost) AddStock(int) (*InspectionDocument, error) { return nil, nil }
func (h *promptsHost) AddItemRecipe(RecordKind, int, int64) (*InspectionDocument, error) {
	return nil, nil
}
func (h *promptsHost) RecordChoices(kind RecordKind, _ int, key string) []InspectionChoice {
	if kind != KindSack || key != "recipeitem" {
		return nil
	}
	return []InspectionChoice{{1, "Item 1: Sword"}, {2, "Item 2: Mace"}}
}

// promptsEditor is the views editor in language lang at size over a
// promptsHost.
func promptsEditor(t *testing.T, lang string, size image.Point) *MapEditor {
	t.Helper()
	e, h := viewsEditor(t, lang)
	e.doc.Edits = &promptsHost{viewsHost: h}
	e.Layout(size.X, size.Y)
	return e
}

// promptScreens open the nine field prompt windows by their own routes.
var promptScreens = []struct {
	name string
	open func(e *MapEditor)
}{
	{"field", func(e *MapEditor) { e.selected = viewsStructure; _ = e.Key("field") }},
	{"open", func(e *MapEditor) { e.pathFallback(pathdialog.OpenFile, pathdialog.ErrUnavailable) }},
	{"save", func(e *MapEditor) {
		e.pathFallback(pathdialog.SaveFile, pathdialog.ErrUnavailable)
		e.TextInput("81.alm")
	}},
	{"newmap", func(e *MapEditor) { e.openPrompt(editorPrompt{kind: promptRectangle, path: "80x80"}) }},
	{"recipe", func(e *MapEditor) { e.selected = viewsSack; _ = e.Key("recipe") }},
	{"speaker", func(e *MapEditor) { e.askSpeaker(0, "", nil) }},
	{"forest", func(e *MapEditor) { e.openPrompt(editorPrompt{kind: promptForestShare, path: "40"}) }},
	{"seed", func(e *MapEditor) {
		e.openPrompt(editorPrompt{kind: promptSeed, scope: "randomterrain", path: "1"})
	}},
	{"message", func(e *MapEditor) {
		e.message = strings.TrimSpace(strings.Repeat(e.plainWord("editor.recipe.created")+" ", 120))
	}},
}

// Each field prompt window, in English and Russian, keeps every element
// inside the window at 640x480, 800x600 and 1920x1080, every label whole at
// 1920x1080, and hit returns each element at its rectangle.
func TestFieldPromptsFitTheirWindowAtThreeSizes(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		for _, screen := range promptScreens {
			for _, size := range []image.Point{{640, 480}, {800, 600}, {1920, 1080}} {
				t.Run(lang+"/"+screen.name+"/"+strconv.Itoa(size.X), func(t *testing.T) {
					e := promptsEditor(t, lang, size)
					screen.open(e)
					s, ok := e.promptSpec()
					if !ok {
						t.Fatal("no window")
					}
					checkWindowProperties(t, e, s, size)
				})
			}
		}
	}
}

// Answer 1: a field with choices shows a drop-down choice row, no "Up, Down"
// in its title; a press opens the list and a press on a row chooses it. A
// field whose choices all carry pictures shows them as a picture grid whose
// cells carry the names as tips; a press on a cell chooses it.
func TestFieldWithChoicesShowsAChoiceRow(t *testing.T) {
	e := promptsEditor(t, "", image.Pt(1280, 800))
	e.selected = viewsStructure
	_ = e.Key("field")
	s, ok := e.promptSpec()
	if !ok || len(s.Rows) != 1 || s.Rows[0].Kind != popupChoice || !strings.Contains(s.Rows[0].Value, "Hall") {
		t.Fatalf("the kind field's rows %+v", s.Rows)
	}
	if strings.Contains(s.Title, "Up") || strings.Contains(s.Title, "/") {
		t.Fatalf("the title %q names keys", s.Title)
	}
	l := layoutPopup(s, e.font, e.bodyFont())
	modesClick(e, center(l.Rows[0].Control))
	d, ok := e.promptDropdown()
	if !ok || !d.Open {
		t.Fatal("a press on the choice row opens no list")
	}
	b, row := d.Bounds(e.bodyFont()), image.Point{}
	for y := b.Min.Y; y < b.Max.Y && row == (image.Point{}); y++ {
		if part, n := d.hit(e.bodyFont(), image.Pt(b.Min.X+8, y)); part == dropdownRow && n == 1 {
			row = image.Pt(b.Min.X+8, y)
		}
	}
	modesClick(e, row)
	if d, _ := e.promptDropdown(); e.prompt.path != "52" || d.Open {
		t.Fatalf("a press on the second row: path %q list open %v", e.prompt.path, d.Open)
	}

	e.leave(leavePrompt)
	e.selected = viewsSack
	_ = e.Key("recipe")
	s, ok = e.promptSpec()
	if !ok || s.Grid == nil || len(s.Grid.Cells) != 2 || s.Grid.Cells[1].Tip != "Item 2: Mace" || s.Grid.Cells[0].Icon == nil {
		t.Fatalf("the recipe window shows no picture grid: %+v", s.Grid)
	}
	l = layoutPopup(s, e.font, e.bodyFont())
	modesClick(e, center(l.GridCells[1]))
	if s, _ = e.promptSpec(); e.prompt.path != "2" || !s.Grid.Cells[1].Pressed || len(s.Rows[0].Icons) != 1 {
		t.Fatalf("a press on the second cell: path %q", e.prompt.path)
	}
}

// Answer 2: a record field's prompt title is the plain view's word for the
// field.
func TestRecordFieldPromptTitleIsThePlainWord(t *testing.T) {
	for lang, want := range map[string]string{"": "Gold", "russian": "Золото"} {
		e := promptsEditor(t, lang, image.Pt(1280, 800))
		e.selected = viewsSack
		_ = e.Key("field")
		s, ok := e.promptSpec()
		if !ok || e.prompt.kind != promptField || e.plainWord("editor.plain.gold") != want || s.Title != e.word("editor.plain.gold") {
			t.Fatalf("%q: the gold field's title %q", lang, s.Title)
		}
	}
}

// Answer 3: when the system dialog did not open the typed path window opens
// and its note says the dialog did not open, with no key or instruction.
func TestPathFallbackNoteSaysTheDialogDidNotOpen(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e := promptsEditor(t, lang, image.Pt(1280, 800))
		e.pathFallback(pathdialog.OpenFile, pathdialog.ErrUnavailable)
		s, ok := e.promptSpec()
		note := strings.Join(s.Notes, " ")
		if !ok || e.prompt.kind != promptOpen || note != e.word("editor.path.nodialog") || s.Title != e.word("editor.prompt.open") {
			t.Fatalf("%q: title %q notes %q", lang, s.Title, s.Notes)
		}
	}
}

// Answer 4: the save window's texts-inside switch is a labelled row whose
// (?) holds the .almx rule; the path row has its label.
func TestSaveWindowSwitchIsALabelledRow(t *testing.T) {
	for _, lang := range []string{"", "russian"} {
		e := promptsEditor(t, lang, image.Pt(1280, 800))
		e.openPrompt(editorPrompt{kind: promptSave, path: "81.alm"})
		row, _, ok := textsInsideSwitch(e)
		s, _ := e.promptSpec()
		if !ok || row.Label != e.word("editor.menu.textsinside") || row.Help != e.word("editor.save.textsinside.almx") || row.Value != e.word("editor.switch.off") {
			t.Fatalf("%q: the switch row %+v", lang, row)
		}
		if s.Rows[0].Label != e.word("editor.prompt.path") || strings.Contains(s.TitleHelp, ".almx") {
			t.Fatalf("%q: path row %q help %q", lang, s.Rows[0].Label, s.TitleHelp)
		}
	}
}

// Answer 5: the new map size is one field holding 80x80 under the window's
// name; its help states the side limits.
func TestNewMapSizeIsOneField(t *testing.T) {
	e := promptsEditor(t, "russian", image.Pt(1280, 800))
	e.openPrompt(editorPrompt{kind: promptRectangle, path: "80x80"})
	s, ok := e.promptSpec()
	if !ok || len(s.Rows) != 1 || s.Rows[0].Edit == nil || s.Rows[0].Value != "80x80" {
		t.Fatalf("the new map rows %+v", s.Rows)
	}
	if s.Title != e.word("editor.prompt.newmap") || !strings.HasPrefix(s.TitleHelp, e.word("editor.prompt.newmap.help")) {
		t.Fatalf("title %q help %q", s.Title, s.TitleHelp)
	}
}

// Answer 6: the kept line shows under an error that left the map as it was
// and under no other message.
func TestKeptLineOnlyUnderAKeptError(t *testing.T) {
	kept := func(e *MapEditor) bool {
		s, _ := e.promptSpec()
		for _, n := range s.Notes {
			if n == e.word("editor.message.kept") {
				return true
			}
		}
		return false
	}
	e := promptsEditor(t, "russian", image.Pt(1280, 800))
	e.pathFallback(pathdialog.SaveFile, pathdialog.ErrUnavailable)
	if kept(e) {
		t.Fatal("the kept line shows under the path fallback")
	}
	e.leave(leavePrompt)
	e.message = e.plainWord("editor.recipe.created")
	if kept(e) {
		t.Fatal("the kept line shows under a report")
	}
	e.message = ""
	e.openPrompt(editorPrompt{kind: promptSave, path: filepath.Join(t.TempDir(), "kept.alm")})
	if err := e.Key("enter"); err == nil || e.message != "no writer" || !kept(e) {
		t.Fatalf("a refused save: %v message %q kept %v", err, e.message, kept(e))
	}
}

// No field prompt window builder and no editor message holds an English
// literal: every caption, instruction and button text is a word.
func TestFieldPromptBuildersHoldNoEnglishLiteral(t *testing.T) {
	builders := map[string]bool{
		"promptSpecBase": false, "promptTitle": false, "promptHelp": false, "promptRows": false,
		"promptChoiceRows": false, "fieldTitle": false, "recipeItemField": false, "textsInsideRow": false,
		"pathFallback": false, "seedTitle": false, "keptError": false, "setItemRecipe": false, "keyRecords": false,
	}
	english := func(n ast.Node, where string, fset *token.FileSet) {
		ast.Inspect(n, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			s, _ := strconv.Unquote(lit.Value)
			letters, upper := false, false
			for _, r := range s {
				letters = letters || unicode.IsLetter(r)
				upper = upper || unicode.IsUpper(r)
			}
			if letters && (upper || strings.Contains(s, " ")) {
				t.Errorf("%s: %s holds the literal %s", fset.Position(lit.Pos()), where, lit.Value)
			}
			return true
		})
	}
	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv != nil {
				if _, ok := builders[fn.Name.Name]; ok {
					builders[fn.Name.Name] = true
					english(fn.Body, fn.Name.Name, fset)
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			a, ok := n.(*ast.AssignStmt)
			if !ok {
				return true
			}
			for i, l := range a.Lhs {
				if sel, ok := l.(*ast.SelectorExpr); ok && sel.Sel.Name == "message" && i < len(a.Rhs) {
					if x, ok := sel.X.(*ast.Ident); ok && x.Name == "e" {
						english(a.Rhs[i], "an editor message", fset)
					}
				}
			}
			return true
		})
	}
	for name, found := range builders {
		if !found {
			t.Errorf("the builder %s is not found", name)
		}
	}
}
