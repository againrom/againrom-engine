package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/render/terrain"
	"againrom/pkg/render/text"
	"againrom/pkg/ui"
)

func workspaceField(e *ui.MapEditor, scope string, index int, value string) error {
	if err := e.Key(scope); err != nil {
		return err
	}
	for n := 0; n < index; n++ {
		if err := e.Key("tab"); err != nil {
			return err
		}
	}
	if err := e.Key("clear"); err != nil {
		return err
	}
	for n, line := range strings.Split(value, "\n") {
		if n > 0 {
			if err := e.Key("newline"); err != nil {
				return err
			}
		}
		e.TextInput(line)
	}
	return e.Key("enter")
}

func workspacePage(t *testing.T, e *ui.MapEditor, name, value string) {
	t.Helper()
	if err := workspaceField(e, "pages", 0, name); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "pages", 1, value); err != nil {
		t.Fatal(err)
	}
}

func workspaceFixture(t *testing.T) (*testPlayFixture, *ui.MapEditor) {
	t.Helper()
	f := newTestPlayFixture(t)
	f.x.tiles = &terrain.Tileset{}
	f.x.font = &text.Font{Selector: 1}
	e := f.x.Editor()
	if err := e.Key("new"); err != nil {
		t.Fatal(err)
	}
	_ = e.Key("clear")
	e.TextInput("40x56")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if e.Document() == nil || e.Document().Width != 40 || e.Document().Height != 56 {
		t.Fatal("window did not create the rectangle")
	}
	return f, e
}

func TestMapEditorWorkspaceMetadataAndPagesShareHistory(t *testing.T) {
	_, e := workspaceFixture(t)
	original := bytes.Clone(e.Document().SourceBytes)
	if err := workspaceField(e, "metadata", 0, "Карта"); err != nil {
		t.Fatal(err)
	}
	name := bytes.Clone(e.Document().SourceBytes)
	meta := inspectionSectionOffsets(t, name)[0]
	wantName := []byte{0xca, 0xe0, 0xf0, 0xf2, 0xe0}
	want := bytes.Clone(original)
	clear(want[meta+0x30 : meta+0x70])
	copy(want[meta+0x30:], wantName)
	if !bytes.Equal(name, want) || e.Document().Name != "Карта" {
		t.Fatal("Cyrillic name changed unrelated bytes or display")
	}
	if err := workspaceField(e, "metadata", 1, "Дорога\nДом"); err != nil {
		t.Fatal(err)
	}
	description := bytes.Clone(e.Document().SourceBytes)
	desc := []byte{0xc4, 0xee, 0xf0, 0xee, 0xe3, 0xe0, '\n', 0xc4, 0xee, 0xec}
	clear(want[meta+0x78 : meta+0xb8])
	copy(want[meta+0x78:], desc)
	if !bytes.Equal(description, want) {
		t.Fatal("description changed bytes outside its fixed slot")
	}
	workspacePage(t, e, "event01.txt", "Привет\nВторая строка")
	s := e.Document().Edits.(*mapInspectionEdits)
	if got := string(s.workspace.Pages["event01.txt"]); got != "Привет\nВторая строка" {
		t.Fatalf("multiline window input became %q", got)
	}
	if !e.Dirty() || !s.CanUndo() {
		t.Fatal("page edit is absent from dirty/history")
	}
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "only.alm")); err == nil {
		t.Fatal("ALM-only save hid unsaved pages")
	}
	for _, before := range [][]byte{description, name, original} {
		if err := e.Key("undo"); err != nil || !bytes.Equal(e.Document().SourceBytes, before) {
			t.Fatalf("undo: %v", err)
		}
	}
	if e.Dirty() || s.CanUndo() || len(s.workspace.Pages) != 0 {
		t.Fatal("undo did not restore the blank workspace")
	}
	for _, after := range [][]byte{name, description, description} {
		if err := e.Key("redo"); err != nil || !bytes.Equal(e.Document().SourceBytes, after) {
			t.Fatalf("redo: %v", err)
		}
	}
	if string(s.workspace.Pages["event01.txt"]) != "Привет\nВторая строка" {
		t.Fatal("redo lost the page")
	}
	for _, row := range []struct {
		field, rel int
		value      string
		word       uint32
	}{{2, 8, "0x7f800001", 0x7f800001}, {3, 0xc, "4294967295", 0xffffffff}, {4, 0x10, "123", 123}, {5, 0x14, "456", 456}, {6, 0x18, "789", 789}, {7, 0x70, "12", 12}, {8, 0x74, "13", 13}} {
		before := bytes.Clone(e.Document().SourceBytes)
		if err := workspaceField(e, "metadata", row.field, row.value); err != nil {
			t.Fatal(err)
		}
		binary.LittleEndian.PutUint32(before[meta+row.rel:], row.word)
		if !bytes.Equal(before, e.Document().SourceBytes) {
			t.Fatalf("raw field %d changed another byte", row.field)
		}
	}
	before, history := bytes.Clone(e.Document().SourceBytes), len(s.undoWorkspace)
	for _, bad := range []struct {
		scope string
		index int
		value string
	}{{"metadata", 0, strings.Repeat("а", 64)}, {"metadata", 0, "a\x00b"}, {"metadata", 3, "4294967296"}, {"pages", 0, "../title.txt"}, {"pages", 1, "cannot encode 中"}} {
		if err := workspaceField(e, bad.scope, bad.index, bad.value); err == nil {
			t.Fatalf("accepted %q", bad.value)
		}
		if !bytes.Equal(before, e.Document().SourceBytes) || len(s.undoWorkspace) != history || string(s.workspace.Pages["event01.txt"]) != "Привет\nВторая строка" {
			t.Fatal("rejection changed map, page or history")
		}
		_ = e.Key("escape")
	}
}

