package sim

import (
	"bytes"
	"reflect"
	"testing"
)

func TestAbsorbedDeadTupleImportsAsTerminal(t *testing.T) {
	w := deadWorld(t)
	authored := deadInput(1, 2, -10001)
	virtual := deadInput(9, 3, -10001)
	virtual.Source.MapUnitID = 0
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{authored, virtual}); err != nil {
		t.Fatal("absorbed dead tuple refused:", err)
	}
	terminal := DeadActorState{Cell: 0x0304, FineX: 128, FineY: 128, Stage: 5, HP: -10001}
	got := w.OriginalDeadActors()
	if len(got) != 2 {
		t.Fatalf("dead records = %+v", got)
	}
	for _, d := range got {
		if d.Current != terminal || d.Source.State != terminal {
			t.Fatalf("record %d holds %+v / %+v, want the terminal tuple", d.ID, d.Current, d.Source.State)
		}
	}
	if indexOfEntity(w.entities, 1) >= 0 || indexOfEntity(w.entities, 9) >= 0 {
		t.Fatal("an absorbed body kept an entity")
	}
	cold := dpReload(t, w)
	if cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.OriginalDeadActors(), got) {
		t.Fatal("terminal tuple failed the binary roundtrip")
	}
	for range 64 {
		Step(cold, nil)
	}
	if !reflect.DeepEqual(cold.OriginalDeadActors(), got) {
		t.Fatal("terminal tuple changed in play", cold.OriginalDeadActors())
	}
}

func TestAbsorbedDeadTupleKeepsOtherRefusals(t *testing.T) {
	for name, change := range map[string]func(*OriginalDeadActor){
		"stage 1":          func(d *OriginalDeadActor) { d.Source.State.Stage = 1 },
		"other health":     func(d *OriginalDeadActor) { d.Source.State.HP = -10002 },
		"terminal residue": func(d *OriginalDeadActor) { d.Source.State.HP = -10017 },
		"signed timer":     func(d *OriginalDeadActor) { d.Source.State.Timer = -128 },
		"position":         func(d *OriginalDeadActor) { d.Source.State.Cell = 0x0800 },
		"virtual fine":     func(d *OriginalDeadActor) { d.Source.MapUnitID, d.ID, d.Source.State.FineX = 0, 9, 64 },
	} {
		t.Run(name, func(t *testing.T) {
			w := deadWorld(t)
			before := w.encode()
			d := deadInput(1, 2, -10001)
			change(&d)
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{d}); err == nil {
				t.Fatal("accepted", d.Source.State)
			}
			if !bytes.Equal(before, w.encode()) {
				t.Fatal("refused import changed the World")
			}
		})
	}
}

func TestAbsorbedTerminalRowImportsAsTerminal(t *testing.T) {
	for _, tc := range []struct {
		row  CurrentTerminalActor
		want uint8
	}{
		{CurrentTerminalActor{ID: 7, Cell: 0x0304, HP: -10001, Stage: 2, MaxHP: 30, MaxHPPresent: true}, 5},
		{CurrentTerminalActor{ID: 7, Cell: 0x0304, HP: -10001, Stage: 4}, 5},
		{CurrentTerminalActor{ID: 7, Cell: 0x0304, HP: -10002, Stage: 2}, 2},
		{CurrentTerminalActor{ID: 7, Cell: 0x0304, HP: -10001, Stage: 1}, 1},
	} {
		w := deadWorld(t)
		if err := w.ImportCurrentTerminalActors([]CurrentTerminalActor{tc.row}); err != nil {
			t.Fatal(err)
		}
		want := tc.row
		want.Stage = tc.want
		if got := w.CurrentTerminalActors(); !reflect.DeepEqual(got, []CurrentTerminalActor{want}) {
			t.Fatalf("row %+v held as %+v, want %+v", tc.row, got, want)
		}
		if cold := dpReload(t, w); !reflect.DeepEqual(cold.CurrentTerminalActors(), []CurrentTerminalActor{want}) {
			t.Fatal("held row failed the binary roundtrip")
		}
	}
}
