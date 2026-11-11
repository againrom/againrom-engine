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

func TestModTreePathsHaveOneTable(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	table := filepath.Join(root, "pkg", "mod", "kinds.go")
	parse := func(p string) *ast.File {
		f, err := parser.ParseFile(token.NewFileSet(), p, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		return f
	}
	paths := map[string]bool{}
	f := parse(table)
	for _, decl := range f.Decls {
		g, ok := decl.(*ast.GenDecl)
		if !ok || g.Tok != token.CONST {
			continue
		}
		for _, spec := range g.Specs {
			v := spec.(*ast.ValueSpec)
			for _, value := range v.Values {
				if b, ok := value.(*ast.BasicLit); ok && b.Kind == token.STRING {
					s, _ := strconv.Unquote(b.Value)
					if strings.Contains(s, "/") || strings.Contains(s, ".toml") || strings.Contains(s, ".star") || strings.Contains(s, ".md") || v.Names[0].Name == "FilesDir" || v.Names[0].Name == "MapsFolder" {
						paths[s] = true
					}
				}
			}
		}
	}
	if len(paths) < 18 {
		t.Fatalf("only %d table paths", len(paths))
	}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") || d.Name() == "knowledge" || d.Name() == "vendor" || d.Name() == "testdata" {
				return fs.SkipDir
			}
			return nil
		}
		if p == table || !strings.HasSuffix(p, ".go") || strings.HasSuffix(p, "_test.go") {
			return nil
		}
		for _, decl := range parse(p).Decls {
			g, ok := decl.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, spec := range g.Specs {
				v := spec.(*ast.ValueSpec)
				for _, value := range v.Values {
					if b, ok := value.(*ast.BasicLit); ok && b.Kind == token.STRING {
						s, _ := strconv.Unquote(b.Value)
						if p == filepath.Join(root, "pkg", "game", "mapinspectionedit.go") && v.Names[0].Name == "editorMapsFolder" {
							continue
						} // Profile folder, outside a mod.
						if paths[s] {
							t.Errorf("tree path %q is declared outside pkg/mod/kinds.go: %s", s, p)
						}
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestModRowReferencesUseReaderResolvers(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		file string
		want []string
		deny []string
	}{
		{"pkg/mod/treedescription.go", []string{"modFilePath", "missionPath", "TownArtFiles"}, []string{"Keys", "ReplaceAll", "MustCompile"}},
		{"pkg/mod/tree.go", []string{"modFilePath"}, nil},
		{"pkg/mod/towns.go", []string{"Keys", "modFilePath"}, nil},
		{"pkg/game/townlayer.go", []string{"TownArtFiles"}, nil},
		{"pkg/mod/missions.go", []string{"missionPath"}, nil},
	} {
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(tc.file)), nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		calls := map[string]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			c, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			switch fn := c.Fun.(type) {
			case *ast.Ident:
				calls[fn.Name] = true
			case *ast.SelectorExpr:
				calls[fn.Sel.Name] = true
			}
			return true
		})
		for _, name := range tc.want {
			if !calls[name] {
				t.Errorf("%s must use %s", tc.file, name)
			}
		}
		for _, name := range tc.deny {
			if calls[name] {
				t.Errorf("%s duplicates the reader's resolver through %s", tc.file, name)
			}
		}
	}
}
