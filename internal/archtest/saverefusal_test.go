package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

var saveRefusalRules = map[string]bool{"townSavePoint": true, "exportable": true, "refuseCompleted": true}

// checkSaveRefusal reports an error a save point rule builds and a refusal
// row outside pkg/game/saverefusal.go.
func checkSaveRefusal(files map[string]string) (rules int, vs []string) {
	for name, src := range files {
		if !strings.HasPrefix(name, "pkg/game/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, src, parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, name+": "+err.Error())
			continue
		}
		for _, d := range f.Decls {
			fn, _ := d.(*ast.FuncDecl)
			rule := fn != nil && fn.Recv != nil && saveRefusalRules[fn.Name.Name]
			if rule {
				rules++
			}
			ast.Inspect(d, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.CallExpr:
					sel, ok := x.Fun.(*ast.SelectorExpr)
					if !ok || !rule {
						return true
					}
					if pkg, ok := sel.X.(*ast.Ident); ok && (pkg.Name == "errors" && sel.Sel.Name == "New" || pkg.Name == "fmt" && sel.Sel.Name == "Errorf") {
						vs = append(vs, name+": "+fn.Name.Name+" builds an error; a closed save point is a row of its game's refusal table")
					}
				case *ast.CompositeLit:
					if id, ok := x.Type.(*ast.Ident); ok && id.Name == "saveRefusal" && name != "pkg/game/saverefusal.go" {
						vs = append(vs, name+": writes a saveRefusal row outside the refusal tables in pkg/game/saverefusal.go")
					}
				}
				return true
			})
		}
	}
	return rules, vs
}

func TestCheckSaveRefusal(t *testing.T) {
	cases := []struct {
		name, file, src string
		bad             bool
	}{
		{"a table row", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) townSavePoint(f *FrontEnd) error { return firstRefusals.refuse(refusedNoGame) }\n", false},
		{"a built refusal", "pkg/game/campaignfirst.go",
			"package game\nfunc (firstCampaignRules) townSavePoint(f *FrontEnd) error { return errors.New(\"no\") }\n", true},
		{"a formatted refusal", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) exportable(s Snapshot) error { return fmt.Errorf(\"no %d\", 1) }\n", true},
		{"a row outside the table", "pkg/game/campaignsecond.go",
			"package game\nfunc (secondCampaignRules) exportable(s Snapshot) error { return saveRefusal{reasonWord: \"w\"} }\n", true},
		{"an error elsewhere", "pkg/game/a.go",
			"package game\nfunc load() error { return errors.New(\"no\") }\n", false},
	}
	for _, tc := range cases {
		_, vs := checkSaveRefusal(map[string]string{tc.file: tc.src})
		if (len(vs) != 0) != tc.bad {
			t.Errorf("%s: violations %v, want bad %v", tc.name, vs, tc.bad)
		}
	}
}

func TestSaveRefusalIsOneTable(t *testing.T) {
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
	rules, vs := checkSaveRefusal(files)
	if rules < 6 {
		t.Fatalf("save point rule methods read = %d, want at least 6 (three per game); the scan guards nothing", rules)
	}
	for _, v := range vs {
		t.Error(v)
	}
}
