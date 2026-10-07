package game

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
)

func cutsceneFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "ALLods"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, archive := range []string{"VIDEO4.RES", "VIDEO8.RES"} {
		b := synth.Archive([]synth.File{
			{Path: "M10/01.SMK", Data: []byte(archive)},
			{Path: "LOGOS/BUKA.SMK", Data: []byte(archive)},
		})
		if err := os.WriteFile(filepath.Join(dir, "ALLods", archive), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func TestCutsceneBankUsesOnlyExplicitArchiveAndCaseFoldedInstall(t *testing.T) {
	root := cutsceneFixture(t)
	for _, tc := range []struct{ archive, want string }{{"video4", "VIDEO4.RES"}, {"video8", "VIDEO8.RES"}} {
		bank := OpenCutscenes(root, tc.archive)
		got, err := bank.Media(`M10\01.SMK`)
		if err != nil || !bytes.Equal(got, []byte(tc.want)) {
			t.Fatalf("%s %q %v", tc.archive, got, err)
		}
		if _, err := bank.Media("missing.smk"); err == nil {
			t.Fatal("missing member resolved")
		}
		if logo, err := bank.Media("logos/buka.smk"); err != nil || string(logo) != "VIDEO4.RES" {
			t.Fatalf("%s logo = %q %v", tc.archive, logo, err)
		}
	}
}

func TestCutsceneBankRejectsInvalidPathsMediaAndOversizedRegistry(t *testing.T) {
	bank := OpenCutscenes(cutsceneFixture(t), "video8")
	for _, name := range []string{"", "../m10/01.smk", "/m10/01.smk", "m10//01.smk", "C:/movie.smk", "m10/01.reg"} {
		if _, err := bank.Media(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if _, err := bank.Open("m10/01.smk"); err == nil {
		t.Fatal("invalid Smacker payload accepted")
	}
	if _, err := OpenCutscenes(bank.root, "other").Media("m10/01.smk"); err == nil {
		t.Fatal("other archive accepted")
	}
	var header [24]byte
	binary.LittleEndian.PutUint32(header[20:], 16385)
	if err := os.WriteFile(filepath.Join(bank.root, "ALLods", "VIDEO8.RES"), header[:], 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenCutscenes(bank.root, "video8").Media("m10/01.smk"); err == nil {
		t.Fatal("oversized registry accepted")
	}
}

func TestCutsceneArchiveFourXOverridesEightX(t *testing.T) {
	for _, tc := range []struct {
		four, eight bool
		want        string
	}{{false, false, "video4"}, {true, false, "video4"}, {false, true, "video8"}, {true, true, "video4"}} {
		if got := CutsceneArchive(tc.four, tc.eight); got != tc.want {
			t.Fatalf("%+v got=%s", tc, got)
		}
	}
}
