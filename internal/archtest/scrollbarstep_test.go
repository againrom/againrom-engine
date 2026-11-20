package archtest

import (
	"os"
	"strings"
	"testing"
)

// No screen computes a bar position from a press, or applies a bar request,
// beside the one step: each holds a scrollOwner that the step reads.
func TestScrollBarsStepThroughOneOwnerValue(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatalf("module root not found from %s: %v", wd, err)
	}
	sites, err := ScrollBarSites(root)
	if err != nil {
		t.Fatalf("measure: %v", err)
	}
	for _, s := range sites {
		t.Errorf("%s; hold a scrollOwner and drive it through scrollBarInput.drive", s)
	}
}

func TestScrollBarScanSeesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, file, src, want string }{
		{"a drag row mapped by a screen", "pkg/ui/x.go", "package ui\nfunc f(b vScrollBar) int { return b.dragPos(3) }", "pkg/ui/x.go:2 dragPos is called outside " + scrollBarHome},
		{"a list request applied by a screen", "pkg/ui/x.go", "package ui\nfunc f(m *Picker) { listBarRequest(m, barLineUp, 0) }", "pkg/ui/x.go:2 listBarRequest is called outside " + scrollListHome},
		{"the step's file", scrollBarHome, "package ui\nfunc f(b vScrollBar) int { return b.dragPos(3) }", ""},
		{"the model's apply", scrollListHome, "package ui\nfunc f(m *Picker) { listBarRequest(m, barLineUp, 0) }", ""},
	} {
		if got := strings.Join(scrollBarInSource(tc.file, []byte(tc.src)), ", "); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}
