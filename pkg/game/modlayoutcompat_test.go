package game

import (
	"crypto/sha256"
	"fmt"
	"image/color"
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// oldLayoutForms are the three layouts the editor and cmd/modmission wrote
// before the tree's layout: the package save, W's Add and the assembled
// mission. pages are written as files; "map" files take the editor's map
// and "png" files a speaker picture. moves is what the next save moves, rows
// the rewritten table and rewritten any other file the save rewrites. The
// editor opens the English map of the first mission, opened.
var oldLayoutForms = []struct {
	name, decl, opened string
	files              map[string]string
	moves              map[string]string
	rows               string
	rewritten          map[string]string
}{{
	name: "package", opened: "mission/map.alm",
	decl: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"mission/map.alm\"\ntexts = \"mission/text/{lang}\"\nafter = 20\nparty = [{character = 'npc22'}]\n",
	files: map[string]string{"mission/map.alm": "map", "mission/text/en/event01.txt": "en one",
		"mission/text/ru/event01.txt": "ru one", "mission/text/title.txt": "common"},
	moves: map[string]string{"mission/map.alm": "maps/map.alm", "mission/text/en/event01.txt": "maps/map.text/en/event01.txt",
		"mission/text/ru/event01.txt": "maps/map.text/ru/event01.txt", "mission/text/title.txt": "maps/map.text/title.txt"},
	rows: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"maps/map.alm\"\nafter = 20\nparty = [{character = 'npc22'}]\n",
}, {
	name: "added", opened: "mission/map.alm",
	decl: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"mission/map.alm\"\nafter = 20\nparty = [{character = 'npc22'}]\n" +
		"\n[[mission]]\nkey = 'second'\nnumber = 26\nmap = \"mission/map26/map.alm\"\ntexts = \"mission/map26/text/{lang}\"\nafter = 25\nparty = [{character = 'npc22'}]\n",
	files: map[string]string{"mission/map.alm": "map", "mission/map26/map.alm": "map",
		"mission/map26/text/en/event01.txt": "en second", "mission/map26/text/ru/event01.txt": "ru second"},
	moves: map[string]string{"mission/map.alm": "maps/map.alm", "mission/map26/map.alm": "maps/map-2.alm",
		"mission/map26/text/en/event01.txt": "maps/map-2.text/en/event01.txt", "mission/map26/text/ru/event01.txt": "maps/map-2.text/ru/event01.txt"},
	rows: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"maps/map.alm\"\nafter = 20\nparty = [{character = 'npc22'}]\n" +
		"\n[[mission]]\nkey = 'second'\nnumber = 26\nmap = \"maps/map-2.alm\"\nafter = 25\nparty = [{character = 'npc22'}]\n",
}, {
	name: "assembled", opened: "mission/en/map.alm",
	decl: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"mission/{lang}/map.alm\"\ntexts = \"mission/{lang}/text\"\nafter = 20\nparty = [{character = 'npc22'}]\n",
	files: map[string]string{"mission/en/map.alm": "map", "mission/ru/map.alm": "map",
		"mission/en/text/event01.txt": "en one", "mission/ru/text/event01.txt": "ru one",
		mod.SpeakersFile:    "[[speaker]]\nid = 20001\nname = 'Karl'\npicture = 'speakers/karl.png'\nmap = 'mission/en/map.alm' # bound\nunit = 7\n",
		"speakers/karl.png": "png"},
	rewritten: map[string]string{mod.SpeakersFile: "[[speaker]]\nid = 20001\nname = 'Karl'\npicture = 'speakers/karl.png'\nmap = \"maps/en/map.alm\" # bound\nunit = 7\n"},
	moves: map[string]string{"mission/en/map.alm": "maps/en/map.alm", "mission/ru/map.alm": "maps/ru/map.alm",
		"mission/en/text/event01.txt": "maps/en/map.text/en/event01.txt", "mission/ru/text/event01.txt": "maps/ru/map.text/ru/event01.txt"},
	rows: "[[mission]]\nkey = 'first'\nnumber = 25\nmap = \"maps/{lang}/map.alm\"\ntexts = \"maps/{lang}/map.text/{lang}\"\nafter = 20\nparty = [{character = 'npc22'}]\n",
}}

// layoutView is what the game reads of each mission of the mod e in each
// language: its key, map digest, pages and other-language page names, and
// the load's warnings.
func layoutView(t *testing.T, e mod.Entry) (view, warnings []string) {
	t.Helper()
	for _, lang := range []string{"en", "ru"} {
		res, err := modrt.Load([]mod.Entry{e}, "rom1-"+lang, nil, modrt.Options{})
		if err != nil {
			t.Fatalf("%s %s: %v", e.Dir, lang, err)
		}
		warnings = append(warnings, res.Warnings...)
		for _, m := range res.Missions.Missions {
			other := slices.Sorted(maps.Keys(m.OtherLanguageTexts))
			view = append(view, fmt.Sprintf("%s %s map=%x pages=%s other=%s", lang, m.Key, sha256.Sum256(m.MapBytes), pageNames(m.TextFiles), strings.Join(other, ",")))
		}
	}
	return view, warnings
}

// oldPaths are the warnings of a load that name an old mission path.
func oldPaths(warnings []string) []string {
	var out []string
	for _, w := range warnings {
		if strings.Contains(w, "uses an old path") {
			out = append(out, w)
		}
	}
	return out
}

// packedEntry publishes the mod folder dir as the editor does and loads the
// packed file.
func packedEntry(t *testing.T, dir string) mod.Entry {
	t.Helper()
	file, err := PackMod(dir, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	e := mod.Load(file)
	if e.Err != nil {
		t.Fatal(e.Err)
	}
	return e
}

// selectProject selects the handwritten mod at dir as the game's mod and
// opens its map file opened, with the mod as the editor's project.
func selectProject(t *testing.T, f *testPlayFixture, e *ui.MapEditor, dir, opened string) *mapInspectionEdits {
	t.Helper()
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	if err := e.Open(filepath.Join(dir, filepath.FromSlash(opened))); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Project == nil || s.inspector.mod.owned {
		t.Fatal("the mod opens as no handwritten project")
	}
	return s
}

// A mod in each older layout opens, checks, test plays and packs as it
// stands; its next save through the project writer moves exactly its
// mission files into the tree's layout and rewrites only their keys, and
// every language reads the same maps and pages before and after.
func TestAnOlderLayoutModPlaysAndItsNextSaveWritesTheTree(t *testing.T) {
	for _, form := range oldLayoutForms {
		t.Run(form.name, func(t *testing.T) {
			f, e := workspaceFixture(t)
			dir := writeEditorMod(t, t.TempDir(), form.decl)
			writeTestPlayInput(t, filepath.Join(dir, "notes.txt"), []byte("author notes"))
			for name, body := range form.files {
				file := filepath.Join(dir, filepath.FromSlash(name))
				b := []byte(body)
				switch body {
				case "map":
					b = e.Document().SourceBytes
				case "png":
					writeTestPlayInput(t, file, nil)
					b = speakerPNG(t, file, color.RGBA{R: 9, G: 8, B: 7, A: 255})
				}
				writeTestPlayInput(t, file, b)
			}
			before := tree(t, dir)
			view, warnings := layoutView(t, mod.Load(dir))
			if len(oldPaths(warnings)) == 0 {
				t.Fatalf("the check names no old path: %q", warnings)
			}
			if packed, _ := layoutView(t, packedEntry(t, dir)); !reflect.DeepEqual(packed, view) {
				t.Fatalf("the packed mod reads\n%q\nthe folder\n%q", packed, view)
			}
			s := selectProject(t, f, e, dir, form.opened)
			if _, err := s.TestPlay(); err != nil || len(f.started) != 1 {
				t.Fatalf("F5: %v, started %d", err, len(f.started))
			}
			played, _ := layoutView(t, mod.Load(filepath.Join(f.profile, TestPlayModID, "ambush")))
			if !reflect.DeepEqual(played, view) || !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatalf("F5 played\n%q\nthe mod reads\n%q\nor changed the mod", played, view)
			}
			if err := workspaceField(e, "package", 8, "73"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			for name, body := range before {
				to, moved := form.moves[name]
				switch {
				case name == mod.MissionsFile:
				case moved && (after[to] != body || after[name] != ""):
					t.Errorf("%s did not move to %s", name, to)
				case form.rewritten[name] != "":
					if after[name] != form.rewritten[name] {
						t.Errorf("%s is\n%s\nwant\n%s", name, after[name], form.rewritten[name])
					}
				case !moved && after[name] != body:
					t.Errorf("%s changed", name)
				}
			}
			if len(after) != len(before) {
				t.Errorf("the save left %d files of %d: %q", len(after), len(before), slices.Sorted(maps.Keys(after)))
			}
			// The table writer adds the new key at the end of its row.
			want := form.rows + "gold = 73\n"
			if at := strings.Index(form.rows, "[[mission]]\nkey = 'second'"); at > 0 {
				want = form.rows[:at] + "gold = 73\n" + form.rows[at:]
			}
			if after[mod.MissionsFile] != want {
				t.Errorf("the table is\n%s\nwant\n%s", after[mod.MissionsFile], want)
			}
			if _, err := os.Stat(filepath.Join(dir, "mission")); !os.IsNotExist(err) {
				t.Errorf("the emptied mission folder stays: %v", err)
			}
			saved, warnings := layoutView(t, mod.Load(dir))
			if !reflect.DeepEqual(saved, view) || len(oldPaths(warnings)) != 0 {
				t.Fatalf("after the save the mod reads\n%q\nwant\n%q\nwarnings %q", saved, view, warnings)
			}
		})
	}
}

// Undo and redo across the save that moves an older-layout mod into the tree
// reach states that name the moved files: the next save writes no new stem
// and leaves no unread file, and the mod, and F5's copy of it, read the same
// map and pages in both languages. In "redo" the undone edit waits on the redo
// side while the save moves the files.
func TestUndoAndRedoAcrossAMigratingSaveKeepWhatTheMissionReads(t *testing.T) {
	for _, run := range []struct {
		name  string
		steps []string
		gold  string // the table's gold line at the end, "" for none
	}{
		{"undo", []string{"gold=73", "save", "undo", "save", "redo", "save"}, "gold = 73"},
		{"redo", []string{"gold=73", "gold=74", "undo", "save", "redo", "save", "undo", "undo", "save"}, ""},
	} {
		steps := run.steps
		t.Run(run.name, func(t *testing.T) {
			form := oldLayoutForms[0]
			f, e := workspaceFixture(t)
			dir := writeEditorMod(t, t.TempDir(), form.decl)
			for name, body := range form.files {
				b := []byte(body)
				if body == "map" {
					b = e.Document().SourceBytes
				}
				writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), b)
			}
			view, _ := layoutView(t, mod.Load(dir))
			s := selectProject(t, f, e, dir, form.opened)
			var files []string
			for _, step := range steps {
				var err error
				switch {
				case strings.HasPrefix(step, "gold="):
					_, err = s.SetWorkspaceField("package", "gold", strings.TrimPrefix(step, "gold="))
				case step == "save":
					if _, err = s.SaveIntoMod(); err == nil && files == nil {
						files = slices.Sorted(maps.Keys(tree(t, dir)))
					}
				default:
					err = e.Key(step)
				}
				if err != nil {
					t.Fatalf("%s: %v", step, err)
				}
				if step != "save" {
					continue
				}
				if got := slices.Sorted(maps.Keys(tree(t, dir))); !slices.Equal(got, files) {
					t.Fatalf("after %q the mod holds %q, want %q", steps, got, files)
				}
				if got, _ := layoutView(t, mod.Load(dir)); !reflect.DeepEqual(got, view) {
					t.Fatalf("the mod reads\n%q\nwant\n%q", got, view)
				}
			}
			table := tree(t, dir)[mod.MissionsFile]
			if gold := strings.Contains(table, "gold"); gold != (run.gold != "") || !strings.Contains(table, run.gold) || !strings.Contains(table, `map = "maps/map.alm"`) {
				t.Fatalf("the table is\n%s\nwant %q", table, run.gold)
			}
			if _, err := s.TestPlay(); err != nil {
				t.Fatal(err)
			}
			if played, _ := layoutView(t, mod.Load(filepath.Join(f.profile, TestPlayModID, "ambush"))); !reflect.DeepEqual(played, view) {
				t.Fatalf("F5 played\n%q\nwant\n%q", played, view)
			}
		})
	}
}