func loadWorkspaceMod(t *testing.T, root, folder string) mod.Mission {
	t.Helper()
	entries, err := mod.Resolve(filepath.Dir(folder), []string{filepath.Base(folder)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := modrt.Load(entries, BaseID(InspectInstall(root)), nil, modrt.Options{})
	if err != nil || len(result.Missions.Missions) != 1 {
		t.Fatalf("normal mod load: %v; missions %d", err, len(result.Missions.Missions))
	}
	return result.Missions.Missions[0]
}

func TestMapEditorWorkspaceCreatesSavesAndReopensBothRoutes(t *testing.T) {
	f, e := workspaceFixture(t)
	pages := map[string]string{"title.txt": "Карта", "briefing.txt": "Начало\nЗадание", "briefmap.txt": "Путь", "event01.txt": "Привет\nДиалог", "tips01.txt": "Совет\nДальше", "event100.txt": "Сотое сообщение", "tips100.txt": "Сотый совет"}
	for _, name := range []string{"title.txt", "briefing.txt", "briefmap.txt", "event01.txt", "tips01.txt", "event100.txt", "tips100.txt"} {
		workspacePage(t, e, name, pages[name])
	}
	if err := workspaceField(e, "package", 1, "workspace"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(e, "package", 2, "Карта автора"); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), "workspace")
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, folder)
	if m.Kind != mod.MissionGuest || m.Number != 21 || m.After != 20 || m.Guest != 22 || m.Chapter != 30 || e.Dirty() {
		t.Fatalf("guest declaration %+v; dirty %v", m, e.Dirty())
	}
	if !bytes.Equal(m.MapBytes, e.Document().SourceBytes) {
		t.Fatal("package does not hold the authored map")
	}
	for name, value := range pages {
		if string(m.TextFiles[name]) != value {
			t.Fatalf("page %s changed", name)
		}
	}
	writeTestPlayInput(t, filepath.Join(folder, "author-notes.txt"), []byte("retain exact author bytes"))
	writeTestPlayInput(t, filepath.Join(folder, "mission/text/event88.txt"), []byte("Retained page"))
	workspacePage(t, e, "event01.txt", "Исправлено\nПродолжение")
	if err := e.Key("saveas"); err != nil || e.Dirty() {
		t.Fatalf("package save: %v", err)
	}
	if got := tree(t, folder); got["author-notes.txt"] != "retain exact author bytes" || got["mission/text/event88.txt"] != "Retained page" {
		t.Fatal("save lost an unrelated file")
	}
	if err := e.Key("testplay"); err != nil {
		t.Fatal(err)
	}
	played := loadWorkspaceMod(t, f.root, filepath.Join(f.profile, TestPlayModID, TestPlayModID))
	if string(played.TextFiles["event01.txt"]) != "Исправлено\nПродолжение" || !bytes.Equal(played.MapBytes, e.Document().SourceBytes) || len(f.started) != 1 {
		t.Fatal("F5 did not stage authored map and pages")
	}
	if string(played.TextFiles["event100.txt"]) != pages["event100.txt"] || string(played.TextFiles["tips100.txt"]) != pages["tips100.txt"] {
		t.Fatal("F5 dropped numbered pages above99")
	}
	x2 := *f.x
	x2.mod = nil
	fresh := x2.Editor()
	if err := fresh.Open("new/32x32"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(fresh, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	s := fresh.Document().Edits.(*mapInspectionEdits)
	if fresh.Dirty() || s.workspace.Package.Kind != mod.MissionGuest || string(s.workspace.Pages["event01.txt"]) != "Исправлено\nПродолжение" {
		t.Fatal("fresh package reopen lost texts or route")
	}
	if err := fresh.Key("saveas"); err != nil {
		t.Fatalf("reopened package with unowned page: %v", err)
	}
	x2.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return bytes.Clone(m.MapBytes), nil })
	if err := workspaceField(fresh, "package", 3, "replace"); err != nil {
		t.Fatal(err)
	}
	if err := workspaceField(fresh, "package", 4, "20"); err != nil {
		t.Fatal(err)
	}
	if err := fresh.Key("saveas"); err != nil {
		t.Fatal(err)
	}
	m = loadWorkspaceMod(t, f.root, folder)
	if m.Kind != mod.MissionReplace || m.Number != 20 || m.After != 0 || m.Guest != 0 {
		t.Fatalf("replacement declaration %+v", m)
	}
	if x2.mod.title != "Карта автора" || !reflect.DeepEqual(s.workspace.Pages["event01.txt"], []byte("Исправлено\nПродолжение")) {
		t.Fatal("route save changed package title or pages")
	}
}

