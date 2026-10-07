package mod

import (
	"strings"
	"testing"
)

func settingsOf(m map[string]int64) func(string) (int64, bool) {
	return func(k string) (int64, bool) { v, ok := m[k]; return v, ok }
}

func TestParseSpellsReadsEveryKey(t *testing.T) {
	src := `# a comment
[global]
damage_mul   = [3, 2]
heal_mul     = [1, 1]
mana_mul     = [1, 2]
range_mul    = [5, 4]
radius_mul   = [2, 1]
duration_mul = [3, 1]

[[spell]]
target = "Fire_Ball"
mana = 20
range = 14
radius = 5
duration = 9
area_duration = 12
damage = { min = 7, max = 13 }
effect = { kind = "speed", magnitude = -3, mode = "duration", duration = 40 }
`
	d, err := ParseSpells("m", SpellsFile, []byte(src), nil)
	if err != nil {
		t.Fatal(err)
	}
	g := d.Global
	if !g.Damage.Set || g.Damage.Num != 3 || g.Damage.Den != 2 || g.Damage.Line != 3 {
		t.Errorf("damage_mul %+v", g.Damage)
	}
	if len(g.Globals()) != 6 {
		t.Errorf("globals %+v", g.Globals())
	}
	if len(d.Rows) != 1 {
		t.Fatalf("%d rows", len(d.Rows))
	}
	r := d.Rows[0]
	if r.Mod != "m" || r.File != SpellsFile || r.Line != 10 || r.Target != "Fire_Ball" {
		t.Errorf("row %+v", r)
	}
	if r.Mana.Val != 20 || r.Range.Val != 14 || r.Radius.Val != 5 || r.Duration.Val != 9 || r.AreaLife.Val != 12 ||
		r.DamageMin.Val != 7 || r.DamageMax.Val != 13 || !r.DamageMax.Set {
		t.Errorf("row values %+v", r)
	}
	e := r.Effect
	if e == nil || e.Kind != "speed" || e.Mode != "duration" || e.Magnitude.Val != -3 || e.Duration.Val != 40 {
		t.Errorf("effect %+v", e)
	}
	if d.Empty() {
		t.Error("data reads as empty")
	}
	if !(SpellData{}).Empty() {
		t.Error("zero data does not read as empty")
	}
}

func TestParseSpellsSettingsReachKeys(t *testing.T) {
	src := "[global]\ndamage_mul = [\"@num\", \"@den\"]\n\n[[spell]]\ntarget = \"Heal\"\nmana = \"@mana\"\ndamage = { min = \"@lo\", max = 9 }\n"
	d, err := ParseSpells("m", SpellsFile, []byte(src), settingsOf(map[string]int64{"num": 7, "den": 4, "mana": 11, "lo": 2}))
	if err != nil {
		t.Fatal(err)
	}
	if d.Global.Damage.Num != 7 || d.Global.Damage.Den != 4 || d.Rows[0].Mana.Val != 11 || d.Rows[0].DamageMin.Val != 2 {
		t.Errorf("%+v", d)
	}
}

