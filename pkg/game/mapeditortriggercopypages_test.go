package game

import (
	"bytes"
	"fmt"
	"maps"
	"path/filepath"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

// copyPagesMap is a 32x32 map of one unit whose script holds three triggers
// that never fire: "Say" shows message 1, "Both" shows messages 1 and 2 by
// actions of its own, and "Deploy" names a drop action at 5,6.
func copyPagesMap(t *testing.T) []byte {
	t.Helper()
	say := func(id, page uint32) scriptNode {
		return scriptNode{label: "say", opcode: 2, id: id, values: []uint32{page}, tags: []uint32{mapload.ScriptInt}}
	}
	actions := []scriptNode{say(1, 1), say(2, 1), say(3, 2), {label: "drop", opcode: 0x10002, id: 4, values: []uint32{5, 6}, tags: []uint32{1, 1}}}
	conditions := []scriptNode{
		{label: "zero", opcode: mapload.ScriptConstant, id: 1, values: []uint32{0}, tags: []uint32{1}},
		{label: "one", opcode: mapload.ScriptConstant, id: 2, values: []uint32{1}, tags: []uint32{1}},
	}
	return copyWordsMap(t, actions, conditions, []goalTrigger{
		{name: "Say", left: 1, right: 2, actions: []uint32{1}},
		{name: "Both", left: 1, right: 2, actions: []uint32{2, 3}},
		{name: "Deploy", actions: []uint32{4}},
	})
}

// copyPagesEditor opens copyPagesMap, as an extended map when extended, in
// l's words with a text for each of pages 1 and 2 and answers the editor,
// its host and the word function.
func copyPagesEditor(t *testing.T, l scriptAddLang, extended bool) (*ui.MapEditor, *MapInspector, func(id string, a ...any) string) {
	t.Helper()
	b := copyPagesMap(t)
	if extended {
		m, err := mapedit.New(b)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.ExtendScript(); err != nil {
			t.Fatal(err)
		}
		if b, err = m.SaveBytes(); err != nil {
			t.Fatal(err)
		}
	}
	e, x, word := scriptAddEditor(t, l, b)
	workspacePage(t, e, "event01.txt", "<part=1>\nFirst "+l.name)
	workspacePage(t, e, "event02.txt", "<part=1>\nSecond "+l.name)
	_ = e.Key("escape")
	return e, x, word
}

func copyEdits(t *testing.T, e *ui.MapEditor) *mapInspectionEdits {
	t.Helper()
	s, ok := e.Document().Edits.(*mapInspectionEdits)
	if !ok {
		t.Fatal("the document has no edit host")
	}
	return s
}

// copyKey selects trigger index and presses the Tools menu's Copy trigger.
func copyKey(t *testing.T, e *ui.MapEditor, index int) {
	t.Helper()
	e.Select(trigRecord(t, e, "Trigger", index), false)
	if err := e.Key("copytrigger"); err != nil {
		t.Fatal(err)
	}
}

// copyPage is the page the action in slot of trigger index shows.
func copyPage(t *testing.T, e *ui.MapEditor, index, slot int) uint32 {
	t.Helper()
	sc := wizScript(t, e.Document().SourceBytes)
	return wizNode(t, sc.Actions, sc.Triggers[index].Acts[slot]).Value[0]
}

// The copy of a trigger that shows a message shows a new page with the
// original's text, EN and RU. A text written to either page afterwards
// leaves the other's bytes; one undo removes the copy and its page and redo
// makes both again; the saved map reopens with both pages, from its texts
// folder and, saved with its texts inside, from the map file alone.
func TestTriggerCopyShowsAPageOfItsOwn(t *testing.T) {
	for _, l := range scriptAddLangs {
		for _, inside := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/inside=%v", l.name, inside), func(t *testing.T) { copyShowsAPageOfItsOwn(t, l, inside) })
		}
	}
}

