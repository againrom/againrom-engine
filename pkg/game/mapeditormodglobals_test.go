package game

import (
	"bytes"
	"maps"
	"math"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/words"
)

func setGlobal(t *testing.T, s *mapInspectionEdits, key, value string) {
	t.Helper()
	steps := len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(ui.ModGlobalsScope, key, value); err != nil {
		t.Fatalf("%s = %q: %v", key, value, err)
	}
	if len(s.undoWorkspace) != steps+1 {
		t.Fatalf("%s made %d undo steps", key, len(s.undoWorkspace)-steps)
	}
}

func heldGlobals(t *testing.T, s *mapInspectionEdits) []mod.Global {
	t.Helper()
	d, err := s.heldRows(s.workspace.Project, nil, mod.GlobalsFile)
	if err != nil {
		t.Fatal(err)
	}
	return d.Globals.Globals
}

// bare is the globals as the fixture compares them: without file and line.
func bareGlobals(list []mod.Global) []mod.Global {
	out := make([]mod.Global, len(list))
	for i, g := range list {
		g.File, g.Line = "", 0
		out[i] = g
	}
	return out
}

// The Globals page of a mod with no file shows the New button and writes
// nothing. The mill's three globals made on the page, each edit one undo
// step, read equal to the committed mill's file; a bad name or a repeated
// name is refused and writes nothing.
func TestModGlobalsPageBuildsTheMillFile(t *testing.T) {
	x, s, folder := charactersPage(t, "en")
	before := tree(t, folder)
	if fields := s.WorkspaceFields(ui.ModGlobalsScope); len(fields) != 1 || fields[0].Key != ui.GlobalNewKey || !fields[0].Action {
		t.Fatalf("the empty page holds %+v", fields)
	}
	if !maps.Equal(tree(t, folder), before) {
		t.Fatal("an empty page wrote")
	}
	for range 3 {
		setGlobal(t, s, ui.GlobalNewKey, "")
	}
	for i, id := range []int32{1, 2, 3} {
		if got := heldGlobals(t, s)[i]; got.ID != id {
			t.Fatalf("global %d got number %d", i, got.ID)
		}
	}
	setGlobal(t, s, "global.0.name", "goblins_driven")
	setGlobal(t, s, "global.1.name", "mill_coins")
	setGlobal(t, s, "global.1.initial", "5")
	setGlobal(t, s, "global.2.name", "near_wrap")
	setGlobal(t, s, "global.2.initial", "2147483646")
	setGlobal(t, s, "global.0.note", "1 once the mill quest began.")
	x.archives.Base.Profile.Language = "russian"
	setGlobal(t, s, "global.0.note", "1, когда задание мельницы началось.")
	x.archives.Base.Profile.Language = "english"
	steps := len(s.undoWorkspace)
	for _, c := range []struct{ key, value string }{
		{"global.0.name", "Bad Name"}, {"global.0.name", "mill_coins"}, {"global.1.initial", "x"}, {"global.1.initial", "2147483648"}, {"global.9.name", "other"},
	} {
		if _, err := s.SetWorkspaceField(ui.ModGlobalsScope, c.key, c.value); err == nil || len(s.undoWorkspace) != steps {
			t.Fatalf("%s = %q was taken: %v", c.key, c.value, err)
		}
	}
	want, err := mod.ParseGlobals("m", mod.GlobalsFile, readGlobalsMill(t, mod.GlobalsFile))
	if err != nil {
		t.Fatal(err)
	}
	got, err := mod.ParseGlobals("m", mod.GlobalsFile, s.workspace.Project.Files[mod.GlobalsFile])
	if err != nil {
		t.Fatal(err)
	}
	for i := range got.Globals {
		got.Globals[i].Mod = "m"
	}
	if !reflect.DeepEqual(bareGlobals(got.Globals), bareGlobals(want.Globals)) {
		t.Fatalf("the page wrote %+v, the mill file reads %+v", got.Globals, want.Globals)
	}
	for range len(s.undoWorkspace) {
		if _, err := s.Undo(); err != nil {
			t.Fatal(err)
		}
	}
	if len(heldGlobals(t, s)) != 0 || !maps.Equal(tree(t, folder), before) {
		t.Fatal("undo left a global")
	}
}

