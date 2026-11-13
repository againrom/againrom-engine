package game

import (
	"bytes"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/words"
)

// keptLanguages are the editor languages a refusal is read in.
var keptLanguages = map[string]string{"en": "english", "ru": "russian"}

// refusalLead is the words of the word-book line id before its first verb.
func refusalLead(x *MapInspector, id string) string {
	lead, _, _ := strings.Cut(words.For(x.pageLanguage()).Text(id), "%")
	return lead
}

// recordMapWithStray is bridge.almx holding its four pages inside, with a
// file that is no page in its texts folder.
func recordMapWithStray(t *testing.T) (string, []byte) {
	t.Helper()
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	original := withInside(t, insideMap(t, true), insidePages())
	writeTestPlayInput(t, source, original)
	writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "readme.txt"), []byte("notes"))
	return source, original
}

// A map that holds its pages inside is never saved without them. With the
// switch off and a texts folder the editor cannot read, the save in place is
// refused in the editor's language and writes nothing: the map keeps its
// record, the folder is as it was, and the game still reads every page. A
// save under a new name whose texts folder is taken is refused the same way
// and leaves no new map.
func TestSwitchOffSaveNeverLosesTheMapsTexts(t *testing.T) {
	leads := map[string]string{}
	for lang, profile := range keptLanguages {
		x := insideInspector(t)
		x.archives.Base.Profile.Language = profile
		source, original := recordMapWithStray(t)
		dir := filepath.Dir(source)
		s := insideHost(t, x, source)
		if !s.TextsInside() || s.Dirty() {
			t.Fatalf("%s: opened with the switch %v, changed %v", lang, s.TextsInside(), s.Dirty())
		}
		if err := s.SetTextsInside(false); err != nil || !s.Dirty() {
			t.Fatalf("%s: switching off: %v, changed %v", lang, err, s.Dirty())
		}
		lead := refusalLead(x, "editor.save.textsinside.kept")
		leads[lang] = lead
		for range 2 {
			target, err := s.SaveInPlace()
			if err == nil || target != "" || !strings.Contains(err.Error(), lead) || !strings.Contains(err.Error(), "readme.txt") {
				t.Fatalf("%s: the save answered %q, %v; want the refusal %q", lang, target, err, lead)
			}
			if b, _ := os.ReadFile(source); !bytes.Equal(b, original) {
				t.Fatalf("%s: the refused save changed the map", lang)
			}
			if got := tree(t, filepath.Join(dir, "bridge.text")); len(got) != 1 || got["readme.txt"] != "notes" {
				t.Fatalf("%s: the refused save changed the folder: %v", lang, got)
			}
		}
		for _, play := range []string{"en", "ru"} {
			files, _, err := mod.Mission{Map: "bridge.almx"}.ReadTexts(dir, play)
			if err != nil || !bytes.Equal(files["briefing.txt"], insidePages()[play+"/briefing.txt"]) || !bytes.Equal(files["event01.txt"], insidePages()[play+"/event01.txt"]) {
				t.Fatalf("%s: the game reads %q in %s: %v", lang, files, play, err)
			}
		}
		if !s.Dirty() {
			t.Fatalf("%s: a refused save left the map unchanged", lang)
		}

		// The folder of the new name is taken: the folder writer refuses, and
		// the map is not written alone.
		out := t.TempDir()
		if err := os.Remove(filepath.Join(dir, "bridge.text", "readme.txt")); err != nil {
			t.Fatal(err)
		}
		writeTestPlayInput(t, filepath.Join(out, "copy.text", "en", "briefing.txt"), []byte("another map's"))
		target, err := s.SaveAs(filepath.Join(out, "copy.almx"))
		if err == nil || target != "" || !strings.Contains(err.Error(), lead) {
			t.Fatalf("%s: the save under a taken folder answered %q, %v", lang, target, err)
		}
		if got := tree(t, out); len(got) != 1 || got["copy.text/en/briefing.txt"] != "another map's" {
			t.Fatalf("%s: the refused save wrote %v", lang, got)
		}

		// With the cause gone the same save writes the map without the record
		// and every page into its folder.
		if _, err := s.SaveInPlace(); err != nil || s.Dirty() {
			t.Fatalf("%s: the save after the fix: %v, changed %v", lang, err, s.Dirty())
		}
		b, _ := os.ReadFile(source)
		if _, held, _ := alm.TextsOf(b); held {
			t.Fatalf("%s: the switch-off save kept the record", lang)
		}
		if got := treeBytes(t, filepath.Join(dir, "bridge.text")); !maps.EqualFunc(got, insidePages(), bytes.Equal) {
			t.Fatalf("%s: the folder holds %q", lang, got)
		}
	}
	if leads["en"] == leads["ru"] || leads["en"] == "" || leads["ru"] == "" {
		t.Fatalf("the refusal reads %q in EN and %q in RU", leads["en"], leads["ru"])
	}
}

