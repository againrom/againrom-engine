package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckCoverScreensNamesEachViolation drives the scan with planted copies
// of each named shape.
func TestCheckCoverScreensNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, src, want string
	}{
		{"a flag for the Shop page over the map", "package ui\ntype flow struct{ shopOverMap bool }\n", "flow.shopOverMap names a screen over the mission"},
		{"a back screen for the panel", "package ui\ntype flow struct{ docBack Screen }\n", "flow.docBack names a screen over the mission"},
		{"a back screen for a new screen of that shape", "package ui\ntype flow struct{ shopBack Screen }\n", "flow.shopBack names a screen over the mission"},
		{"a request flag on the Viewer", "package ui\ntype Viewer struct{ docRequest bool }\n", "Viewer.docRequest is a request"},
		{"a second raise", "package ui\nfunc (v *Viewer) RaiseCover() {}\nfunc (v *Viewer) RaiseShop() {}\n", "more than one Raise method"},
		{"a second take", "package ui\nfunc (v *Viewer) TakeCoverRequest() {}\nfunc (v *Viewer) TakeShopRequest() {}\n", "more than one Take...Request method"},
		{"the cover record", "package ui\ntype flow struct{ cover coverShown; menuBack Screen; docPanel *documentPanel }\n" +
			"type Viewer struct{ coverRequests uint8 }\nfunc (v *Viewer) RaiseCover() {}\nfunc (v *Viewer) TakeCoverRequest() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckCoverScreens(map[string]string{"pkg/ui/a.go": tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckCoverScreens = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckCoverScreens = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckCoverScreens(nil); len(vs) != 1 {
		t.Fatalf("CheckCoverScreens(nil) = %v, want one violation", vs)
	}
}

// TestCoverScreensHaveOneRecord is the live-tree half.
func TestCoverScreensHaveOneRecord(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	files, err := LoadDrawnTextSources(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckCoverScreens(files) {
		t.Errorf("%s", v)
	}
}
