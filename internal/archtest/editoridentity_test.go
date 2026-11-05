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

func editorIdentityCalls(src string) map[string]map[string]int {
	f, err := parser.ParseFile(token.NewFileSet(), "", src, parser.SkipObjectResolution)
	if err != nil {
		return nil
	}
	out := map[string]map[string]int{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		calls := map[string]int{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				switch fun := call.Fun.(type) {
				case *ast.Ident:
					calls[fun.Name]++
				case *ast.SelectorExpr:
					calls[fun.Sel.Name]++
				}
			}
			return true
		})
		out[fn.Name.Name] = calls
	}
	return out
}

func TestEditorIdentityPictureAndPlacementShareTierBuilders(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range []struct {
		file, function, builder string
	}{
		{"pkg/game/tiers.go", "entityTiers", "placementTier"},
		{"pkg/game/tiers.go", "placementTier", "NewUnitDef"},
		{"pkg/game/mapinspectionidentity.go", "identityFigure", "placementTier"},
		{"pkg/game/mapinspectionidentity.go", "identityFigure", "unitPreview"},
		{"pkg/game/mapinspectionadd.go", "NewRecordPreview", "unitPreview"},
		{"pkg/game/mapinspectionadd.go", "unitPreview", "TierFrames"},
		{"pkg/game/mapinspectionadd.go", "NewRecordStep", "identityGroups"},
	} {
		src, err := os.ReadFile(filepath.Join(root, row.file))
		if err != nil {
			t.Fatal(err)
		}
		calls := editorIdentityCalls(string(src))
		if calls[row.function][row.builder] != 1 {
			t.Errorf("%s must call %s once", row.function, row.builder)
		}
		if row.builder == "TierFrames" || row.builder == "NewUnitDef" {
			for function, uses := range calls {
				if function != row.function && uses[row.builder] > 0 {
					t.Errorf("%s duplicates %s outside %s", function, row.builder, row.function)
				}
			}
		}
	}
}

func TestEditorIdentityBuilderGuardSeesAChangedCall(t *testing.T) {
	src := "package game; func identityFigure() { placementTier(); unitPreview() }"
	if editorIdentityCalls(src)["identityFigure"]["placementTier"] != 1 || editorIdentityCalls(strings.ReplaceAll(src, "placementTier", "otherTier"))["identityFigure"]["placementTier"] != 0 {
		t.Fatal("tier call guard does not distinguish a competing rule")
	}
}
