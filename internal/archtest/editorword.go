package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
)

func checkEditorWordClasses(files map[string]string) []Violation {
	var vs []Violation
	builders := 0
	for _, name := range sortedNames(files) {
		if !strings.HasPrefix(name, "pkg/ui/") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: err.Error()})
			continue
		}
		unicodeName := ""
		for _, imp := range f.Imports {
			if imp.Path.Value == `"unicode"` {
				unicodeName = "unicode"
				if imp.Name != nil {
					unicodeName = imp.Name.Name
				}
				if unicodeName == "." {
					vs = append(vs, Violation{From: name, Reason: "dot import hides word classification"})
				}
			}
		}
		for _, decl := range f.Decls {
			function := "package declaration"
			wordRunes := false
			classFunction := false
			if fn, ok := decl.(*ast.FuncDecl); ok {
				function = fn.Name.Name
				if name == "pkg/ui/editortextmodel.go" && function == "editorWordClass" && fn.Recv == nil {
					builders++
					continue
				}
				if fn.Type.Results != nil && len(fn.Type.Results.List) == 1 {
					typ := fn.Type.Results.List[0].Type
					classFunction = isIdent(typ, "bool") || isIdent(typ, "int") && strings.Contains(strings.ToLower(function), "word")
				}
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if id, ok := n.(*ast.Ident); ok && (id.Name == "rune" && classFunction || id.Name == "TextInputState") {
					wordRunes = true
				}
				return true
			})
			report := func(n ast.Node, reason string) {
				vs = append(vs, Violation{From: name + ":" + strconv.Itoa(fset.Position(n.Pos()).Line), Reason: function + " " + reason})
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.SelectorExpr:
					if unicodeName != "" && isIdent(x.X, unicodeName) &&
						(strings.HasPrefix(x.Sel.Name, "Is") && x.Sel.Name != "IsControl" ||
							x.Sel.Name == "In" || x.Sel.Name == "Letter" || x.Sel.Name == "Digit" ||
							x.Sel.Name == "Number" || x.Sel.Name == "Space" || x.Sel.Name == "White_Space" ||
							x.Sel.Name == "Punct" || x.Sel.Name == "Symbol" || x.Sel.Name == "Categories") {
						report(x, "classifies a word rune outside editorWordClass: "+x.Sel.Name)
					}
				case *ast.Ident:
					if x.Name == "editorWordClass" && function != "deleteWordEdge" {
						report(x, "uses editorWordClass outside the shared deleteWordEdge step")
					}
				case *ast.BinaryExpr:
					if wordRunes && (x.Op == token.EQL || x.Op == token.NEQ || x.Op == token.LSS || x.Op == token.LEQ || x.Op == token.GTR || x.Op == token.GEQ) &&
						(editorWordLiteral(x.X) || editorWordLiteral(x.Y)) {
						report(x, "compares a word rune outside editorWordClass")
					}
				case *ast.CaseClause:
					if wordRunes {
						for _, value := range x.List {
							if editorWordLiteral(value) {
								report(value, "switches on a word rune outside editorWordClass")
							}
						}
					}
				}
				return true
			})
		}
	}
	if builders != 1 {
		vs = append(vs, Violation{From: "pkg/ui/editortextmodel.go", Reason: "found " + strconv.Itoa(builders) + " editorWordClass builders, want one"})
	}
	return vs
}

func editorWordLiteral(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	if !ok || lit.Kind != token.CHAR {
		return false
	}
	value, err := strconv.Unquote(lit.Value)
	return err == nil && value != "\r" && value != "\n" && value != "\t"
}
