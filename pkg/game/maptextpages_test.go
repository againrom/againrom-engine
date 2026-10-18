package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
)

func TestMapPagesSaveAsCarriesPages(t *testing.T) {
	f, e := workspaceFixture(t)
	before := bytes.Clone(e.Document().SourceBytes)
	workspacePage(t, e, "event01.txt", "<part=1>\nMap page")
	s := e.Document().Edits.(*mapInspectionEdits)
	path := filepath.Join(t.TempDir(), "saved.alm")
	if _, err := s.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(b, before) {
		t.Fatalf("ALM bytes changed: %v", err)
	}
	page, err := os.ReadFile(filepath.Join(filepath.Dir(path), "saved.text", "en", "event01.txt"))
	if err != nil || string(page) != "<part=1>\nMap page" || s.Dirty() {
		t.Fatalf("page/checkpoint: %q dirty=%v: %v", page, s.Dirty(), err)
	}
	d, err := f.x.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	loaded := d.Edits.(*mapInspectionEdits)
	if !reflect.DeepEqual(loaded.workspace.Pages, s.workspace.Pages) {
		t.Fatal("cold editor reopen lost pages")
	}
	m := s.workspace.Package.mission(b, s.workspace.Pages)
	files := missionPackageFiles("Map", "Map", m, "en")
	if !bytes.Equal(files["mission/text/en/event01.txt"], page) || !strings.Contains(string(files[mod.MissionsFile]), "mission/text/{lang}") {
		t.Fatal("package did not copy the saved page bytes or declare the language folder")
	}
	t.Logf("ALM=%d bytes identical; page=%d bytes identical in SaveAs, cold Open and W builder", len(b), len(page))
}

func TestMapPagesSourceOrder(t *testing.T) {
	for _, lang := range []string{"en", "ru"} {
		for _, arm := range []string{"language", "unlanguaged", "absent", "install", "campaign"} {
			t.Run(lang+"/"+arm, func(t *testing.T) {
				f := newTestPlayFixture(t)
				f.x.archives.Base.Profile.Language = map[string]string{"en": "english", "ru": "russian"}[lang]
				source := filepath.Join(t.TempDir(), "map.alm")
				writeTestPlayInput(t, source, synth.ALM(synth.ALMOptions{Width: 4, Height: 4}))
				folder := filepath.Join(filepath.Dir(source), "map.text")
				writeTestPlayInput(t, filepath.Join(folder, map[string]string{"en": "ru", "ru": "en"}[lang], "event02.txt"), []byte("Wrong language"))
				if arm != "absent" && arm != "campaign" {
					f.x.mod = &editorMod{dir: filepath.Dir(source), id: "map", missions: mod.MissionData{Missions: []mod.Mission{{Kind: mod.MissionReplace, Number: 20, Map: "map.alm"}}}}
				}
				want := ""
				if arm == "language" || arm == "unlanguaged" {
					writeTestPlayInput(t, filepath.Join(folder, "event02.txt"), []byte("Fallback"))
					want = "Fallback"
				}
				if arm == "language" {
					writeTestPlayInput(t, filepath.Join(folder, lang, "event02.txt"), []byte("Local "+lang))
					want = "Local " + lang
				}
				if arm == "install" {
					want = "Page two"
				}
				if arm == "campaign" {
					source, want = "scenario/20.alm", "Page two"
				}
				p := f.x.editorPages(source)
				if p.err != nil || string(p.files["event02.txt"]) != want {
					t.Fatalf("source: %q want %q: %v", p.files["event02.txt"], want, p.err)
				}
				m, err := f.x.testPlayMission(source, nil)
				if err != nil || !reflect.DeepEqual(m.TextFiles, p.files) {
					t.Fatalf("test play chose another source: %v", err)
				}
			})
		}
	}
}

func TestMapPagesWriteFenceAndCollisions(t *testing.T) {
	f, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "Page")
	s := e.Document().Edits.(*mapInspectionEdits)
	before := tree(t, f.root)
	if _, err := s.SaveAs(filepath.Join(f.root, "forbidden.alm")); err == nil || !reflect.DeepEqual(before, tree(t, f.root)) {
		t.Fatalf("install page fence: %v", err)
	}
	dir := t.TempDir()
	writeTestPlayInput(t, filepath.Join(dir, "taken.text", "en", "event01.txt"), []byte("Keep"))
	before = tree(t, dir)
	if _, err := s.SaveAs(filepath.Join(dir, "taken.alm")); err == nil || !reflect.DeepEqual(before, tree(t, dir)) {
		t.Fatalf("page collision changed destination: %v", err)
	}
	if _, err := s.SaveAs(filepath.Join(dir, "free.alm")); err != nil {
		t.Fatalf("outside-install loss control: %v", err)
	}
}

