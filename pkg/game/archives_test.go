package game_test

// Tests for the startup archive opening. Every fixture is a synthetic archive
// written to a temp dir; no game install is read.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/game"
	"againrom/pkg/vfs"
)

// archiveBytes is a small valid .res archive.
func archiveBytes(t *testing.T) []byte {
	t.Helper()
	return synth.Archive([]synth.File{{Path: "a.bin", Data: []byte("hello")}})
}

// worldArchiveBytes is a .res archive holding a definition table the walk
// accepts, at the address the front-end resolves it by. Its collections are
// empty: startup requires a table that reads, not one with rows in it.
func worldArchiveBytes(t *testing.T) []byte {
	t.Helper()
	return synth.Archive([]synth.File{
		{Path: "data/data.bin", Data: synth.DataBinUnitsTable(nil, nil)},
	})
}

// installDir lays out an asset root holding the named archives. A name mapped to
// nil is left out; a name mapped to non-nil bytes is written verbatim, so a
// caller can plant a file that exists but is not an archive.
func installDir(t *testing.T, files map[string][]byte) string {
	t.Helper()
	dir := t.TempDir()
	for name, data := range files {
		if data == nil {
			continue
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	return dir
}

// fullInstall is every required archive present and valid.
func fullInstall(t *testing.T) map[string][]byte {
	t.Helper()
	b := archiveBytes(t)
	return map[string][]byte{
		game.MainArchive:     b,
		game.GraphicsArchive: b,
		game.ScenarioArchive: b,
		game.WorldArchive:    worldArchiveBytes(t),
		game.MoviesArchive:   b,
	}
}

func TestOpenArchives(t *testing.T) {
	t.Run("a complete install opens all five", func(t *testing.T) {
		dir := installDir(t, fullInstall(t))

		a, err := game.OpenArchives(dir)
		if err != nil {
			t.Fatalf("OpenArchives: %v", err)
		}
		if a.Root != dir {
			t.Errorf("Root = %q, want %q", a.Root, dir)
		}
		// The teardown's shape: an install opens into TWO FILESYSTEMS and nothing
		// else. The three `*res.Archive` handles the consumers were flipped off
		// one at a time are gone, so there is no second way to reach a container's
		// bytes left in this type.
		if a.Containers == nil || a.Loose == nil {
			t.Errorf("filesystems: containers=%v loose=%v, want both non-nil",
				a.Containers != nil, a.Loose != nil)
		}
	})

	// Each required archive, missing on its own. graphics.res and movies.res
	// get their own cases below as well, because they are the two whose
	// necessity is not self-evident.
	for _, missing := range []string{
		game.MainArchive, game.GraphicsArchive, game.ScenarioArchive, game.WorldArchive, game.MoviesArchive,
	} {
		t.Run("a missing "+missing+" is reported by path", func(t *testing.T) {
			files := fullInstall(t)
			files[missing] = nil
			dir := installDir(t, files)

			a, err := game.OpenArchives(dir)
			if err == nil {
				t.Fatalf("OpenArchives succeeded with %s missing", missing)
			}
			if a != nil {
				t.Errorf("OpenArchives returned a non-nil *Archives alongside an error")
			}
			want := filepath.Join(dir, missing)
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not name the missing archive %q", err, want)
			}
		})
	}

	t.Run("graphics.res is required even though the menu never reads it", func(t *testing.T) {
		// Stated as its own case because it is a deliberate requirement rather
		// than a side effect: an install missing a piece the application will
		// need should say so before the user picks a map. A lazy open would also
		// let the headless check pass on an install the windowed run then fails.
		files := fullInstall(t)
		files[game.GraphicsArchive] = nil
		dir := installDir(t, files)

		if _, err := game.OpenArchives(dir); err == nil {
			t.Fatalf("an install with main.res and scenario.res but no graphics.res was accepted")
		}
	})

	t.Run("movies.res is required even though the menu never reads it", func(t *testing.T) {
		// Same deliberate requirement as graphics.res, for the shop merchant's
		// own static picture (SHOP-MERCHANT-046, 1009): an install missing it
		// should say so before the user opens a shop, not after.
		files := fullInstall(t)
		files[game.MoviesArchive] = nil
		dir := installDir(t, files)

		if _, err := game.OpenArchives(dir); err == nil {
			t.Fatalf("an install with every other archive but no movies.res was accepted")
		}
	})

	t.Run("with two missing, the first in the fixed order is reported", func(t *testing.T) {
		// The order is main.res, graphics.res, scenario.res, world.res,
		// movies.res. Fixing it is what keeps the windowed and headless modes
		// from naming different faults on the same broken install, and
		// world.res and movies.res are LAST so that each joining moved none of
		// the pairs already pinned here.
		cases := []struct {
			label   string
			missing []string
			want    string
		}{
			{"main and graphics", []string{game.MainArchive, game.GraphicsArchive}, game.MainArchive},
			{"graphics and scenario", []string{game.GraphicsArchive, game.ScenarioArchive}, game.GraphicsArchive},
			{"main and scenario", []string{game.MainArchive, game.ScenarioArchive}, game.MainArchive},
			{"all three", []string{game.MainArchive, game.GraphicsArchive, game.ScenarioArchive}, game.MainArchive},
			{"scenario and world", []string{game.ScenarioArchive, game.WorldArchive}, game.ScenarioArchive},
			{"world alone", []string{game.WorldArchive}, game.WorldArchive},
			{"world and movies", []string{game.WorldArchive, game.MoviesArchive}, game.WorldArchive},
			{"movies alone", []string{game.MoviesArchive}, game.MoviesArchive},
		}
		for _, tc := range cases {
			t.Run(tc.label, func(t *testing.T) {
				files := fullInstall(t)
				for _, m := range tc.missing {
					files[m] = nil
				}
				dir := installDir(t, files)

				_, err := game.OpenArchives(dir)
				if err == nil {
					t.Fatalf("OpenArchives succeeded with %v missing", tc.missing)
				}
				if !strings.Contains(err.Error(), filepath.Join(dir, tc.want)) {
					t.Errorf("error %q does not name %q, the first missing archive in order", err, tc.want)
				}
				// ...and names only that one, so the report is unambiguous.
				for _, m := range tc.missing {
					if m == tc.want {
						continue
					}
					if strings.Contains(err.Error(), filepath.Join(dir, m)) {
						t.Errorf("error %q also names %q; it should stop at the first failure", err, m)
					}
				}
			})
		}
	})

	t.Run("a file that is not an archive is reported by path", func(t *testing.T) {
		files := fullInstall(t)
		files[game.ScenarioArchive] = []byte("this is not a res archive")
		dir := installDir(t, files)

		a, err := game.OpenArchives(dir)
		if err == nil {
			t.Fatalf("OpenArchives accepted a non-archive file")
		}
		if a != nil {
			t.Errorf("OpenArchives returned a non-nil *Archives alongside an error")
		}
		if want := filepath.Join(dir, game.ScenarioArchive); !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not name %q", err, want)
		}
	})

	t.Run("a nonexistent root is an error", func(t *testing.T) {
		if _, err := game.OpenArchives(filepath.Join(t.TempDir(), "no-such-dir")); err == nil {
			t.Errorf("OpenArchives succeeded on a nonexistent root")
		}
	})
}

