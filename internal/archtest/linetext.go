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

// The line-text scan: every install text file pkg/game reads is read by
// installText under a text source row, in pkg/game/installtext.go. Outside
// that file it refuses, in any pkg/game function:
//   - an entry source read (ReadFile, Entry, ReadEntry) whose argument names
//     text: a literal holding "text/", a constant whose value does, a name
//     spelling Text, or a name the function assigned from one of those;
//   - a decode of install text: Bytes or FileBytes on a TextCode, or a call of
//     DecodeInstallTextFile, SplitTextTable, secondGameTextSection or the vfs
//     code page converters;
//   - a #key section scan: a strings or bytes search for "#" + key;
//   - a trim or split of a reader's answer;
//   - a text source row, address or line text written as a literal.

const lineTextBuilderFile = "pkg/game/installtext.go"

var (
	lineTextDecoders  = map[string]bool{"DecodeInstallTextFile": true, "SplitTextTable": true, "secondGameTextSection": true, "WindowsCyrillicToDOS": true, "DOSCyrillicToWindows": true}
	lineTextReads     = map[string]bool{"ReadFile": true, "Entry": true, "ReadEntry": true}
	lineTextTypes     = map[string]bool{"lineText": true, "textAddress": true, "textSource": true}
	lineTextReaders   = map[string]bool{"installText": true, "ReadEventText": true, "ReadEventTextFor": true, "ReadShopTip": true, "readTipText": true, "MissionObjectives": true, "MissionObjectivesFor": true, "secondGameBriefing": true, "messageUTF8": true}
	lineTextCodeCalls = map[string]bool{"textCode": true, "InstallTextCode": true, "pageCode": true}
)

// CheckLineText reports every second way pkg/game reads install text. files
// is keyed by module-relative slash path, as LoadDrawnTextSources returns it.
func CheckLineText(files map[string]string) []Violation {
	var names []string
	for name := range files {
		if lineTextGameFile(name) {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return []Violation{{From: "pkg/game", Reason: "no sources scanned - the line-text scan found nothing to read"}}
	}
	sort.Strings(names)
	var vs []Violation
	parsed := map[string]*ast.File{}
	fset := token.NewFileSet()
	consts := map[string]ast.Expr{}
	for _, name := range names {
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		parsed[name] = f
		collectStringConsts(f, consts)
	}
	values := evalStringConsts(consts)
	for _, name := range names {
		f := parsed[name]
		if f == nil || name == lineTextBuilderFile {
			continue
		}
		at := func(n ast.Node) string { return fmt.Sprintf("%s:%d", name, fset.Position(n.Pos()).Line) }
		ast.Inspect(f, func(n ast.Node) bool {
			if lit, ok := n.(*ast.CompositeLit); ok {
				if id, ok := lit.Type.(*ast.Ident); ok && lineTextTypes[id.Name] {
					vs = append(vs, Violation{From: at(lit), Reason: "a " + id.Name + " literal outside " + lineTextBuilderFile + "; state the row there and read it with installText"})
				}
			}
			return true
		})
		check := func(fn string, recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt) {
			codes, texts, addrs := lineTextLocals(recv, typ, body, values)
			ast.Inspect(body, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				callee := calleeName(call)
				switch {
				case lineTextReads[callee] && !isPkgSelector(call, "os") && argsNameText(call, values, addrs):
					vs = append(vs, Violation{From: at(call), Reason: fn + " reads an install text address through " + callee + "; read it with installText under a text source row"})
				case lineTextDecoders[callee]:
					vs = append(vs, Violation{From: at(call), Reason: fn + " calls " + callee + " on install text; the codec and the sections belong to installText's rows"})
				case (callee == "Bytes" || callee == "FileBytes") && isTextCode(call, codes):
					vs = append(vs, Violation{From: at(call), Reason: fn + " decodes install text with TextCode." + callee + "; read it with installText under a text source row"})
				case isStringsCall(call) && searchesSection(call):
					vs = append(vs, Violation{From: at(call), Reason: fn + " scans a #key section; read it with a section accessor of installText"})
				case isStringsCall(call) && lineTextCuts[callee] && mentionsReader(call, texts):
					vs = append(vs, Violation{From: at(call), Reason: fn + " trims or splits a reader's install text; the row's form and edge do that"})
				}
				return true
			})
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.FuncDecl:
				if d.Body != nil {
					check(d.Name.Name, d.Recv, d.Type, d.Body)
				}
			case *ast.GenDecl:
				ast.Inspect(d, func(n ast.Node) bool {
					if lit, ok := n.(*ast.FuncLit); ok {
						check("a package-level function literal", nil, lit.Type, lit.Body)
						return false
					}
					return true
				})
			}
		}
	}
	return vs
}

