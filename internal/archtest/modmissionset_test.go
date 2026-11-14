package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The mod mission set's one builder is mod.MissionData.Listed; the campaign
// flow's one constructor is newCampaignFlow; ByFileName only where listed.
func TestModMissionSetHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string][]string{
		"pkg/game/mapeditorcampaign.go": {"campaignRow"},
		"pkg/game/mapeditormod.go":      {"selected", "fileMissionFixed", "retainedPackageFiles"},
		"pkg/game/mapedittestplay.go":   {"testPlayAsDeclared"},
		"cmd/starter/app.go":            {"missionLabel"},
	}
	var files []string
	for _, dir := range []string{"pkg/game", "cmd/starter"} {
		found, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(dir), "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		files = append(files, found...)
	}
	flows := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.CompositeLit:
					if id, ok := n.Type.(*ast.Ident); ok && id.Name == "campaignFlow" {
						flows++
						if fn.Name.Name != "newCampaignFlow" {
							t.Errorf("%s: %s builds a campaignFlow; newCampaignFlow is its one constructor", rel, fn.Name.Name)
						}
					}
				case *ast.SelectorExpr:
					if n.Sel.Name == "ByFileName" && !slices.Contains(byName[rel], fn.Name.Name) {
						t.Errorf("%s: %s asks ByFileName; the mission set is mod.MissionData.Listed's", rel, fn.Name.Name)
					}
					if n.Sel.Name == "AdmitMaps" {
						t.Errorf("%s: %s calls AdmitMaps; Listed is the one join", rel, fn.Name.Name)
					}
				}
				return true
			})
		}
	}
	if flows != 1 {
		t.Fatalf("%d campaignFlow literals, want the one in newCampaignFlow", flows)
	}
}
