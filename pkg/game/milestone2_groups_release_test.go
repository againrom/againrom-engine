package game

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func group1155Entity(t *testing.T, world *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range world.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatal("Group movement actor absent", id)
	return sim.Entity{}
}

func group1155MoveCell(t *testing.T, world *sim.World, id sim.EntityID) (int32, int32) {
	t.Helper()
	e := group1155Entity(t, world, id)
	for _, delta := range [][2]int32{{1, 0}, {0, 1}, {-1, 0}, {0, -1}, {1, 1}, {-1, -1}, {-1, 1}, {1, -1}} {
		x, y := e.X+delta[0], e.Y+delta[1]
		if rate, _, adjacent, ok := world.StepRate(id, x, y); adjacent && ok && rate > 0 {
			return x, y
		}
	}
	t.Fatal("natural Group member lacks a traversable adjacent cell")
	return 0, 0
}

func TestReleaseMilestone2Groups1155(t *testing.T) {
	_ = releaseFront(t)
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	want, join, _ := groups1155Inputs(t, raw)
	for _, fromMission := range []bool{false, true} {
		t.Run(map[bool]string{false: "title", true: "mission-menu"}[fromMission], func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			app := f.App("Group acceptance")
			app.Layout(1024, 768)
			if fromMission {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			} else if app.Screen() != ui.ScreenMenu {
				t.Fatal("title LOAD door absent")
			}
			sourcePath := filepath.Join(t.TempDir(), "groups-original.sav")
			if err := os.WriteFile(sourcePath, raw, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(sourcePath)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(sourcePath))
			// No Snapshot has normalized/reindexed this retained initial graph.
			d := groups1155DocumentDifferences(want, join, f.live.mission.state.savedDocument.Document)
			l, population := groups1155LiveDifferences(want, join, f.live.mission.state.savedDocument, f.live.world)
			if len(d)+len(l) != 0 || population.bound == 0 {
				t.Fatal("raw source/load comparison", d, l, population)
			}
			beforeGroups, _, _ := f.live.world.SavedGroups()
			hero := f.live.mission.ids[0]
			beforeActor := group1155Entity(t, f.live.world, hero)
			x, y := group1155MoveCell(t, f.live.world, hero)
			var oldGroup uint32
			var oldMembers int
			for _, g := range beforeGroups {
				for _, m := range g.Members {
					if m.Bound && m.Entity == hero {
						oldGroup, oldMembers = g.ID, len(g.Members)
					}
				}
			}
			if oldGroup == 0 {
				t.Fatal("natural hero Group binding absent")
			}
			// Real mission command dispatch detaches the member and constructs
			// the native movement Group. No import/setter rewrites the fixture.
			f.live.enqueue(uint32(hero), int(x), int(y))
			f.live.tick()
			groups, _, _ := f.live.world.SavedGroups()
			created, detached := false, false
			for _, g := range groups {
				if g.ID == oldGroup {
					detached = len(g.Members) == oldMembers-1
				}
				if g.Authored && len(g.Members) == 1 && g.Members[0].Bound && g.Members[0].Entity == hero && g.ContainerID != 0 {
					created = true
				}
			}
			if !created || !detached || len(groups) != len(beforeGroups)+1 {
				t.Fatal("ordinary Move did not create/detach Group membership")
			}
			checkCurrent := func(front *FrontEnd) Snapshot {
				t.Helper()
				s, _, err := front.Snapshot(true)
				if err != nil {
					t.Fatal(err)
				}
				if d := groups1155CurrentDifferences(want, s.SavedDocument, front.live.world); len(d) != 0 {
					t.Fatal(d)
				}
				return s
			}
			current := checkCurrent(f)
			if err := os.Remove(sourcePath); err != nil {
				t.Fatal(err)
			}
			fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
			reloaded := checkCurrent(fresh)
			if !bytes.Equal(current.World, reloaded.World) || !reflect.DeepEqual(current.SavedDocument, reloaded.SavedDocument) {
				t.Fatal("menu SAVE/fresh LOAD changed current Group graph or World")
			}
			initial := f.live.world.Tick()
			for step := 0; step < 20; step++ {
				f.live.tick()
				fresh.live.tick()
				if f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("Group continuation hash differs at advancing step %d", step)
				}
			}
			if f.live.world.Tick() <= initial {
				t.Fatal("twenty continuation ticks did not advance")
			}
			checkCurrent(f)
			checkCurrent(fresh)
			actor := group1155Entity(t, fresh.live.world, hero)
			if actor.X == beforeActor.X && actor.Y == beforeActor.Y {
				t.Fatal("continued ordinary movement never changed the actor's cell")
			}
			// A fresh native session also accepts the next real movement command.
			x, y = group1155MoveCell(t, fresh.live.world, hero)
			f.live.enqueue(uint32(hero), int(x), int(y))
			fresh.live.enqueue(uint32(hero), int(x), int(y))
			f.live.tick()
			fresh.live.tick()
			if f.live.world.Hash() != fresh.live.world.Hash() {
				t.Fatal("next real movement action differs")
			}
			checkCurrent(fresh)
			t.Logf("%d raw Groups/%d member identities; original LOAD, real Move membership producer, current Document comparison, ordinary menu SAVE, source removal, fresh FrontEnd LOAD,20 advancing equal hashes and next Move", len(want.groups), len(want.actors))
		})
	}
}
