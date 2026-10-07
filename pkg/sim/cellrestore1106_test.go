package sim

import (
	"reflect"
	"testing"
)

func TestOriginalCellTails1106OverlayUnionAtomicityAndNative(t *testing.T) {
	w := entryWorld1084(t, DomainGround, 1)
	w.writeCellTail(tailKey(4, 4), [6]byte{13, 7, 8, 9, 10, 11})
	w.grid[6*16+6] = blockGround
	tails := []CellTail{
		{X: 3, Y: 3, Bytes: [6]byte{13, 9, 8, 7, 6, 5}},
		{X: 6, Y: 6, Bytes: [6]byte{13, 0, 12, 11, 10, 9}},
		{X: 255, Y: 254, Bytes: [6]byte{255, 1, 2, 3, 4, 5}},
		{X: 7, Y: 7, Bytes: [6]byte{26, 6, 5, 4, 3, 2}},
		{X: 3, Y: 3, Bytes: [6]byte{0, 9, 8, 7, 6, 5}},
	}
	if err := w.ImportOriginalCellTails(tails); err != nil {
		t.Fatal(err)
	}
	want := []CellTail{tails[4], {X: 4, Y: 4, Bytes: [6]byte{13, 7, 8, 9, 10, 11}}, tails[1], tails[3], tails[2]}
	if !reflect.DeepEqual(w.CellTails(), want) || w.Tick() != 0 || len(w.ScriptCasts()) != 0 {
		t.Fatalf("overlay = %+v tick %d casts %+v", w.CellTails(), w.Tick(), w.ScriptCasts())
	}
	for _, bad := range []CellTail{{X: -1}, {Y: -1}, {X: 256}, {Y: 256}, {X: 1 << 30}} {
		before := w.Hash()
		if err := w.ImportOriginalCellTails([]CellTail{{X: 3, Y: 3, Bytes: [6]byte{13}}, bad}); err == nil || w.Hash() != before {
			t.Fatalf("late invalid record not atomic: %+v %v", bad, err)
		}
	}
	tails[4].Bytes[0] = 13
	if !reflect.DeepEqual(w.CellTails(), want) {
		t.Fatal("saved input aliases world")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back World
	if err := back.UnmarshalBinary(form); err != nil || back.Hash() != w.Hash() || !reflect.DeepEqual(back.CellTails(), want) {
		t.Fatalf("native overlay roundtrip: %v", err)
	}
	if w.attachFootprint(0, 6, 6) || len(w.casts) != 0 {
		t.Fatal("saved overlay bypassed normal terrain movement refusal")
	}
	for _, xy := range [][2]int32{{3, 3}, {7, 7}} {
		w.requestFootprintCasts(0, xy[0], xy[1])
	}
	if len(w.casts) != 0 {
		t.Fatal("zero or relocation operation requested an ordinary spell")
	}
}

func TestOriginalCellTails1106NextFootprintAndPendingContinuation(t *testing.T) {
	for _, domain := range []Domain{DomainGround, DomainGhost, DomainAir} {
		w := entryWorld1084(t, domain, 2)
		tails := []CellTail{{X: 3, Y: 3, Bytes: [6]byte{13, 0, 7, 8, 29, 30}},
			{X: 4, Y: 4, Bytes: [6]byte{13, 2, 11, 12, 31, 32}}}
		if err := w.ImportOriginalCellTails(tails); err != nil {
			t.Fatal(err)
		}
		for stage := 0; stage < 3; stage++ {
			form, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var back World
			if err := back.UnmarshalBinary(form); err != nil {
				t.Fatal(err)
			}
			var commands []Command
			if stage == 0 {
				commands = []Command{{Kind: KindMoveTo, Entity: 7, X: 3, Y: 3}}
			}
			a, b := StepReported(w, commands), StepReported(&back, commands)
			if !reflect.DeepEqual(a, b) || w.Hash() != back.Hash() {
				t.Fatalf("domain %d stage %d continuation", domain, stage)
			}
			if stage == 0 {
				want := []ScriptCast{{FromX: 7, FromY: 8, Spell: 13, Power: 0, Target: 7, AtUnit: true},
					{FromX: 11, FromY: 12, Spell: 13, Power: 2, Target: 7, AtUnit: true}}
				if domain == DomainAir {
					want = nil
				}
				if got := w.ScriptCasts(); len(got) != len(want) || len(want) > 0 && !reflect.DeepEqual(got, want) {
					t.Fatalf("domain %d casts %+v", domain, got)
				}
			}
		}
		// No local once flag: two actual blocked attachment attempts each
		// visit both footprint records before rejecting the occupied slot.
		if domain != DomainAir {
			w.entities = append(w.entities, Entity{ID: 8, X: 4, Y: 4, HP: 100, MaxHP: 100})
			w.routes = append(w.routes, nil)
			for attempt := 1; attempt <= 2; attempt++ {
				if w.attachFootprint(0, 3, 3) || len(w.casts) != 2*attempt {
					t.Fatalf("blocked repeated attempt %d: %+v", attempt, w.casts)
				}
			}
		}
	}
}
