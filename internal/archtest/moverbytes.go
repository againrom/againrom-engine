package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strconv"
	"strings"
)

// The Mover byte scan: every byte of a Mover record (U154) the engine writes
// comes from the one producer, pkg/sim/moverbytes.go. Outside it, production
// code in pkg and cmd may not, into a Mover, write an index or a slice, call
// a binary Put, copy, clear, or build a composite literal holding a byte. A Mover is
// a field named Mover or Raw154, a [180]byte or [MoverLen]byte value or
// pointer, a variable named mover, a variable taken from a call naming
// "U154", or a Raw["U154"] index.

// MoverProducerFile is the one file that writes Mover bytes.
const MoverProducerFile = "pkg/sim/moverbytes.go"

// moverBytesAllowed names each writer the scan admits outside the producer,
// keyed file|function|shape, with its reason.
var moverBytesAllowed = map[string]string{
	"pkg/formats/sav/actorgraph.go|SetActorMoverRoute|copy": "the SAV format's setter carries a whole Mover into the file; it computes no byte",
	"pkg/formats/sav/actorgraph.go|currentActorGraph|copy":  "the SAV reader carries a whole Mover out of the file; it computes no byte",
}

// LoadMoverBytesSources reads every production .go file under pkg and cmd.
func LoadMoverBytesSources(moduleRoot string) (map[string]string, error) {
	all, err := LoadTreeSources(moduleRoot, "pkg", "cmd")
	if err != nil {
		return nil, err
	}
	for name := range all {
		if strings.HasSuffix(name, "_test.go") {
			delete(all, name)
		}
	}
	return all, nil
}

// CheckMoverBytes reports every Mover byte written outside the producer.
// files is keyed by module-relative slash path.
func CheckMoverBytes(files map[string]string) []Violation {
	if len(files) == 0 {
		return []Violation{{From: "pkg", Reason: "no sources scanned - the Mover byte scan found nothing to read"}}
	}
	names := make([]string, 0, len(files))
	for name := range files {
		if name != MoverProducerFile {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	var vs []Violation
	for _, name := range names {
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, files[name], parser.SkipObjectResolution)
		if err != nil {
			vs = append(vs, Violation{From: name, Reason: "source does not parse: " + err.Error()})
			continue
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok {
				s := moverScan{name: name, fset: fset, fn: "package"}
				ast.Inspect(d, s.visit(&vs))
				continue
			}
			if fn.Body == nil {
				continue
			}
			s := moverScan{name: name, fset: fset, fn: fn.Name.Name, movers: moverNames(fn)}
			ast.Inspect(fn.Body, s.visit(&vs))
		}
	}
	return vs
}

type moverScan struct {
	name, fn string
	fset     *token.FileSet
	movers   map[string]bool
}

func (s moverScan) visit(vs *[]Violation) func(ast.Node) bool {
	report := func(n ast.Node, shape, what string) {
		if _, ok := moverBytesAllowed[s.name+"|"+s.fn+"|"+shape]; ok {
			return
		}
		*vs = append(*vs, Violation{From: s.name + ":" + strconv.Itoa(s.fset.Position(n.Pos()).Line),
			Reason: s.fn + " " + what + "; Mover bytes come from " + MoverProducerFile})
	}
	return func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for _, l := range n.Lhs {
				if s.part(l) {
					report(n, "index", "writes an index of a Mover")
					break
				}
			}
		case *ast.IncDecStmt:
			if s.part(n.X) {
				report(n, "index", "writes an index of a Mover")
			}
		case *ast.CallExpr:
			if len(n.Args) == 0 {
				break
			}
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "copy" && (s.part(n.Args[0]) || s.mover(n.Args[0])) {
				report(n, "copy", "copies into a Mover")
			}
			if id, ok := n.Fun.(*ast.Ident); ok && id.Name == "clear" && (s.part(n.Args[0]) || s.mover(n.Args[0])) {
				report(n, "clear", "clears bytes of a Mover")
			}
			if sel, ok := n.Fun.(*ast.SelectorExpr); ok && strings.HasPrefix(sel.Sel.Name, "Put") && isBinaryOrder(sel.X) && (s.part(n.Args[0]) || s.mover(n.Args[0])) {
				report(n, "put", "puts a binary value into a Mover")
			}
		case *ast.CompositeLit:
			if isMoverArray(n.Type) && len(n.Elts) > 0 {
				report(n, "literal", "builds a Mover literal holding a byte")
			}
			for _, e := range n.Elts {
				kv, ok := e.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, ok := kv.Key.(*ast.Ident)
				lit, isLit := kv.Value.(*ast.CompositeLit)
				if ok && isLit && (key.Name == "Mover" || key.Name == "Raw154") && len(lit.Elts) > 0 {
					report(n, "literal", "builds a Mover literal holding a byte")
				}
			}
		}
		return true
	}
}

