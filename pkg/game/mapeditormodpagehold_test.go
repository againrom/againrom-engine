package game

import (
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// A frame of the mod window on the fields page builds the fields of no other
// page: the tab row asks only whether each page holds any, and the answer is
// the one the fields give.
func TestModWindowFrameBuildsNoOtherPageToShowItsTabs(t *testing.T) {
	saved := workspaceScopes
	t.Cleanup(func() { workspaceScopes = saved })
	_, e, spare, _ := makeMapEditor(t, "")
	if !e.ModWindowOpen() {
		if err := e.Key("package"); err != nil {
			t.Fatal(err)
		}
	}
	scopes := []string{ui.ModTerrainScope, ui.ModTextsScope, ui.ModCharactersScope, ui.ModItemsScope}
	for _, scope := range scopes {
		if got, want := spare.WorkspaceHolds(scope), len(spare.WorkspaceFields(scope)) > 0; got != want || !got {
			t.Fatalf("scope %s holds %v, fields say %v", scope, got, want)
		}
	}
	built := map[string]int{}
	rows := slices.Clone(saved)
	for i := range rows {
		r := rows[i]
		if !slices.Contains(scopes, r.scope) {
			continue
		}
		inner := r.fields
		rows[i].fields = func(s *mapInspectionEdits, scope string) []ui.InspectionField {
			built[scope]++
			return inner(s, scope)
		}
	}
	workspaceScopes = rows
	for range 3 {
		e.ChromeImage()
	}
	for scope, n := range built {
		if n != 0 {
			t.Errorf("a frame built the %s page %d times", scope, n)
		}
	}
}
