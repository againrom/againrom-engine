package archtest

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"sort"
)

// compositionRoot is the package and type this file measures: pkg/game's
// FrontEnd, which assembles a ready install and a game session.
const (
	compositionPackage = "pkg/game"
	compositionType    = "FrontEnd"
)

// coordinationThreshold is how many owned components one function must reach
// before it counts as a coordination point.
//
// THREE, not two. A function that reads one component sits where it belongs,
// and one that reads two is usually joining two halves of a single answer -
// the install's art and the session it is drawn for. A function that reaches
// three separate lifecycles is coordinating them.
const coordinationThreshold = 3

// CompositionReport is the measured shape of the composition root.
type CompositionReport struct {
	// Components is each owned component and how many fields it declares, in
	// FrontEnd's own declaration order.
	Components []ComponentFields
	// Fields is the total across every component: the number a reader means by
	// "how big is FrontEnd".
	Fields int
	// Coordination is every non-test function declaration in pkg/game whose
	// body reaches
	// coordinationThreshold or more owned components through a FrontEnd-typed
	// value, module-relative and sorted, each with the components it touched.
	// The unit is the FUNCTION, not the file: see LoadComposition.
	Coordination []FileComponents
}

// ComponentFields is one owned component and the number of fields it declares.
type ComponentFields struct {
	Name   string
	Fields int
}

// FileComponents is one coordinating function and the owned components its
// body reaches through a FrontEnd-typed value. The name is kept from the
// file-level ratchet this replaces; Func is what makes an entry unique
// within a file now that the unit is the function.
type FileComponents struct {
	File       string
	Func       string
	Components []string
}

// CompositionBaseline is the one committed record of both ratcheted numbers.
// Each may only fall; a fall requires updating this value in the same commit,
// and a rise is a regression the check reports.
type CompositionBaseline struct {
	Fields       int
	Coordinators int
}

// LoadComposition measures pkg/game's composition root: the field count is
// still read off the FrontEnd struct's own syntax, but the coordination walk
// is TYPE-AWARE, sharing the whole module's go/types pass with
// LoadCommandLiterals (loadTypeCheckedModule) rather than paying for a second
// one.
//
// The unit is the FUNCTION, not the file. A file union let three methods
// that each touch one component read as a single coordinator, which
// overstated concentration; per function, each is judged on what it alone
// reaches. A method or free function counts when a selector inside its body
// - anywhere in that body, including a nested closure - names a field this
// package assigns to one of the five owned components, AND go/types resolves
// the selector's own base expression to FrontEnd or *FrontEnd. That base can
// be a receiver, a parameter, a local alias at any depth (same := f, then
// local := same), or a chain of struct fields ending in one that holds a
// FrontEnd - go/types answers the same question for all of them, so the walk
// needs no separate case for "reached through a field" the way the syntactic
// version did.
//
// What it cannot see: a package-level func literal, a struct embedding
// *FrontEnd reached by promotion, and a reach spelled through a component
// itself (pc := &f.PersistenceContext, or f.PersistenceContext.x), which
// lowers the count with no structural change.
func LoadComposition(root string) (CompositionReport, error) {
	m, err := loadTypeCheckedModule(root)
	if err != nil {
		return CompositionReport{}, err
	}
	importPath := ModulePath + "/" + compositionPackage
	cp, ok := m.Packages[importPath]
	if !ok || cp.NumProd == 0 {
		return CompositionReport{}, fmt.Errorf("%s holds no non-test sources", compositionPackage)
	}

	structs := map[string]*ast.StructType{}
	for _, f := range cp.Files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts := spec.(*ast.TypeSpec)
				if st, ok := ts.Type.(*ast.StructType); ok {
					structs[ts.Name.Name] = st
				}
			}
		}
	}

	rootStruct, ok := structs[compositionType]
	if !ok {
		return CompositionReport{}, fmt.Errorf("%s declares no %s struct", compositionPackage, compositionType)
	}
	var report CompositionReport
	owner := map[string]string{}
	for _, fl := range rootStruct.Fields.List {
		if len(fl.Names) != 0 {
			return CompositionReport{}, fmt.Errorf("%s declares field %s directly; every field belongs to one owned component",
				compositionType, fl.Names[0].Name)
		}
		id, ok := fl.Type.(*ast.Ident)
		if !ok {
			return CompositionReport{}, fmt.Errorf("%s embeds a non-local type", compositionType)
		}
		comp, ok := structs[id.Name]
		if !ok {
			return CompositionReport{}, fmt.Errorf("embedded component %s is not a struct in this package", id.Name)
		}
		n := 0
		for _, cf := range comp.Fields.List {
			for _, name := range cf.Names {
				if prior, dup := owner[name.Name]; dup {
					return CompositionReport{}, fmt.Errorf("field %s is declared in both %s and %s", name.Name, prior, id.Name)
				}
				owner[name.Name] = id.Name
				n++
			}
		}
		report.Components = append(report.Components, ComponentFields{Name: id.Name, Fields: n})
		report.Fields += n
	}

	for i := 0; i < cp.NumProd; i++ {
		file := cp.Files[i]
		rel := cp.RelPaths[i]
		for _, d := range file.Decls {
			fd, ok := d.(*ast.FuncDecl)
			if !ok || fd.Body == nil {
				continue
			}
			touched := map[string]bool{}
			ast.Inspect(fd.Body, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				comp, ok := owner[sel.Sel.Name]
				if !ok {
					return true
				}
				if isFrontEndType(cp.Info.Types[sel.X].Type) {
					touched[comp] = true
				}
				return true
			})
			if len(touched) < coordinationThreshold {
				continue
			}
			names := make([]string, 0, len(touched))
			for c := range touched {
				names = append(names, c)
			}
			sort.Strings(names)
			report.Coordination = append(report.Coordination, FileComponents{File: rel, Func: funcLabel(fd), Components: names})
		}
	}
	sort.Slice(report.Coordination, func(i, j int) bool {
		a, b := report.Coordination[i], report.Coordination[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Func < b.Func
	})
	return report, nil
}

