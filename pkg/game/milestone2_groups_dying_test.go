package game

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func groups1155DyingFixture(t *testing.T) []byte {
	t.Helper()
	_, raw := groups1155Fixture(t, false)
	source, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	locations, err := source.DocumentActorLocations()
	if err != nil {
		t.Fatal(err)
	}
	changed := 0
	for _, a := range locations {
		if binary.LittleEndian.Uint32(source.Body[a.Off+29:]) != newGroupC1115 {
			continue
		}
		// SAV-TOKEN-034 identifies C independently. SAV-UNITPROG-156 and
		// SAV-DEADLOAD-126 locate the signed HP/timer and stage scalars.
		// Change only this synthetic source; Group and order bytes stay intact.
		p := a.StateOff
		n := int(source.Body[p])
		if n == 255 || p+1+n+55 > len(source.Body) {
			t.Fatal("synthetic Unit state is outside the bounded CString layout")
		}
		p += 1 + n
		binary.LittleEndian.PutUint16(source.Body[p+16:], 0xfff8)
		source.Body[p+46], source.Body[a.ControlOff+18] = 1, 7
		changed++
	}
	if changed != 1 {
		t.Fatal("synthetic dying identity is not unique", changed)
	}
	return source.Marshal()
}

func TestGroups1155DyingBothDoorsRetainSourceAndCurrentGraph(t *testing.T) {
	raw := groups1155DyingFixture(t)
	want, join, _ := groups1155Inputs(t, raw)
	for _, fromMission := range []bool{false, true} {
		t.Run(map[bool]string{false: "title", true: "mission-menu"}[fromMission], func(t *testing.T) {
			f := newGroupFront(t, -1)
			f.SetDeterministicFrames(true)
			app := f.App("restored dying Group")
			if fromMission {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			path, store := filepath.Join(t.TempDir(), "dying-group.sav"), SaveStore{Dir: t.TempDir()}
			if err := os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			// Compare raw initial membership before any Snapshot normalizes and
			// reindexes it. The live oracle independently applies Group append.
			initial := f.live.mission.state.savedDocument
			d := groups1155DocumentDifferences(want, join, initial.Document)
			l, population := groups1155LiveDifferences(want, join, initial, f.live.world)
			if len(d)+len(l) != 0 || population.bound != 3 || population.nulls != 1 || len(population.unavailable) != 0 {
				t.Fatal("dying original LOAD lost the raw/normalized graph", d, l, population)
			}
			actor := newGroupActors1115(t, f.live.world)[newGroupC1115]
			if actor.Alive() || actor.HP != -8 || actor.Decay != sim.DecayFallen || actor.Dwell != 7 || actor.Owner != 2 || actor.HasTarget {
				t.Fatal("LOAD changed the dying tuple, Player suffix owner or active order", actor)
			}
			groups, orders, _ := f.live.world.SavedGroups()
			g := groups[4]
			if g.ID != 5 || g.ContainerID != 2 || g.Owner.Key != newGroupLeft1115 || g.Owner.Owner != 1 || len(g.Members) != 2 || !g.Members[0].Bound || g.Members[0].Entity != actor.ID || g.Members[0].Archive != actor.SourceBinding.ArchiveIndex {
				t.Fatal("dying Group identity/owner/container/member became conflated", g)
			}
			checkCurrent := func(front *FrontEnd) Snapshot {
				t.Helper()
				s := groupDocumentSnapshot(t, front)
				if differences := groups1155CurrentDifferences(want, s.SavedDocument, front.live.world); len(differences) != 0 {
					t.Fatal("dying current Document is unavailable or differs", differences)
				}
				return s
			}
			checkCurrent(f)
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			for cycle := range 2 {
				store = SaveStore{Dir: t.TempDir()}
				save, list, load = f.SaveSeams(store, OriginalStore{}, nil)
				app.SetSaveSeams(save, list, load)
				groups, orders, _ = f.live.world.SavedGroups()
				if err := app.HeadlessKey("escape"); err != nil {
					t.Fatal(err)
				}
				if err := app.HeadlessGameMenuAction("save"); err != nil {
					t.Fatal(err)
				}
				entries, err := store.List()
				if err != nil || len(entries) != 1 {
					t.Fatal("ordinary native SAVE failed", entries, err)
				}
				fresh := newGroupFront(t, -1)
				fresh.SetDeterministicFrames(true)
				freshApp := fresh.App("fresh restored dying Group")
				fs, fl, fn := fresh.SaveSeams(store, OriginalStore{}, nil)
				freshApp.SetSaveSeams(fs, fl, fn)
				groundAppLoad(t, freshApp, fl, entries[0].Name)
				gg, oo, present := fresh.live.world.SavedGroups()
				if !present || !reflect.DeepEqual(groups, gg) || !reflect.DeepEqual(orders, oo) || f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("fresh native LOAD changed the restored Group/order or World")
				}
				// The complete producer adds ordinary reachability roots to the
				// partial source graph. The current comparator checks every actual
				// Group, member, AI byte and Order independently of those roots.
				checkCurrent(fresh)
				start := f.live.world.Tick()
				for range 20 {
					f.live.tick()
					fresh.live.tick()
					if f.live.world.Hash() != fresh.live.world.Hash() {
						t.Fatal("dying Group continuation differs")
					}
					continued := group1155Entity(t, fresh.live.world, actor.ID)
					if continued.Alive() || continued.HasTarget || continued.X != actor.X || continued.Y != actor.Y {
						t.Fatal("saved order ran on a dying actor", continued)
					}
				}
				if f.live.world.Tick() != start+20 {
					t.Fatal("native continuation did not advance 20 ticks")
				}
				checkCurrent(f)
				checkCurrent(fresh)
				t.Logf("cycle %d: 5 Groups, 3 bound identities, source alias/null normalization; dying member retains distinct owner/container, saved order and reachable current Document through menu SAVE/fresh LOAD and 20 ticks", cycle)
				f, app = fresh, freshApp
			}
		})
	}
}
