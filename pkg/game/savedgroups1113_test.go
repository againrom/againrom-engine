package game

import (
	"againrom/pkg/sim"
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// Literal fixture-writer offsets, not values emitted by the importer. The
// archive list A,B,A,C becomes CURRENT B,A,C; each Move cell is distinct from
// the Group cell and from its neighbors. No command is replayed to cause the
// first incoming Move decision.
func savedGroupPayload1113() []byte {
	body, actors := actorRegistryBody1111()
	group := actors[0].off - 98
	body[group+2+0x20], body[group+2+0x45] = 4, 1
	body[group+2+10], body[group+2+11] = 23, 24
	for i, a := range actors {
		ord := a.off + 359
		body[ord+10], body[ord+11] = 18+byte(i), 16+2*byte(i)
		body[ord+8] = 1
		binary.LittleEndian.PutUint32(body[ord+0x50:], 0)
		binary.LittleEndian.PutUint32(body[a.off+513:], 0xb)
	}
	return savedContainer(body)
}

func savedGroupSourceCheck1113(t *testing.T, w *sim.World) {
	t.Helper()
	g, o, present := w.SavedGroups()
	if !present || len(g) != 1 || len(o) != 3 || g[0].ID != 1 || g[0].Selector != 0 || g[0].Owner.Class != 0 || g[0].Authored {
		t.Fatalf("incoming registry: %t %+v %+v", present, g, o)
	}
	var archive []uint16
	for _, m := range g[0].Members {
		archive = append(archive, m.Archive)
	}
	if !reflect.DeepEqual(archive, []uint16{6, 4, 7}) || g[0].AI[0x20] != 4 || g[0].AI[10] != 23 {
		t.Fatal("incoming ordered fields", g)
	}
	for _, ord := range o {
		for _, e := range w.Entities() {
			if e.ID != ord.Entity {
				continue
			}
			var n byte
			switch e.SourceBinding.TypeID {
			case 35:
				n = 0
			case 3:
				n = 1
			case 33:
				n = 2
			default:
				t.Fatal("unknown source", e)
			}
			if ord.State != 0xb || ord.Raw[10] != 18+n || ord.Raw[11] != 16+2*n {
				t.Fatal("actor order bytes", ord)
			}
		}
	}
}

func firstGroupPass1113(t *testing.T, w *sim.World) {
	t.Helper()
	for range 20 {
		if sim.StepTraced(w, nil).Pass {
			return
		}
	}
	t.Fatal("no incoming decision phase")
}

func TestSavedGroups1113BothDoorsOrdinaryMenuFreshLoadAndNextAction(t *testing.T) {
	payload := savedGroupPayload1113()
	f := actorRegistryFront1111(t)
	ms, report, err := loadOriginalMission(f, payload)
	if err != nil {
		t.Fatal(err)
	}
	if report.GroupsRestored != 1 {
		t.Fatal("operator report omitted restored Groups", report)
	}
	savedGroupSourceCheck1113(t, ms.World)
	for _, fromMap := range []bool{false, true} {
		t.Run(map[bool]string{false: "main-menu", true: "mission-menu"}[fromMap], func(t *testing.T) {
			f := actorRegistryFront1111(t)
			app := f.App("saved Groups")
			if fromMap {
				if err := app.OpenMission(f.MissionOpener(10)); err != nil {
					t.Fatal(err)
				}
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "game1113.sav"), payload, 0600); err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game1113.sav")
			savedGroupSourceCheck1113(t, f.live.world)
			firstGroupPass1113(t, f.live.world)
			for _, e := range f.live.world.Entities() {
				if e.SourceBinding.Class == 0 {
					continue
				}
				var n int32
				switch e.SourceBinding.TypeID {
				case 35:
					n = 0
				case 3:
					n = 1
				case 33:
					n = 2
				}
				if !e.HasTarget || e.TargetX != 18+n || e.TargetY != 16+2*n {
					t.Fatal("saved Move did not execute", e)
				}
			}
			// Actual ordinary map command, not source reconstruction replay.
			source := registryActors1111(t, f.live.world)
			f.live.enqueue(uint32(source[35].ID), 20, 16)
			f.live.tick()
			groups, orders, _ := f.live.world.SavedGroups()
			if len(groups) != 2 || len(groups[0].Members) != 2 || groups[1].Members[0].Entity != source[35].ID {
				t.Fatal("ordinary command did not mutate registry", groups)
			}
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 {
				t.Fatal(entries, err)
			}
			fresh := actorRegistryFront1111(t)
			freshApp := fresh.App("fresh saved Groups")
			s, l, next := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(s, l, next)
			groundAppLoad(t, freshApp, l, entries[0].Name)
			gg, oo, present := fresh.live.world.SavedGroups()
			if !present || !reflect.DeepEqual(groups, gg) || !reflect.DeepEqual(orders, oo) || f.live.world.Hash() != fresh.live.world.Hash() {
				t.Logf("present=%t Groups equal=%t Orders equal=%t World %x/%x", present, reflect.DeepEqual(groups, gg), reflect.DeepEqual(orders, oo), f.live.world.Hash(), fresh.live.world.Hash())
				currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
				t.Fatal("ordinary SAVE/fresh LOAD changed current Groups")
			}
			for range 20 {
				f.live.tick()
				fresh.live.tick()
				if f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatal("next action diverged after fresh App LOAD")
				}
			}
		})
	}
}

