package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckDoubleClickNamesEachViolation drives the scan with synthetic
// sources, one case per arm.
func TestCheckDoubleClickNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a press time on a screen", "pkg/ui/a.go",
			"package ui\ntype s struct{ rowClickAt time.Time }\n", "rowClickAt holds a click time"},
		{"a window constant", "pkg/ui/a.go",
			"package ui\nconst useTapWindow = 500 * time.Millisecond\n", "useTapWindow holds a click time"},
		{"a window variable in the game package", "pkg/game/a.go",
			"package game\nvar doublePressWindow time.Duration\n", "doublePressWindow holds a click time"},
		{"a detector by name", "pkg/ui/a.go",
			"package ui\nfunc listDoubleClick() bool { return false }\n", "listDoubleClick is a double-click detector"},
		{"a detector type by name", "pkg/game/a.go",
			"package game\ntype double_click struct{}\n", "double_click is a double-click detector"},
		{"the system's time read elsewhere", "pkg/ui/a.go",
			"package ui\nvar p = dll.NewProc(\"GetDoubleClickTime\")\n", "GetDoubleClickTime reads the system's double-click setting"},
		{"the macOS interval read elsewhere", "pkg/ui/a.go",
			"package ui\nfunc f() { _ = objc.RegisterName(\"doubleClickInterval\") }\n", "doubleClickInterval reads the system's double-click setting"},
		{"the detector itself", "pkg/ui/doubleclick.go",
			"package ui\ntype doubleClick struct{ at time.Time }\n", ""},
		{"the system reading itself", "pkg/ui/systemclick/x_windows.go",
			"package systemclick\nvar p = dll.NewProc(\"GetDoubleClickTime\")\n", ""},
		{"a time that is no click", "pkg/ui/a.go",
			"package ui\ntype s struct{ worldMapTickAt time.Time }\n", ""},
		{"a click that is no time", "pkg/ui/a.go",
			"package ui\ntype s struct{ invClickFrames int; shopUseTap *shopUseTap }\n", ""},
		{"a test file outside pkg", "cmd/a/a.go",
			"package main\nfunc listDoubleClick() {}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckDoubleClick(map[string]string{tc.file: tc.src}) {
				if !strings.Contains(v.Reason, "nothing matches it") {
					got = append(got, v.Reason)
				}
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckDoubleClick = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckDoubleClick = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestCheckDoubleClickRefusesAStaleListEntry pins the lists: an entry
// nothing matches is a violation, so the debt only falls.
func TestCheckDoubleClickRefusesAStaleListEntry(t *testing.T) {
	vs := CheckDoubleClick(map[string]string{"pkg/ui/a.go": "package ui\n"})
	if want := len(doubleClickNotADetector) + len(doubleClickDebt); len(vs) != want {
		t.Fatalf("CheckDoubleClick over an empty package = %d violations, want %d stale entries: %v", len(vs), want, vs)
	}
	if vs := CheckDoubleClick(nil); len(vs) != 1 {
		t.Fatalf("CheckDoubleClick(nil) = %v, want one violation", vs)
	}
}

// TestDoubleClicksHaveOneDetector is the live-tree half.
func TestDoubleClicksHaveOneDetector(t *testing.T) {
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
	for _, v := range CheckDoubleClick(files) {
		t.Errorf("%s", v)
	}
}
