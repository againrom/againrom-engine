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

func TestOrderTargetSelectionHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for path, names := range map[string][]string{
		"escort.go": {"acquireStanding", "coverEngage"}, "savedgroupsai.go": {"savedDecision"},
		"reacquire.go": {"reacquisitionVictim"}, "groupengaged.go": {"joinEngagedGroup"},
	} {
		file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg", "sim", path), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range names {
			found, calls := false, 0
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || fn.Name.Name != name {
					continue
				}
				found = true
				ast.Inspect(fn.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					method, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					switch method.Sel.Name {
					case "chooseTarget":
						calls++
					case "candidateCost", "facingArc", "coverPreferred":
						t.Errorf("%s.%s ranks outside chooseTarget", path, name)
					}
					return true
				})
			}
			if !found || calls != 1 {
				t.Errorf("%s.%s has %d chooser calls, found=%v", path, name, calls, found)
			}
		}
	}
}

func TestAcquireActivityHasOneWriter(t *testing.T) {
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
	writers := 0
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
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				assign, ok := node.(*ast.AssignStmt)
				if !ok || len(assign.Lhs) != len(assign.Rhs) {
					return true
				}
				for i, lhs := range assign.Lhs {
					index, ok := lhs.(*ast.IndexExpr)
					if !ok {
						continue
					}
					field, ok := index.X.(*ast.SelectorExpr)
					if !ok || field.Sel.Name != "Raw" {
						continue
					}
					offset, ok := index.Index.(*ast.BasicLit)
					if !ok || offset.Value != "8" {
						continue
					}
					value, ok := assign.Rhs[i].(*ast.BasicLit)
					if ok && value.Value == "6" {
						writers++
						if filepath.Base(path) != "currentorder.go" || fn.Name.Name != "ProjectActorAcquireOrder" {
							t.Errorf("%s.%s writes acquisition activity outside the projection", path, fn.Name.Name)
						}
					}
				}
				return true
			})
		}
	}
	if writers != 1 {
		t.Fatalf("acquisition activity has %d literal writers, want 1", writers)
	}
}

func TestLookAroundGateHasOneBuilder(t *testing.T) {
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
	gates := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			value, ok := node.(*ast.BasicLit)
			if ok && value.Value == "0xcd" {
				gates++
				if filepath.Base(path) != "facing.go" {
					t.Errorf("%s has a second look-around random gate", path)
				}
			}
			return true
		})
	}
	if gates != 1 {
		t.Fatalf("look-around has %d random gates, want 1", gates)
	}
}
