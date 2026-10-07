package mod

import (
	"strings"
	"testing"
)

func TestParseSpellsReadsFormulaTables(t *testing.T) {
	src := `[global]
power = [0, 5, "@top"]
damage_factor = [30, 60]
range_bonus = [1]
duration_factor = [1000, 2000]

[[spell]]
target = "Heal"
power = [10]
damage_factor = [90]
range_bonus = [4, 5]
duration_factor = [1500]
magnitude = [-3, "@top"]
`
	d, err := ParseSpells("m", SpellsFile, []byte(src), settingsOf(map[string]int64{"top": 200}))
	if err != nil {
		t.Fatal(err)
	}
	g := d.Global
	if !g.Power.Set || g.Power.Line != 2 || len(g.Power.Vals) != 3 || g.Power.Vals[2] != 200 {
		t.Errorf("global power %+v", g.Power)
	}
	if !g.DamageFactor.Set || !g.RangeBonus.Set || !g.DurationFactor.Set || g.DurationFactor.Line != 5 {
		t.Errorf("globals %+v", g)
	}
	keys := g.Globals()
	if len(keys) != 4 || keys[0].Key != "power" || keys[0].Line != 2 || keys[3].Key != "duration_factor" {
		t.Errorf("global keys %+v", keys)
	}
	r := d.Rows[0]
	f := r.Formulas()
	if !f[0].Set || f[0].Vals[0] != 10 || !f[1].Set || !f[2].Set || len(f[2].Vals) != 2 || !f[3].Set || !f[4].Set || f[4].Vals[1] != 200 || f[4].Line != 13 {
		t.Errorf("row formulas %+v", f)
	}
	if d.Empty() || (SpellData{Global: SpellGlobal{Power: g.Power}}).Empty() {
		t.Error("data with tables reads as empty")
	}
	if !(SpellData{}).Empty() {
		t.Error("no data is not empty")
	}
}

func TestSpellFormulaTablesRefuseWithTheKeyLine(t *testing.T) {
	long := func(n int) string { return strings.TrimSuffix(strings.Repeat("1, ", n), ", ") }
	for name, c := range map[string]struct {
		src, want string
	}{
		"not an array":     {"[[spell]]\ntarget = \"Heal\"\nrange_bonus = 3\n", "spells.toml:3: range_bonus must be an array of integers, not an integer"},
		"empty":            {"[[spell]]\ntarget = \"Heal\"\npower = []\n", "power has 0 entries, want 1..511"},
		"too long":         {"[[spell]]\ntarget = \"Heal\"\ndamage_factor = [" + long(257) + "]\n", "damage_factor has 257 entries, want 1..256"},
		"power too long":   {"[[spell]]\ntarget = \"Heal\"\npower = [" + long(512) + "]\n", "power has 512 entries, want 1..511"},
		"power too big":    {"[[spell]]\ntarget = \"Heal\"\npower = [1, 256]\n", "spells.toml:3: power entry 1 is 256, outside 0..255"},
		"range negative":   {"[[spell]]\ntarget = \"Heal\"\nrange_bonus = [0, -1]\n", "range_bonus entry 1 is -1, outside 0..255"},
		"damage too big":   {"[[spell]]\ntarget = \"Heal\"\ndamage_factor = [30001]\n", "outside 0..30000"},
		"duration big":     {"[[spell]]\ntarget = \"Heal\"\nduration_factor = [1000001]\n", "outside 0..1000000"},
		"magnitude low":    {"[[spell]]\ntarget = \"Heal\"\nmagnitude = [-32769]\n", "magnitude entry 0 is -32769, outside -32768..32767"},
		"text entry":       {"[[spell]]\ntarget = \"Heal\"\npower = [\"high\"]\n", "power entry 0 must be an integer or \"@setting\", not the text \"high\""},
		"missing setting":  {"[[spell]]\ntarget = \"Heal\"\npower = [\"@nope\"]\n", "power entry 0 names the setting \"nope\""},
		"nested array":     {"[[spell]]\ntarget = \"Heal\"\npower = [[1]]\n", "power entry 0 must be an integer"},
		"global magnitude": {"[global]\nmagnitude = [1]\n", "spells.toml:2: magnitude has no [global] table"},
		"global bad":       {"[global]\ndamage_factor = [-1]\n", "spells.toml:2: damage_factor entry 0 is -1"},
		"global unknown":   {"[global]\npower_mul = [1, 1]\n", `unknown key "power_mul" in [global]`},
	} {
		_, err := ParseSpells("m", SpellsFile, []byte(c.src), nil)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: %v, want %q", name, err, c.want)
		}
	}
	for _, key := range SpellFieldNames() {
		if key == "magnitude" {
			return
		}
	}
	t.Error("the unknown-key list omits the formula keys")
}

func TestSpellFormulaTableAtItsLargestSizes(t *testing.T) {
	long := func(n int, v string) string { return strings.TrimSuffix(strings.Repeat(v+", ", n), ", ") }
	src := "[[spell]]\ntarget = \"Heal\"\npower = [" + long(511, "255") + "]\ndamage_factor = [" + long(256, "30000") + "]\n"
	d, err := ParseSpells("m", SpellsFile, []byte(src), nil)
	if err != nil || len(d.Rows[0].Power.Vals) != 511 || len(d.Rows[0].DamageFactor.Vals) != 256 {
		t.Fatalf("%v", err)
	}
}
