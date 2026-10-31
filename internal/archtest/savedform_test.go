package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"
)

// savedFormOwners names, for each part of a SAV saved form, the one
// function that may reference it; "" is package scope, where the form rows
// name their parts. Every SAVE takes its form from its game's row of the
// document's kind, which the producer applies.
var savedFormOwners = map[string]string{
	"DocumentInProfile":      "savedForm.write",
	"writeSecondTownHolders": "savedForm.write",
	"StoreEngineDocument":    "FrontEnd.ExportCurrentSave",
	"saveForm":               "FrontEnd.ExportCurrentSave",
	"secondCampaignTail":     "",
	"secondFormKeeps":        "",
	"secondFormHolders":      "",
}

// savedFormBypasses reports each declaration of the retired town form, each
// saveForm that does not read a row of a saved form table, and each
// reference to a form part outside its owner, in the production files of
// every package but the SAV codec.
func savedFormBypasses(t *testing.T, files map[string]string) (findings []string, tables int) {
	t.Helper()
	parsed := map[string]*ast.File{}
	rows := map[string]bool{}
	for name, source := range files {
		if strings.HasPrefix(name, "pkg/formats/sav/") {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), name, source, parser.SkipObjectResolution)
		if err != nil {
			t.Fatal(err)
		}
		parsed[name] = f
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, value := range vs.Values {
					if lit, ok := value.(*ast.CompositeLit); ok && i < len(vs.Names) {
						if typ, ok := lit.Type.(*ast.Ident); ok && typ.Name == "savedForms" {
							rows[vs.Names[i].Name] = true
						}
					}
				}
			}
		}
	}
	for name, f := range parsed {
		for _, decl := range f.Decls {
			owner := ""
			if fn, ok := decl.(*ast.FuncDecl); ok {
				owner = savedFormFuncName(fn)
				if fn.Name.Name == "secondTownForm" {
					findings = append(findings, name+":secondTownForm is declared")
				}
				if fn.Recv != nil && fn.Name.Name == "saveForm" && !readsSavedFormRow(fn, rows) {
					findings = append(findings, name+":"+owner+" does not read a saved form row")
				}
			}
			savedFormRefs(decl, func(used string) {
				if allowed, listed := savedFormOwners[used]; listed && allowed != owner {
					findings = append(findings, name+":"+owner+" references "+used)
				}
			})
		}
	}
	sort.Strings(findings)
	return findings, len(rows)
}

// savedFormRefs calls visit with each name n references: a selector's name
// and its operand's, every other identifier but a declared one (a function's,
// a field's, a parameter's or a variable's name).
func savedFormRefs(n ast.Node, visit func(string)) {
	ast.Inspect(n, func(m ast.Node) bool {
		switch m := m.(type) {
		case *ast.SelectorExpr:
			visit(m.Sel.Name)
			savedFormRefs(m.X, visit)
			return false
		case *ast.Ident:
			visit(m.Name)
		case *ast.FuncDecl:
			if m.Recv != nil {
				savedFormRefs(m.Recv, visit)
			}
			savedFormRefs(m.Type, visit)
			if m.Body != nil {
				savedFormRefs(m.Body, visit)
			}
			return false
		case *ast.Field:
			savedFormRefs(m.Type, visit)
			return false
		case *ast.ValueSpec:
			if m.Type != nil {
				savedFormRefs(m.Type, visit)
			}
			for _, v := range m.Values {
				savedFormRefs(v, visit)
			}
			return false
		}
		return true
	})
}

// savedFormFuncName is Recv.Name for a method, else Name.
func savedFormFuncName(fn *ast.FuncDecl) string {
	if fn.Recv == nil || len(fn.Recv.List) == 0 {
		return fn.Name.Name
	}
	typ := fn.Recv.List[0].Type
	if star, ok := typ.(*ast.StarExpr); ok {
		typ = star.X
	}
	if id, ok := typ.(*ast.Ident); ok {
		return id.Name + "." + fn.Name.Name
	}
	return fn.Name.Name
}

