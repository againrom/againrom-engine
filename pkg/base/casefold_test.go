package base

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"againrom/pkg/installpath"
)

// TestDetectOnCaseSensitiveFS detects an exact build whose every file is spelt
// lower, upper or mixed case, on a file system that refuses a name differing
// only by case, and reports two spellings of one archive.
func TestDetectOnCaseSensitiveFS(t *testing.T) {
	names := append(RequiredArchives(), "video4.res", "allods2.exe", "scenario.dll")
	for label, spell := range map[string]func(string) string{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"mixed": func(s string) string { return strings.ToUpper(s[:1]) + s[1:] },
	} {
		t.Run(label, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for _, n := range names[:6] {
				body := "x"
				if n == MainArchive {
					body = "demo-main"
				}
				fsys[spell(n)] = &fstest.MapFile{Data: []byte(body)}
			}
			if _, err := fsys.Open(MainArchive); label != "lower" && !errors.Is(err, fs.ErrNotExist) {
				t.Fatalf("loss control: the exact name opened on a %s install: %v", label, err)
			}
			m, err := detect(installpath.NewFS(fsys), "root", profilesFor("demo-main"), nil)
			if err != nil || m.ID() != "demo" || !m.Exact {
				t.Fatalf("match = %+v, %v; want exact demo", m, err)
			}
			// The second game's marks resolve the same way.
			for _, n := range names[6:] {
				fsys[spell(n)] = &fstest.MapFile{}
			}
			m, err = detect(installpath.NewFS(fsys), "root", Profiles, nil)
			if err != nil || m.Profile.GameOf() != GameROM2 {
				t.Fatalf("match = %+v, %v; want the second game", m, err)
			}
		})
	}
	fsys := fstest.MapFS{}
	for _, n := range RequiredArchives() {
		fsys[n] = &fstest.MapFile{}
	}
	delete(fsys, "world.res")
	fsys["WORLD.RES"] = &fstest.MapFile{}
	fsys["World.res"] = &fstest.MapFile{}
	_, err := detect(installpath.NewFS(fsys), "root", profilesFor("demo-main"), nil)
	if !errors.Is(err, installpath.ErrAmbiguous) || !strings.Contains(err.Error(), "WORLD.RES") || !strings.Contains(err.Error(), "World.res") {
		t.Fatalf("detect = %v, want both spellings of world.res reported", err)
	}
}
