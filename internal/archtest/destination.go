package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
)

// Finding names one assignment that gives an entity's destination flag —
// Entity.HasTarget — the value true. A Finding is not itself a violation
// — this type carries no notion of what is expected, only what was found
// — so whether a given set of Findings is right is the caller's question,
// checked against the pinned expected set where this package's tests do so.
type Finding struct {
	File string // module-relative source path, e.g. "pkg/sim/combat.go"
	Line int    // 1-based line of the assignment's left-hand side
	Func string // enclosing named function ("" if none contains it)
}

func (f Finding) String() string {
	return f.File + ":" + strconv.Itoa(f.Line) + " in " + f.Func
}

// CheckDestinationWriters collects literal-true HasTarget assignments, pairing
// equal-length tuples by position. Unequal tuples remain outside the scan.
// A lexical receiver declared as another struct's own HasTarget field is a
// record, not Entity's destination. Unknown receiver types remain findings.
// Results follow file and source order, attributed to the enclosing named
// function even inside a closure. The caller owns the expected writer set.
func CheckDestinationWriters(files map[string]string) []Finding {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	parsed := make(map[string]*ast.File, len(files))
	positions := make(map[string]*token.FileSet, len(files))
	records := make(map[string]bool)
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], 0)
		if err != nil {
			// A file this scan cannot read is a file it can say nothing
			// about. Unlike CheckSimDeterminism this is a collector rather
			// than a pass/fail gate on the file's own syntax, so it is
			// skipped rather than turned into a synthetic finding; the live
			// tree's sources already parse (LoadSimSources reads what
			// go build itself compiles), so this path is not exercised by
			// TestDestinationWritersMatchFR2.
			continue
		}
		parsed[name], positions[name] = f, fset
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.TYPE {
				continue
			}
			for _, spec := range gen.Specs {
				typeSpec, ok := spec.(*ast.TypeSpec)
				if !ok || typeSpec.Name.Name == "Entity" {
					continue
				}
				structure, ok := typeSpec.Type.(*ast.StructType)
				if !ok {
					continue
				}
				for _, field := range structure.Fields.List {
					for _, id := range field.Names {
						if id.Name == "HasTarget" {
							records[typeSpec.Name.Name] = true
						}
					}
				}
			}
		}
	}
	var out []Finding
	for _, name := range names {
		f, fset := parsed[name], positions[name]
		if f == nil {
			continue
		}

		// Map each top-level function's byte range to its name, so an
		// assignment is attributed to the function whose source physically
		// contains it, without needing a push/pop stack through ast.Inspect.
		type span struct {
			name       string
			start, end token.Pos
		}
		var spans []span
		for _, decl := range f.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			spans = append(spans, span{fd.Name.Name, fd.Pos(), fd.End()})
		}
		enclosing := func(p token.Pos) string {
			for _, s := range spans {
				if p >= s.start && p < s.end {
					return s.name
				}
			}
			return ""
		}

		ast.Inspect(f, func(n ast.Node) bool {
			asn, ok := n.(*ast.AssignStmt)
			if !ok || asn.Tok != token.ASSIGN || len(asn.Lhs) != len(asn.Rhs) {
				return true
			}
			for i, lhs := range asn.Lhs {
				sel, ok := lhs.(*ast.SelectorExpr)
				if !ok || sel.Sel.Name != "HasTarget" {
					continue
				}
				id, ok := asn.Rhs[i].(*ast.Ident)
				if !ok || id.Name != "true" {
					continue
				}
				if destinationRecordType(sel.X, make(map[*ast.Object]bool), records) {
					continue
				}
				out = append(out, Finding{
					File: name,
					Line: fset.Position(lhs.Pos()).Line,
					Func: enclosing(lhs.Pos()),
				})
			}
			return true
		})
	}
	return out
}

func destinationRecordType(expr ast.Expr, seen map[*ast.Object]bool, records map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.ParenExpr:
		return destinationRecordType(e.X, seen, records)
	case *ast.UnaryExpr:
		if e.Op == token.AND || e.Op == token.MUL {
			return destinationRecordType(e.X, seen, records)
		}
	case *ast.StarExpr:
		return destinationRecordType(e.X, seen, records)
	case *ast.CompositeLit:
		return destinationRecordName(e.Type, records)
	case *ast.Ident:
		if e.Obj == nil || seen[e.Obj] {
			return false
		}
		seen[e.Obj] = true
		switch decl := e.Obj.Decl.(type) {
		case *ast.Field:
			return destinationRecordName(decl.Type, records)
		case *ast.ValueSpec:
			if decl.Type != nil {
				return destinationRecordName(decl.Type, records)
			}
			for i, id := range decl.Names {
				if id.Obj == e.Obj && len(decl.Names) == len(decl.Values) {
					return destinationRecordType(decl.Values[i], seen, records)
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range decl.Lhs {
				if id, ok := lhs.(*ast.Ident); ok && id.Obj == e.Obj && len(decl.Lhs) == len(decl.Rhs) {
					return destinationRecordType(decl.Rhs[i], seen, records)
				}
			}
		}
	}
	return false
}

func destinationRecordName(expr ast.Expr, records map[string]bool) bool {
	switch e := expr.(type) {
	case *ast.StarExpr:
		return destinationRecordName(e.X, records)
	case *ast.ParenExpr:
		return destinationRecordName(e.X, records)
	case *ast.Ident:
		if e.Obj != nil && e.Obj.Kind == ast.Typ {
			spec, ok := e.Obj.Decl.(*ast.TypeSpec)
			if !ok || spec.Name.Name == "Entity" {
				return false
			}
			structure, ok := spec.Type.(*ast.StructType)
			if !ok {
				return false
			}
			for _, field := range structure.Fields.List {
				for _, id := range field.Names {
					if id.Name == "HasTarget" {
						return true
					}
				}
			}
			return false
		}
		return records[e.Name]
	}
	return false
}
