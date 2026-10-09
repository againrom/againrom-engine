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

// Literal file transcriptions on the two owner inputs that retained a living
// ALM replacement at f566f49. The asserted state is independent of the load
// report and of the decoder's ActorHoldings/ActorGraph values.
func TestReleaseOriginalDying1144OwnerGraphDoesNotResurrect(t *testing.T) {
	for _, tc := range []struct {
		rel, hash string
		mapID     uint16
		hp, maxHP int32
		terminal  bool
	}{
		{"2027-09-07/game0031.sav", "9c829564c15df1f65727ada6411384d70fe585cb534204016e9e910394f95c33", 32, 0, 20, false},
		{"2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed", 54, -8, 30, false},
		{"2027-09-07/game0032.sav", "7375f0c08c8361b2fa32d20564802acba688d5e5a445ab5ca635c609632499ed", 54, -10, 30, true},
	} {
		name := tc.rel
		if tc.terminal {
			name += "/synthetic-terminal-zero-timer"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			path, payload := groundCorpusFile(t, tc.rel, tc.hash)
			if tc.terminal {
				// Review F-1: only HP and timer change in memory. The lawful
				// input remains untouched; both LOAD doors use a temporary copy.
				sf, err := sav.Open(payload)
				if err != nil {
					t.Fatal(err)
				}
				graph, err := sf.ActorGraph()
				if err != nil {
					t.Fatal(err)
				}
				changed := 0
				for _, a := range graph.Actors {
					if a.MapUnitID == tc.mapID {
						p := a.StateOff + 1 + int(sf.Body[a.StateOff])
						binary.LittleEndian.PutUint16(sf.Body[p+16:], 0xfff6)
						sf.Body[a.ControlOff+18] = 0
						changed++
					}
				}
				if changed != 1 {
					t.Fatalf("synthetic boundary changed %d actors", changed)
				}
				payload = sf.Marshal()
				path = filepath.Join(t.TempDir(), filepath.Base(path))
				if err := os.WriteFile(path, payload, 0600); err != nil {
					t.Fatal(err)
				}
			}
			ms, _, err := loadOriginalMission(f, payload)
			if err != nil {
				t.Fatal(err)
			}
			assert := func(w *sim.World) {
				t.Helper()
				e := poolEntity(t, w, tc.mapID)
				if e.Alive() || e.HP != tc.hp || e.MaxHP != tc.maxHP || e.Decay != sim.DecayFallen || e.SourceBinding.Class == 0 {
					t.Fatalf("map unit %d: live HP=%d/%d stage=%d alive=%v, want HP=%d/%d stage=1", tc.mapID, e.HP, e.MaxHP, e.Decay, e.Alive(), tc.hp, tc.maxHP)
				}
				if tc.terminal && e.Dwell != 0 {
					t.Fatal("import replaced the zero source timer")
				}
				if tc.mapID == 54 {
					worn, _ := w.EquippedItems(e.ID)
					if worn[0].Code != 0x8114 || !e.ActorLoad.Present || e.ActorLoad.OwnWeight != 3 || e.Load != 3 {
						t.Fatal("dying unit lost its held weapon or stored weight/load")
					}
				}
			}
			assert(ms.World)
			f.SetDeterministicFrames(true)
			app := f.App("1144 dying owner graph")
			app.Layout(1024, 768)
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: filepath.Dir(path)}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, filepath.Base(path))
			assert(f.live.world)
			if !reflect.DeepEqual(ms.World.Sacks(), f.live.world.Sacks()) || !reflect.DeepEqual(ms.World.OriginalDeadActors(), f.live.world.OriginalDeadActors()) {
				t.Fatal("original LOAD doors disagree on existing loot or late-dead records")
			}
			driver := f.live
			fresh, _ := holdingsNativeFresh(t, f, app, store, nil)
			assert(fresh.live.world)
			initialTick := driver.world.Tick()
			for tick := 0; tick < 64; tick++ {
				driver.tick()
				fresh.live.tick()
				if driver.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("fresh native continuation differs at tick %d", tick)
				}
				if poolEntity(t, driver.world, tc.mapID).Alive() {
					t.Fatal("dying actor resurrected during continuation")
				}
			}
			if driver.world.Tick() == initialTick {
				t.Fatal("continuation did not advance")
			}
			t.Log("both original LOAD doors and fresh native reload retain the dying actor; 64 driver ticks agree")
		})
	}
}
