package archtest

import (
	"os"
	"strings"
	"testing"
)

// Outside the placement store, no production code of the editor builds where
// a popup stands or how big it is: each popup builder asks the store for its
// corner and its size, and the one drag, border drag and reopen runs on the
// popup's row.
func TestEditorPopupPlacementIsOneStore(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	holds, err := PopupPlacementHomeHolds(root)
	if err != nil {
		t.Fatal(err)
	}
	if !holds {
		t.Fatalf("%s no longer declares %v", popupStoreHome, popupStoreFuncs)
	}
	sites, err := PopupPlacementSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; read and write it through the placement store", s)
	}
}

func popupHits(name, src string) string {
	return strings.Join(popupPlacementInSource(name, []byte(src)), ", ")
}

// One planted instance per shape: each is named, and a look-alike is not.
func TestPopupPlacementScanSeesEachShape(t *testing.T) {
	const editor = "pkg/ui/mapeditorx.go"
	for _, tc := range []struct{ name, file, src, want string }{
		{"a popup corner field assigned", editor, `package ui
func f(e *E) { e.listAt = P{} }`, "pkg/ui/mapeditorx.go:2 writes a popup's place or size state outside the placement store"},
		{"a size assigned by index", editor, `package ui
func f(e *E) { e.sizes[1] = P{} }`, "pkg/ui/mapeditorx.go:2 writes a popup's place or size state outside the placement store"},
		{"a store row assigned", editor, `package ui
func f(e *E) { e.places[1].size = P{} }`, "pkg/ui/mapeditorx.go:2 writes a popup's place or size state outside the placement store"},
		{"a shift added to", editor, `package ui
func f(e *E) { e.promptShift = e.promptShift.Add(d) }`, "pkg/ui/mapeditorx.go:2 writes a popup's place or size state outside the placement store"},
		{"a counter bumped", editor, `package ui
func f(e *E) { e.listFit++ }`, "pkg/ui/mapeditorx.go:2 writes a popup's place or size state outside the placement store"},
		{"the store row's address taken", editor, `package ui
func f(e *E) { p := &e.places[1]; _ = p }`, "pkg/ui/mapeditorx.go:2 takes the address of a popup's place or size state outside the placement store"},
		{"a spec corner assigned a point", editor, `package ui
func f(s *S, e *E) { s.At = image.Pt(1, 2) }`, "pkg/ui/mapeditorx.go:2 writes a popup rectangle as a literal outside the placement store, pkg/ui/mapeditorx.go:2 sets a popup spec's corner from something but the placement store"},
		{"a spec size assigned", editor, `package ui
func f(s *S) { s.Size = P{} }`, "pkg/ui/mapeditorx.go:2 sets a popup spec's size or slack outside the placement store"},
		{"a spec slack assigned in a tuple", editor, `package ui
func f(s *S) { s.Slack, s.Top = true, 1 }`, "pkg/ui/mapeditorx.go:2 sets a popup spec's size or slack outside the placement store"},
		{"a spec literal with its own size", editor, `package ui
func f() popupSpec { return popupSpec{Size: P{}} }`, "pkg/ui/mapeditorx.go:2 sets a popup spec's size or slack outside the placement store"},
		{"a spec literal with its own corner", editor, `package ui
func f(e *E) popupSpec { return popupSpec{Title: "x", At: e.pointer} }`, "pkg/ui/mapeditorx.go:2 sets a popup spec's corner from something but the placement store"},
		{"a field kept for a popup's place", editor, `package ui
type chrome struct{ helpAt image.Point; objectAt image.Point }`, "pkg/ui/mapeditorx.go:2 declares field objectAt to keep a popup's place or size outside the placement store"},
		{"a popup size name", "pkg/ui/other.go", `package ui
var popupSize = 3`, "pkg/ui/other.go:2 names a popup corner, shift or size outside the placement store"},
		{"a corner constant name", "pkg/ui/other.go", `package ui
const popupCorner = 16`, "pkg/ui/other.go:2 names a popup corner, shift or size outside the placement store"},
		{"a shift map", "pkg/ui/other.go", `package ui
var popupShifts map[int]P`, "pkg/ui/other.go:2 names a popup corner, shift or size outside the placement store"},
		{"a rectangle literal", "pkg/ui/editorpopup.go", `package ui
func f(l *L) { l.Rect = image.Rect(0, 0, 1, 1) }`, "pkg/ui/editorpopup.go:2 writes a popup rectangle as a literal outside the placement store"},
		// Look-alikes that are not a second path.
		{"a spec corner from the store", editor, `package ui
func f(e *E) popupSpec { s := popupSpec{At: e.placedCorner(1, h)}; s.At = e.promptCorner(c, z); return s }`, ""},
		{"a read of the store", editor, `package ui
func f(e *E) P { return e.places[1].size }`, ""},
		{"a drag row's pointer", editor, `package ui
func f(e *E) *popupRowDrag { return &popupRowDrag{Row: 1, At: e.pointer} }`, ""},
		{"a write outside the editor files", "pkg/ui/other.go", `package ui
func f(e *E) { e.sizes[1] = P{}; s.At = P{} }`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := popupHits(tc.file, tc.src); got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}
