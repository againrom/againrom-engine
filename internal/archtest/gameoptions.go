package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
)

// The game option scan: a Game Options row is one gameOptionTable row in
// pkg/ui/gameoptions.go (page side, profile key) and one gameOptionRows row
// in pkg/game/gameoptions.go (game side). Production code in pkg/game and
// pkg/ui outside those two declarations decides nothing by an option's
// identity. A constant passed as a call argument (a shortcut naming its
// option) is not a decision. A GameOption value is a parameter, result,
// variable or range value declared with the type, read without go/types.

const (
	gameOptionConstFile = "pkg/ui/gameoptions.go"
	gameOptionTypeName  = "GameOption"
	gameOptionValuesTyp = "GameOptionValues"
	// gameMenuOptionBlock is the first option's page action, the start of
	// the block optionAction indexes.
	gameMenuOptionBlock = "gameMenuDayNight"
)

// gameOptionTables are the two table declarations, by file.
var gameOptionTables = map[string]string{
	"pkg/ui/gameoptions.go":   "gameOptionTable",
	"pkg/game/gameoptions.go": "gameOptionRows",
}

// CheckGameOptionRows reports, outside the two option tables: a switch over a
// GameOption value or conversion; a case naming a GameOption constant; a
// comparison with a constant, or of a GameOption value with a literal (a
// range test is two); arithmetic on a constant; an index by a constant; a
// positional GameOptionValues literal; a constant inside any other composite
// literal (an option list or a per-option mapping); a page action named for
// an option other than the block's first. files is keyed by
// module-relative slash path and must hold pkg/ui/gameoptions.go, whose
// GameOption constants it reads.
func CheckGameOptionRows(files map[string]string) []Violation {
	consts, err := gameOptionConstants(files[gameOptionConstFile])
	if err != nil || len(consts) == 0 {
		return []Violation{{From: gameOptionConstFile, Reason: fmt.Sprintf("no GameOption constants read (%v) - the game option scan has nothing to look for", err)}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if (strings.HasPrefix(name, "pkg/game/") || strings.HasPrefix(name, "pkg/ui/")) &&
			strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
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
		s := gameOptionScan{consts: consts, at: func(n ast.Node) string { return fmt.Sprintf("%s:%d", name, fset.Position(n.Pos()).Line) }}
		table := gameOptionTables[name]
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				s.typed = gameOptionTyped(d)
				if d.Body != nil {
					ast.Inspect(d.Body, s.visit)
				}
			case *ast.GenDecl:
				s.typed = nil
				for _, spec := range d.Specs {
					v, ok := spec.(*ast.ValueSpec)
					if ok && table != "" && len(v.Names) == 1 && v.Names[0].Name == table {
						continue
					}
					if ok {
						s.menuActions(v)
					}
					ast.Inspect(spec, s.visit)
				}
			}
		}
		vs = append(vs, s.found...)
	}
	return vs
}

// gameOptionConstants reads the exported constants of type GameOption.
func gameOptionConstants(src string) (map[string]bool, error) {
	if src == "" {
		return nil, fmt.Errorf("%s not given", gameOptionConstFile)
	}
	f, err := parser.ParseFile(token.NewFileSet(), gameOptionConstFile, src, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, decl := range f.Decls {
		d, ok := decl.(*ast.GenDecl)
		if !ok || d.Tok != token.CONST {
			continue
		}
		inBlock := false
		for _, spec := range d.Specs {
			v := spec.(*ast.ValueSpec)
			if v.Type != nil {
				id, ok := v.Type.(*ast.Ident)
				inBlock = ok && id.Name == gameOptionTypeName
			}
			for _, n := range v.Names {
				if inBlock && ast.IsExported(n.Name) && strings.HasPrefix(n.Name, gameOptionTypeName) {
					out[n.Name] = true
				}
			}
		}
	}
	return out, nil
}

// isGameOptionType reports ui.GameOption or, inside pkg/ui, GameOption.
func isGameOptionType(e ast.Expr, name string) bool {
	switch t := e.(type) {
	case *ast.Ident:
		return t.Name == name
	case *ast.SelectorExpr:
		return t.Sel.Name == name
	}
	return false
}

// gameOptionTyped is the set of names fn declares with the GameOption type.
func gameOptionTyped(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	fields := func(l *ast.FieldList) {
		if l == nil {
			return
		}
		for _, f := range l.List {
			if isGameOptionType(f.Type, gameOptionTypeName) {
				for _, n := range f.Names {
					out[n.Name] = true
				}
			}
		}
	}
	fields(fn.Recv)
	fields(fn.Type.Params)
	fields(fn.Type.Results)
	if fn.Body == nil {
		return out
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch s := n.(type) {
		case *ast.ValueSpec:
			if s.Type != nil && isGameOptionType(s.Type, gameOptionTypeName) {
				for _, n := range s.Names {
					out[n.Name] = true
				}
			}
		case *ast.AssignStmt:
			for i, rhs := range s.Rhs {
				if call, ok := rhs.(*ast.CallExpr); ok && len(call.Args) == 1 && isGameOptionType(call.Fun, gameOptionTypeName) && i < len(s.Lhs) {
					if id, ok := s.Lhs[i].(*ast.Ident); ok {
						out[id.Name] = true
					}
				}
			}
		case *ast.RangeStmt:
			if call, ok := s.X.(*ast.CallExpr); ok && isGameOptionType(call.Fun, "GameOptions") {
				if id, ok := s.Value.(*ast.Ident); ok {
					out[id.Name] = true
				}
			}
		}
		return true
	})
	return out
}

