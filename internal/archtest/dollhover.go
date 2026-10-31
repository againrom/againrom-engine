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

// dollHoverHome is the one file that answers which slot of a paper doll lies
// under a pixel and how the tip of that slot is placed.
const dollHoverHome = "pkg/ui/dollhover.go"

// dollSlotFunc is the one function that answers a slot of a doll for every
// screen; dollInfoFunc the one that reads a slot's item text from it.
const (
	dollSlotFunc = "dollSlotAt"
	dollInfoFunc = "Lines"
)

// dollSlotInfoFields are the fields that hold a doll's item text per slot.
var dollSlotInfoFields = map[string]bool{
	"Info": true, "SlotInfo": true, "HeroSlotInfo": true, "CandidateSlotInfo": true,
}

// dollGeometryHomes are the functions allowed to call a doll geometry
// function: the builders that describe a doll from the data it is drawn with,
// and the drawing of the figure itself.
var dollGeometryHomes = map[string]map[string]bool{
	"wornSlotRects": {"wornSlotRects": true, "wornHover": true, "RenderWorn": true},
	"drawShopHover": {"DrawBeside": true},
	"characterPaneFigureRect": {
		"characterPaneFigureRect": true, "characterPaneDollHover": true,
		"dollHoverOver": true, "drawCharacterPaneBody": true,
	},
	"characterPaneFigureOrigin": {
		"characterPaneFigureOrigin": true, "characterPaneFigureRect": true,
		"characterPaneDollHover": true, "dollHoverOver": true, "drawCharacterPaneBody": true,
	},
}

// dollRangeName is what a collection of a doll's slot rectangles is called.
var dollRangeName = regexp.MustCompile(`(?i)slot|cell|worn|doll`)

var dollHitFuncName = regexp.MustCompile(`([dD]oll|[wW]orn)[A-Za-z]*(Slot|Area)At$`)

// DollHoverSites lists the places in production code of pkg/ui and pkg/game
// that answer a question of a paper doll beside the one description and its
// hit function: a function named for a doll's slot or area hit, a lookup in a
// slot map, a loop over slot rectangles that returns the rectangle's index,
// a read of a slot's item text by index, a call of the geometry the
// descriptions are built from outside their builders, a tip placed beside the
// pointer, and a tip key spelled outside a DollTip. Tests are not scanned.
func DollHoverSites(root string) ([]string, error) {
	var sites []string
	for _, top := range []string{"pkg/ui", "pkg/game"} {
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
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			sites = append(sites, dollHoverInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// DollHoverHomeHolds reports whether the one hit function and the one text
// read stand in their home file, so that a rename cannot leave the scan with
// nothing to exempt and nothing to find.
func DollHoverHomeHolds(root string) (bool, error) {
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(dollHoverHome)))
	if err != nil {
		return false, err
	}
	file, err := parser.ParseFile(token.NewFileSet(), dollHoverHome, src, 0)
	if err != nil {
		return false, err
	}
	found := map[string]bool{}
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			found[f.Name.Name] = true
		}
	}
	return found[dollSlotFunc] && found[dollInfoFunc] && found["DrawBeside"], nil
}

func dollHoverInSource(name string, src []byte) []string {
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
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		home := fn.Name.Name
		if dollHitFuncName.MatchString(home) && home != dollSlotFunc {
			add(fn.Pos(), home+" answers a doll's hit beside "+dollSlotFunc)
		}
		if fn.Body == nil {
			continue
		}
		writes := map[ast.Node]bool{}
		var stack []ast.Node
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			if n == nil {
				stack = stack[:len(stack)-1]
				return true
			}
			stack = append(stack, n)
			switch n := n.(type) {
			case *ast.AssignStmt:
				for _, l := range n.Lhs {
					writes[l] = true
				}
				if len(n.Lhs) == 2 && len(n.Rhs) == 1 && home != dollSlotFunc {
					if c, ok := n.Rhs[0].(*ast.CallExpr); ok && len(c.Args) == 2 {
						if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "At" {
							add(c.Pos(), "a slot map is read for a slot and whether it is marked")
						}
					}
				}
			case *ast.RangeStmt:
				if home != dollSlotFunc && rangeReturnsHitIndex(n) {
					add(n.Pos(), "a loop over rectangles returns the index of the one a point is in")
				}
			case *ast.IndexExpr:
				if sel, ok := n.X.(*ast.SelectorExpr); ok && dollSlotInfoFields[sel.Sel.Name] && !writes[n] && home != dollInfoFunc {
					add(n.Pos(), "a slot's item text is read by index")
				}
			case *ast.CallExpr:
				callee := calleeName(n)
				if homes, ok := dollGeometryHomes[callee]; ok && !homes[home] {
					add(n.Pos(), callee+" is called outside the doll builders")
				}
			case *ast.BasicLit:
				if n.Kind == token.STRING && strings.Contains(n.Value, "item/%") && !inDollTip(stack) {
					add(n.Pos(), "a tip key is spelled outside a DollTip")
				}
			}
			return true
		})
	}
	return hits
}

// rangeReturnsHitIndex reports the loop shape that walks rectangles for the
// one a point is in: for i, r := range cells { if p.In(r) { return i, ... } }.
func rangeReturnsHitIndex(r *ast.RangeStmt) bool {
	key, ok := r.Key.(*ast.Ident)
	val, ok2 := r.Value.(*ast.Ident)
	if !ok || !ok2 || r.Body == nil {
		return false
	}
	var ranged string
	switch x := r.X.(type) {
	case *ast.Ident:
		ranged = x.Name
	case *ast.SelectorExpr:
		ranged = x.Sel.Name
	case *ast.CallExpr:
		ranged = calleeName(x)
	}
	if !dollRangeName.MatchString(ranged) {
		return false
	}
	found := false
	ast.Inspect(r.Body, func(n ast.Node) bool {
		ifs, ok := n.(*ast.IfStmt)
		if !ok {
			return true
		}
		call, ok := ifs.Cond.(*ast.CallExpr)
		if !ok || calleeName(call) != "In" || len(call.Args) != 1 {
			return true
		}
		if a, ok := call.Args[0].(*ast.Ident); !ok || a.Name != val.Name {
			return true
		}
		for _, s := range ifs.Body.List {
			if ret, ok := s.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
				if id, ok := ret.Results[0].(*ast.Ident); ok && id.Name == key.Name {
					found = true
				}
			}
		}
		return true
	})
	return found
}

// inDollTip reports whether the innermost enclosing composite literal on the
// stack is a DollTip.
func inDollTip(stack []ast.Node) bool {
	for i := len(stack) - 1; i >= 0; i-- {
		lit, ok := stack[i].(*ast.CompositeLit)
		if !ok {
			continue
		}
		switch t := lit.Type.(type) {
		case *ast.Ident:
			return t.Name == "DollTip"
		case *ast.SelectorExpr:
			return t.Sel.Name == "DollTip"
		}
		return false
	}
	return false
}
