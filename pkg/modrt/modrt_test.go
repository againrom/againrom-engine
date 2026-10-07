package modrt

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.starlark.net/starlark"

	"againrom/pkg/mod"
	"againrom/pkg/rules"
)

// makeMods writes each mod as a folder under a temp directory (files maps a
// slash path to its text; mod.toml defaults to a minimal manifest) and resolves
// them.
func makeMods(t *testing.T, mods map[string]map[string]string) []mod.Entry {
	t.Helper()
	root := t.TempDir()
	var ids []string
	for id, files := range mods {
		ids = append(ids, id)
		if _, ok := files["mod.toml"]; !ok {
			files["mod.toml"] = "id = \"" + id + "\"\ntitle = \"" + id + "\"\nversion = \"1.0\"\n"
		}
		for name, body := range files {
			p := filepath.Join(root, id, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	entries, err := mod.Resolve(root, ids)
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func run(t *testing.T, script string) (Result, error) {
	t.Helper()
	return Load(makeMods(t, map[string]map[string]string{"m": {"main.star": script}}), "rom1-en", nil, Options{})
}

func exampleMods(t *testing.T) []mod.Entry {
	t.Helper()
	entries, err := mod.Resolve(filepath.Join("testdata", "mods"), []string{"skill-cap"})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func flag(t *testing.T, arg string) mod.SettingFlag {
	t.Helper()
	f, err := mod.ParseSettingFlag(arg)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestTheExampleModRaisesTheSkillCapToItsDefaultAndToAGivenValue(t *testing.T) {
	res, err := Load(exampleMods(t), "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Rules.SkillCap() != 150 {
		t.Fatalf("default cap %d", res.Rules.SkillCap())
	}
	if len(res.Set.Mods) != 1 || res.Set.Mods[0].ID != "skill-cap" || res.Set.Mods[0].Version != "1.0.0" ||
		len(res.Set.Mods[0].Digest) != 64 || res.Set.Mods[0].Settings[0].Value.Int != 150 || res.Set.Base != "rom1-en" {
		t.Fatalf("%+v", res.Set)
	}
	res2, err := Load(exampleMods(t), "rom1-en", []mod.SettingFlag{flag(t, "skill-cap.skill_cap=120")}, Options{})
	if err != nil || res2.Rules.SkillCap() != 120 {
		t.Fatalf("cap %d err %v", res2.Rules.SkillCap(), err)
	}
	if res2.Set.Digest() == res.Set.Digest() {
		t.Fatal("the digest ignores a setting")
	}
}

func TestSettingFlagRefusalsNameTheModAndKey(t *testing.T) {
	for _, c := range []struct {
		arg  string
		want string
	}{
		{"skill-cap.skill_cap=99", `mod "skill-cap" setting skill_cap: 99 is outside 100..150`},
		{"skill-cap.skill_cap=lots", `mod "skill-cap" setting skill_cap: "lots" is not an integer`},
		{"skill-cap.speed=3", `mod "skill-cap" has no setting "speed"`},
		{"other.skill_cap=3", "names a mod that is not enabled"},
	} {
		_, err := Load(exampleMods(t), "rom1-en", []mod.SettingFlag{flag(t, c.arg)}, Options{})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.arg, err, c.want)
		}
	}
	_, err := Load(exampleMods(t), "rom1-en", []mod.SettingFlag{flag(t, "skill-cap.skill_cap=110"), flag(t, "skill-cap.skill_cap=120")}, Options{})
	if err == nil || !strings.Contains(err.Error(), "given twice") {
		t.Errorf("a repeated setting: %v", err)
	}
	if _, err := Load(nil, "rom1-en", []mod.SettingFlag{flag(t, "skill-cap.skill_cap=110")}, Options{}); err == nil {
		t.Error("a setting with no mods was accepted")
	}
}

func TestNoModsLeavesTheOriginalRules(t *testing.T) {
	res, err := Load(nil, "rom1-ru", nil, Options{})
	if err != nil || !res.Rules.IsDefault() || !res.Set.Empty() || res.Set.Base != "rom1-ru" {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestAModWithoutAScriptChangesNothing(t *testing.T) {
	res, err := Load(makeMods(t, map[string]map[string]string{"data-only": {"data/x.toml": ""}}), "rom1-en", nil, Options{})
	if err != nil || !res.Rules.IsDefault() || len(res.Set.Mods) != 1 {
		t.Fatalf("%+v %v", res, err)
	}
}

func TestScriptErrorsNameTheModFileAndLine(t *testing.T) {
	for _, c := range []struct {
		name, script, want string
	}{
		{"fail", "def init(game, settings):\n    x = 1\n    fail(\"boom\")\n", `mod "m": main.star:3: fail: boom`},
		{"syntax", "def init(game, settings)\n    pass\n", `mod "m": main.star:`},
		{"undefined name", "def init(game, settings):\n    pass\nx = nosuch\n", `mod "m": main.star:3: undefined: nosuch`},
		{"runtime", "def init(game, settings):\n    y = 1 // 0\n", `mod "m": main.star:2:`},
		{"no init", "x = 1\n", `mod "m": main.star defines no init(game, settings)`},
		{"unknown parameter", "def init(game, settings):\n    game.rules.hp_cap = 5\n", `main.star:2: game.rules has no parameter "hp_cap" (declared: skill_cap)`},
		{"read unknown parameter", "def init(game, settings):\n    x = game.rules.hp_cap\n", `main.star:2: game.rules has no parameter "hp_cap"`},
		{"out of range", "def init(game, settings):\n    game.rules.skill_cap = 5000\n", `main.star:2: game.rules.skill_cap: 5000 is outside the declared range 1..150`},
		{"huge", "def init(game, settings):\n    game.rules.skill_cap = 1 << 100\n", `outside the declared range`},
		{"float", "def init(game, settings):\n    game.rules.skill_cap = 1.5\n", `main.star:2: game.rules.skill_cap must be an int, got float`},
		{"bool", "def init(game, settings):\n    game.rules.skill_cap = True\n", `must be an int, got bool`},
		{"string", "def init(game, settings):\n    game.rules.skill_cap = \"150\"\n", `must be an int, got string`},
		{"unknown setting", "def init(game, settings):\n    x = settings.skill_cap\n", `main.star:2:`},
		{"settings are read only", "def init(game, settings):\n    settings.x = 1\n", `main.star:2:`},
		{"while is not offered", "def init(game, settings):\n    while True:\n        pass\n", `main.star:2:`},
		{"recursion is refused", "def f(n):\n    return f(n)\ndef init(game, settings):\n    f(1)\n", `main.star:2: function f called recursively`},
		{"too long", "def init(game, settings):\n    for i in range(1000000000):\n        pass\n", "too many steps"},
	} {
		_, err := run(t, c.script)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: got %v, want %q", c.name, err, c.want)
		}
	}
}

func TestNothingForTheOutsideWorldIsReachable(t *testing.T) {
	for _, name := range []string{"os", "time", "random", "open", "exec", "http", "io", "file", "sys", "json", "math", "input", "rand", "now"} {
		_, err := run(t, "def init(game, settings):\n    x = "+name+"\n")
		if err == nil || !strings.Contains(err.Error(), "undefined: "+name) {
			t.Errorf("%s: got %v, want an undefined name", name, err)
		}
	}
}

func TestLoadReadsOnlyTheModsOwnFolder(t *testing.T) {
	files := map[string]string{
		"main.star":        "load(\"scripts/a.star\", \"double\")\ndef init(game, settings):\n    game.rules.skill_cap = double(60)\n",
		"scripts/a.star":   "load(\"b.star\", \"one\")\ndef double(n):\n    return n * 2 * one\n",
		"scripts/b.star":   "one = 1\n",
		"scripts/bad.star": "x = 1 // 0\n",
	}
	res, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
	if err == nil {
		// scripts/a.star loads "b.star" relative to the mod root, not to itself
		t.Fatalf("a load relative to the loading file resolved: cap %d", res.Rules.SkillCap())
	}
	if !strings.Contains(err.Error(), `scripts/a.star:1`) {
		t.Fatalf("error does not name the loading file and line: %v", err)
	}
	files["scripts/a.star"] = "load(\"scripts/b.star\", \"one\")\ndef double(n):\n    return n * 2 * one\n"
	res, err = Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
	if err != nil || res.Rules.SkillCap() != 120 {
		t.Fatalf("cap %d err %v", res.Rules.SkillCap(), err)
	}
	for _, target := range []string{"../outside.star", "/etc/passwd.star", "C:/x.star", "scripts\\a.star", "scripts/../../x.star", "", "scripts/a.txt", "nothere.star"} {
		script := "load(\"" + strings.ReplaceAll(target, "\\", "\\\\") + "\", \"x\")\ndef init(game, settings):\n    pass\n"
		_, err := run(t, script)
		if err == nil || !strings.Contains(err.Error(), "main.star:1") {
			t.Errorf("load %q: got %v", target, err)
		}
	}
	files2 := map[string]string{"main.star": "load(\"scripts/bad.star\", \"x\")\ndef init(game, settings):\n    pass\n", "scripts/bad.star": "x = 1 // 0\n"}
	_, err = Load(makeMods(t, map[string]map[string]string{"m": files2}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "scripts/bad.star:1") {
		t.Errorf("an error inside a loaded file: %v", err)
	}
}

func TestLoadCyclesAreRefused(t *testing.T) {
	files := map[string]string{
		"main.star": "load(\"a.star\", \"x\")\ndef init(game, settings):\n    pass\n",
		"a.star":    "load(\"b.star\", \"y\")\nx = 1\n",
		"b.star":    "load(\"a.star\", \"x\")\ny = 1\n",
	}
	_, err := Load(makeMods(t, map[string]map[string]string{"m": files}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("got %v", err)
	}
}

func TestLoadRefusesASymbolicLinkOutOfTheFolder(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(root, "outside.star")
	if err := os.WriteFile(outside, []byte("x = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "m")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, "link.star")); err != nil {
		t.Skipf("cannot create a symbolic link here: %v", err)
	}
	os.WriteFile(filepath.Join(dir, "mod.toml"), []byte("id = \"m\"\ntitle = \"m\"\nversion = \"1\"\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.star"), []byte("load(\"link.star\", \"x\")\ndef init(game, settings):\n    pass\n"), 0o644)
	entries, err := mod.Resolve(root, []string{"m"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(entries, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), "outside the mod folder") {
		t.Fatalf("got %v", err)
	}
}

func TestValuesAreFrozenAfterInit(t *testing.T) {
	// Top-level values are frozen once the file has run, so init cannot keep
	// state in them.
	_, err := run(t, "keep = []\ndef init(game, settings):\n    keep.append(game)\n")
	if err == nil || !strings.Contains(err.Error(), "main.star:3") || !strings.Contains(err.Error(), "frozen") {
		t.Fatalf("a global list was mutable in init: %v", err)
	}
	// The handle a script is given on game is frozen when init returns.
	sh := &shared{params: rules.Defaults()}
	game := &gameValue{rules: &rulesValue{sh: sh}, data: &dataValue{sh: sh}}
	if err := game.rules.SetField("skill_cap", starlark.MakeInt(120)); err != nil || sh.params.SkillCap != 120 {
		t.Fatalf("%v %d", err, sh.params.SkillCap)
	}
	game.Freeze()
	err = game.rules.SetField("skill_cap", starlark.MakeInt(130))
	if err == nil || !strings.Contains(err.Error(), "frozen") || sh.params.SkillCap != 120 {
		t.Fatalf("a frozen handle was written: %v %d", err, sh.params.SkillCap)
	}
}

func TestModsRunInLoadOrderAndShareTheRules(t *testing.T) {
	mods := makeMods(t, map[string]map[string]string{
		"first": {"main.star": "def init(game, settings):\n    game.rules.skill_cap = 120\n"},
		"second": {
			"mod.toml":  "id = \"second\"\ntitle = \"s\"\nversion = \"1\"\nload-after = [\"first\"]\n",
			"main.star": "def init(game, settings):\n    game.rules.skill_cap = game.rules.skill_cap + 10\n",
		},
	})
	res, err := Load([]mod.Entry{mods[0], mods[1]}, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Rules.SkillCap() != 130 {
		t.Fatalf("cap %d", res.Rules.SkillCap())
	}
	got := []string{res.Ordered[0].Manifest.ID, res.Ordered[1].Manifest.ID}
	if !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Fatalf("order %v", got)
	}
	// Naming them the other way round changes nothing.
	res2, _ := Load([]mod.Entry{mods[1], mods[0]}, "rom1-en", nil, Options{})
	if res2.Rules.SkillCap() != 130 || res2.Set.Digest() != res.Set.Digest() {
		t.Fatal("the order the mods were named in changed the outcome")
	}
}

func TestOrderRefusalsReachTheCaller(t *testing.T) {
	mods := makeMods(t, map[string]map[string]string{
		"a": {"mod.toml": "id = \"a\"\ntitle = \"a\"\nversion = \"1\"\nconflicts = [\"b\"]\n"},
		"b": {},
	})
	if _, err := Load(mods, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), `mod "a" conflicts with the enabled mod "b"`) {
		t.Fatalf("got %v", err)
	}
	wrong := makeMods(t, map[string]map[string]string{"a": {"mod.toml": "id = \"a\"\ntitle = \"a\"\nversion = \"1\"\napplies-to = [\"rom1-ru\"]\n"}})
	if _, err := Load(wrong, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), "does not include the active base rom1-en") {
		t.Fatalf("got %v", err)
	}
}

func TestEntryNamedInTheManifestMustExist(t *testing.T) {
	mods := makeMods(t, map[string]map[string]string{"m": {"mod.toml": "id = \"m\"\ntitle = \"m\"\nversion = \"1\"\nentry = \"nope.star\"\n"}})
	if _, err := Load(mods, "rom1-en", nil, Options{}); err == nil || !strings.Contains(err.Error(), `mod "m": entry nope.star`) {
		t.Fatalf("got %v", err)
	}
	mods = makeMods(t, map[string]map[string]string{"m": {"mod.toml": "id = \"m\"\ntitle = \"m\"\nversion = \"1\"\nentry = \"../x.star\"\n"}})
	if _, err := Load(mods, "rom1-en", nil, Options{}); err == nil {
		t.Fatal("an entry outside the folder was accepted")
	}
}

func TestPrintGoesToTheCallerOnly(t *testing.T) {
	var got []string
	mods := makeMods(t, map[string]map[string]string{"m": {"main.star": "def init(game, settings):\n    print(\"hello\", game.rules.skill_cap)\n"}})
	if _, err := Load(mods, "rom1-en", nil, Options{Print: func(id, msg string) { got = append(got, id+": "+msg) }}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{"m: hello 100"}) {
		t.Fatalf("%v", got)
	}
}

func TestRulesDeclareTheirFields(t *testing.T) {
	if !reflect.DeepEqual(FieldNames(), []string{"skill_cap"}) {
		t.Fatalf("%v", FieldNames())
	}
	f := fieldByName("skill_cap")
	if f.min != int64(rules.MinSkillCap) || f.max != int64(rules.MaxSkillCap) {
		t.Fatalf("the declared range is not the rules package's: %d..%d", f.min, f.max)
	}
}
