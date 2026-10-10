package mod

import (
	"slices"
	"strings"
	"testing"
)

const goodWeaponBody = `# the war hammer as a mace
[[weapon]]
weapon = "War Hammer"
row    = 19
body   = "clubman"
`

const goodBody = `[[body]]
name       = "clubman2h"
heroes     = "bodies/clubman2h-heroes.png"
heroes_l   = "bodies/clubman2h-heroes_l.png"
frame      = [96, 80]
origin     = [48, 70]
directions = 8
move       = { frames = 8, ticks = 1 }
attack     = { frames = 3, ticks = [2, 3, 4] }
idle       = { frames = 2, ticks = 6 }
weapon-last = true
selection  = [30, 10, 66, 75]
`

func TestParseWeaponBodiesReadsAMapping(t *testing.T) {
	got, err := ParseWeaponBodies("m", WeaponBodiesFile, []byte(goodWeaponBody))
	if err != nil {
		t.Fatal(err)
	}
	want := WeaponBody{Mod: "m", File: WeaponBodiesFile, Line: 2, Weapon: "War Hammer", WeaponLine: 3, Row: 19, RowLine: 4, Body: "clubman", BodyLine: 5}
	if len(got) != 1 || got[0] != want {
		t.Fatalf("%+v, want %+v", got, want)
	}
	byRow, err := ParseWeaponBodies("m", WeaponBodiesFile, []byte("[[weapon]]\nrow = 7\nbody = \"axeman\"\n"))
	if err != nil || len(byRow) != 1 || byRow[0].Row != 7 || byRow[0].Weapon != "" {
		t.Fatalf("a row alone: %+v %v", byRow, err)
	}
}

func TestParseWeaponBodiesRefusalsNameFileAndLine(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"outside a table", "body = \"clubman\"\n", "data/weapon-bodies.toml:1: body is outside a [[weapon]] table"},
		{"unknown table", "[[mace]]\n", `data/weapon-bodies.toml:1: unknown table "mace"`},
		{"unknown key", goodWeaponBody + "shield = 1\n", `data/weapon-bodies.toml:6: unknown key "shield" in [[weapon]]`},
		{"empty weapon", strings.Replace(goodWeaponBody, `"War Hammer"`, `""`, 1), "data/weapon-bodies.toml:3: weapon must be the name of a weapon table row"},
		{"row zero", strings.Replace(goodWeaponBody, "row    = 19", "row    = 0", 1), "data/weapon-bodies.toml:4: row is 0; a weapon row is 1 to 31"},
		{"row text", strings.Replace(goodWeaponBody, "row    = 19", `row    = "19"`, 1), "data/weapon-bodies.toml:4: row must be an integer, not a string"},
		{"bad body name", strings.Replace(goodWeaponBody, `"clubman"`, `"Club Man"`, 1), `data/weapon-bodies.toml:5: body "Club Man" is not a body name`},
		{"no body", strings.Replace(goodWeaponBody, "body   = \"clubman\"\n", "", 1), "data/weapon-bodies.toml:2: [[weapon]] has no body"},
		{"no weapon", "[[weapon]]\nbody = \"clubman\"\n", "data/weapon-bodies.toml:1: [[weapon]] names no weapon"},
	}
	for _, c := range cases {
		_, err := ParseWeaponBodies("m", WeaponBodiesFile, []byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want %q", c.name, err, c.want)
		}
	}
}

func TestParseBodiesReadsASheet(t *testing.T) {
	got, err := ParseBodies("m", BodiesFile, []byte(goodBody))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%+v", got)
	}
	b := got[0]
	if b.Name != "clubman2h" || b.Heroes != "bodies/clubman2h-heroes.png" || b.HeroesLight != "bodies/clubman2h-heroes_l.png" ||
		b.Width != 96 || b.Height != 80 || b.OriginX != 48 || b.OriginY != 70 || b.Directions != 8 || !b.WeaponLast ||
		!b.HasSelection || b.Selection != [4]int{30, 10, 66, 75} || b.Line != 1 || b.StandingFrames() != 16 {
		t.Fatalf("%+v", b)
	}
	if b.Move.Frames != 8 || !slices.Equal(b.Move.Ticks, []int{1, 1, 1, 1, 1, 1, 1, 1}) ||
		b.Attack.Frames != 3 || !slices.Equal(b.Attack.Ticks, []int{2, 3, 4}) ||
		b.Idle.Frames != 2 || !slices.Equal(b.Idle.Ticks, []int{6, 6}) {
		t.Fatalf("actions %+v %+v %+v", b.Move, b.Attack, b.Idle)
	}
	five := strings.Replace(goodBody, "directions = 8", "directions = 5", 1)
	five = strings.Replace(five, "idle       = { frames = 2, ticks = 6 }\n", "", 1)
	got, err = ParseBodies("m", BodiesFile, []byte(five))
	if err != nil || got[0].StandingFrames() != 9 || got[0].Idle.Frames != 0 {
		t.Fatalf("five directions, no idle: %+v %v", got, err)
	}
}

