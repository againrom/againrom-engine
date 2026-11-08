package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The inn page's one builder of each part: the profile table, the function
// that picks a profile from it, the row producer and the function that
// calls it, and the table of named words a caption slot reads.
const (
	innProfileTable  = "hostInnProfiles"
	innProfilePicker = "hostInnProfile"
	innRowProducer   = "innPageRows"
	innRowCaller     = "innRows"
	innCaptionWords  = "namedMainWords"
)

// innProfileTypes are the profile types only the profile table builds.
var innProfileTypes = map[string]bool{"innProfile": true, "innRowProfile": true}

// innCaptionFields are the named words a tavern button caption reads; only
// the caption word table names them.
var innCaptionFields = map[string]bool{"TavernHire": true, "TavernFire": true, "TavernTalk": true, "TavernExit": true}

// innPageNameEntries are the functions that may still compare a room's page
// with the inn page names, each with the reason it is not an inn page fork.
var innPageNameEntries = map[string]string{
	"innPage":               "the one statement of the inn page key; a description may hold an inn page with no seats",
	"checkTownStateBinding": "the catalog check binding a page to its offer source, not a row, caption or profile",
}

// CheckInnPage reports, over pkg/game production sources as
// LoadDrawnTextSources returns them, each second builder of the inn page:
// a read of the profile table outside its picker, a profile literal, a call
// of the row producer outside its caller, a read of a named caption word
// outside the caption word table, and a comparison or case of a .Page
// selector with "tavern" or "inn" outside innPageNameEntries.
func CheckInnPage(files map[string]string) []Violation {
	names := make([]string, 0, len(files))
	for name := range files {
		if strings.HasPrefix(name, "pkg/game/") {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the inn page scan found nothing to read"}}
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		for _, decl := range f.Decls {
			owner, body := "", ast.Node(decl)
			switch d := decl.(type) {
			case *ast.FuncDecl:
				owner = d.Name.Name
			case *ast.GenDecl:
				owner = genDeclName(d)
			}
			vs = append(vs, innPageForks(body, owner, at)...)
		}
	}
	return vs
}

// genDeclName is the name a one-spec var declaration declares, else "".
func genDeclName(d *ast.GenDecl) string {
	if len(d.Specs) != 1 {
		return ""
	}
	if v, ok := d.Specs[0].(*ast.ValueSpec); ok && len(v.Names) == 1 {
		return v.Names[0].Name
	}
	return ""
}

func innPageForks(body ast.Node, owner string, at func(ast.Node) string) []Violation {
	var vs []Violation
	report := func(n ast.Node, reason string) { vs = append(vs, Violation{From: at(n), Reason: reason}) }
	ast.Inspect(body, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.Ident:
			if x.Name == innProfileTable && owner != innProfileTable && owner != innProfilePicker {
				report(x, owner+" reads "+innProfileTable+"; an inn profile is picked by "+innProfilePicker+" only")
			}
		case *ast.CompositeLit:
			if id, ok := x.Type.(*ast.Ident); ok && innProfileTypes[id.Name] && owner != innProfileTable {
				report(x, owner+" builds an "+id.Name+" literal; inn profiles are rows of "+innProfileTable)
			}
		case *ast.CallExpr:
			if id, ok := x.Fun.(*ast.Ident); ok && id.Name == innRowProducer && owner != innRowCaller {
				report(x, owner+" calls "+innRowProducer+"; the inn page rows are produced by "+innRowCaller+" only")
			}
		case *ast.SelectorExpr:
			if innCaptionFields[x.Sel.Name] && owner != innCaptionWords {
				report(x, owner+" reads the named word "+x.Sel.Name+"; a button caption comes from its button row through "+innCaptionWords)
			}
		case *ast.BinaryExpr:
			if (x.Op == token.EQL || x.Op == token.NEQ) && (innPageName(x.X, x.Y) || innPageName(x.Y, x.X)) {
				if _, ok := innPageNameEntries[owner]; !ok {
					report(x, owner+" compares a page with an inn page name; an inn page is a room with seats")
				}
			}
		case *ast.SwitchStmt:
			if sel, ok := x.Tag.(*ast.SelectorExpr); !ok || sel.Sel.Name != "Page" {
				return true
			}
			if _, ok := innPageNameEntries[owner]; ok {
				return true
			}
			for _, c := range x.Body.List {
				for _, e := range c.(*ast.CaseClause).List {
					if innPageLiteral(e) {
						report(e, owner+" switches a page on an inn page name; an inn page is a room with seats")
					}
				}
			}
		}
		return true
	})
	return vs
}

// innPageName reports whether page is a .Page selector and name an inn page
// name literal.
func innPageName(page, name ast.Expr) bool {
	sel, ok := page.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Page" && innPageLiteral(name)
}

func innPageLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && (lit.Value == `"tavern"` || lit.Value == `"inn"`)
}