func TestMapPagesSaveAsKeepsUnsavedPackageRoute(t *testing.T) {
	_, e := workspaceFixture(t)
	if err := workspaceField(e, "package", 1, "unsaved-route"); err != nil {
		t.Fatal(err)
	}
	workspacePage(t, e, "event01.txt", "Page")
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SaveAs(filepath.Join(t.TempDir(), "map.alm")); err != nil {
		t.Fatal(err)
	}
	if !s.Dirty() {
		t.Fatal("map SaveAs hid unsaved mod route")
	}
}

func TestMapPagesPackagePreservesOtherLanguage(t *testing.T) {
	_, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "English page")
	s := e.Document().Edits.(*mapInspectionEdits)
	folder := filepath.Join(t.TempDir(), s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	other := "mission/text/ru/event01.txt"
	page := []byte("<part=1>Русская страница")
	writeTestPlayInput(t, filepath.Join(folder, filepath.FromSlash(other)), page)
	old, err := missionPackageOwnership(folder, editorPackageRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	writeTestPlayInput(t, filepath.Join(folder, editorPackageRecord), testPlayRecordBytes(append(old, other)))
	workspacePage(t, e, "event01.txt", "Changed English page")
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(folder, filepath.FromSlash(other)))
	if err != nil || !bytes.Equal(got, page) {
		t.Fatalf("W lost the other language: %q: %v", got, err)
	}
	got, err = os.ReadFile(filepath.Join(folder, "mission", "text", "en", "event01.txt"))
	if err != nil || string(got) != "Changed English page" {
		t.Fatalf("W lost the active edit: %q: %v", got, err)
	}
}

func TestMapPagesEmptySaveAsRefusesStaleTree(t *testing.T) {
	_, e := workspaceFixture(t)
	dir := t.TempDir()
	writeTestPlayInput(t, filepath.Join(dir, "stale.text", "en", "event01.txt"), []byte("Another map"))
	before := tree(t, dir)
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveAs(filepath.Join(dir, "stale.alm")); err == nil || !reflect.DeepEqual(before, tree(t, dir)) {
		t.Fatalf("empty SaveAs adopted or changed stale tree: %v", err)
	}
}

func TestMapPagesSaveAsPreservesLanguageTree(t *testing.T) {
	f, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "English")
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.alm"), filepath.Join(dir, "b.alm")
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveAs(a); err != nil {
		t.Fatal(err)
	}
	writeTestPlayInput(t, filepath.Join(dir, "a.text", "ru", "event01.txt"), []byte("Russian"))
	writeTestPlayInput(t, filepath.Join(dir, "a.text", "event02.txt"), []byte("Plain"))
	d, err := f.x.Open(a)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Edits.(*mapInspectionEdits).SaveAs(b); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(tree(t, filepath.Join(dir, "a.text")), tree(t, filepath.Join(dir, "b.text"))) {
		t.Fatal("SaveAs changed source page tree")
	}
	s := d.Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("pages", "page", "event02.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Changed common on EN"); err != nil {
		t.Fatal(err)
	}
	c := filepath.Join(dir, "c.alm")
	if _, err := s.SaveAs(c); err != nil {
		t.Fatal(err)
	}
	got := tree(t, filepath.Join(dir, "c.text"))
	if len(got) != 4 || string(got["event02.txt"]) != "Plain" || string(got["en/event02.txt"]) != "Changed common on EN" {
		t.Fatalf("common edit tree=%q", got)
	}
	f.x.archives.Base.Profile.Language = "russian"
	pa, pb := f.x.editorPages(a), f.x.editorPages(b)
	if pa.err != nil || pb.err != nil || !reflect.DeepEqual(pa.files, pb.files) {
		t.Fatalf("RU pages: a=%q b=%q errors=%v/%v", pa.files, pb.files, pa.err, pb.err)
	}
	if pc := f.x.editorPages(c); pc.err != nil || !reflect.DeepEqual(pa.files, pc.files) {
		t.Fatalf("EN edit changed RU view=%q: %v", pc.files, pc.err)
	}
}

