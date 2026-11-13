package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func numberNamedMod(t *testing.T, rows bool) (dir, open string) {
	t.Helper()
	blank := func(name string) []byte { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: name}) }
	if !rows {
		dir = fileFormMod(t, map[string][]byte{"10.alm": blank("ten"), "20.alm": blank("twenty"), "31.alm": blank("side"), "maps/detour.alm": blank("detour")})
		return dir, "20.alm"
	}
	dir = writeEditorMod(t, t.TempDir(), graphEditableRows)
	for _, file := range []string{"maps/first.alm", "maps/20.alm", "maps/detour.alm"} {
		writeFileAll(t, filepath.Join(dir, filepath.FromSlash(file)), blank(file))
	}
	return dir, "maps/20.alm"
}

func numberNamedFixture(t *testing.T, entry string, rows bool) (*ui.MapEditor, *mapInspectionEdits, string) {
	t.Helper()
	f, e := graphFixtureIn(t, entry, nil, []string{"Ford", "Ambush", "Vault"})
	f.x.table = companionTable(t)
	f.x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
	dir, open := numberNamedMod(t, rows)
	if err := f.x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if f.x.mod.refusal != "" || f.x.mod.contextIssue != "" {
		t.Fatalf("open: %s %s", f.x.mod.refusal, f.x.mod.contextIssue)
	}
	if err := e.Open(filepath.Join(dir, filepath.FromSlash(open))); err != nil {
		t.Fatal(err)
	}
	showGraph(t, e)
	return e, e.Document().Edits.(*mapInspectionEdits), dir
}

func reopenedMod(t *testing.T, dir string) *MapInspector {
	t.Helper()
	x := contextInspector(t)
	x.table = companionTable(t)
	x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
	for _, n := range []string{"10", "20", "31"} {
		x.archives.Containers.Overlay("scenario/"+n+".alm", func() ([]byte, error) { return synth.ALM(synth.ALMOptions{Width: 24, Height: 24}), nil })
	}
	if err := x.SetMod(dir); err != nil {
		t.Fatal(err)
	}
	if x.mod.refusal != "" || x.mod.contextIssue != "" {
		t.Fatalf("reopen: %s %s", x.mod.refusal, x.mod.contextIssue)
	}
	return x
}

func TestCampaignGraphEditsAModOfNumberNamedMaps(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		for _, rows := range []bool{false, true} {
			name := entry + "/files"
			if rows {
				name = entry + "/row and file"
			}
			t.Run(name, func(t *testing.T) {
				e, s, dir := numberNamedFixture(t, entry, rows)
				files := tree(t, dir)
				graphAction(t, e, "graph.edit")
				clickGraphMission(t, e, 20)
				for _, edit := range []struct{ key, value string }{{"AutoGetMission", "-1"}, {"landing", "1"}, {"insert", "21 maps/detour.alm"}} {
					before, steps := workspaceBytes(s.workspace), len(s.undoWorkspace)
					if err := graphField(t, e, edit.key, edit.value); err != nil {
						t.Fatalf("%s: %v", edit.key, err)
					}
					after := workspaceBytes(s.workspace)
					if bytes.Equal(before, after) || len(s.undoWorkspace) != steps+1 {
						t.Fatalf("%s: not one draft step", edit.key)
					}
					if err := e.Key("undo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), before) {
						t.Fatalf("%s undo: %v", edit.key, err)
					}
					if err := e.Key("redo"); err != nil || !bytes.Equal(workspaceBytes(s.workspace), after) {
						t.Fatalf("%s redo: %v", edit.key, err)
					}
				}
				if !reflect.DeepEqual(files, tree(t, dir)) {
					t.Fatal("graph edit wrote before Ctrl+S")
				}
				if err := e.Key("saveas"); err != nil {
					t.Fatal(err)
				}
				if s.Dirty() {
					t.Fatal("save kept draft dirty")
				}
				saved := tree(t, dir)
				written := saved[mod.MissionsFile]
				for _, want := range []string{"number  = 20\n", "AutoGetMission = 21\n", "party = [{ character = \"npc22\" }]\n", "map     = \"maps/detour.alm\"\n"} {
					if !strings.Contains(written, want) {
						t.Fatalf("written rows lack %q:\n%s", want, written)
					}
				}
				if !rows {
					if _, held := saved["20.alm"]; held || saved["maps/m20.alm"] != files["20.alm"] {
						t.Fatalf("the edited mission's map: %v", slices.Sorted(mapsKeys(saved)))
					}
					delete(files, "20.alm")
				}
				for name, body := range files {
					if name != mod.MissionsFile && name != "main.star" && saved[name] != body {
						t.Fatalf("save changed %s", name)
					}
				}
				x := reopenedMod(t, dir)
				var numbers []int
				for _, m := range x.mod.missions.Missions {
					numbers = append(numbers, m.Number)
				}
				slices.Sort(numbers)
				want := []int{10, 20, 21, 31}
				if rows {
					want = []int{10, 20, 21}
				}
				if !slices.Equal(numbers, want) {
					t.Fatalf("reopened missions %v, want %v", numbers, want)
				}
			})
		}
	}
}

