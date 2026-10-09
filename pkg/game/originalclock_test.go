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

// Compatibility assertions in older source-LOAD witnesses must no longer
// equate "no repair tick" with zero. Read the first wire word directly; do
// not use the new SessionState/import projection to build its expectation.
func rawSavedSubTick1112(t *testing.T, payload []byte) uint64 {
	t.Helper()
	source, err := sav.Open(payload)
	if err != nil || source.World == nil || len(source.Body) < 8 {
		t.Fatal("source clock head", err)
	}
	return uint64(binary.LittleEndian.Uint32(source.Body[0:4]))
}

func clockFixture1112(sub, full uint32, actors ...*poolFixtureActor) []byte {
	body := poolFixtureBody([]*poolFixturePlayer{{}, {groups: [][]*poolFixtureActor{actors}}}, nil)
	binary.LittleEndian.PutUint32(body[0:4], sub)
	binary.LittleEndian.PutUint32(body[4:8], full)
	return savedContainer(body)
}

// Existing synthetic producer tests choose an explicit periodic cut, without
// changing production LOAD or deriving FullTick from SubTick.
func withClockFixture1112(t *testing.T, payload []byte, sub, full uint32) []byte {
	t.Helper()
	source, err := sav.Open(payload)
	if err != nil {
		t.Fatal(err)
	}
	body := append([]byte(nil), source.Body...)
	binary.LittleEndian.PutUint32(body[:4], sub)
	binary.LittleEndian.PutUint32(body[4:8], full)
	return savedContainer(body)
}

func checkWorldClock1112(t *testing.T, w *sim.World, sub, full uint32) {
	t.Helper()
	got, ok := w.SessionClock()
	if !ok || got != (sim.SessionClock{SubTick: sub, FullTick: full}) || w.Tick() != uint64(sub) {
		t.Fatalf("clock %+v present=%t Tick=%d want%d/%d", got, ok, w.Tick(), sub, full)
	}
}

func TestOriginalClockBothDoorsAndLateFailure(t *testing.T) {
	for _, fromMap := range []bool{false, true} {
		f := poolFixtureFront(t, 91)
		app := f.App("1112-clock")
		if fromMap {
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatal(err)
			}
		}
		actor := &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: literalProfile1107()}
		payload := clockFixture1112(9343, 584, actor)
		ms, _, err := loadOriginalMission(f, payload)
		if err != nil {
			t.Fatal(err)
		}
		checkWorldClock1112(t, ms.World, 9343, 584)
		dir, store := t.TempDir(), SaveStore{Dir: t.TempDir()}
		if err := os.WriteFile(filepath.Join(dir, "game1112.sav"), payload, 0600); err != nil {
			t.Fatal(err)
		}
		s, list, load := f.SaveSeams(store, OriginalStore{Dir: dir}, nil)
		app.SetSaveSeams(s, list, load)
		groundAppLoad(t, app, list, "game1112.sav")
		checkWorldClock1112(t, f.live.world, 9343, 584)
		if sun := f.live.view.Sun(); sun.Theta < 0.4799 || sun.Theta > 0.4801 {
			t.Fatalf("first source frame kept zero/seed/current-minute sun: %+v", sun)
		}
		old := f.live
		before, _, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		badProfile := literalProfile1107()
		badProfile.periods[1] = 0
		bad := clockFixture1112(12, 7, &poolFixtureActor{mapID: 91, cell: 0x0605, hp: 10, maxHP: 100, mana: 10, maxMana: 100, profile: badProfile})
		opener, _, err := f.RestoreOriginal(bad)
		if err == nil {
			err = app.OpenMission(opener)
		}
		if err == nil {
			t.Fatal("late profile refusal accepted")
		}
		after, _, err := f.Snapshot(true)
		if err != nil || f.live != old || !reflect.DeepEqual(before, after) {
			t.Fatal("late candidate failure changed live clock/session", err)
		}
		checkWorldClock1112(t, f.live.world, 9343, 584)
	}
}
