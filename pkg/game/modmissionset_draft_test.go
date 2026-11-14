package game

import (
	"strings"
	"testing"
)

func TestCampaignGraphDraftReadsADeletedMapAsMissing(t *testing.T) {
	_, s, _ := numberNamedFixture(t, "english", true)
	w := cloneWorkspace(s.workspace)
	name := ""
	for _, m := range w.Project.Missions {
		if !m.Off && m.Map != "" {
			name = m.Map
			break
		}
	}
	if name == "" {
		t.Fatal("no row with a map")
	}
	if w.Project.Files == nil {
		w.Project.Files = map[string][]byte{}
	}
	w.Project.Files[name] = nil
	_, err := s.missionDraft(w)
	if err == nil || !strings.Contains(err.Error(), s.inspector.word("editor.mod.err.mapmissing")) {
		t.Fatalf("the graph draft of a deleted map: %v", err)
	}
}
