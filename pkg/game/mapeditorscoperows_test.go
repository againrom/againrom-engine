package game

import (
	"slices"
	"testing"

	"againrom/pkg/ui"
)

// A page scope added only as a row of workspaceScopes shows its fields and
// sets them through the two entries the mod window asks.
func TestWorkspaceScopeAddedAsARowShowsAndSetsItsFields(t *testing.T) {
	saved := workspaceScopes
	t.Cleanup(func() { workspaceScopes = saved })
	var set []string
	workspaceScopes = append(slices.Clone(saved), workspaceScope{"probe",
		func(_ *mapInspectionEdits, scope string) []ui.InspectionField {
			return []ui.InspectionField{{Key: scope + ".field"}}
		},
		func(_ *mapInspectionEdits, scope, key, value string) (*ui.InspectionDocument, error) {
			set = append(set, scope+":"+key+"="+value)
			return nil, nil
		}, nil})
	s := &mapInspectionEdits{}
	if got := s.WorkspaceFields("probe"); len(got) != 1 || got[0].Key != "probe.field" {
		t.Fatalf("fields %+v", got)
	}
	if _, err := s.SetWorkspaceField("probe", "field", "1"); err != nil || !slices.Equal(set, []string{"probe:field=1"}) {
		t.Fatalf("set %v %v", err, set)
	}
}