func TestMissionsPageEditsANumberNamedMission(t *testing.T) {
	for _, rows := range []bool{false, true} {
		_, s, dir := numberNamedFixture(t, "english", rows)
		for _, edit := range [][2]string{{"landing", "1"}, {"AutoGetMission", "-1"}} {
			steps := len(s.undoWorkspace)
			if _, err := s.SetWorkspaceField("package", edit[0], edit[1]); err != nil {
				t.Fatalf("rows %t %v: %v", rows, edit, err)
			}
			if len(s.undoWorkspace) != steps+1 {
				t.Fatalf("rows %t %v: not one draft step", rows, edit)
			}
		}
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(mod.MissionsFile)))
		if err != nil || !strings.Contains(string(b), "number  = 20\n") || strings.Contains(string(b), " 31\n") {
			t.Fatalf("rows %t written: %v\n%s", rows, err, b)
		}
		reopenedMod(t, dir)
	}
}

func campaignFieldsOf(s *mapInspectionEdits, scope string) map[string]ui.InspectionField {
	out := map[string]ui.InspectionField{}
	for _, f := range s.WorkspaceFields(scope) {
		if f.Key == "AutoGetMission" || f.Key == "landing" {
			out[f.Key] = f
		}
	}
	return out
}

func TestCampaignFieldsThatCannotChangeAreLines(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		_, e := graphFixtureIn(t, entry, nil, []string{"Ford", "Ambush", "Vault"})
		p := filepath.Join(t.TempDir(), "loose.alm")
		writeFileAll(t, p, synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "loose"}))
		if err := e.Open(p); err != nil {
			t.Fatal(err)
		}
		s := e.Document().Edits.(*mapInspectionEdits)
		if _, err := s.SetWorkspaceField("package", "kind", "1"); err != nil {
			t.Fatal(err)
		}
		fields := campaignFieldsOf(s, "package")
		for _, key := range []string{"AutoGetMission", "landing"} {
			f, ok := fields[key]
			if !ok || !f.ReadOnly || len(f.Lines) != 1 {
				t.Fatalf("%s no mod: field %s: shown %t, read-only %t, %d lines", entry, key, ok, f.ReadOnly, len(f.Lines))
			}
			if _, err := s.SetWorkspaceField("package", key, "1"); err == nil || err.Error() != s.inspector.word("editor.graph.readonly") {
				t.Fatalf("%s no mod: %s: %v", entry, key, err)
			}
		}

		f, _, s, _ := graphEditFixture(t, entry)
		for _, scope := range []string{"package", "campaign.10"} {
			for key, field := range campaignFieldsOf(s, scope) {
				if field.ReadOnly {
					t.Fatalf("%s first game: %s %s is read-only", entry, scope, key)
				}
			}
		}
		second, ok := base.Find(base.ROM2EN)
		if !ok {
			t.Fatal("missing ROM2 profile")
		}
		second.Language = entry
		f.x.archives.Base.Profile = second
		if page := s.CampaignGraph(); !page.Unread {
			t.Fatal("the second game's graph is read")
		}
		for _, scope := range []string{"package", "campaign.10"} {
			fields := campaignFieldsOf(s, scope)
			for _, key := range []string{"AutoGetMission", "landing"} {
				f, ok := fields[key]
				if !ok || !f.ReadOnly || len(f.Lines) != 1 {
					t.Fatalf("%s second game: %s %s: shown %t, read-only %t", entry, scope, key, ok, f.ReadOnly)
				}
				before := workspaceBytes(s.workspace)
				if _, err := s.SetWorkspaceField(scope, key, "1"); err == nil || err.Error() != s.inspector.word("editor.graph.unread") {
					t.Fatalf("%s second game: %s %s: %v", entry, scope, key, err)
				}
				if !bytes.Equal(before, workspaceBytes(s.workspace)) {
					t.Fatal("refusal changed the draft")
				}
			}
		}
	}
}

