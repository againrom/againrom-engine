package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/alm"
)

// TestReleaseMapEditorScriptVariablesSaveUnchanged opens every installed
// map: its Scripts tree lists the Conditions and Variables the script holds,
// each variable row named Variable N in the install's words, and a save with
// no edit writes the map's bytes unchanged and no label file beside it.
func TestReleaseMapEditorScriptVariablesSaveUnchanged(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	keys, sources := installedMapBytes(t, x)
	totals := [2]int{}
	for n, key := range keys {
		t.Run(key, func(t *testing.T) {
			e := x.Editor()
			e.Layout(1920, 1080)
			if err := e.Open(key); err != nil {
				t.Fatal(err)
			}
			closeMapPanel(t, e)
			if e.Message() != "" {
				t.Fatalf("the map opened with the note %q", e.Message())
			}
			m, err := alm.Open(sources[n])
			if err != nil {
				t.Fatal(err)
			}
			sc, err := m.Script()
			if err != nil {
				t.Fatal(err)
			}
			sections := expectedScriptSections(e.Document(), sc)
			want := 0
			for _, ids := range sections {
				want += len(ids)
			}
			if got := treeMemberCount(e, "scripts"); got != want {
				t.Fatalf("Scripts members=%d, want %d", got, want)
			}
			lines := e.TreeLines()
			variable := x.inspectionUTF8(x.word("editor.scripts.variable"))
			lead, _, _ := strings.Cut(variable, "%")
			for i := range sections[1] {
				row := e.TreeMemberRow("scripts", len(sections[0])+i)
				if row < 0 || row >= len(lines) || !strings.HasPrefix(strings.TrimSpace(lines[row]), lead) {
					t.Fatalf("variable row %d reads %q, want %q", i, lines[max(0, min(row, len(lines)-1))], lead)
				}
			}
			totals[0], totals[1] = totals[0]+len(sections[1]), totals[1]+len(sections[2])
			dir := t.TempDir()
			target := filepath.Join(dir, "unchanged.alm")
			if _, err := e.Document().Edits.SaveAs(target); err != nil {
				t.Fatal(err)
			}
			if saved, err := os.ReadFile(target); err != nil || !bytes.Equal(saved, sources[n]) {
				t.Fatalf("the save with no edit changed the map: %v", err)
			}
			if _, err := os.Lstat(filepath.Join(dir, "unchanged.variables")); !os.IsNotExist(err) {
				t.Fatalf("the save with no edit wrote a label file: %v", err)
			}
		})
	}
	t.Logf("root=%s maps=%d variables=%d listed conditions=%d", os.Getenv("AGAINROM_ASSETS"), len(keys), totals[0], totals[1])
}
