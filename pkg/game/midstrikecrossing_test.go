package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func midStrikeCrossingFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := midStrikeCastFront(t)
	actors := f.live.world.Entities()
	actors[0].X, actors[0].Y = 15, 16
	actors[0].AttackPhase, actors[0].AttackCountdown = sim.AttackReady, 0
	actors[0].Facing, actors[0].DesiredFacing, actors[0].RotationSpeed = 64, 64, 64
	actors[1].X, actors[1].Y = 17, 16
	actors[2].X, actors[2].Y = 18, 16
	for i := range actors {
		actors[i].SourceBinding = sim.SourceBinding{Class: sim.GeneratedUnitBinding, Identity: 7001 + uint32(i), RuntimeID: 8001 + uint32(i), TokenRow: 1, TypeID: 1}
		actors[i].Reach = max(actors[i].Reach, 1)
		source := currentActorSource(actors[i], sim.SourceActor{})
		source.Fighter, source.HasSpellbook, source.HasOwner, source.ManaReservePercent = true, true, true, 95
		actors[i].ActorLoad = sim.ActorLoad{Present: true, Source: source}
	}
	w, err := sim.NewStockedSpelledWorld(17, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, f.live.world.Relations(), nil, nil, mapload.SpellRules(f.Table))
	if err != nil {
		t.Fatal(err)
	}
	order := sim.SavedActorOrder{Entity: 1, State: 0xb}
	order.Raw[8], order.Raw[9] = 1, 3
	group := sim.SavedGroup{ID: 1, Selector: 1, Members: []sim.SavedGroupMember{{Archive: 1, Entity: 1, Bound: true}}}
	if err := w.ImportSavedGroups([]sim.SavedGroup{group}, []sim.SavedActorOrder{order}); err != nil {
		t.Fatal(err)
	}
	motion := sim.SavedActorMotion{Entity: 1, Position: sim.SavedActorPosition{Cell: 0x100f, PackedCell: 0x100f, FineX: 42, FineY: 128}, ActorAction: 1}
	motion.Mover[0], motion.Mover[1], motion.Mover[10], motion.Mover[0xb0] = 64, 64, 64, 32
	binary.LittleEndian.PutUint16(motion.Mover[0xaa:], 16)
	binary.LittleEndian.PutUint16(motion.Mover[0xac:], 10)
	binary.LittleEndian.PutUint16(motion.Mover[0xae:], 2)
	if err := w.ImportOriginalActorMotions([]sim.SavedActorMotion{motion}, nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := w.ImportOriginalActorActions([]sim.OriginalActorAction{{Entity: 1, HasTarget: true, Target: 3}}); err != nil {
		t.Fatal(err)
	}
	if !w.ActorMotionActive(1) || len(w.SavedActorMotionIssues()) != 0 || w.ActorOrderProgress(1) != 3 {
		t.Fatal("fixture crossing was not admitted", w.SavedActorMotionIssues())
	}
	f.live.world, f.live.mission.state.World = w, w
	manifest := &SnapshotActorManifest{Version: actorManifestVersion}
	for _, e := range actors {
		manifest.Actors = append(manifest.Actors, SnapshotActor{ID: e.ID})
	}
	f.live.mission.state.ActorManifest = manifest
	return f
}

func midStrikeCrossingMotion(t *testing.T, w *sim.World) sim.SavedActorMotion {
	t.Helper()
	motions, _, _, _ := w.SavedActorMotions()
	for _, m := range motions {
		if m.Entity == 1 {
			return m
		}
	}
	t.Fatal("caster lost its retained motion")
	return sim.SavedActorMotion{}
}

func midStrikeCrossingProgress(t *testing.T, w *sim.World) byte {
	t.Helper()
	_, orders, _ := w.SavedGroups()
	for _, o := range orders {
		if o.Entity == 1 {
			return o.Raw[9]
		}
	}
	t.Fatal("caster lost its current order")
	return 0
}

