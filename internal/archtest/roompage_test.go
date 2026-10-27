package archtest

import (
	"os"
	"strings"
	"testing"
)

// TestCheckRoomPageNamesASecondLayout drives the scan with synthetic sources.
func TestCheckRoomPageNamesASecondLayout(t *testing.T) {
	cases := []struct {
		name, file, src, want string
	}{
		{"a second seat layout", "pkg/game/a.go",
			"package game\nfunc f(r *R) { _ = r.Seats.Seat(1) }\n", "reads Seats.Seat"},
		{"a second card placement", "pkg/game/a.go",
			"package game\nfunc f(r *R) { _ = r.Seats.Card.Rect }\n", "reads Seats.Card"},
		{"the builder", roomPageBuilder,
			"package game\nfunc f(r *R) { _ = r.Seats.Seat(1); _ = r.Seats.Figure }\n", ""},
		{"a seat clock", "pkg/game/a.go",
			"package game\nfunc f(r *R) { _ = r.Seats.FrameMS }\n", ""},
		{"a number in the builder", roomPageBuilder,
			"package game\nimport \"image\"\nfunc f() image.Point { return image.Pt(12, 0) }\n", "image.Pt with a number"},
		{"a number in the painter", roomPagePainter,
			"package ui\nimport \"image\"\nfunc f() image.Rectangle { return image.Rect(0, 238, 160, 480) }\n", "image.Rect with a number"},
		{"data in the builder", roomPageBuilder,
			"package game\nimport \"image\"\nfunc f(x, y int) image.Point { return image.Pt(x, y) }\n", ""},
		{"a number elsewhere", "pkg/ui/b.go",
			"package ui\nimport \"image\"\nfunc f() image.Point { return image.Pt(12, 0) }\n", ""},
		{"a test", "pkg/game/a_test.go",
			"package game\nfunc f(r *R) { _ = r.Seats.Seat(1) }\n", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []string
			for _, v := range CheckRoomPage(map[string]string{tc.file: tc.src}) {
				got = append(got, v.Reason)
			}
			if tc.want == "" {
				if len(got) != 0 {
					t.Fatalf("CheckRoomPage = %v, want none", got)
				}
				return
			}
			if len(got) == 0 || !strings.Contains(strings.Join(got, "\n"), tc.want) {
				t.Fatalf("CheckRoomPage = %v, want one naming %q", got, tc.want)
			}
		})
	}
}

// TestRoomPageHasOneBuilder is the live-tree half.
func TestRoomPageHasOneBuilder(t *testing.T) {
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
	if !strings.Contains(files[roomPageBuilder], "func buildRoomPage(") || !strings.Contains(files[roomPagePainter], "func DrawRoomPage(") {
		t.Fatalf("%s or %s lost its builder; the scan guards nothing", roomPageBuilder, roomPagePainter)
	}
	for _, v := range CheckRoomPage(files) {
		t.Errorf("%s: %s", v.From, v.Reason)
	}
}
