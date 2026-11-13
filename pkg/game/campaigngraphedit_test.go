package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/mod"
	"againrom/pkg/modrt"
	"againrom/pkg/ui"
)

const graphEditableRows = "# Kept comment\n[[mission]]\nkind = \"campaign\"\nkey = \"first\"\nnumber = 10\nmap = \"maps/first.alm\"\n"

func graphEditFixture(t *testing.T, entry string) (*testPlayFixture, *ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	f, e := graphFixtureIn(t, entry, nil, []string{"Ford", "Ambush", "Vault"})
	f.x.table = companionTable(t)
	f.x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
	dir := writeEditorMod(t, t.TempDir(), graphEditableRows)
	for _, file := range []string{"maps/first.alm", "maps/detour.alm", "maps/en/detour.alm", "maps/ru/detour.alm"} {
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(file)), synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: file}))
	}
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
		t.Fatalf("open: %s %s", f.x.mod.refusal, f.x.mod.contextIssue)
	}
	if err := e.Open(filepath.Join(dir, "maps", "first.alm")); err != nil {
		t.Fatal(err)
	}
	showGraph(t, e)
	s := e.Document().Edits.(*mapInspectionEdits)
	return f, e, s, dir
}

func graphField(t *testing.T, e *ui.MapEditor, key, value string) error {
	t.Helper()
	if !e.FocusModField(key) {
		t.Fatalf("no graph field %s in %v", key, e.ObjectFields())
	}
	if err := e.Key("clear"); err != nil {
		return err
	}
	e.TextInput(value)
	return e.Key("enter")
}

func graphAction(t *testing.T, e *ui.MapEditor, key string) {
	t.Helper()
	if !e.FocusModField(key) {
		t.Fatalf("no graph action %s in %v", key, e.ObjectFields())
	}
	if err := e.Key("enter"); err != nil {
		t.Fatal(err)
	}
}

func TestCampaignGraphEditsAreDraftsWithOneUndo(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, e, s, dir := graphEditFixture(t, entry)
			files := tree(t, dir)
			graphAction(t, e, "graph.edit")
			clickGraphMission(t, e, 10)
			for _, edit := range []struct{ key, value string }{{"AutoGetMission", "-1"}, {"landing", "1"}, {"insert", "11 maps/detour.alm"}} {
				before, steps := workspaceBytes(s.workspace), len(s.undoWorkspace)
				page := graphLines(s.CampaignGraph())
				if err := graphField(t, e, edit.key, edit.value); err != nil {
					t.Fatalf("%s: %v", edit.key, err)
				}
				after := workspaceBytes(s.workspace)
				if bytes.Equal(before, after) || len(s.undoWorkspace) != steps+1 {
					t.Fatalf("%s: not one draft step", edit.key)
				}
				if edit.key != "landing" && slices.Equal(page, graphLines(s.CampaignGraph())) {
					t.Fatalf("%s: graph kept the saved route", edit.key)
				}
				if err := e.Key("undo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), before) {
					t.Fatalf("%s undo: %v", edit.key, err)
				}
				if !slices.Equal(page, graphLines(s.CampaignGraph())) {
					t.Fatalf("%s: undo did not restore graph", edit.key)
				}
				if err := e.Key("redo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), after) {
					t.Fatalf("%s redo: %v", edit.key, err)
				}
				if err := e.Key("undo"); err != nil {
					t.Fatal(err)
				}
			}
			graphAction(t, e, "graph.link")
			clickGraphMission(t, e, 20)
			if s.workspace.Project.Missions[0].AutoGetMission == nil || *s.workspace.Project.Missions[0].AutoGetMission != 20 {
				t.Fatal("canvas link missing")
			}
			graphAction(t, e, "graph.unlink")
			if m := s.workspace.Project.Missions[len(s.workspace.Project.Missions)-1]; m.AutoGetMission == nil || *m.AutoGetMission != -1 {
				t.Fatal("canvas unlink missing")
			}
			if !reflect.DeepEqual(files, tree(t, dir)) {
				t.Fatal("graph edit wrote before Ctrl+S")
			}
		})
	}
}

