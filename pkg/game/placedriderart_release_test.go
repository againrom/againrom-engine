package game

import (
	"fmt"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
)

// A placed rider is drawn with his own class record, the mounted sheet, fresh
// and after SAVE and a cold LOAD. The equipment law derives a foot body class
// for his roster template and that class must not replace his sheet.
func TestReleasePlacedRiderMapSpriteIsHisOwnMountedClass(t *testing.T) {
	for _, mission := range []struct {
		number  int
		lancers []sim.EntityID
	}{
		{71, []sim.EntityID{1, 2, 62}},
		{131, []sim.EntityID{9}},
	} {
		f := releaseFront(t)
		f.Options = OptionsStore{}
		f.SetDeterministicFrames(true)
		a := f.App("placed rider sprite")
		t.Cleanup(a.StopAudio)
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(mission.number)); err != nil {
			t.Fatal(err)
		}
		path, _ := writeOrdinarySAV(t, f, "placed-rider-sprite.sav")
		placedRiderSprites(t, f.live, fmt.Sprintf("mission %d fresh", mission.number), mission.lancers)

		g, b := lancerLoad(t, filepath.Dir(path), filepath.Base(path))
		_ = b
		placedRiderSprites(t, g.live, fmt.Sprintf("mission %d loaded", mission.number), mission.lancers)
	}
}

func placedRiderSprites(t *testing.T, live *mapWorld, when string, ids []sim.EntityID) {
	t.Helper()
	for _, id := range ids {
		e, ok := live.entity(id)
		if !ok {
			t.Fatalf("%s: entity %d absent", when, id)
		}
		want := live.units.Classes[e.TypeID]
		if want == nil {
			t.Fatalf("%s: no class %d", when, e.TypeID)
		}
		var found bool
		for _, draw := range live.entityDraws() {
			if draw.ID != uint32(id) {
				continue
			}
			found = true
			if draw.Art != want {
				t.Errorf("%s: entity %d of type %d draws another class's art", when, id, e.TypeID)
			}
		}
		if !found {
			t.Fatalf("%s: entity %d has no draw", when, id)
		}
	}
}
