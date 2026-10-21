package main

import (
	"image"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
)

func writeModFile(t *testing.T, modsDir, id, rel string, data []byte) {
	t.Helper()
	p := filepath.Join(modsDir, id, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// -check takes a mod's install files on the synthetic install: the picture
// of the install's size and depth and the text of the install's language
// are counted, and the mod is listed as one the SAV mod mark leaves out.
// Each file the game cannot use is refused naming the mod, the file and the
// reason; two mods serving one path are refused without an order and with
// one the later wins and the check names the override.
func TestCheckTakesAModsInstallFilesAndRefusesWhatTheGameCannotUse(t *testing.T) {
	root := defaultInstall(t)
	mods := t.TempDir()
	const picture, names = "main/graphics/chrgen/leftup.bmp", "main/text/npcnames.txt"
	writeMod(t, mods, "pic", goodManifest("pic"))
	writeModFile(t, mods, "pic", "files/common/"+picture, installChargenBitmap(160, 238))
	code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "pic", "-mods-dir", mods}, "")
	if code != 0 || !strings.Contains(out, "againrom: install files=1 state-free=pic (a mod with no rule") || strings.Contains(stderr, "override") {
		t.Fatalf("exit %d\nout %s\nerr %s", code, out, stderr)
	}
	// A text can change state, so the mod that serves one is not state-free.
	writeMod(t, mods, "words", goodManifest("words"))
	writeModFile(t, mods, "words", "files/en/"+names, []byte(strings.Repeat("Hero\r\n", 24)))
	writeModFile(t, mods, "words", "files/ru/"+names, []byte("Герой\r\n"))
	code, out, stderr = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "pic,words", "-mods-dir", mods}, "")
	if code != 0 || !strings.Contains(out, "againrom: install files=2 state-free=pic (a mod with no rule") || strings.Contains(stderr, "override") {
		t.Fatalf("exit %d\nout %s\nerr %s", code, out, stderr)
	}

	for id, c := range map[string]struct {
		file string
		data []byte
		want string
	}{
		"wide":   {"files/common/" + picture, synth.BMP24(image.NewRGBA(image.Rect(0, 0, 161, 238))), "files/common/" + picture + ": the picture is 161x238; the install's is 160x238"},
		"absent": {"files/common/main/text/absent.txt", []byte("x"), "files/common/main/text/absent.txt: the install holds no file main/text/absent.txt"},
		"lang":   {"files/de/" + names, []byte("x"), `files/de: "de" is not a language folder`},
		"kind":   {"files/en/main/graphics/a.256", []byte("x"), "files/en/main/graphics/a.256: the game takes no install file of this kind"},
		"han":    {"files/en/" + names, []byte(strings.Repeat("中\r\n", 24)), "files/en/" + names + `: the install's alphabet has no '中' (U+4E2D)`},
	} {
		writeMod(t, mods, id, goodManifest(id))
		writeModFile(t, mods, id, c.file, c.data)
		code, out, stderr := checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", id, "-mods-dir", mods}, "")
		if code == 0 || !strings.Contains(stderr, `mod "`+id+`": `+c.want) || out != "" {
			t.Errorf("%s: exit %d stderr %q stdout %q, want %q", id, code, stderr, out, c.want)
		}
	}

	writeMod(t, mods, "other", goodManifest("other"))
	writeModFile(t, mods, "other", "files/common/"+names, []byte(strings.Repeat("Other\r\n", 24)))
	code, _, stderr = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "words,other", "-mods-dir", mods}, "")
	if code == 0 || !strings.Contains(stderr, "install file "+names+` is served by mod "other"`) || !strings.Contains(stderr, "neither loads after the other") {
		t.Fatalf("unordered: exit %d stderr %q", code, stderr)
	}
	writeMod(t, mods, "other", goodManifest("other")+"load-after = [\"words\"]\n")
	writeModFile(t, mods, "other", "files/common/"+names, []byte(strings.Repeat("Other\r\n", 24)))
	code, out, stderr = checkRun(t, []string{"-assets", root, "-check", "-picker", "-mods", "words,other", "-mods-dir", mods}, "")
	want := `againrom: override: install file ` + names + `: mod "other" (files/common/` + names + `) overrides mod "words" (files/en/` + names + `)`
	if code != 0 || !strings.Contains(stderr, want) || !strings.Contains(out, "againrom: install files=2 state-free= (a mod with no rule") {
		t.Fatalf("ordered: exit %d\nout %s\nerr %s", code, out, stderr)
	}
}
