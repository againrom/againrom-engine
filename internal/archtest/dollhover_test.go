package archtest

import (
	"os"
	"strings"
	"testing"
)

// Outside the doll description and its hit function, no production code of
// the game or its screens answers a question of a paper doll: each screen
// builds a description of its doll and asks dollSlotAt.
func TestPaperDollIsAnsweredByOneHitFunction(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	holds, err := DollHoverHomeHolds(root)
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatalf("%s no longer holds %s, %s and DrawBeside", dollHoverHome, dollSlotFunc, dollInfoFunc)
	}
	sites, err := DollHoverSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; describe the doll with DollHover and ask dollSlotAt", s)
	}
}

func dollHoverHits(src string) string {
	return strings.Join(dollHoverInSource("x.go", []byte(src)), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestPaperDollScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, src, want string }{
		{"a hit function named for a worn slot", `package x
func (v *V) inventoryWornSlotAt(x, y int) (int, bool) { return 0, false }`, "x.go:2 inventoryWornSlotAt answers a doll's hit beside dollSlotAt"},
		{"a hit function named for a doll area", `package x
func shopDollAreaAt(p P) bool { return false }`, "x.go:2 shopDollAreaAt answers a doll's hit beside dollSlotAt"},
		{"a hit function named for a figure slot", `package x
func (c *C) dollFigureSlotAt(x, y int) (int, bool) { return 0, false }`, "x.go:2 dollFigureSlotAt answers a doll's hit beside dollSlotAt"},
		{"a slot map read", `package x
func f(m *M, p P) (int, bool) {
	n, ok := m.At(p.X, p.Y)
	return n, ok
}`, "x.go:3 a slot map is read for a slot and whether it is marked"},
		{"a slot map read through a field", `package x
func f(d D, p P) int {
	n, ok := d.Preview.Mask.At(p.X-1, p.Y-1)
	if !ok { return 0 }
	return n
}`, "x.go:3 a slot map is read for a slot and whether it is marked"},
		{"a loop over cells returning the index", `package x
func f(cells []R, p P) (int, bool) {
	for i, r := range cells {
		if p.In(r) {
			return i, true
		}
	}
	return 0, false
}`, "x.go:3 a loop over rectangles returns the index of the one a point is in"},
		{"a slot's text read by index", `package x
func f(v *V, i int) []string { return v.SlotInfo[i] }`, "x.go:2 a slot's item text is read by index"},
		{"a slot's text read by index from a hero", `package x
func f(v *V, n int) []string { return v.HeroSlotInfo[n-1] }`, "x.go:2 a slot's item text is read by index"},
		{"the worn slot rectangles walked", `package x
func f() { for range wornSlotRects() {} }`, "x.go:2 wornSlotRects is called outside the doll builders"},
		{"a tip drawn beside the pointer", `package x
func f(d *I) { drawShopHover(d, nil, nil, P{}) }`, "x.go:2 drawShopHover is called outside the doll builders"},
		{"the pane's figure area read", `package x
func f(r R, p P) bool { return p.In(characterPaneFigureRect(r)) }`, "x.go:2 characterPaneFigureRect is called outside the doll builders"},
		{"the pane's figure origin read", `package x
func f(r R) P { return characterPaneFigureOrigin(r) }`, "x.go:2 characterPaneFigureOrigin is called outside the doll builders"},
		{"a tip key spelled in a target", `package x
func f(id int) tooltipTarget { return tooltipTarget{tooltipItem, fmt.Sprintf("item/%d", id), nil, nil} }`, "x.go:2 a tip key is spelled outside a DollTip"},

		{"the hit function itself", `package x
func dollSlotAt(d D, p P) (int, bool) {
	for i, r := range d.Cells {
		if p.In(r) {
			return i, true
		}
	}
	n, ok := d.Mask.At(p.X, p.Y)
	return n - 1, ok
}`, ""},
		{"the text read of the description", `package x
func (d D) Lines(slot int) []string { return d.Info[slot] }`, ""},
		{"a builder reading the geometry", `package x
func (v *V) dollHoverOver(m *M) D {
	return D{At: characterPaneFigureOrigin(v.box), Clip: characterPaneFigureRect(v.box)}
}
func (v *V) wornHover() D {
	for i, r := range wornSlotRects() { _ = i; _ = r }
	return D{}
}`, ""},
		{"a tip key inside a DollTip", `package x
func f(id int) D { return D{Tip: DollTip{Key: fmt.Sprintf("hero-item/%d", id)}} }`, ""},
		{"a write of a slot's text", `package x
func f(v *V, n int) { v.SlotInfo[n] = nil }`, ""},
		{"an image read", `package x
func f(img *I, p P) { c := img.At(p.X, p.Y); _ = c }`, ""},
		{"a loop returning an index without a point test", `package x
func f(cells []R) (int, bool) {
	for i, r := range cells {
		if r.Empty() {
			return i, true
		}
	}
	return 0, false
}`, ""},
		{"a loop over buttons returning the index a point is in", `package x
func f(buttons []R, p P) int {
	for i, r := range buttons {
		if p.In(r) {
			return i
		}
	}
	return -1
}`, ""},
		{"a hit function of another kind", `package x
func (v *V) gameAreaAt(x, y int) bool { return false }`, ""},
		{"an index of a field that is not a slot's text", `package x
func f(v *V, i int) string { return v.Names[i] }`, ""},
	} {
		if got := dollHoverHits(tc.src); got != tc.want {
			t.Errorf("%s: hits = %q, want %q", tc.name, got, tc.want)
		}
	}
}