func TestMidStrikeImportedCrossingCurrentSAVKeepsProgressUntilCastAdmission(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command sim.Command
		pending sim.PendingOrder
	}{
		{"actor", sim.Cast(1, 2, 6), sim.PendingOrder{Kind: sim.PendingActorCast, Target: 2, Spell: 6}},
		{"cell", sim.CastAt(1, 3, sim.CellPoint{X: 19, Y: 16}), sim.PendingOrder{Kind: sim.PendingCellCast, Spell: 3, X: 19, Y: 16}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := midStrikeCrossingFront(t)
			w := f.live.world
			f.live.pending = []sim.Command{tc.command}
			f.live.tick()
			m := midStrikeCrossingMotion(t, w)
			h := pendingVictimEntity(t, w, 1)
			if !m.Current || !m.Active || m.Position.Cell != 0x100f || m.Position.PackedCell != 0x100f || m.Position.FineX != 74 || m.Position.FineY != 128 || binary.LittleEndian.Uint16(m.Mover[0xaa:]) != 16 || binary.LittleEndian.Uint16(m.Mover[0xac:]) != 11 || m.Mover[0xb0] != 32 || midStrikeCrossingProgress(t, w) != 3 || w.ActorOrderProgress(1) != 3 {
				t.Fatalf("manual setter replaced the independently current crossing: %+v", m)
			}
			if h.PendingOrder != tc.pending || h.AttackTarget != 3 || !h.HasAttackTarget || h.AttackPhase != sim.AttackReady || h.AttackCountdown != 0 || len(w.Actions().Books) != 0 || castOrderHP(w, 3) != 200 {
				t.Fatalf("logical crossing admitted a book or applied the ready physical carrier: actor=%+v books=%+v", h, w.Actions().Books)
			}
			raw, doc, a := saveCurrentEffect(t, f)
			r := castOrderRecord(t, doc, a, 1)
			position := savedRecordRawForTest(t, r, "Block12")
			mover := savedRecordRawForTest(t, r, "U154")
			order := savedRecordRawForTest(t, r, "U158")
			if binary.LittleEndian.Uint16(position) != 0x100f || binary.LittleEndian.Uint16(position[2:]) != 0x100f || position[4] != 74 || position[5] != 128 || binary.LittleEndian.Uint16(mover[0xaa:]) != 16 || binary.LittleEndian.Uint16(mover[0xac:]) != 11 || mover[0xb0] != 32 || order[8] != 0 || order[9] != 3 || binary.LittleEndian.Uint32(savedRecordRawForTest(t, r, "U54")) != 1 {
				t.Fatal("current SAV discarded crossing operands or published the cast action early")
			}
			if binary.LittleEndian.Uint32(order[0x30:]) == 0 {
				t.Fatal("pending crossing SAV has no requested spell key")
			}
			if tc.pending.Kind == sim.PendingActorCast {
				if binary.LittleEndian.Uint32(order[0x28:]) != savedRecordValueForTest(t, castOrderRecord(t, doc, a, 2), "Identity") {
					t.Fatal("pending crossing SAV lost the requested actor")
				}
			} else if order[0x3c] != 19 || order[0x3d] != 16 {
				t.Fatal("pending crossing SAV lost the requested cell")
			}
			cold := openCurrentEffectSave(t, f, raw)
			assertCurrentWorldEqual(t, w, cold.live.world, "crossing pending cold LOAD")
			retainedLoss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) {
				row.ImportedMotion, row.MotionIssue = false, ""
				if row.Order == nil {
					t.Fatal("retained crossing control has no order")
				}
				row.Order.Progress = 0
			})
			operandLoss := midStrikeSAVControl(t, f, doc, func(row *sim.ActorContinuation) {
				if tc.pending.Kind == sim.PendingActorCast {
					row.PendingOrder.Target = 1
				} else {
					row.PendingOrder.X = 18
				}
			})
			for range 2 {
				sim.Step(retainedLoss.live.world, nil)
			}
			if b, ok := castOrderBook(retainedLoss.live.world, 1); !ok || b.Phase != 1 || castOrderHP(retainedLoss.live.world, 3) != 200 {
				t.Fatal("loss of retained crossing/progress did not advance the first book admission")
			}
			for paid := 1; paid <= 5; paid++ {
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				sim.Step(operandLoss.live.world, nil)
				assertCurrentWorldEqual(t, w, cold.live.world, "retained crossing cold next step")
				m = midStrikeCrossingMotion(t, w)
				progress, fineX := byte(3), byte(74+32*paid)
				if paid == 5 {
					progress, fineX = 0, 128
				}
				if m.Position.Cell != 0x100f || m.Position.FineX != fineX || m.Position.FineY != 128 || m.Active != (paid < 5) || midStrikeCrossingProgress(t, w) != progress || w.ActorOrderProgress(1) != progress || len(w.Actions().Books) != 0 || pendingVictimEntity(t, w, 1).PendingOrder.RowAdmitted || castOrderHP(w, 3) != 200 {
					t.Fatalf("crossing payment %d lost movement/progress before its arrival writer: motion=%+v actor=%+v", paid, m, pendingVictimEntity(t, w, 1))
				}
			}
			for _, next := range []*sim.World{w, cold.live.world, operandLoss.live.world} {
				sim.Step(next, nil)
				if !pendingVictimEntity(t, next, 1).PendingOrder.RowAdmitted || midStrikeCrossingProgress(t, next) != 0 || next.ActorOrderProgress(1) != 2 || len(next.Actions().Books) != 0 || castOrderHP(next, 3) != 200 {
					t.Fatal("requested row did not admit separately at logical zero")
				}
			}
			assertCurrentWorldEqual(t, w, cold.live.world, "crossing row admission")
			raw, doc, a = saveCurrentEffect(t, f)
			r = castOrderRecord(t, doc, a, 1)
			order = savedRecordRawForTest(t, r, "U158")
			inner, action := byte(8), uint32(0xd)
			if tc.pending.Kind == sim.PendingCellCast {
				inner, action = 9, 0xe
			}
			if order[8] != inner || order[9] != 2 || binary.LittleEndian.Uint32(savedRecordRawForTest(t, r, "U54")) != action {
				t.Fatal("admitted row SAV did not publish the requested cast action")
			}
			marked := openCurrentEffectSave(t, f, raw)
			assertCurrentWorldEqual(t, w, marked.live.world, "crossing admitted row cold LOAD")
			for _, next := range []*sim.World{w, cold.live.world, marked.live.world, operandLoss.live.world} {
				sim.Step(next, nil)
				b, ok := castOrderBook(next, 1)
				if !ok || b.Phase != 1 || b.Spell != tc.pending.Spell || pendingVictimEntity(t, next, 1).PendingOrder.Kind != sim.PendingNone || castOrderHP(next, 3) != 200 {
					t.Fatal("first admitted cast did not execute after crossing completion", next.Actions().Books)
				}
			}
			assertCurrentWorldEqual(t, w, cold.live.world, "crossing first book admission")
			assertCurrentWorldEqual(t, w, marked.live.world, "admitted row first book admission")
			book, _ := castOrderBook(w, 1)
			lostBook, _ := castOrderBook(operandLoss.live.world, 1)
			if tc.pending.Kind == sim.PendingActorCast {
				if book.AtCell || book.Target != 2 || lostBook.Target != 1 {
					t.Fatal("requested actor operand did not determine the first admitted book")
				}
			} else if !book.AtCell || book.X != 19 || book.Y != 16 || !lostBook.AtCell || lostBook.X != 18 || lostBook.Y != 16 {
				t.Fatal("requested cell operands did not determine the first admitted book")
			}
			applied := false
			for range 64 {
				sim.Step(w, nil)
				sim.Step(cold.live.world, nil)
				sim.Step(marked.live.world, nil)
				sim.Step(operandLoss.live.world, nil)
				assertCurrentWorldEqual(t, w, cold.live.world, "crossing first cast application")
				assertCurrentWorldEqual(t, w, marked.live.world, "admitted row first cast application")
				if tc.pending.Kind == sim.PendingActorCast {
					if castOrderHP(w, 2) <= 40 {
						continue
					}
					if castOrderHP(operandLoss.live.world, 2) != 40 || castOrderHP(operandLoss.live.world, 1) <= castOrderHP(w, 1) || castOrderHP(w, 3) != 200 {
						t.Fatal("requested actor loss did not change the first heal application")
					}
				} else {
					effects, lostEffects := w.CellEffects(), operandLoss.live.world.CellEffects()
					if len(effects) == 0 {
						continue
					}
					if len(effects) != 1 || effects[0].Spell != 3 || effects[0].X != 19 || effects[0].Y != 16 || len(lostEffects) != 1 || lostEffects[0].Spell != 3 || lostEffects[0].X != 18 || lostEffects[0].Y != 16 {
						t.Fatal("requested cell loss did not change the first area application", effects, lostEffects)
					}
				}
				applied = true
				break
			}
			if !applied {
				t.Fatal("first admitted cast did not apply within the fixture wind-up")
			}
		})
	}
}