func TestParseSpellsRefusalsNameFileAndLine(t *testing.T) {
	row := "[[spell]]\ntarget = \"Heal\"\n"
	for name, c := range map[string]struct{ src, want string }{
		"pair outside a table":   {"mana = 1\n", "data/spells.toml:1: mana is outside a [global] or [[spell]] table"},
		"unknown table":          {"[spells]\n", `data/spells.toml:1: unknown table "spells"`},
		"unknown row key":        {row + "armour = 3\n", `data/spells.toml:3: unknown key "armour" in [[spell]]`},
		"unknown global key":     {"[global]\nluck_mul = [1, 1]\n", `data/spells.toml:2: unknown key "luck_mul" in [global]`},
		"no target":              {"[[spell]]\nmana = 3\n", "data/spells.toml:1: [[spell]] has no target"},
		"no change":              {row, `data/spells.toml:1: [[spell]] of "Heal" changes nothing`},
		"empty target":           {"[[spell]]\ntarget = \" \"\n", "data/spells.toml:2: target is empty"},
		"mana text":              {row + "mana = \"lots\"\n", "data/spells.toml:3: mana must be an integer or \"@setting\", not the text \"lots\""},
		"mana high":              {row + "mana = 32768\n", "data/spells.toml:3: mana is 32768, outside 0..32767"},
		"mana negative":          {row + "mana = -1\n", "data/spells.toml:3: mana is -1, outside 0..32767"},
		"range high":             {row + "range = 256\n", "data/spells.toml:3: range is 256, outside 0..255"},
		"radius high":            {row + "radius = 256\n", "data/spells.toml:3: radius is 256, outside 0..255"},
		"duration high":          {row + "duration = 4096\n", "data/spells.toml:3: duration is 4096, outside 0..4095"},
		"area duration high":     {row + "area_duration = 4001\n", "data/spells.toml:3: area_duration is 4001, outside 0..4000"},
		"damage not a table":     {row + "damage = 5\n", "data/spells.toml:3: damage must be { min = N, max = N }"},
		"damage one column":      {row + "damage = { max = 5 }\n", "data/spells.toml:3: damage needs both min and max"},
		"damage reversed":        {row + "damage = { min = 6, max = 5 }\n", "data/spells.toml:3: damage.min 6 is above damage.max 5"},
		"damage zero":            {row + "damage = { min = 0, max = 0 }\n", "data/spells.toml:3: damage is 0 to 0"},
		"damage high":            {row + "damage = { min = 1, max = 256 }\n", "data/spells.toml:3: damage.max is 256, outside 0..255"},
		"damage key":             {row + "damage = { min = 1, max = 2, mean = 3 }\n", `data/spells.toml:3: unknown key "mean" in damage`},
		"effect kind":            {row + "effect = { kind = \"fear\" }\n", `data/spells.toml:3: effect.kind is "fear" (use none, health`},
		"effect mode":            {row + "effect = { mode = \"always\" }\n", `data/spells.toml:3: effect.mode is "always" (use duration`},
		"effect empty":           {row + "effect = { }\n", "data/spells.toml:3: effect changes nothing"},
		"effect magnitude":       {row + "effect = { magnitude = 40000 }\n", "data/spells.toml:3: effect.magnitude is 40000, outside -32768..32767"},
		"effect duration":        {row + "effect = { duration = 5000 }\n", "data/spells.toml:3: effect.duration is 5000, outside 0..4095"},
		"ratio shape":            {"[global]\nmana_mul = 2\n", "data/spells.toml:2: mana_mul must be [numerator, denominator]"},
		"ratio three":            {"[global]\nmana_mul = [1, 2, 3]\n", "data/spells.toml:2: mana_mul must be [numerator, denominator]"},
		"ratio zero denominator": {"[global]\nmana_mul = [1, 0]\n", "data/spells.toml:2: mana_mul denominator is 0, outside 1..1000000"},
		"ratio negative":         {"[global]\nmana_mul = [-1, 2]\n", "data/spells.toml:2: mana_mul numerator is -1, outside 0..1000000"},
		"ratio large":            {"[global]\nmana_mul = [1000001, 1]\n", "data/spells.toml:2: mana_mul numerator is 1000001"},
		"missing setting":        {row + "mana = \"@nope\"\n", `data/spells.toml:3: mana names the setting "nope", which is not an integer setting of this mod`},
		"ratio setting missing":  {"[global]\nmana_mul = [\"@a\", 1]\n", `data/spells.toml:2: mana_mul numerator names the setting "a"`},
		"duplicate key":          {row + "mana = 1\nmana = 2\n", "data/spells.toml:4: mana is given twice"},
		"float":                  {row + "mana = 1.5\n", "data/spells.toml:3: "},
		"setting outside domain": {row + "mana = \"@big\"\n", "data/spells.toml:3: mana is 40000, outside 0..32767"},
		"array of tables global": {"[[global]]\nmana_mul = [1, 1]\n", `data/spells.toml:1: unknown table "global"`},
		"table of spell":         {"[spell]\ntarget = \"Heal\"\n", `data/spells.toml:1: unknown table "spell"`},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte(c.src), settingsOf(map[string]int64{"big": 40000}))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestSpellEffectNamesAreTheDataGrammar(t *testing.T) {
	if k := SpellEffectKinds(); k[0] != "none" || len(k) != 9 {
		t.Errorf("kinds %v", k)
	}
	if m := SpellEffectModes(); len(m) != 4 {
		t.Errorf("modes %v", m)
	}
}

func TestParseSpellsReadsTargetFilterKeys(t *testing.T) {
	src := "[[spell]]\ntarget = \"Heal\"\nheal_hostile = true\n\n[[spell]]\ntarget = \"Fire_Ball\"\nself_cast = \"@on\"\narea_hits = \"not_own\"\n\n[[spell]]\ntarget = \"Wall_of_Fire\"\narea_hits = \"hostile\"\n"
	d, err := ParseSpells("m", SpellsFile, []byte(src), settingsOf(map[string]int64{"on": 1}))
	if err != nil {
		t.Fatal(err)
	}
	if h := d.Rows[0].HealHostile; !h.Set || !h.Val || h.Line != 3 {
		t.Errorf("heal_hostile %+v", h)
	}
	if s := d.Rows[1].SelfCast; !s.Set || !s.Val || s.Line != 7 {
		t.Errorf("self_cast %+v", s)
	}
	if a := d.Rows[1].AreaHits; !a.Set || a.Val != "not_own" || a.Line != 8 {
		t.Errorf("area_hits %+v", a)
	}
	if a := d.Rows[2].AreaHits; a.Val != "hostile" {
		t.Errorf("area_hits %+v", a)
	}
	off, err := ParseSpells("m", SpellsFile, []byte("[[spell]]\ntarget = \"Heal\"\nheal_hostile = false\n"), nil)
	if err != nil || !off.Rows[0].HealHostile.Set || off.Rows[0].HealHostile.Val {
		t.Errorf("false: %v %+v", err, off.Rows)
	}
}

func TestParseSpellsReadsAndBoundsRays(t *testing.T) {
	d, err := ParseSpells("m", SpellsFile, []byte("[[spell]]\ntarget = \"Prismatic_Spray\"\nrays = \"@n\"\n"), settingsOf(map[string]int64{"n": 100}))
	if err != nil || !d.Rows[0].Rays.Set || d.Rows[0].Rays.Val != 100 || d.Rows[0].Rays.Line != 3 {
		t.Fatalf("rays: %v %+v", err, d.Rows)
	}
	row := "[[spell]]\ntarget = \"Prismatic_Spray\"\n"
	for name, c := range map[string]struct{ src, want string }{
		"zero":      {row + "rays = 0\n", "data/spells.toml:3: rays is 0, outside 1..100"},
		"too many":  {row + "rays = 101\n", "data/spells.toml:3: rays is 101, outside 1..100"},
		"text":      {row + "rays = \"many\"\n", "data/spells.toml:3: rays must be an integer"},
		"duplicate": {row + "rays = 3\nrays = 4\n", "data/spells.toml:4: rays is given twice"},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte(c.src), nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}

func TestParseSpellsRefusesBadTargetFilterValues(t *testing.T) {
	row := "[[spell]]\ntarget = \"Heal\"\n"
	for name, c := range map[string]struct{ src, want string }{
		"heal number":     {row + "heal_hostile = 1\n", "data/spells.toml:3: heal_hostile must be true, false or \"@setting\", not an integer"},
		"heal text":       {row + "heal_hostile = \"yes\"\n", "data/spells.toml:3: heal_hostile must be an integer or \"@setting\", not the text \"yes\""},
		"self setting":    {row + "self_cast = \"@two\"\n", "data/spells.toml:3: self_cast is 2 from its setting, which must be 0 or 1"},
		"missing setting": {row + "self_cast = \"@nope\"\n", `data/spells.toml:3: self_cast names the setting "nope"`},
		"area unknown":    {row + "area_hits = \"enemies\"\n", "data/spells.toml:3: area_hits must be one of all, not_own, hostile"},
		"area number":     {row + "area_hits = 2\n", "data/spells.toml:3: area_hits must be one of all, not_own, hostile"},
		"duplicate":       {row + "self_cast = true\nself_cast = false\n", "data/spells.toml:4: self_cast is given twice"},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte(c.src), settingsOf(map[string]int64{"two": 2}))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
}