// readsSavedFormRow reports whether a saveForm's body is one return of a
// row of a saved form table, written: return table[kind].write(...).
func readsSavedFormRow(fn *ast.FuncDecl, rows map[string]bool) bool {
	if fn.Body == nil || len(fn.Body.List) != 1 {
		return false
	}
	ret, ok := fn.Body.List[0].(*ast.ReturnStmt)
	if !ok || len(ret.Results) != 1 {
		return false
	}
	call, ok := ret.Results[0].(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "write" {
		return false
	}
	index, ok := sel.X.(*ast.IndexExpr)
	if !ok {
		return false
	}
	table, ok := index.X.(*ast.Ident)
	return ok && rows[table.Name]
}

func TestSavedFormScanNamesEachBypass(t *testing.T) {
	got, tables := savedFormBypasses(t, map[string]string{
		"pkg/game/savform.go": "package game\n" +
			"var firstSavedForms = savedForms{townRecord: {records: sav.FirstGame}}\n" +
			"var secondSavedForms = savedForms{townRecord: {records: sav.SecondGame, holders: secondFormHolders, tail: secondCampaignTail}}\n" +
			"func (form savedForm) write(f *FrontEnd, s Snapshot, doc sav.DocumentData) { writeSecondTownHolders(&doc, nil, s, nil); sav.DocumentInProfile(doc, nil, sav.ProfileKeeps{}) }\n" +
			"func secondTownForm(s Snapshot) { secondCampaignTail(s) }\n",
		"pkg/game/campaignfirst.go": "package game\n" +
			"func (firstCampaignRules) saveForm(f *FrontEnd, s Snapshot, doc sav.DocumentData) (sav.DocumentData, bool, error) { return firstSavedForms[savedFormKind(s)].write(f, s, doc) }\n",
		"pkg/game/campaignsecond.go": "package game\n" +
			"func (secondCampaignRules) saveForm(f *FrontEnd, s Snapshot, doc sav.DocumentData) (sav.DocumentData, bool, error) { return doc, false, nil }\n",
		"pkg/game/currentsave.go": "package game\n" +
			"func (f *FrontEnd) ExportCurrentSave(s Snapshot) { campaignOf(s.game).saveForm(f, s, doc); sav.StoreEngineDocument(native, raw) }\n" +
			"func (f *FrontEnd) other(s Snapshot) { sav.StoreEngineDocument(native, raw); _ = secondFormKeeps }\n",
		"cmd/savtool/main.go":            "package main\nfunc run() { sav.DocumentInProfile(doc, sav.SecondGame, sav.ProfileKeeps{}) }\n",
		"pkg/formats/sav/engine_form.go": "package sav\nfunc x() { DocumentInProfile(d, p, k); StoreEngineDocument(d, e) }\n",
	})
	want := []string{
		"cmd/savtool/main.go:run references DocumentInProfile",
		"pkg/game/campaignsecond.go:secondCampaignRules.saveForm does not read a saved form row",
		"pkg/game/currentsave.go:FrontEnd.other references StoreEngineDocument",
		"pkg/game/currentsave.go:FrontEnd.other references secondFormKeeps",
		"pkg/game/savform.go:secondTownForm is declared",
		"pkg/game/savform.go:secondTownForm references secondCampaignTail",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") || tables != 2 {
		t.Fatalf("savedFormBypasses = %d tables\n%s", tables, strings.Join(got, "\n"))
	}
}

// Every SAVE takes its saved form from its game's row of the document's
// kind: the retired town form is gone, each saveForm reads a row, and each
// form part is referenced only by its owner.
func TestEverySaveTakesItsSavedFormFromItsRow(t *testing.T) {
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
	for _, name := range []string{"pkg/game/savform.go", "pkg/game/currentsave.go", "pkg/game/campaignfirst.go", "pkg/game/campaignsecond.go"} {
		if _, ok := files[name]; !ok {
			t.Fatalf("%s is missing; the scan guards nothing", name)
		}
	}
	got, tables := savedFormBypasses(t, files)
	if tables != 2 {
		t.Fatalf("the scan finds %d saved form tables, want the two games'", tables)
	}
	if len(got) != 0 {
		t.Fatalf("saved form bypasses:\n%s", strings.Join(got, "\n"))
	}
}
