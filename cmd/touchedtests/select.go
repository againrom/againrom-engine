package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"againrom/internal/archtest"
)

// Span is a closed range of lines of the new version of a file.
type Span struct{ From, To int }

// Change is one changed file: its slash path and the touched lines of its new
// version.
type Change struct {
	Path  string
	Spans []Span
}

// Test is one selected test function.
type Test struct {
	Package string // import path
	Name    string
	Release bool // declared in a *_release_test.go file
}

// node is one top-level declaration, or one spec of a grouped declaration.
type node struct {
	pkg        string // import path
	dir        string // slash path of the directory
	file       string // slash path of the file
	test       bool   // a Test function of a _test.go file
	release    bool
	pkgwide    bool // init, TestMain or a blank var: reaches every test of its package
	name       string
	from, to   int // line range
	start, end int // byte range of the source text
	defs       []string
	refs       map[string]struct{}
}

// Explain, when set, receives one line per changed declaration with the number
// of symbols and tests it reaches alone, and for a wide reach the chain of
// references to the farthest test.
var Explain io.Writer

// useKey names the module symbol an object stands for, "" for any other. A
// method of a concrete type is that type's; a method called through an
// interface is every method of its name. A field is every field of its name in
// its package. Package-level names are those of the package.
func useKey(o types.Object) string {
	if o == nil || o.Pkg() == nil {
		return ""
	}
	pkg := ownPath(o.Pkg().Path())
	if pkg == "" {
		return ""
	}
	switch o := o.(type) {
	case *types.Func:
		o = o.Origin()
		sig, _ := o.Type().(*types.Signature)
		if sig != nil && sig.Recv() != nil {
			named := namedOf(sig.Recv().Type())
			if named == nil {
				return "m\x00" + o.Name()
			}
			if types.IsInterface(sig.Recv().Type()) {
				return "i\x00" + pkg + "." + named.Obj().Name() + "." + o.Name()
			}
			return pkg + "\x00" + named.Obj().Name() + "." + o.Name()
		}
		if o.Parent() == o.Pkg().Scope() {
			return pkg + "\x00" + o.Name()
		}
	case *types.Var:
		o = o.Origin()
		if o.IsField() {
			return pkg + "\x00." + o.Name()
		}
		if o.Parent() == o.Pkg().Scope() {
			return pkg + "\x00" + o.Name()
		}
	case *types.Const, *types.TypeName:
		if o.Parent() == o.Pkg().Scope() {
			return pkg + "\x00" + o.Name()
		}
	}
	return ""
}

// concreteMethod reports whether a key names a method of a concrete type.
func concreteMethod(k string) bool {
	return !strings.HasPrefix(k, "m\x00") && !strings.HasPrefix(k, "i\x00")
}

func namedOf(t types.Type) *types.Named {
	for {
		switch x := t.(type) {
		case *types.Pointer:
			t = x.Elem()
		case *types.Alias:
			t = types.Unalias(x)
		case *types.Named:
			return x
		default:
			return nil
		}
	}
}

// ownPath normalizes the path of a package of the module, "" for any other.
func ownPath(p string) string {
	p = strings.TrimSuffix(p, " [external test]")
	p = strings.TrimSuffix(p, "_test")
	if p == archtest.ModulePath || strings.HasPrefix(p, archtest.ModulePath+"/") {
		return p
	}
	return ""
}

// testdataOf splits a path under a testdata directory into the directory that
// holds it and the file's name, ok false for any other path.
func testdataOf(p string) (dir, name string, ok bool) {
	parts := strings.Split(p, "/")
	for i, part := range parts[:len(parts)-1] {
		if part == "testdata" {
			dir = path.Join(parts[:i]...)
			if dir == "" {
				dir = "."
			}
			return dir, parts[len(parts)-1], true
		}
	}
	return "", "", false
}

