package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"slices"
	"sort"
	"strings"
)

// GroupWordsSetter is the one production file that writes a Group record's
// AI words (sim.SavedGroup.AI).
const GroupWordsSetter = "pkg/sim/groupwords.go"

// GroupWordsSeams are the files that copy or hand over a whole AI block: the
// World's byte form and the SAV reader and writer. Only a whole-block shape
// is allowed in them.
var GroupWordsSeams = []string{
	"pkg/sim/savedgroupsbinary.go",
	"pkg/formats/sav/actorgraph.go",
	"pkg/game/originalgroups.go",
	"pkg/game/savgroupfields.go",
	"pkg/game/citygroupcarry.go",
}

// GroupWordWrites names each write into an AI block outside the setter in
// files (module-relative slash path to source), as "path:line:col shape". A
// write is any store through an `.AI` selector: an index store, increment or
// compound assignment, a Put call, a copy or clear into a slice of it, the
// address of the block or a word, a slice of it handed on, an `AI:` field of
// a composite literal, or a store of the whole block. A copy source, a
// binary Uint read and an append source are reads. A whole-block shape
// in a seam file is allowed.
func GroupWordWrites(files map[string]string) ([]string, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		if name == GroupWordsSetter || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		if !strings.Contains(files[name], "AI") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		seam := slices.Contains(GroupWordsSeams, name)
		report := func(n ast.Node, shape string, whole bool) {
			if whole && seam {
				return
			}
			out = append(out, fmt.Sprintf("%s %s", fset.Position(n.Pos()), shape))
		}
		read := map[ast.Node]bool{}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, l := range n.Lhs {
					if aiIndexed(l) {
						report(l, "index store", false)
					} else if isAIBlock(l) {
						report(l, "whole-block store", true)
					}
				}
			case *ast.IncDecStmt:
				if aiIndexed(n.X) {
					report(n.X, "index increment", false)
				}
			case *ast.CallExpr:
				groupWordCall(n, read, report)
			case *ast.UnaryExpr:
				if n.Op == token.AND && aiIndexed(n.X) {
					report(n, "word address", false)
				} else if n.Op == token.AND && isAIBlock(n.X) {
					report(n, "block address", true)
				}
			case *ast.SliceExpr:
				if isAIBlock(n.X) && !read[n] {
					report(n, "slice handed on", wholeSlice(n))
				}
			case *ast.KeyValueExpr:
				if k, ok := n.Key.(*ast.Ident); ok && k.Name == "AI" {
					report(n, "AI literal field", false)
				}
			}
			return true
		})
	}
	return out, nil
}

// groupWordCall reports a call that writes into an AI block and marks the
// slices of one it reads.
func groupWordCall(call *ast.CallExpr, read map[ast.Node]bool, report func(ast.Node, string, bool)) {
	mark := func(e ast.Expr) {
		if s, ok := ast.Unparen(e).(*ast.SliceExpr); ok && isAIBlock(s.X) {
			read[s] = true
		}
	}
	switch fun := ast.Unparen(call.Fun).(type) {
	case *ast.Ident:
		switch {
		case (fun.Name == "copy" || fun.Name == "clear") && len(call.Args) > 0:
			if s, ok := ast.Unparen(call.Args[0]).(*ast.SliceExpr); ok && isAIBlock(s.X) {
				read[s] = true
				report(call, fun.Name+" into the block", wholeSlice(s))
			}
			if fun.Name == "copy" && len(call.Args) == 2 {
				mark(call.Args[1])
			}
		case fun.Name == "append" && call.Ellipsis.IsValid() && len(call.Args) == 2:
			mark(call.Args[1])
		}
	case *ast.SelectorExpr:
		switch {
		case strings.HasPrefix(fun.Sel.Name, "Put") && len(call.Args) > 0 && aiRooted(call.Args[0]):
			mark(call.Args[0])
			report(call, fun.Sel.Name+" into the block", false)
		case fun.Sel.Name == "Uint16" || fun.Sel.Name == "Uint32" || fun.Sel.Name == "Uint64":
			for _, a := range call.Args {
				mark(a)
			}
		}
	}
}

// isAIBlock reports an `x.AI` selector.
func isAIBlock(e ast.Expr) bool {
	s, ok := ast.Unparen(e).(*ast.SelectorExpr)
	return ok && s.Sel.Name == "AI"
}

// aiIndexed reports one word of an AI block, `x.AI[i]`.
func aiIndexed(e ast.Expr) bool {
	i, ok := ast.Unparen(e).(*ast.IndexExpr)
	return ok && isAIBlock(i.X)
}

// aiRooted reports a word or a slice of an AI block.
func aiRooted(e ast.Expr) bool {
	if s, ok := ast.Unparen(e).(*ast.SliceExpr); ok {
		return isAIBlock(s.X)
	}
	return aiIndexed(e)
}

// wholeSlice reports `x.AI[:]`.
func wholeSlice(s *ast.SliceExpr) bool {
	return s.Low == nil && s.High == nil && s.Max == nil
}
