package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// mapBytesWriters are the methods that write one map's bytes into a save or
// a mod draft. Each gets the bytes from mapBytes.
var mapBytesWriters = []string{
	"mapInspectionEdits.encodeForSave",
	"mapInspectionEdits.saveWorkspacePackage",
	"mapInspectionEdits.syncModDocument",
	"mapInspectionEdits.newModDocument",
	"mapInspectionEdits.draftOpenMap",
	"mapInspectionEdits.Shown",
	"mapInspectionEdits.testPlayWith",
	"MapInspector.WriteNewMap",
}

// TestMapBytesIsTheOnlyMapWriter holds the texts record to one home: only
// mapBytes calls withTexts, and every map writer calls mapBytes.
func TestMapBytesIsTheOnlyMapWriter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, filepath.Join(root, "pkg", "game"), func(fi os.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatal(err)
	}
	asksBuilder := map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, decl := range file.Decls {
				fd, ok := decl.(*ast.FuncDecl)
				if !ok || fd.Body == nil {
					continue
				}
				key := fd.Name.Name
				if fd.Recv != nil && len(fd.Recv.List) == 1 {
					recv := fd.Recv.List[0].Type
					if star, ok := recv.(*ast.StarExpr); ok {
						recv = star.X
					}
					if id, ok := recv.(*ast.Ident); ok {
						key = id.Name + "." + key
					}
				}
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					switch sel.Sel.Name {
					case "withTexts":
						if key != "mapInspectionEdits.mapBytes" {
							t.Errorf("%s: %s calls withTexts; only mapBytes may", fset.Position(call.Pos()), key)
						}
					case "SaveBytes":
						if key != "mapInspectionEdits.mapBytes" {
							t.Errorf("%s: %s encodes the map itself; only mapBytes may", fset.Position(call.Pos()), key)
						}
					case "AppendTextsRecord":
						if key != "mapInspectionEdits.withTexts" {
							t.Errorf("%s: %s writes the record; only withTexts may", fset.Position(call.Pos()), key)
						}
					case "mapBytes":
						asksBuilder[key] = true
					}
					return true
				})
			}
		}
	}
	for _, key := range mapBytesWriters {
		if !asksBuilder[key] {
			t.Errorf("%s writes a map's bytes without mapBytes", key)
		}
	}
}
