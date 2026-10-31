package archtest

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path"
	"sort"
	"strconv"
	"strings"
)

const picturePackage = "pkg/rules"

var pictureConversions = map[string]bool{
	"int": true, "int8": true, "int16": true, "int32": true, "int64": true,
	"uint": true, "uint8": true, "uint16": true, "uint32": true, "uint64": true,
}

func pictureUnwrap(x ast.Expr) ast.Expr {
	for {
		switch e := x.(type) {
		case *ast.ParenExpr:
			x = e.X
		case *ast.CallExpr:
			id, ok := e.Fun.(*ast.Ident)
			if !ok || !pictureConversions[id.Name] || len(e.Args) != 1 {
				return x
			}
			x = e.Args[0]
		default:
			return x
		}
	}
}

func pictureNamed(x ast.Expr) bool {
	switch x := pictureUnwrap(x).(type) {
	case *ast.Ident:
		return x.Name == "Picture" || x.Name == "picture"
	case *ast.SelectorExpr:
		return x.Sel.Name == "Picture" || x.Sel.Name == "picture"
	}
	return false
}

// PictureConstantTests names each ==, != or case of a Picture/picture against
// an integer literal or package-level constant outside pkg/rules.
func PictureConstantTests(files map[string]string) ([]string, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	fset := token.NewFileSet()
	parsed := map[string]*ast.File{}
	dirConsts := map[string]map[string]bool{}
	allConsts := map[string]bool{}
	for _, name := range names {
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			return nil, err
		}
		parsed[name] = f
		dir := path.Dir(name)
		if dirConsts[dir] == nil {
			dirConsts[dir] = map[string]bool{}
		}
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || g.Tok != token.CONST {
				continue
			}
			for _, s := range g.Specs {
				for _, id := range s.(*ast.ValueSpec).Names {
					dirConsts[dir][id.Name] = true
					allConsts[id.Name] = true
				}
			}
		}
	}
	var out []string
	for _, name := range names {
		if strings.HasPrefix(name, picturePackage+"/") {
			continue
		}
		f := parsed[name]
		local := dirConsts[path.Dir(name)]
		imports := map[string]bool{}
		for _, im := range f.Imports {
			p, _ := strconv.Unquote(im.Path.Value)
			base := path.Base(p)
			if im.Name != nil {
				base = im.Name.Name
			}
			imports[base] = true
		}
		number := func(x ast.Expr) bool {
			switch x := pictureUnwrap(x).(type) {
			case *ast.BasicLit:
				return x.Kind == token.INT
			case *ast.UnaryExpr:
				lit, ok := x.X.(*ast.BasicLit)
				return ok && lit.Kind == token.INT && (x.Op == token.SUB || x.Op == token.ADD)
			case *ast.Ident:
				return local[x.Name]
			case *ast.SelectorExpr:
				pkg, ok := x.X.(*ast.Ident)
				return ok && imports[pkg.Name] && allConsts[x.Sel.Name]
			}
			return false
		}
		report := func(at token.Pos) {
			out = append(out, fmt.Sprintf("%s:%d", name, fset.Position(at).Line))
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.BinaryExpr:
				if (n.Op == token.EQL || n.Op == token.NEQ) &&
					(pictureNamed(n.X) && number(n.Y) || number(n.X) && pictureNamed(n.Y)) {
					report(n.Pos())
				}
			case *ast.SwitchStmt:
				if n.Tag == nil || !pictureNamed(n.Tag) {
					return true
				}
				for _, s := range n.Body.List {
					c := s.(*ast.CaseClause)
					for _, e := range c.List {
						if number(e) {
							report(e.Pos())
						}
					}
				}
			}
			return true
		})
	}
	return out, nil
}