func TestMapPagesLegacyPackagePreservesCommon(t *testing.T) {
	_, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "Legacy page")
	s := e.Document().Edits.(*mapInspectionEdits)
	folder := filepath.Join(t.TempDir(), s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	old, err := missionPackageOwnership(folder, editorPackageRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	for i, n := range old {
		if n == "mission/text/en/event01.txt" {
			old[i] = "mission/text/event01.txt"
		}
	}
	legacy := filepath.Join(folder, "mission", "text", "event01.txt")
	if err := os.Rename(filepath.Join(folder, "mission", "text", "en", "event01.txt"), legacy); err != nil {
		t.Fatal(err)
	}
	mb, err := os.ReadFile(filepath.Join(folder, mod.MissionsFile))
	if err != nil {
		t.Fatal(err)
	}
	writeTestPlayInput(t, filepath.Join(folder, mod.MissionsFile), []byte(strings.ReplaceAll(string(mb), "mission/text/{lang}", "mission/text")))
	writeTestPlayInput(t, filepath.Join(folder, editorPackageRecord), testPlayRecordBytes(old))
	readRU := func() {
		t.Helper()
		b, err := os.ReadFile(filepath.Join(folder, mod.MissionsFile))
		if err != nil {
			t.Fatal(err)
		}
		d, err := mod.ParseMissions(s.workspace.Package.ID, mod.MissionsFile, b, nil, "ru")
		if err != nil {
			t.Fatal(err)
		}
		m := d.Missions[0]
		if err := m.ReadFiles(folder); err != nil || string(m.TextFiles["event01.txt"]) != "Legacy page" {
			t.Fatalf("RU legacy page=%q: %v", m.TextFiles, err)
		}
	}
	readRU()
	workspacePage(t, e, "event02.txt", "Added on EN")
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	readRU()
	owned, err := missionPackageOwnership(folder, editorPackageRecord, nil)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, n := range owned {
		found = found || n == "mission/text/event01.txt"
	}
	if !found {
		t.Fatal("common fallback lost ownership record")
	}
}

func TestMapPagesPackageCopiesLooseTreeAndEdits(t *testing.T) {
	f, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "English")
	dir := t.TempDir()
	source := filepath.Join(dir, "a.alm")
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveAs(source); err != nil {
		t.Fatal(err)
	}
	writeTestPlayInput(t, filepath.Join(dir, "a.text", "ru", "event01.txt"), []byte("Russian"))
	writeTestPlayInput(t, filepath.Join(dir, "a.text", "event02.txt"), []byte("Plain"))
	d, err := f.x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	s := d.Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("pages", "page", "event01.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Changed EN"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "page", "event02.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Changed common on EN"); err != nil {
		t.Fatal(err)
	}
	folder := filepath.Join(t.TempDir(), s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(folder); err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"en/event01.txt": "Changed EN", "ru/event01.txt": "Russian", "event02.txt": "Plain", "en/event02.txt": "Changed common on EN"}
	got := tree(t, filepath.Join(folder, "mission", "text"))
	if len(got) != len(want) {
		t.Fatalf("tree=%q", got)
	}
	for name, b := range want {
		if string(got[name]) != b {
			t.Fatalf("%s=%q want %q", name, got[name], b)
		}
	}
}

func TestMapPagesDiskEditsRefreshReport(t *testing.T) {
	f, e := workspaceFixture(t)
	workspacePage(t, e, "event01.txt", "Before")
	dir := t.TempDir()
	source := filepath.Join(dir, "map.alm")
	if _, err := e.Document().Edits.(*mapInspectionEdits).SaveAs(source); err != nil {
		t.Fatal(err)
	}
	d, err := f.x.Open(source)
	if err != nil {
		t.Fatal(err)
	}
	s := d.Edits.(*mapInspectionEdits)
	f.x.rules = []mapRule{{ID: "page", run: func(c *mapCheck) []mapHit { return []mapHit{{Text: string(c.pages.files["event01.txt"])}} }}}
	if got := s.Report(); len(got) != 1 || got[0].Text != "Before" {
		t.Fatalf("initial report=%v", got)
	}
	writeTestPlayInput(t, filepath.Join(dir, "map.text", "en", "event01.txt"), []byte("After!"))
	if got := s.Report(); len(got) != 1 || got[0].Text != "After!" {
		t.Fatalf("stale report=%v", got)
	}
	stamp := s.textsStamp()
	writeTestPlayInput(t, filepath.Join(dir, "map.text", "ru", "event01.txt"), []byte("Other"))
	if stamp == s.textsStamp() {
		t.Fatal("other-language tree absent from cache stamp")
	}
	if _, err := s.SetWorkspaceField("pages", "page", "event01.txt"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetWorkspaceField("pages", "body", "Workspace"); err != nil {
		t.Fatal(err)
	}
	if got := s.Report(); len(got) != 1 || got[0].Text != "Workspace" {
		t.Fatalf("workspace override=%v", got)
	}
}
