package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// installNameIdents are the constants that spell a file under an install root.
// Joining one onto a root builds the exact-case path a case-sensitive host
// refuses; such a path is built by pkg/installpath instead.
var installNameIdents = map[string]bool{
	"MainArchive": true, "GraphicsArchive": true, "ScenarioArchive": true,
	"WorldArchive": true, "MoviesArchive": true, "SfxArchive": true,
	"patchArchive": true, "WindowIconExecutable": true, "graphicsArchive": true,
}

// installNameExempt is the resolver itself.
const installNameExempt = "pkg/installpath/"

// CheckInstallNames scans production sources, keyed by module-relative slash
// path, for a filepath.Join that builds a path under an install root by hand:
// an argument naming an install file constant, a ".res" string literal, or an
// install root selected as a Root field. Findings are sorted.
func CheckInstallNames(files map[string]string) []string {
	var names []string
	for name := range files {
		if !strings.HasPrefix(name, installNameExempt) {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var out []string
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			out = append(out, fmt.Sprintf("%s: %v", name, err))
			continue
		}
		ast.Inspect(f, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel.Name != "Join" {
				return true
			}
			if pkg, ok := sel.X.(*ast.Ident); !ok || pkg.Name != "filepath" {
				return true
			}
			for _, arg := range call.Args {
				if why := installNameArg(arg); why != "" {
					out = append(out, fmt.Sprintf("%s: filepath.Join %s; build the path through pkg/installpath", fset.Position(call.Pos()), why))
					break
				}
			}
			return true
		})
	}
	return out
}

func installNameArg(arg ast.Expr) string {
	switch a := arg.(type) {
	case *ast.Ident:
		if installNameIdents[a.Name] {
			return "joins the install file name " + a.Name
		}
	case *ast.SelectorExpr:
		if installNameIdents[a.Sel.Name] {
			return "joins the install file name " + a.Sel.Name
		}
		if a.Sel.Name == "Root" {
			return "joins onto an install Root"
		}
	case *ast.BasicLit:
		if a.Kind == token.STRING {
			if v, err := strconv.Unquote(a.Value); err == nil && strings.HasSuffix(strings.ToLower(v), ".res") {
				return "joins the archive name " + a.Value
			}
		}
	}
	return ""
}
