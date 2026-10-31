package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// Outside lifeStateRuleFile, production pkg/sim, pkg/game and pkg/ui test no
// health against a life threshold by hand; they ask a sim.LifeState.
const lifeStateRuleFile = "pkg/sim/world.go"

// lifeStateKept names each ordered comparison of a health value with an
// integer constant that is not a life threshold, or that no LifeState can be
// asked from. The key is "<file>:<function>:<constant as written>"; the value
// is the reason.
var lifeStateKept = map[string]string{
	"pkg/game/worldsave.go:projectDeadActors:-32768":                            "the 16-bit bounds of a saved health word, a range check",
	"pkg/game/worldsave.go:projectDeadActors:32767":                             "the 16-bit bounds of a saved health word, a range check",
	"pkg/sim/originalpools.go:ImportOriginalActorPools:-32768":                  "the 16-bit bounds of a saved health word, a range check",
	"pkg/sim/originalpools.go:ImportOriginalActorPools:32767":                   "the 16-bit bounds of a saved health word, a range check",
	"pkg/game/currenthealthadmission.go:currentAdmissionHealthByIdentity:65535": "the 16-bit width of a parsed health word, a range check",
	"pkg/ui/sound.go:stepSound:0":                                               "pkg/ui takes no import from pkg/sim, so the sound step keeps its stored-health bleed band",
	"pkg/ui/sound.go:stepSound:BleedCeiling":                                    "pkg/ui takes no import from pkg/sim, so the sound step keeps its stored-health bleed band",
}

var lifeStateIntTypes = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
}

// lifeStateHealthName reports whether a bare identifier names a health value.
func lifeStateHealthName(name string) bool {
	l := strings.ToLower(name)
	if strings.Contains(l, "max") {
		return false
	}
	return l == "hp" || l == "health" || strings.HasSuffix(l, "hp") && len(l) <= 9 || strings.HasPrefix(l, "health")
}

// CheckLifeState reports each ordered comparison, outside lifeStateRuleFile,
// of a health value with an integer constant expression. A health value is a
// .HP, .hp or .Health selector, a Stats[8] word, an integer conversion of one,
// a local assigned from one, or a bare identifier that names one. A constant
// expression is a literal, a negated literal, a const identifier or integer
// arithmetic and conversion over those. Comparing two health values passes.
func CheckLifeState(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the life-state scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	scanned := func(name string) bool {
		return (strings.HasPrefix(name, "pkg/sim/") || strings.HasPrefix(name, "pkg/game/") || strings.HasPrefix(name, "pkg/ui/")) &&
			!strings.HasSuffix(name, "_test.go")
	}
	var vs []Violation
	parsed := map[string]*ast.File{}
	fsets := map[string]*token.FileSet{}
	consts := map[string]bool{}
	for _, name := range names {
		if !scanned(name) {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		parsed[name], fsets[name] = f, fset
		ast.Inspect(f, func(n ast.Node) bool {
			if gd, ok := n.(*ast.GenDecl); ok && gd.Tok == token.CONST {
				for _, spec := range gd.Specs {
					for _, id := range spec.(*ast.ValueSpec).Names {
						consts[id.Name] = true
					}
				}
			}
			return true
		})
	}
	used := map[string]bool{}
	for _, name := range names {
		f := parsed[name]
		if f == nil || name == lifeStateRuleFile {
			continue
		}
		fset := fsets[name]
		for _, decl := range f.Decls {
			fn := ""
			if fd, ok := decl.(*ast.FuncDecl); ok {
				fn = fd.Name.Name
			}
			locals := lifeStateLocals(decl)
			isHealth := func(x ast.Expr) bool { return lifeStateIsHealth(x, locals, true) }
			var isConst func(x ast.Expr) bool
			isConst = func(x ast.Expr) bool {
				switch x := x.(type) {
				case *ast.BasicLit:
					return x.Kind == token.INT
				case *ast.ParenExpr:
					return isConst(x.X)
				case *ast.UnaryExpr:
					return (x.Op == token.SUB || x.Op == token.ADD) && isConst(x.X)
				case *ast.Ident:
					return consts[x.Name]
				case *ast.SelectorExpr:
					_, ok := x.X.(*ast.Ident)
					return ok && consts[x.Sel.Name]
				case *ast.CallExpr:
					id, ok := x.Fun.(*ast.Ident)
					return ok && lifeStateIntTypes[id.Name] && len(x.Args) == 1 && isConst(x.Args[0])
				case *ast.BinaryExpr:
					switch x.Op {
					case token.ADD, token.SUB, token.MUL, token.QUO, token.REM, token.SHL, token.SHR, token.AND, token.OR:
						return isConst(x.X) && isConst(x.Y)
					}
				}
				return false
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				b, ok := n.(*ast.BinaryExpr)
				if !ok {
					return true
				}
				switch b.Op {
				case token.LSS, token.LEQ, token.GTR, token.GEQ:
				default:
					return true
				}
				if !(isHealth(b.X) && isConst(b.Y) || isConst(b.X) && isHealth(b.Y)) {
					return true
				}
				c := b.Y
				if isConst(b.X) {
					c = b.X
				}
				key := name + ":" + fn + ":" + files[name][fset.Position(c.Pos()).Offset:fset.Position(c.End()).Offset]
				if _, kept := lifeStateKept[key]; kept {
					used[key] = true
					return true
				}
				vs = append(vs, Violation{
					From:   name + ":" + strconv.Itoa(fset.Position(b.Pos()).Line),
					Reason: "a health value compared with a constant outside " + lifeStateRuleFile + "; ask a sim.LifeState by name",
				})
				return true
			})
		}
	}
	keys := make([]string, 0, len(lifeStateKept))
	for key := range lifeStateKept {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		file := key[:strings.Index(key, ":")]
		if parsed[file] != nil && !used[key] {
			vs = append(vs, Violation{From: key, Reason: "a kept health comparison that no longer exists; drop it from lifeStateKept"})
		}
	}
	return vs
}