// A map without the record keeps the save it had: an unreadable texts folder
// is a note, and the map is written.
func TestSwitchOffSaveOfAMapWithoutTheRecordStillNotes(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	writeTestPlayInput(t, source, insideMap(t, true))
	writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "readme.txt"), []byte("notes"))
	s := insideHost(t, x, source)
	if target, err := s.SaveInPlace(); err != nil || target != source || s.saveNote == "" {
		t.Fatalf("the save answered %q, %v, note %q", target, err, s.saveNote)
	}
}

// A folder swap that fails after the map was replaced returns the map that
// held its pages inside. Windows refuses to rename a folder while a file in
// it is open; elsewhere the swap does not fail this way.
func TestFailedFolderSwapReturnsTheMapWithItsTexts(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("an open file holds its folder in place only on Windows")
	}
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	original := withInside(t, insideMap(t, true), insidePages())
	writeTestPlayInput(t, source, original)
	page := filepath.Join(dir, "bridge.text", "en", "event02.txt")
	writeTestPlayInput(t, page, []byte("Hold on."))
	s := insideHost(t, x, source)
	if err := s.SetTextsInside(false); err != nil {
		t.Fatal(err)
	}
	held, err := os.Open(page)
	if err != nil {
		t.Fatal(err)
	}
	target, err := s.SaveInPlace()
	_ = held.Close()
	if err == nil {
		t.Skipf("the folder swap did not fail with a page open (saved %s)", target)
	}
	if !strings.Contains(err.Error(), refusalLead(x, "editor.save.textsinside.kept")) {
		t.Fatalf("the failed swap answered %v", err)
	}
	if b, _ := os.ReadFile(source); !bytes.Equal(b, original) {
		t.Fatal("the failed swap left the map without its texts")
	}
	if got := tree(t, dir); len(got) != 2 || got["bridge.text/en/event02.txt"] != "Hold on." {
		t.Fatalf("the failed swap left %v", got)
	}
}

// Saved with the switch on, a map that with its pages inside is over the
// size the game reads a map's record from is refused in the editor's
// language and nothing is written; with the switch off the same pages save
// into the folder.
func TestSwitchOnSaveRefusesAMapTheGameReadsNoPageFrom(t *testing.T) {
	leads := map[string]string{}
	for lang, profile := range keptLanguages {
		x := insideInspector(t)
		x.archives.Base.Profile.Language = profile
		dir := t.TempDir()
		source := filepath.Join(dir, "bridge.almx")
		writeTestPlayInput(t, source, insideMap(t, true))
		big := bytes.Repeat([]byte("a"), 600<<10)
		writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "en", "briefing.txt"), big)
		writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "ru", "briefing.txt"), big)
		writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "en", "event01.txt"), []byte("They come."))
		s := insideHost(t, x, source)
		if err := s.SetTextsInside(true); err != nil {
			t.Fatal(err)
		}
		lead := refusalLead(x, "editor.save.textsinside.size")
		leads[lang] = lead
		out := t.TempDir()
		target, err := s.SaveAs(filepath.Join(out, "big.almx"))
		if err == nil || target != "" || !strings.Contains(err.Error(), lead) || !strings.Contains(err.Error(), refusalLead(x, "editor.save.noencode")) {
			t.Fatalf("%s: the save answered %q, %v; want the refusal %q", lang, target, err, lead)
		}
		if got := tree(t, out); len(got) != 0 {
			t.Fatalf("%s: the refused save wrote %d files", lang, len(got))
		}
		if _, err := s.SaveInPlace(); err == nil || !strings.Contains(err.Error(), lead) {
			t.Fatalf("%s: the save in place answered %v", lang, err)
		}
		if b, _ := os.ReadFile(source); !bytes.Equal(b, insideMap(t, true)) {
			t.Fatalf("%s: the refused save changed the map", lang)
		}
		if err := s.SetTextsInside(false); err != nil {
			t.Fatal(err)
		}
		if _, err := s.SaveAs(filepath.Join(out, "big.almx")); err != nil {
			t.Fatalf("%s: the switch-off save: %v", lang, err)
		}
		files, _, err := mod.Mission{Map: "big.almx"}.ReadTexts(out, "en")
		if err != nil || len(files) != 2 || len(files["briefing.txt"]) != len(big) {
			t.Fatalf("%s: the game reads %d pages from the folder: %v", lang, len(files), err)
		}
	}
	if leads["en"] == leads["ru"] || leads["en"] == "" || leads["ru"] == "" {
		t.Fatalf("the refusal reads %q in EN and %q in RU", leads["en"], leads["ru"])
	}
}

