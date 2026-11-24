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

// The empty-route reach rule has one home: patrolTailNext. Native and loaded
// patrols both reach it from patrolRefusedTail.
func TestPatrolTailReachHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := filepath.Glob(filepath.Join(root, "pkg", "sim", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	var readers []string
	tailCalls := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && id.Name == "patrolTailReach" {
					readers = append(readers, fn.Name.Name)
				}
				if call, ok := n.(*ast.CallExpr); ok && fn.Name.Name == "patrolRefusedTail" {
					if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "patrolTailNext" {
						tailCalls++
					}
				}
				return true
			})
		}
	}
	if len(readers) != 1 || readers[0] != "patrolTailNext" {
		t.Fatalf("patrolTailReach is read by %v, want only patrolTailNext", readers)
	}
	if tailCalls != 1 {
		t.Fatalf("patrolRefusedTail calls patrolTailNext %d times, want once", tailCalls)
	}
}