// isTestFunc reports a func TestXxx(t *testing.T) declaration.
func isTestFunc(d *ast.FuncDecl) bool {
	if !strings.HasPrefix(d.Name.Name, "Test") || d.Type.TypeParams != nil || d.Recv != nil {
		return false
	}
	if rest := strings.TrimPrefix(d.Name.Name, "Test"); rest != "" && rest[0] >= 'a' && rest[0] <= 'z' {
		return false
	}
	ps := d.Type.Params.List
	if len(ps) != 1 || len(ps[0].Names) > 1 {
		return false
	}
	star, ok := ps[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "T"
}

// nodesOf turns the declarations of one checked file into nodes.
func nodesOf(fset *token.FileSet, f *ast.File, rel, pkg string, info *types.Info, defKeys func(types.Object) []string) []*node {
	dir := path.Dir(rel)
	isTest := strings.HasSuffix(rel, "_test.go")
	release := strings.HasSuffix(rel, "_release_test.go")
	line := func(p token.Pos) int { return fset.Position(p).Line }
	offset := func(p token.Pos) int { return fset.Position(p).Offset }
	var nodes []*node
	add := func(n *node, root ast.Node, from, to token.Pos) {
		n.pkg, n.dir, n.file, n.release = pkg, dir, rel, release
		n.from, n.to = line(from), line(to)
		n.start, n.end = offset(from), offset(to)
		n.refs = map[string]struct{}{}
		ast.Inspect(root, func(x ast.Node) bool {
			if id, ok := x.(*ast.Ident); ok {
				if k := useKey(info.Uses[id]); k != "" {
					n.refs[k] = struct{}{}
				}
			}
			return true
		})
		nodes = append(nodes, n)
	}
	for _, d := range f.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			from := d.Pos()
			if d.Doc != nil {
				from = d.Doc.Pos()
			}
			n := &node{name: d.Name.Name}
			if d.Recv == nil && (d.Name.Name == "init" || d.Name.Name == "_" || (isTest && d.Name.Name == "TestMain")) {
				n.pkgwide = true
			} else {
				n.defs = defKeys(info.Defs[d.Name])
				n.test = isTest && isTestFunc(d)
			}
			add(n, d, from, d.End())
		case *ast.GenDecl:
			if d.Tok == token.IMPORT {
				continue
			}
			for _, spec := range d.Specs {
				from, to := spec.Pos(), spec.End()
				if len(d.Specs) == 1 {
					from, to = d.Pos(), d.End()
					if d.Doc != nil {
						from = d.Doc.Pos()
					}
				}
				n := &node{}
				switch s := spec.(type) {
				case *ast.TypeSpec:
					n.name = s.Name.Name
					n.defs = defKeys(info.Defs[s.Name])
					ast.Inspect(s.Type, func(x ast.Node) bool {
						switch x := x.(type) {
						case *ast.Field:
							for _, id := range x.Names {
								n.defs = append(n.defs, defKeys(info.Defs[id])...)
							}
						}
						return true
					})
				case *ast.ValueSpec:
					for _, id := range s.Names {
						n.name = id.Name
						if id.Name == "_" {
							n.pkgwide = true
							continue
						}
						n.defs = append(n.defs, defKeys(info.Defs[id])...)
					}
				}
				add(n, spec, from, to)
			}
		}
	}
	return nodes
}