type gameOptionScan struct {
	consts map[string]bool
	typed  map[string]bool
	at     func(ast.Node) string
	found  []Violation
}

func (s *gameOptionScan) report(n ast.Node, reason string) {
	s.found = append(s.found, Violation{From: s.at(n), Reason: reason + "; make it a gameOptionTable or gameOptionRows row"})
}

// unwrapOption drops parentheses and one-argument conversions.
func unwrapOption(e ast.Expr) ast.Expr {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
		case *ast.CallExpr:
			if len(x.Args) != 1 {
				return e
			}
			switch fn := x.Fun.(type) {
			case *ast.Ident:
				if fn.Name != "int" && fn.Name != "uint8" && fn.Name != "int32" && fn.Name != gameOptionTypeName {
					return e
				}
			case *ast.SelectorExpr:
				if fn.Sel.Name != gameOptionTypeName {
					return e
				}
			default:
				return e
			}
			e = x.Args[0]
		default:
			return e
		}
	}
}

// constName is the GameOption constant e names, or "".
func (s *gameOptionScan) constName(e ast.Expr) string {
	switch x := unwrapOption(e).(type) {
	case *ast.Ident:
		if s.consts[x.Name] {
			return x.Name
		}
	case *ast.SelectorExpr:
		if _, ok := x.X.(*ast.Ident); ok && s.consts[x.Sel.Name] {
			return x.Sel.Name
		}
	}
	return ""
}

// typedValue reports a GameOption value: a typed name or a conversion.
func (s *gameOptionScan) typedValue(e ast.Expr) bool {
	for {
		switch x := e.(type) {
		case *ast.ParenExpr:
			e = x.X
			continue
		case *ast.Ident:
			return s.typed[x.Name]
		case *ast.CallExpr:
			return len(x.Args) == 1 && isGameOptionType(x.Fun, gameOptionTypeName)
		}
		return false
	}
}

// menuActions reports a declared name gameMenu<Option> for an option other
// than the first: the page's option actions are one block from
// gameMenuDayNight, never one name per option.
func (s *gameOptionScan) menuActions(v *ast.ValueSpec) {
	for _, n := range v.Names {
		option := gameOptionTypeName + strings.TrimPrefix(n.Name, "gameMenu")
		if strings.HasPrefix(n.Name, "gameMenu") && s.consts[option] && n.Name != gameMenuOptionBlock {
			s.report(n, "a menu action named for "+option)
		}
	}
}

func (s *gameOptionScan) visit(n ast.Node) bool {
	switch x := n.(type) {
	case *ast.SwitchStmt:
		if x.Tag != nil && s.typedValue(x.Tag) {
			s.report(x, "a switch over a GameOption value")
		}
	case *ast.CaseClause:
		for _, e := range x.List {
			if c := s.constName(e); c != "" {
				s.report(e, "a case names "+c)
			}
		}
	case *ast.BinaryExpr:
		switch x.Op {
		case token.EQL, token.NEQ, token.LSS, token.LEQ, token.GTR, token.GEQ:
			if c := s.constName(x.X) + s.constName(x.Y); c != "" {
				s.report(x, "a comparison with "+c)
			} else if _, lit := unwrapOption(x.Y).(*ast.BasicLit); lit && s.typedValue(x.X) {
				s.report(x, "a comparison of a GameOption value with a literal")
			} else if _, lit := unwrapOption(x.X).(*ast.BasicLit); lit && s.typedValue(x.Y) {
				s.report(x, "a comparison of a GameOption value with a literal")
			}
		case token.ADD, token.SUB:
			if c := s.constName(x.X) + s.constName(x.Y); c != "" {
				s.report(x, "arithmetic on "+c)
			}
		}
	case *ast.IndexExpr:
		if c := s.constName(x.Index); c != "" {
			s.report(x, "an index by "+c)
		}
	case *ast.CompositeLit:
		if isGameOptionType(x.Type, gameOptionValuesTyp) && len(x.Elts) != 0 {
			if _, keyed := x.Elts[0].(*ast.KeyValueExpr); !keyed {
				s.report(x, "a positional GameOptionValues literal")
				return true
			}
		}
		for _, e := range x.Elts {
			if kv, ok := e.(*ast.KeyValueExpr); ok {
				if c := s.constName(kv.Key); c != "" {
					s.report(e, "a composite literal keyed by "+c)
				}
				e = kv.Value
			}
			if c := s.constName(e); c != "" {
				s.report(e, "a composite literal lists "+c)
			}
		}
	}
	return true
}
