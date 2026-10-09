package sim

import (
	"fmt"
	"testing"
)

func escortReplacementWorld(t *testing.T, sub int32, order uint8, saved bool) *World {
	t.Helper()
	x := int32(22)
	if order == escortOrderClose {
		x = 12
	}
	w := esWorld(t, sub, 3, engRel(t),
		laFighter(1, 2, 7, 20, 20), laFighter(2, 2, 7, x, 20), laFighter(3, 3, 9, 40, 40))
	if sub == subCommandFollow {
		w.armFollow(1)
	} else {
		w.armDefend(1)
	}
	if w.entities[1].EscortOrder != order {
		t.Fatalf("initial escort order %d, want %d", w.entities[1].EscortOrder, order)
	}
	if saved {
		g := SavedGroup{ID: 1, Selector: 7, Owner: SavedGroupReference{Class: 1, Owner: 2}, Authored: true,
			Members: []SavedGroupMember{{Entity: 1, Bound: true}, {Entity: 2, Bound: true}}}
		g.AI[0x45] = 1
		w.savedGroups = &savedGroupState{HighWater: 1, Groups: []SavedGroup{g}, Orders: []SavedActorOrder{
			{Entity: 1, State: uint32(w.entities[0].ActorState), Authored: true},
			{Entity: 2, State: uint32(w.entities[1].ActorState), Authored: true},
		}}
		w.savedGroups.Orders[1].Raw[0x70] = 3
	}
	w.flipOnBlow(2, 1)
	if !w.entities[1].EscortTurnPending {
		t.Fatal("blow did not arm escort turn")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(b); err != nil {
		t.Fatalf("initial escort fixture: %v", err)
	}
	return w
}

func assertEscortReplacementSave(t *testing.T, w *World) {
	t.Helper()
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(b); err != nil {
		t.Fatal(err)
	}
	if cold.Hash() != w.Hash() {
		t.Fatal("order replacement changed on cold load")
	}
	for range 32 {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("order replacement diverged after cold load")
		}
		if _, err := w.MarshalBinary(); err != nil {
			t.Fatal(err)
		}
	}
}

func assertEscortEnded(t *testing.T, w *World) {
	t.Helper()
	e := w.entities[1]
	if escortState(e.ActorState) || e.HasEscortTarget || e.EscortTarget != 0 || e.EscortRange != 0 ||
		e.EscortOrder != escortOrderNone || e.EscortTurnPending {
		t.Fatalf("replacement retained escort: state=%d target=%d/%v range=%d order=%d pending=%v",
			e.ActorState, e.EscortTarget, e.HasEscortTarget, e.EscortRange, e.EscortOrder, e.EscortTurnPending)
	}
	if o := w.savedOrder(e.ID); o != nil && (o.Raw[0x54] != 0 || o.Raw[0x70] != 0 || o.EscortBound || escortState(uint8(o.State))) {
		t.Fatalf("saved order retained escort: state=%d alarm=%d range=%d bound=%v", o.State, o.Raw[0x54], o.Raw[0x70], o.EscortBound)
	}
}

func TestScriptPatrolAfterEscortBlowCanSaveAndColdLoad(t *testing.T) {
	w := escortReplacementWorld(t, subCommandFollow, escortOrderIdle, false)
	w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
		Args: [scriptParams]int32{subCommandPatrol, 40, 20}})
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatalf("Patrol after blow refused save: %v", err)
	}
	assertEscortEnded(t, w)
	assertEscortReplacementSave(t, w)
}

func TestScriptMoveAndSwarmReplaceEscortBeforeActorPass(t *testing.T) {
	for _, saved := range []bool{false, true} {
		for _, sub := range []int32{subCommandDefend, subCommandFollow} {
			for _, order := range []uint8{escortOrderIdle, escortOrderClose} {
				for _, command := range []int32{2, 4, 5} {
					t.Run(fmt.Sprintf("saved=%v/state=%d/order=%d/command=%d", saved, sub, order, command), func(t *testing.T) {
						w := escortReplacementWorld(t, sub, order, saved)
						w.entities[2].OffMap = true
						w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
							Args: [scriptParams]int32{command, 40, 20}})
						if command == 2 {
							if saved {
								w.savedDecision(&w.savedGroups.Groups[0], orderSwarm, nil)
							} else {
								w.armSwarm(aiGroup{owner: 2, group: 7, members: []int{1}}, nil)
							}
						}
						assertEscortEnded(t, w)
						before := w.entities[1]
						w.tick = 1
						Step(w, nil)
						if e := w.entities[1]; !e.HasTarget || e.TargetX != before.TargetX || e.TargetY != before.TargetY ||
							e.X == before.X && e.Y == before.Y && e.TurnRemaining == 0 {
							t.Fatal("escort overrode the scripted destination before the next actor pass")
						}
						assertEscortReplacementSave(t, w)
					})
				}
			}
		}
	}
}

