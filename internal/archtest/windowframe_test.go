package archtest

import (
	"os"
	"strings"
	"testing"
)

// Every window, dialog and panel frame is a frameSpec drawn by drawFrame in
// pkg/ui/widgetframe.go: no other file of pkg/ui or cmd/starter paints a
// frame in a shape of its own. The sites that draw a mark or an outline that
// is not a frame are named in WindowFrameNotAFrame.
func TestEveryFrameIsDrawnByTheFrameBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	if _, err := os.Stat(root + "/" + windowFrameHome); err != nil {
		t.Fatalf("the frame builder's home is gone: %v", err)
	}
	sites, stale, err := WindowFrameSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; describe the frame as a frameSpec and draw it with drawFrame", s)
	}
	for _, s := range stale {
		t.Errorf("WindowFrameNotAFrame names %s, which no longer draws an outline", s)
	}
}

func windowFrameHits(src string) string {
	var out []string
	for _, h := range windowFrameInSource("x.go", []byte(src)) {
		out = append(out, h.site)
	}
	return strings.Join(out, ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestWindowFrameScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"a function named for drawing a panel", `package x
func drawInfoPanel(dst int) {}`, "x.go:2 drawInfoPanel paints a frame of its own"},
		{"a function named for drawing a border", `package x
func paintWindowBorder(dst int) {}`, "x.go:2 paintWindowBorder paints a frame of its own"},
		{"a stroke helper of a package's own", `package x
func strokeRect(dst int) {}`, "x.go:2 strokeRect paints a frame of its own"},
		{"four edge strips cut from one rectangle", `package x
func f(r R) {
	for _, e := range []R{
		image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+1),
		image.Rect(r.Min.X, r.Max.Y-1, r.Max.X, r.Max.Y),
		image.Rect(r.Min.X, r.Min.Y, r.Min.X+1, r.Max.Y),
		image.Rect(r.Max.X-1, r.Min.Y, r.Max.X, r.Max.Y),
	} {
		draw.Draw(dst, e, c, image.Point{}, draw.Src)
	}
}`, "x.go:4 a rectangle is stroked by hand from 4 edge strips"},
		{"edge strips cut only to lay out", `package x
func f(r R) (a, b R) {
	a = image.Rect(r.Min.X, r.Min.Y, r.Max.X, r.Min.Y+4)
	b = image.Rect(r.Min.X, r.Max.Y-4, r.Max.X, r.Max.Y)
	return
}`, ""},
		{"a pixel set on the last row", `package x
func f(dst D, r R) {
	for x := r.Min.X; x < r.Max.X; x++ {
		dst.SetRGBA(x, r.Max.Y-1, c)
	}
}`, "x.go:4 a pixel is set on a rectangle's last row or column"},
		{"a vector stroke of a rectangle", `package x
func f(dst D) {
	vector.StrokeRect(dst, 0, 0, 1, 1, 1, c, false)
}`, "x.go:3 vector.StrokeRect draws a rectangle's border, x.go:3 vector.StrokeRect is held outside the frame's home"},
		{"the vector stroke held in a variable", `package x
var stroke = vector.StrokeRect`, ""},
		{"a uniform fill in a border colour", `package x
func f(dst D) {
	draw.Draw(dst, r, &image.Uniform{C: ink.Border}, image.Point{}, draw.Src)
}`, "x.go:3 a uniform fill in a border colour"},
		{"a uniform fill made by NewUniform in an edge colour", `package x
func f(dst D) {
	draw.Draw(dst, r, image.NewUniform(ink.Edge), image.Point{}, draw.Src)
}`, "x.go:3 a uniform fill in a border colour"},
		{"a single rule is not a frame", `package x
func f(dst D, bar R) {
	draw.Draw(dst, image.Rect(bar.Min.X, bar.Max.Y-1, bar.Max.X, bar.Max.Y), image.NewUniform(gold), image.Point{}, draw.Src)
}`, ""},
		{"a filled rectangle with an ordinary colour", `package x
func f(dst D, r R) {
	draw.Draw(dst, r, &image.Uniform{C: ground}, image.Point{}, draw.Src)
}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := windowFrameHits(tc.src); got != tc.want {
				t.Errorf("sites = %q, want %q", got, tc.want)
			}
		})
	}
}

// A site in the not-a-frame table is skipped, and an entry no site matches is
// reported as stale.
func TestWindowFrameTableNamesOnlyLiveSites(t *testing.T) {
	hits := windowFrameInSource("pkg/ui/shopscreen.go", []byte(`package ui
func outline(dst D, r R) { dst.SetRGBA(r.Max.X-1, 0, c) }`))
	if len(hits) == 0 || hits[0].home != "pkg/ui/shopscreen.go:outline" {
		t.Fatalf("the mark helper is not seen: %v", hits)
	}
	if _, ok := widgetNotAFrame[hits[0].home]; !ok {
		t.Fatalf("%s is not in the table", hits[0].home)
	}
}
