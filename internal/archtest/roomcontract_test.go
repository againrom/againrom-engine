package archtest

import (
	"os"
	"strings"
	"testing"
)

func TestCheckRoomContractNamesEachShape(t *testing.T) {
	for _, tc := range []struct{ name, file, src, want string }{
		{"surface painter", "pkg/ui/a.go", `package ui; func ComposeTownSurface(v V) I { paint(v); return ComposeTownRoom(v) }`, "ComposeTownSurface must only return"},
		{"square painter", "pkg/ui/a.go", `package ui; func ComposeTownSquare(v V) I { return another(v) }`, "ComposeTownSquare must only return"},
		{"surface hit test", "pkg/ui/a.go", `package ui; func TownSurfaceControlAt(v V,p P) (C,bool) { if p.In(r) {} ; return v.SurfaceControlAt(p) }`, "TownSurfaceControlAt must only return"},
		{"panel builder", "pkg/ui/a.go", `package ui; func buildButtonPanel(c C,a A,b B) P { for range b {} ; return p.build() }`, "buildButtonPanel must only return"},
		{"retired panel", "pkg/ui/a.go", `package ui; func townSurfacePanel(v V) P { return p }`, "retired townSurfacePanel"},
		{"parallel view", "pkg/ui/a.go", `package ui; type TownSurfaceView struct { Page P }`, "TownSurfaceView must alias"},
		{"parallel square view", "pkg/ui/a.go", `package ui; type TownSquareView struct { Scene S }`, "TownSquareView must alias"},
		{"room layer", "pkg/game/a.go", `package game; var x=ui.RoomLayer{Kind:0}`, "second room data builder constructs RoomLayer"},
		{"school layer", "pkg/game/a.go", `package game; var x=ui.RoomSchool{Class:0}`, "second room data builder constructs RoomSchool"},
		{"control area", "pkg/game/a.go", `package game; var x=ui.RoomHit{Enabled:true}`, "second room data builder constructs RoomHit"},
		{"room panel", "pkg/game/a.go", `package game; var x=ui.RoomPanel{TrackPointer:true}`, "second room data builder constructs RoomPanel"},
		{"second frame composer", "pkg/ui/a.go", `package ui; func ComposeOther(v V) { DrawRoomPage(dst,v.Page) }`, "second room frame composer"},
		{"adapter hit test", "pkg/game/secondtown.go", `package game; func (v roomPageView) ControlAt(p P) { p.In(r) }`, "hit-tests room geometry outside"},
		{"game fork", roomPageBuilder, `package game; func f() { if g==GameROM1 {} }`, "game identity chooses"},
		{"forward", "pkg/ui/a.go", `package ui; type TownSurfaceView = TownRoomView; func ComposeTownSurface(v V) I { return ComposeTownRoom(v) }`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			vs := CheckRoomContract(map[string]string{tc.file: tc.src})
			if tc.want == "" {
				if len(vs) != 0 {
					t.Fatalf("unexpected: %v", vs)
				}
			} else if len(vs) != 1 || !strings.Contains(vs[0].Reason, tc.want) {
				t.Fatalf("got %v, want one naming %q", vs, tc.want)
			}
		})
	}
}

func TestRoomContractLiveTree(t *testing.T) {
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
	for _, v := range CheckRoomContract(files) {
		t.Error(v)
	}
}
