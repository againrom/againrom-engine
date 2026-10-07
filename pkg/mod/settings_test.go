package mod

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleSettings = `# knobs
[skill_cap]
label   = { en = "Skill cap", ru = "Предел навыка" }
type    = "int"
default = 150
min     = 100
max     = 150

[cloaks_for_mages]
label   = "Cloaks for mages too"
type    = "bool"
default = false

[difficulty]
label   = { en = "Mood" }
type    = "choice"
choices = ["calm", "grim"]
default = "calm"
`

func TestParseSettingsReadsEveryKind(t *testing.T) {
	got, err := ParseSettings([]byte(sampleSettings))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("%d settings", len(got))
	}
	cap := got[0]
	if cap.Key != "skill_cap" || cap.Kind != KindInt || cap.Min != 100 || cap.Max != 150 || cap.Default.Int != 150 {
		t.Fatalf("%+v", cap)
	}
	if cap.Label("ru") != "Предел навыка" || cap.Label("en") != "Skill cap" || cap.Label("de") != "Skill cap" {
		t.Fatalf("labels %q %q", cap.Label("ru"), cap.Label("en"))
	}
	if got[1].Kind != KindBool || got[1].Default.Bool || got[1].Label("ru") != "Cloaks for mages too" {
		t.Fatalf("%+v", got[1])
	}
	if got[2].Kind != KindChoice || got[2].Default.Str != "calm" || got[2].Label("ru") != "Mood" {
		t.Fatalf("%+v", got[2])
	}
}

func TestParseSettingsRefusals(t *testing.T) {
	cases := []struct{ name, src, want string }{
		{"no type", "[a]\ndefault = 1\n", "type is missing"},
		{"bad type", "[a]\ntype = \"float\"\ndefault = 1\n", "not int, bool or choice"},
		{"no default", "[a]\ntype = \"bool\"\n", "default is missing"},
		{"int without range", "[a]\ntype = \"int\"\ndefault = 1\n", "needs min and max"},
		{"default outside", "[a]\ntype = \"int\"\ndefault = 5\nmin = 1\nmax = 3\n", "outside 1..3"},
		{"min above max", "[a]\ntype = \"int\"\ndefault = 2\nmin = 3\nmax = 1\n", "above max"},
		{"unknown key", "[a]\ntype = \"bool\"\ndefault = true\ncolour = \"red\"\n", `unknown key "colour"`},
		{"bool with range", "[a]\ntype = \"bool\"\ndefault = true\nmin = 1\n", "takes no min"},
		{"choice default", "[a]\ntype = \"choice\"\nchoices = [\"x\"]\ndefault = \"y\"\n", "not one of x"},
		{"bad label language", "[a]\nlabel = { fr = \"x\" }\ntype = \"bool\"\ndefault = true\n", "no language"},
		{"bad key", "[Bad]\ntype = \"bool\"\ndefault = true\n", "the key must be"},
		{"outside a table", "type = \"bool\"\n", "outside one"},
		{"float", "[a]\ntype = \"int\"\ndefault = 1.5\nmin = 1\nmax = 3\n", "only decimal integers"},
		{"duplicate table", "[a]\ntype = \"bool\"\ndefault = true\n[a]\n", "given twice"},
		{"array of tables", "[[a]]\n", "not supported"},
	}
	for _, c := range cases {
		_, err := ParseSettings([]byte(c.src))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
}

func TestResolveSettingsAppliesGivenValuesOverDefaults(t *testing.T) {
	decl, _ := ParseSettings([]byte(sampleSettings))
	got, err := ResolveSettings("m", decl, map[string]string{"skill_cap": "120", "cloaks_for_mages": "on"})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Value.Int != 120 || !got[1].Value.Bool || got[2].Value.Str != "calm" {
		t.Fatalf("%+v", got)
	}
	for _, c := range []struct {
		given map[string]string
		want  string
	}{
		{map[string]string{"skill_cap": "99"}, "outside 100..150"},
		{map[string]string{"skill_cap": "many"}, "not an integer"},
		{map[string]string{"cloaks_for_mages": "maybe"}, "not true or false"},
		{map[string]string{"difficulty": "wild"}, "not one of calm, grim"},
		{map[string]string{"other": "1"}, `has no setting "other"`},
	} {
		_, err := ResolveSettings("m", decl, c.given)
		if err == nil || !strings.Contains(err.Error(), c.want) || !strings.Contains(err.Error(), `mod "m"`) {
			t.Errorf("%v: got %v, want %q", c.given, err, c.want)
		}
	}
}

func TestParseSettingFlag(t *testing.T) {
	f, err := ParseSettingFlag("skill-cap.v2.skill_cap=150")
	if err != nil || f != (SettingFlag{Mod: "skill-cap.v2", Key: "skill_cap", Value: "150"}) {
		t.Fatalf("%+v %v", f, err)
	}
	for _, bad := range []string{"skill_cap=150", "m.=1", ".k=1", "m.K=1", "M.k=1", "m.k"} {
		if _, err := ParseSettingFlag(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}

func TestLoadSettingsMissingFileDeclaresNothing(t *testing.T) {
	dir := t.TempDir()
	got, err := LoadSettings(dir)
	if err != nil || got != nil {
		t.Fatalf("%v %v", got, err)
	}
	if err := os.WriteFile(filepath.Join(dir, SettingsName), []byte("[a]\ntype = \"x\"\ndefault = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSettings(dir); err == nil || !strings.Contains(err.Error(), SettingsName) {
		t.Fatalf("error does not name the file: %v", err)
	}
}
