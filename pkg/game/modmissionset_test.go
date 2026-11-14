package game

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

// missionKeys names each mission of mod id in list.
func missionKeys(list []mod.Mission, id string) []string {
	var out []string
	for _, m := range list {
		if m.Mod != id {
			continue
		}
		key := m.RowID()
		if m.ByFileName() {
			key = "by name " + key
		}
		out = append(out, fmt.Sprintf("%s %d", key, m.Number))
	}
	slices.Sort(out)
	return out
}

// The open, the graph's draft and the fields page read one mission set.
func TestEditorOpenDraftAndFieldsShareOneMissionSet(t *testing.T) {
	for _, entry := range []string{"english", "russian"} {
		t.Run(entry, func(t *testing.T) {
			_, s, _ := numberNamedFixture(t, entry, true)
			x := s.inspector
			id := x.mod.id
			want := []string{"by name maps/20.alm 20", "first 10"}
			if got := missionKeys(x.mod.missions.Missions, id); !slices.Equal(got, want) {
				t.Fatalf("open: %q, want %q", got, want)
			}
			listed, _, err := x.mod.description().Listed(x.missionAnswer())
			if err != nil || !slices.Equal(missionKeys(listed.Missions, id), want) {
				t.Fatalf("the description lists %q, %v", missionKeys(listed.Missions, id), err)
			}
			flow, err := s.missionDraft(cloneWorkspace(s.workspace))
			if err != nil {
				t.Fatal(err)
			}
			if got := missionKeys(flow.missions.Missions, id); !slices.Equal(got, want) {
				t.Fatalf("graph draft: %q, want %q", got, want)
			}
			fields, err := x.modMissions(x.mod.description(), s.workspace.Project.Missions)
			if err != nil {
				t.Fatal(err)
			}
			if got := missionKeys(fields.Missions, id); !slices.Equal(got, want) {
				t.Fatalf("mod fields: %q, want %q", got, want)
			}
			if got := missionKeys(x.mod.flow.missions.Missions, id); !slices.Equal(got, want) {
				t.Fatalf("the open's campaign flow: %q, want %q", got, want)
			}
		})
	}
}

// A key added to the key table is an editor field with no editor edit.
func TestAManifestKeyAddedToTheTableIsAnEditorField(t *testing.T) {
	saved := mod.ManifestKeys
	t.Cleanup(func() { mod.ManifestKeys = saved })
	mod.ManifestKeys = append(slices.Clone(saved), mod.ManifestKey{Name: "launch", Value: mod.ManifestString, Edit: mod.EditField,
		Get: func(m mod.Manifest) any { return m.Entry }, Set: func(m *mod.Manifest, v any) { m.Entry = v.(string) }})
	p := editorPackageSpec{ID: "keyed", Title: "Keyed", Manifest: defaultPackageManifest()}
	source, err := p.manifestBytes(nil)
	if err != nil {
		t.Fatal(err)
	}
	s := &mapInspectionEdits{inspector: &MapInspector{}}
	s.workspace = &editorWorkspaceState{Package: p, Project: &editorModDocument{Manifest: source, Files: map[string][]byte{}, Selected: -1}}
	s.workspaceImage = workspaceBytes(s.workspace)
	if !isManifestField("launch") || !slices.ContainsFunc(s.manifestFields(), func(f ui.InspectionField) bool { return f.Key == "launch" }) {
		t.Fatal("the new key is no field of the mod's page")
	}
	if _, err := s.SetWorkspaceField("package", "launch", "start.star"); err != nil {
		t.Fatal(err)
	}
	out, err := s.workspace.Package.manifestBytes(source)
	if err != nil {
		t.Fatal(err)
	}
	m, err := mod.ParseManifest(out, true)
	if err != nil || !strings.Contains(string(out), `launch = "start.star"`) || m.Entry != "start.star" {
		t.Fatalf("the manifest\n%s\nparses to %+v, %v", out, m, err)
	}
}

// The editor offers both games' bases; a new package takes the install's.
func TestEditorOffersTheBasesOfBothGames(t *testing.T) {
	s := &mapInspectionEdits{inspector: &MapInspector{}}
	s.workspace = &editorWorkspaceState{Package: editorPackageSpec{ID: "bases", Title: "Bases", Manifest: defaultPackageManifest()}}
	var offered []string
	for _, f := range s.manifestFields() {
		if f.Key == "applies-to" {
			for _, c := range f.Choices {
				offered = append(offered, c.Label)
			}
		}
	}
	if want := []string{"common", "rom1", "rom1-en", "rom1-ru", "rom2", "rom2-en", "rom2-ru"}; !slices.Equal(offered, want) {
		t.Fatalf("applies-to offers %q, want %q", offered, want)
	}
	for _, tc := range []struct {
		game base.Game
		want string
	}{{base.GameROM1, "rom1"}, {base.GameROM2, "rom2"}} {
		x := &MapInspector{archives: &Archives{}}
		x.archives.Base.Profile.Game = tc.game
		if got := x.packageBase(); got != tc.want {
			t.Fatalf("a package on a %q install applies to %q, want %q", tc.game, got, tc.want)
		}
	}
	if got := (&MapInspector{}).packageBase(); got != "rom1" {
		t.Fatalf("with no install a package applies to %q", got)
	}
}
