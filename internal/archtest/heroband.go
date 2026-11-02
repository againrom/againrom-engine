package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The hero-band scan: which actor answers a script's 10001+k is decided by
// one binder, mapload.BindHeroBand, over the identity source an edition's
// HeroBand row names. In pkg it refuses, outside the site each is allowed:
//   - a read of an edition's HeroBand, outside the binder's heroBand;
//   - a HeroBand row constant, outside pkg/base and the binder's source table;
//   - ResolveHero, outside the role source's bind;
//   - HeroOrdinalMatches, outside ResolveHero;
//   - BindsByIdentity, and a HeroIdentity compared with zero, anywhere:
//     both are an edition test outside the binder;
//   - the first game's edition, Game("").Edition() or GameROM1.Edition(),
//     outside mapload.TableEdition, the one rule for a table read without
//     an edition, and the two sites that are no table's.

const heroBandBinder = "pkg/mapload/partyhumanrow.go"

type heroBandShape struct {
	reason string
	// allowed reports whether a use in file, inside the top-level
	// declaration named context, is the shape's own site.
	allowed func(file, context string) bool
}

func heroBandSite(file, context string) func(string, string) bool {
	return func(f, c string) bool { return f == file && c == context }
}

var heroBandShapes = map[string]heroBandShape{
	"HeroBand": {
		"an edition's identity source is read here; ask mapload.BindHeroBand or mapload.PartyIdentity",
		heroBandSite(heroBandBinder, "heroBand")},
	"HeroBandRole": {
		"an identity source is named here; a caller asks the binder, which reads the edition's row",
		heroBandRow},
	"HeroBandIdentity": {
		"an identity source is named here; a caller asks the binder, which reads the edition's row",
		heroBandRow},
	"ResolveHero": {
		"the hero ordinal role test is run here; the role source's bind is its one caller, reached through mapload.BindHeroBand",
		heroBandSite(heroBandBinder, "roleBand.bind")},
	"HeroOrdinalMatches": {
		"a hero ordinal is tested here; the role source reaches it through ResolveHero only",
		heroBandSite("pkg/data/heroordinal.go", "NPCDefs.ResolveHero")},
	"BindsByIdentity": {
		"an edition test decides the hero band here; ask mapload.BindHeroBand or mapload.PartyIdentity",
		func(string, string) bool { return false }},
	"HeroIdentity == 0": {
		"a hero identity of zero is read as the role test here; the edition's HeroBand row names the source, through the binder",
		func(string, string) bool { return false }},
	"Game(\"\").Edition()": {
		"the first game's edition stands in for a table's here; ask mapload.TableEdition",
		heroBandFirstEdition},
}

// heroBandFirstEdition admits the first game's edition in pkg/base, in
// mapload.TableEdition, in the first game's cutscene bank and for a nil
// mission notices, which has no table.
func heroBandFirstEdition(file, context string) bool {
	return strings.HasPrefix(file, "pkg/base/") ||
		file == heroBandBinder && context == "TableEdition" ||
		file == "pkg/game/cutscene.go" && context == "OpenCutscenes" ||
		file == "pkg/game/campaignservice.go" && context == "missionNotices.edition"
}

func heroBandRow(file, context string) bool {
	return strings.HasPrefix(file, "pkg/base/") || file == heroBandBinder && context == "heroBandSources"
}

// CheckHeroBand reports every hero-band decision outside the binder. files
// is keyed by module-relative slash path.
func CheckHeroBand(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the hero-band scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	sites := map[string]int{}
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, unit := range declUnits(f) {
			context := unit.context
			ast.Inspect(unit.node, func(n ast.Node) bool {
				shape, node := heroBandUse(n)
				if shape == "" {
					return true
				}
				if heroBandShapes[shape].allowed(name, context) {
					sites[shape]++
				} else {
					vs = append(vs, Violation{
						From:   name + ":" + strconv.Itoa(fset.Position(node.Pos()).Line),
						Reason: heroBandShapes[shape].reason,
					})
				}
				_, selector := n.(*ast.SelectorExpr)
				return !selector
			})
		}
	}
	if _, ok := files[heroBandBinder]; ok {
		for _, shape := range []string{"HeroBand", "ResolveHero"} {
			if sites[shape] == 0 {
				vs = append(vs, Violation{From: heroBandBinder, Reason: "the binder's " + shape + " site is gone - the hero-band scan would hold nothing"})
			}
		}
	}
	return vs
}

// heroBandUse names the hero-band shape n is, with the node to report.
func heroBandUse(n ast.Node) (string, ast.Node) {
	switch n := n.(type) {
	case *ast.SelectorExpr:
		if _, ok := heroBandShapes[n.Sel.Name]; ok {
			return n.Sel.Name, n
		}
	case *ast.Ident:
		if n.Name == "HeroBandRole" || n.Name == "HeroBandIdentity" || n.Name == "BindsByIdentity" {
			return n.Name, n
		}
	case *ast.CallExpr:
		if firstEditionCall(n) {
			return "Game(\"\").Edition()", n
		}
	case *ast.BinaryExpr:
		if (n.Op == token.EQL || n.Op == token.NEQ) &&
			(namesHeroIdentity(n.X) && isZero(n.Y) || namesHeroIdentity(n.Y) && isZero(n.X)) {
			return "HeroIdentity == 0", n
		}
	}
	return "", nil
}

// firstEditionCall reports whether call is Edition() on Game(""),
// base.Game(""), GameROM1 or base.GameROM1.
func firstEditionCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != "Edition" || len(call.Args) != 0 {
		return false
	}
	switch x := sel.X.(type) {
	case *ast.Ident:
		return x.Name == "GameROM1"
	case *ast.SelectorExpr:
		return x.Sel.Name == "GameROM1"
	case *ast.CallExpr:
		if len(x.Args) != 1 {
			return false
		}
		lit, ok := x.Args[0].(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING || lit.Value != `""` {
			return false
		}
		switch f := x.Fun.(type) {
		case *ast.Ident:
			return f.Name == "Game"
		case *ast.SelectorExpr:
			return f.Sel.Name == "Game"
		}
	}
	return false
}

func namesHeroIdentity(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.SelectorExpr:
		return e.Sel.Name == "HeroIdentity"
	case *ast.Ident:
		return e.Name == "HeroIdentity"
	}
	return false
}

func isZero(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value == "0"
}

// declUnit is one top-level function, method or spec with its name.
type declUnit struct {
	context string
	node    ast.Node
}

// declUnits are the top-level units of f: Recv.Name for a method, the name
// for a function, the first name of each var, const or type spec.
func declUnits(f *ast.File) []declUnit {
	var out []declUnit
	for _, decl := range f.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			name := d.Name.Name
			if d.Recv != nil && len(d.Recv.List) == 1 {
				t := d.Recv.List[0].Type
				if star, ok := t.(*ast.StarExpr); ok {
					t = star.X
				}
				if id, ok := t.(*ast.Ident); ok {
					name = id.Name + "." + name
				}
			}
			out = append(out, declUnit{name, d})
		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					out = append(out, declUnit{s.Names[0].Name, s})
				case *ast.TypeSpec:
					out = append(out, declUnit{s.Name.Name, s})
				}
			}
		}
	}
	return out
}
