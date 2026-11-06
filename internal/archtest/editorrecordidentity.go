package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// CheckEditorRecordIdentity keeps UI record tracking on the host's Key.
func CheckEditorRecordIdentity(files map[string]string) []Violation {
	var names []string
	for name := range files {
		if strings.HasPrefix(name, "pkg/ui/") && !strings.HasSuffix(name, "_test.go") {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return []Violation{{From: "pkg/ui", Reason: "no sources scanned for editor record identity"}}
	}
	var out []Violation
	for _, name := range names {
		fs := token.NewFileSet()
		file, err := parser.ParseFile(fs, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			out = append(out, Violation{From: name, Reason: err.Error()})
			continue
		}
		report := func(n ast.Node, reason string) {
			out = append(out, Violation{From: name + ":" + strconv.Itoa(fs.Position(n.Pos()).Line), Reason: reason + "; use the host's InspectionRecord.Key"})
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				switch n.Name {
				case "barRecordKeys", "carryBarRecordKeys", "barRecordVersion", "barRecordState", "followLoot":
					report(n, "retired editor identity predictor "+n.Name)
				}
			case *ast.MapType:
				key, _ := n.Key.(*ast.Ident)
				value, _ := n.Value.(*ast.Ident)
				if key != nil && key.Name == "RecordRef" && value != nil && (value.Name == "uint64" || value.Name == "InspectionRecord") {
					report(n, "UI identity table keyed by mutable record address")
				}
			case *ast.FuncDecl:
				checkEditorRecordFunction(n, fs, files[name], report)
			}
			return true
		})
	}
	return out
}