// Select returns the tests of the module that reach a changed symbol. The
// changed symbols are the declarations whose lines a change touches, and the
// fields and interface methods of a type are the type's. A test
// reaches a symbol when a chain of references, resolved by go/types, leads
// from it to the symbol; a method called through an interface is a reference
// to every method of that name, so a test may be selected that cannot reach the
// change, and none that can is missed. A changed file under testdata selects
// the test declarations of its directory that name it. ignored lists the
// changed paths that are not in the type-checked packages and not testdata.
func Select(m *archtest.TypeCheckedModule, root string, changes []Change) (tests []Test, ignored []string, err error) {
	disp := analyse(m)
	defKeys := disp.defKeys
	var nodes []*node
	byFile := map[string][]int{}
	defs := map[string][]int{}
	byDir := map[string][]int{}
	var pkgs []string
	for p := range m.Packages {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	for _, p := range pkgs {
		cp := m.Packages[p]
		groups := []struct {
			files []*ast.File
			rels  []string
			info  *types.Info
		}{{cp.Files, cp.RelPaths, cp.Info}, {cp.XTestFiles, cp.XTestRelPaths, cp.XTestInfo}}
		for _, g := range groups {
			for i, f := range g.files {
				for _, n := range nodesOf(m.Fset, f, g.rels[i], p, g.info, defKeys) {
					id := len(nodes)
					nodes = append(nodes, n)
					byFile[n.file] = append(byFile[n.file], id)
					byDir[n.dir] = append(byDir[n.dir], id)
					for _, k := range n.defs {
						defs[k] = append(defs[k], id)
					}
				}
			}
		}
	}
	// users[i] are the nodes that refer to a symbol node i defines.
	users := make([][]int, len(nodes))
	for id, n := range nodes {
		for k := range n.refs {
			for _, d := range defs[k] {
				if d != id {
					users[d] = append(users[d], id)
				}
			}
		}
	}
	explain := func(root int, file string) {
		n := nodes[root]
		parent := map[int]int{root: -1}
		q := []int{root}
		last, count := -1, 0
		for i := 0; i < len(q); i++ {
			id := q[i]
			if nodes[id].test {
				count++
				last = id
			}
			for _, u := range users[id] {
				if _, ok := parent[u]; !ok {
					parent[u] = id
					q = append(q, u)
				}
			}
		}
		fmt.Fprintf(Explain, "%s %s: %d symbols, %d tests\n", file, n.name, len(parent), count)
		if count > 100 {
			for id := last; id >= 0; id = parent[id] {
				fmt.Fprintf(Explain, "  <- %s %s\n", nodes[id].file, nodes[id].name)
			}
		}
	}
	var work []int
	seen := make([]bool, len(nodes))
	seed := func(id int) {
		if !seen[id] {
			seen[id] = true
			work = append(work, id)
		}
	}
	for _, c := range changes {
		if dir, name, ok := testdataOf(c.Path); ok {
			for _, id := range byDir[dir] {
				n := nodes[id]
				if !strings.HasSuffix(n.file, "_test.go") {
					continue
				}
				src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(n.file)))
				if err != nil {
					return nil, nil, err
				}
				if n.end <= len(src) && strings.Contains(string(src[n.start:n.end]), name) {
					seed(id)
				}
			}
			continue
		}
		ns := byFile[c.Path]
		if len(ns) == 0 {
			ignored = append(ignored, c.Path)
			continue
		}
		for _, id := range ns {
			n := nodes[id]
			if !touches(c.Spans, n.from, n.to) {
				continue
			}
			if Explain != nil {
				explain(id, c.Path)
			}
			if n.pkgwide {
				for _, id := range byDir[n.dir] {
					if nodes[id].test {
						seed(id)
					}
				}
			}
			seed(id)
		}
	}
	for len(work) > 0 {
		id := work[len(work)-1]
		work = work[:len(work)-1]
		for _, u := range users[id] {
			seed(u)
		}
	}
	for id, n := range nodes {
		if seen[id] && n.test {
			tests = append(tests, Test{n.pkg, n.name, n.release})
		}
	}
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].Package != tests[j].Package {
			return tests[i].Package < tests[j].Package
		}
		return tests[i].Name < tests[j].Name
	})
	tests = compact(tests)
	sort.Strings(ignored)
	return tests, ignored, nil
}

func compact(ts []Test) []Test {
	out := ts[:0]
	for i, t := range ts {
		if i > 0 && t == ts[i-1] {
			continue
		}
		out = append(out, t)
	}
	return out
}

func touches(spans []Span, from, to int) bool {
	for _, s := range spans {
		if s.From <= to && s.To >= from {
			return true
		}
	}
	return false
}