func copyShowsAPageOfItsOwn(t *testing.T, l scriptAddLang, inside bool) {
	t.Helper()
	e, x, _ := copyPagesEditor(t, l, inside)
	s := copyEdits(t, e)
	start, pages := bytes.Clone(e.Document().SourceBytes), maps.Clone(s.workspace.Pages)
	first := bytes.Clone(pages["event01.txt"])
	if len(pages) != 2 || len(first) == 0 {
		t.Fatalf("the map opens with pages %v", pages)
	}
	steps := len(s.undoWorkspace)
	copyKey(t, e, 0)
	s = copyEdits(t, e)
	copied := bytes.Clone(e.Document().SourceBytes)
	if copyPage(t, e, 0, 0) != 1 || copyPage(t, e, 1, 0) != 3 || copyPage(t, e, 2, 0) != 1 || copyPage(t, e, 2, 1) != 2 {
		t.Fatalf("the original shows page %d and the copy page %d: %q", copyPage(t, e, 0, 0), copyPage(t, e, 1, 0), e.Message())
	}
	if len(s.workspace.Pages) != 3 || !bytes.Equal(s.workspace.Pages["event03.txt"], first) || !bytes.Equal(s.workspace.Pages["event01.txt"], first) {
		t.Fatalf("the pages after the copy are %q", s.workspace.Pages)
	}
	if len(s.undoWorkspace) != steps+1 || !e.Dirty() {
		t.Fatalf("the copy is %d undo steps", len(s.undoWorkspace)-steps)
	}
	if _, err := s.SetMessage(3, "The copy's "+l.name, -1); err != nil {
		t.Fatal(err)
	}
	own := bytes.Clone(s.workspace.Pages["event03.txt"])
	if bytes.Equal(own, first) || !bytes.Equal(s.workspace.Pages["event01.txt"], first) {
		t.Fatalf("a text written to the copy's page left the pages %q", s.workspace.Pages)
	}
	if _, err := s.SetMessage(1, "The original's "+l.name, -1); err != nil {
		t.Fatal(err)
	}
	edited := bytes.Clone(s.workspace.Pages["event01.txt"])
	if bytes.Equal(edited, first) || !bytes.Equal(s.workspace.Pages["event03.txt"], own) {
		t.Fatalf("a text written to the original's page left the pages %q", s.workspace.Pages)
	}
	out := t.TempDir()
	saved := filepath.Join(out, "saved.alm")
	if inside {
		saved += "x"
		if err := s.SetTextsInside(true); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.SaveAs(saved); err != nil || s.SaveNote() != "" {
		t.Fatalf("the save: %v %q", err, s.SaveNote())
	}
	if files := tree(t, out); inside != (len(files) == 1) {
		t.Fatalf("the save wrote %d files", len(files))
	}
	again := x.Editor()
	if err := again.Open(saved); err != nil {
		t.Fatal(err)
	}
	r := copyEdits(t, again)
	if err := r.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if len(r.workspace.Pages) != 3 || !bytes.Equal(r.workspace.Pages["event01.txt"], edited) || !bytes.Equal(r.workspace.Pages["event03.txt"], own) ||
		copyPage(t, again, 0, 0) != 1 || copyPage(t, again, 1, 0) != 3 {
		t.Fatalf("the saved map reopens with the pages %q", r.workspace.Pages)
	}
	for range 2 {
		_ = e.Key("undo")
	}
	s = copyEdits(t, e)
	if !bytes.Equal(s.workspace.Pages["event03.txt"], first) || !bytes.Equal(s.workspace.Pages["event01.txt"], first) || !bytes.Equal(e.Document().SourceBytes, copied) {
		t.Fatalf("two undos leave the pages %q", s.workspace.Pages)
	}
	_ = e.Key("undo")
	s = copyEdits(t, e)
	if !bytes.Equal(e.Document().SourceBytes, start) || !maps.EqualFunc(s.workspace.Pages, pages, bytes.Equal) {
		t.Fatalf("one undo of the copy leaves the pages %q", s.workspace.Pages)
	}
	_ = e.Key("redo")
	s = copyEdits(t, e)
	if !bytes.Equal(e.Document().SourceBytes, copied) || len(s.workspace.Pages) != 3 || !bytes.Equal(s.workspace.Pages["event03.txt"], first) {
		t.Fatalf("redo leaves the pages %q", s.workspace.Pages)
	}
}

// The copy of a trigger with two message actions shows two new pages, each
// with the text of the page its action showed, and a message action naming a
// page the map does not have keeps its number. The window's Copy and the
// command copy_trigger make the same pages as the menu row.
func TestTriggerCopyGivesEachMessageActionAPage(t *testing.T) {
	for _, l := range scriptAddLangs {
		t.Run(l.name, func(t *testing.T) {
			e, _, _ := copyPagesEditor(t, l, false)
			s := copyEdits(t, e)
			first, second := bytes.Clone(s.workspace.Pages["event01.txt"]), bytes.Clone(s.workspace.Pages["event02.txt"])
			check := func(e *ui.MapEditor, route string) []byte {
				t.Helper()
				s := copyEdits(t, e)
				if copyPage(t, e, 1, 0) != 1 || copyPage(t, e, 1, 1) != 2 || copyPage(t, e, 2, 0) != 3 || copyPage(t, e, 2, 1) != 4 {
					t.Fatalf("%s: the copy shows pages %d and %d: %q", route, copyPage(t, e, 2, 0), copyPage(t, e, 2, 1), e.Message())
				}
				p := s.workspace.Pages
				if len(p) != 4 || !bytes.Equal(p["event01.txt"], first) || !bytes.Equal(p["event02.txt"], second) || !bytes.Equal(p["event03.txt"], first) || !bytes.Equal(p["event04.txt"], second) {
					t.Fatalf("%s: the pages after the copy are %q", route, p)
				}
				return bytes.Clone(e.Document().SourceBytes)
			}
			copyKey(t, e, 1)
			got := check(e, "menu row")
			_ = e.Key("undo")
			if s = copyEdits(t, e); len(s.workspace.Pages) != 2 {
				t.Fatalf("undo leaves the pages %q", s.workspace.Pages)
			}
			e.Layout(1280, 800)
			copyPress(t, e, 1)
			if b := check(e, "window"); !bytes.Equal(b, got) {
				t.Fatal("the window's Copy wrote other bytes")
			}
			_ = e.Key("undo")
			_, must := editorMCP(t, e)
			if r := commandReport(t, must("copy_trigger", map[string]any{"index": 1})); r["changed"] != true {
				t.Fatalf("copy_trigger reports %v", r)
			}
			if b := check(e, "command"); !bytes.Equal(b, got) {
				t.Fatal("the command wrote other bytes")
			}
		})
	}
	e, _, _ := scriptAddEditor(t, scriptAddLangs[0], copyPagesMap(t))
	copyKey(t, e, 0)
	if s := copyEdits(t, e); copyPage(t, e, 1, 0) != 1 || s.workspace != nil && len(s.workspace.Pages) != 0 {
		t.Fatalf("a message with no page copies to page %d: %q", copyPage(t, e, 1, 0), e.Message())
	}
}

// A trigger with a drop action is not copied, EN and RU: the menu row, the
// window's Copy and the command say why in the word book's words, the map
// and its pages stay as they were, the document stays unchanged and takes no
// undo step. The window still shows its Copy.
func TestTriggerCopyRefusesADropZone(t *testing.T) {
	routes := []struct {
		name  string
		press func(t *testing.T, e *ui.MapEditor)
	}{
		{"menu row", func(t *testing.T, e *ui.MapEditor) { copyKey(t, e, 2) }},
		{"window", func(t *testing.T, e *ui.MapEditor) {
			e.Layout(1280, 800)
			copyPress(t, e, 2)
		}},
	}
	for _, l := range scriptAddLangs {
		for _, route := range routes {
			t.Run(l.name+"/"+route.name, func(t *testing.T) {
				e, _, word := scriptAddEditor(t, l, copyPagesMap(t))
				start := bytes.Clone(e.Document().SourceBytes)
				want := word("editor.trigger.copydropzone", 4)
				if want == "editor.trigger.copydropzone" || want == "" || e.Message() == want {
					t.Fatal("the word book has no refusal line")
				}
				route.press(t, e)
				s := copyEdits(t, e)
				if e.Message() != want {
					t.Fatalf("the editor says %q, want %q", e.Message(), want)
				}
				if !bytes.Equal(e.Document().SourceBytes, start) || e.Dirty() || s.CanUndo() || s.workspace != nil && len(s.workspace.Pages) != 0 {
					t.Fatal("the refused copy changed the document")
				}
				tool, _ := editorMCP(t, e)
				if r := tool("copy_trigger", map[string]any{"index": 2}); !r.IsError {
					t.Fatalf("copy_trigger copies a drop trigger: %+v", r)
				}
				if !bytes.Equal(e.Document().SourceBytes, start) || e.Dirty() {
					t.Fatal("the refused command changed the document")
				}
			})
		}
	}
}

// In a mod whose mission keeps its pages in the mod's folder, EN and RU, the
// copy's page is saved into the folder of the mod's saved copy beside the
// original's: the loaded mod's mission holds both texts, and the other
// language's page of the original stays.
func TestTriggerCopyPageReachesTheModsFolder(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			f, e := workspaceFixture(t)
			language := map[string]string{"en": "english 0", "ru": "russian 1"}[lang]
			writeTestPlayInput(t, filepath.Join(f.root, MainArchive), synth.Archive([]synth.File{{Path: "id", Data: []byte(language)}}))
			for _, name := range []string{GraphicsArchive, WorldArchive, MoviesArchive} {
				writeTestPlayInput(t, filepath.Join(f.root, name), synth.Archive(nil))
			}
			f.x.archives.Base.Profile.Language = map[string]string{"en": "english", "ru": "russian"}[lang]
			dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey = 'first'\nnumber = 25\nmap = 'maps/first.alm'\nafter = 20\nparty = [{character = 'npc22'}]\n")
			writeTestPlayInput(t, filepath.Join(dir, "maps", "first.alm"), copyPagesMap(t))
			for _, l := range []string{"en", "ru"} {
				writeTestPlayInput(t, filepath.Join(dir, "maps", "first.text", l, "event01.txt"), []byte("<part=1>\nFirst "+l))
			}
			if err := workspaceField(e, "package", 0, dir); err != nil {
				t.Fatal(err)
			}
			// An imported mod is read-only; the save goes to a copy of it.
			saved := filepath.Join(t.TempDir(), "ambush")
			if err := workspaceField(e, "package", 0, saved); err != nil {
				t.Fatal(err)
			}
			_ = e.Key("escape")
			s := copyEdits(t, e)
			if len(wizScript(t, e.Document().SourceBytes).Triggers) != 3 || string(s.workspace.Pages["event01.txt"]) != "<part=1>\nFirst "+lang {
				t.Fatalf("the mod's mission opens with the pages %q", s.workspace.Pages)
			}
			copyKey(t, e, 0)
			s = copyEdits(t, e)
			page := copyPage(t, e, 1, 0)
			if page == 1 || string(s.workspace.Pages[messageFilename(page)]) != "<part=1>\nFirst "+lang {
				t.Fatalf("the copy shows page %d of %q: %q", page, s.workspace.Pages, e.Message())
			}
			if _, err := s.SetMessage(page, "The copy's "+lang, -1); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			for _, l := range []string{"en", "ru"} {
				result, err := modrt.Load([]mod.Entry{mod.Load(saved)}, "rom1-"+l, nil, modrt.Options{})
				if err != nil || len(result.Missions.Missions) != 1 {
					t.Fatalf("load %s: %v", l, err)
				}
				pages := result.Missions.Missions[0].TextFiles
				own := string(pages[messageFilename(page)])
				if string(pages["event01.txt"]) != "<part=1>\nFirst "+l || l == lang && own != "<part=1>\nThe copy's "+lang {
					t.Fatalf("%s: the saved mod holds the pages %q", l, pages)
				}
			}
		})
	}
}
