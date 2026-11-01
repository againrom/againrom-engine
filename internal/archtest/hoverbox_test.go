package archtest

import (
	"os"
	"strings"
	"testing"
)

// Every hover box is a description the one composer draws in the one style
// table: no other production code builds a hover style, paints a box or
// places a description.
func TestHoverBoxHasOneBuilder(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	holds, err := HoverBoxHomesHold(root)
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatalf("%s and %s no longer hold the hover style table and composer", hoverStyleHome, hoverBoxHome)
	}
	sites, err := HoverBoxSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; describe the box and draw it through composeHoverBox", s)
	}
}

func hoverBoxHits(name, src string) string {
	return strings.Join(hoverBoxInSource(name, []byte(src)), ", ")
}

// One planted instance per shape is named, and a look-alike is not.
func TestHoverBoxScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, file, src, want string }{
		{"a style literal", "pkg/ui/x.go", `package x
func f() *hoverStyle { return &hoverStyle{} }`, "pkg/ui/x.go:2 hoverStyle is built outside the hover style table"},
		{"a box colour literal", "pkg/ui/x.go", `package x
var b = hoverBoxInk{}`, "pkg/ui/x.go:2 hoverBoxInk is built outside the hover style table"},
		{"a shape literal", "pkg/ui/x.go", `package x
var s = &hoverShape{}`, "pkg/ui/x.go:2 hoverShape is built outside the hover style table"},
		{"the composer called", "pkg/ui/x.go", `package x
func f() { composeHoverBox(nil, nil, nil, nil) }`, "pkg/ui/x.go:2 composeHoverBox is called outside the hover composer file"},
		{"the size called", "pkg/ui/x.go", `package x
func f() { _ = hoverBoxSize(nil, nil, nil) }`, "pkg/ui/x.go:2 hoverBoxSize is called outside the hover composer file"},
		{"a description placed", "pkg/ui/x.go", `package x
func f() { placeHover(d, nil) }`, "pkg/ui/x.go:2 placeHover places a hover description outside its builders"},
		{"a description drawn", "pkg/ui/x.go", `package x
func f() { drawHover(dst, d) }`, "pkg/ui/x.go:2 drawHover places a hover description outside its builders"},
		{"a box recorded", "pkg/ui/x.go", `package x
func f() { recordWidget(widgetHoverBox, r, nil) }`, "pkg/ui/x.go:2 a hover box is recorded outside the composer"},
		{"a box colour used", "pkg/ui/x.go", `package x
var c = hoverFill`, "pkg/ui/x.go:2 hoverFill is a hover box colour used outside the table and the composer"},
		{"a painter under another name", "pkg/ui/x.go", `package x
func drawShopHover(dst *I) {}`, "pkg/ui/x.go:2 drawShopHover paints a hover box beside composeHoverBox"},
		{"a tooltip painter", "pkg/town/x.go", `package x
func (s *S) paintTooltipBox() {}`, "pkg/town/x.go:2 paintTooltipBox paints a hover box beside composeHoverBox"},

		{"the table building a style", hoverStyleHome, `package x
func f() *hoverStyle { return &hoverStyle{ink: hoverFill} }`, ""},
		{"the composer composing", hoverBoxHome, `package x
func composeHoverBox() { recordWidget(widgetHoverBox, r, nil); _ = hoverBoxSize(a, b, c) }`, ""},
		{"the tooltip placing a description", "pkg/ui/tooltip.go", `package x
func f() { placeHover(d, nil) }`, ""},
		{"the doll tip drawing one", "pkg/ui/dollhover.go", `package x
func f() { drawHover(dst, d) }`, ""},
		{"the presented tooltip", "pkg/ui/tooltip_runtime.go", `package x
func (a *A) drawTooltip(dst *I) {}`, ""},
		{"a look-alike that is a painter by name", "pkg/ui/x.go", `package x
func (a *A) hoverStyle() *hoverStyle { return a.tooltip.style }
func drawHoverDelay() {}`, "pkg/ui/x.go:3 drawHoverDelay paints a hover box beside composeHoverBox"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := hoverBoxHits(tc.file, tc.src)
			switch {
			case tc.want == "" && got != "":
				t.Errorf("flagged %q", got)
			case tc.want != "" && !strings.Contains(got, tc.want):
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
