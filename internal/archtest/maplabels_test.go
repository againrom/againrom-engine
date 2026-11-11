package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

// The tree table owns the name; the editor owns the JSON reader and writer.
const labelFileHome = "pkg/mod/kinds.go"
const labelCodecHome = "pkg/game/maptextpages.go"

var labelFileNames = map[string]bool{"mapLabelsFile": true, "mapLabelEntry": true}

func checkMapLabels(name, src string) (found int, vs []string) {
	f, err := parser.ParseFile(token.NewFileSet(), name, src, parser.SkipObjectResolution)
	if err != nil {
		return 0, []string{name + ": " + err.Error()}
	}
	home := name == labelFileHome
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.BasicLit:
			if n.Kind == token.STRING && n.Value == `".variables"` {
				if home {
					found++
				} else {
					vs = append(vs, name+": names the variable labels file; MapLabelFile in "+labelFileHome+" names it")
				}
			}
		case *ast.FuncDecl:
			if n.Name.Name == "MapLabelFile" && !home {
				vs = append(vs, name+": builds the variable labels file name outside "+labelFileHome)
			}
		case *ast.Ident:
			if n.Name == "mapLabelFile" {
				vs = append(vs, name+": uses the former mapLabelFile builder; read mod.MapLabelFile instead")
			}
			if labelFileNames[n.Name] && name != labelCodecHome {
				vs = append(vs, name+": uses "+n.Name+"; the labels file is read and written in "+labelCodecHome+" alone")
			}
		}
		return true
	})
	return found, vs
}

func TestCheckMapLabels(t *testing.T) {
	cases := []struct {
		name, src string
		bad       bool
	}{
		{labelFileHome, "package mod\nfunc MapLabelFile(p string) string { return p + \".variables\" }\n", false},
		{labelCodecHome, "package game\ntype mapLabelsFile struct {}\nvar f mapLabelEntry\nfunc labels(p string) string { return mod.MapLabelFile(p) }\n", false},
		{"pkg/game/other.go", "package game\nfunc labels(p string) string { return p + \".variables\" }\n", true},
		{"pkg/game/other.go", "package game\nfunc labels(p string) string { return mapLabelFile(p) }\n", true},
		{labelCodecHome, "package game\nfunc mapLabelFile(p string) string { return mod.MapLabelFile(p) }\n", true},
		{labelCodecHome, "package game\nfunc MapLabelFile(p string) string { return p }\n", true},
		{"pkg/mod/other.go", "package mod\nfunc labels(p string) string { return MapLabelFile(p) }\n", false},
		{"pkg/ui/other.go", "package ui\nvar f mapLabelsFile\n", true},
		{"pkg/game/other.go", "package game\nfunc texts(p string) string { return p + \".text\" }\n", false},
	}
	for _, tc := range cases {
		_, vs := checkMapLabels(tc.name, tc.src)
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s %q: violations %v, want bad %v", tc.name, tc.src, vs, tc.bad)
		}
	}
}

func TestMapLabelFileHasOneHome(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for name, src := range files {
		n, vs := checkMapLabels(name, src)
		found += n
		for _, v := range vs {
			t.Error(v)
		}
	}
	if found != 1 {
		t.Fatalf("label file suffixes in %s = %d, want 1; the scan guards nothing", labelFileHome, found)
	}
}