func checkEditorRecordFunction(fn *ast.FuncDecl, fs *token.FileSet, source string, report func(ast.Node, string)) {
	if fn.Body == nil {
		return
	}
	origins := map[string]bool{}
	shop := map[string]bool{}
	records := map[string]bool{}
	scalars := map[string]bool{}
	requests := map[string]bool{}
	if fn.Type.Params != nil {
		for _, field := range fn.Type.Params.List {
			t := field.Type
			if a, ok := t.(*ast.ArrayType); ok {
				t = a.Elt
			}
			if t, ok := t.(*ast.Ident); ok {
				for _, n := range field.Names {
					if t.Name == "InspectionRecord" {
						origins[n.Name] = true
						records[n.Name] = true
					}
					if t.Name == "ShopControl" {
						shop[n.Name] = true
					}
				}
			}
		}
	}
	hash, document := false, false
	isRecord := func(expr ast.Expr) bool {
		if name, ok := expr.(*ast.Ident); ok {
			return records[name.Name]
		}
		if at, ok := expr.(*ast.IndexExpr); ok {
			if source, ok := at.X.(*ast.SelectorExpr); ok {
				return source.Sel.Name == "Records"
			}
		}
		return false
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		if loop, ok := n.(*ast.RangeStmt); ok {
			if source, ok := loop.X.(*ast.SelectorExpr); ok && source.Sel.Name == "Records" {
				if value, ok := loop.Value.(*ast.Ident); ok {
					records[value.Name] = true
				}
			}
		}
		if assign, ok := n.(*ast.AssignStmt); ok {
			for i, rhs := range assign.Rhs {
				if i >= len(assign.Lhs) {
					break
				}
				name, ok := assign.Lhs[i].(*ast.Ident)
				if !ok {
					continue
				}
				if lit, ok := rhs.(*ast.CompositeLit); ok {
					if typ, ok := lit.Type.(*ast.Ident); ok && typ.Name == "InspectionNewRecord" {
						requests[name.Name] = true
					}
				}
				if at, ok := rhs.(*ast.IndexExpr); ok {
					if source, ok := at.X.(*ast.SelectorExpr); ok && source.Sel.Name == "Records" {
						records[name.Name] = true
					}
				}
				if field, ok := rhs.(*ast.SelectorExpr); ok {
					if isRecord(field.X) {
						scalars[name.Name] = true
					}
				}
			}
		}
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if sel.Sel.Name == "Records" {
				origins[source[fs.Position(sel.X.Pos()).Offset:fs.Position(sel.X.End()).Offset]] = true
			}
			document = document || sel.Sel.Name == "SourceBytes"
			if x, ok := sel.X.(*ast.Ident); ok && x.Name == "sha256" {
				hash = true
			}
		}
		return true
	})
	if hash && (document || len(origins) != 0) {
		report(fn, "document hash used to recognise record history")
	}
	index := func(e ast.Expr) bool {
		s, ok := e.(*ast.SelectorExpr)
		if !ok || s.Sel.Name != "Index" {
			return false
		}
		if x, ok := s.X.(*ast.Ident); ok && shop[x.Name] {
			return false
		}
		return true
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CallExpr:
			if len(origins) < 2 {
				break
			}
			var details int
			for _, arg := range n.Args {
				if field, ok := arg.(*ast.SelectorExpr); ok && field.Sel.Name == "Details" && isRecord(field.X) {
					details++
				}
			}
			if details > 1 {
				report(n, "cross-document record match by Details")
			}
		case *ast.IncDecStmt:
			if index(n.X) {
				report(n, "record index arithmetic")
			}
		case *ast.AssignStmt:
			for _, lhs := range n.Lhs {
				if field, ok := lhs.(*ast.SelectorExpr); ok && field.Sel.Name == "Key" && isRecord(field.X) {
					report(n, "UI writes record identity")
				}
				if !index(lhs) {
					continue
				}
				arithmetic := n.Tok != token.ASSIGN && n.Tok != token.DEFINE
				for _, rhs := range n.Rhs {
					ast.Inspect(rhs, func(v ast.Node) bool {
						_, binary := v.(*ast.BinaryExpr)
						arithmetic = arithmetic || binary
						return true
					})
				}
				if arithmetic {
					report(n, "record index arithmetic")
				}
			}
		case *ast.CompositeLit:
			if typ, ok := n.Type.(*ast.Ident); ok && typ.Name == "InspectionRecord" {
				for _, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Key" {
							report(kv, "UI builds record identity")
						}
					}
				}
			}
			if t, ok := n.Type.(*ast.Ident); ok && t.Name == "RecordRef" {
				for _, elt := range n.Elts {
					if kv, ok := elt.(*ast.KeyValueExpr); ok {
						if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "Index" {
							if _, ok := kv.Value.(*ast.BinaryExpr); ok {
								report(kv, "record index arithmetic in a reference")
							}
						}
					}
				}
			}
		case *ast.BinaryExpr:
			if len(origins) < 2 || n.Op != token.EQL && n.Op != token.NEQ && n.Op != token.LSS && n.Op != token.LEQ && n.Op != token.GEQ && n.Op != token.GTR {
				break
			}
			field, ok := n.X.(*ast.SelectorExpr)
			if !ok {
				break
			}
			r, ok := field.X.(*ast.Ident)
			if !ok || !records[r.Name] || requests[r.Name] {
				break
			}
			switch field.Sel.Name {
			case "Kind", "Index", "Label", "Cell", "ScriptID":
				if other, ok := n.Y.(*ast.SelectorExpr); ok && other.Sel.Name == field.Sel.Name {
					if r, ok := other.X.(*ast.Ident); ok && records[r.Name] && !requests[r.Name] {
						report(n, "cross-document record match by "+field.Sel.Name)
					}
				} else if other, ok := n.Y.(*ast.Ident); ok && scalars[other.Name] {
					report(n, "cross-document record match by remembered "+field.Sel.Name)
				} else if _, ok := n.Y.(*ast.IndexExpr); ok && field.Sel.Name == "Index" {
					report(n, "cross-document record match by index boundary")
				}
			}
		}
		return true
	})
}
