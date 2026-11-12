package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// keyReaders are the engine calls that read a key's press edge or level.
var keyReaders = map[string]bool{
	"inpututil.IsKeyJustPressed": true, "inpututil.IsKeyJustReleased": true, "inpututil.KeyPressDuration": true,
	"inpututil.AppendJustPressedKeys": true, "inpututil.AppendPressedKeys": true, "ebiten.IsKeyPressed": true,
	"ebiten.AppendInputChars": true, "heldkey.Pressed": true, "heldkey.AppendPressed": true,
}

// otherKeySites are the functions outside the mission key table that read a
// key, each with the kind of key it reads. "*" accepts the whole file.
var otherKeySites = map[string]string{
	"pkg/ui/app.go altHeld":                             "modifier",
	"pkg/ui/commandmodifier.go platformCommandModifier": "modifier",
	"pkg/ui/viewer.go readInput":                        "camera pan and modifiers, shared with the map editor and the viewer",
	"pkg/ui/viewer.go Viewer.Update":                    "the standalone viewer's own Esc",
	"pkg/ui/editorinputnative.go readEditorInput":       "map editor",
	"pkg/ui/heldkey/heldkey.go *":                       "the held-key level the readers share",
	"cmd/starter/window.go *":                           "the starter window",
}

// otherKeyFields are the fields of readAppInput's literal that read a key of
// another kind; every other key the front-end reads is a mission key row.
var otherKeyFields = map[string]string{
	"AnyKey": "any key", "AnyHeld": "any key", "Escape": "menus and dialogues", "Enter": "menus and dialogues",
	"Up": "lists", "Down": "lists", "PageUp": "lists", "PageDown": "lists", "Left": "lists and text fields",
	"Right": "lists and text fields", "Typed": "text field", "Backspace": "text field",
	"Delete": "text field", "Home": "text field", "End": "text field", "SelectText": "text field",
	"AltLetter": "console", "Load": "main menu", "PaneMode": "character pane",
}

// missionKeyTable is what the table names: its row names and the appInput
// fields its reader sets.
type missionKeyTable struct{ rows, fields map[string]bool }

func readMissionKeyTable(src string) (missionKeyTable, error) {
	f, err := parser.ParseFile(token.NewFileSet(), "missionbindings.go", src, 0)
	if err != nil {
		return missionKeyTable{}, err
	}
	table := missionKeyTable{rows: map[string]bool{}, fields: map[string]bool{}}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		switch fn.Name.Name {
		case "missionBindings":
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if kv, ok := n.(*ast.KeyValueExpr); ok {
					if id, ok := kv.Key.(*ast.Ident); ok && id.Name == "name" {
						if lit, ok := kv.Value.(*ast.BasicLit); ok {
							table.rows[strings.Trim(lit.Value, `"`)] = true
						}
					}
				}
				return true
			})
		case "readMissionKeyRows":
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if assign, ok := n.(*ast.AssignStmt); ok {
					for _, lhs := range assign.Lhs {
						if field := inField(lhs); field != "" {
							table.fields[field] = true
						}
					}
				}
				return true
			})
		}
	}
	if !table.rows["p"] || !table.fields["Patrol"] || !table.fields["QuickSpell"] {
		return table, fmt.Errorf("the mission key table names rows %d and fields %d: the scan found no table", len(table.rows), len(table.fields))
	}
	return table, nil
}

// inField is the appInput field an assignment target names: in.X or in.X[i].
func inField(lhs ast.Expr) string {
	if index, ok := lhs.(*ast.IndexExpr); ok {
		lhs = index.X
	}
	if sel, ok := lhs.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "in" {
			return sel.Sel.Name
		}
	}
	return ""
}

func readsAKey(n ast.Node) (token.Pos, bool) {
	var at token.Pos
	ast.Inspect(n, func(node ast.Node) bool {
		if sel, ok := node.(*ast.SelectorExpr); ok && at == token.NoPos {
			if pkg, ok := sel.X.(*ast.Ident); ok && keyReaders[pkg.Name+"."+sel.Sel.Name] {
				at = sel.Pos()
			}
		}
		return at == token.NoPos
	})
	return at, at != token.NoPos
}

func funcSite(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	recv := fn.Recv.List[0].Type
	if star, ok := recv.(*ast.StarExpr); ok {
		recv = star.X
	}
	if id, ok := recv.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// missionKeyBypasses lists each place in file rel that reads a key outside
// the mission key table and outside the named other kinds, and each scenario
// key name of a row that sets a mission field without the table.
func missionKeyBypasses(rel, src string, table missionKeyTable) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return []string{rel + ": " + err.Error()}
	}
	var out []string
	refuse := func(at token.Pos, format string, args ...any) {
		out = append(out, fmt.Sprintf("%s:%d: ", rel, fset.Position(at).Line)+fmt.Sprintf(format, args...))
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			if at, reads := readsAKey(decl); reads && otherKeySites[rel+" *"] == "" {
				refuse(at, "a package-level key read outside the mission key table")
			}
			continue
		}
		site := funcSite(fn)
		switch {
		case rel == "pkg/ui/app.go" && site == "readAppInput":
			out = append(out, appInputBypasses(fset, rel, fn, table)...)
		case rel == "pkg/ui/headless.go" && site == "App.HeadlessKey":
			out = append(out, headlessBypasses(fset, rel, fn, table)...)
		case otherKeySites[rel+" "+site] != "" || otherKeySites[rel+" *"] != "":
		default:
			if at, reads := readsAKey(fn); reads {
				refuse(at, "%s reads a key outside the mission key table and the named other kinds", site)
			}
		}
	}
	return out
}