// openMillGlobals opens a copy of the committed mill mod in the mod window's
// host, its globals file in the project.
func openMillGlobals(t *testing.T, lang string) (*MapInspector, *mapInspectionEdits, string) {
	t.Helper()
	x, s, _ := charactersPage(t, lang)
	for _, n := range []string{"10", "20"} {
		raw := readGlobalsMill(t, "maps/mill.almx")
		x.archives.Containers.Overlay("scenario/"+n+".alm", func() ([]byte, error) { return raw, nil })
	}
	dir := globalsMillCopy(t, nil, nil)
	doc, err := s.SetWorkspaceField("package", "folder", dir)
	if err != nil || doc == nil {
		t.Fatalf("open the mill: %v", err)
	}
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	doc, err = doc.Edits.(*mapInspectionEdits).SetWorkspaceField("package", "folder", own)
	if err != nil || doc == nil {
		t.Fatalf("copy the mill: %v", err)
	}
	return x, doc.Edits.(*mapInspectionEdits), own
}

// A global a map or a reply uses shows its users and is not removed: the
// refusal names the map and the place. An unused global is removed. A
// number a map names is not given to a new global.
func TestModGlobalsPageRefusesRemovingAUsedGlobal(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		t.Run(lang, func(t *testing.T) {
			_, s, _ := openMillGlobals(t, lang)
			fields := s.WorkspaceFields(ui.ModGlobalsScope)
			var users string
			for _, f := range fields {
				if f.Key == "global.1.users" {
					users = f.TextValue
				}
			}
			for _, want := range []string{"maps/mill.almx", "Setup", "Paid", "mill.toml"} {
				if !strings.Contains(users, want) {
					t.Fatalf("the users of mill_coins read %q, want %q", users, want)
				}
			}
			steps := len(s.undoWorkspace)
			_, err := s.SetWorkspaceField(ui.ModGlobalsScope, "global.1.remove", "")
			if err == nil || len(s.undoWorkspace) != steps || !strings.Contains(err.Error(), "maps/mill.almx") || !strings.Contains(err.Error(), "mill_coins") {
				t.Fatalf("the used global was removed or the refusal names no user: %v", err)
			}
			setGlobal(t, s, ui.GlobalNewKey, "")
			if got := heldGlobals(t, s); len(got) != 4 || got[3].ID != 4 {
				t.Fatalf("a new global took %+v", got)
			}
			setGlobal(t, s, "global.3.remove", "")
			if len(heldGlobals(t, s)) != 3 {
				t.Fatal("an unused global stayed")
			}
		})
	}
}

// globalsMillEditor opens a copy of the committed mill mod in the map
// editor's mod window on its mill map, in the book's language.
func globalsMillEditor(t *testing.T, book string) (*ui.MapEditor, *MapInspector, string) {
	t.Helper()
	f, e := workspaceFixture(t)
	f.x.archives.Base.Profile.Language = map[string]string{"": "english", "ru": "russian"}[book]
	e.SetWords(words.For(book))
	e.Layout(1920, 1080)
	for _, n := range []string{"10", "20"} {
		raw := readGlobalsMill(t, "maps/mill.almx")
		f.x.archives.Containers.Overlay("scenario/"+n+".alm", func() ([]byte, error) { return raw, nil })
	}
	dir := globalsMillCopy(t, nil, nil)
	own := filepath.Join(t.TempDir(), filepath.Base(dir))
	for _, folder := range []string{dir, own} {
		if err := workspaceField(e, "package", 0, folder); err != nil {
			t.Fatal(err)
		}
	}
	return e, f.x, own
}