// lineTextCuts are the strings and bytes functions that trim or split.
var lineTextCuts = map[string]bool{
	"Split": true, "SplitN": true, "SplitAfter": true, "SplitAfterN": true, "SplitSeq": true, "Lines": true,
	"Fields": true, "FieldsFunc": true, "Trim": true, "TrimSpace": true, "TrimLeft": true, "TrimRight": true,
	"TrimPrefix": true, "TrimSuffix": true, "TrimFunc": true, "Replace": true, "ReplaceAll": true,
	"Cut": true, "CutPrefix": true, "CutSuffix": true,
}

func lineTextGameFile(name string) bool {
	rest, ok := strings.CutPrefix(name, "pkg/game/")
	return ok && !strings.Contains(rest, "/") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")
}

// collectStringConsts records every package-level constant's expression.
func collectStringConsts(f *ast.File, out map[string]ast.Expr) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.CONST {
			continue
		}
		for _, spec := range gd.Specs {
			vs := spec.(*ast.ValueSpec)
			for i, id := range vs.Names {
				if i < len(vs.Values) {
					out[id.Name] = vs.Values[i]
				}
			}
		}
	}
}

// evalStringConsts is the string value of each constant written as literals
// and other constants joined by +.
func evalStringConsts(exprs map[string]ast.Expr) map[string]string {
	out := map[string]string{}
	var eval func(e ast.Expr, depth int) (string, bool)
	eval = func(e ast.Expr, depth int) (string, bool) {
		if depth > 16 {
			return "", false
		}
		switch e := e.(type) {
		case *ast.BasicLit:
			if e.Kind != token.STRING {
				return "", false
			}
			s, err := strconv.Unquote(e.Value)
			return s, err == nil
		case *ast.Ident:
			if v, ok := exprs[e.Name]; ok {
				return eval(v, depth+1)
			}
		case *ast.ParenExpr:
			return eval(e.X, depth+1)
		case *ast.BinaryExpr:
			if e.Op == token.ADD {
				a, ok := eval(e.X, depth+1)
				b, ok2 := eval(e.Y, depth+1)
				return a + b, ok && ok2
			}
		}
		return "", false
	}
	for name, e := range exprs {
		if v, ok := eval(e, 0); ok {
			out[name] = v
		}
	}
	return out
}

// lineTextLocals is the function's names that hold a TextCode (a parameter,
// result or variable of that type, or one assigned from a text code call),
// the names assigned from a reader, and the names assigned directly from an
// expression naming a text address.
func lineTextLocals(recv *ast.FieldList, typ *ast.FuncType, body *ast.BlockStmt, values map[string]string) (codes, texts, addrs map[string]bool) {
	codes, texts, addrs = map[string]bool{}, map[string]bool{}, map[string]bool{}
	fields := func(fl *ast.FieldList) {
		if fl == nil {
			return
		}
		for _, f := range fl.List {
			if isTextCodeType(f.Type) {
				for _, id := range f.Names {
					codes[id.Name] = true
				}
			}
		}
	}
	fields(recv)
	fields(typ.Params)
	fields(typ.Results)
	ast.Inspect(body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncLit:
			fields(n.Type.Params)
		case *ast.ValueSpec:
			for i, id := range n.Names {
				if isTextCodeType(n.Type) || (i < len(n.Values) && isTextCodeValue(n.Values[i])) {
					codes[id.Name] = true
				}
				if i < len(n.Values) && callsReader(n.Values[i]) {
					texts[id.Name] = true
				}
				if i < len(n.Values) && namesText(n.Values[i], values, nil) {
					addrs[id.Name] = true
				}
			}
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				id, ok := lhs.(*ast.Ident)
				if !ok {
					continue
				}
				var rhs ast.Expr
				if len(n.Rhs) == len(n.Lhs) {
					rhs = n.Rhs[i]
				} else if len(n.Rhs) == 1 {
					rhs = n.Rhs[0]
				}
				if rhs == nil {
					continue
				}
				if isTextCodeValue(rhs) {
					codes[id.Name] = true
				}
				if callsReader(rhs) {
					texts[id.Name] = true
				}
				if namesText(rhs, values, nil) {
					addrs[id.Name] = true
				}
			}
		}
		return true
	})
	return codes, texts, addrs
}

