package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"strings"
	"testing"
)

// editorCommandTable is the one file of the map editor that calls the host
// operations the editor command table owns.
const editorCommandTable = "pkg/ui/mapeditorcommands.go"

// editorCommandOps are the host operations the table owns; editorEditsOps
// are those named on a document's Edits directly.
var (
	editorCommandOps = map[string]bool{"Paint": true, "AddRecord": true, "SetObject": true, "SetFields": true, "SetTextField": true,
		"WriteTrigger": true, "SetDropCell": true, "AddDropZone": true, "MoveDropZone": true, "DeleteDropZone": true,
		"SetWorkspaceField": true, "RandomTerrain": true, "RandomHeights": true}
	editorEditsOps = map[string]bool{"Move": true, "Clone": true, "Delete": true, "SetField": true, "Undo": true, "Redo": true}
)

func editorCommandViolations(files map[string]string) []string {
	var out []string
	for p, source := range files {
		base := path.Base(p)
		if strings.HasSuffix(p, "_test.go") || path.Dir(p) != "pkg/ui" || p == editorCommandTable ||
			!strings.HasPrefix(base, "mapeditor") && !strings.HasPrefix(base, "editor") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, source, 0)
		if err != nil {
			out = append(out, p+": "+err.Error())
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			s, ok := n.(*ast.SelectorExpr)
			if !ok {
				return true
			}
			on, _ := s.X.(*ast.SelectorExpr)
			if editorCommandOps[s.Sel.Name] || editorEditsOps[s.Sel.Name] && on != nil && on.Sel.Name == "Edits" {
				out = append(out, p+": "+s.Sel.Name+" is an edit of the editor command table")
			}
			return true
		})
	}
	return out
}

func TestEditorEditsGoThroughTheCommandTable(t *testing.T) {
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
	if _, ok := files[editorCommandTable]; !ok {
		t.Fatal("the command table file is missing", editorCommandTable)
	}
	for _, v := range editorCommandViolations(files) {
		t.Error(v)
	}
	for _, source := range []string{`package ui;func f(e *MapEditor){e.doc.Edits.Delete("Unit",0)}`, `package ui;func f(h InspectionBrush){h.Paint(BrushStroke{})}`, `package ui;func f(h InspectionWholeMap){run:=h.RandomTerrain;_=run}`} {
		if len(editorCommandViolations(map[string]string{"pkg/ui/mapeditorrecords.go": source})) == 0 {
			t.Fatal("guard accepted an edit beside the command table", source)
		}
		if len(editorCommandViolations(map[string]string{editorCommandTable: source})) != 0 {
			t.Fatal("guard refused the command table itself", source)
		}
	}
}

// editorAddWalkFiles are the command table's files; add_record and add_steps
// ask the Add window's walk (addOffer, addRecord) for each step's offer, its
// skip and the record, so no file of the table asks a step's offer, reads its
// Absent mark or assembles a new record itself.
var editorAddWalkFiles = []string{editorCommandTable, "pkg/ui/mapeditorcommandread.go"}

func editorAddWalkViolations(files map[string]string) []string {
	var out []string
	for _, p := range editorAddWalkFiles {
		source, ok := files[p]
		if !ok {
			out = append(out, p+": missing")
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), p, source, 0)
		if err != nil {
			out = append(out, p+": "+err.Error())
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if n.Sel.Name == "NewRecordStep" || n.Sel.Name == "Absent" {
					out = append(out, p+": "+n.Sel.Name+" is the Add window's walk")
				}
			case *ast.CompositeLit:
				if id, ok := n.Type.(*ast.Ident); ok && id.Name == "InspectionNewRecord" && len(n.Elts) > 0 {
					out = append(out, p+": an InspectionNewRecord is assembled by the Add window's addRecord")
				}
			}
			return true
		})
	}
	return out
}

// editorPlacementFile is the placement preview's file; a click on a cell and
// the preview take the record from addRecord (addPlacement, in the Add
// window's file), so the file asks no step's offer and fills no record.
const editorPlacementFile = "pkg/ui/mapeditorplacement.go"

func editorPlacementViolations(files map[string]string) []string {
	source, ok := files[editorPlacementFile]
	if !ok {
		return []string{editorPlacementFile + ": missing"}
	}
	f, err := parser.ParseFile(token.NewFileSet(), editorPlacementFile, source, 0)
	if err != nil {
		return []string{editorPlacementFile + ": " + err.Error()}
	}
	var out []string
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			if n.Sel.Name == "NewRecordStep" || n.Sel.Name == "Values" {
				out = append(out, editorPlacementFile+": "+n.Sel.Name+" is the Add window's record assembly")
			}
		case *ast.CompositeLit:
			if id, ok := n.Type.(*ast.Ident); ok && id.Name == "InspectionFieldValue" {
				out = append(out, editorPlacementFile+": an InspectionFieldValue is assembled by the Add window's addRecord")
			}
		}
		return true
	})
	return out
}

func TestEditorPlacementTakesTheRecordFromAddRecord(t *testing.T) {
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
	for _, v := range editorPlacementViolations(files) {
		t.Error(v)
	}
	for _, source := range []string{`package ui;func f(h InspectionNewRecords){h.NewRecordStep(InspectionNewRecord{}, "tier")}`, `package ui;func f(r *InspectionNewRecord){r.Values=nil}`, `package ui;var v = InspectionFieldValue{Key: "tier"}`} {
		if len(editorPlacementViolations(map[string]string{editorPlacementFile: source})) == 0 {
			t.Fatal("guard accepted a second record assembly in the placement file", source)
		}
	}
	if v := editorPlacementViolations(map[string]string{editorPlacementFile: `package ui;func f(e *MapEditor) InspectionNewRecord {return e.addPlacement(image.Point{})}`}); len(v) != 0 {
		t.Fatal("guard refused a call of addPlacement", v)
	}
}

func TestEditorAddRecordAsksTheAddWindowsWalk(t *testing.T) {
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
	for _, v := range editorAddWalkViolations(files) {
		t.Error(v)
	}
	ok := map[string]string{"pkg/ui/mapeditorcommandread.go": "package ui"}
	for _, source := range []string{`package ui;func f(h InspectionNewRecords){h.NewRecordStep(InspectionNewRecord{}, "tier")}`, `package ui;func f(o InspectionNewStep) bool {return o.Absent}`, `package ui;var r = InspectionNewRecord{Kind: "Unit"}`} {
		ok[editorCommandTable] = source
		if len(editorAddWalkViolations(ok)) == 0 {
			t.Fatal("guard accepted a second Add walk in the command table", source)
		}
	}
	ok[editorCommandTable] = `package ui;func f() InspectionNewRecord {return InspectionNewRecord{}}`
	if v := editorAddWalkViolations(ok); len(v) != 0 {
		t.Fatal("guard refused an empty record", v)
	}
}
