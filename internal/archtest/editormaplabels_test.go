package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

func checkEditorMapLabels(files map[string]string) []string {
	var findings []string
	for name, source := range files {
		if !strings.HasPrefix(name, "pkg/ui/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.TypeSpec:
				st, ok := n.Type.(*ast.StructType)
				if !ok {
					break
				}
				rect, text, namedRect, namedText := false, false, false, false
				for _, field := range st.Fields.List {
					isRect := false
					if sel, ok := field.Type.(*ast.SelectorExpr); ok {
						isRect = sel.Sel.Name == "Rectangle"
					}
					id, ok := field.Type.(*ast.Ident)
					isText := ok && id.Name == "string"
					rect, text = rect || isRect, text || isText
					for _, id := range field.Names {
						namedRect = namedRect || isRect && strings.EqualFold(id.Name, "rect")
						namedText = namedText || isText && strings.EqualFold(id.Name, "text")
					}
				}
				label := namedRect && namedText || rect && text && strings.Contains(strings.ToLower(n.Name.Name), "label")
				if label && !mapLabelWidget(name, n.Name.Name) && (name != "pkg/ui/mapeditorlabels.go" || n.Name.Name != "mapLabel") {
					findings = append(findings, fmt.Sprintf("%s:%s: second map label row", name, n.Name.Name))
				}
			case *ast.FuncDecl:
				if n.Body == nil {
					break
				}
				measure, overlap, move := false, false, false
				ast.Inspect(n.Body, func(node ast.Node) bool {
					call, ok := node.(*ast.CallExpr)
					if !ok {
						return true
					}
					if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
						measure = measure || sel.Sel.Name == "Advance"
						overlap = overlap || sel.Sel.Name == "Overlaps"
					}
					return true
				})
				ast.Inspect(n.Body, func(node ast.Node) bool {
					assignment, ok := node.(*ast.AssignStmt)
					if !ok {
						return true
					}
					for i, rhs := range assignment.Rhs {
						call, ok := rhs.(*ast.CallExpr)
						if !ok || i >= len(assignment.Lhs) {
							continue
						}
						sel, ok := call.Fun.(*ast.SelectorExpr)
						if !ok || sel.Sel.Name != "Add" && sel.Sel.Name != "Sub" {
							continue
						}
						lhs, lok := assignment.Lhs[i].(*ast.Ident)
						receiver, rok := sel.X.(*ast.Ident)
						move = move || lok && rok && lhs.Name == receiver.Name
					}
					return true
				})
				if measure && overlap && move && (name != "pkg/ui/mapeditorlabels.go" || n.Name.Name != "mapLabels") {
					findings = append(findings, fmt.Sprintf("%s:%s: second map label layout (Advance, Overlaps, translated rectangle)", name, n.Name.Name))
				}
			}
			return true
		})
	}
	sort.Strings(findings)
	return findings
}

func mapLabelWidget(file, typ string) bool {
	// These rectangles belong to widgets or pointer tooltips, not map points.
	switch file + ":" + typ {
	case "pkg/ui/townpointer.go:townTipOwner", "pkg/ui/chargen.go:ChargenTip",
		"pkg/ui/tippanel.go:TipPanelView", "pkg/ui/widgetbutton.go:plaqueCaption",
		"pkg/ui/widgetdropdown.go:dropdown":
		return true
	}
	return false
}

func TestMapLabelsHaveOneBuilder(t *testing.T) {
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
	for _, finding := range checkEditorMapLabels(files) {
		t.Error(finding)
	}
}

func TestMapLabelsGuardNamesPlantedBuilders(t *testing.T) {
	for _, typ := range []string{"areaLabel", "recordMarkerLabel", "heightLabel", "copiedRow"} {
		t.Run(typ, func(t *testing.T) {
			files := map[string]string{"pkg/ui/extra.go": "package ui; type " + typ + " struct { rect image.Rectangle; text string }"}
			findings := checkEditorMapLabels(files)
			if len(findings) != 1 || !strings.Contains(findings[0], "extra.go:"+typ+": second map label row") {
				t.Fatalf("planted %s: %v", typ, findings)
			}
		})
	}
	for _, function := range []string{"areaLabels", "markerLabels", "heightLabels", "relocatedBoxes"} {
		t.Run(function, func(t *testing.T) {
			source := "package ui; func " + function + "() { w := f.Advance(text); r := image.Rect(0, 0, w, h); for _, l := range earlier { if r.Overlaps(l.rect) { r = r.Add(image.Pt(0, h)) } } }"
			findings := checkEditorMapLabels(map[string]string{"pkg/ui/extra.go": source})
			if len(findings) != 1 || !strings.Contains(findings[0], "extra.go:"+function+": second map label layout") {
				t.Fatalf("planted %s: %v", function, findings)
			}
		})
	}
	clean := map[string]string{
		"pkg/ui/mapeditorlabels.go": "package ui; type mapLabel struct { rect image.Rectangle; text string }; func mapLabels() { w := f.Advance(text); for r.Overlaps(old) { r = r.Add(step) }; _ = w }",
		"pkg/ui/adapter.go":         "package ui; func sourceRows() { w := f.Advance(text); _ = w }; type field struct { text string }",
	}
	if findings := checkEditorMapLabels(clean); len(findings) != 0 {
		t.Fatal(findings)
	}
	for _, file := range []string{"pkg/ui/extra.go", "pkg/ui/mapeditorlabels.go"} {
		findings := checkEditorMapLabels(map[string]string{file: "package ui; type extraLabel struct { Box image.Rectangle; Caption string }"})
		if len(findings) != 1 || !strings.Contains(findings[0], "extraLabel") {
			t.Fatalf("renamed fields in %s: %v", file, findings)
		}
	}
}
