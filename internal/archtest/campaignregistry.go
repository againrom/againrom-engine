package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// LoadCampaignReleaseTests reads pkg/game/campaign*_release_test.go.
func LoadCampaignReleaseTests(moduleRoot string) (map[string]string, error) {
	paths, err := filepath.Glob(filepath.Join(moduleRoot, "pkg", "game", "campaign*_release_test.go"))
	if err != nil {
		return nil, err
	}
	out := make(map[string]string)
	for _, p := range paths {
		b, rerr := os.ReadFile(p)
		if rerr != nil {
			return nil, rerr
		}
		out["pkg/game/"+filepath.Base(p)] = string(b)
	}
	return out, nil
}

// CheckCampaignRegistryReaders reports each reg.Parse in those tests.
func CheckCampaignRegistryReaders(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the campaign registry scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		base := filepath.Base(name)
		if !strings.HasPrefix(base, "campaign") || !strings.HasSuffix(base, "_release_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Parse" {
				return true
			}
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "reg" {
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(call.Pos()).Line),
					Reason: "reg.Parse in a campaign release test; read the registry through installedCampaign",
				})
			}
			return true
		})
	}
	return vs
}