func writtenRows(t *testing.T, s *mapInspectionEdits) string {
	t.Helper()
	d := s.workspace.Project
	b, err := mod.WriteMissions(d.Source, d.Original, d.Missions)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestTheKindChoiceKeepsEachKindsRoute(t *testing.T) {
	_, e := graphFixtureIn(t, "english", nil, []string{"Ford", "Ambush", "Vault"})
	p := filepath.Join(t.TempDir(), "loose.alm")
	writeFileAll(t, p, synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "loose"}))
	if err := e.Open(p); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	if _, err := s.SetWorkspaceField("package", "gold", "300"); err != nil {
		t.Fatal(err)
	}
	guest := s.workspace.Package
	if guest.Kind != mod.MissionGuest || guest.After == 0 || guest.Guest == 0 || guest.Gold != 300 {
		t.Fatalf("guest draft: %+v", guest)
	}
	if _, err := s.SetWorkspaceField("package", "kind", "1"); err != nil {
		t.Fatal(err)
	}
	if r := s.workspace.Package; r.Kind != mod.MissionReplace || r.After != 0 || r.Guest != 0 || r.Chapter != 0 || r.Gold != 0 || len(r.Party) != 0 || r.Class != "" || r.Sex != "" || r.MainParty {
		t.Fatalf("replace draft carries the guest route: %+v", r)
	}
	if _, err := s.SetWorkspaceField("package", "kind", "0"); err != nil {
		t.Fatal(err)
	}
	back := s.workspace.Package
	back.Kept = nil
	if !reflect.DeepEqual(back, guest) {
		t.Fatalf("guest route after the turn back:\n%+v\nwant\n%+v", back, guest)
	}

	const row = "[[mission]]\nkind = \"replace\"\nkey = \"amb\"\nnumber = 20\nmap = \"maps/first.alm\"\nAutoGetMission = -1\nparty = \"main\"\n"
	f, e := graphFixtureIn(t, "english", nil, []string{"Ford", "Ambush", "Vault"})
	f.x.table = companionTable(t)
	f.x.archives.Containers.Overlay(ScenarioRegistry, func() ([]byte, error) { return testPlayRegistry(), nil })
	dir := writeEditorMod(t, t.TempDir(), row)
	writeFileAll(t, filepath.Join(dir, "maps", "first.alm"), synth.ALM(synth.ALMOptions{Width: 24, Height: 24, Name: "first"}))
	if err := f.x.SetMod(dir); err != nil || f.x.mod.refusal != "" {
		t.Fatalf("open: %v %s", err, f.x.mod.refusal)
	}
	if err := e.Open(filepath.Join(dir, "maps", "first.alm")); err != nil {
		t.Fatal(err)
	}
	s = e.Document().Edits.(*mapInspectionEdits)
	if got := writtenRows(t, s); got != row {
		t.Fatalf("untouched row:\n%s", got)
	}
	if _, err := s.SetWorkspaceField("package", "kind", "0"); err != nil {
		t.Fatal(err)
	}
	if g := s.workspace.Package; g.MainParty || g.AutoGetMission != nil {
		t.Fatalf("guest draft keeps the replacing route: %+v", g)
	}
	if got := writtenRows(t, s); strings.Contains(got, "party") || strings.Contains(got, "AutoGetMission") || strings.Contains(got, "replace") {
		t.Fatalf("guest row:\n%s", got)
	}
	if _, err := s.SetWorkspaceField("package", "kind", "1"); err != nil {
		t.Fatal(err)
	}
	if got := writtenRows(t, s); got != row {
		t.Fatalf("replacing row after the turn back:\n%s", got)
	}
	if _, err := mod.ParseMissions("ambush", mod.MissionsFile, []byte(writtenRows(t, s)), nil, "en"); err != nil {
		t.Fatal(err)
	}
}

func TestFollowsAtOnceRefusesANonNumberInWords(t *testing.T) {
	for entry, want := range map[string]string{"english": "Follows at once takes", "russian": "принимает номер миссии"} {
		_, _, s, _ := graphEditFixture(t, entry)
		for _, scope := range []string{"campaign.10", "package"} {
			steps := len(s.undoWorkspace)
			_, err := s.SetWorkspaceField(scope, "AutoGetMission", "abc")
			if err == nil || !strings.Contains(err.Error(), want) || strings.Contains(err.Error(), "strconv") || err.Error() != s.inspector.word("editor.graph.err.successor") {
				t.Fatalf("%s %s: %v", entry, scope, err)
			}
			if len(s.undoWorkspace) != steps {
				t.Fatal("refusal left a draft step")
			}
		}
	}
}