// A map just under the size with its pages inside saves, and the game reads
// its pages: the refusal is the reader's own bound, not a lower one.
func TestSwitchOnSaveWritesAMapTheGameReads(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	writeTestPlayInput(t, source, insideMap(t, true))
	writeTestPlayInput(t, filepath.Join(dir, "bridge.text", "en", "briefing.txt"), bytes.Repeat([]byte("a"), 900<<10))
	s := insideHost(t, x, source)
	if err := s.SetTextsInside(true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SaveInPlace(); err != nil {
		t.Fatal(err)
	}
	if st, _ := os.Stat(source); st.Size() > mod.MaxFileBytes {
		t.Fatalf("the map is %d bytes", st.Size())
	}
	if err := os.RemoveAll(filepath.Join(dir, "bridge.text")); err != nil {
		t.Fatal(err)
	}
	if files, _, err := (mod.Mission{Map: "bridge.almx"}).ReadTexts(dir, "en"); err != nil || len(files["briefing.txt"]) != 900<<10 {
		t.Fatalf("the game reads %d pages: %v", len(files), err)
	}
}

// Opening a mod that shows no mission of its own re-inspects the open map
// from its saved bytes, which hold no record; the switch stays as it was.
func TestOpeningAModKeepsTheSwitch(t *testing.T) {
	x := insideInspector(t)
	dir := t.TempDir()
	source := filepath.Join(dir, "bridge.almx")
	writeTestPlayInput(t, source, withInside(t, insideMap(t, true), insidePages()))
	s := insideHost(t, x, source)
	if !s.TextsInside() {
		t.Fatal("the map opened with the switch off")
	}
	d, err := s.openWorkspace(writeEditorMod(t, t.TempDir(), ""))
	if err != nil {
		t.Fatal(err)
	}
	next := d.Edits.(*mapInspectionEdits)
	if next == s || !next.TextsInside() || next.Dirty() {
		t.Fatalf("after the mod opened the switch is %v, changed %v", next.TextsInside(), next.Dirty())
	}
}

// keptModMissions are two missions of a mod, each on its own extended map.
const keptModMissions = "[[mission]]\nkey='first'\nnumber=25\nmap='maps/first.almx'\nafter=20\nparty=[{character='npc22'}]\n" +
	"[[mission]]\nkey='second'\nnumber=26\nmap='maps/second.almx'\nafter=25\nparty=[{character='npc22'}]\n"

// keptMod is a mod of two missions; first is the bytes of the first map.
func keptMod(t *testing.T, first []byte) (*mapInspectionEdits, string) {
	t.Helper()
	f, e := workspaceFixture(t)
	_ = f
	dir := writeEditorMod(t, t.TempDir(), keptModMissions)
	writeTestPlayInput(t, filepath.Join(dir, "maps", "first.almx"), first)
	writeTestPlayInput(t, filepath.Join(dir, "maps", "second.almx"), insideMap(t, true))
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	return e.Document().Edits.(*mapInspectionEdits), dir
}

// A mod's draft of a map never drops the pages the map holds inside. When
// the record cannot be made the draft stays as it was; with the switch off
// the record's pages go into the drafts of the map's folder with the map.
func TestModDraftNeverLosesTheMapsTexts(t *testing.T) {
	t.Run("the record cannot be made", func(t *testing.T) {
		s, dir := keptMod(t, insideMap(t, true))
		big := bytes.Repeat([]byte("a"), 600<<10)
		writeTestPlayInput(t, filepath.Join(dir, "maps", "first.text", "en", "briefing.txt"), big)
		writeTestPlayInput(t, filepath.Join(dir, "maps", "first.text", "ru", "briefing.txt"), big)
		if err := s.SetTextsInside(true); err != nil || !s.Dirty() {
			t.Fatalf("switching on: %v, changed %v", err, s.Dirty())
		}
		if _, err := s.withTexts(s.model.Bytes()); err == nil {
			t.Fatal("the fixture's record fits")
		}
		if _, err := s.selectWorkspaceMission(2); err != nil {
			t.Fatal(err)
		}
		if b, drafted := s.workspace.Project.Files["maps/first.almx"]; drafted {
			t.Fatalf("a draft of %d bytes replaced the map whose record cannot be made", len(b))
		}
	})
	t.Run("the switch is off", func(t *testing.T) {
		s, _ := keptMod(t, withInside(t, insideMap(t, true), insidePages()))
		if !s.TextsInside() {
			t.Fatal("the mod's map opened with the switch off")
		}
		if err := s.SetTextsInside(false); err != nil {
			t.Fatal(err)
		}
		if _, err := s.selectWorkspaceMission(2); err != nil {
			t.Fatal(err)
		}
		files := s.workspace.Project.Files
		b, drafted := files["maps/first.almx"]
		if _, held, _ := alm.TextsOf(b); !drafted || held {
			t.Fatalf("the draft: present %v, record %v", drafted, held)
		}
		for name, page := range insidePages() {
			if got := files[path.Join("maps/first.text", name)]; !bytes.Equal(got, page) {
				t.Fatalf("the draft of %s holds %q", name, got)
			}
		}
	})
	t.Run("the switch is on", func(t *testing.T) {
		s, _ := keptMod(t, insideMap(t, true))
		if err := s.SetTextsInside(true); err != nil {
			t.Fatal(err)
		}
		if _, err := s.selectWorkspaceMission(2); err != nil {
			t.Fatal(err)
		}
		if _, held, _ := alm.TextsOf(s.workspace.Project.Files["maps/first.almx"]); !held {
			t.Fatal("the switch alone did not reach the mod's draft")
		}
		if _, err := s.selectWorkspaceMission(1); err != nil || !s.TextsInside() {
			t.Fatalf("back on the map the switch is %v: %v", s.TextsInside(), err)
		}
	})
}
