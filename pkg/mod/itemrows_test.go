package mod

import (
	"strings"
	"testing"
)

const goodMaterialsAndWeapons = `[[material]]
item     = "Spear"
tier     = "Common"
material = "Meteoric"
allow    = true
sprite   = "assets/sprites/meteoric_spear.png"
name     = "item.meteoric_spear"

[[material]]
item     = "Spear"
tier     = "Common"
material = "Iron"
allow    = false

[[weapon]]
key        = "star_spear"
name       = "item.star_spear"
base       = "Spear"
damage-min = 9
damage-max = 14
range      = 3
price      = 700
`

func TestParseItemsReadsMaterialsAndWeaponRows(t *testing.T) {
	d, err := ParseItems("m", ItemsFile, []byte(goodMaterialsAndWeapons),
		textOf(map[string]string{"item.meteoric_spear": "Star Iron Spear", "item.star_spear": "Star Spear"}), "en")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Materials) != 2 || len(d.Weapons) != 1 || d.Empty() {
		t.Fatalf("materials %d weapons %d", len(d.Materials), len(d.Weapons))
	}
	m := d.Materials[0]
	if m.Item != "Spear" || m.Tier != "Common" || m.Material != "Meteoric" || !m.Allow || m.AllowLine != 5 ||
		m.Sprite != "assets/sprites/meteoric_spear.png" || m.Name != "Star Iron Spear" || m.Line != 1 {
		t.Fatalf("%+v", m)
	}
	if f := d.Materials[1]; f.Allow || f.Material != "Iron" || f.Line != 9 {
		t.Fatalf("%+v", f)
	}
	w := d.Weapons[0]
	if w.Key != "star_spear" || w.Name != "Star Spear" || w.Base != "Spear" || w.BaseLine != 18 || w.Line != 15 {
		t.Fatalf("%+v", w)
	}
	want := map[int]int32{6: 9, 7: 14, 11: 3, 2: 700}
	if len(w.Params) != len(want) {
		t.Fatalf("params %+v", w.Params)
	}
	for _, p := range w.Params {
		if want[p.Column] != p.Value {
			t.Fatalf("column %d = %d", p.Column, p.Value)
		}
	}
}

func TestParseItemsRefusesBadMaterialsAndWeaponRows(t *testing.T) {
	text := textOf(map[string]string{"n": "Name"})
	for _, c := range []struct{ src, want string }{
		{"[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nmaterial = \"Iron\"\n", "data/items.toml:1: [[material]] has no allow"},
		{"[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nallow = true\n", "data/items.toml:1: [[material]] has no material"},
		{"[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nmaterial = \"Iron\"\nallow = \"yes\"\n", "data/items.toml:5: allow must be true or false"},
		{"[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nmaterial = \"Iron\"\nallow = false\nsprite = \"a.png\"\n", "data/items.toml:5: a forbidden material takes no sprite and no name"},
		{"[[material]]\nitem = \"Spear\"\ntier = \"Common\"\nmaterial = \"Iron\"\nallow = true\ncolour = 3\n", `data/items.toml:6: unknown key "colour" in [[material]]`},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\nattack-type = 2\n", `data/items.toml:5: "attack-type" is not a weapon parameter a mod may set`},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\ncharge = 2\n", `data/items.toml:5: "charge" is not a weapon parameter a mod may set`},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nrange = 2\n", "data/items.toml:1: [[weapon]] has no base"},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\nrange = 0\n", "data/items.toml:5: range is 0, outside 1..255"},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\nrange = \"far\"\n", "data/items.toml:5: range must be an integer"},
		{"[[weapon]]\nkey = \"W\"\nname = \"n\"\nbase = \"Spear\"\n", `data/items.toml:2: key "W" must be`},
		{"[[weapon]]\nkey = \"w\"\nname = \"x\"\nbase = \"Spear\"\n", `data/items.toml:3: text key "x"`},
		{"[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\n[[weapon]]\nkey = \"w\"\nname = \"n\"\nbase = \"Spear\"\n", `data/items.toml:5: key "w" is already used at line 1`},
	} {
		_, err := ParseItems("m", ItemsFile, []byte(c.src), text, "en")
		if err == nil || !strings.HasPrefix(err.Error(), c.want) {
			t.Errorf("%q: got %v, want prefix %q", c.src, err, c.want)
		}
	}
}

func TestWeaponParamsAreTheClaimedColumns(t *testing.T) {
	want := map[string]int{"price": 2, "weight": 3, "damage-min": 6, "damage-max": 7, "to-hit": 8, "defence": 9, "range": 11}
	if len(WeaponParams) != len(want) {
		t.Fatalf("%d settable parameters", len(WeaponParams))
	}
	for _, p := range WeaponParams {
		if want[p.Key] != p.Column {
			t.Errorf("%s is column %d", p.Key, p.Column)
		}
		for _, unknown := range []int{0, 1, 4, 10, 16} {
			if p.Column == unknown {
				t.Errorf("%s sets column %d, whose reader is Unknown (ITEM-219)", p.Key, unknown)
			}
		}
	}
}
