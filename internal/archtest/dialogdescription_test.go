package archtest

import (
	"os"
	"strings"
	"testing"
)

// Every modal dialog is one value of the one description, painted by the one
// painter: no second description type, no branch on a notice style, and no
// other function that draws a frame and pushes a button itself.
func TestDialogHasOneDescriptionAndOnePainter(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := DialogDescriptionSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Error(s)
	}
}

func dialogHits(name, src string) string {
	return strings.Join(dialogDescriptionInSource(name, []byte(src), nil), ", ")
}

// One planted instance per shape is named, and a look-alike is not.
func TestDialogDescriptionScanSeesEachShape(t *testing.T) {
	const file = "pkg/ui/x.go"
	for _, tc := range []struct{ name, src, want string }{
		{"a retired type", `package ui
type NoticeLayout struct{ X int }`, "pkg/ui/x.go:2 declares NoticeLayout, a dialog description beside popupSpec"},
		{"a parallel description", `package ui
type second struct {
	Box   R
	Frame *DialogFrame
}`, "pkg/ui/x.go:2 declares second with a Box and a DialogFrame, a dialog description beside popupSpec"},
		{"a retired painter", `package ui
func renderDialogueNotice() {}`, "pkg/ui/x.go:2 declares renderDialogueNotice, a dialog painter beside drawPopup"},
		{"a style branch", `package ui
func f(l L) bool { return l.Style == NoticeStyleDialogue }`, "pkg/ui/x.go:2 branches on a notice style; read a row of noticeStyles"},
		{"a style switch", `package ui
func f(l L) {
	switch l.Style {
	case NoticeStylePlain:
	}
}`, "pkg/ui/x.go:4 branches on a notice style; read a row of noticeStyles"},
		{"a private painter", `package ui
func paintBox(d I, b B) {
	drawFrame(d, f)
	drawPushButton(d, b)
}`, "pkg/ui/x.go:2 paintBox draws a frame and pushes a button itself; make it a value drawPopup paints"},
		{"the one description", `package ui
type popupSpec struct {
	Box   R
	Frame *DialogFrame
}`, ""},
		{"the one painter", `package ui
func drawPopup(d I, b B) {
	drawFrame(d, f)
	drawPushButton(d, b)
}`, ""},
		{"a frame alone", `package ui
func paintFrame(d I) { drawFrame(d, f) }`, ""},
		{"a value of a style", `package ui
var v = Dialog{Style: NoticeStyleDialogue}`, ""},
		{"a Box without the art", `package ui
type box struct {
	Box   R
	Frame *OtherFrame
}`, ""},
	} {
		if got := dialogHits(file, tc.src); got != tc.want {
			t.Errorf("%s:\n got  %q\n want %q", tc.name, got, tc.want)
		}
	}
}
