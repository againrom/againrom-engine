package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strings"
)

// The push button scan: every button rectangle in pkg/ui is painted by
// drawPushButton (or drawPlaque) from a pushButton value, in
// pkg/ui/widgetbutton.go. Elsewhere in pkg/ui it refuses a call of
// ebitenutil.DrawRect, and a function body that fills a rectangle and edges
// it with a two-pixel border: five rectangle fills (box, draw.Draw from an
// image.Uniform, vector.DrawFilledRect, ebitenutil.DrawRect), four of them
// carrying the literal 2. A closure counts for its function. A border drawn
// by a loop is not seen; CheckWidgetKit's painter and colour rules are the
// other half.

const (
	pushButtonKitFile    = "pkg/ui/widgetbutton.go"
	ebitenutilImportPath = "github.com/hajimehoshi/ebiten/v2/ebitenutil"
	vectorImportPath     = "github.com/hajimehoshi/ebiten/v2/vector"
)

// CheckPushButton reports every ebitenutil.DrawRect call in pkg/ui and every
// function outside the kit file that fills a rectangle and edges it with a
// two-pixel border. files is keyed by module-relative slash path, as
// LoadDrawnTextSources returns it.
func CheckPushButton(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned - the push button scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		if !strings.HasPrefix(name, "pkg/ui/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		at := func(n ast.Node) string { return fmt.Sprintf("%s:%d", name, fset.Position(n.Pos()).Line) }
		pkgs := importedPackages(f)
		ast.Inspect(f, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok && isPkgCall(call, pkgs, "ebitenutil", "DrawRect") {
				vs = append(vs, Violation{From: at(call), Reason: "ebitenutil.DrawRect paints a rectangle outside the push button painter; draw the button with drawPushButton"})
			}
			return true
		})
		if name == pushButtonKitFile {
			continue
		}
		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			fills, edges := 0, 0
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isRectFill(call, pkgs) {
					return true
				}
				fills++
				if mentionsLiteralTwo(call) {
					edges++
				}
				return true
			})
			if fills >= 5 && edges >= 4 {
				vs = append(vs, Violation{From: at(fn), Reason: fn.Name.Name + " fills a rectangle and draws a two-pixel border on it (" +
					fmt.Sprintf("%d rectangle fills, %d with a 2", fills, edges) + "); draw the button with drawPushButton from a pushButton style"})
			}
		}
	}
	return vs
}

// importedPackages maps each local package name a file uses to the last
// element of its import path.
func importedPackages(f *ast.File) map[string]string {
	out := map[string]string{}
	for _, imp := range f.Imports {
		p := strings.Trim(imp.Path.Value, `"`)
		local := path.Base(p)
		if imp.Name != nil {
			local = imp.Name.Name
		}
		out[local] = path.Base(p)
	}
	return out
}

// isPkgCall reports whether call is pkg.fn with pkg the imported package
// whose path ends in pkg. A file with no import of the name is read by the
// identifier itself.
func isPkgCall(call *ast.CallExpr, pkgs map[string]string, pkg, fn string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || sel.Sel.Name != fn {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	if !ok {
		return false
	}
	if base, imported := pkgs[id.Name]; imported {
		return base == pkg
	}
	return id.Name == pkg
}

// isRectFill reports whether call fills a rectangle with one colour.
func isRectFill(call *ast.CallExpr, pkgs map[string]string) bool {
	if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "box" {
		return true
	}
	if isPkgCall(call, pkgs, "ebitenutil", "DrawRect") || isPkgCall(call, pkgs, "vector", "DrawFilledRect") {
		return true
	}
	if isPkgCall(call, pkgs, "draw", "Draw") && len(call.Args) >= 3 {
		switch src := call.Args[2].(type) {
		case *ast.UnaryExpr:
			if lit, ok := src.X.(*ast.CompositeLit); ok {
				return isUniformType(lit.Type)
			}
		case *ast.CallExpr:
			return isUniformCall(src)
		}
	}
	return false
}

func isUniformType(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Uniform"
}

func isUniformCall(call *ast.CallExpr) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "NewUniform"
}

// mentionsLiteralTwo reports whether the call's arguments hold the integer
// literal 2, the edge width of a two-pixel border.
func mentionsLiteralTwo(call *ast.CallExpr) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok && lit.Kind == token.INT && lit.Value == "2" {
				found = true
			}
			return !found
		})
	}
	return found
}
