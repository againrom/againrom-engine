package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// scrollListHome is the one file that holds the steps of every scrolling list:
// each list is a row of its scrollSpec table, and the wheel, key, page and bar
// steps are read there.
const scrollListHome = "pkg/ui/scrolllist.go"

// scrollListFuncs are the functions and methods of the model.
var scrollListFuncs = []string{"listKey", "listBarRequest", "editorWheelRows", "wheelStep", "keyStep", "barStep", "wheelAccumulated"}

// scrollListRoots are the directories whose production code is scanned.
var scrollListRoots = []string{"pkg/ui", "cmd/starter"}

// scrollWheelReaders are the functions allowed to read a wheel reading outside
// the model, with what they use it for: the two input samplers, the brush
// radius and the viewer's zoom, none of which moves a list.
var scrollWheelReaders = map[string]map[string]bool{
	"Wheel":        {"pkg/ui/app.go:readAppInput": true, "pkg/ui/viewer.go:readInput": true},
	"wheelNotches": {"pkg/ui/mapeditorbrush.go:brushWheel": true},
	"wheelSign":    {"pkg/ui/viewer.go:step": true},
}

// scrollFocusMovers are the functions whose Move with a literal moves a
// character-creation control focus, not a list.
var scrollFocusMovers = map[string]bool{
	"pkg/ui/app.go:stepChargen": true, "pkg/ui/app.go:stepChargenDetailed": true, "pkg/ui/app.go:stepPreCreate": true,
}

// scrollBarDispatchers are the functions allowed to dispatch a bar request
// outside the model: the help text, whose steps come from the model row.
var scrollBarDispatchers = map[string]bool{"pkg/ui/help.go:helpBarRequest": true}

// scrollStepCalls are the calls that move a list by a count.
var scrollStepCalls = map[string]bool{
	"Move": true, "MoveChoice": true, "Scroll": true, "scroll": true, "ScrollPack": true,
	"WorldMapMove": true, "scrollShop": true, "helpScrollBy": true,
}

var scrollStepConst = regexp.MustCompile(`(?i)wheel(rows|lines|step)`)

var scrollBarRequests = map[string]bool{"barLineUp": true, "barLineDown": true, "barPageUp": true, "barPageDown": true}

// ScrollListSites lists the places in production code of pkg/ui and
// cmd/starter that step a list beside the one list model: a read of a wheel
// field, a call of the sampler or of a wheel accessor outside its use, a
// function of the model declared elsewhere, a scrollSpec literal outside the
// model, a step count written as a number, a named wheel step declared
// elsewhere, and a bar request dispatched by a switch. Tests are not scanned.
func ScrollListSites(root string) ([]string, error) {
	var sites []string
	for _, top := range scrollListRoots {
		err := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(top)), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if d.Name() == "testdata" {
					return filepath.SkipDir
				}
				return nil
			}
			rel, rerr := filepath.Rel(root, path)
			if rerr != nil {
				return rerr
			}
			rel = filepath.ToSlash(rel)
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || rel == scrollListHome {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, scrollListInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// ScrollListHomeHolds reports whether the model file declares the model's
// functions and the spec type, so that a rename cannot leave the scan with
// nothing to exempt.
func ScrollListHomeHolds(root string) (bool, error) {
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(scrollListHome)))
	if err != nil {
		return false, err
	}
	file, err := parser.ParseFile(token.NewFileSet(), scrollListHome, src, 0)
	if err != nil {
		return false, err
	}
	found := map[string]bool{}
	for _, d := range file.Decls {
		switch d := d.(type) {
		case *ast.FuncDecl:
			found[d.Name.Name] = true
		case *ast.GenDecl:
			for _, s := range d.Specs {
				if ts, ok := s.(*ast.TypeSpec); ok {
					found[ts.Name.Name] = true
				}
			}
		}
	}
	for _, name := range scrollListFuncs {
		if !found[name] {
			return false, nil
		}
	}
	return found["scrollSpec"], nil
}

func scrollListInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var hits []string
	add := func(pos token.Pos, what string) {
		site := name + ":" + strconv.Itoa(fset.Position(pos).Line) + " " + what
		if !seen[site] {
			seen[site] = true
			hits = append(hits, site)
		}
	}
	inModel := map[string]bool{}
	for _, f := range scrollListFuncs {
		inModel[f] = true
	}
	for _, decl := range file.Decls {
		if gd, ok := decl.(*ast.GenDecl); ok && (gd.Tok == token.CONST || gd.Tok == token.VAR) {
			for _, s := range gd.Specs {
				if vs, ok := s.(*ast.ValueSpec); ok {
					for _, v := range vs.Values {
						ast.Inspect(v, func(n ast.Node) bool {
							if lit, ok := n.(*ast.CompositeLit); ok {
								if id, ok := lit.Type.(*ast.Ident); ok && id.Name == "scrollSpec" {
									add(lit.Pos(), "a scrollSpec row is written beside the list model")
								}
							}
							return true
						})
					}
					for _, id := range vs.Names {
						if scrollStepConst.MatchString(id.Name) {
							add(id.Pos(), id.Name+" is a wheel step declared beside the list model")
						}
					}
				}
			}
		}
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		home := name + ":" + fn.Name.Name
		if inModel[fn.Name.Name] && (fn.Recv == nil || scrollSpecReceiver(fn.Recv)) {
			add(fn.Pos(), fn.Name.Name+" is declared beside the list model")
		}
		if fn.Body == nil {
			continue
		}
		hasListWheel := false
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if c, ok := n.(*ast.CallExpr); ok && calleeName(c) == "ListWheelLines" {
				hasListWheel = true
			}
			return true
		})
		writes := map[ast.Node]bool{}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, l := range n.Lhs {
					writes[l] = true
				}
			case *ast.SelectorExpr:
				if n.Sel.Name == "WheelY" && !writes[n] {
					add(n.Pos(), "a wheel field is read beside the list model")
				}
			case *ast.CompositeLit:
				if id, ok := n.Type.(*ast.Ident); ok && id.Name == "scrollSpec" {
					add(n.Pos(), "a scrollSpec row is written beside the list model")
				}
			case *ast.CaseClause:
				if !scrollBarDispatchers[home] {
					for _, e := range n.List {
						if id, ok := e.(*ast.Ident); ok && scrollBarRequests[id.Name] {
							add(e.Pos(), "a bar request "+id.Name+" is dispatched beside the list model")
						}
					}
				}
			case *ast.CallExpr:
				callee := calleeName(n)
				if homes, ok := scrollWheelReaders[callee]; ok && !homes[home] && !(callee == "Wheel" && hasListWheel) {
					add(n.Pos(), callee+" is called outside the list model")
				}
				if scrollStepCalls[callee] && !scrollFocusMovers[home] {
					for _, a := range n.Args {
						if scrollLiteralStep(a) {
							add(a.Pos(), callee+" is given a step written as a number")
						}
					}
				}
			}
			return true
		})
	}
	return hits
}

// scrollSpecReceiver reports a method of the model's spec type.
func scrollSpecReceiver(r *ast.FieldList) bool {
	if len(r.List) != 1 {
		return false
	}
	t := r.List[0].Type
	if star, ok := t.(*ast.StarExpr); ok {
		t = star.X
	}
	id, ok := t.(*ast.Ident)
	return ok && id.Name == "scrollSpec"
}

// scrollLiteralStep reports a non-zero integer literal, signed or not.
func scrollLiteralStep(e ast.Expr) bool {
	if u, ok := e.(*ast.UnaryExpr); ok && (u.Op == token.SUB || u.Op == token.ADD) {
		e = u.X
	}
	lit, ok := e.(*ast.BasicLit)
	return ok && lit.Kind == token.INT && lit.Value != "0"
}