func savedEscortPayload1113(state, key uint32) []byte {
	body, actors := actorRegistryBody1111()
	group := actors[0].off - 98
	body[group+2+0x20], body[group+2+0x45] = 0, 1
	for _, a := range actors {
		binary.LittleEndian.PutUint32(body[a.off+513:], 0)
	}
	a := actors[0]
	binary.LittleEndian.PutUint32(body[a.off+513:], state)
	binary.LittleEndian.PutUint32(body[a.off+359+0x10:], key)
	body[a.off+359+0x70] = 1
	return savedContainer(body)
}

func checkSavedEscort1113(t *testing.T, w *sim.World, key uint32, targetType uint16) {
	t.Helper()
	actors := registryActors1111(t, w)
	_, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity != actors[35].ID {
			continue
		}
		if !o.EscortBound || o.EscortTarget != actors[targetType].ID || o.Authored || o.RepairStage != 0 || binary.LittleEndian.Uint32(o.Raw[0x10:]) != key || uint32(o.EscortTarget) == key {
			t.Fatal("source-key binding conflated with raw or native identity", o)
		}
		return
	}
	t.Fatal("missing escort order")
}

func savedSourceTypes1113(w *sim.World) map[uint16]sim.Entity {
	out := map[uint16]sim.Entity{}
	for _, e := range w.Entities() {
		if e.SourceBinding.Class != 0 {
			out[e.SourceBinding.TypeID] = e
		}
	}
	return out
}

func TestSavedGroups1113IncomingEscortBothDoorsFreshSaveNextStep(t *testing.T) {
	for _, state := range []uint32{8, 0x11} {
		for _, target := range []struct {
			key uint32
			typ uint16
		}{{1006, 3}, {1007, 33}} {
			payload := savedEscortPayload1113(state, target.key)
			f := actorRegistryFront1111(t)
			ms, _, err := loadOriginalMission(f, payload)
			if err != nil {
				t.Fatal(err)
			}
			checkSavedEscort1113(t, ms.World, target.key, target.typ)
			firstGroupPass1113(t, ms.World)
			if e := savedSourceTypes1113(ms.World)[35]; !e.HasEscortTarget || !e.HasTarget {
				t.Fatal("direct LOAD escort did not dispatch", e)
			}
			for _, fromMap := range []bool{false, true} {
				f := actorRegistryFront1111(t)
				app := f.App("incoming escort")
				if fromMap {
					if err := app.OpenMission(f.MissionOpener(10)); err != nil {
						t.Fatal(err)
					}
				}
				dir := t.TempDir()
				if err := os.WriteFile(filepath.Join(dir, "escort.sav"), payload, 0600); err != nil {
					t.Fatal(err)
				}
				store := SaveStore{Dir: t.TempDir()}
				s, l, next := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
				app.SetSaveSeams(s, l, next)
				groundAppLoad(t, app, l, "escort.sav")
				checkSavedEscort1113(t, f.live.world, target.key, target.typ)
				for cycle := range 2 {
					t.Logf("state=%d target=%d fromMap=%t cycle=%d", state, target.key, fromMap, cycle)
					store = SaveStore{Dir: t.TempDir()}
					s, l, next = f.SaveSeams(store, OriginalStore{}, nil)
					app.SetSaveSeams(s, l, next)
					// The first SAVE precedes the incoming decision. The second
					// saves the actual escort action after the cold continuation.
					if err := app.HeadlessKey("escape"); err != nil {
						t.Fatal(err)
					}
					if err := app.HeadlessGameMenuAction("save"); err != nil {
						t.Fatal(err)
					}
					entries, err := store.List()
					if err != nil || len(entries) != 1 {
						t.Fatal(entries, err, app.HeadlessMessage())
					}
					fresh := actorRegistryFront1111(t)
					freshApp := fresh.App("fresh incoming escort")
					fs, fl, fn := fresh.SaveSeams(store, OriginalStore{}, nil)
					freshApp.SetSaveSeams(fs, fl, fn)
					groundAppLoad(t, freshApp, fl, entries[0].Name)
					if cycle == 0 {
						checkSavedEscort1113(t, fresh.live.world, target.key, target.typ)
					}
					if f.live.world.Hash() != fresh.live.world.Hash() {
						currentMenuWorldDiagnostics(t, f.live.world, fresh.live.world)
						t.Fatal("incoming escort changed at cold LOAD")
					}
					before := savedSourceTypes1113(f.live.world)[35]
					for range 20 {
						f.live.tick()
						fresh.live.tick()
						if f.live.world.Hash() != fresh.live.world.Hash() {
							t.Fatal("incoming escort next step changed after fresh LOAD")
						}
					}
					actors := savedSourceTypes1113(fresh.live.world)
					e := actors[35]
					if !e.HasEscortTarget || e.EscortTarget != actors[target.typ].ID || cycle == 0 && e.X == before.X && e.Y == before.Y {
						t.Fatal("incoming escort did not move toward exact source target", state, target, e)
					}
					f, app = fresh, freshApp
				}
			}
		}
	}
}