// lifeStateIsHealth reports whether x is a health value: a .HP or .hp
// selector, a Stats[8] word, an integer conversion of a health value, a local
// in locals and, when byName, a bare identifier that names health.
func lifeStateIsHealth(x ast.Expr, locals map[string]bool, byName bool) bool {
	switch x := x.(type) {
	case *ast.ParenExpr:
		return lifeStateIsHealth(x.X, locals, byName)
	case *ast.SelectorExpr:
		return x.Sel.Name == "HP" || x.Sel.Name == "hp"
	case *ast.IndexExpr:
		sel, ok := x.X.(*ast.SelectorExpr)
		lit, ok2 := x.Index.(*ast.BasicLit)
		return ok && ok2 && sel.Sel.Name == "Stats" && lit.Value == "8"
	case *ast.CallExpr:
		id, ok := x.Fun.(*ast.Ident)
		return ok && lifeStateIntTypes[id.Name] && len(x.Args) == 1 && lifeStateIsHealth(x.Args[0], locals, byName)
	case *ast.Ident:
		return locals[x.Name] || byName && lifeStateHealthName(x.Name)
	}
	return false
}

// lifeStateLocals names the locals of one declaration that are assigned a
// health value, directly or through another such local.
func lifeStateLocals(decl ast.Node) map[string]bool {
	locals := map[string]bool{}
	isHealth := func(x ast.Expr) bool { return lifeStateIsHealth(x, locals, false) }
	for range 3 {
		ast.Inspect(decl, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				if len(n.Lhs) == len(n.Rhs) {
					for i, l := range n.Lhs {
						if id, ok := l.(*ast.Ident); ok && id.Name != "_" && isHealth(n.Rhs[i]) {
							locals[id.Name] = true
						}
					}
				}
			case *ast.ValueSpec:
				if len(n.Names) == len(n.Values) {
					for i, id := range n.Names {
						if isHealth(n.Values[i]) {
							locals[id.Name] = true
						}
					}
				}
			}
			return true
		})
	}
	return locals
}
