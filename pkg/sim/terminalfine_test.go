package sim

import (
	"reflect"
	"testing"
)

func TestOriginalTerminalFinePositionSurvivesBinaryAndTicks(t *testing.T) {
	w := deadWorld(t)
	d := deadInput(3, 5, -10001)
	d.Source.Class = 2
	d.Source.State.FineX, d.Source.State.FineY = 72, 184
	purse, sacks := w.Purse(1), w.Sacks()
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err != nil {
		t.Fatal(err)
	}
	for cycle := range 2 {
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(form); err != nil {
			t.Fatal("terminal fine binary LOAD", err)
		}
		for tick := range 65 {
			if cold.Hash() != w.Hash() {
				t.Fatalf("cycle %d tick %d changed World", cycle, tick)
			}
			for _, e := range cold.Entities() {
				if e.ID == d.ID {
					t.Fatal("terminal Human resurrected")
				}
			}
			if rows := cold.OriginalDeadActors(); len(rows) != 1 || rows[0].OriginalDeadActor != d || rows[0].Current != d.Source.State {
				t.Fatalf("terminal source/current changed: %+v", rows)
			}
			if cold.Purse(1) != purse || !reflect.DeepEqual(cold.Sacks(), sacks) || len(cold.ActiveEffects()) != 0 {
				t.Fatal("terminal Human replayed loot, rewards or effects")
			}
			Step(w, nil)
			Step(&cold, nil)
		}
		w = &cold
	}
}

func TestOriginalTerminalFinePositionKeepsValidationBounds(t *testing.T) {
	valid := deadInput(3, 5, -10001)
	valid.Source.State.FineX, valid.Source.State.FineY = 72, 184
	for name, alter := range map[string]func(*OriginalDeadActor){
		"virtual fine":  func(d *OriginalDeadActor) { d.Source.MapUnitID = 0 },
		"out of bounds": func(d *OriginalDeadActor) { d.Source.State.Cell = 0x0800 },
		"live runtime":  func(d *OriginalDeadActor) { d.Source.State.RuntimeID = 9 },
		"live health":   func(d *OriginalDeadActor) { d.Source.State.HP = -9999 },
		"timer":         func(d *OriginalDeadActor) { d.Source.State.Timer = 1 },
	} {
		t.Run(name, func(t *testing.T) {
			w := deadWorld(t)
			hash := w.Hash()
			bad := valid
			alter(&bad)
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{bad}); err == nil || w.Hash() != hash {
				t.Fatal("invalid terminal import accepted or changed World", err)
			}
		})
	}
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{valid}); err != nil {
		t.Fatal(err)
	}
	section := make([]byte, w.originalDeadSectionLen())
	w.encodeOriginalDead(section, 0)
	changed := valid.Source.State
	changed.FineX++
	putDeadState(section[62:], changed)
	if _, err := decodeOriginalDead(section[:originalDeadRecordLen], w.bounds, w.entities, w.carried, w.equipment); err == nil {
		t.Fatal("binary LOAD accepted changed terminal fine history")
	}
}
