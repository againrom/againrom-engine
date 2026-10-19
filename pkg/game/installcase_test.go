package game

import (
	"errors"
	"io/fs"
	"strings"
	"testing"
	"testing/fstest"

	"againrom/pkg/base"
	"againrom/pkg/installpath"
)

// installReadNames is every name under an install root the game, the starter
// and the map editor open, for both games: the five required archives, the
// optional archives, the data files, the movie and music archives each
// edition names, and a loose map.
func installReadNames() []string {
	names := append(RequiredArchives(), SfxArchive, patchArchive, "speech.res", fameFileName, WindowIconExecutable, "smackw32.dll", "Kids.alm")
	seen := map[string]bool{}
	for _, n := range names {
		seen[installpath.Fold(n)] = true
	}
	add := func(n string) {
		if !seen[installpath.Fold(n)] {
			seen[installpath.Fold(n)] = true
			names = append(names, n)
		}
	}
	for _, g := range []base.Game{base.GameROM1, base.GameROM2} {
		for _, c := range g.Edition().Cutscenes {
			add(c.Path)
		}
		add(GameMusic(base.Profile{Game: g}).Archive.Path)
	}
	return names
}

// TestInstallNamesResolveWhateverTheirCase builds lower-case, upper-case and
// mixed-case installs on a file system that refuses a name differing only by
// case. Every name opens through the resolver, the discovery accepts each
// install, and the exact name, as the readers built it before, does not open.
func TestInstallNamesResolveWhateverTheirCase(t *testing.T) {
	names := installReadNames()
	if len(names) < 15 {
		t.Fatalf("only %d install names: %q", len(names), names)
	}
	spellings := map[string]func(string) string{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"mixed": func(s string) string {
			parts := strings.Split(s, "/")
			for i, p := range parts {
				if (len(p)+i)%2 == 0 {
					parts[i] = strings.ToUpper(p)
				} else {
					parts[i] = strings.ToUpper(p[:1]) + strings.ToLower(p[1:])
				}
			}
			return strings.Join(parts, "/")
		},
	}
	for label, spell := range spellings {
		t.Run(label, func(t *testing.T) {
			fsys := fstest.MapFS{}
			for _, n := range names {
				fsys[spell(n)] = &fstest.MapFile{Data: []byte(n)}
			}
			files := installpath.NewFS(fsys)
			if !requiredArchivesIn(files) {
				t.Fatal("the discovery refused the install")
			}
			lost := 0
			for _, n := range names {
				b, err := files.ReadFile(n)
				if err != nil || string(b) != n {
					t.Fatalf("ReadFile(%q) = %q, %v", n, b, err)
				}
				if _, err := fsys.Open(n); errors.Is(err, fs.ErrNotExist) {
					lost++
				}
			}
			if label != "lower" && lost == 0 {
				t.Fatal("loss control: every exact name opened, so the install proves nothing")
			}
		})
	}
}

func TestDiscoveryRefusesTwoSpellingsOfOneArchive(t *testing.T) {
	fsys := fstest.MapFS{}
	for _, n := range RequiredArchives() {
		fsys[strings.ToUpper(n)] = &fstest.MapFile{}
	}
	fsys["Graphics.res"] = &fstest.MapFile{}
	files := installpath.NewFS(fsys)
	if requiredArchivesIn(files) {
		t.Fatal("the discovery accepted an install holding GRAPHICS.RES and Graphics.res")
	}
	if _, err := files.Path(GraphicsArchive); !errors.Is(err, installpath.ErrAmbiguous) {
		t.Fatalf("Path = %v, want both spellings reported", err)
	}
}
