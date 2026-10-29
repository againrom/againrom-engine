package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestEquipmentEditorAndStockUseSharedBuilders(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/game/rearm.go"), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"EquipTarget": "EquipTarget", "simDerivedBlock": "DerivedBlock", "RearmWithLayers": "RearmWithLayers", "applyRearmLoadout": "ApplyRearmLoadout"}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || want[fn.Name.Name] == "" {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			if name, ok := selector.X.(*ast.Ident); ok && name.Name == "mapload" && selector.Sel.Name == want[fn.Name.Name] {
				found = true
			}
			if selector.Sel.Name == "Recompute" || selector.Sel.Name == "WeaponFromCode" || selector.Sel.Name == "ArmorFromCode" {
				t.Errorf("%s builds equipment instead of delegating", fn.Name.Name)
			}
			return true
		})
		if !found {
			t.Errorf("%s does not call its mapload builder", fn.Name.Name)
		}
		delete(want, fn.Name.Name)
	}
	if len(want) != 0 {
		t.Fatalf("equipment wrappers missing: %v", want)
	}
	f, err = parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/mod/items.go"), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	ast.Inspect(f, func(n ast.Node) bool {
		value, ok := n.(*ast.ValueSpec)
		if !ok {
			return true
		}
		for i, name := range value.Names {
			if name.Name != "ItemSlots" && name.Name != "AnchorSlots" {
				continue
			}
			count++
			if i >= len(value.Values) {
				t.Errorf("%s has no slot builder", name.Name)
				continue
			}
			call, ok := value.Values[i].(*ast.CallExpr)
			if !ok {
				t.Errorf("%s duplicates the slot table", name.Name)
				continue
			}
			builder, ok := call.Fun.(*ast.Ident)
			if !ok || builder.Name != "equipmentSlotNames" {
				t.Errorf("%s has another slot builder", name.Name)
			}
		}
		return true
	})
	if count != 2 {
		t.Fatalf("slot declarations=%d, want 2", count)
	}
	f, err = parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/game/mapinspectionsheet.go"), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	count = 0
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 1 {
				return true
			}
			literal, ok := call.Args[0].(*ast.BasicLit)
			if !ok || literal.Kind != token.STRING {
				return true
			}
			key, _ := strconv.Unquote(literal.Value)
			if strings.HasPrefix(key, "editor.unit.loot.") {
				count++
				if fn.Name.Name != "corpseDropLines" {
					t.Errorf("%s builds corpse loot outside corpseDropLines", fn.Name.Name)
				}
			}
			return true
		})
	}
	if count == 0 {
		t.Fatal("corpse loot builder has no lines")
	}
	f, err = parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/game/mapinspectionitemrecipe.go"), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	want = map[string]string{"itemRecipeFields": "Fields", "setUnitRecipeField": "setField"}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || want[fn.Name.Name] == "" {
			continue
		}
		found := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if ok && sel.Sel.Name == want[fn.Name.Name] {
				found = true
			}
			return true
		})
		if !found {
			t.Errorf("%s bypasses the recipe field builder", fn.Name.Name)
		}
		delete(want, fn.Name.Name)
	}
	if len(want) != 0 {
		t.Fatal("missing item recipe field routes", want)
	}
}

func TestNativeItemEncoderChecksTheExistingDecoder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, "pkg/game/mapinspectionequipment.go"), nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"ItemInstanceFromLoot": false, "ItemRecipeEqual": false}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != "writeStockItem" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			selector, ok := call.Fun.(*ast.SelectorExpr)
			if ok {
				if name, ok := selector.X.(*ast.Ident); ok && name.Name == "mapload" {
					if _, exists := want[selector.Sel.Name]; exists {
						want[selector.Sel.Name] = true
					}
				}
			}
			return true
		})
	}
	for builder, found := range want {
		if !found {
			t.Errorf("native item encoder must use mapload.%s", builder)
		}
	}
}
