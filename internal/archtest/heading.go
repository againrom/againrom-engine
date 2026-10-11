package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The heading scan: outside pkg/sim/heading.go, production pkg/sim code
// converts no value to heading, calls neither headingOf nor facingToward, and
// calls requestFacing only in the turn core (facing.go) and the idle turn's
// random byte (escort.go).
const headingRuleFile = "pkg/sim/heading.go"

var headingRequestFiles = map[string]bool{"pkg/sim/facing.go": true, "pkg/sim/escort.go": true}

// CheckHeadingLaws reports every heading computed or requested outside its law.
func CheckHeadingLaws(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the heading scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/sim/") || name == headingRuleFile {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			fun := ""
			switch f := call.Fun.(type) {
			case *ast.Ident:
				fun = f.Name
			case *ast.SelectorExpr:
				fun = f.Sel.Name
			}
			switch {
			case fun == "heading":
				vs = append(vs, Violation{From: at(call), Reason: "a conversion to heading outside " + headingRuleFile + "; take the byte from a law there"})
			case fun == "headingOf" || fun == "facingToward":
				vs = append(vs, Violation{From: at(call), Reason: fun + " outside " + headingRuleFile + " is a second computation of a heading law"})
			case fun == "requestFacing" && !headingRequestFiles[name]:
				vs = append(vs, Violation{From: at(call), Reason: "requestFacing outside the turn core; turn through turnTo with a law's heading"})
			}
			return true
		})
	}
	return vs
}
