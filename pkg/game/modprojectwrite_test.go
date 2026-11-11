package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func stagedMapForTest(dir, rel string, b []byte) (string, error) {
	x := &MapInspector{}
	err := x.writeEditorProject(dir, filepath.Base(dir), nil, map[string][]byte{rel: b}, missionPackagePlan{Source: dir}, true)
	return filepath.Join(dir, filepath.FromSlash(rel)), err
}

func TestManifestDraftFieldsUndoAndOneSave(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	for _, row := range []struct{ key, value string }{
		{"id", "manifest-project"}, {"title", "Project title"}, {"version", "2.3.4"}, {"author", "A writer"}, {"description", "Two\nlines"},
		{"applies-to", "common"}, {"requires", "rules>=1.2, extra"}, {"conflicts", "other"}, {"load-after", "rules"},
	} {
		before := workspaceBytes(s.workspace)
		if _, err := s.SetWorkspaceField("package", row.key, row.value); err != nil {
			t.Fatalf("%s: %v", row.key, err)
		}
		after := workspaceBytes(s.workspace)
		if bytes.Equal(before, after) {
			t.Fatalf("%s did not edit draft", row.key)
		}
		if _, err := s.Undo(); err != nil || !bytes.Equal(workspaceBytes(s.workspace), before) {
			t.Fatalf("%s undo: %v", row.key, err)
		}
		if _, err := s.Redo(); err != nil || !bytes.Equal(workspaceBytes(s.workspace), after) {
			t.Fatalf("%s redo: %v", row.key, err)
		}
	}
	dir := filepath.Join(t.TempDir(), s.workspace.Package.ID)
	if _, err := s.saveWorkspacePackage(dir); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dir, mod.ManifestName))
	if err != nil {
		t.Fatal(err)
	}
	m, err := mod.ParseManifest(b)
	if err != nil || m.Title != "Project title" || m.Version != "2.3.4" || m.Author != "A writer" || m.Description != "Two\nlines" || !reflect.DeepEqual(m.AppliesTo, []string{"common"}) || !reflect.DeepEqual(m.Requires, []string{"rules>=1.2", "extra"}) || !reflect.DeepEqual(m.Conflicts, []string{"other"}) || !reflect.DeepEqual(m.LoadAfter, []string{"rules"}) {
		t.Fatalf("%+v %v", m, err)
	}
	if _, err := os.Stat(filepath.Join(dir, mod.EntryName)); !os.IsNotExist(err) {
		t.Fatalf("new package entry: %v", err)
	}
	if _, err := s.SetWorkspaceField("package", "id", "another"); err == nil {
		t.Fatal("opened id changed")
	}
}

func TestManifestFieldRefusalIsTheReadersMessage(t *testing.T) {
	_, e := workspaceFixture(t)
	s := e.Document().Edits.(*mapInspectionEdits)
	for _, row := range []struct {
		key, value string
		encoded    any
	}{{"version", "", ""}, {"title", " ", " "}, {"requires", "core=>1", []string{"core=>1"}}, {"conflicts", "BAD ID", []string{"BAD ID"}}, {"load-after", "a>=1", []string{"a>=1"}}} {
		source, err := s.workspace.Package.manifestBytes(nil)
		if err != nil {
			t.Fatal(err)
		}
		bad, err := mod.WriteManifest(source, map[string]any{row.key: row.encoded})
		if err != nil {
			t.Fatal(err)
		}
		_, want := mod.ParseManifest(bad, true)
		before := workspaceBytes(s.workspace)
		_, got := s.SetWorkspaceField("package", row.key, row.value)
		if want == nil || got == nil || got.Error() != want.Error() || !bytes.Equal(before, workspaceBytes(s.workspace)) {
			t.Fatalf("%s: got %v want %v", row.key, got, want)
		}
	}
}

