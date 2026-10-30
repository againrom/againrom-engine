package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

func campaignActorViolations(files map[string]string) []string {
	var out []string
	for name, source := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, 0)
		if err != nil {
			out = append(out, err.Error())
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if fn.Name.Name == "CampaignActor" && name != "pkg/mapload/campaignactor.go" {
				out = append(out, name+": campaign actor resolver has another owner")
			}
			switch fn.Name.Name {
			case "CharacterMember", "buildMercenarySquad", "buildSiegeSquad", "siegeHireMember":
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							switch sel.Sel.Name {
							case "NewHumanDef", "NewUnitDef", "HumanRowEquipment":
								out = append(out, name+": "+fn.Name.Name+" decodes an actor outside CampaignActor")
							}
						}
					}
					if literal, ok := n.(*ast.CompositeLit); ok {
						typ := literal.Type
						if sel, ok := typ.(*ast.SelectorExpr); ok {
							typ = sel.Sel
						}
						if id, ok := typ.(*ast.Ident); ok && id.Name == "PartyMember" {
							out = append(out, name+": "+fn.Name.Name+" builds a campaign member outside CampaignActor")
						}
					}
					return true
				})
			}
		}
	}
	return out
}

func TestCampaignActorEntryPointsDelegateConstruction(t *testing.T) {
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
	if !strings.Contains(files["pkg/mapload/campaignactor.go"], "func CampaignActor(") {
		t.Fatal("campaign actor resolver missing")
	}
	for _, violation := range campaignActorViolations(files) {
		t.Error(violation)
	}
	for _, body := range []string{
		"func buildSiegeSquad() { _ = mapload.PartyMember{} }",
		"func CharacterMember() { data.NewHumanDef() }",
		"func siegeHireMember() { data.NewUnitDef() }",
	} {
		if len(campaignActorViolations(map[string]string{"pkg/game/copy.go": "package game;" + body})) == 0 {
			t.Fatal("actor builder guard admitted", body)
		}
	}
}
