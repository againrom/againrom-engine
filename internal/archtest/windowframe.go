package archtest

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/printer"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// windowFrameHome is the one file that paints a window, dialog or panel frame:
// a frameSpec is drawn there by drawFrame, and its shadow by frameShadowPlan.
// windowFrameFlatHome holds the flat panel painter drawFrame calls.
const (
	windowFrameHome     = "pkg/ui/widgetframe.go"
	windowFrameFlatHome = "pkg/ui/widgetkit/widgetpanel.go"
)

// windowFrameRoots are the trees the guard reads.
var windowFrameRoots = []string{"pkg/ui", "cmd/starter"}

// WindowFrameNotAFrame names, by file and function, the sites that draw an
// outline or a border that is not a frame and that widgetNotAFrame (the
// widget kit scan's list of the same kind, which also holds the mark helper,
// the autocast mark and the video frames) does not name: a glyph stroke.
// Each stays where it is.
var WindowFrameNotAFrame = map[string]string{
	"pkg/ui/editorpopup.go:drawPopup": "the card grip: three glyph strokes",
}

var windowFrameFuncName = regexp.MustCompile(`^(draw|paint)[A-Za-z]*(Frame|Panel|Border)$|^stroke(Rect|Box)$|^outline$`)

var windowFrameInk = regexp.MustCompile(`(?i)border|edge`)

// WindowFrameSites lists the places in the production code of pkg/ui and
// cmd/starter, outside the frame's home file, that paint a frame in a shape of
// their own: a function named for drawing a frame, panel or border; four
// strip rectangles cut from one rectangle's edges; a pixel set on a
// rectangle's last row or column; a vector stroke of a rectangle; a uniform
// fill in a border colour. A site named in WindowFrameNotAFrame is not listed;
// stale lists the entries of that table no site matched. Tests are not read.
func WindowFrameSites(root string) (sites, stale []string, err error) {
	used := map[string]bool{}
	for _, top := range windowFrameRoots {
		werr := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(top)), func(path string, d fs.DirEntry, err error) error {
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
			if !strings.HasSuffix(rel, ".go") || strings.HasSuffix(rel, "_test.go") || rel == windowFrameHome || rel == windowFrameFlatHome {
				return nil
			}
			src, rerr := os.ReadFile(path)
			if rerr != nil {
				return rerr
			}
			for _, h := range windowFrameInSource(rel, src) {
				if _, ok := WindowFrameNotAFrame[h.home]; ok {
					used[h.home] = true
					continue
				}
				if _, ok := widgetNotAFrame[h.home]; ok {
					continue
				}
				sites = append(sites, h.site)
			}
			return nil
		})
		if werr != nil {
			return nil, nil, werr
		}
	}
	for k := range WindowFrameNotAFrame {
		if !used[k] {
			stale = append(stale, k)
		}
	}
	sort.Strings(sites)
	sort.Strings(stale)
	return sites, stale, nil
}

type windowFrameHit struct{ home, site string }

func windowFrameInSource(name string, src []byte) []windowFrameHit {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, name, src, 0)
	if err != nil {
		return nil
	}
	text := func(n ast.Node) string {
		var b bytes.Buffer
		_ = printer.Fprint(&b, fset, n)
		return strings.Join(strings.Fields(b.String()), "")
	}
	var hits []windowFrameHit
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		home := name + ":" + fn.Name.Name
		seen := map[string]bool{}
		add := func(pos token.Pos, what string) {
			site := name + ":" + strconv.Itoa(fset.Position(pos).Line) + " " + what
			if !seen[site] {
				seen[site] = true
				hits = append(hits, windowFrameHit{home, site})
			}
		}
		if windowFrameFuncName.MatchString(fn.Name.Name) && !callsDrawFrame(fn) {
			add(fn.Pos(), fn.Name.Name+" paints a frame of its own")
		}
		if fn.Body == nil {
			continue
		}
		strips, paints := 0, false
		var firstStrip token.Pos
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.CallExpr:
				callee := text(n.Fun)
				if callee == "draw.Draw" || strings.HasSuffix(callee, ".SetRGBA") {
					paints = true
				}
				switch {
				case callee == "image.Rect" && isEdgeStrip(n, text):
					if strips == 0 {
						firstStrip = n.Pos()
					}
					strips++
				case strings.HasSuffix(callee, ".SetRGBA") || strings.HasSuffix(callee, ".Set"):
					for _, a := range n.Args {
						if lastLine.MatchString(text(a)) {
							add(n.Pos(), "a pixel is set on a rectangle's last row or column")
						}
					}
				case callee == "vector.StrokeRect" || callee == "ebitenutil.DrawRect":
					add(n.Pos(), callee+" draws a rectangle's border")
				case callee == "image.NewUniform":
					if len(n.Args) == 1 && windowFrameInk.MatchString(text(n.Args[0])) {
						add(n.Pos(), "a uniform fill in a border colour")
					}
				}
			case *ast.SelectorExpr:
				if text(n) == "vector.StrokeRect" {
					add(n.Pos(), "vector.StrokeRect is held outside the frame's home")
				}
			case *ast.CompositeLit:
				if text(n.Type) == "image.Uniform" && windowFrameInk.MatchString(text(n)) {
					add(n.Pos(), "a uniform fill in a border colour")
				}
			}
			return true
		})
		if strips >= 2 && paints {
			add(firstStrip, "a rectangle is stroked by hand from "+strconv.Itoa(strips)+" edge strips")
		}
	}
	return hits
}

var lastLine = regexp.MustCompile(`\.Max\.[XY]-1`)

// isEdgeStrip reports an image.Rect whose four arguments cut one edge strip
// off a rectangle B: its full width or height, a few pixels thick, flush with
// one side of B.
func isEdgeStrip(c *ast.CallExpr, text func(ast.Node) string) bool {
	if len(c.Args) != 4 {
		return false
	}
	a := [4]string{text(c.Args[0]), text(c.Args[1]), text(c.Args[2]), text(c.Args[3])}
	base := func(s, suffix string) (string, bool) {
		b, ok := strings.CutSuffix(s, suffix)
		return b, ok && b != ""
	}
	x0, ok0 := base(a[0], ".Min.X")
	y0, ok1 := base(a[1], ".Min.Y")
	x1, ok2 := base(a[2], ".Max.X")
	y1, ok3 := base(a[3], ".Max.Y")
	switch {
	case ok0 && ok1 && ok2 && x0 == y0 && x0 == x1 && strings.HasPrefix(a[3], x0+".Min.Y+"):
		return true // top
	case ok0 && ok2 && ok3 && x0 == x1 && x0 == y1 && strings.HasPrefix(a[1], x0+".Max.Y-"):
		return true // bottom
	case ok0 && ok1 && ok3 && x0 == y0 && x0 == y1 && strings.HasPrefix(a[2], x0+".Min.X+"):
		return true // left
	case ok1 && ok2 && ok3 && y0 == x1 && y0 == y1 && strings.HasPrefix(a[0], y0+".Max.X-"):
		return true // right
	}
	return false
}
