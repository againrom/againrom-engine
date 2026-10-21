package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackModFlagsGoTogetherAndExcludeOtherModes(t *testing.T) {
	o, err := parse([]string{"-pack-mod", "m", "-pack-out", "out", "-pack-replace"})
	if err != nil || o.packMod != "m" || o.packOut != "out" || !o.packReplace {
		t.Fatalf("parsed %+v: %v", o, err)
	}
	for args, want := range map[string]string{
		"-pack-mod m":                            "-pack-mod needs -pack-out",
		"-pack-out out":                          "-pack-out and -pack-replace need -pack-mod",
		"-pack-replace":                          "-pack-out and -pack-replace need -pack-mod",
		"-pack-mod m -pack-out out -check":       "-pack-mod excludes",
		"-pack-mod m -pack-out out -mods a":      "-pack-mod excludes",
		"-pack-mod m -pack-out out -dump-base d": "-pack-mod excludes",
	} {
		if _, err := parse(strings.Fields(args)); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", args, err, want)
		}
	}
}

// writePackFixture writes a mod folder fix into dir: a rule set from a
// setting, a picture and a text of the synthetic install's.
func writePackFixture(t *testing.T, dir string) {
	t.Helper()
	writeMod(t, dir, "fix", goodManifest("fix"))
	writeModFile(t, dir, "fix", "settings.toml", []byte("[cap]\nlabel = \"Cap\"\ntype = \"int\"\ndefault = 120\nmin = 100\nmax = 150\n"))
	writeModFile(t, dir, "fix", "main.star", []byte("def init(game, settings):\n    game.rules.skill_cap = settings.cap\n"))
	writeModFile(t, dir, "fix", "files/common/main/graphics/chrgen/leftup.bmp", installChargenBitmap(160, 238))
	writeModFile(t, dir, "fix", "files/en/main/text/npcnames.txt", []byte(strings.Repeat("Hero\r\n", 24)))
	writeModFile(t, dir, "fix", "files/ru/main/text/npcnames.txt", []byte("Герой\r\n"))
}

// -pack-mod writes mod-<id>.res; -check reports the packed mod as it reports
// its folder, line for line but the path, and stderr alike. A folder and the
// packed file of one id in one mods directory are refused naming both.
func TestCheckReportsAPackedModAsItsFolder(t *testing.T) {
	root := defaultInstall(t)
	src, mods := t.TempDir(), t.TempDir()
	writePackFixture(t, src)
	folder := filepath.Join(src, "fix")
	code, out, stderr := checkRun(t, []string{"-pack-mod", folder, "-pack-out", mods}, "")
	packed := filepath.Join(mods, "mod-fix.res")
	if code != 0 || out != "againrom: packed "+folder+" into "+packed+"\n" || stderr != "" {
		t.Fatalf("pack: exit %d out %q err %q", code, out, stderr)
	}
	args := []string{"-assets", root, "-check", "-picker", "-mods", "fix", "-mod-setting", "fix.cap=140"}
	code, want, wantErr := checkRun(t, append(args, "-mods-dir", src), "")
	if code != 0 || !strings.Contains(want, "dir="+folder) || !strings.Contains(want, "skill_cap=140") || !strings.Contains(want, "install files=2") {
		t.Fatalf("folder: exit %d\n%s\n%s", code, want, wantErr)
	}
	code, got, gotErr := checkRun(t, append(args, "-mods-dir", mods), "")
	if code != 0 || got != strings.Replace(want, "dir="+folder, "dir="+packed, 1) || gotErr != wantErr {
		t.Fatalf("packed: exit %d\n%s\nwant\n%s\nstderr %q, want %q", code, got, want, gotErr, wantErr)
	}

	writePackFixture(t, mods)
	code, out, stderr = checkRun(t, append(args, "-mods-dir", mods), "")
	both := `mod "fix" is both ` + filepath.Join(mods, "fix") + " and " + packed
	if code == 0 || out != "" || !strings.Contains(stderr, both) {
		t.Fatalf("folder and packed file: exit %d out %q err %q", code, out, stderr)
	}
}

// -pack-mod writes over an existing file only with -pack-replace, never into
// an install, and refuses a name the RES cannot hold naming the file; a
// refused run leaves no file.
func TestPackModWritesOnlyWhereAskedAndNamesARefusedFile(t *testing.T) {
	src, out := t.TempDir(), t.TempDir()
	writePackFixture(t, src)
	folder := filepath.Join(src, "fix")
	packed := filepath.Join(out, "mod-fix.res")
	if code, _, stderr := checkRun(t, []string{"-pack-mod", folder, "-pack-out", out}, ""); code != 0 {
		t.Fatalf("first pack: %s", stderr)
	}
	before, err := os.ReadFile(packed)
	if err != nil {
		t.Fatal(err)
	}
	writeModFile(t, src, "fix", "notes.txt", []byte("more"))
	code, _, stderr := checkRun(t, []string{"-pack-mod", folder, "-pack-out", out}, "")
	if code != 1 || !strings.Contains(stderr, packed+" exists: a packed mod is written over an existing file only when asked") {
		t.Fatalf("second pack: exit %d %q", code, stderr)
	}
	if now, _ := os.ReadFile(packed); string(now) != string(before) {
		t.Fatal("a refused pack changed the file")
	}
	if code, _, stderr := checkRun(t, []string{"-pack-mod", folder, "-pack-out", out, "-pack-replace"}, ""); code != 0 {
		t.Fatalf("replace: %s", stderr)
	}
	if now, _ := os.ReadFile(packed); string(now) == string(before) {
		t.Fatal("-pack-replace left the old file")
	}

	root := defaultInstall(t)
	inside := filepath.Join(root, "mods")
	if err := os.Mkdir(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	code, _, stderr = checkRun(t, []string{"-pack-mod", folder, "-pack-out", inside}, "")
	if code != 1 || !strings.Contains(stderr, "a packed mod is never written into an install") {
		t.Fatalf("into the install: exit %d %q", code, stderr)
	}

	writeModFile(t, src, "fix", "data/weapon-bodies.toml", []byte("x"))
	empty := t.TempDir()
	code, _, stderr = checkRun(t, []string{"-pack-mod", folder, "-pack-out", empty}, "")
	if code != 1 || !strings.Contains(stderr, `mod "fix": data/weapon-bodies.toml: cannot be packed: the name "weapon-bodies.toml" is 18 bytes`) {
		t.Fatalf("long name: exit %d %q", code, stderr)
	}
	for _, dir := range []string{inside, empty} {
		if left, _ := os.ReadDir(dir); len(left) != 0 {
			t.Errorf("a refused pack left %v in %s", left, dir)
		}
	}
}
