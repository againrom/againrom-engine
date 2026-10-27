package game

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// originalReplayLimit is the number of loadable mission saves of the corpus, in
// path order, that the replay witnesses take.
const originalReplayLimit = 40

func originalReplayLoad(t *testing.T, raw []byte) (*FrontEnd, bool) {
	t.Helper()
	f := releaseFront(t)
	f.SetTipsOff(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		return nil, false
	}
	a := f.App("original replay")
	a.SetCutscenes(nil)
	if err := a.OpenMission(open); err != nil {
		return nil, false
	}
	return f, true
}

func originalReplaySave(t *testing.T, f *FrontEnd) []byte {
	t.Helper()
	before := f.live.world.Hash()
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "Replay")
	if err != nil || f.live.world.Hash() != before {
		t.Fatal("SAVE failed or changed the World", err)
	}
	return raw
}

// originalReplayTarget is the cell one click names for a group led by e: twelve
// cells right and eight down, or back the other way at the map edge.
func originalReplayTarget(w *sim.World, e sim.Entity) (int, int) {
	b := w.Bounds()
	x, y := int(e.X)+12, int(e.Y)+8
	if x >= int(b.Width)-8 {
		x = int(e.X) - 12
	}
	if y >= int(b.Height)-8 {
		y = int(e.Y) - 8
	}
	return x, y
}

// originalReplayOwn lists up to four living participant units on the map.
func originalReplayOwn(w *sim.World) []sim.Entity {
	var own []sim.Entity
	for _, e := range w.Entities() {
		if len(own) == 4 {
			break
		}
		if e.Alive() && !e.OffMap && e.Owner == sim.SelfSlot {
			own = append(own, e)
		}
	}
	return own
}

func originalReplayCorpus(t *testing.T) []string {
	t.Helper()
	root := os.Getenv("AGAINROM_SAVE_CORPUS")
	if root == "" {
		t.Skip("AGAINROM_SAVE_CORPUS is not set")
	}
	paths, _ := filepath.Glob(filepath.Join(root, "*", "*.sav"))
	sort.Strings(paths)
	return paths
}

// originalReplayRoute orders and ticks one loaded save to its state before SAVE.
type originalReplayRoute func(t *testing.T, f *FrontEnd)

// originalReplayOneClick orders every unit through the map's order entry in one
// click, then runs 200 production ticks.
func originalReplayOneClick(t *testing.T, f *FrontEnd) {
	own := originalReplayOwn(f.live.world)
	if len(own) == 0 {
		return
	}
	x, y := originalReplayTarget(f.live.world, own[0])
	for _, e := range own {
		f.live.enqueue(uint32(e.ID), x, y)
	}
	for range 200 {
		f.live.tick()
	}
}

// originalReplayOwnClicks orders each unit to its own cell, one click each, in
// separate ticks, then runs 200 production ticks.
func originalReplayOwnClicks(t *testing.T, f *FrontEnd) {
	for _, e := range originalReplayOwn(f.live.world) {
		x, y := originalReplayTarget(f.live.world, e)
		f.live.enqueue(uint32(e.ID), x, y)
		f.live.tick()
	}
	for range 200 {
		f.live.tick()
	}
}

// originalReplayRun returns the files whose cold LOAD of the SAVE differs from
// the live World, and how many mission saves it took.
func originalReplayRun(t *testing.T, route originalReplayRoute, verbose bool, witnessed []string) (differ []string, took int) {
	seen := map[string]bool{}
	paths := originalReplayCorpus(t)
	defer func() {
		for _, rel := range witnessed {
			for _, path := range paths {
				if strings.HasSuffix(filepath.ToSlash(path), "/"+rel) && !seen[rel] {
					t.Errorf("%s is in the corpus but not in the replayed population", rel)
				}
			}
		}
	}()
	for _, path := range paths {
		if took == originalReplayLimit {
			break
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		f, ok := originalReplayLoad(t, raw)
		if !ok {
			continue
		}
		took++
		route(t, f)
		for _, rel := range witnessed {
			if strings.HasSuffix(filepath.ToSlash(path), "/"+rel) {
				seen[rel] = true
			}
		}
		cold, ok := originalReplayLoad(t, originalReplaySave(t, f))
		label := filepath.Base(filepath.Dir(path)) + "/" + filepath.Base(path)
		if !ok {
			differ = append(differ, label+" (cold LOAD refused)")
			continue
		}
		if adoptLoadedAllocator(t, f.live.world, cold.live.world).Hash() != cold.live.world.Hash() {
			differ = append(differ, label)
			if verbose {
				t.Log(label)
				currentMenuWorldDiagnostics(t, f.live.world, cold.live.world)
			}
		}
	}
	return differ, took
}

// originalReplayWitnessed are mission saves of the corpus whose replay once
// reached a different World after SAVE and cold LOAD; each must be in the
// population when the corpus holds it.
var originalReplayWitnessed = []string{
	"2026-10-06/game0007-original-m70-boltcoming.sav",
	"2026-10-07/saveorcsdontgo.sav",
}

// Each mission save of the corpus, loaded, ordered through the map's order entry,
// played 200 production ticks, SAVEd and cold LOADed, reaches the World hash the
// live World had before SAVE.
func TestReleaseOriginalSaveReplayColdLoadEqualsLive(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	routes := map[string]originalReplayRoute{"one click": originalReplayOneClick, "one click per unit": originalReplayOwnClicks}
	for name, route := range routes {
		t.Run(name, func(t *testing.T) {
			differ, took := originalReplayRun(t, route, false, originalReplayWitnessed)
			if took == 0 {
				t.Skip("the corpus holds no mission save of this edition")
			}
			t.Logf("%d original mission saves replayed, %d differ", took, len(differ))
			if len(differ) != 0 {
				t.Fatalf("cold LOAD of the SAVE differs from the live World for %d of %d saves: %v", len(differ), took, differ)
			}
		})
	}
}
