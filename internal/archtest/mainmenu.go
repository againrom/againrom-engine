package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// The main-menu scan: menu.Load(set, layout) is the one loader and each game
// is one Layout row. It refuses a second Load*, a nil or empty-value test of an
// Assets or Layout field outside menu.go, and any other loader, loader-typed
// value, Assets literal or row name outside menuart.go and gamefiles.go.

const (
	mainMenuDir        = "pkg/render/menu/"
	mainMenuFile       = "pkg/render/menu/menu.go"
	mainMenuImport     = "againrom/pkg/render/menu"
	mainMenuCaller     = "pkg/game/menuart.go"
	mainMenuRows       = "pkg/game/gamefiles.go"
	mainMenuWrapper    = "LoadMainMenuArt"
	mainMenuButtonName = "LoadGameButton"
)

var mainMenuRowNames = map[string]bool{
	"FirstLayout": true, "SecondLayout": true, "FirstActions": true, "SecondActions": true,
}

// LoadMainMenuSources reads the module's Go files, keyed by relative path.
func LoadMainMenuSources(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	err := filepath.WalkDir(moduleRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(moduleRoot, path)
		if rerr != nil {
			return rerr
		}
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			if rel == "knowledge" || rel == ".git" || rel == "internal/archtest" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		out[rel] = string(b)
		return nil
	})
	return out, err
}

func isTestSource(name string) bool { return strings.HasSuffix(name, "_test.go") }

// CheckMainMenu reports every second path to the main menu's builder.
func CheckMainMenu(files map[string]string) []Violation {
	src, ok := files[mainMenuFile]
	if !ok {
		return []Violation{{From: mainMenuFile, Reason: "no menu.go scanned - the main-menu scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var vs []Violation
	fields, fvs := mainMenuFields(src)
	vs = append(vs, fvs...)
	loads := 0
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		if strings.HasPrefix(name, mainMenuDir) && !strings.Contains(strings.TrimPrefix(name, mainMenuDir), "/") {
			if isTestSource(name) {
				continue
			}
			loads += mainMenuInPackage(f, name, at, fields, &vs)
			continue
		}
		mainMenuOutside(f, name, at, &vs)
	}
	if loads != 1 {
		vs = append(vs, Violation{From: mainMenuDir, Reason: "pkg/render/menu declares " + strconv.Itoa(loads) + " exported Load functions named Load, want exactly one"})
	}
	return vs
}

// mainMenuFields reads the field names of Assets and Layout out of menu.go.
func mainMenuFields(src string) (map[string]bool, []Violation) {
	f, err := parser.ParseFile(token.NewFileSet(), mainMenuFile, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, []Violation{{From: mainMenuFile, Reason: "source does not parse: " + err.Error()}}
	}
	fields := map[string]bool{}
	for _, d := range f.Decls {
		gd, ok := d.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, s := range gd.Specs {
			ts, ok := s.(*ast.TypeSpec)
			if !ok || (ts.Name.Name != "Assets" && ts.Name.Name != "Layout") {
				continue
			}
			if str, ok := ts.Type.(*ast.StructType); ok {
				for _, fl := range str.Fields.List {
					for _, id := range fl.Names {
						fields[id.Name] = true
					}
				}
			}
		}
	}
	if !fields["Layout"] || !fields["Actions"] {
		return fields, []Violation{{From: mainMenuFile, Reason: "menu.go declares no Assets.Layout or Layout.Actions - the scan reads the wrong shape"}}
	}
	return fields, nil
}

// mainMenuInPackage scans a menu package file; it returns its count of Load.
func mainMenuInPackage(f *ast.File, name string, at func(ast.Node) string, fields map[string]bool, vs *[]Violation) int {
	loads := 0
	for _, d := range f.Decls {
		if fd, ok := d.(*ast.FuncDecl); ok && strings.HasPrefix(fd.Name.Name, "Load") {
			if fd.Name.Name == "Load" && fd.Recv == nil {
				loads++
				continue
			}
			*vs = append(*vs, Violation{From: at(fd.Name), Reason: fd.Name.Name + " is a second main-menu loader; pass a Layout row to menu.Load"})
		}
	}
	if name == mainMenuFile {
		return loads
	}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BinaryExpr:
			if n.Op != token.EQL && n.Op != token.NEQ {
				return true
			}
			for _, pair := range [2][2]ast.Expr{{n.X, n.Y}, {n.Y, n.X}} {
				field := mainMenuField(pair[0], fields)
				if field == "" {
					continue
				}
				if mainMenuEmpty(pair[1]) {
					*vs = append(*vs, Violation{From: at(n), Reason: "the " + field + " field is compared with nil or an empty value outside menu.go; put the difference in the Layout row"})
				} else if id := mainMenuRowName(pair[1]); id != "" {
					*vs = append(*vs, Violation{From: at(n), Reason: "the " + field + " field is compared with " + id + "; put the difference in the Layout row"})
				}
			}
		case *ast.CompositeLit:
			if id, ok := n.Type.(*ast.Ident); ok && id.Name == "Assets" {
				*vs = append(*vs, Violation{From: at(n), Reason: "an Assets literal outside menu.go is a second builder; menu.Load builds Assets"})
			}
		}
		return true
	})
	return loads
}

