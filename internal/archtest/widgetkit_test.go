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
		{"a canvas button painter named as one", "pkg/ui/a.go",
			"package ui\nfunc drawRowButton(dst *ebiten.Image, x, y, w, h int, selected bool) { ebitenutil.DrawRect(dst, 0, 0, 1, 1, c) }\n",
			"drawRowButton is a push button painter"},
		{"a plaque painter choosing its ink", "pkg/ui/a.go",
			"package ui\nfunc composeCommands(hover bool) { ink := plaqueCommandInk.Rest; _ = ink }\n",
			"composeCommands is a push button painter outside the kit (picks the plaque ink plaqueCommandInk.Rest)"},
		{"a plaque painter with its own ink table", "pkg/ui/a.go",
			"package ui\nfunc composeCommands() { ink := plaqueInk{Rest: c}; _ = ink }\n",
			"composeCommands is a push button painter outside the kit (builds a plaqueInk)"},
		{"a plaque painter moving its pressed caption", "pkg/ui/a.go",
			"package ui\nfunc composeCommands(down bool) { at = at.Add(plaqueSink) }\n",
			"composeCommands is a push button painter outside the kit (moves a caption by plaqueSink)"},
		{"a plaque face handed to the kit", "pkg/ui/a.go",
			"package ui\nfunc commands() pushButton { return pushButton{Face: &plaqueFace{Ink: plaqueCommandInk, Sink: plaqueSink}} }\n", ""},
		{"a plaque painter picking its pressed picture", "pkg/ui/a.go",
			"package ui\nfunc drawCommands(p *P, i, down int) { copyNative(dst, p.NavButtons[i][down], r.Min, r) }\n",
			"drawCommands is a push button painter outside the kit (picks a button picture by state from NavButtons)"},
		{"a painter reading a button colour", "pkg/ui/a.go",
			"package ui\nfunc paintRows(l Layout) { fill := l.ButtonFill; _ = fill }\n",
			"paintRows is a push button painter outside the kit (reads the button colour ButtonFill)"},
		{"an image twin of a button in a closure", "pkg/ui/a.go",
			"package ui\nfunc composeRows() { paint(func(x, y, w, h int, selected bool) { box(r, rowButtonBorder) }) }\n",
			"composeRows is a push button painter outside the kit (reads the button colour rowButtonBorder in a closure)"},
		{"a button drawn through the kit", "pkg/ui/a.go",
			"package ui\nfunc paintRows(l Layout) { drawPushButton(dst, f, pushButton{Rect: r}); _ = Layout{ButtonFill: c} }\n", ""},
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
	want := len(widgetFrameDebt) + len(widgetNotAFrame) + len(widgetButtonDebt) + len(widgetNotAButton) + len(widgetLatchDebt) + len(widgetNotALatch)
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