// isFrontEndType reports whether t, after unwrapping a pointer and any type
// alias, is the named type pkg/game.FrontEnd.
func isFrontEndType(t types.Type) bool {
	if t == nil {
		return false
	}
	t = types.Unalias(t)
	if ptr, ok := t.(*types.Pointer); ok {
		t = types.Unalias(ptr.Elem())
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj != nil && obj.Pkg() != nil && obj.Pkg().Path() == ModulePath+"/"+compositionPackage && obj.Name() == compositionType
}

// funcLabel names a FuncDecl the way a reader would: "(*FrontEnd).Method" for
// a method, or the bare name for a free function.
func funcLabel(fd *ast.FuncDecl) string {
	if fd.Recv != nil && len(fd.Recv.List) > 0 {
		recv := fd.Recv.List[0].Type
		star := ""
		if s, ok := recv.(*ast.StarExpr); ok {
			star = "*"
			recv = s.X
		}
		if id, ok := recv.(*ast.Ident); ok {
			return fmt.Sprintf("(%s%s).%s", star, id.Name, fd.Name.Name)
		}
	}
	return fd.Name.Name
}

// CheckComposition compares a measured report against the committed baseline
// and returns every violation. Both numbers ratchet downward only.
func CheckComposition(report CompositionReport, baseline CompositionBaseline) []string {
	var out []string
	out = append(out, ratchetDown("composition root fields", report.Fields, baseline.Fields,
		"internal/archtest/composition_baseline.go: Fields")...)
	out = append(out, ratchetDown("coordination points", len(report.Coordination), baseline.Coordinators,
		"internal/archtest/composition_baseline.go: Coordinators")...)
	return out
}

func ratchetDown(what string, got, want int, field string) []string {
	switch {
	case got == want:
		return nil
	case got > want:
		return []string{fmt.Sprintf("%s rose from %d to %d: regression, move the new state or the new reach rather than raising %s",
			what, want, got, field)}
	default:
		return []string{fmt.Sprintf("%s fell from %d to %d: set %s to %d in the same commit", what, want, got, field, got)}
	}
}