func appInputBypasses(fset *token.FileSet, rel string, fn *ast.FuncDecl, table missionKeyTable) []string {
	var out []string
	refuse := func(at token.Pos, format string, args ...any) {
		out = append(out, fmt.Sprintf("%s:%d: ", rel, fset.Position(at).Line)+fmt.Sprintf(format, args...))
	}
	for _, stmt := range fn.Body.List {
		if assign, ok := stmt.(*ast.AssignStmt); ok && len(assign.Rhs) == 1 {
			if id, ok := assign.Lhs[0].(*ast.Ident); ok && id.Name == "shiftHeld" {
				continue // modifier
			}
			if lit, ok := assign.Rhs[0].(*ast.CompositeLit); ok {
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					field, ok := kv.Key.(*ast.Ident)
					if !ok {
						continue
					}
					if table.fields[field.Name] {
						refuse(kv.Pos(), "readAppInput sets mission field %s outside the mission key table", field.Name)
					} else if at, reads := readsAKey(kv.Value); reads && otherKeyFields[field.Name] == "" {
						refuse(at, "readAppInput's %s reads a key and is no named other kind", field.Name)
					}
				}
				continue
			}
		}
		if expr, ok := stmt.(*ast.ExprStmt); ok {
			if call, ok := expr.X.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "readMissionKeyRows" {
					continue
				}
			}
		}
		if at, reads := readsAKey(stmt); reads {
			refuse(at, "readAppInput reads a key outside its literal and the mission key table")
		}
		ast.Inspect(stmt, func(n ast.Node) bool {
			if assign, ok := n.(*ast.AssignStmt); ok {
				for _, lhs := range assign.Lhs {
					if field := inField(lhs); table.fields[field] {
						refuse(lhs.Pos(), "readAppInput sets mission field %s outside the mission key table", field)
					}
				}
			}
			return true
		})
	}
	return out
}

// headlessBypasses refuses a scenario case that names a row's key, with or
// without ctrl-, shift- or alt-, and sets a mission field itself.
func headlessBypasses(fset *token.FileSet, rel string, fn *ast.FuncDecl, table missionKeyTable) []string {
	var out []string
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		clause, ok := n.(*ast.CaseClause)
		if !ok {
			return true
		}
		physical := false
		for _, label := range clause.List {
			if lit, ok := label.(*ast.BasicLit); ok {
				name := strings.Trim(lit.Value, "\"'`")
				for _, prefix := range []string{"ctrl-", "shift-", "alt-"} {
					name = strings.TrimPrefix(name, prefix)
				}
				physical = physical || table.rows[name]
			}
		}
		if !physical {
			return true
		}
		for _, stmt := range clause.Body {
			ast.Inspect(stmt, func(node ast.Node) bool {
				if assign, ok := node.(*ast.AssignStmt); ok {
					for _, lhs := range assign.Lhs {
						if field := inField(lhs); table.fields[field] {
							out = append(out, fmt.Sprintf("%s:%d: the scenario key of a row sets mission field %s without the table",
								rel, fset.Position(lhs.Pos()).Line, field))
						}
					}
				}
				return true
			})
		}
		return true
	})
	return out
}

func moduleFile(t *testing.T, root, rel string) string {
	t.Helper()
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(src)
}

// Every mission key is a row of the mission key table: production code under
// pkg and cmd reads a key only through the table's reader or at a site of a
// named other kind, and a scenario key name of a row presses the row.
func TestMissionKeysAreReadOnlyThroughTheTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	table, err := readMissionKeyTable(moduleFile(t, root, "pkg/ui/missionbindings.go"))
	if err != nil {
		t.Fatal(err)
	}
	var bypasses []string
	for _, dir := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			src, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			text := string(src)
			if !strings.Contains(text, "inpututil.") && !strings.Contains(text, "heldkey.") &&
				!strings.Contains(text, "ebiten.IsKeyPressed") && !strings.Contains(text, "ebiten.AppendInputChars") &&
				!strings.HasSuffix(filepath.ToSlash(path), "pkg/ui/headless.go") {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			bypasses = append(bypasses, missionKeyBypasses(filepath.ToSlash(rel), text, table)...)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, b := range bypasses {
		t.Error(b)
	}
	for rel, call := range map[string]string{"pkg/ui/app.go": "in.readMissionKeyRows(keys,", "pkg/ui/headless.go": "in.missionKeyNamed(keys,"} {
		if !strings.Contains(moduleFile(t, root, rel), call) {
			t.Errorf("%s no longer calls %s", rel, call)
		}
	}
}