// A global declared and used through the editor's windows: the mill's
// goblins_driven is renamed on the page, a trigger built in the wizard
// reads it by its name and sets a global the choice «New global...» made,
// and the saved mod plays: the mill's own triggers, which name the globals
// by number, still run and the new trigger acts on the renamed global.
func TestGlobalsDeclaredInTheEditorPlayAfterARename(t *testing.T) {
	for _, book := range []string{"", "ru"} {
		t.Run("lang="+book, func(t *testing.T) {
			e, x, own := globalsMillEditor(t, book)
			word := func(id string) string { return x.inspectionUTF8(words.For(book).Text(id)) }
			if err := modMapsOpen(t, e, 1); err != nil {
				t.Fatal(err)
			}
			s := formHost(e)
			setGlobal(t, s, "global.0.name", "driven")
			wizNewTrigger(t, e, "Built")
			wizPress(t, e, "", 0)
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "c0.item"), word("editor.wizard.c.global"))
			if names := heldGlobals(t, formHost(e)); names[0].Name != "driven" {
				t.Fatalf("the rename left %+v", names)
			}
			trigChoose(t, e, wizField(t, e, "c0.p0"), "driven")
			wizType(t, e, "c0.p1", "1")
			trigChoose(t, e, wizField(t, e, "a0.item"), word("editor.wizard.a.setglobal"))
			trigChoose(t, e, wizField(t, e, "a0.p0"), word("editor.param.newglobal"))
			wizType(t, e, "a0.p1", "42")
			wizApply(t, e)
			tr := wizScript(t, e.Document().SourceBytes).Triggers[2]
			if tr.Name != "Built" || len(tr.Left) != 1 || len(tr.Acts) != 1 {
				t.Fatalf("Apply wrote %+v", tr)
			}
			if got := heldGlobals(t, formHost(e)); len(got) != 4 || got[3].ID != 4 {
				t.Fatalf("New global... left %+v", got)
			}
			_ = e.Key("escape")
			saveMod(t, e)
			saved := tree(t, own)
			g, err := mod.ParseGlobals("m", mod.GlobalsFile, []byte(saved[mod.GlobalsFile]))
			if err != nil || len(g.Globals) != 4 || g.Globals[0].Name != "driven" || g.Globals[0].ID != 1 {
				t.Fatalf("the saved globals (%v):\n%s", err, saved[mod.GlobalsFile])
			}
			f := globalsMillFront(t, own)
			w := openExtendedMission(t, f, 40)
			missionSteps(f.live, 2)
			w = f.live.world
			want := []sim.ScriptGlobal{{ID: 1, Value: 1}, {ID: 2, Value: 7}, {ID: 3, Value: math.MinInt32}, {ID: 4, Value: 42}}
			if got := w.ScriptGlobals(); !reflect.DeepEqual(got, want) {
				t.Fatalf("the saved mod played %v, want %v", got, want)
			}
			f.live.advanceNotice(ui.NoticeReply1)
			missionSteps(f.live, 2)
			if w = f.live.world; w.ScriptGlobals()[1].Value != 10 || w.ScriptRegister(50) != 1 {
				t.Fatalf("the mill's own rows broke: %v, register 50 = %d", w.ScriptGlobals(), w.ScriptRegister(50))
			}
		})
	}
}

// On a map that is still an .alm the first global action asks the question
// that makes it an .almx, and the global slot lists the mod's names beside
// «New global...»; nothing is written before the answer.
func TestGlobalActionAsksToExtendThePlainMap(t *testing.T) {
	for _, book := range []string{"", "ru"} {
		t.Run("lang="+book, func(t *testing.T) {
			_, e, _ := millMod(t, book)
			x := formHost(e).inspector
			word := func(id string) string { return x.inspectionUTF8(words.For(book).Text(id)) }
			setGlobal(t, formHost(e), ui.GlobalNewKey, "")
			setGlobal(t, formHost(e), "global.0.name", "gold_paid")
			wizNewTrigger(t, e, "Pay")
			wizPress(t, e, "+action", 0)
			trigChoose(t, e, wizField(t, e, "a0.item"), word("editor.wizard.a.addglobal"))
			r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: wizField(t, e, "a0.p0")})
			trigClick(t, e, r, ok, "global slot")
			choices := strings.Join(e.DropdownChoices(), "|")
			if !strings.Contains(choices, "gold_paid") || !strings.Contains(choices, word("editor.param.newglobal")) {
				t.Fatalf("the global slot offers %q", choices)
			}
			row := slices.IndexFunc(e.DropdownChoices(), func(c string) bool { return strings.Contains(c, "gold_paid") })
			r, ok = e.FindControl(ui.ControlQuery{Role: ui.ControlDropdownRow, N: row})
			trigClick(t, e, r, ok, "gold_paid")
			before := e.Document().SourceBytes
			wizApply(t, e)
			if !slices.Contains(e.OpenWindows(), "prompt") || e.PromptTitle() == "" || !bytes.Equal(before, e.Document().SourceBytes) {
				t.Fatalf("no question before the first global action: %v %q", e.OpenWindows(), e.PromptTitle())
			}
		})
	}
}