func isTextCodeType(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.Ident:
		return e.Name == "TextCode"
	case *ast.SelectorExpr:
		return e.Sel.Name == "TextCode"
	}
	return false
}

func isTextCodeValue(e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.CallExpr:
		return lineTextCodeCalls[calleeName(e)]
	case *ast.CompositeLit:
		return isTextCodeType(e.Type)
	}
	return false
}

// isTextCode reports whether call's receiver is a TextCode.
func isTextCode(call *ast.CallExpr, codes map[string]bool) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	switch x := sel.X.(type) {
	case *ast.Ident:
		return codes[x.Name]
	case *ast.CallExpr, *ast.CompositeLit:
		return isTextCodeValue(x)
	case *ast.ParenExpr:
		return isTextCodeValue(x.X)
	}
	return false
}

func isPkgSelector(call *ast.CallExpr, pkg string) bool {
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == pkg
}

// argsNameText reports whether an argument names an install text address, or
// is a name the function assigned from one.
func argsNameText(call *ast.CallExpr, values map[string]string, addrs map[string]bool) bool {
	for _, arg := range call.Args {
		if namesText(arg, values, addrs) {
			return true
		}
	}
	return false
}

// namesText reports whether e holds a literal with "text/", a constant whose
// value does, a name spelling Text, or a name in addrs.
func namesText(e ast.Expr, values map[string]string, addrs map[string]bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.SelectorExpr:
			// A pkg/namefold policy is a function name, not a name spelling
			// Text: ArchiveText folds the case of the address it is given.
			if id, ok := n.X.(*ast.Ident); ok && id.Name == "namefold" {
				return false
			}
		case *ast.BasicLit:
			if n.Kind == token.STRING && strings.Contains(strings.ToLower(n.Value), "text/") {
				found = true
			}
		case *ast.Ident:
			if addrs[n.Name] || strings.Contains(strings.ToLower(values[n.Name]), "text/") || strings.Contains(n.Name, "Text") || strings.HasPrefix(n.Name, "text") {
				found = true
			}
		}
		return !found
	})
	return found
}

func isStringsCall(call *ast.CallExpr) bool {
	return isPkgSelector(call, "strings") || isPkgSelector(call, "bytes")
}

// searchesSection reports whether a strings or bytes call searches for "#"
// joined to a key.
func searchesSection(call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if b, ok := arg.(*ast.BinaryExpr); ok && b.Op == token.ADD {
			if lit, ok := b.X.(*ast.BasicLit); ok && lit.Value == `"#"` {
				return true
			}
		}
	}
	return false
}

// mentionsReader reports whether a strings or bytes call's argument holds a
// reader's answer: a reader call, or a name assigned from one.
func mentionsReader(call *ast.CallExpr, texts map[string]bool) bool {
	found := false
	for _, arg := range call.Args {
		ast.Inspect(arg, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				if lineTextReaders[calleeName(n)] {
					found = true
				}
			case *ast.Ident:
				if texts[n.Name] {
					found = true
				}
			}
			return !found
		})
	}
	return found
}

func callsReader(e ast.Expr) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		if call, ok := n.(*ast.CallExpr); ok && lineTextReaders[calleeName(call)] {
			found = true
		}
		return !found
	})
	return found
}
