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

// A room data literal is the type the compiler gives it, whatever its spelling.
func TestCheckRoomLiteralsNamesEachShape(t *testing.T) {
	const uiSource = `package ui
type RoomLayer struct{ Kind int }
type RoomSchool struct{ Class int }
type RoomHit struct{ Enabled bool }
type RoomPanel struct{ TrackPointer bool }
type Other struct{ Enabled bool }
type HitAlias = RoomHit
`
	for _, tc := range []struct {
		name, file, src, want string
	}{
		{"room layer", "pkg/game/a.go", `var x = ui.RoomLayer{Kind: 0}`, "constructs RoomLayer"},
		{"school layer", "pkg/game/a.go", `var x = ui.RoomSchool{Class: 0}`, "constructs RoomSchool"},
		{"control area", "pkg/game/a.go", `var x = ui.RoomHit{Enabled: true}`, "constructs RoomHit"},
		{"room panel", "pkg/game/a.go", `var x = ui.RoomPanel{TrackPointer: true}`, "constructs RoomPanel"},
		{"an elided element type", "pkg/game/a.go", `var x = []ui.RoomHit{{Enabled: true}}`, "constructs RoomHit"},
		{"an elided map value", "pkg/game/a.go", `var x = map[string]ui.RoomLayer{"a": {Kind: 1}}`, "constructs RoomLayer"},
		{"a pointer element", "pkg/game/a.go", `var x = []*ui.RoomHit{{Enabled: true}}`, "constructs RoomHit"},
		{"an alias", "pkg/game/a.go", `var x = ui.HitAlias{Enabled: true}`, "constructs RoomHit"},
		{"the builder", roomPageBuilder, `var x = []ui.RoomHit{{Enabled: true}}`, ""},
		{"the painter", roomPagePainter, `var x = []ui.RoomHit{{Enabled: true}}`, ""},
		{"an empty literal", "pkg/game/a.go", `var x = ui.RoomHit{}`, ""},
		{"another type with the same field", "pkg/game/a.go", `var x = []ui.Other{{Enabled: true}}`, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pkgs := []fixturePackage{{path: "againrom/pkg/ui", files: map[string]string{"pkg/ui/types.go": uiSource}}}
			if tc.file == roomPagePainter {
				pkgs[0].files[tc.file] = "package ui\n" + strings.ReplaceAll(tc.src, "ui.", "") + "\n"
			} else {
				pkgs = append(pkgs, fixturePackage{path: "againrom/pkg/game", files: map[string]string{
					tc.file: "package game\nimport \"againrom/pkg/ui\"\n" + tc.src + "\n"}})
			}
			vs := CheckRoomLiterals(checkedFixture(t, pkgs...))
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

func TestRoomLiteralsLiveTree(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root, err := FindModuleRoot(wd)
	if err != nil {
		t.Fatal(err)
	}
	module, err := loadTypeCheckedModule(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range CheckRoomLiterals(module) {
		t.Error(v)
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
