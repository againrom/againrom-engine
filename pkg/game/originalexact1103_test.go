package game

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
)

func TestOriginalExact1103SparseAppLoadSaveFreshLoad(t *testing.T) {
	for _, n := range []int{1, 2} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			owner := &poolFixturePlayer{}
			npcs := &poolFixturePlayer{}
			for i := 0; i < n; i++ {
				npcs.groups = append(npcs.groups, []*poolFixtureActor{{mapID: uint16(91 + i), cell: uint16(0x0907 + i),
					hp: uint16(7 + i), maxHP: 31, mana: 5, maxMana: 23, human: i == 1}})
			}
			session := make([]byte, 4374)
			session[400+17] = 1
			session[1856+3*50+4] = 1
			// The literal fixture has no terrain records and includes leading
			// nulls and a repeated human Player, but only one human object.
			body := poolFixtureBodyWithSession([]*poolFixturePlayer{nil, owner, nil, npcs, owner}, nil, session)
			payload := savedContainer(body)
			originals := t.TempDir()
			if err := os.WriteFile(filepath.Join(originals, "game9999.sav"), payload, 0o600); err != nil {
				t.Fatal(err)
			}
			f := currentPoolFixtureFront(t, 91, 92)
			app := f.App("exact-sav")
			store := SaveStore{Dir: t.TempDir()}
			save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
			app.SetSaveSeams(save, list, load)
			groundAppLoad(t, app, list, "game9999.sav")
			assert := func(front *FrontEnd) {
				t.Helper()
				w := front.live.world
				for i := 0; i < n; i++ {
					e := poolEntity(t, w, uint16(91+i))
					if e.X != int32(7+i) || e.Y != 9 || e.HP != int32(7+i) {
						t.Fatalf("NPC%d position/pools = (%d,%d), HP%d", i, e.X, e.Y, e.HP)
					}
				}
				if n == 1 {
					e := poolEntity(t, w, 92)
					if e.X != 16 || e.Y != 16 {
						t.Fatal("unmatched map actor was moved")
					}
				}
				if !w.ScriptLatched(17) || !w.Relations().Hostile(3, 4) || w.Relations().Hostile(4, 3) {
					t.Fatal("empty terrain reset supported session state")
				}
			}
			assert(f)
			if f.live.world.Tick() != rawSavedSubTick1112(t, payload) {
				t.Fatal("LOAD advanced before restored state was visible")
			}
			hash := f.live.world.Hash()
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessGameMenuAction("save"); err != nil {
				t.Fatal(err)
			}
			entries, err := store.List()
			if err != nil || len(entries) != 1 || filepath.Ext(entries[0].Name) != ".sav" {
				t.Fatalf("ordinary SAVE: %+v %v: %s", entries, err, app.HeadlessMessage())
			}
			fresh := currentPoolFixtureFront(t, 91, 92)
			freshApp := fresh.App("fresh-exact-sav")
			fsave, flist, fload := fresh.SaveSeams(store, OriginalStore{}, nil)
			freshApp.SetSaveSeams(fsave, flist, fload)
			groundAppLoad(t, freshApp, flist, entries[0].Name)
			assert(fresh)
			if fresh.live.world.Hash() != hash {
				t.Fatal("fresh native LOAD changed world hash")
			}
			for range 32 {
				f.live.tick()
				fresh.live.tick()
				if f.live.world.Hash() != fresh.live.world.Hash() {
					t.Fatalf("continuation diverged at %d", f.live.world.Tick())
				}
			}
			ms, report, err := loadOriginalMission(f, payload)
			if err != nil || ms == nil || report.Joined != n || report.Moved != n || !report.SessionApplied {
				t.Fatalf("diagnostic route: %+v %v", report, err)
			}
		})
	}
}

func TestOriginalExact1103PositionJoinIsAtomicAndClassFiltered(t *testing.T) {
	a := &poolFixtureActor{mapID: 91, cell: 0x0907, hp: 7, maxHP: 31}
	b := &poolFixtureActor{mapID: 92, cell: 0x0a08, hp: 9, maxHP: 31}
	for _, duplicateTarget := range []bool{false, true} {
		if !duplicateTarget {
			b.mapID = 91
		} else {
			b.mapID = 92
		}
		f, err := sav.Open(poolFixtureSave(a, b))
		if err != nil {
			t.Fatal(err)
		}
		m := &alm.Map{Units: []alm.Unit{{UnitID: 91, X: 128, Y: 128}, {UnitID: 92, X: 256, Y: 256}}}
		if duplicateTarget {
			m.Units[1].UnitID = 91
		}
		before := append([]alm.Unit(nil), m.Units...)
		var report OriginalSaveResume
		if err := applyOriginalPositions(m, f, &report); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("duplicate join accepted: %v", err)
		}
		if !reflect.DeepEqual(before, m.Units) || report.Joined != 0 || report.Moved != 0 {
			t.Fatal("join mutated its target before rejecting ambiguity")
		}
	}
	// An absent NPC is ignored, never created. A terminal actor with a
	// colliding ID is not a living join source.
	b.mapID, b.stage = 91, 5
	f, err := sav.Open(poolFixtureSave(a, b))
	if err != nil {
		t.Fatal(err)
	}
	m := &alm.Map{Units: []alm.Unit{{UnitID: 91}, {UnitID: 999, X: 123, Y: 234}}}
	var report OriginalSaveResume
	if err := applyOriginalPositions(m, f, &report); err != nil || len(m.Units) != 2 || report.Joined != 1 || m.Units[1].X != 123 {
		t.Fatalf("eligibility boundary: %+v %v", report, err)
	}
}
