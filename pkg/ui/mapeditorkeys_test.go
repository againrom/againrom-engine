package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// helpTokens splits a help line into the words a key can be named by.
func helpTokens(line string) map[string]bool {
	out := map[string]bool{}
	for _, f := range strings.FieldsFunc(line, func(r rune) bool { return r == ' ' || r == '/' || r == ':' || r == ',' }) {
		out[f] = true
	}
	return out
}

// keyTableProblems lists what is wrong with rows: a key with no action or
// two rows, a row in neither the help nor the unlisted marks or in both, and
// a help line that does not name the row's key.
func keyTableProblems(rows []keyRow) []string {
	var out []string
	seen := map[string]bool{}
	for _, r := range rows {
		if r.key != "" {
			if r.act == nil {
				out = append(out, "key "+r.key+" has no action")
			}
			if seen[r.key] {
				out = append(out, "key "+r.key+" has two rows")
			}
			seen[r.key] = true
		}
		if len(r.help) == 0 && r.unlisted == "" {
			out = append(out, "key "+r.key+" is neither in the help nor marked unlisted")
		}
		if len(r.help) > 0 && r.unlisted != "" {
			out = append(out, "key "+r.key+" is in the help and marked unlisted")
		}
		for _, h := range r.help {
			if !helpTokens(h.line)[r.label] {
				out = append(out, "help line \""+h.line+"\" does not name key "+r.key+" as "+r.label)
			}
		}
	}
	return out
}

// TestMapEditorKeyTableRowsAreComplete: the table has no problem
// keyTableProblems knows.
func TestMapEditorKeyTableRowsAreComplete(t *testing.T) {
	for _, p := range keyTableProblems(editorKeyRows) {
		t.Error(p)
	}
}

// TestMapEditorKeyTableCheckFindsEachProblem is the loss control of the check
// above: a table with each fault is refused.
func TestMapEditorKeyTableCheckFindsEachProblem(t *testing.T) {
	act := func(*MapEditor, string) error { return nil }
	line := []helpEntry{{screenKeys, "X: do it"}}
	for name, rows := range map[string][]keyRow{
		"no action":    {{key: "x", label: "X", help: line}},
		"two rows":     {{key: "x", label: "X", act: act, help: line}, {key: "x", label: "X", act: act, help: line}},
		"nowhere":      {{key: "x", label: "X", act: act}},
		"both":         {{key: "x", label: "X", act: act, help: line, unlisted: "why"}},
		"wrong key":    {{key: "y", label: "Y", act: act, help: line}},
		"fine, listed": nil,
	} {
		got := keyTableProblems(rows)
		if (name == "fine, listed") != (len(got) == 0) {
			t.Errorf("%s: problems %v", name, got)
		}
	}
	if got := keyTableProblems([]keyRow{{key: "x", label: "X", act: act, help: line}, {key: "u", label: "U", act: act, unlisted: "why"}}); len(got) != 0 {
		t.Errorf("a sound table was refused: %v", got)
	}
}

