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

// CheckDestinationWriters scans files — pkg/sim's non-test sources, keyed by
// module-relative path; the live set comes from LoadSimSources, already home
// to CheckSimDeterminism — for every assignment whose left-hand side selects
// .HasTarget and whose right-hand side, AT THE SAME POSITION, is the literal
// true.
//
// The assignment may be single-valued (`e.HasTarget = true`) or one arm of a
// multi-value assignment (`e.TargetX, e.TargetY, e.HasTarget = x, y, true`),
// which is how three of pkg/sim's four writers are actually written (plan.md,
// "Note the shape of the assignment"). So the two sides of an *ast.AssignStmt
// are walked IN LOCKSTEP, index by index, never by asking "does .HasTarget
// appear on the left" and "does true appear on the right" as two independent
// questions — a check built that way would match every clearing assignment
// too (`e.TargetX, e.TargetY, e.HasTarget, e.Stall = 0, 0, false, 0` carries a
// true-shaped statement nowhere, but a same-length check that paired wrong
// could still be fooled by a longer tuple elsewhere). A multi-value
// assignment is only inspected when the two sides have equal length: a form
// like `a, b = f()` cannot be matched by position and is silently not
// reported, which is a known narrowing rather than an oversight — no writer
// in pkg/sim takes that shape today, and CheckSimDeterminism's own AST walk
// has the same "equal-length tuple" limit for the same reason.
//
// An assignment is attributed to the innermost top-level function whose body
// contains it. pkg/sim writes no writer inside a closure, but a func literal
// nested in a named function would still be attributed to that outer name —
// there is no better one to give it.
//
// This function carries no notion of an expected set: it reports what it
// finds, not whether the result is right.
//
// Findings come back in file order, then in source order within a file —
// matching CheckSimDeterminism's violations — so a report reads the same on
// every run regardless of map iteration order.
func CheckDestinationWriters(files map[string]string) []Finding {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)

	var out []Finding
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
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
