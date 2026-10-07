package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const (
	turtleMissionNumber = 60
	turtleHolder        = sim.EntityID(85)
)

type turtleSwap struct{ ogre, turtle sim.EntityID }

var turtleSwaps = []turtleSwap{{43, 80}, {42, 81}, {46, 82}}

func turtleState(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	w, ok := f.LiveWorld()
	if !ok {
		t.Fatal("no live World")
	}
	e, ok := w.Entity(id)
	if !ok {
		t.Fatalf("entity %d is not in the World", id)
	}
	return e
}

func turtleSettle(t *testing.T, f *FrontEnd, app *ui.App, ticks int, done func() bool) {
	t.Helper()
	for range ticks {
		if done() {
			return
		}
		ogreStep(t, f, app)
	}
	if !done() {
		t.Fatal("the swap did not happen within", ticks, "App ticks")
	}
}

func TestReleaseHeroOrdinalTurtleSwap(t *testing.T) {
	path := os.Getenv("AGAINROM_OGRES2_SAV")
	if path == "" {
		t.Skip("AGAINROM_OGRES2_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("source ogres2.sav sha256=%x", sha256.Sum256(raw))
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "ogres2.sav")
	t.Cleanup(app.StopAudio)
	w, _ := f.LiveWorld()
	if f.liveMission != turtleMissionNumber {
		t.Fatal("source save is not mission 60", f.liveMission)
	}

	holder := turtleState(t, f, turtleHolder)
	carried, _ := w.Carried(turtleHolder)
	has3617 := false
	for _, code := range carried {
		has3617 = has3617 || code == 3617
	}
	if !has3617 {
		t.Fatal("entity 85 does not carry the Metamorph", carried)
	}
	var bound []int
	for i, c := range w.Script().Checks() {
		if c.Op == sim.ScriptCheckItemTest && c.HasItem && c.Item == 3617 {
			if !c.HasUnit || c.Unit != turtleHolder {
				t.Fatalf("check %d tests the Metamorph on %+v, want the party member holding it", i, c)
			}
			bound = append(bound, i)
		}
	}
	if len(bound) != 1 {
		t.Fatal("expected one Metamorph item test", bound)
	}
	t.Logf("hero ordinal 2 resolves to entity %d (type %d, face byte per party member, carried %v)", holder.ID, holder.TypeID, carried)
	for _, s := range turtleSwaps {
		if _, present := w.Entity(s.ogre); !present {
			t.Logf("ogre %d is not in the source World (dead and removed); its swap is not exercised", s.ogre)
			continue
		}
		if o := turtleState(t, f, s.ogre); o.OffMap || o.TypeID != 66 {
			t.Fatalf("ogre %d is not on the map before the swap: %+v", s.ogre, o)
		}
		if u := turtleState(t, f, s.turtle); !u.OffMap {
			t.Fatalf("turtle %d is on the map before the swap", s.turtle)
		}
	}
	for _, latch := range []int32{16, 17, 18} {
		if w.ScriptLatched(latch) {
			t.Fatalf("trigger %d already fired in the source", latch)
		}
	}
	for range 8 {
		ogreStep(t, f, app)
	}
	if o := turtleState(t, f, 43); o.OffMap {
		t.Fatal("the Ogre was swapped with the Metamorph holder far away")
	}

	ogre := turtleState(t, f, 43)
	if err := w.HeadlessPlace(turtleHolder, ogre.X-2, ogre.Y); err != nil {
		t.Fatal(err)
	}
	turtleSettle(t, f, app, 64, func() bool { return turtleState(t, f, 80).OffMap == false })
	if o := turtleState(t, f, 43); !o.OffMap {
		t.Fatal("Ogre 43 is still on the map after the turtle arrived", o)
	}
	turtle := turtleState(t, f, 80)
	t.Logf("turtle 80 now at (%d,%d) type %d; ogre 43 off map; holder at (%d,%d)", turtle.X, turtle.Y, turtle.TypeID, turtleState(t, f, turtleHolder).X, turtleState(t, f, turtleHolder).Y)
	if turtle.TypeID == 66 {
		t.Fatal("the turtle kept the ogre type")
	}
	if !w.ScriptLatched(16) || w.ScriptLatched(17) || w.ScriptLatched(18) {
		t.Fatal("trigger latches after the first swap are wrong")
	}

	dir, saved := ogreF2Save(t, f, app)
	coldPath := filepath.Join(dir, "ogre-current.sav")
	f2, app2 := ogreColdApp(t, coldPath)
	w2, _ := f2.LiveWorld()
	if f2.liveMission != turtleMissionNumber || !w2.ScriptLatched(16) {
		t.Fatal("cold LOAD lost the mission or the first swap", f2.liveMission)
	}
	if turtleState(t, f2, 43).OffMap == false || turtleState(t, f2, 80).OffMap {
		t.Fatal("cold LOAD did not keep the turtle in place of the Ogre")
	}
	var again []int
	for i, c := range w2.Script().Checks() {
		if c.Op == sim.ScriptCheckItemTest && c.HasItem && c.Item == 3617 && c.HasUnit && c.Unit == turtleHolder {
			again = append(again, i)
		}
	}
	if len(again) != 1 {
		t.Fatal("cold LOAD lost the Metamorph binding", again)
	}
	t.Logf("SAV bytes %d sha256=%x", len(saved), sha256.Sum256(saved))

	second := turtleState(t, f2, 42)
	if err := w2.HeadlessPlace(turtleHolder, second.X-2, second.Y); err != nil {
		t.Fatal(err)
	}
	turtleSettle(t, f2, app2, 64, func() bool { return turtleState(t, f2, 81).OffMap == false })
	if o := turtleState(t, f2, 42); !o.OffMap {
		t.Fatal("Ogre 42 is still on the map after the post-LOAD swap")
	}
	if !w2.ScriptLatched(17) {
		t.Fatal("the second trigger did not latch after the cold LOAD")
	}
	t.Log(fmt.Sprintf("post-LOAD swap: ogre 42 off map, turtle 81 at (%d,%d)", turtleState(t, f2, 81).X, turtleState(t, f2, 81).Y))
}
