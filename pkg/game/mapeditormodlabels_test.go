package game

import (
	"maps"
	"os"
	"path/filepath"
	"testing"
)

// Each mission of a mod carries the variable labels of its own map: switching
// to the other mission, and back, shows only that map's labels, and Save As
// writes only that map's label file.
func TestEditorMissionSwitchKeepsEachMapsOwnLabels(t *testing.T) {
	_, e := workspaceFixture(t)
	dir := writeEditorMod(t, t.TempDir(), "[[mission]]\nkey='first'\nnumber=25\nmap='first.alm'\nafter=20\nparty=[{character='npc22'}]\n[[mission]]\nkey='second'\nnumber=26\nmap='second.alm'\nafter=25\nparty=[{character='npc22'}]\n")
	for _, name := range []string{"first", "second"} {
		writeTestPlayInput(t, filepath.Join(dir, name+".alm"), e.Document().SourceBytes)
	}
	writeFile(t, filepath.Join(dir, "first.variables"), []byte(`{"variables":[{"register":99,"label":"FirstOnly"}]}`))
	writeFile(t, filepath.Join(dir, "second.variables"), []byte(`{"variables":[{"register":98,"label":"SecondOnly"}]}`))
	if err := workspaceField(e, "package", 0, dir); err != nil {
		t.Fatal(err)
	}
	s := e.Document().Edits.(*mapInspectionEdits)
	want := map[int]map[uint32]string{1: {99: "FirstOnly"}, 2: {98: "SecondOnly"}}
	for step, index := range []int{1, 2, 1, 2} {
		if step > 0 {
			if _, err := s.selectWorkspaceMission(index); err != nil {
				t.Fatal(err)
			}
		}
		if !maps.Equal(s.workspace.Labels, want[index]) {
			t.Fatalf("mission %d: the workspace labels are %v, want %v", index, s.workspace.Labels, want[index])
		}
		out := filepath.Join(t.TempDir(), "out.alm")
		if _, err := s.SaveAs(out); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(filepath.Dir(out), "out.variables"))
		if err != nil {
			t.Fatalf("mission %d: no label file written: %v", index, err)
		}
		if string(got) != string(encodeMapLabels(want[index])) {
			t.Fatalf("mission %d: Save As wrote %q", index, got)
		}
	}
}
