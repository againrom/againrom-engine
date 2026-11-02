package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"testing"
)

// labelFileHome is the one file that names, reads and writes the variable
// labels file beside a map, beside the map's texts folder it extends.
const labelFileHome = "pkg/game/maptextpages.go"

// labelFileNames are the label file's name builder and its JSON form; only
// labelFileHome may use them.
var labelFileNames = map[string]bool{"mapLabelFile": true, "mapLabelsFile": true, "mapLabelEntry": true}

// checkMapLabels reports, in one file, the label file's suffix or its name
// builder and form used outside labelFileHome; found counts the suffix
// literals labelFileHome holds.
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
					vs = append(vs, name+": names the variable labels file; mapLabelFile in "+labelFileHome+" names it")
				}
			}
		case *ast.Ident:
			if labelFileNames[n.Name] && !home {
				vs = append(vs, name+": uses "+n.Name+"; the labels file is read and written in "+labelFileHome+" alone")
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
		{labelFileHome, "package game\nfunc mapLabelFile(p string) string { return p + \".variables\" }\n", false},
		{"pkg/game/other.go", "package game\nfunc labels(p string) string { return p + \".variables\" }\n", true},
		{"pkg/game/other.go", "package game\nfunc labels(p string) string { return mapLabelFile(p) }\n", true},
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

// The variable labels file is named, read and written in one file.
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