func TestEveryEscortReplacementWriterCanSaveAndColdLoad(t *testing.T) {
	commands := []int32{1, 2, 3, 4, 5, 10, 11, 14, 15, 17}
	for sub := int32(0); sub <= 20; sub++ {
		found := false
		for _, command := range commands {
			found = found || sub == command
		}
		if groupOrderSupported(sub) != found {
			t.Fatalf("script order %d changed the writer census", sub)
		}
	}
	for _, saved := range []bool{false, true} {
		for _, sub := range []int32{subCommandDefend, subCommandFollow} {
			for _, order := range []uint8{escortOrderIdle, escortOrderClose} {
				for _, command := range commands {
					t.Run(fmt.Sprintf("saved=%v/state=%d/order=%d/script=%d", saved, sub, order, command), func(t *testing.T) {
						w := escortReplacementWorld(t, sub, order, saved)
						w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
							Unit: 1, HasUnit: true, Args: [scriptParams]int32{command, 40, 20}})
						if command == 3 || command == 4 || command == 5 || command == 10 || command == 14 {
							assertEscortEnded(t, w)
						} else if command == 11 || command == 15 {
							if e := w.entities[1]; e.EscortOrder != escortOrderNone || e.EscortTurnPending {
								t.Fatal("new escort inherited the replaced order's continuation")
							}
							if o := w.savedOrder(2); o != nil && o.Raw[0x54] != 0 {
								t.Fatal("new saved escort inherited the replaced order's alarm")
							}
						}
						assertEscortReplacementSave(t, w)
					})
				}
				for _, command := range []Command{
					MoveTo(2, CellPoint{40, 20}), Attack(2, 3), AttackStructure(2, 0), PickUp(2, CellPoint{40, 20}),
					Cast(2, 1, 6), CastAt(2, 26, CellPoint{25, 20}),
					UseScroll(2, 0, 1), UseScrollAt(2, 0, CellPoint{25, 20}), UseStructure(2, 0),
					GroupMoveTo(2, CellPoint{40, 20}, 1), GroupSwarmTo(2, CellPoint{40, 20}, 1),
					GroupStance(2, int32(orderGuard), 1), GroupStance(2, int32(orderStandGround), 1),
					GroupPatrolTo(2, CellPoint{40, 20}, 1), GroupDefend(2, 2, 1), GroupRetreat(2, 2, 1), Kill(2),
				} {
					t.Run(fmt.Sprintf("saved=%v/state=%d/order=%d/player=%d/x=%d", saved, sub, order, command.Kind, command.X), func(t *testing.T) {
						w := escortReplacementWorld(t, sub, order, saved)
						w.spells = []SpellRule{
							{ID: 6, ManaCost: 5, MaxRange: 30, TargetsUnit: true, Restorative: true, DamageMin: 1, DamageMax: 2},
							{ID: 26, ManaCost: 7, MaxRange: 30},
						}
						if command.Kind == KindCast || command.Kind == KindCastAt {
							w.entities[0].HP--
							w.entities[1].Mana, w.entities[1].MaxMana, w.entities[1].KnownSpells = 1000, 1000, 1<<6|1<<26
							w.entities[1].ScanRange = 30
						}
						spell := uint32(6)
						if command.Kind == KindUseScrollAt {
							spell = 26
						}
						w.carried[1] = []ItemStack{StackItem(ItemInstance{Code: 0xe10, Kind: 4,
							Effects: []ItemEffect{{Kind: 41, Operand: spell | 60<<16}}}, 1)}
						w.sacks = []Sack{{X: 40, Y: 20, Gold: 1}}
						w.structures = []Structure{{ID: 0, Kind: 28, Width: 1, Height: 1, Col: 40, Row: 30,
							Field42: 100, MaxHealth: 100, Attach: 1}}
						w.tick = 1
						Step(w, []Command{command})
						if (command.Kind == KindCast || command.Kind == KindCastAt) && !w.actorCastBusy(1) &&
							w.entities[1].PendingOrder.Kind != PendingActorCast && w.entities[1].PendingOrder.Kind != PendingCellCast {
							t.Fatal("manual cast witness was not admitted")
						}
						if (command.Kind == KindUseScroll || command.Kind == KindUseScrollAt) && len(w.scrollCasts) == 0 {
							t.Fatal("scroll witness was not admitted")
						}
						if command.Kind == KindUseStructure && len(w.structureUses) == 0 {
							t.Fatal("structure use witness was not admitted")
						}
						assertEscortEnded(t, w)
						assertEscortReplacementSave(t, w)
					})
				}
				t.Run(fmt.Sprintf("saved=%v/state=%d/order=%d/pickup-completion", saved, sub, order), func(t *testing.T) {
					w := escortReplacementWorld(t, sub, order, saved)
					if !w.CompleteSackPickup(2) {
						t.Fatal("pickup completion was not admitted")
					}
					assertEscortEnded(t, w)
					assertEscortReplacementSave(t, w)
				})
			}
		}
	}
}