func TestMissionPageNamesUseTheRuntimeNumberDomain(t *testing.T) {
	for _, name := range []string{"title.txt", "briefing.txt", "briefmap.txt", "event00.txt", "event01.txt", "event99.txt", "event100.txt", "event2147483647.txt", "tips01.txt", "tips100.txt", "tips2147483647.txt"} {
		if !missionPageName(name) {
			t.Errorf("refused canonical runtime filename %s", name)
		}
	}
	for _, name := range []string{"event1.txt", "event001.txt", "event0100.txt", "event-1.txt", "event+01.txt", "tips2147483648.txt", "event18446744073709551616.txt", "../title.txt", "notes.txt"} {
		if missionPageName(name) {
			t.Errorf("accepted noncanonical/overflow filename %s", name)
		}
	}
	if path, ok := EventTextPath(20, 100); !ok || path != "main/text/battle/m20/event100.txt" {
		t.Fatalf("runtime event100 path %q (%v)", path, ok)
	}
}

func TestMapEditorWorkspaceRefusesBeforeChangingFiles(t *testing.T) {
	f, e := workspaceFixture(t)
	if err := workspaceField(e, "package", 1, "workspace"); err != nil {
		t.Fatal(err)
	}
	for _, folder := range []string{filepath.Join(f.root, "workspace"), filepath.Join(t.TempDir(), "wrong-id")} {
		before := tree(t, f.root)
		if err := workspaceField(e, "package", 0, folder); err == nil {
			t.Fatalf("accepted protected folder %s", folder)
		}
		if !reflect.DeepEqual(before, tree(t, f.root)) || !e.Dirty() {
			t.Fatal("refusal changed input files or checkpoint")
		}
		_ = e.Key("escape")
	}
	folder := filepath.Join(t.TempDir(), "workspace")
	if err := workspaceField(e, "package", 0, folder); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	workspacePage(t, e, "title.txt", "Changed")
	before := tree(t, folder)
	if err := workspaceField(e, "package", 0, folder); err == nil {
		t.Fatal("opening a package discarded dirty pages")
	}
	_ = e.Key("escape")
	if !reflect.DeepEqual(before, tree(t, folder)) || string(s.workspace.Pages["title.txt"]) != "Changed" {
		t.Fatal("refused reopen lost a draft")
	}
	writeTestPlayInput(t, filepath.Join(folder, "mission/text/event99.txt"), bytes.Repeat([]byte{'A'}, mod.MaxFileBytes+1))
	before = tree(t, folder)
	if _, err := s.SaveIntoMod(); err == nil {
		t.Fatal("published a package the normal loader refuses")
	}
	if !reflect.DeepEqual(before, tree(t, folder)) || !e.Dirty() {
		t.Fatal("staging failure changed files or draft")
	}
	if entries, _ := os.ReadDir(filepath.Dir(folder)); len(entries) != 1 {
		t.Fatalf("failed staging left %d folders", len(entries))
	}
	t.Log(fmt.Sprintf("preserved %d package files on staged loader refusal", len(before)))
}