// part reports an index or slice of a Mover.
func (s moverScan) part(e ast.Expr) bool {
	switch e := unparen(e).(type) {
	case *ast.IndexExpr:
		return s.mover(e.X)
	case *ast.SliceExpr:
		return s.mover(e.X)
	}
	return false
}

// mover reports an expression that names a Mover.
func (s moverScan) mover(e ast.Expr) bool {
	switch e := unparen(e).(type) {
	case *ast.StarExpr:
		return s.mover(e.X)
	case *ast.SelectorExpr:
		return e.Sel.Name == "Mover" || e.Sel.Name == "Raw154"
	case *ast.Ident:
		return s.movers[e.Name] || strings.EqualFold(e.Name, "mover")
	case *ast.IndexExpr:
		return isU154(e.Index)
	}
	return false
}

func isU154(e ast.Expr) bool {
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.STRING && lit.Value == `"U154"`
}

func isBinaryOrder(e ast.Expr) bool {
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "binary"
}

// isMoverArray reports the type [180]byte or [MoverLen]byte, or a pointer to it.
func isMoverArray(e ast.Expr) bool {
	if star, ok := e.(*ast.StarExpr); ok {
		e = star.X
	}
	a, ok := e.(*ast.ArrayType)
	if !ok || a.Len == nil {
		return false
	}
	switch l := a.Len.(type) {
	case *ast.BasicLit:
		return l.Value == "180"
	case *ast.Ident:
		return l.Name == "MoverLen"
	case *ast.SelectorExpr:
		return l.Sel.Name == "MoverLen"
	}
	return false
}

// moverNames are the names in fn that hold a Mover: parameters, results and
// variables of a Mover array type, and variables assigned from an expression
// naming "U154".
func moverNames(fn *ast.FuncDecl) map[string]bool {
	out := map[string]bool{}
	fields := func(l *ast.FieldList) {
		if l == nil {
			return
		}
		for _, f := range l.List {
			if isMoverArray(f.Type) {
				for _, n := range f.Names {
					out[n.Name] = true
				}
			}
		}
	}
	fields(fn.Type.Params)
	fields(fn.Type.Results)
	if fn.Recv != nil {
		fields(fn.Recv)
	}
	namesU154 := func(e ast.Expr) bool {
		found := false
		ast.Inspect(e, func(n ast.Node) bool {
			if b, ok := n.(*ast.BasicLit); ok && isU154(b) {
				found = true
			}
			return !found
		})
		return found
	}
	ast.Inspect(fn.Body, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.ValueSpec:
			if n.Type != nil && isMoverArray(n.Type) {
				for _, id := range n.Names {
					out[id.Name] = true
				}
			}
			for i, v := range n.Values {
				if i < len(n.Names) && namesU154(v) {
					out[n.Names[i].Name] = true
				}
			}
		case *ast.AssignStmt:
			if len(n.Rhs) == 1 && namesU154(n.Rhs[0]) {
				if id, ok := n.Lhs[0].(*ast.Ident); ok {
					out[id.Name] = true
				}
			}
		}
		return true
	})
	return out
}
