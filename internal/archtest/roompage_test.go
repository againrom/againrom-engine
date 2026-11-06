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

func TestCheckRoomPageNamesAlternateShapes(t *testing.T) {
	cases := []struct{ file, source, want string }{
		{"pkg/ui/townshell.go", `package ui; import "image"; func innButton() image.Rectangle { return image.Rect(1,2,3,4) }`, "Go rectangle for room control in innButton"},
		{"pkg/ui/townshell.go", `package ui; import img "image"; var innControls=[]img.Rectangle{img.Rect(1,2,3,4)}`, "Go rectangle for room control innControls"},
		{"pkg/ui/townshell.go", `package ui; import "image"; var innControl=image.Rectangle{Min:image.Point{1,2},Max:image.Point{3,4}}`, "Go rectangle for room control innControl"},
		{"pkg/ui/townshell.go", `package ui; func townSurfacePanel(){ local:=image.Rect(1,2,3,4); _=local }`, "Go rectangle for room control in townSurfacePanel"},
		{"pkg/ui/townshell.go", `package ui; func drawCharacterPaneBody(){ a:=image.Rect(1,2,3,4); b:=image.Rect(5,6,7,8); _,_=a,b }`, "Go rectangle for room control in drawCharacterPaneBody"},
		{"pkg/ui/townshell.go", `package ui; func drawSelectionStatus(){ a:=image.Rect(1,2,3,4); _=a }`, "Go rectangle for room control in drawSelectionStatus"},
		{"pkg/game/other.go", `package game; type copied struct { p *town.Scene }; func (v copied) Paint(){}`, "second room page adapter copied"},
		{"pkg/game/other.go", `package game; type copied struct {}; func (v copied) Paint(){}; func (v copied) ControlAt(){}`, "second room page adapter copied"},
		{"pkg/game/other.go", `package game; func f(){ _=ui.RoomSeat{Frame:1} }`, "second room page builder constructs RoomSeat"},
		{"pkg/game/other.go", `package game; func f(){ _=ui.RoomPage{Phase:1} }`, "second room page builder constructs RoomPage"},
		{"pkg/game/other.go", `package game; func f(){ _=roomPageView{room:"inn"} }`, "second room page reader"},
		{"pkg/game/townshell.go", `package game; func patron(){ _=partyPanelSubject(m,t,w) }`, "second room person builder"},
		{"pkg/game/taverntalk.go", `package game; func patron(){ _=composeMemberPortrait(s,u,i,e,m) }`, "second room person builder"},
		{"pkg/game/secondtown.go", `package game; func f(){ if p.Game==base.GameROM2 { draw() } }`, "game identity chooses a room page path"},
	}
	for _, tc := range cases {
		var got []string
		for _, v := range CheckRoomPage(map[string]string{tc.file: tc.source}) {
			got = append(got, v.From+": "+v.Reason)
		}
		if !strings.Contains(strings.Join(got, "\n"), tc.want) {
			t.Errorf("%s: got %v, want %s", tc.file, got, tc.want)
		}
	}
}

// TestRoomShellBudgetsAreExact lowers each admitted painter's budget by one on
// the live source: every budget must then be refused, so none admits a
// rectangle the painter does not use.
func TestRoomShellBudgetsAreExact(t *testing.T) {
	raw, err := os.ReadFile("../../pkg/ui/townshell.go")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"pkg/ui/townshell.go": string(raw)}
	if vs := CheckRoomPage(files); len(vs) != 0 {
		t.Fatalf("live source: %v", vs)
	}
	saved := map[string]int{}
	for k, v := range roomShellGeometry {
		saved[k] = v
	}
	defer func() {
		for k, v := range saved {
			roomShellGeometry[k] = v
		}
	}()
	for k, v := range saved {
		roomShellGeometry[k] = v - 1
		named := false
		for _, x := range CheckRoomPage(files) {
			named = named || strings.Contains(x.Reason, k)
		}
		roomShellGeometry[k] = v
		if !named {
			t.Errorf("budget %s=%d is larger than its use", k, v)
		}
	}
}
