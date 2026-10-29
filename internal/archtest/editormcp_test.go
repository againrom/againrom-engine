package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func editorMCPViolations(files map[string]string) []string {
	var out []string
	for path, source := range files {
		if strings.HasSuffix(path, "_test.go") || !strings.HasPrefix(path, "pkg/ui/") || !strings.Contains(path, "mcp") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, source, 0)
		if err != nil {
			out = append(out, path+": "+err.Error())
			continue
		}
		for _, imp := range f.Imports {
			if imp.Path.Value == `"os"` || strings.HasPrefix(imp.Path.Value, `"againrom/pkg/mapedit`) {
				out = append(out, path+": MCP must use window file and edit routes")
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			if s, ok := n.(*ast.SelectorExpr); ok && (s.Sel.Name == "Edits" || s.Sel.Name == "HeadlessFrame") {
				out = append(out, path+": MCP reaches the edit host directly")
			}
			return true
		})
	}
	return out
}

func TestEditorMCPUsesWindowDoors(t *testing.T) {
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
	for _, v := range editorMCPViolations(files) {
		t.Error(v)
	}
	for _, source := range []string{`package ui;func mcp(e *MapEditor){e.doc.Edits.Delete("Unit",0)}`, `package ui;import "againrom/pkg/mapedit"`, `package ui;import "os"`} {
		if len(editorMCPViolations(map[string]string{"pkg/ui/mapeditormcp.go": source})) == 0 {
			t.Fatal("guard accepted a second edit or file path", source)
		}
	}
}

func TestEditorMCPIsAProtocolLeaf(t *testing.T) {
	for _, target := range []string{"pkg/mapedit", "pkg/ui", "pkg/sim", "pkg/game", "pkg/random"} {
		if vs := Check(map[string][]string{"pkg/editormcp": {ModulePath + "/" + target}}); len(vs) != 1 {
			t.Fatal("protocol leaf gained a state dependency", target, vs)
		}
	}
	source := "package editormcp;import \"crypto/rand\";var _=rand.Read"
	if vs := CheckRandomness(map[string]string{"pkg/editormcp/http.go": source}); len(vs) != 0 {
		t.Fatal(vs)
	}
	if vs := CheckRandomness(map[string]string{"pkg/editormcp/other.go": source}); len(vs) != 1 {
		t.Fatal("credential entropy allowance reached another file", vs)
	}
	source = "package editormcp;import \"math/rand\";var _=rand.Intn"
	if vs := CheckRandomness(map[string]string{"pkg/editormcp/http.go": source}); len(vs) != 1 {
		t.Fatal("credential entropy allowance admitted a predictable generator", vs)
	}
}
