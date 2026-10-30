package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// The World's flight producer is the one builder of a projectile record:
// outside pkg/sim no production file calls the record writers it calls.
func TestProjectileRecordsHaveOneProducer(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadRandomnessSources(root)
	if err != nil {
		t.Fatal(err)
	}
	writers := map[string]bool{"ReleaseCast": true, "ReleaseUnitShot": true, "ReleaseAreaBurst": true}
	for name, source := range files {
		if strings.HasPrefix(name, "pkg/sim/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && writers[sel.Sel.Name] {
				t.Errorf("%s calls %s outside the World's flight producer", name, sel.Sel.Name)
			}
			return true
		})
	}
}
