package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
)

func TestParseReadsModSettingFlags(t *testing.T) {
	o, err := parse([]string{"-mods", "skill-cap", "-mod-setting", "skill-cap.skill_cap=120", "-mod-setting", "skill-cap.other=on", "-mods-accept-unmarked"})
	if err != nil {
		t.Fatal(err)
	}
	if len(o.modSettings) != 2 || o.modSettings[0].Mod != "skill-cap" || o.modSettings[0].Key != "skill_cap" ||
		o.modSettings[0].Value != "120" || !o.modsAcceptUnmarked {
		t.Fatalf("%+v %v", o.modSettings, o.modsAcceptUnmarked)
	}
	for _, args := range [][]string{
		{"-mods", "a", "-mod-setting", "nodot=1"},
		{"-mods", "a", "-mod-setting", "a.K=1"},
		{"-mod-setting", "a.k=1"},
		{"-mods-accept-unmarked"},
	} {
		if _, err := parse(args); err == nil {
			t.Errorf("%v accepted", args)
		}
	}
}

const exampleModsDir = "../../pkg/modrt/testdata/mods"

func TestCheckRunsTheExampleModAndReportsItsRules(t *testing.T) {
	root := defaultInstall(t)
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir}, "")
	if code != 0 || !strings.Contains(out, "againrom: rules skill_cap=150") || !strings.Contains(out, "settings=skill_cap=150") {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
	code, out, stderr = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir, "-mod-setting", "skill-cap.skill_cap=120"}, "")
	if code != 0 || !strings.Contains(out, "againrom: rules skill_cap=120") {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
}

func TestCheckRefusesABadSettingByName(t *testing.T) {
	root := defaultInstall(t)
	for _, c := range []struct{ setting, want string }{
		{"skill-cap.skill_cap=99", `mod "skill-cap" setting skill_cap: 99 is outside 100..150`},
		{"skill-cap.unknown=1", `mod "skill-cap" has no setting "unknown"`},
	} {
		code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir, "-mod-setting", c.setting}, "")
		if code != 2 || out != "" || !strings.Contains(stderr, c.want) {
			t.Errorf("%s: exit %d stderr %q stdout %q", c.setting, code, stderr, out)
		}
	}
}

func TestAScriptErrorStopsTheLaunchNamingFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeMod(t, mods, "broken", goodManifest("broken"))
	script := "def init(game, settings):\n    game.rules.skill_cap = 5000\n"
	if err := os.WriteFile(filepath.Join(mods, "broken", "main.star"), []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "broken", "-mods-dir", mods}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "broken": main.star:2: game.rules.skill_cap: 5000 is outside the declared range 1..150`) {
		t.Fatalf("exit %d stderr %q stdout %q", code, stderr, out)
	}
}

func TestAModForAnotherBaseIsRefusedByName(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeMod(t, mods, "wrong", "id = \"wrong\"\ntitle = \"W\"\nversion = \"1\"\napplies-to = [\"rom2\"]\n")
	code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "wrong", "-mods-dir", mods}, "")
	if code != 2 || !strings.Contains(stderr, `mod "wrong": applies-to rom2 does not include the active base`) {
		t.Fatalf("exit %d stderr %q", code, stderr)
	}
}

func TestCheckRefusesAnItemChangeOfARowTheInstallLacks(t *testing.T) {
	root := defaultInstall(t)
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "heavy-armor", "-mods-dir", exampleModsDir}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "heavy-armor": data/items.toml:53: no row named "Chain Mail"`) {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
	code, out, _ = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir}, "")
	if code != 0 || strings.Contains(out, "items added") {
		t.Fatalf("a mod without items printed an item line: %q", out)
	}
}

