package archtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The launch and display options of an opened map world are set in one place,
// missionPorts.dressWorld, and the viewer's seams in missionPorts.bindSeams.
// The map list's loader and the mission entry both call them, so a new option
// reaches both routes.
func TestMapWorldSetupHasOneHome(t *testing.T) {
	dir := filepath.Join("..", "..", "pkg", "game")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	once := []string{
		"mw.world.SetSmoothRegen(", "mw.view.SetPathfinding(f.showPathfinding)", "mw.view.SetGraphicsOptions(", "mw.view.SetTextSmoothing(",
		".SetAutocastSink(", ".SetWeaponSwapSink(", ".SetFormationSink(", ".SetDefendSink(",
		".SetStructureUseSink(", ".SetPlayerRetreatSink(", ".SetRetreatSink(", ".SetQuickSpells(",
		".restoreStructureUseMetadata(", ".wireReplies(", ".wireEffects(",
	}
	count := map[string]int{}
	callers := map[string]int{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		src := string(b)
		for _, s := range once {
			if n := strings.Count(src, s); n > 0 && name != "missionentry.go" && name != "missionaudio.go" {
				t.Errorf("%s calls %s outside missionentry.go; set it in missionPorts.dressWorld or bindSeams", name, s)
			}
			count[s] += strings.Count(src, s)
		}
		if name == "frontend.go" || name == "missionentry.go" {
			callers[name+" dressWorld"] += strings.Count(src, ".dressWorld(")
			callers[name+" bindSeams"] += strings.Count(src, ".bindSeams(")
		}
	}
	for _, s := range once {
		if s == ".wireReplies(" || s == ".wireEffects(" {
			continue
		}
		want := 1
		if strings.Contains(s, "f.showPathfinding") {
			want = 0
		}
		if count[s] != want {
			t.Errorf("%s appears %d times in pkg/game, want %d", s, count[s], want)
		}
	}
	for _, k := range []string{"frontend.go dressWorld", "frontend.go bindSeams", "missionentry.go dressWorld", "missionentry.go bindSeams"} {
		if callers[k] != 1 {
			t.Errorf("%s: %d calls, want one", k, callers[k])
		}
	}
}