func TestModSpeakerIsOneDraftEditUntilSave(t *testing.T) {
	s, dir := modSpeakerProject(t)
	before := tree(t, dir)
	steps := len(s.undoWorkspace)
	draft := workspaceBytes(s.workspace)
	image := facePNG(t)
	id, err := s.AddSpeaker(speakerLady, "Lady", image)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, tree(t, dir)) {
		t.Fatal("add wrote into mod folder")
	}
	if len(s.undoWorkspace) != steps+1 || !s.Dirty() {
		t.Fatalf("undo steps %d dirty %v", len(s.undoWorkspace), s.Dirty())
	}
	if got, ok := s.boundSpeaker(speakerLady); !ok || got != id {
		t.Fatal("draft binding absent")
	}
	added := workspaceBytes(s.workspace)
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if len(s.undoWorkspace) != steps || !bytes.Equal(draft, workspaceBytes(s.workspace)) {
		t.Fatal("undo did not restore the whole preceding draft")
	}
	if _, ok := s.boundSpeaker(speakerLady); ok {
		t.Fatal("undo kept draft binding")
	}
	if _, err := s.Redo(); err != nil {
		t.Fatal(err)
	}
	if len(s.undoWorkspace) != steps+1 || !bytes.Equal(added, workspaceBytes(s.workspace)) {
		t.Fatal("redo did not restore the whole speaker edit")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	after := tree(t, dir)
	face, _ := os.ReadFile(image)
	if !strings.Contains(after[mod.SpeakersFile], "Lady") || after[speakerPictureFile(int32(id))] != string(face) {
		t.Fatal("save lost speaker or face")
	}
	if _, err := s.Undo(); err != nil {
		t.Fatal(err)
	}
	if _, ok := s.boundSpeaker(speakerLady); ok {
		t.Fatal("undo after save kept binding")
	}
	if _, err := s.SaveIntoMod(); err != nil {
		t.Fatal(err)
	}
	undone := tree(t, dir)
	if _, ok := undone[mod.SpeakersFile]; ok {
		t.Fatal("undo save kept speaker file")
	}
	if _, ok := undone[speakerPictureFile(int32(id))]; ok {
		t.Fatal("undo save kept new face")
	}
	if _, err := os.Stat(filepath.Dir(filepath.Join(dir, speakerPictureFile(int32(id))))); !os.IsNotExist(err) {
		t.Fatalf("undo save kept the empty speaker folder: %v", err)
	}
}

func TestWorkspaceBindsOpenedModAndKeepsPendingPages(t *testing.T) {
	model, err := mapedit.New(speakerPickMap())
	if err != nil {
		t.Fatal(err)
	}
	s := &mapInspectionEdits{inspector: &MapInspector{}, model: model, source: scenarioPrefix + "25.alm"}
	w := &editorWorkspaceState{Pages: map[string][]byte{"event01.txt": []byte("saved")}, Package: editorPackageSpec{ID: "authored-map", Manifest: defaultPackageManifest()}}
	s.savedWorkspace = workspaceBytes(w)
	s.undoWorkspace = []editorWorkspaceSnapshot{{workspace: cloneWorkspace(w), model: model, source: s.source}}
	s.workspace = cloneWorkspace(w)
	s.workspace.Pages["event01.txt"] = []byte("pending")
	s.workspaceImage = workspaceBytes(s.workspace)
	s.inspector.mod = &editorMod{id: "ambush", title: "Ambush", manifest: []byte("id='ambush'\ntitle='Ambush'\nversion='2'\n"), missions: mod.MissionData{Missions: []mod.Mission{{Mod: "ambush", Key: "first", Number: 25, Map: "map.alm", Texts: "pages/en", Kind: mod.MissionGuest}}}}
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	if s.workspace.Package.ID != "ambush" || s.workspace.Package.Manifest.Version != "2" || s.workspace.Project.Selected != 0 || string(s.workspace.Project.Files["pages/en/event01.txt"]) != "pending" {
		t.Fatal("opened mod did not bind the existing pending draft", s.workspace)
	}
	var saved editorWorkspaceState
	decodeWorkspace(s.savedWorkspace, &saved)
	if saved.Package.ID != "ambush" || string(saved.Pages["event01.txt"]) != "saved" || string(s.undoWorkspace[0].workspace.Pages["event01.txt"]) != "saved" || s.undoWorkspace[0].workspace.Project == nil || len(s.undoWorkspace) != 1 {
		t.Fatal("binding lost the checkpoint or page history")
	}
	before := workspaceBytes(s.workspace)
	if err := s.prepareWorkspace(); err != nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
		t.Fatal("binding changed an already bound draft", err)
	}
}

func TestModSaveKeepsReportAndMissingMissionPriority(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(map[bool]string{false: "handwritten", true: "owned"}[owned], func(t *testing.T) {
			model, err := mapedit.New(speakerPickMap())
			if err != nil {
				t.Fatal(err)
			}
			x := &MapInspector{mod: &editorMod{id: "ambush", owned: owned}, rules: []mapRule{{ID: "bad-map", Error: true, run: func(*mapCheck) []mapHit { return []mapHit{{Text: "invalid map"}} }}}}
			w := &editorWorkspaceState{Package: editorPackageSpec{ID: "ambush"}, Project: &editorModDocument{Selected: -1}}
			s := &mapInspectionEdits{inspector: x, model: model, source: scenarioPrefix + "20.alm", workspace: w, workspaceImage: workspaceBytes(w), savedWorkspace: workspaceBytes(w)}
			_, err = s.SaveIntoMod()
			var refused *ui.ReportRefusal
			if owned {
				if !errors.As(err, &refused) || !refused.Ask || len(refused.Findings) != 1 || refused.Findings[0].Rule != "bad-map" {
					t.Fatalf("owned Save must ask about the report before missing mission: %v", err)
				}
			} else if err == nil || errors.As(err, &refused) || !strings.Contains(err.Error(), "no mission of mod") {
				t.Fatalf("handwritten Save must refuse missing mission before report: %v", err)
			}
		})
	}
}

