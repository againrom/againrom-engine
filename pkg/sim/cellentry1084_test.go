package sim

import (
	"reflect"
	"testing"
)

func entryWorld1084(t *testing.T, domain Domain, side uint8) *World {
	t.Helper()
	w, err := NewSpelledWorld(1084, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
		[]Entity{{ID: 7, X: 2, Y: 3, HP: 1000, MaxHP: 1000, Domain: domain, TokenSize: side}}, nil,
		[]SpellRule{{ID: 13, School: 3, DamageMin: 20, DamageMax: 20, Damaging: true, TargetsUnit: true}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.DeclareCellTails([]CellTail{{X: 3, Y: 3, Bytes: [6]byte{13, 1, 9, 10, 11, 12}}}); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestCellEntry1084MovementDomainsStandingAndReentry(t *testing.T) {
	for _, domain := range []Domain{DomainGround, DomainGhost, DomainAir} {
		t.Run(string(rune('0'+domain)), func(t *testing.T) {
			w := entryWorld1084(t, domain, 1)
			for cycle := 0; cycle < 2; cycle++ {
				before := w.entities[0].HP
				Step(w, []Command{{Kind: KindMoveTo, Entity: 7, X: 3, Y: 3}})
				casts := w.ScriptCasts()
				if domain == DomainAir {
					if len(casts) != 0 {
						t.Fatal("air attachment requested a cast")
					}
					return
				}
				if len(casts) != 1 || casts[0] != (ScriptCast{FromX: 9, FromY: 10, Spell: 13, Power: 1, Target: 7, AtUnit: true}) {
					t.Fatalf("entry cast = %+v", casts)
				}
				report := StepReported(w, nil)
				if len(report.ScriptCasts) != 1 || report.ScriptCasts[0] != (ScriptCastEvent{Spell: 13, FromX: 9, FromY: 10, ToX: 3, ToY: 3}) || w.entities[0].HP >= before {
					t.Fatalf("no ordinary effect: report=%+v hp=%d before=%d", report.ScriptCasts, w.entities[0].HP, before)
				}
				after := w.entities[0].HP
				for i := 0; i < 20; i++ {
					Step(w, nil)
				}
				if w.entities[0].HP != after || len(w.casts) != 0 {
					t.Fatal("standing rearmed the cell")
				}
				Step(w, []Command{{Kind: KindMoveTo, Entity: 7, X: 2, Y: 3}})
				if len(w.casts) != 0 {
					t.Fatal("nearby cell requested a cast")
				}
			}
		})
	}
}

func TestCellEntry1084AttemptPrecedesOccupancyButNotSearch(t *testing.T) {
	w := entryWorld1084(t, DomainGround, 2)
	w.entities = append(w.entities, Entity{ID: 8, X: 4, Y: 4, HP: 100, MaxHP: 100})
	w.routes = append(w.routes, nil)
	// The selected anchor is open; the far footprint corner is occupied.
	for i := 0; i < 10; i++ {
		if w.placementOpen(0, 3, 3) {
			t.Fatal("occupied footprint accepted")
		}
	}
	if len(w.casts) != 0 {
		t.Fatal("read-only fit probes cast")
	}
	if w.placeAt(0, 3, 3) || w.entities[0].X != 2 || len(w.casts) != 1 {
		t.Fatalf("failed attach did not retain one cast: pos=%d casts=%+v", w.entities[0].X, w.casts)
	}
	if w.casts[0].Target != 7 {
		t.Fatal("blocked attempt targeted occupant instead of entering actor")
	}
	// Another attempt is another cast. No cooldown, consumed flag or tower id.
	w.placeAt(0, 3, 3)
	if len(w.casts) != 2 || w.cellTails[0].Bytes != ([6]byte{13, 1, 9, 10, 11, 12}) {
		t.Fatal("repeat attempt mutated or consumed the binding")
	}
}

func TestCellEntry1084UsesOrdinaryProtectionAndEveryCoveredCell(t *testing.T) {
	w := entryWorld1084(t, DomainGhost, 2)
	w.entities[0].Owner = 9
	w.entities[0].Protection[2] = 100 // Air protection is not flat absorption.
	if err := w.DeclareCellTails([]CellTail{{X: 4, Y: 3, Bytes: [6]byte{13, 99, 15, 15}}}); err != nil {
		t.Fatal(err)
	}
	w.requestFootprintCasts(0, 3, 3)
	if len(w.casts) != 2 || w.casts[0].Power != 1 || w.casts[1].Power != 99 {
		t.Fatalf("footprint did not visit both cells in row order: %+v", w.casts)
	}
	report := StepReported(w, nil)
	if len(report.ScriptCasts) != 2 || w.entities[0].HP != 1000 || w.entities[0].SpellFXSpell != 13 {
		t.Fatalf("cell cast bypassed ordinary immunity or invented an owner/range gate: %+v hp=%d", report.ScriptCasts, w.entities[0].HP)
	}
}

func TestCellEntry1084FootprintPayloadAndInitialWriterGuards(t *testing.T) {
	w := entryWorld1084(t, DomainGround, 2)
	for _, spell := range []byte{0, 26, 255} {
		w.writeCellTail(tailKey(3, 3), [6]byte{spell})
		w.requestFootprintCasts(0, 2, 2)
		if len(w.casts) != 0 {
			t.Fatalf("disabled/missing spell %d cast", spell)
		}
	}
	w.writeCellTail(tailKey(3, 3), [6]byte{13, 0, 9, 10})
	w.requestFootprintCasts(0, 2, 2) // non-anchor far corner
	if len(w.casts) != 1 || w.casts[0].Power != 0 {
		t.Fatal("non-anchor cell lost or zero power became instant 21's 99")
	}
	form, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var zeroPower World
	if err := zeroPower.UnmarshalBinary(form); err != nil || zeroPower.Hash() != w.Hash() {
		t.Fatalf("zero-power pending cast did not survive native decode: %v", err)
	}
	w.grid[6*16+6] = blockGround
	if err := w.DeclareCellTails([]CellTail{{X: 6, Y: 6, Bytes: [6]byte{13}}}); err != nil || len(w.cellTails) != 1 {
		t.Fatalf("initial block bit accepted a binding: %v %+v", err, w.cellTails)
	}
	before := w.Hash()
	for _, x := range []int32{-1, 16, 256, 1 << 30} {
		if err := w.DeclareCellTails([]CellTail{{X: x, Y: 3}}); err == nil || w.Hash() != before {
			t.Fatalf("invalid binding x=%d not atomically refused: %v", x, err)
		}
	}
}

func TestCellEntry1084NativeBeforePendingAfter(t *testing.T) {
	w := entryWorld1084(t, DomainGround, 1)
	for stage := 0; stage < 3; stage++ {
		form, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var restored World
		if err := restored.UnmarshalBinary(form); err != nil || restored.Hash() != w.Hash() {
			t.Fatalf("stage %d native decode: %v", stage, err)
		}
		var cmds []Command
		if stage == 0 {
			cmds = []Command{{Kind: KindMoveTo, Entity: 7, X: 3, Y: 3}}
		}
		a, b := StepReported(w, cmds), StepReported(&restored, cmds)
		if !reflect.DeepEqual(a, b) || w.Hash() != restored.Hash() {
			t.Fatalf("stage %d diverged across restore", stage)
		}
	}
}