func TestMissionKeyGuardRefusesReadsOutsideTheTable(t *testing.T) {
	table := missionKeyTable{
		rows:   map[string]bool{"p": true, "x": true, "f4": true},
		fields: map[string]bool{"Patrol": true, "Worn": true, "QuickSave": true, "QuickSpell": true},
	}
	for _, tc := range []struct {
		rel, src string
		want     bool
	}{
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { in := appInput{Worn: inpututil.IsKeyJustPressed(ebiten.KeyX)}; return in }`, true},
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { in := appInput{Worn: false}; return in }`, true},
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { in := appInput{Spare: inpututil.IsKeyJustPressed(ebiten.KeyV)}; return in }`, true},
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { var in appInput; in.QuickSpell[0] = true; return in }`, true},
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { in := appInput{}; if inpututil.IsKeyJustPressed(ebiten.KeyV) {}; return in }`, true},
		{"pkg/ui/app.go", `package ui; func readAppInput() appInput { in := appInput{Escape: inpututil.IsKeyJustPressed(ebiten.KeyEscape)}; in.readMissionKeyRows(keys, inpututil.IsKeyJustPressed, heldkey.Pressed, ctrlHeld()); return in }`, false},
		{"pkg/ui/viewer.go", `package ui; func stepMap() bool { return inpututil.IsKeyJustPressed(ebiten.KeyV) }`, true},
		{"pkg/ui/viewer.go", `package ui; func readInput() Input { return Input{PanLeft: ebiten.IsKeyPressed(ebiten.KeyLeft)} }`, false},
		{"pkg/game/session.go", `package game; func held() bool { return heldkey.Pressed(ebiten.KeyZ) }`, true},
		{"pkg/game/session.go", `package game; var z = ebiten.IsKeyPressed(ebiten.KeyZ)`, true},
		{"pkg/ui/headless.go", `package ui; func (a *App) HeadlessKey(name string) error { switch name { case "x": in.Worn = true }; return nil }`, true},
		{"pkg/ui/headless.go", `package ui; func (a *App) HeadlessKey(name string) error { switch name { case "ctrl-f4": in.QuickSave = true }; return nil }`, true},
		{"pkg/ui/headless.go", `package ui; func (a *App) HeadlessKey(name string) error { switch name { case "x": in.missionKeyNamed(keys, "x") }; return nil }`, false},
		{"pkg/ui/headless.go", `package ui; func (a *App) HeadlessKey(name string) error { switch name { case "patrol": in.Patrol = true }; return nil }`, false},
	} {
		got := missionKeyBypasses(tc.rel, tc.src, table)
		if (len(got) != 0) != tc.want {
			t.Errorf("guard %v, want %v: %s %s", got, tc.want, tc.rel, tc.src)
		}
	}
	for _, kinds := range []map[string]string{otherKeyFields, otherKeySites} {
		for site, kind := range kinds {
			if kind == "" {
				t.Errorf("%s reads a key of no named kind", site)
			}
		}
	}
}

func textBackspaceClearsMessages(rel, src string) []string {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, rel, src, 0)
	if err != nil {
		return []string{rel + ": " + err.Error()}
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		stmt, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		names, clears := false, false
		ast.Inspect(stmt.Cond, func(c ast.Node) bool {
			if sel, ok := c.(*ast.SelectorExpr); ok && sel.Sel.Name == "Backspace" {
				names = true
			}
			return true
		})
		ast.Inspect(stmt.Body, func(c ast.Node) bool {
			if call, ok := c.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "ClearMessages" {
					clears = true
				}
			}
			return true
		})
		if names && clears {
			out = append(out, fmt.Sprintf("%s:%d: the map's message clear reads the text-field Backspace, not the ClearMessages row",
				rel, fset.Position(stmt.Pos()).Line))
		}
		return true
	})
	return out
}

func TestMapMessageClearIsNotTheTextFieldBackspace(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(moduleFile(t, root, "pkg/ui/app.go"), "if in.ClearMessages {") {
		t.Error("the map arm no longer reads in.ClearMessages")
	}
	for _, rel := range []string{"pkg/ui/app.go"} {
		for _, b := range textBackspaceClearsMessages(rel, moduleFile(t, root, rel)) {
			t.Error(b)
		}
	}
	for src, want := range map[string]bool{
		`package ui; func f() { if in.Backspace { v.ClearMessages() } }`:                  true,
		`package ui; func f() { if in.Backspace && !x { v.ClearMessages() } }`:            true,
		`package ui; func f() { if in.ClearMessages { v.ClearMessages() } }`:              false,
		`package ui; func f() { if in.Backspace { c.EditName(in.Typed, in.Backspace) } }`: false,
	} {
		if got := textBackspaceClearsMessages("pkg/ui/app.go", src); (len(got) != 0) != want {
			t.Errorf("guard %v, want %v: %s", got, want, src)
		}
	}
}