func TestManifestDraftWithoutAMapSharesHistory(t *testing.T) {
	s := &mapInspectionEdits{inspector: &MapInspector{}}
	if err := s.prepareWorkspace(); err != nil {
		t.Fatal(err)
	}
	before := workspaceBytes(s.workspace)
	if next, err := s.SetWorkspaceField("package", "author", "Mapless author"); err != nil || next != nil {
		t.Fatalf("edit %v %v", next, err)
	}
	if !s.Dirty() || !s.CanUndo() {
		t.Fatal("mapless edit has no history")
	}
	if next, err := s.Undo(); err != nil || next != nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
		t.Fatalf("undo %v %v", next, err)
	}
	if next, err := s.Redo(); err != nil || next != nil || s.workspace.Package.Manifest.Author != "Mapless author" {
		t.Fatalf("redo %v %v", next, err)
	}
}

func TestNewPackageUsesManifestDefaultsWithoutAnEntry(t *testing.T) {
	p := editorPackageSpec{ID: "example", Title: "Example", Manifest: defaultPackageManifest()}
	files := missionPackageFiles(p.Title, p.Manifest.Description, p.mission([]byte("map bytes"), nil), "en")
	want := "id = \"example\"\ntitle = \"Example\"\nversion = \"1.0.0\"\nauthor = \"Againrom\"\ndescription = \"Map editor mission package\"\napi = 1\napplies-to = [\"rom1\"]\n"
	if string(files[mod.ManifestName]) != want {
		t.Fatalf("manifest:\n%s", files[mod.ManifestName])
	}
	if _, ok := files[mod.EntryName]; ok {
		t.Fatal("package emits an unnecessary script")
	}
}

func TestProjectSavePreservesOtherHandwrittenAndOwnedFiles(t *testing.T) {
	for _, owned := range []bool{false, true} {
		t.Run(map[bool]string{false: "handwritten", true: "owned"}[owned], func(t *testing.T) {
			s, dir := modSpeakerProject(t)
			if owned {
				dir = filepath.Join(t.TempDir(), "ambush")
				if _, err := s.saveWorkspacePackage(dir); err != nil {
					t.Fatal(err)
				}
			}
			entry, ok := s.inspector.mod.entryFor(s.source)
			if !ok {
				t.Fatal("fixture has no selected mission")
			}
			before := tree(t, dir)
			if _, err := s.SetWorkspaceField("metadata", "description", "Changed map"); err != nil {
				t.Fatal(err)
			}
			if _, err := s.SaveIntoMod(); err != nil {
				t.Fatal(err)
			}
			after := tree(t, dir)
			for name, data := range before {
				if name != entry.Map && name != editorPackageRecord && after[name] != data {
					t.Errorf("changed retained file %s", name)
				}
			}
			for name := range after {
				if _, existed := before[name]; !existed && name != editorPackageRecord {
					t.Errorf("unexpected new file %s", name)
				}
			}
			if after[entry.Map] == before[entry.Map] || after[entry.Map] != string(s.model.Bytes()) {
				t.Fatal("map change not written")
			}
			if _, ok := after[editorPackageRecord]; ok != owned || s.inspector.mod.owned != owned {
				t.Fatal("save changed package ownership")
			}
		})
	}
}

func TestBaseDumpManifestUsesTheSharedWriter(t *testing.T) {
	got, err := baseDumpManifest("base-dump", "rom1-en")
	if err != nil {
		t.Fatal(err)
	}
	want := "# The base dump of rom1-en: layer 0 of each layered catalog of this install,\n# in the format a mod reads. To make a mod, copy this folder under a new\n# name, set id to that name, keep the records you change and delete the rest.\nid = \"base-dump\"\ntitle = \"Base dump of rom1-en\"\nversion = \"1.0.0\"\napi = 1\napplies-to = [\"rom1-en\"]\n"
	if string(got) != want || len(got) != 327 {
		t.Fatalf("%d bytes:\n%s", len(got), got)
	}
}
