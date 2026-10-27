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
		{"an aliased seat grid", "pkg/game/a.go",
			"package game\nfunc f(r *R) { seats := r.Seats; _ = seats.Seat(1) }\n", "name bound to Seats"},
		{"an aliased card placement", "pkg/game/a.go",
			"package game\nfunc f(r *R) { var seats = r.Seats; _ = seats.Card }\n", "name bound to Seats"},
		{"an alias of an alias", "pkg/game/a.go",
			"package game\nfunc f(r *R) { a := r.Seats; b := a; _ = b.Figure }\n", "name bound to Seats"},
		{"a seat spec parameter", "pkg/game/a.go",
			"package game\nfunc f(s *town.SeatSpec) { _ = s.Count() }\n", "name bound to Seats"},
		{"an aliased seat clock", "pkg/game/a.go",
			"package game\nfunc f(r *R, s *town.SeatSpec) { seats := r.Seats; _ = seats.Actor; _ = s.FrameMS }\n", ""},
		{"the builder through an alias", roomPageBuilder,
			"package game\nfunc f(r *R) { seats := r.Seats; _ = seats.Seat(1) }\n", ""},
		{"a second reader of a seat rectangle", "pkg/ui/a.go",
			"package ui\nfunc f(v V) { _ = v.Page.Seats[0].Rect }\n", "Rect of a built page's seat"},
		{"a seat rectangle by range", "pkg/game/a.go",
			"package game\nfunc f(p P) { for _, s := range p.Seats { _ = s.Rect } }\n", "Rect of a built page's seat"},
		{"a seat rectangle by pointer", "pkg/game/a.go",
			"package game\nfunc f(p P) { s := &p.Seats[0]; _ = s.Rect }\n", "Rect of a built page's seat"},
		{"a seat rectangle through an alias", "pkg/game/a.go",
			"package game\nfunc f(p P) { seats := p.Seats[:2]; seat := seats[1]; _ = seat.Rect }\n", "Rect of a built page's seat"},
		{"a second painter", "pkg/game/a.go",
			"package game\nfunc f(p P, dst D) { for i := range p.Seats { draw.Draw(dst, p.Seats[i].Rect, nil, pt, draw.Src) } }\n", "Rect of a built page's seat"},
		{"the painter reads a rectangle", roomPagePainter,
			"package ui\nfunc (p RoomPage) f(pt P) { for i, s := range p.Seats { _ = s.Rect; _ = i } }\n", ""},
		{"the builder writes a rectangle", roomPageBuilder,
			"package game\nfunc f(p *P, r R) { p.Seats = append(p.Seats, S{Rect: r}) }\n", ""},
		{"a seat field that is no rectangle", "pkg/ui/a.go",
			"package ui\nfunc f(p P) { for _, s := range p.Seats { _ = s.Hidden } }\n", ""},
		{"a rectangle of another list", "pkg/ui/a.go",
			"package ui\nfunc f(p P) { _ = p.Buttons[0].Rect; for _, b := range p.Buttons { _ = b.Rect } }\n", ""},
		{"a rectangle read in a test", "pkg/ui/a_test.go",
			"package ui\nfunc f(v V) { _ = v.Page.Seats[0].Rect }\n", ""},
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
