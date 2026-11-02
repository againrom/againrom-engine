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

// The install text address scan: every install text file with a numbered
// address is a textSource row of pkg/game/installtext.go, and a mission's
// text files are read by the one page resolver of pkg/mod. A numbered address
// formed anywhere else is a second table of addresses, in any of these shapes:
//   - a format string that names a text/ path and a verb (Sprintf, Sscanf,
//     or a constant that holds the pattern);
//   - a concatenation of a text/ path with a formatted number (strconv.Itoa,
//     FormatInt, Sprint);
//
// and a call of ReadTexts outside the resolver's file reads a mission's pages
// a second way.
const (
	textAddressRowsFile = "pkg/game/installtext.go"
	textResolverFile    = "pkg/mod/pageresolve.go"
)

var textVerb = regexp.MustCompile(`%[0-9]*[dsvq]`)

var textNumberCalls = map[string]bool{"Itoa": true, "FormatInt": true, "FormatUint": true, "Sprint": true, "Sprintf": true}

// CheckTextAddress reports every numbered install text address formed outside
// installtext.go and every ReadTexts call outside the resolver. files is keyed
// by module-relative slash path, as LoadDrawnTextSources returns it.
func CheckTextAddress(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the text address scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
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
		at := func(n ast.Node, reason string) {
			vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line), Reason: reason})
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.BasicLit:
				if s, ok := textPathLiteral(x); ok && name != textAddressRowsFile && textVerb.MatchString(s) {
					at(x, "formats an install text address ("+strconv.Quote(s)+"); numbered addresses are textSource rows in "+textAddressRowsFile)
				}
			case *ast.BinaryExpr:
				if name != textAddressRowsFile && x.Op == token.ADD && textConcatNumbered(x) {
					at(x, "concatenates an install text address with a number; numbered addresses are textSource rows in "+textAddressRowsFile)
					return false
				}
			case *ast.CallExpr:
				if schoolCallName(x.Fun) == "ReadTexts" && name != textResolverFile {
					at(x, "calls ReadTexts; a mission's pages are read by the resolver in "+textResolverFile)
				}
			}
			return true
		})
	}
	return vs
}

// textPathLiteral is a string literal that names a text/ path.
func textPathLiteral(l *ast.BasicLit) (string, bool) {
	if l.Kind != token.STRING {
		return "", false
	}
	s, err := strconv.Unquote(l.Value)
	return s, err == nil && strings.Contains(s, "text/")
}

// textConcatNumbered reports whether a chain of + holds a text/ path literal
// and a formatted number.
func textConcatNumbered(e *ast.BinaryExpr) bool {
	var path, number bool
	var walk func(ast.Expr)
	walk = func(e ast.Expr) {
		switch x := e.(type) {
		case *ast.BinaryExpr:
			if x.Op == token.ADD {
				walk(x.X)
				walk(x.Y)
			}
		case *ast.BasicLit:
			if _, ok := textPathLiteral(x); ok {
				path = true
			}
		case *ast.ParenExpr:
			walk(x.X)
		case *ast.CallExpr:
			if textNumberCalls[schoolCallName(x.Fun)] {
				number = true
			}
		}
	}
	walk(e)
	return path && number
}