// An extended map saves into a mod package as maps/<stem>.almx: the mod
// loader admits it, F5 plays it and the packed mod reads it as the folder
// does. A handwritten row that names it .alm is renamed on the next save.
func TestAnExtendedMapSavesIntoAModAsAlmx(t *testing.T) {
	f, e := workspaceFixture(t)
	b := extendedScriptMap(t, nil, 1)
	loose := filepath.Join(t.TempDir(), "crypt.alm")
	writeTestPlayInput(t, loose, b)
	if e.ModWindowOpen() {
		_ = e.Key("escape")
	}
	if err := e.Open(loose); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	folder := filepath.Join(t.TempDir(), s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	got := tree(t, folder)
	if _, plain := got["maps/crypt.alm"]; got["maps/crypt.almx"] != string(b) || plain {
		t.Fatalf("the package holds %q", slices.Sorted(maps.Keys(got)))
	}
	m := loadWorkspaceMod(t, f.root, folder)
	if m.Map != "maps/crypt.almx" || string(m.MapBytes) != string(b) {
		t.Fatalf("the loader reads %s", m.Map)
	}
	for _, r := range f.x.CheckModMap(m, filepath.Join(folder, "maps", "crypt.almx")) {
		if r.Rule == ruleExtendedName {
			t.Fatalf("the check warns of the name: %+v", r)
		}
	}
	view, _ := layoutView(t, mod.Load(folder))
	if packed, _ := layoutView(t, packedEntry(t, folder)); !reflect.DeepEqual(packed, view) {
		t.Fatalf("the packed mod reads\n%q\nthe folder\n%q", packed, view)
	}
	if _, err := s.TestPlay(); err != nil || len(f.started) != 1 {
		t.Fatalf("F5: %v, started %d", err, len(f.started))
	}
	if played := tree(t, filepath.Join(f.profile, TestPlayModID)); !slices.ContainsFunc(slices.Collect(maps.Keys(played)), func(name string) bool {
		return strings.HasSuffix(name, "/maps/crypt.almx") && played[name] == string(b)
	}) {
		t.Fatalf("F5 wrote %q", slices.Sorted(maps.Keys(played)))
	}

	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'maps/crypt.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n")
	writeTestPlayInput(t, filepath.Join(dir, "maps", "crypt.alm"), b)
	writeTestPlayInput(t, filepath.Join(dir, "maps", "crypt.text", "en", "event01.txt"), []byte("en one"))
	s = selectProject(t, f, e, dir, "maps/crypt.alm")
	if err := workspaceField(e, "package", 8, "73"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	got = tree(t, dir)
	if _, plain := got["maps/crypt.alm"]; got["maps/crypt.almx"] != string(b) || plain || got["maps/crypt.text/en/event01.txt"] != "en one" ||
		!strings.Contains(got[mod.MissionsFile], `map = "maps/crypt.almx"`) {
		t.Fatalf("the rename left %q\n%s", slices.Sorted(maps.Keys(got)), got[mod.MissionsFile])
	}
}

// A mod in the tree's layout, opened and saved without an edit, keeps every
// byte and name it had.
func TestATreeLayoutModSavedWithoutAnEditKeepsItsBytes(t *testing.T) {
	f, e := workspaceFixture(t)
	decl := "# the author's table\r\n[[mission]]\r\nkey = 'first'\r\nnumber = 25\r\nmap = 'maps/{lang}/first.alm' # per language\r\nafter = 20\r\nparty = [{character = 'npc22'}]\r\n"
	dir := writeEditorMod(t, t.TempDir(), decl)
	for name, body := range map[string]string{"maps/en/first.alm": "map", "maps/ru/first.alm": "map",
		"maps/en/first.text/en/event01.txt": "en one", "maps/ru/first.text/ru/event01.txt": "ru one",
		"maps/en/first.text/title.txt": "common", "notes.txt": "author notes"} {
		b := []byte(body)
		if body == "map" {
			b = e.Document().SourceBytes
		}
		writeTestPlayInput(t, filepath.Join(dir, filepath.FromSlash(name)), b)
	}
	before := tree(t, dir)
	if _, err := selectProject(t, f, e, dir, "maps/en/first.alm").SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	if after := tree(t, dir); !reflect.DeepEqual(after, before) {
		t.Fatalf("an unedited save changed the mod:\n%q\nwant\n%q", after, before)
	}
}
