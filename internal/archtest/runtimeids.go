package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// The runtime-id scan: every runtime id the engine gives, for a World or for a
// SAV city document, is the lowest clear bit of one sim.RuntimeIDSet, and the
// search for that bit lives in pkg/sim/runtimeids.go. Anywhere else under pkg
// it refuses a function named like an allocator (next, alloc, free or lowest
// before RuntimeID, or lowestClear), and, in a function that names a runtime
// id, a loop that walks a map or a bitset to the first free entry: a for whose
// condition indexes, or a for that tests an index in an if and counts up. A
// function counts as naming a runtime id by its own name, and a loop by an
// identifier of its own; a loop that searches for a free identity is no
// runtime-id walk.

const runtimeIDBuilderFile = "pkg/sim/runtimeids.go"

var runtimeIDAllocatorName = regexp.MustCompile(`(?i)^(next|alloc|free|lowest)\w*runtimeid|^lowestclear$|^lowestfree$`)

// CheckRuntimeIDAllocator reports every second runtime-id allocator.
// files is keyed by module-relative slash path.
func CheckRuntimeIDAllocator(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the runtime-id scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/") || name == runtimeIDBuilderFile || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(p token.Pos) string { return name + ":" + strconv.Itoa(fset.Position(p).Line) }
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				continue
			}
			if runtimeIDAllocatorName.MatchString(fn.Name.Name) {
				vs = append(vs, Violation{From: at(fn.Pos()), Reason: fn.Name.Name + " is a second runtime-id allocator; take the id from sim.RuntimeIDSet.LowestClear"})
			}
			if fn.Body == nil {
				continue
			}
			byName := strings.Contains(strings.ToLower(fn.Name.Name), "runtimeid")
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				if loop, ok := n.(*ast.ForStmt); ok && (byName || namesRuntimeID(loop)) && walksToFreeEntry(loop) {
					vs = append(vs, Violation{From: at(loop.Pos()), Reason: "a loop walks to the first free runtime id; take it from sim.RuntimeIDSet.LowestClear"})
				}
				return true
			})
		}
	}
	return vs
}

func namesRuntimeID(loop *ast.ForStmt) bool {
	found := false
	ast.Inspect(loop, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.Ident:
			found = found || strings.Contains(strings.ToLower(n.Name), "runtimeid")
		case *ast.SelectorExpr:
			found = found || strings.Contains(strings.ToLower(n.Sel.Name), "runtimeid")
		}
		return !found
	})
	return found
}

func containsIndex(e ast.Node) bool {
	found := false
	if e == nil {
		return false
	}
	ast.Inspect(e, func(n ast.Node) bool {
		if _, ok := n.(*ast.IndexExpr); ok {
			found = true
		}
		return !found
	})
	return found
}

func walksToFreeEntry(loop *ast.ForStmt) bool {
	if loop.Cond != nil && containsIndex(loop.Cond) {
		return true
	}
	counts, tests := false, false
	ast.Inspect(loop.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			return false
		case *ast.IncDecStmt:
			counts = counts || n.Tok == token.INC
		case *ast.IfStmt:
			tests = tests || containsIndex(n.Cond)
		}
		return true
	})
	return counts && tests
}
