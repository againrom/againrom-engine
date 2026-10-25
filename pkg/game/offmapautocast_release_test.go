package game

import (
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

// An owner save: two idle autocasting mages, a dragon in reach and a nearer
// off-map actor.
const (
	offMapMageA  = sim.EntityID(225)
	offMapMageB  = sim.EntityID(226)
	offMapDragon = sim.EntityID(148)
	offMapActor  = sim.EntityID(219)
)

func TestReleaseIdleMagesDoNotCastAtAnOffMapActor(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	path := os.Getenv("AGAINROM_WHATISATTACKED_SAV")
	if path == "" {
		t.Skip("AGAINROM_WHATISATTACKED_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "whatisattacked.sav")
	t.Cleanup(app.StopAudio)
	w, _ := f.LiveWorld()
	if f.liveMission != 131 {
		t.Fatal("source save is not mission 131", f.liveMission)
	}
	if e, ok := w.Entity(offMapActor); !ok || !e.OffMap || !e.Alive() {
		t.Fatal("the save does not hold the living off-map actor", e.ID, e.OffMap)
	}
	for _, id := range []sim.EntityID{offMapMageA, offMapMageB} {
		if e, _ := w.Entity(id); e.AutoSpell == 0 || e.HasAttackTarget {
			t.Fatal("the mage is not idle with an autocast spell", id)
		}
	}
	start := w.Tick()
	dragon := false
	for range 160 {
		ogreStep(t, f, app)
		for _, d := range w.NativeSpellDeliverySaveStates() {
			if !d.HasCaster || d.Caster != offMapMageA && d.Caster != offMapMageB || !d.HasTarget {
				continue
			}
			if e, _ := w.Entity(d.Target); e.OffMap && w.Tick() > start+12 {
				t.Fatalf("tick %d: mage %d casts at off-map actor %d", w.Tick(), d.Caster, d.Target)
			}
			dragon = dragon || d.Target == offMapDragon
		}
	}
	if !dragon {
		t.Fatal("no idle cast reached the dragon in reach")
	}
}