func TestCampaignGraphSavesReopensAndPlaysTheInsertedMission(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, e, s, dir := graphEditFixture(t, entry)
			graphAction(t, e, "graph.edit")
			clickGraphMission(t, e, 10)
			if err := graphField(t, e, "insert", "11 maps/detour.alm"); err != nil {
				t.Fatal(err)
			}
			before := tree(t, dir)
			if err := e.Key("saveas"); err != nil {
				t.Fatal(err)
			}
			if s.Dirty() {
				t.Fatal("save kept draft dirty")
			}
			want := graphEditableRows + "AutoGetMission = 11\n[[mission]]\nkind = \"campaign\"\nkey     = \"mission11\"\nnumber  = 11\nmap     = \"maps/detour.alm\"\nAutoGetMission = 20\nparty = \"main\"\n"
			b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.MissionsFile)))
			if err != nil || string(b) != want {
				t.Fatalf("written rows:\n%s\nwant:\n%s\n%v", b, want, err)
			}
			before[mod.MissionsFile] = want
			if !reflect.DeepEqual(before, tree(t, dir)) {
				t.Fatal("save changed unrelated files")
			}
			if _, err := s.Undo(); err != nil || !s.Dirty() {
				t.Fatalf("undo after save: %v", err)
			}
			if _, err := s.Redo(); err != nil {
				t.Fatal(err)
			}
			x := contextInspector(t)
			x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
			x.archives.Containers.Overlay("scenario/10.alm", func() ([]byte, error) { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24}), nil })
			x.archives.Containers.Overlay("scenario/20.alm", func() ([]byte, error) { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24}), nil })
			if err := x.SetMod(dir); err != nil || x.mod.refusal != "" {
				t.Fatalf("reopen: %v %s", err, x.mod.refusal)
			}
			result, err := modrt.Load([]mod.Entry{mod.Load(dir)}, "rom1-en", nil, modrt.Options{})
			if err != nil {
				t.Fatal(err)
			}
			f := missionFlowFront(t, false, result.Missions)
			for _, n := range []int{10, 11, 20} {
				openMissionFlow(t, f, n)
				want := map[int]int{10: 11, 11: 20, 20: 0}[n]
				winMissionFlow(t, f, want)
			}
			if !f.Town.Open() || !f.Town.Done(11) {
				t.Fatal("graph-made campaign did not reach its town")
			}
		})
	}
}

func TestCampaignGraphRefusesWithTheGameWordsAndNoDraft(t *testing.T) {
	_, _, s, dir := graphEditFixture(t, "english")
	for _, value := range []string{"10", "31", "999"} {
		w := cloneWorkspace(s.workspace)
		n := 0
		fmt.Sscan(value, &n)
		w.Project.Missions[0].AutoGetMission = &n
		_, want := s.missionDraft(w)
		before, steps, files := workspaceBytes(s.workspace), len(s.undoWorkspace), tree(t, dir)
		_, got := s.SetWorkspaceField("campaign.10", "AutoGetMission", value)
		if got == nil || want == nil || got.Error() != want.Error() {
			t.Fatalf("%s: got %v, want %v", value, got, want)
		}
		if !bytes.Equal(before, workspaceBytes(s.workspace)) || steps != len(s.undoWorkspace) || !reflect.DeepEqual(files, tree(t, dir)) {
			t.Fatal("refusal changed draft, history or files")
		}
	}
	if _, err := s.SetWorkspaceField("campaign.10", "insert", "11 ../escape.alm"); err == nil || !strings.Contains(err.Error(), "map") {
		t.Fatalf("bad map: %v", err)
	}
}

func TestCampaignGraphEachFieldSavesAndReopens(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		for _, edit := range []struct{ name, key, value, suffix string }{
			{"arrow", "AutoGetMission", "-1", "AutoGetMission = -1\n"},
			{"party", "landing", "1", "party = [{ character = \"npc22\" }]\n"},
			{"companion", "landing", "0", "guest = 22\nchapter = 30\n"},
			{"main", "landing", "2", "party = \"main\"\n"},
			{"language map", "insert", "11 maps/{lang}/detour.alm", "AutoGetMission = 11\n[[mission]]\nkind = \"campaign\"\nkey     = \"mission11\"\nnumber  = 11\nmap     = \"maps/{lang}/detour.alm\"\nAutoGetMission = 20\nparty = \"main\"\n"},
		} {
			t.Run(entry+"/"+edit.name, func(t *testing.T) {
				_, e, s, dir := graphEditFixture(t, entry)
				files := tree(t, dir)
				graphAction(t, e, "graph.edit")
				clickGraphMission(t, e, 10)
				if edit.name == "main" {
					if err := graphField(t, e, "landing", "1"); err != nil {
						t.Fatal(err)
					}
				}
				if err := graphField(t, e, edit.key, edit.value); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(files, tree(t, dir)) {
					t.Fatal("field edit wrote files")
				}
				if err := e.Key("saveas"); err != nil {
					t.Fatal(err)
				}
				if s.Dirty() {
					t.Fatal("save did not checkpoint")
				}
				files[mod.MissionsFile] = graphEditableRows + edit.suffix
				if !reflect.DeepEqual(files, tree(t, dir)) {
					t.Fatalf("save differs from hand-made row:\n%s", tree(t, dir)[mod.MissionsFile])
				}
				x := contextInspector(t)
				x.table = companionTable(t)
				x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
				x.archives.Containers.Overlay("scenario/10.alm", func() ([]byte, error) { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24}), nil })
				if err := x.SetMod(dir); err != nil {
					t.Fatal(err)
				}
				if x.mod.refusal != "" || x.mod.contextIssue != "" {
					t.Fatalf("reopen: %s %s", x.mod.refusal, x.mod.contextIssue)
				}
				if got := x.mod.source; string(got) != files[mod.MissionsFile] {
					t.Fatalf("reopen row:\n%s", got)
				}
			})
		}
	}
}