// A trigger deleted from the map leaves its nodes in the script. A global
// those nodes still name keeps its users, is not removed and its number is
// not given to a new global.
func TestModGlobalsPageCountsNodesNoTriggerNames(t *testing.T) {
	_, s, _ := openMillGlobals(t, "en")
	before := heldGlobals(t, s)
	if len(before) != 3 || before[2].ID != 3 {
		t.Fatalf("the mill declares %+v", before)
	}
	users := func(key string) string {
		for _, f := range s.WorkspaceFields(ui.ModGlobalsScope) {
			if f.Key == key {
				return f.TextValue
			}
		}
		return ""
	}
	if !strings.Contains(users("global.2.users"), "trigger") {
		t.Fatalf("near_wrap users before: %q", users("global.2.users"))
	}
	for _, i := range []int{1, 0} {
		if _, err := s.Delete(ui.KindTrigger, i); err != nil {
			t.Fatal(err)
		}
	}
	if got := users("global.2.users"); !strings.Contains(got, "maps/mill.almx") || !strings.Contains(got, "no trigger or reply names it") {
		t.Fatalf("the page lists no user of near_wrap after the triggers went: %q", got)
	}
	steps := len(s.undoWorkspace)
	if _, err := s.SetWorkspaceField(ui.ModGlobalsScope, "global.2.remove", ""); err == nil || len(s.undoWorkspace) != steps || !strings.Contains(err.Error(), "near_wrap") || !strings.Contains(err.Error(), "maps/mill.almx") {
		t.Fatalf("a global a node names was removed or the refusal names no node: %v", err)
	}
	setGlobal(t, s, ui.GlobalNewKey, "")
	if got := heldGlobals(t, s); len(got) != 4 || got[3].ID != 4 {
		t.Fatalf("a new global took %+v", got)
	}
}

// Outside a mod no global exists to name: the wizard's item lists carry no
// global row, and in a mod they do.
func TestTriggerWizardOffersNoGlobalRowOutsideAMod(t *testing.T) {
	items := func(e *ui.MapEditor, key string) string {
		n := wizField(t, e, key)
		r, ok := e.FindControl(ui.ControlQuery{Role: ui.ControlField, N: n})
		trigClick(t, e, r, ok, key)
		return strings.Join(e.DropdownChoices(), "|")
	}
	global := []string{"editor.wizard.c.global", "editor.wizard.a.setglobal", "editor.wizard.a.addglobal"}
	e, _ := kindsEditor(t, oneTriggerMap(t), "")
	e.Select(trigRecord(t, e, "Trigger", 0), true)
	wizPress(t, e, "", 0)
	wizPress(t, e, "+action", 0)
	for _, key := range []string{"c0.item", "a1.item"} {
		got := items(e, key)
		for _, id := range global {
			if w := words.For("").Text(id); strings.Contains(got, w) {
				t.Fatalf("%s outside a mod offers %q", key, w)
			}
		}
	}
	_, m, _ := millMod(t, "")
	wizNewTrigger(t, m, "Pay")
	wizPress(t, m, "+action", 0)
	if got := items(m, "a0.item"); !strings.Contains(got, words.For("").Text("editor.wizard.a.setglobal")) {
		t.Fatalf("a mod's action list has no global row: %q", got)
	}
}
