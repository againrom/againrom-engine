package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// pathSources parses the production Go files under dirs of the module.
func pathSources(t *testing.T, dirs ...string) (*token.FileSet, map[string]*ast.File) {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	fset, files := token.NewFileSet(), map[string]*ast.File{}
	for _, dir := range dirs {
		err := filepath.WalkDir(filepath.Join(root, dir), func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return err
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			file, err := parser.ParseFile(fset, path, nil, 0)
			files[filepath.ToSlash(rel)] = file
			return err
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return fset, files
}

// eachFunc calls see with every function declaration of files.
func eachFunc(files map[string]*ast.File, see func(rel string, fn *ast.FuncDecl)) {
	for rel, file := range files {
		for _, d := range file.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Body != nil {
				see(rel, fn)
			}
		}
	}
}

// TestPathRequestHasOneBuilder fails on a Request literal outside NewRequest.
func TestPathRequestHasOneBuilder(t *testing.T) {
	fset, files := pathSources(t, "pkg", "cmd")
	builder := 0
	for rel, file := range files {
		inBuilder := func(token.Pos) bool { return false }
		for _, d := range file.Decls {
			if fn, ok := d.(*ast.FuncDecl); ok && fn.Recv == nil && fn.Name.Name == "NewRequest" && rel == "pkg/pathdialog/pathdialog.go" {
				inBuilder = func(p token.Pos) bool { return p >= fn.Pos() && p < fn.End() }
			}
		}
		ast.Inspect(file, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			request := false
			if id, ok := lit.Type.(*ast.Ident); ok && id.Name == "Request" && strings.HasPrefix(rel, "pkg/pathdialog/") {
				request = true
			}
			if sel, ok := lit.Type.(*ast.SelectorExpr); ok && sel.Sel.Name == "Request" {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == "pathdialog" {
					request = true
				}
			}
			switch {
			case request && inBuilder(lit.Pos()):
				builder++
			case request:
				t.Errorf("%s writes a pathdialog.Request literal; build it with pathdialog.NewRequest", fset.Position(lit.Pos()))
			}
			return true
		})
	}
	if builder != 1 {
		t.Fatalf("NewRequest writes %d Request literals, want 1", builder)
	}
}

// TestEditorPathQuestionsHaveOneOpener fails when askChoice is called outside
// askPathQuestion, or a function that asks a path question tests a key of
// modPathRows.
func TestEditorPathQuestionsHaveOneOpener(t *testing.T) {
	fset, files := pathSources(t, "pkg/ui")
	keys := map[string]bool{}
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			spec, ok := n.(*ast.ValueSpec)
			if !ok || len(spec.Names) != 1 || spec.Names[0].Name != "modPathRows" || len(spec.Values) != 1 {
				return true
			}
			if lit, ok := spec.Values[0].(*ast.CompositeLit); ok {
				for _, el := range lit.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						if s, ok := kv.Key.(*ast.BasicLit); ok {
							key, _ := strconv.Unquote(s.Value)
							keys[key] = true
						}
					}
				}
			}
			return false
		})
	}
	if len(keys) == 0 {
		t.Fatal("pkg/ui declares no modPathRows keys")
	}
	called := func(fn *ast.FuncDecl, name string) bool {
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == name {
					found = true
				}
			}
			return !found
		})
		return found
	}
	keyLit := func(x ast.Expr) bool {
		s, ok := x.(*ast.BasicLit)
		if !ok || s.Kind != token.STRING {
			return false
		}
		v, _ := strconv.Unquote(s.Value)
		return keys[v]
	}
	eachFunc(files, func(rel string, fn *ast.FuncDecl) {
		if fn.Name.Name != "askPathQuestion" && called(fn, "askChoice") {
			t.Errorf("%s: %s calls askChoice; ask through askPathQuestion", fset.Position(fn.Pos()), fn.Name.Name)
		}
		if !called(fn, "askPathQuestion") {
			return
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BinaryExpr:
				if (x.Op == token.EQL || x.Op == token.NEQ) && (keyLit(x.X) || keyLit(x.Y)) {
					t.Errorf("%s: %s tests a path row's key; give the row its question in modPathRows", fset.Position(x.Pos()), fn.Name.Name)
				}
			case *ast.CaseClause:
				for _, v := range x.List {
					if keyLit(v) {
						t.Errorf("%s: %s switches on a path row's key; give the row its question in modPathRows", fset.Position(v.Pos()), fn.Name.Name)
					}
				}
			}
			return true
		})
	})
}