func TestSavedGroups1113IncomingEscortMissingKeyDoesNotBorrowID(t *testing.T) {
	for _, key := range []uint32{0, 2, 3, 900, 502, 0xdeadbeef} {
		f := actorRegistryFront1111(t)
		ms, report, err := loadOriginalMission(f, savedEscortPayload1113(8, key))
		if err != nil {
			t.Fatal(err)
		}
		if len(report.GroupIssues) == 0 {
			t.Fatal("missing key not diagnosed", key)
		}
		firstGroupPass1113(t, ms.World)
		if e := registryActors1111(t, ms.World)[35]; e.HasEscortTarget || e.HasTarget {
			t.Fatal("missing key borrowed runtime/map/native ID", key, e)
		}
		_, orders, _ := ms.World.SavedGroups()
		if orders[0].EscortBound || binary.LittleEndian.Uint32(orders[0].Raw[0x10:]) != key {
			t.Fatal("miss did not retain raw key", orders[0])
		}
		if _, err := ms.World.MarshalBinary(); err != nil {
			t.Fatal("unsupported incoming operation prevented SAVE", err)
		}
	}
}

func TestSavedGroups1113ArbitraryPatrolRingBothDoorsSaveBeforeDispatch(t *testing.T) {
	body, actors := actorRegistryBody1111()
	a := actors[0]
	group := a.off - 98
	body[group+2+0x20], body[group+2+0x45] = 0, 1
	for _, actor := range actors {
		binary.LittleEndian.PutUint32(body[actor.off+513:], 0)
	}
	binary.LittleEndian.PutUint32(body[a.off+513:], 0xa)
	ord := a.off + 359
	binary.LittleEndian.PutUint16(body[ord+2:], 0x100f)
	binary.LittleEndian.PutUint32(body[ord+4:], 0x12345678)
	body[ord+8] = 0xb
	// Four literal cells, duplicate first/third. This is the separately
	// serialized order+90 list, not the unrelated GroupAI list.
	binary.LittleEndian.PutUint16(body[a.off+507:], 4)
	ring := []byte{15, 16, 16, 18, 15, 16, 17, 20}
	pos := a.off + 509
	body = append(append(append([]byte(nil), body[:pos]...), ring...), body[pos:]...)
	payload := savedContainer(body)
	for _, fromMap := range []bool{false, true} {
		f := actorRegistryFront1111(t)
		app := f.App("saved arbitrary patrol")
		if fromMap {
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
		}
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "patrol.sav"), payload, 0600); err != nil {
			t.Fatal(err)
		}
		store := SaveStore{Dir: t.TempDir()}
		s, l, next := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
		app.SetSaveSeams(s, l, next)
		groundAppLoad(t, app, l, "patrol.sav")
		_, orders, _ := f.live.world.SavedGroups()
		if !reflect.DeepEqual(orders[0].Patrol, []uint16{0x100f, 0x1210, 0x100f, 0x1411}) || binary.LittleEndian.Uint32(orders[0].Raw[4:]) != 0x12345678 {
			t.Fatal("raw patrol ring/latch changed at LOAD", orders)
		}
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
		if err := app.HeadlessGameMenuAction("save"); err != nil {
			t.Fatal(err)
		}
		entries, err := store.List()
		if err != nil || len(entries) != 1 {
			t.Fatal(entries, err)
		}
		fresh := actorRegistryFront1111(t)
		freshApp := fresh.App("fresh saved patrol")
		fs, fl, fn := fresh.SaveSeams(store, OriginalStore{}, nil)
		freshApp.SetSaveSeams(fs, fl, fn)
		groundAppLoad(t, freshApp, fl, entries[0].Name)
		firstGroupPass1113(t, f.live.world)
		firstGroupPass1113(t, fresh.live.world)
		if f.live.world.Hash() != fresh.live.world.Hash() {
			t.Fatal("patrol first dispatch changed across native SAVE")
		}
		_, got, _ := fresh.live.world.SavedGroups()
		if binary.LittleEndian.Uint16(got[0].Raw[2:]) != 0x1210 || binary.LittleEndian.Uint32(got[0].Raw[4:]) != 1 || !reflect.DeepEqual(got[0].Patrol, orders[0].Patrol) {
			t.Fatal("patrol did not choose first-equal successor and set latch", got[0])
		}
		if e := savedSourceTypes1113(fresh.live.world)[35]; !e.HasTarget || e.TargetX != 16 || e.TargetY != 18 {
			t.Fatal("saved patrol did not issue next movement", e)
		}
	}
}
