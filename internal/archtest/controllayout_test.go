package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckControlLayoutNamesEachShape plants one instance of each shape the
// scan refuses, and the shapes it admits.
func TestCheckControlLayoutNamesEachShape(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a switch over controls returning rectangles", "pkg/ui/a.go",
			"package ui\nfunc rectOf(c int) image.Rectangle {\nswitch c {\ncase 0:\nreturn image.Rect(1, 2, 3, 4)\ncase 1:\nreturn image.Rect(5, 6, 7, 8)\n}\nreturn image.Rectangle{}\n}\n",
			"switch over control identities"},
		{"the same switch in pkg/game", "pkg/game/a.go",
			"package game\nfunc rectOf(c int) image.Rectangle {\nswitch c {\ncase 0:\nreturn image.Rect(1, 2, 3, 4)\ncase 1:\nreturn image.Rect(5, 6, 7, 8)\n}\nreturn image.Rectangle{}\n}\n",
			"switch over control identities"},
		{"a hit test with a rectangle literal", "pkg/ui/save_dialog.go",
			"package ui\nfunc (d *x) hit(p image.Point) bool { return p.In(image.Rect(1, 2, 3, 4)) }\n",
			"a rectangle literal"},
		{"a hit test with a package variable", "pkg/ui/savepage.go",
			"package ui\nfunc hit(p image.Point) bool { return p.In(saveBoxRect) }\n",
			"the variable saveBoxRect"},
		{"a hit test with a page field", "pkg/ui/load_window.go",
			"package ui\nfunc hit(v pageView, p image.Point) bool { return p.In(v.page.NameBox) }\n",
			"the page field NameBox"},
		{"a hit test with a page button", "pkg/ui/load_window.go",
			"package ui\nfunc hit(v pageView, p image.Point) bool { return p.In(v.page.Buttons[0]) }\n",
			"the page's button row"},
		{"a control rectangle in a package variable", "pkg/ui/save_dialog_draw.go",
			"package ui\nvar saveExtraRect = image.Rect(1, 2, 3, 4)\n",
			"holds a rectangle in a package variable"},
		{"a second descriptor shape", "pkg/ui/savepage.go",
			"package ui\ntype saveControlGeometry struct {\nID int\nRect image.Rectangle\n}\n",
			"second descriptor shape"},
		{"a layout read", "pkg/ui/load_window.go",
			"package ui\nfunc hit(v pageView, p image.Point) bool { return v.page.layout().holds(pageNameControl, p) }\n", ""},
		{"a hover test on a local", "pkg/ui/load_draw.go",
			"package ui\nfunc draw(pointer image.Point, r image.Rectangle) bool { return pointer.In(r) }\n", ""},
		{"a hit test off the surfaces", "pkg/ui/shopscreen.go",
			"package ui\nfunc hit(p image.Point) bool { return p.In(image.Rect(1, 2, 3, 4)) }\n", ""},
		{"a test file", "pkg/ui/save_dialog_test.go",
			"package ui\nfunc hit(p image.Point) bool { return p.In(image.Rect(1, 2, 3, 4)) }\n", ""},
		{"a switch that returns one rectangle", "pkg/ui/a.go",
			"package ui\nfunc rectOf(c int) image.Rectangle {\nswitch c {\ncase 0:\nreturn image.Rect(1, 2, 3, 4)\n}\nreturn image.Rectangle{}\n}\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckControlLayout(map[string]string{tc.file: tc.src}) {
				if !strings.Contains(v.Reason, "nothing there needs it") {
					got = append(got, v.Reason)
				}
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckControlLayout = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckControlLayout = %v, want one naming %q", got, tc.want)
			}
		})
	}
	if vs := CheckControlLayout(nil); len(vs) != 1 {
		t.Fatalf("CheckControlLayout(nil) = %v, want one violation", vs)
	}
}

// TestControlLayoutHasOneDescriptor is the live-tree half.
func TestControlLayoutHasOneDescriptor(t *testing.T) {
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
	for _, v := range CheckControlLayout(files) {
		t.Errorf("%s", v)
	}
}