// TestOpenGraphicsOverOneArchive is the single-archive entry point the
// developer front-ends take. It replaces TestOpenTileset, which tested the
// wrapper the teardown removed: with the transitional `*res.Archive` gone,
// OpenGraphics' remaining second return is the filesystem, and that is what
// its one call site wanted all along.
func TestOpenGraphicsOverOneArchive(t *testing.T) {
	t.Run("one open yields the filesystem and the tileset over that archive", func(t *testing.T) {
		dir := installDir(t, map[string][]byte{game.GraphicsArchive: archiveBytes(t)})
		path := filepath.Join(dir, game.GraphicsArchive)

		containers, set, err := game.OpenGraphics(path)
		if err != nil {
			t.Fatalf("OpenGraphics: %v", err)
		}
		if containers == nil || set == nil {
			t.Fatalf("OpenGraphics: containers=%v tiles=%v, want both non-nil",
				containers != nil, set != nil)
		}
		// The fixture archive holds no terrain strips, so nothing is loaded. That
		// is not an error: a tileset with absent slots draws placeholders rather
		// than failing, which is the shipped behaviour.
		if set.Loaded != 0 {
			t.Errorf("Loaded = %d on an archive with no tile strips, want 0", set.Loaded)
		}
		// The filesystem is over THAT archive, not merely non-nil: the entry the
		// fixture wrote reads back through it under the identity derived from the
		// host name, which is the only claim that makes "one open serves both"
		// mean anything.
		identity, err := vfs.Identity(game.GraphicsArchive)
		if err != nil {
			t.Fatalf("vfs.Identity(%q): %v", game.GraphicsArchive, err)
		}
		got, err := containers.ReadFile(identity + "/a.bin")
		if err != nil {
			t.Fatalf("read the fixture entry through the returned filesystem: %v", err)
		}
		if string(got) != "hello" {
			t.Errorf("entry bytes = %q, want the fixture's own %q", got, "hello")
		}
	})

	t.Run("a missing archive is reported with the contract wording", func(t *testing.T) {
		// `open <path>: <err>` is what the standalone viewer has always printed,
		// and that text is frozen.
		path := filepath.Join(t.TempDir(), "absent.res")

		containers, set, err := game.OpenGraphics(path)
		if err == nil {
			t.Fatalf("OpenGraphics succeeded on a missing archive")
		}
		if containers != nil || set != nil {
			t.Errorf("OpenGraphics returned containers=%v tiles=%v alongside an error",
				containers != nil, set != nil)
		}
		if !strings.HasPrefix(err.Error(), "open ") {
			t.Errorf("error %q does not start with %q", err, "open ")
		}
		if !strings.Contains(err.Error(), path) {
			t.Errorf("error %q does not name the path %q", err, path)
		}
	})
}
