package modrt

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
)

func TestHeldDataLoadsWithoutScriptAndMarksState(t *testing.T) {
	entries := makeMods(t, map[string]map[string]string{"held": {
		mod.ItemsFile: "[[change]]\nitem = \"Iron Pike\"\nprice = 17\n",
	}})
	res, err := Load(entries, "rom1-en", nil, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items.Changes) != 1 || res.Set.Mods[0].StateFree {
		t.Fatalf("held item data was not loaded or marked: %+v", res)
	}
}

func TestHeldKindsDetermineStateWithoutScriptCalls(t *testing.T) {
	for _, k := range mod.DataKinds() {
		t.Run(k.Name, func(t *testing.T) {
			res, err := Load(makeMods(t, map[string]map[string]string{"held": {k.Files[0]: "# empty\n"}}), "rom1-en", nil, Options{})
			if err != nil {
				t.Fatal(err)
			}
			if res.Set.Mods[0].StateFree == k.State {
				t.Fatalf("kind %s: state-free %t", k.Name, res.Set.Mods[0].StateFree)
			}
		})
	}
}

func TestHeldDataIsValidatedBeforeAnOptionalScript(t *testing.T) {
	res, err := Load(makeMods(t, map[string]map[string]string{"held": {
		mod.ItemsFile: "[[change]]\nitem = \"Iron Pike\"\nprice = 17\n",
		mod.EntryName: "def init(game, settings):\n    if False:\n        game.data.add(\"data/items.toml\")\n",
	}}), "rom1-en", nil, Options{})
	if err != nil || len(res.Items.Changes) != 1 {
		t.Fatalf("conditional script suppressed held data: %v", err)
	}
	_, err = Load(makeMods(t, map[string]map[string]string{"held": {mod.ItemsFile: "not TOML"}}), "rom1-en", nil, Options{})
	if err == nil || !strings.Contains(err.Error(), mod.ItemsFile) {
		t.Fatalf("held malformed data: %v", err)
	}
}

func TestExplicitAddConfirmsHeldFileOnce(t *testing.T) {
	for _, entry := range []string{mod.EntryName, "logic/custom.rule"} {
		for _, second := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/%t", entry, second), func(t *testing.T) {
				script := "def init(game, settings):\n    game.data.add(\"./data/items.toml\")\n"
				if second {
					script += "    game.data.add(\"DATA/ITEMS.TOML\")\n"
				}
				files := map[string]string{mod.ManifestName: "id = \"held\"\ntitle = \"Held\"\nversion = \"1\"\nentry = " + fmt.Sprintf("%q", entry) + "\n", mod.ItemsFile: "[[change]]\nitem = \"Iron Pike\"\nprice = 17\n", entry: script}
				res, err := Load(makeMods(t, map[string]map[string]string{"held": files}), "rom1-en", nil, Options{})
				if second {
					if err == nil || !strings.Contains(err.Error(), "the file is already loaded") {
						t.Fatalf("second add: %v", err)
					}
					return
				}
				if err != nil || len(res.Items.Changes) != 1 {
					t.Fatalf("first add: changes=%d, %v", len(res.Items.Changes), err)
				}
			})
		}
	}
}

func TestFixtureDataWithoutScriptsAndOriginalDigests(t *testing.T) {
	ids := fixtureMods(t)
	if len(ids) != 18 {
		t.Fatalf("%d fixtures, want 18", len(ids))
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			dir := filepath.Join(t.TempDir(), id)
			if err := os.CopyFS(dir, os.DirFS(filepath.Join("testdata", "mods", id))); err != nil {
				t.Fatal(err)
			}
			for p, b := range fixtureCompletions[id] {
				full := filepath.Join(dir, filepath.FromSlash(p))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(b), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			tree, _ := mod.OpenTree(dir)
			var paths []string
			if err := tree.WalkDir(func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if p != "." && strings.HasPrefix(d.Name(), ".") {
					if d.IsDir() {
						return fs.SkipDir
					}
					return nil
				}
				if !d.IsDir() && p != mod.ReadmeName {
					paths = append(paths, p)
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			slices.Sort(paths)
			h := sha256.New()
			for _, p := range paths {
				b, err := tree.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				switch path.Ext(p) {
				case ".toml", ".star", ".txt", ".md":
					b = []byte(strings.ReplaceAll(string(b), "\r\n", "\n"))
				}
				fmt.Fprintf(h, "%s\x00%d\x00", p, len(b))
				h.Write(b)
			}
			want := hex.EncodeToString(h.Sum(nil))
			got, err := tree.Digest()
			if err != nil || got != want {
				t.Fatalf("fixture digest moved from %s to %s: %v", want, got, err)
			}
			t.Logf("digest %s %s unchanged", id, got)
			entry := mod.Load(dir)
			if entry.Err != nil {
				t.Fatal(entry.Err)
			}
			manifestPath := filepath.Join(dir, mod.ManifestName)
			manifest, err := os.ReadFile(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			var dataOnly []string
			for _, line := range strings.Split(string(manifest), "\n") {
				if key, _, ok := strings.Cut(line, "="); ok && strings.TrimSpace(key) == "entry" {
					continue
				}
				dataOnly = append(dataOnly, line)
			}
			for _, base := range []string{"rom1-en", "rom1-ru"} {
				with, err := Load([]mod.Entry{entry}, base, nil, Options{})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(filepath.Join(dir, mod.EntryName), filepath.Join(dir, ".disabled-entry")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(manifestPath, []byte(strings.Join(dataOnly, "\n")), 0o644); err != nil {
					t.Fatal(err)
				}
				without, err := Load([]mod.Entry{mod.Load(dir)}, base, nil, Options{})
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(with.Data, without.Data) {
					t.Errorf("%s: Data moved without script", base)
				}
				if err := os.Rename(filepath.Join(dir, ".disabled-entry"), filepath.Join(dir, mod.EntryName)); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(manifestPath, manifest, 0o644); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