// handledKeyNames are the key names the window's own code in dir passes to
// Key: the calls and the name column of the table Update reads the keyboard
// through.
func handledKeyNames(t *testing.T, dir string) []string {
	t.Helper()
	names := map[string]bool{}
	var files []string
	for _, pat := range []string{"mapeditor*.go", "editor*.go"} {
		m, err := filepath.Glob(filepath.Join(dir, pat))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, m...)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		str := func(x ast.Expr) (string, bool) {
			if l, ok := x.(*ast.BasicLit); ok && l.Kind == token.STRING {
				s, err := strconv.Unquote(l.Value)
				return s, err == nil
			}
			return "", false
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if sel, ok := n.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Key" && len(n.Args) == 1 {
					if s, ok := str(n.Args[0]); ok {
						names[s] = true
					}
				}
			case *ast.CompositeLit:
				// {ebiten.KeyEscape, "escape"}: the keyboard table in Update.
				if len(n.Elts) == 2 {
					if sel, ok := n.Elts[0].(*ast.SelectorExpr); ok && strings.HasPrefix(sel.Sel.Name, "Key") {
						if s, ok := str(n.Elts[1]); ok {
							names[s] = true
						}
					}
				}
			}
			return true
		})
	}
	var out []string
	for n := range names {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// TestMapEditorEveryKeyTheWindowSendsHasARow: a key name the keyboard table or
// a rail button passes to Key is a row of the table.
func TestMapEditorEveryKeyTheWindowSendsHasARow(t *testing.T) {
	names := handledKeyNames(t, ".")
	if len(names) < 20 {
		t.Fatalf("found only %d key names in the window's code: %v", len(names), names)
	}
	for _, n := range names {
		if editorKeyByName[n] == nil {
			t.Errorf("the window sends key %q and the table has no row for it", n)
		}
	}
}

// TestMapEditorKeyScanFindsASentKey is the loss control of the scan above: a
// file that sends a key by call and by keyboard table is read.
func TestMapEditorKeyScanFindsASentKey(t *testing.T) {
	dir := t.TempDir()
	src := "package ui\nfunc (e *MapEditor) sneak() {\n\t_ = e.Key(\"sneaked\")\n\tfor _, k := range []struct{ key ebiten.Key; name string }{{ebiten.KeyQ, \"sneaked2\"}} { _ = k }\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "mapeditorsneak.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := handledKeyNames(t, dir); !reflect.DeepEqual(got, []string{"sneaked", "sneaked2"}) {
		t.Errorf("scan found %v", got)
	}
}

// helpEntries lists the keys the table names on a help screen.
func helpEntries(screen helpScreen) []keyRow {
	var out []keyRow
	for _, r := range editorKeyRows {
		for _, h := range r.help {
			if h.screen == screen {
				out = append(out, r)
				break
			}
		}
	}
	return out
}

// TestMapEditorEveryKeyTheHelpNamesActs: reach each help screen with real
// keys; each key the screen names, pressed there, changes what the window
// shows or does.
func TestMapEditorEveryKeyTheHelpNamesActs(t *testing.T) {
	entry := map[helpScreen][]string{
		screenKeys:        {"ui:select0"},
		screenClone:       {"ui:select0", "clone"},
		screenDrop:        {"drop"},
		screenGoal:        {"goals", "enter"},
		screenGoals:       {"goals"},
		screenTerrain:     {"terrain"},
		screenHeight:      {"height"},
		screenObjects:     {"objects"},
		screenPassability: {"passability"},
		screenPlayers:     {"players"},
		screenPrompt:      {"ui:select0", "field"},
		screenMove:        {"ui:select0", "ui:pick-move"},
		screenAdd:         {"unit", "escape"},
		screenAddCell:     {"unit", "enter", "enter", "ui:add-pick"},
	}
	for screen := helpScreen(0); screen < screenCount; screen++ {
		rows := helpEntries(screen)
		if len(rows) == 0 {
			t.Errorf("screen %d has no keys", screen)
		}
		for _, r := range rows {
			if r.key == "" {
				continue
			}
			e, calls := newModesEditor(t, "full")
			for _, s := range entry[screen] {
				modesAct(e, s)
			}
			if got := e.helpScreen(); got != screen {
				t.Fatalf("path %v reached help screen %d, want %d", entry[screen], got, screen)
			}
			if r.key == "up" {
				modesAct(e, "down") // the first row has nothing above it
			}
			before := modesProbe(e, calls, "", true)
			err := modesAct(e, r.key)
			if modesProbe(e, calls, err, true) == before {
				t.Errorf("screen %d names %q and pressing it there changes nothing", screen, r.key)
			}
		}
	}
}

// idleMode reports that no layer of the mode is set. The list's wheel
// remainder belongs to the catalogue and the report panel only.
func idleMode(e *MapEditor, wheel bool) bool {
	m := e.editorMode
	return !m.catalog && m.prompt == (editorPrompt{}) && m.pick == pickNone && m.brush.mode == brushOff && m.brush.stroke == nil && !m.brush.down && reflect.DeepEqual(m.report, reportState{}) && (!wheel || e.listWheel == 0)
}

// TestMapEditorLeavingAModeClearsWhatItOwned: from each mode, each way out
// leaves the idle mode; the catalogue and the report panel, which own the
// list, also drop the wheel's remainder.
func TestMapEditorLeavingAModeClearsWhatItOwned(t *testing.T) {
	escapes := []string{"escape", "escape", "escape", "escape"}
	cases := []struct {
		name  string
		enter []string
		list  bool // the mode owns the list: turn the wheel in it, expect it dropped
		exit  []string
		busy  bool // the entry path itself ends back at the idle mode
	}{
		{"open prompt", []string{"open", "ui:text-7"}, false, escapes, false},
		{"save prompt", []string{"saveas", "ui:text-7"}, false, escapes, false},
		{"field prompt", []string{"ui:select0", "field", "tab"}, false, escapes, false},
		{"player name prompt", []string{"players", "field"}, false, escapes, false},
		{"new sack window", []string{"sack", "escape", "ui:text-7"}, false, escapes, false},
		{"new unit window", []string{"unit", "escape", "ui:text-7"}, false, escapes, false},
		{"new unit cell pick", []string{"unit", "enter", "enter", "ui:add-pick"}, false, escapes, false},
		{"new unit completes", []string{"unit", "enter", "enter", "ui:add-pick", "ui:click-cell", "enter"}, false, nil, true},
		{"field completes", []string{"ui:select0", "field", "ui:text-7", "enter"}, false, nil, true},
		{"clone pick", []string{"ui:select0", "clone"}, false, escapes, false},
		{"clone completes", []string{"ui:select0", "clone", "ui:click-cell"}, false, nil, true},
		{"sack pick", []string{"sack"}, false, escapes, false},
		{"unit pick", []string{"unit"}, false, escapes, false},
		{"drop pick", []string{"drop"}, false, escapes, false},
		{"drop completes", []string{"drop", "ui:click-cell"}, false, nil, true},
		{"goal pick", []string{"goals", "enter"}, true, escapes, false},
		{"script panel", []string{"goals"}, true, escapes, false},
		{"script panel closes", []string{"goals"}, true, []string{"goals"}, false},
		{"report", []string{"report"}, true, escapes, false},
		{"report closes", []string{"report"}, true, []string{"report"}, false},
		{"save completes into a report", []string{"saveas", "ui:text-7", "enter"}, true, []string{"escape"}, false},
		{"refused save report", []string{"saveas", "ui:text-bad", "enter"}, true, escapes, false},
		{"terrain brush", []string{"terrain", "ui:press-cell"}, false, escapes, false},
		{"terrain brush toggles", []string{"terrain", "ui:press-cell"}, false, []string{"terrain"}, false},
		{"height brush", []string{"height"}, false, escapes, false},
		{"objects brush", []string{"objects"}, false, escapes, false},
		{"brush under a pick", []string{"ui:select0", "terrain", "clone"}, false, escapes, false},
		{"catalogue", []string{"catalog"}, true, escapes, false},
		{"catalogue toggles", []string{"catalog"}, true, []string{"catalog"}, false},
		{"new map prompt", []string{"new"}, false, escapes, false},
		{"new map returns after cancel", []string{"new", "escape"}, false, escapes, true},
		{"catalogue under a prompt", []string{"catalog", "open"}, false, escapes, false},
	}
	for _, c := range cases {
		e, _ := newModesEditor(t, "full")
		for _, s := range c.enter {
			modesAct(e, s)
		}
		if idleMode(e, false) != c.busy {
			t.Errorf("%s: after %v the window is idle=%v, want %v", c.name, c.enter, idleMode(e, false), c.busy)
		}
		if c.list {
			modesWheel(e)
			if e.listWheel == 0 {
				t.Fatalf("%s: the wheel left no remainder to clear", c.name)
			}
		}
		for _, s := range c.exit {
			modesAct(e, s)
		}
		if !idleMode(e, c.list) {
			t.Errorf("%s: after %v the window keeps %+v wheel %v", c.name, c.exit, e.editorMode, e.listWheel)
		}
	}
	// Opening another map leaves every mode the report, the prompt and the
	// catalogue hold; the author's brush choice stays.
	for _, enter := range [][]string{{"report"}, {"catalog"}, {"open", "ui:text-7"}, {"terrain", "ui:press-cell"}} {
		e, _ := newModesEditor(t, "full")
		for _, s := range enter {
			modesAct(e, s)
		}
		if enter[0] == "report" || enter[0] == "catalog" {
			modesWheel(e)
		}
		if err := e.Open("map1"); err != nil {
			t.Fatal(err)
		}
		brush := e.brush.mode
		e.brush.mode = brushOff
		if !idleMode(e, true) || (enter[0] == "terrain") != (brush == brushTerrain) {
			t.Errorf("opening a map after %v keeps %+v wheel %v", enter, e.editorMode, e.listWheel)
		}
	}
}

// modeFields are the selector chains that enter or leave a mode.
var modeFields = []string{".catalog", ".pick", ".prompt", ".prompt.kind", ".report", ".report.open", ".report.goals", ".brush.mode", ".panel", ".editorMode"}

func selectorChain(x ast.Expr) string {
	switch x := x.(type) {
	case *ast.SelectorExpr:
		return selectorChain(x.X) + "." + x.Sel.Name
	case *ast.Ident:
		return x.Name
	}
	return "?"
}

// modeAssignments lists the assignments in dir's window files, outside the
// one mode file, that set a mode field.
func modeAssignments(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	var files []string
	for _, pat := range []string{"mapeditor*.go", "editor*.go"} {
		m, _ := filepath.Glob(filepath.Join(dir, pat))
		files = append(files, m...)
	}
	for _, f := range files {
		base := filepath.Base(f)
		if strings.HasSuffix(base, "_test.go") || base == "mapeditormode.go" {
			continue
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, f, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		check := func(x ast.Expr, pos token.Pos) {
			chain := selectorChain(x)
			for _, m := range modeFields {
				if strings.HasSuffix(chain, m) && strings.Count(chain, ".") == strings.Count(m, ".") {
					out = append(out, fset.Position(pos).String()+" "+chain)
				}
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, l := range n.Lhs {
					check(l, n.Pos())
				}
			case *ast.IncDecStmt:
				check(n.X, n.Pos())
			}
			return true
		})
	}
	return out
}

// TestMapEditorModeIsSetInOnePlace: the window files other than the mode file
// assign no mode field.
func TestMapEditorModeIsSetInOnePlace(t *testing.T) {
	if got := modeAssignments(t, "."); len(got) > 0 {
		t.Errorf("mode fields set outside mapeditormode.go:\n%s", strings.Join(got, "\n"))
	}
	if _, err := os.Stat("mapeditormode.go"); err != nil {
		t.Fatal(err)
	}
}

// TestMapEditorModeScanFindsAnAssignment is the loss control of the scan
// above: a window file that sets each mode field is reported, and one that
// sets the data a mode owns is not.
func TestMapEditorModeScanFindsAnAssignment(t *testing.T) {
	dir := t.TempDir()
	src := "package ui\nfunc (e *MapEditor) sneak() {\n\te.pick = pickClone\n\te.report.open = true\n\te.report.goals = nil\n\te.brush.mode = brushOff\n\te.catalog = true\n\te.prompt.kind = promptOpen\n\te.prompt = editorPrompt{}\n\te.report = reportState{}\n\te.report.sel = 1\n\te.prompt.path = \"x\"\n}\n"
	if err := os.WriteFile(filepath.Join(dir, "mapeditorsneak.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := modeAssignments(t, dir); len(got) != 8 {
		t.Errorf("scan found %d assignments, want 8: %v", len(got), got)
	}
}
