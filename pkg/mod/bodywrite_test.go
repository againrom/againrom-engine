package mod

import (
	"slices"
	"strings"
	"testing"
)

const bodyWriteSource = `# my bodies
[[body]]
name       = "hammer2h"
heroes     = "assets/bodies/hammer2h.png"
heroes_l   = "assets/bodies/hammer2h_l.png"
frame      = [64, 80]
origin     = [32, 70]
directions = 8
move       = { frames = 6, ticks = 2 }
attack     = { frames = 3, ticks = [2, 3, 2] }
selection  = [16, 10, 48, 74]
`

func TestBodiesWriterRoundTripsOneBodyThroughItsReader(t *testing.T) {
	read := func(src string) Data {
		var d Data
		if err := readBodies(DataReader{Mod: "m"}, &d, BodiesFile, []byte(src)); err != nil {
			t.Fatal(err)
		}
		return d
	}
	d := read(bodyWriteSource)
	same, err := writeBodies(BodiesFile, []byte(bodyWriteSource), d)
	if err != nil || string(same) != bodyWriteSource {
		t.Fatalf("an unchanged sheet is rewritten: %q, %v", same, err)
	}
	d.Bodies.Bodies[0].Move.Frames, d.Bodies.Bodies[0].Move.Ticks = 4, []int{3, 3, 3, 3}
	d.Bodies.Bodies = append(d.Bodies.Bodies, BodySheet{Name: "club", Heroes: "assets/bodies/c.png", HeroesLight: "assets/bodies/c_l.png",
		Width: 32, Height: 48, OriginX: 16, OriginY: 40, Directions: 5, WeaponLast: true,
		Move: BodyAction{Frames: 2, Ticks: []int{4, 4}}, Attack: BodyAction{Frames: 2, Ticks: []int{1, 5}},
		Idle: BodyAction{Frames: 1, Ticks: []int{9}}})
	out, err := writeBodies(BodiesFile, []byte(bodyWriteSource), d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(out), "# my bodies\n[[body]]\nname       = \"hammer2h\"") || !strings.Contains(string(out), "move       = { frames = 4, ticks = 3 }") {
		t.Fatalf("the kept sheet lost its lines:\n%s", out)
	}
	back := read(string(out))
	if len(back.Bodies.Bodies) != 2 {
		t.Fatalf("sheets %d", len(back.Bodies.Bodies))
	}
	got := back.Bodies.Bodies[1]
	if got.Name != "club" || got.Directions != 5 || !got.WeaponLast || got.Idle.Frames != 1 || !slices.Equal(got.Attack.Ticks, []int{1, 5}) || got.Width != 32 || got.OriginY != 40 {
		t.Fatalf("new sheet read back as %+v", got)
	}
	d.Bodies.Bodies = d.Bodies.Bodies[1:]
	out, err = writeBodies(BodiesFile, out, d)
	if err != nil || strings.Contains(string(out), "hammer2h") || !strings.Contains(string(out), "club") {
		t.Fatalf("removal: %q, %v", out, err)
	}
}

func TestWeaponBodiesWriterRoundTripsOneRowThroughItsReader(t *testing.T) {
	src := "# choices\n[[weapon]]\nweapon = \"War Hammer\"\nbody   = \"clubman\"\n"
	var d Data
	if err := readWeaponBodies(DataReader{Mod: "m"}, &d, WeaponBodiesFile, []byte(src)); err != nil {
		t.Fatal(err)
	}
	if same, err := writeWeaponBodies(WeaponBodiesFile, []byte(src), d); err != nil || string(same) != src {
		t.Fatalf("an unchanged row is rewritten: %q, %v", same, err)
	}
	d.Bodies.Weapons[0].Body = "hammer2h"
	d.Bodies.Weapons = append(d.Bodies.Weapons, WeaponBody{Row: 7, Body: "club"})
	out, err := writeWeaponBodies(WeaponBodiesFile, []byte(src), d)
	if err != nil || !strings.HasPrefix(string(out), "# choices\n[[weapon]]\nweapon = \"War Hammer\"\nbody   = \"hammer2h\"") {
		t.Fatalf("%q, %v", out, err)
	}
	back, err := ParseWeaponBodies("m", WeaponBodiesFile, out)
	if err != nil || len(back) != 2 || back[0].Body != "hammer2h" || back[1].Row != 7 || back[1].Body != "club" {
		t.Fatalf("read back %+v, %v", back, err)
	}
	d.Bodies.Weapons = nil
	out, err = writeWeaponBodies(WeaponBodiesFile, out, d)
	if err != nil || strings.Contains(string(out), "[[weapon]]") {
		t.Fatalf("emptied: %q, %v", out, err)
	}
}
