package game

import (
	"bytes"
	"testing"

	"againrom/pkg/mod"
)

func campaignDraftUnit(t *testing.T) *mapInspectionEdits {
	t.Helper()
	d, err := mod.ParseMissions("draft", mod.MissionsFile, []byte(graphEditableRows), nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	w := &editorWorkspaceState{Project: &editorModDocument{Source: []byte(graphEditableRows), Original: d.Missions, Missions: []mod.Mission{mod.CloneMission(d.Missions[0])}, Files: map[string][]byte{}}}
	w.Package = modPackageSpec(d.Missions[0], editorPackageSpec{ID: "draft"})
	s := &mapInspectionEdits{inspector: &MapInspector{}, workspace: w, workspaceImage: workspaceBytes(w), savedWorkspace: workspaceBytes(w)}
	return s
}

func TestCampaignMissionRowEditsShareWorkspaceHistory(t *testing.T) {
	s := campaignDraftUnit(t)
	for _, apply := range []func(*editorWorkspaceState) error{
		func(w *editorWorkspaceState) error { return s.editPackageField(w, "AutoGetMission", "20") },
		func(w *editorWorkspaceState) error { return s.editModControl(w, "landing", "2") },
	} {
		before, steps := workspaceBytes(s.workspace), len(s.undoWorkspace)
		if _, err := s.changeWorkspace(apply); err != nil {
			t.Fatal(err)
		}
		after := workspaceBytes(s.workspace)
		if bytes.Equal(before, after) || len(s.undoWorkspace) != steps+1 {
			t.Fatal("row edit is not one workspace step")
		}
		if _, err := s.Undo(); err != nil || !bytes.Equal(before, workspaceBytes(s.workspace)) {
			t.Fatalf("undo: %v", err)
		}
		if _, err := s.Redo(); err != nil || !bytes.Equal(after, workspaceBytes(s.workspace)) {
			t.Fatalf("redo: %v", err)
		}
	}
	src, err := mod.WriteMissions(s.workspace.Project.Source, s.workspace.Project.Original, s.workspace.Project.Missions)
	if err != nil {
		t.Fatal(err)
	}
	want := graphEditableRows + "AutoGetMission = 20\nparty = \"main\"\n"
	if string(src) != want {
		t.Fatalf("row bytes:\n%s\nwant:\n%s", src, want)
	}
}

func TestCampaignInsertedDraftUsesTheSessionWinRoute(t *testing.T) {
	s := campaignDraftUnit(t)
	c := graphCampaign(t)
	if _, err := s.changeWorkspace(func(w *editorWorkspaceState) error {
		s.insertCampaignRow(w, c, 11, "maps/detour.alm")
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	doc := s.workspace.Project
	src, err := mod.WriteMissions(doc.Source, doc.Original, doc.Missions)
	if err != nil {
		t.Fatal(err)
	}
	want := graphEditableRows + "AutoGetMission = 11\n[[mission]]\nkind = \"campaign\"\nkey     = \"mission11\"\nnumber  = 11\nmap     = \"maps/detour.alm\"\nAutoGetMission = 20\nparty = \"main\"\n"
	if string(src) != want {
		t.Fatalf("insert bytes:\n%s\nwant:\n%s", src, want)
	}
	d, err := mod.ParseMissions("draft", mod.MissionsFile, src, nil, "en")
	if err != nil {
		t.Fatal(err)
	}
	c, err = c.withModMissions(d)
	if err != nil {
		t.Fatal(err)
	}
	if got := graphAfter(campaignGraphOf(graphCampaign(t), c, d)); len(got[10]) != 1 || got[10][0] != "mission 11" || len(got[11]) != 1 || got[11][0] != "mission 20" {
		t.Fatalf("draft graph: %v", got)
	}
	session := CampaignSession{Town: NewTown(c)}
	for _, row := range []struct{ won, next int }{{10, 11}, {11, 20}, {20, 0}} {
		next, message := session.recordWin(c, row.won, nil, nil, nil, nil, townArrival{arrive: func() { session.Town.Arrive() }, welcome: func(int) {}})
		if next != row.next || !session.Town.Done(row.won) {
			t.Fatalf("win %d: %d (%s)", row.won, next, message)
		}
		wantRoute := winMission
		if row.next == 0 {
			wantRoute = winTown
		}
		if route := session.routeAfterWin(c, row.won, next); route != wantRoute {
			t.Fatalf("win %d: screen %d want %d", row.won, route, wantRoute)
		}
	}
}
