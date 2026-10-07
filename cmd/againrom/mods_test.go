package main

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writeMod(t *testing.T, modsDir, id, manifest string) {
	t.Helper()
	dir := filepath.Join(modsDir, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mod.toml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
}

func goodManifest(id string) string {
	return "id = \"" + id + "\"\ntitle = \"Title of " + id + "\"\nversion = \"0.3\"\napplies-to = [\"rom1\"]\n"
}

func TestParseReadsModFlags(t *testing.T) {
	o, err := parse([]string{"-mods", "a, b,c", "-mods-dir", "somewhere"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(o.mods, []string{"a", "b", "c"}) || o.modsDir != "somewhere" {
		t.Fatalf("got %v %q", o.mods, o.modsDir)
	}
	o, err = parse(nil)
	if err != nil || o.mods != nil || o.modsDir != "" {
		t.Fatalf("defaults %v %q %v", o.mods, o.modsDir, err)
	}
}

func checkRun(t *testing.T, args []string, executable string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := runWithProfilePaths(args, noEnv, &out, &errOut, executable, t.TempDir())
	return code, out.String(), errOut.String()
}

func TestCheckListsResolvedModsAndLeavesOtherOutputAlone(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeMod(t, mods, "alpha", goodManifest("alpha"))
	writeMod(t, mods, "beta", goodManifest("beta"))

	code, plain, _ := checkRun(t, []string{"-assets", root, "-check", "-picker"}, "")
	if code != 0 {
		t.Fatalf("plain check exit %d", code)
	}
	code, withMods, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "beta,alpha", "-mods-dir", mods}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.HasPrefix(withMods, plain) {
		t.Fatalf("the check summary changed:\n%s\nvs\n%s", withMods, plain)
	}
	lines := strings.Split(strings.TrimSpace(strings.TrimPrefix(withMods, plain)), "\n")
	if len(lines) != 4 || !strings.Contains(lines[0], "2 mod(s) active on rom1-en, mod-set ") {
		t.Fatalf("mod lines %q", lines)
	}
	if !strings.Contains(lines[1], "mod 1: alpha 0.3 \"Title of alpha\" applies-to=rom1 settings= dir="+filepath.Join(mods, "alpha")) ||
		!strings.HasPrefix(lines[2], "againrom: mod 2: beta ") || lines[3] != "againrom: rules skill_cap=100" {
		t.Fatalf("mod lines %q", lines)
	}
}

func TestModsDirDefaultsToTheFolderBesideTheBinary(t *testing.T) {
	root := defaultInstall(t)
	beside := t.TempDir()
	writeMod(t, filepath.Join(beside, "mods"), "alpha", goodManifest("alpha"))
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "alpha"}, filepath.Join(beside, "againrom.exe"))
	if code != 0 || !strings.Contains(out, "mod 1: alpha ") {
		t.Fatalf("exit %d out %q err %q", code, out, stderr)
	}
}

func TestMissingOrMalformedModIsRefusedByName(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	writeMod(t, mods, "alpha", goodManifest("alpha"))
	writeMod(t, mods, "bad", "id = \"bad\"\n")
	for _, c := range []struct{ mods, want string }{
		{"alpha,ghost", `mod "ghost"`},
		{"bad", `mod "bad"`},
		{"../alpha", "not a valid mod id"},
		{"alpha,alpha", "named twice"},
	} {
		code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", c.mods, "-mods-dir", mods}, "")
		if code == 0 || !strings.Contains(stderr, c.want) || out != "" {
			t.Errorf("-mods %s: exit %d stderr %q stdout %q, want a refusal naming %q", c.mods, code, stderr, out, c.want)
		}
	}
	code, _, stderr := checkRun(t, []string{"-assets", root, "-check", "-mods", "alpha"}, "")
	if code == 0 || !strings.Contains(stderr, "executable's folder is unknown") {
		t.Errorf("no mods dir: exit %d stderr %q", code, stderr)
	}
}

// TestReleaseCheckListsAModOverTheInstall runs -check over the lawful install
// named by AGAINROM_ASSETS with a constructed mod folder: the mod is listed and
// the check passes, and a missing mod fails naming it.
func TestReleaseCheckListsAModOverTheInstall(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: the check over a lawful install needs one")
	}
	mods := t.TempDir()
	writeMod(t, mods, "witness", goodManifest("witness"))

	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-mods", "witness", "-mods-dir", mods}, "")
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(out, "1 mod(s) active on ") || !strings.Contains(out, "mod 1: witness 0.3 ") {
		t.Fatalf("the check did not list the mod:\n%s", out)
	}
	code, _, stderr = checkRun(t, []string{"-assets", root, "-check", "-mods", "witness,absent", "-mods-dir", mods}, "")
	if code == 0 || !strings.Contains(stderr, `mod "absent"`) {
		t.Fatalf("exit %d stderr %q, want a refusal naming the missing mod", code, stderr)
	}
}