func TestParseBodiesRefusalsNameFileAndLine(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"outside a table", "name = \"x\"\n", "data/bodies.toml:1: name is outside a [[body]] table"},
		{"unknown key", goodBody + "colour = 1\n", `data/bodies.toml:13: unknown key "colour" in [[body]]`},
		{"bad name", strings.Replace(goodBody, `"clubman2h"`, `"2h"`, 1), `data/bodies.toml:2: name "2h" is not a body name`},
		{"not png", strings.Replace(goodBody, "clubman2h-heroes.png", "clubman2h-heroes.bmp", 1), `data/bodies.toml:3: heroes "bodies/clubman2h-heroes.bmp" is not a .png file`},
		{"outside the folder", strings.Replace(goodBody, "bodies/clubman2h-heroes_l.png", "../x.png", 1), `data/bodies.toml:4: heroes_l "../x.png" is not a clean path inside the mod folder`},
		{"frame shape", strings.Replace(goodBody, "[96, 80]", "[96]", 1), "data/bodies.toml:5: frame must be an array of 2 integers"},
		{"frame zero", strings.Replace(goodBody, "[96, 80]", "[0, 80]", 1), "data/bodies.toml:5: frame is 0x80"},
		{"frame large", strings.Replace(goodBody, "[96, 80]", "[96, 900]", 1), "data/bodies.toml:5: frame must hold a width and a height of 0 to 512"},
		{"origin outside", strings.Replace(goodBody, "[48, 70]", "[48, 81]", 1), "data/bodies.toml:6: origin (48, 81) is outside the 96x80 frame"},
		{"directions", strings.Replace(goodBody, "directions = 8", "directions = 4", 1), "data/bodies.toml:7: directions must be 8, or 5"},
		{"move not a table", strings.Replace(goodBody, "{ frames = 8, ticks = 1 }", "8", 1), "data/bodies.toml:8: move must be an inline table"},
		{"move no frames", strings.Replace(goodBody, "{ frames = 8, ticks = 1 }", "{ ticks = 1 }", 1), "data/bodies.toml:8: move has no frames"},
		{"attack zero frames", strings.Replace(goodBody, "frames = 3, ticks = [2, 3, 4]", "frames = 0, ticks = 1", 1), "data/bodies.toml:9: attack.frames must be an integer of 1 to 64"},
		{"ticks count", strings.Replace(goodBody, "[2, 3, 4]", "[2, 3]", 1), "data/bodies.toml:9: attack.ticks holds 2 counts for 3 frames"},
		{"ticks zero", strings.Replace(goodBody, "ticks = 6", "ticks = 0", 1), "data/bodies.toml:10: idle.ticks must hold integers of 1 to 255"},
		{"no ticks", strings.Replace(goodBody, "frames = 2, ticks = 6", "frames = 2", 1), "data/bodies.toml:10: idle has no ticks"},
		{"action key", strings.Replace(goodBody, "frames = 2, ticks = 6", "frames = 2, ticks = 6, loop = true", 1), `data/bodies.toml:10: unknown key "loop" in idle`},
		{"weapon-last", strings.Replace(goodBody, "weapon-last = true", "weapon-last = 1", 1), "data/bodies.toml:11: weapon-last must be true or false"},
		{"selection outside", strings.Replace(goodBody, "[30, 10, 66, 75]", "[30, 10, 66, 90]", 1), "data/bodies.toml:12: selection [30 10 66 90] is not a box inside the 96x80 frame"},
		{"missing attack", strings.Replace(goodBody, "attack     = { frames = 3, ticks = [2, 3, 4] }\n", "", 1), "data/bodies.toml:1: [[body]] has no attack"},
		{"twice", goodBody + goodBody, `data/bodies.toml:13: body "clubman2h" is already supplied at line 1`},
	}
	for _, c := range cases {
		_, err := ParseBodies("m", BodiesFile, []byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err %v, want %q", c.name, err, c.want)
		}
	}
}