func TestMapEditorWorkspaceImportsManualModAsAnOwnedCopy(t *testing.T) {
	f, e := workspaceFixture(t)
	original := bytes.Clone(e.Document().SourceBytes)
	second := bytes.Clone(original)
	meta := inspectionSectionOffsets(t, second)[0]
	clear(second[meta+0x30 : meta+0x70])
	copy(second[meta+0x30:], "Second map")
	f.x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return bytes.Clone(original), nil })
	declarations := editorModMissions + "texts = \"mission/{lang}/text\"\n[[mission]]\nkey = \"second\"\nnumber = 21\nmap = \"mission/en/second.alm\"\ntexts = \"mission/en/second-text\"\nafter = 20\nguest = 22\nchapter = 30\n"
	dir := writeEditorMod(t, t.TempDir(), declarations)
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/map.alm"), original)
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/text/title.txt"), []byte("Original title"))
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/text/event01.txt"), []byte("First mission only"))
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/second.alm"), second)
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/second-text/title.txt"), []byte("Second title"))
	writeTestPlayInput(t, filepath.Join(dir, "mission/en/second-text/event02.txt"), []byte("Second mission only"))
	writeTestPlayInput(t, filepath.Join(dir, "author-notes.txt"), []byte("Keep me"))
	before := tree(t, dir)
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, original) {
		t.Fatal("initial import did not select first mission")
	}
	if err := e.Key("package"); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 9; n++ {
		_ = e.Key("tab")
	}
	_ = e.Key("down")
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(e.Document().SourceBytes, second) || e.Document().Name != "Second map" {
		t.Fatal("source mission choice did not open second map")
	}
	source := e.Document().Edits.(*mapInspectionEdits)
	if string(source.workspace.Pages["event02.txt"]) != "Second mission only" || source.workspace.Pages["event01.txt"] != nil {
		t.Fatal("source choice retained first mission pages")
	}
	if err := e.Key("saveas"); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("imported writer: %v", err)
	}
	_ = e.Key("escape")
	workspacePage(t, e, "title.txt", "Копия\nИзменена")
	if err := workspaceField(e, "package", 9, "1"); err == nil {
		t.Fatal("source change discarded unsaved pages")
	}
	_ = e.Key("escape")
	if err := workspaceField(e, "package", 1, "ambush-copy"); err != nil {
		t.Fatal(err)
	}
	copyDir := filepath.Join(t.TempDir(), "ambush-copy")
	if err := workspaceField(e, "package", 0, copyDir); err != nil {
		t.Fatal(err)
	}
	m := loadWorkspaceMod(t, f.root, copyDir)
	if m.Kind != mod.MissionGuest || m.Number != 21 || string(m.TextFiles["title.txt"]) != "Копия\nИзменена" || string(m.TextFiles["event02.txt"]) != "Second mission only" || m.TextFiles["event01.txt"] != nil || !bytes.Equal(m.MapBytes, second) {
		t.Fatal("owned copy lost the imported mission or text draft")
	}
	if !reflect.DeepEqual(before, tree(t, dir)) || !f.x.mod.owned || f.x.mod.copyOnly || e.Dirty() {
		t.Fatal("import/copy changed source or did not adopt the owned copy")
	}
}