func TestEscortReplacementKeepsLoadedBlow(t *testing.T) {
	for _, saved := range []bool{false, true} {
		for _, command := range []int32{4, 5, subCommandPatrol} {
			t.Run(fmt.Sprintf("saved=%v/command=%d", saved, command), func(t *testing.T) {
				w := escortReplacementWorld(t, subCommandFollow, escortOrderIdle, saved)
				w.entities[2].X, w.entities[2].Y = 23, 20
				w.entities[1].Reach, w.entities[1].AttackCharge = 2, 5
				w.tick = 1
				w.orderAttack(1, 3)
				for range 16 {
					Step(w, nil)
					if w.entities[1].AttackPhase == AttackCharging && w.entities[1].AttackCountdown > 0 {
						break
					}
				}
				before := w.entities[1]
				if before.AttackPhase != AttackCharging || before.AttackCountdown == 0 {
					t.Fatal("witness never loaded a blow")
				}
				w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
					Args: [scriptParams]int32{command, 40, 20}})
				after := w.entities[1]
				if !after.HasAttackTarget || after.AttackTarget != before.AttackTarget ||
					after.AttackPhase != before.AttackPhase || after.AttackCountdown != before.AttackCountdown {
					t.Fatal("escort replacement dropped the loaded blow")
				}
				assertEscortEnded(t, w)
				hp := w.entities[2].HP
				assertEscortReplacementSave(t, w)
				if w.entities[2].HP >= hp {
					t.Fatal("retained blow never landed")
				}
			})
		}
	}
}

func TestInternalEscortOrdersKeepBlowForIdleTurn(t *testing.T) {
	for _, sub := range []int32{subCommandDefend, subCommandFollow} {
		w := escortReplacementWorld(t, sub, escortOrderClose, false)
		w.escortClose(1, 0)
		w.escortStepAway(1, 0, 3)
		if !w.entities[1].EscortTurnPending || !w.entities[1].HasEscortTarget {
			t.Fatal("internal escort destination consumed the blow flag or standing order")
		}
	}
}

func TestScriptDestinationsKeepNonEscortPatrolRing(t *testing.T) {
	for _, saved := range []bool{false, true} {
		for _, command := range []int32{2, 4, 5} {
			t.Run(fmt.Sprintf("saved=%v/command=%d", saved, command), func(t *testing.T) {
				w := escortReplacementWorld(t, subCommandFollow, escortOrderIdle, saved)
				w.entities[2].OffMap = true
				w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
					Args: [scriptParams]int32{subCommandPatrol, 40, 20}})
				before := w.entities[1]
				w.cmdGroupOrder(ScriptInstant{Op: ScriptInstantGroupOrder, Group: 7, HasGroup: true,
					Args: [scriptParams]int32{command, 40, 20}})
				if command == 2 {
					if saved {
						w.savedDecision(&w.savedGroups.Groups[0], orderSwarm, nil)
					} else {
						w.armSwarm(aiGroup{owner: 2, group: 7, members: []int{1}}, nil)
					}
				}
				e := w.entities[1]
				if e.ActorState != before.ActorState || e.PatrolHeadX != before.PatrolHeadX || e.PatrolHeadY != before.PatrolHeadY ||
					e.PatrolTailX != before.PatrolTailX || e.PatrolTailY != before.PatrolTailY || e.PatrolLeg != before.PatrolLeg {
					t.Fatal("escort cleanup changed the non-escort Patrol ring")
				}
				assertEscortReplacementSave(t, w)
			})
		}
	}
}

func TestSwarmKeepsNonEscortRouteAndStall(t *testing.T) {
	w := engWorld(t, engRel(t), laFighter(1, 2, 7, 20, 20))
	w.cmdGroupSwarm(7, 40, 20)
	w.entities[0].TargetX, w.entities[0].TargetY, w.entities[0].HasTarget = 40, 20, true
	w.entities[0].Stall = 3
	w.routes[0] = []cell{{21, 20}, {40, 20}}
	w.armSwarm(aiGroup{owner: 2, group: 7, members: []int{0}}, nil)
	if w.entities[0].Stall != 3 || len(w.routes[0]) != 2 {
		t.Fatal("escort cleanup reset the non-escort Swarm route or stall")
	}
}

func TestDefenderHealExtraAdmissionGatesFallThroughToCover(t *testing.T) {
	for _, gate := range []string{"restorative", "range"} {
		t.Run(gate, func(t *testing.T) {
			w := residueDefender(t, 49, 20, 0)
			if gate == "restorative" {
				w.spells[0].Restorative, w.spells[0].Damaging = false, true
				if r := w.bookSpellRefusal(1, 1, 6, false, false); r != "" {
					t.Fatalf("restorative control also refused by %q", r)
				}
			} else {
				w.spells[0].MaxRange = 0
				if r := w.bookSpellRefusal(1, 1, 6, false, false); r != refusalTargetOutOfRange {
					t.Fatalf("range control refused by %q", r)
				}
			}
			w.armDefend(1)
			if len(w.bookCasts) != 0 || !w.entities[1].HasAttackTarget || w.entities[1].AttackTarget != 3 {
				t.Fatal("extra heal refusal did not fall through to cover")
			}
		})
	}
}
