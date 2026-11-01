package game

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/rules"
	"againrom/pkg/sim"
)

// worldProfileRow is a loaded World's answer to which game it runs and
// whether it holds the Group registry, with its hash when loaded and after
// stepping.
type worldProfileRow struct {
	second, registry bool
	loaded, stepped  uint64
}

func (r worldProfileRow) String() string {
	return fmt.Sprintf("{second: %v, registry: %v, loaded: %#x, stepped: %#x}", r.second, r.registry, r.loaded, r.stepped)
}

// worldProfileRows are the answers of a World each game's original SAV
// fixture loads, recorded before the game was a profile chosen once.
var worldProfileRows = map[string]worldProfileRow{
	"first original":  {second: false, registry: true, loaded: 0xee61c705caf6140, stepped: 0xc40b737404768be1},
	"second original": {second: true, registry: true, loaded: 0x30a4c9b222372ade, stepped: 0x8fbe6ccbff3221b9},
}

// playsSecondGame reports whether every profile answer pkg/game reads is the
// second game's.
func playsSecondGame(w *sim.World) bool {
	return w.SpellGame() == rules.SpellGameSecond && w.NativeMovers() && w.KeepsCastRetain() && w.ResidueOverBookCasts() &&
		w.SavesMissionFog() && !w.RestoresCompanionObjective()
}

func worldProfileAnswers(w *sim.World) worldProfileRow {
	_, _, registry := w.SavedGroups()
	row := worldProfileRow{second: playsSecondGame(w), registry: registry, loaded: w.Hash()}
	for range 300 {
		sim.Step(w, nil)
	}
	row.stepped = w.Hash()
	return row
}

// A World loaded from each game's original SAV fixture answers the game and
// the Group registry as recorded, and its hash when loaded and after 300
// steps is the recorded one.
func TestOriginalSAVWorldsAnswerTheRecordedGameProfile(t *testing.T) {
	got := map[string]worldProfileRow{}
	first := currentPoolFixtureFront(t, 91, 92)
	app := first.App("profile original")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "game9998.sav"), absorbedDeadPayload(), 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := first.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, nil)
	app.SetSaveSeams(save, list, load)
	groundAppLoad(t, app, list, "game9998.sav")
	got["first original"] = worldProfileAnswers(first.live.world)
	second, _, _ := secondMissionLoadFile(t, secondMissionFile(secondMissionFixtureHero(), secondMissionStore(4, 0, -1)))
	got["second original"] = worldProfileAnswers(second.live.world)
	for name, row := range got {
		if want, ok := worldProfileRows[name]; !ok || want != row {
			t.Errorf("%q: %v, want %v", name, row, want)
		}
	}
	if len(worldProfileRows) != len(got) {
		t.Errorf("%d recorded rows, %d loaded Worlds", len(worldProfileRows), len(got))
	}
}