func TestAnItemFileErrorStopsTheLaunchNamingModFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeMod(t, mods, "bad-items", goodManifest("bad-items"))
	files := map[string]string{
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/items.toml\")\n",
		"data/items.toml":      "[[item]]\nkey = \"a\"\nname = \"item.a\"\nslot = \"belt\"\nstand-in = \"Soft Mail\"\n",
		"text/en/strings.toml": "\"item.a\" = \"A\"\n",
	}
	for name, body := range files {
		path := filepath.Join(mods, "bad-items", filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "bad-items", "-mods-dir", mods}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "bad-items": data/items.toml:4: unknown slot "belt"`) {
		t.Fatalf("exit %d stderr %q stdout %q", code, stderr, out)
	}
}

func writeModFiles(t *testing.T, mods, id string, files map[string]string) {
	t.Helper()
	writeMod(t, mods, id, goodManifest(id))
	for name, body := range files {
		path := filepath.Join(mods, id, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

const screenModScript = "def init(game, settings):\n    game.data.add(\"data/screens.toml\")\n"

func screenModFiles(screens string) map[string]string {
	return map[string]string{
		"main.star":            screenModScript,
		"data/screens.toml":    screens,
		"text/en/strings.toml": "\"t\" = \"Title\"\n\"m\" = \"Entry\"\n\"p\" = \"Body\"\n",
	}
}

const oneScreen = "[[screen]]\nkey = \"a\"\nkind = \"info\"\ntitle = \"t\"\nmenu = \"m\"\nplace = \"both\"\ntext = [\"p\"]\n"

func TestCheckReportsTheScreensAModDeclares(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeModFiles(t, mods, "pages", screenModFiles(oneScreen))
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "pages", "-mods-dir", mods}, "")
	if code != 0 || !strings.Contains(out, "againrom: screens=1 main-menu=1 game-menu=1") {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
	code, out, _ = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir}, "")
	if code != 0 || strings.Contains(out, "screens=") {
		t.Fatalf("a mod without screens printed a screen line: %q", out)
	}
}

func TestAScreenFileErrorStopsTheLaunchNamingModFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	var four strings.Builder
	for _, key := range []string{"ka", "kb", "kc", "kd"} {
		four.WriteString(strings.Replace(strings.Replace(oneScreen, `key = "a"`, `key = "`+key+`"`, 1), `"both"`, `"main"`, 1))
	}
	for name, c := range map[string]struct{ src, want string }{
		"unknown kind": {strings.Replace(oneScreen, `"info"`, `"carousel"`, 1), `mod "bad-screens": data/screens.toml:3: unknown screen kind "carousel"`},
		"missing text": {strings.Replace(oneScreen, `["p"]`, `["nope"]`, 1), `mod "bad-screens": data/screens.toml:7: text key "nope" is not in text/en/strings.toml`},
		"too many":     {four.String(), `mod "bad-screens": data/screens.toml:22: screen "kd" does not fit: the main menu has 3 slots for mod screens`},
	} {
		mods := t.TempDir()
		writeModFiles(t, mods, "bad-screens", screenModFiles(c.src))
		code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "bad-screens", "-mods-dir", mods}, "")
		if code != 2 || out != "" || !strings.Contains(stderr, c.want) {
			t.Errorf("%s: exit %d stderr %q stdout %q", name, code, stderr, out)
		}
	}
}

func TestAJoinConditionTheCampaignDoesNotBearOutStopsTheLaunchNamingModFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeModFiles(t, mods, "joins", map[string]string{
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/companions.toml\")\n",
		"data/companions.toml": "[[join]]\nkey = \"a\"\ncompanion = 22\nchapter = 30\nbuilding = \"tavern\"\ntalk = 22\n",
	})
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "joins", "-mods-dir", mods}, "")
	want := `mod "joins": data/companions.toml:1: chapter 30 is not a chapter of this install's campaign`
	if code != 2 || out != "" || !strings.Contains(stderr, want) {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
}

func TestModLinesCountTheCompanionJoins(t *testing.T) {
	var res modrt.Result
	for _, l := range modLines(res) {
		if strings.Contains(l, "companion joins") {
			t.Fatalf("a mod set without joins printed %q", l)
		}
	}
	res.Companions.Joins = []mod.CompanionJoin{{Key: "a"}, {Key: "b"}}
	lines := modLines(res)
	if lines[len(lines)-1] != "againrom: companion joins=2" {
		t.Fatalf("%q", lines)
	}
}

func TestCheckRefusesACharacterEditOfARowTheInstallLacks(t *testing.T) {
	root := defaultInstall(t)
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "archer-girl", "-mods-dir", exampleModsDir}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "archer-girl": data/characters.toml:3: no definition row is named "NPC06"`) {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
	code, out, _ = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "skill-cap", "-mods-dir", exampleModsDir}, "")
	if code != 0 || strings.Contains(out, "characters edited") {
		t.Fatalf("a mod without characters printed a character line: %q", out)
	}
}

func TestACharacterFileErrorStopsTheLaunchNamingModFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeModFiles(t, mods, "bad-characters", map[string]string{
		"main.star":            "def init(game, settings):\n    game.data.add(\"data/characters.toml\")\n",
		"data/characters.toml": "[[character]]\ntarget = \"NPC06\"\nstrip = [\"boots\"]\n",
	})
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "bad-characters", "-mods-dir", mods}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "bad-characters": data/characters.toml:3: unknown strip group "boots"`) {
		t.Fatalf("exit %d stderr %q stdout %q", code, stderr, out)
	}
}

func TestASpellEditTheInstallCannotTakeStopsTheLaunchNamingModFileAndLine(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeModFiles(t, mods, "spells", map[string]string{
		"main.star":        "def init(game, settings):\n    game.data.add(\"data/spells.toml\")\n",
		"data/spells.toml": "[[spell]]\ntarget = \"Fire Storm\"\nmana = 4\n",
	})
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "spells", "-mods-dir", mods}, "")
	if code != 2 || out != "" || !strings.Contains(stderr, `mod "spells": data/spells.toml:1: `) {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
}

func TestModLinesCountTheSpellEdits(t *testing.T) {
	var res modrt.Result
	for _, l := range modLines(res) {
		if strings.Contains(l, "spells edited") {
			t.Fatalf("a mod set without spell edits printed %q", l)
		}
	}
	res.Spells.Rows = []mod.SpellRow{{Target: "a"}, {Target: "b"}}
	res.Spells.Global.Mana = mod.SpellRatio{Set: true, Num: 1, Den: 2}
	lines := modLines(res)
	if lines[len(lines)-1] != "againrom: spells edited=2 global=1" {
		t.Fatalf("%q", lines)
	}
}
