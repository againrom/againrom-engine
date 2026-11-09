package archtest

import (
	"os"
	"strings"
	"testing"
)

// Outside the list model, no production code of the screens or the launcher
// reads a wheel field to move a list: every list is a row of the model's
// table and steps through it.
func TestScrollingListsStepThroughOneModel(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	holds, err := ScrollListHomeHolds(root)
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatalf("%s no longer declares %v and scrollSpec", scrollListHome, scrollListFuncs)
	}
	sites, err := ScrollListSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; give the list a scrollSpec row and step it through the model", s)
	}
}

func scrollHits(name, src string) string {
	return strings.Join(scrollListInSource(name, []byte(src)), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestScrollListScanSeesEachShape(t *testing.T) {
	const ui = "pkg/ui/screenx.go"
	for _, tc := range []struct{ name, file, src, want string }{
		{"a wheel field read", ui, `package ui
func f(in appInput, l *L) { if in.WheelY > 0 { l.top-- } }`, "pkg/ui/screenx.go:2 a wheel field is read beside the list model"},
		{"a wheel field of the viewer read", ui, `package ui
func f(in appInput) bool { return in.Viewer.WheelY != 0 }`, "pkg/ui/screenx.go:2 a wheel field is read beside the list model"},
		{"the sampler called", ui, `package ui
func f() float64 { _, wy := ebiten.Wheel(); return wy }`, "pkg/ui/screenx.go:2 Wheel is called outside the list model"},
		{"the sampler called in the launcher", "cmd/starter/x.go", `package main
func f() float64 { _, wy := ebiten.Wheel(); return wy }`, "cmd/starter/x.go:2 Wheel is called outside the list model"},
		{"a wheel accessor called", ui, `package ui
func f(in appInput) float64 { return in.wheelNotches() }`, "pkg/ui/screenx.go:2 wheelNotches is called outside the list model"},
		{"the wheel sign called", ui, `package ui
func f(in appInput) int { return wheelSign(in) }`, "pkg/ui/screenx.go:2 wheelSign is called outside the list model"},
		{"a model function declared elsewhere", ui, `package ui
func listKey(m *Picker, up, down, pageUp, pageDown bool) bool { return false }`, "pkg/ui/screenx.go:2 listKey is declared beside the list model"},
		{"a model method declared elsewhere", ui, `package ui
func (s scrollSpec) wheelStep(in appInput) int { return 0 }`, "pkg/ui/screenx.go:2 wheelStep is declared beside the list model"},
		{"a scrollSpec row written elsewhere", ui, `package ui
var rows = scrollSpec{wheel: 3}`, "pkg/ui/screenx.go:2 a scrollSpec row is written beside the list model"},
		{"a scrollSpec row written in a function", ui, `package ui
func f() int { s := scrollSpec{wheel: 3}; return s.wheel }`, "pkg/ui/screenx.go:2 a scrollSpec row is written beside the list model"},
		{"a list moved by a number", ui, `package ui
func f(l *Picker) { l.Move(-3) }`, "pkg/ui/screenx.go:2 Move is given a step written as a number"},
		{"a pack scrolled by a number", ui, `package ui
func f(v *Viewer) { v.ScrollPack(+1) }`, "pkg/ui/screenx.go:2 ScrollPack is given a step written as a number"},
		{"a named wheel step declared", ui, `package ui
const helpWheelLines = 4`, "pkg/ui/screenx.go:2 helpWheelLines is a wheel step declared beside the list model"},
		{"a bar request dispatched", ui, `package ui
func f(req barRequest, t *int) {
	switch req {
	case barLineUp:
		*t--
	}
}`, "pkg/ui/screenx.go:4 a bar request barLineUp is dispatched beside the list model"},
		{"another type's keyStep is not the model's", ui, `package ui
func (e *MapEditor) keyStep(key string) error { return nil }`, ""},
		{"a wheel field written is not a read", ui, `package ui
func f(in *appInput) { in.WheelY = 0 }`, ""},
		{"a move by zero is not a step", ui, `package ui
func f(l *Picker) { l.Move(0) }`, ""},
		{"a move by a computed step is not a literal", ui, `package ui
func f(l *Picker, in appInput) { l.Move(pickerList.wheelStep(in)) }`, ""},
		{"the brush reads its accessor", "pkg/ui/mapeditorbrush.go", `package ui
func (e *MapEditor) brushWheel(s wheelSource) { _ = s.wheelNotches() }`, ""},
		{"the launcher takes the model's lines", "cmd/starter/window.go", `package main
func f() { if _, dy := ebiten.Wheel(); dy != 0 { a.wheel(p, widgetkit.ListWheelLines(dy)) } }`, ""},
		{"the chargen focus moves by a number", "pkg/ui/app.go", `package ui
func (a *App) stepChargen(c *Chargen) { c.Move(-1) }`, ""},
	} {
		if got := scrollHits(tc.file, tc.src); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