// mainMenuField names the Assets or Layout field e selects, or "".
func mainMenuField(e ast.Expr, fields map[string]bool) string {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.IndexExpr:
			e = x.X
		case *ast.SelectorExpr:
			if fields[x.Sel.Name] {
				return x.Sel.Name
			}
			return ""
		default:
			return ""
		}
	}
}

// mainMenuEmpty reports whether e is nil or an empty composite literal.
func mainMenuEmpty(e ast.Expr) bool {
	for {
		p, ok := e.(*ast.ParenExpr)
		if !ok {
			break
		}
		e = p.X
	}
	switch x := e.(type) {
	case *ast.Ident:
		return x.Name == "nil"
	case *ast.CompositeLit:
		return len(x.Elts) == 0
	}
	return false
}

func mainMenuRowName(e ast.Expr) string {
	switch x := e.(type) {
	case *ast.ParenExpr:
		return mainMenuRowName(x.X)
	case *ast.Ident:
		if mainMenuRowNames[x.Name] {
			return x.Name
		}
	case *ast.SelectorExpr:
		if mainMenuRowNames[x.Sel.Name] {
			return x.Sel.Name
		}
	}
	return ""
}

func mainMenuOutside(f *ast.File, name string, at func(ast.Node) string, vs *[]Violation) {
	local := ""
	for _, imp := range f.Imports {
		if p, err := strconv.Unquote(imp.Path.Value); err != nil || p != mainMenuImport {
			continue
		}
		local = "menu"
		if imp.Name != nil {
			local = imp.Name.Name
		}
		if local == "." {
			*vs = append(*vs, Violation{From: at(imp), Reason: "a dot import of the menu package hides its loader names"})
		}
	}
	if local == "" {
		return
	}
	test := isTestSource(name)
	isMenu := func(e ast.Expr) (string, bool) {
		sel, ok := e.(*ast.SelectorExpr)
		if !ok {
			return "", false
		}
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == local {
			return sel.Sel.Name, true
		}
		return "", false
	}
	returnsAssets := func(ft *ast.FuncType) bool {
		if ft == nil || ft.Results == nil {
			return false
		}
		for _, r := range ft.Results.List {
			if star, ok := r.Type.(*ast.StarExpr); ok {
				if sel, ok := isMenu(star.X); ok && sel == "Assets" {
					return true
				}
			}
		}
		return false
	}
	declared := map[*ast.FuncType]bool{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			sel, ok := isMenu(n)
			if !ok {
				return true
			}
			if strings.HasPrefix(sel, "Load") && sel != "Load" && sel != mainMenuButtonName {
				*vs = append(*vs, Violation{From: at(n), Reason: local + "." + sel + " is a second main-menu loader; pass a Layout row to menu.Load"})
			}
			if sel == "Load" && !test && name != mainMenuCaller {
				*vs = append(*vs, Violation{From: at(n), Reason: local + ".Load outside " + mainMenuCaller + " is a second place that builds the main menu; call game.LoadMainMenuArt"})
			}
			if mainMenuRowNames[sel] && !test && name != mainMenuRows {
				*vs = append(*vs, Violation{From: at(n), Reason: local + "." + sel + " outside " + mainMenuRows + " is a per-game choice of main-menu layout; read the game's gameFiles row"})
			}
		case *ast.FuncType:
			if returnsAssets(n) && !declared[n] {
				*vs = append(*vs, Violation{From: at(n), Reason: "a function type returning *menu.Assets is a per-game choice of main-menu loader; name a Layout row instead"})
			}
		case *ast.FuncDecl:
			declared[n.Type] = true
			if !test && returnsAssets(n.Type) && !(name == mainMenuCaller && n.Name.Name == mainMenuWrapper) {
				*vs = append(*vs, Violation{From: at(n.Name), Reason: n.Name.Name + " returns *menu.Assets: a second builder of the main menu beside game." + mainMenuWrapper})
			}
		case *ast.CompositeLit:
			if sel, ok := isMenu(n.Type); ok && sel == "Assets" && !test {
				*vs = append(*vs, Violation{From: at(n), Reason: "a menu.Assets literal outside the menu package is a second builder; menu.Load builds Assets"})
			}
		}
		return true
	})
}
