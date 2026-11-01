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

// The NPC face scan: every NPC face list is built by the one face source in
// pkg/game/gamefiles.go. Outside that file and pkg/data, which defines the
// catalog, production code under pkg/ and cmd/ refuses:
//
//   - a call to the catalog's Portraits method;
//   - a read of the catalog's Faces map, through the NPCCatalog field, through
//     a call to LoadNPCCatalog, or through a name that was assigned either;
//   - a call to LoadNPCFaces.
//
// Ask pkg/game's speakerFaces for the game's list and CatalogFaces for the
// catalog's rows alone.

func npcFaceSourceFile(name string) bool {
	return name == "pkg/game/gamefiles.go" || strings.HasPrefix(name, "pkg/data/")
}

// LoadNPCFaceSources reads the non-test Go sources under pkg/ and cmd/, keyed
// by module-relative slash path.
func LoadNPCFaceSources(moduleRoot string) (map[string]string, error) {
	out := make(map[string]string)
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(moduleRoot, top), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			name := d.Name()
			if d.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				return nil
			}
			b, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			rel, rerr := filepath.Rel(moduleRoot, path)
			if rerr != nil {
				return rerr
			}
			out[filepath.ToSlash(rel)] = string(b)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

// isCatalogExpr reports whether e is the catalog: a field or name called
// NPCCatalog, or a call to LoadNPCCatalog.
func isCatalogExpr(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.ParenExpr:
		return isCatalogExpr(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name == "NPCCatalog"
	case *ast.Ident:
		return e.Name == "NPCCatalog"
	case *ast.CallExpr:
		switch f := e.Fun.(type) {
		case *ast.SelectorExpr:
			return f.Sel.Name == "LoadNPCCatalog"
		case *ast.Ident:
			return f.Name == "LoadNPCCatalog"
		}
	}
	return false
}

// CheckNPCFaces reports every read of the catalog's faces outside the one
// source. files is keyed by module-relative slash path.
func CheckNPCFaces(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the NPC face scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if npcFaceSourceFile(name) || !(strings.HasPrefix(name, "pkg/") || strings.HasPrefix(name, "cmd/")) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		held := map[string]bool{}
		note := func(lhs, rhs []ast.Expr) {
			for i, l := range lhs {
				if id, ok := l.(*ast.Ident); ok && i < len(rhs) && isCatalogExpr(rhs[i]) {
					held[id.Name] = true
				}
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				note(n.Lhs, n.Rhs)
			case *ast.ValueSpec:
				lhs := make([]ast.Expr, len(n.Names))
				for i, id := range n.Names {
					lhs[i] = id
				}
				note(lhs, n.Values)
			}
			return true
		})
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				var fn string
				switch c := n.Fun.(type) {
				case *ast.SelectorExpr:
					fn = c.Sel.Name
				case *ast.Ident:
					fn = c.Name
				}
				switch fn {
				case "Portraits":
					if _, method := n.Fun.(*ast.SelectorExpr); method {
						vs = append(vs, Violation{From: at(n), Reason: "Portraits reads the catalog's faces outside the face source; ask speakerFaces or CatalogFaces"})
					}
				case "LoadNPCFaces":
					vs = append(vs, Violation{From: at(n), Reason: "LoadNPCFaces reads the catalog's faces outside the face source; ask speakerFaces or CatalogFaces"})
				}
			case *ast.SelectorExpr:
				if n.Sel.Name != "Faces" {
					return true
				}
				if id, ok := n.X.(*ast.Ident); ok && held[id.Name] || isCatalogExpr(n.X) {
					vs = append(vs, Violation{From: at(n), Reason: "the catalog's Faces map is read outside the face source; ask speakerFaces or CatalogFaces"})
				}
			}
			return true
		})
	}
	return vs
}
