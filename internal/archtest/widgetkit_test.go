package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckWidgetKitNamesEachViolation drives the evaluator with synthetic
// sources, one case per arm.
func TestCheckWidgetKitNamesEachViolation(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a named frame painter outside the kit", "pkg/ui/a.go",
			"package ui\nfunc drawPanelBorder() {}\n", "drawPanelBorder is a frame painter"},
		{"a function calling outline", "pkg/ui/a.go",
			"package ui\nfunc paintRow() { outline(nil, r, c) }\n", "paintRow is a frame painter"},
		{"a function reading frame pieces", "pkg/ui/a.go",
			"package ui\nfunc tiles(a *DialogFrame) { _ = a.Pieces }\n", "tiles is a frame painter"},
		{"a frame painter inside the kit", "pkg/ui/widgetx.go",
			"package ui\nfunc drawPanelBorder() { outline(nil, r, c) }\n", ""},
		{"a bool press latch", "pkg/ui/a.go",
			"package ui\ntype s struct{ okPress bool }\n", "okPress is a press latch outside the kit"},
		{"an int press latch in render", "pkg/render/x/a.go",
			"package x\ntype s struct{ press int }\n", "press is a press latch outside the kit"},
		{"the kit latch", "pkg/ui/a.go",
			"package ui\ntype s struct{ okPress buttonLatch; press latch.Latch }\n", ""},
		{"a field that is not a latch by name", "pkg/ui/a.go",
			"package ui\ntype s struct{ PrimaryPressed, pressX bool }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckWidgetKit(map[string]string{tc.file: tc.src}) {
				if !strings.Contains(v.Reason, "nothing matches it") {
					got = append(got, v.Reason)
				}
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckWidgetKit = %v, want none", got)
				}
				return
			}
			if len(got) != 1 || !strings.Contains(got[0], tc.want) {
				t.Fatalf("CheckWidgetKit = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestCheckWidgetKitRefusesAStaleListEntry pins the falling-only lists: an
// entry nothing matches is a violation.
func TestCheckWidgetKitRefusesAStaleListEntry(t *testing.T) {
	vs := CheckWidgetKit(map[string]string{"pkg/ui/a.go": "package ui\n"})
	want := len(widgetFrameDebt) + len(widgetNotAFrame) + len(widgetLatchDebt) + len(widgetNotALatch)
	if len(vs) != want {
		t.Fatalf("CheckWidgetKit over an empty package = %d violations, want %d stale entries: %v", len(vs), want, vs)
	}
	if vs := CheckWidgetKit(nil); len(vs) != 1 {
		t.Fatalf("CheckWidgetKit(nil) = %v, want one violation", vs)
	}
}

// TestFramesAndLatchesHaveOneBuilder is the live-tree half.
func TestFramesAndLatchesHaveOneBuilder(t *testing.T) {
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
	for _, v := range CheckWidgetKit(files) {
		t.Errorf("%s", v)
	}
}
