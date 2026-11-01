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

// popupStoreHome is the one file that keeps where each movable editor popup
// stands and how big it is, and that drags, resizes and reopens it.
const popupStoreHome = "pkg/ui/mapeditorpopupstore.go"

// popupStoreFuncs are the functions the store must keep, so that a rename
// cannot leave the scan with nothing to exempt and nothing to find.
var popupStoreFuncs = []string{"placedCorner", "homeCorner", "promptCorner", "personCorner", "placeSpec", "dragPopup", "resizeFloating", "placePopup", "dropForeignSize"}

// popupStateNames are the fields a popup's place and size lived in, and the
// store row that holds them now.
var popupStateNames = map[string]bool{
	"places": true, "listAt": true, "objectAt": true, "listPlaced": true, "objPlaced": true,
	"promptShift": true, "promptAbs": true, "sizes": true, "minWide": true, "sizeKey": true,
	"slack": true, "listFit": true,
}

// popupCornerSources are the store methods whose result may become a popup
// spec's corner.
var popupCornerSources = map[string]bool{"placedCorner": true, "homeCorner": true, "promptCorner": true, "personCorner": true}

// popupSpecSize are the spec fields that hold a popup's size and slack.
var popupSpecSize = map[string]bool{"Size": true, "MinWidth": true, "Slack": true}

// popupFieldName is a field declared to keep a popup's place or size.
var popupFieldName = regexp.MustCompile(`^(list|object|prompt|popup|obj|window)[A-Za-z]*(At|Shift|Abs|Size|Sizes|Placed|Fit|Corner|Pos)$`)

var (
	popupCornerLine = regexp.MustCompile(`[Pp]opup(Corner|Shift|Size|Pos|At)\b|popup[A-Za-z]*(Shift|Corner)s? *(map|\[\])`)
	popupRectLine   = regexp.MustCompile(`\.(Rect|At|Pos) *= *image\.(Rect|Pt)\(`)
)

// PopupPlacementSites lists the places in production code of pkg/ui, outside
// the placement store, that build where an editor popup stands or how big it
// is: a write to a field that kept a popup's corner, size, shift, slack or fit
// (or to the store's rows), the address of the store, a popup spec's corner
// that does not come from the store, a field declared to keep a popup's place,
// and the two source patterns the audit census counts. Tests are not scanned.
func PopupPlacementSites(root string) ([]string, error) {
	var sites []string
	dir := filepath.Join(root, "pkg", "ui")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
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
		if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || rel == popupStoreHome {
			return nil
		}
		src, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		sites = append(sites, popupPlacementInSource(rel, src)...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(sites)
	return sites, nil
}

// PopupPlacementHomeHolds reports whether the store file still declares every
// function of popupStoreFuncs.
func PopupPlacementHomeHolds(root string) (bool, error) {
	src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(popupStoreHome)))
	if err != nil {
		return false, err
	}
	file, err := parser.ParseFile(token.NewFileSet(), popupStoreHome, src, 0)
	if err != nil {
		return false, err
	}
	found := map[string]bool{}
	for _, d := range file.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			found[f.Name.Name] = true
		}
	}
	for _, name := range popupStoreFuncs {
		if !found[name] {
			return false, nil
		}
	}
	return true, nil
}

// popupMentions reports whether expression x reads or indexes a selector named
// in popupStateNames.
func popupMentions(x ast.Expr) bool {
	found := false
	ast.Inspect(x, func(n ast.Node) bool {
		if s, ok := n.(*ast.SelectorExpr); ok && popupStateNames[s.Sel.Name] {
			found = true
		}
		return !found
	})
	return found
}

// popupCornerFromStore reports whether x is a call of a store method that
// answers a corner.
func popupCornerFromStore(x ast.Expr) bool {
	call, ok := x.(*ast.CallExpr)
	if !ok {
		return false
	}
	s, ok := call.Fun.(*ast.SelectorExpr)
	return ok && popupCornerSources[s.Sel.Name]
}

func popupPlacementInSource(name string, src []byte) []string {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, parser.ParseComments)
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var hits []string
	add := func(line int, what string) {
		site := name + ":" + strconv.Itoa(line) + " " + what
		if !seen[site] {
			seen[site] = true
			hits = append(hits, site)
		}
	}
	at := func(pos token.Pos) int { return fset.Position(pos).Line }
	editor := strings.HasPrefix(filepath.Base(name), "mapeditor") || filepath.Base(name) == "editorpopup.go"

	for i, line := range strings.Split(string(src), "\n") {
		if popupCornerLine.MatchString(line) {
			add(i+1, "names a popup corner, shift or size outside the placement store")
		}
		if editor && popupRectLine.MatchString(line) {
			add(i+1, "writes a popup rectangle as a literal outside the placement store")
		}
	}
	if !editor {
		return hits
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.AssignStmt:
			for i, lhs := range n.Lhs {
				if popupMentions(lhs) {
					add(at(n.Pos()), "writes a popup's place or size state outside the placement store")
				}
				if s, ok := lhs.(*ast.SelectorExpr); ok && popupSpecSize[s.Sel.Name] {
					add(at(n.Pos()), "sets a popup spec's size or slack outside the placement store")
				}
				if s, ok := lhs.(*ast.SelectorExpr); ok && s.Sel.Name == "At" {
					if len(n.Lhs) != len(n.Rhs) || !popupCornerFromStore(n.Rhs[i]) {
						add(at(n.Pos()), "sets a popup spec's corner from something but the placement store")
					}
				}
			}
		case *ast.IncDecStmt:
			if popupMentions(n.X) {
				add(at(n.Pos()), "writes a popup's place or size state outside the placement store")
			}
		case *ast.UnaryExpr:
			if n.Op == token.AND && popupMentions(n.X) {
				add(at(n.Pos()), "takes the address of a popup's place or size state outside the placement store")
			}
		case *ast.CompositeLit:
			if id, ok := n.Type.(*ast.Ident); ok && id.Name == "popupSpec" {
				for _, el := range n.Elts {
					if kv, ok := el.(*ast.KeyValueExpr); ok {
						if k, ok := kv.Key.(*ast.Ident); ok && k.Name == "At" && !popupCornerFromStore(kv.Value) {
							add(at(kv.Pos()), "sets a popup spec's corner from something but the placement store")
						} else if ok && popupSpecSize[k.Name] {
							add(at(kv.Pos()), "sets a popup spec's size or slack outside the placement store")
						}
					}
				}
			}
		case *ast.StructType:
			for _, f := range n.Fields.List {
				for _, nm := range f.Names {
					if popupFieldName.MatchString(nm.Name) || popupStateNames[nm.Name] && nm.Name != "places" {
						add(at(nm.Pos()), "declares field "+nm.Name+" to keep a popup's place or size outside the placement store")
					}
				}
			}
		}
		return true
	})
	return hits
}
