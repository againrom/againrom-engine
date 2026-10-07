package game

import "testing"

func TestSecondCompletionUsesTheSelectedOutputTableOnlyForFirstMission(t *testing.T) {
	src := missionSource{mainPrefix + "text/cutpaths.txt": []byte("unused\r\nselected-stem\r\nlater-stem\r\n")}
	for _, mission := range []int{0, 1, 10, 20, 30} {
		want := ""
		if mission == 10 {
			want = "selected-stem"
		}
		if got := secondGameCompletionDirectory(src, mission); got != want {
			t.Fatalf("mission%d selected %q, want %q", mission, got, want)
		}
	}
	if got := secondGameCompletionDirectory(nil, 10); got != "" {
		t.Fatal("missing table manufactured a movie", got)
	}
}
