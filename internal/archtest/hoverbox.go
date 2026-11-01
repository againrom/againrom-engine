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

// Every hover box is a description (lines, style, anchor) that one composer
// draws in one style table. The table file builds every hoverStyle,
// hoverBoxInk and hoverShape and holds the box colours; the composer file
// paints a box and keeps its size, its frame and its recording; a
// description is placed by tooltipPicture and drawn beside the pointer by
// DollTip.DrawBeside.
const (
	hoverStyleHome = "pkg/ui/hoverstyles.go"
	hoverBoxHome   = "pkg/ui/widgethover.go"
)

// hoverStyleTypes are the types only the style table builds.
var hoverStyleTypes = map[string]bool{"hoverStyle": true, "hoverBoxInk": true, "hoverShape": true}

// hoverComposerNames are the functions only the composer file calls.
var hoverComposerNames = map[string]bool{"composeHoverBox": true, "hoverBoxSize": true, "paintHoverBevel": true}

// hoverColourNames are the box colours only the table and composer files use.
var hoverColourNames = map[string]bool{"hoverFill": true, "hoverLight": true, "hoverDark": true, "hoverSharedBox": true}

// hoverPlacers are the functions that turn a description into a picture, and
// the files allowed to call them.
var hoverPlacers = map[string]map[string]bool{
	"placeHover": {"pkg/ui/tooltip.go": true},
	"drawHover":  {"pkg/ui/dollhover.go": true},
}

// hoverPainterName matches a function that paints a hover box or a tooltip;
// hoverNotAPainter names the matches that only present a finished box.
var hoverPainterName = regexp.MustCompile(`^(?i:draw|paint|compose|render)[A-Za-z]*(Hover|Tooltip)`)

var hoverNotAPainter = map[string]bool{
	"drawTooltip": true, "paintTooltip": true, "ComposeTooltipHint": true,
}

// HoverBoxSites lists the places in production code of pkg and cmd that
// build a hover box beside the one builder: a hoverStyle, hoverBoxInk or
// hoverShape built outside the style table, a call of the composer outside
// its file, a description placed or drawn outside tooltipPicture and
// DrawBeside, a hover box recorded as a widget outside the composer, a box
// colour used outside the table and the composer, and a function that paints
// a hover box or a tooltip under another name. Tests are not scanned.
func HoverBoxSites(root string) ([]string, error) {
	var sites []string
	for _, top := range []string{"pkg", "cmd"} {
		err := filepath.WalkDir(filepath.Join(root, top), func(path string, d fs.DirEntry, err error) error {
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
			sites = append(sites, hoverBoxInSource(rel, src)...)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(sites)
	return sites, nil
}

// HoverBoxHomesHold reports whether the table builds the three types and the
// composer file holds the composer, so that a rename cannot leave the scan
// with nothing to exempt and nothing to find.
func HoverBoxHomesHold(root string) (bool, error) {
	found := map[string]bool{}
	for _, home := range []string{hoverStyleHome, hoverBoxHome} {
		src, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(home)))
		if err != nil {
			return false, err
		}
		file, err := parser.ParseFile(token.NewFileSet(), home, src, 0)
		if err != nil {
			return false, err
		}
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
	}
	return found["hoverStyle"] && found["hoverBoxInk"] && found["hoverShape"] &&
		found["hoverStyleFor"] && found["composeHoverBox"] && found["placeHover"] && found["drawHover"], nil
}

func hoverBoxInSource(name string, src []byte) []string {
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
	inTable, inComposer := name == hoverStyleHome, name == hoverBoxHome
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && hoverPainterName.MatchString(fn.Name.Name) && !hoverNotAPainter[fn.Name.Name] && !inComposer {
			add(fn.Pos(), fn.Name.Name+" paints a hover box beside composeHoverBox")
		}
	}
	ast.Inspect(file, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.CompositeLit:
			if id, ok := n.Type.(*ast.Ident); ok && hoverStyleTypes[id.Name] && !inTable {
				add(n.Pos(), id.Name+" is built outside the hover style table")
			}
		case *ast.CallExpr:
			id, ok := n.Fun.(*ast.Ident)
			if sel, isSel := n.Fun.(*ast.SelectorExpr); isSel {
				id, ok = sel.Sel, true
			}
			if !ok {
				return true
			}
			if hoverComposerNames[id.Name] && !inComposer {
				add(n.Pos(), id.Name+" is called outside the hover composer file")
			}
			if homes, placer := hoverPlacers[id.Name]; placer && !homes[name] && !inComposer {
				add(n.Pos(), id.Name+" places a hover description outside its builders")
			}
			if id.Name == "recordWidget" && len(n.Args) > 0 && !inComposer {
				if a, ok := n.Args[0].(*ast.Ident); ok && a.Name == "widgetHoverBox" {
					add(n.Pos(), "a hover box is recorded outside the composer")
				}
			}
		case *ast.Ident:
			if hoverColourNames[n.Name] && !inTable && !inComposer {
				add(n.Pos(), n.Name+" is a hover box colour used outside the table and the composer")
			}
		}
		return true
	})
	return hits
}
