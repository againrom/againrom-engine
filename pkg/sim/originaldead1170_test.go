package sim

import (
	"reflect"
	"testing"
)

func TestCurrentSourceTerminal1170SurvivesCompactionAndColdSave(t *testing.T) {
	w := sourceBindingWorld1111(t)
	e := &w.entities[0]
	e.HP, e.Decay, e.Dwell = -600, decayLast, 0
	e.MapUnitID = 91
	w.tick = decayPhase
	Step(w, nil)
	if len(w.Entities()) != 0 {
		t.Fatal("native decay did not remove actor")
	}
	rows := w.OriginalDeadActors()
	if len(rows) != 1 || rows[0].Source.Identity != 0x10000004 || rows[0].Source.ArchiveIndex != 4 || rows[0].Current.Stage != 5 || rows[0].Current.HP != -10001 || rows[0].Current.RuntimeID != 0 {
		t.Fatal("terminal provenance lost", rows)
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(form); err != nil {
		t.Fatal(err)
	}
	for range 20 {
		Step(&cold, nil)
	}
	if len(cold.Entities()) != 0 || !reflect.DeepEqual(cold.OriginalDeadActors(), rows) {
		t.Fatal("terminal actor resurrected or tuple changed")
	}
}
