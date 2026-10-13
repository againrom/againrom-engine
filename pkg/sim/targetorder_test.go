package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"strconv"
	"testing"
)

func targetOrderWorld(t *testing.T) *World {
	t.Helper()
	actors := []Entity{
		{ID: 1, Owner: SelfSlot, X: 2, Y: 2, HP: 100, MaxHP: 100, Reach: 3, DyingTime: 200},
		{ID: 2, Owner: 2, X: 3, Y: 2, HP: 100, MaxHP: 100, DyingTime: 200},
		{ID: 3, Owner: 2, X: 4, Y: 2, HP: 100, MaxHP: 100, DyingTime: 200},
	}
	for i := range actors {
		actors[i].SourceBinding = SourceBinding{Class: 1, ArchiveIndex: uint16(i + 1), Identity: uint32(i + 1)}
		actors[i].ActorLoad = ActorLoad{Present: true, Source: SourceActor{Class: 1}}
	}
	w := mustWorld(t, 7, Bounds{Width: 8, Height: 8}, actors)
	w.entities[0].ActorState = actorStateEngage
	o := SavedActorOrder{Entity: 1, State: uint32(actorStateEngage), Authored: true}
	o.Raw[8], o.Raw[0x68] = 5, 0x7d
	if err := w.ImportSavedGroups(nil, []SavedActorOrder{o}); err != nil {
		t.Fatal(err)
	}
	return w
}

func requireTargetStand(t *testing.T, w *World) {
	t.Helper()
	e := w.entities[0]
	o := w.savedOrder(e.ID)
	if e.HasAttackTarget || e.AttackPhase != AttackReady || e.AttackCountdown != 0 || e.HasTarget || len(w.routes[0]) != 0 {
		t.Fatalf("stand retained attack or route: %+v", e)
	}
	if e.ActorState != actorStateAcquire || e.PostX != e.X || e.PostY != e.Y || o.State != uint32(actorStateAcquire) || o.Raw[8] != 0 || o.Raw[0x14] != e.Reach || binary.LittleEndian.Uint16(o.Raw[:]) != uint16(e.X)|uint16(e.Y)<<8 || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 1 {
		t.Fatalf("stand did not publish its current cell and idle order: entity=%+v order=%+v", e, o)
	}
	if o.Raw[0x68] != 0x7d {
		t.Fatal("standing down changed an unrelated order operand")
	}
}

func TestTargetOrderDeathBoundaryPreservesFinishing(t *testing.T) {
	for _, hp := range []int32{0, -1, -9, -10, -25} {
		t.Run(strconv.Itoa(int(hp)), func(t *testing.T) {
			w := targetOrderWorld(t)
			e, o := &w.entities[0], w.savedOrder(1)
			e.AttackTarget, e.HasAttackTarget = 2, true
			e.AttackPhase, e.AttackCountdown = AttackCharging, 3
			o.Raw[9] = 1
			binary.LittleEndian.PutUint32(o.Raw[0xc:], 2)
			before := *o
			w.entities[1].setCurrentHealth(hp)
			w.clearInvalidTargetReferences(2)
			if hp > -10 {
				if !e.HasAttackTarget || e.AttackTarget != 2 || e.AttackPhase != AttackCharging || e.AttackCountdown != 3 || !reflect.DeepEqual(*o, before) {
					t.Fatal("finishing target ended before the terminal boundary")
				}
				return
			}
			requireTargetStand(t, w)
			if o.Raw[9] != before.Raw[9] {
				t.Fatal("terminal stand changed the retained native progress byte")
			}
		})
	}
}

func TestMissingTargetRowsStandOnlyLivingActors(t *testing.T) {
	for _, row := range []byte{5, 6, 8} {
		for _, key := range []uint32{0, 0xdeadbeef} {
			for _, alive := range []bool{true, false} {
				name := strconv.Itoa(int(row)) + "/" + strconv.FormatUint(uint64(key), 16) + "/" + strconv.FormatBool(alive)
				t.Run(name, func(t *testing.T) {
					w := targetOrderWorld(t)
					o := w.savedOrder(1)
					o.Raw[8], o.Raw[9] = row, 1
					at := 0xc
					if row == 8 {
						at = 0x28
						o.State, o.Raw[9] = 0xd, 2
					}
					binary.LittleEndian.PutUint32(o.Raw[at:], key)
					if !alive {
						w.entities[0].setCurrentHealth(-20)
						w.entities[0].Decay, w.entities[0].Dwell = DecayBones, 0
						w.entities[0].HeldOrder = HeldOrder{Kind: HeldOrderFrozen, Phase: 5}
					}
					before, actor := *o, w.entities[0]
					w.normaliseSavedTargetOrders()
					if !alive {
						if !reflect.DeepEqual(*o, before) || !reflect.DeepEqual(w.entities[0], actor) {
							t.Fatal("normalization changed a frozen dead actor")
						}
						return
					}
					requireTargetStand(t, w)
					if o.Raw[9] != before.Raw[9] {
						t.Fatal("missing-target stand changed the retained native progress byte")
					}
				})
			}
		}
	}
}

func TestMissingAttackCarrierKeepsAnIndependentDestination(t *testing.T) {
	for _, row := range []byte{5, 6} {
		t.Run(strconv.Itoa(int(row)), func(t *testing.T) {
			w := targetOrderWorld(t)
			e, o := &w.entities[0], w.savedOrder(1)
			e.HasTarget, e.TargetX, e.TargetY = true, 6, 3
			e.PostX, e.PostY = 1, 1
			o.Raw[8], o.Raw[9] = row, 1
			before := *e
			w.normaliseSavedTargetOrders()
			if !reflect.DeepEqual(*e, before) || o.Raw[8] != 1 || binary.LittleEndian.Uint16(o.Raw[10:]) != 0x0306 || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 0 || o.Raw[9] != 1 {
				t.Fatal("missing attack carrier discarded the current destination or cycle")
			}
		})
	}
}

func TestCompletedPickupHasNoTargetlessAttackCarrierBeforeNextStep(t *testing.T) {
	w := targetOrderWorld(t)
	if !w.CompleteSackPickup(1) {
		t.Fatal("pickup completion refused the living actor")
	}
	e, o := w.entities[0], w.savedOrder(1)
	if e.ActorState != actorStatePickupComplete || o.State != uint32(actorStatePickupComplete) || o.Raw[8] != 0 || binary.LittleEndian.Uint32(o.Raw[0x50:]) != 1 {
		t.Fatal("pickup completion retained a targetless attack before the next Step")
	}
	if m := w.motionFor(1); m != nil && m.ActorAction != 0 {
		t.Fatal("completed pickup retained an attack action")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

func TestTargetAttackRowsKeepResolvedCorpseCarriers(t *testing.T) {
	for _, row := range []byte{5, 6} {
		for _, retained := range []bool{false, true} {
			t.Run(strconv.Itoa(int(row))+"/"+strconv.FormatBool(retained), func(t *testing.T) {
				w := targetOrderWorld(t)
				w.entities[0].ActorState = actorStateAcquire
				key := uint32(2)
				if retained {
					key = 7
					dead := OriginalDeadActor{ID: 7, Source: OriginalDeadSource{Identity: key, ArchiveIndex: 4, Class: 1,
						State: DeadActorState{Cell: 0x0303, FineX: 128, FineY: 128, Stage: 5, HP: -10001}}}
					if err := w.ImportOriginalDeadActors([]OriginalDeadActor{dead}); err != nil {
						t.Fatal(err)
					}
				} else {
					w.entities[1].setCurrentHealth(-11)
					w.entities[1].Decay, w.entities[1].Dwell = DecayFallen, 0
				}
				o := w.savedOrder(1)
				o.State = uint32(actorStateAcquire)
				o.Raw[8], o.Raw[9] = row, 1
				o.Authored = false
				binary.LittleEndian.PutUint32(o.Raw[0xc:], key)
				before, actor := *o, w.entities[0]
				if err := w.ImportOriginalActorActions([]OriginalActorAction{{Entity: 1, ActorState: actor.ActorState, PostX: actor.PostX, PostY: actor.PostY}}); err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(*o, before) || !reflect.DeepEqual(w.entities[0], actor) {
					t.Fatal("resolved corpse carrier was normalized as a missing target")
				}
			})
		}
	}
}

func TestTargetCastRowsRetainResolvedSelfAndCorpses(t *testing.T) {
	for _, target := range []string{"self", "living", "corpse", "retained corpse", "Control Spirit"} {
		t.Run(target, func(t *testing.T) {
			w := targetOrderWorld(t)
			o := w.savedOrder(1)
			o.Raw[8] = 8
			key := uint32(2)
			switch target {
			case "self":
				key = 1
			case "corpse", "Control Spirit":
				w.entities[1].setCurrentHealth(-20)
				w.entities[1].Decay, w.entities[1].Dwell = DecayBones, 0
				if target == "Control Spirit" {
					w.bookCasts = []bookCast{{Caster: 1, Target: 2, Spell: controlSpiritSpellID, Phase: bookCharging, Remaining: 3, Retained: true}}
				}
			case "retained corpse":
				key = 7
				dead := OriginalDeadActor{ID: 7, Source: OriginalDeadSource{Identity: key, ArchiveIndex: 4, Class: 1,
					State: DeadActorState{Cell: 0x0303, FineX: 128, FineY: 128, Stage: 5, HP: -10001}}}
				if err := w.ImportOriginalDeadActors([]OriginalDeadActor{dead}); err != nil {
					t.Fatal(err)
				}
			}
			binary.LittleEndian.PutUint32(o.Raw[0x28:], key)
			before := *o
			w.clearInvalidTargetReferences(2)
			w.normaliseSavedTargetOrders()
			if !reflect.DeepEqual(*o, before) || w.entities[0].ActorState != actorStateEngage {
				t.Fatal("resolved cast endpoint was replaced by stand")
			}
			if target == "Control Spirit" && (len(w.bookCasts) != 1 || w.bookCasts[0].Target != 2) {
				t.Fatal("terminal boundary cancelled Control Spirit's corpse cast")
			}
		})
	}
}

func TestRestoredMissingTargetOrderCannotResumeItsExecutorRow(t *testing.T) {
	for _, row := range []byte{5, 6, 8} {
		t.Run(strconv.Itoa(int(row)), func(t *testing.T) {
			w := targetOrderWorld(t)
			w.savedOrder(1).Raw[8] = row
			actions := w.Actions()
			cold := targetOrderWorld(t)
			cold.savedOrder(1).Raw[8] = 0
			if err := cold.RestoreActions(actions, nil); err != nil {
				t.Fatal(err)
			}
			requireTargetStand(t, cold)
		})
	}
}

func TestTargetOrderKeepsTypedNativeVictimBeforeIdentityMinting(t *testing.T) {
	w := targetOrderWorld(t)
	w.entities[1].SourceBinding, w.entities[1].ActorLoad = SourceBinding{}, ActorLoad{}
	e, o := &w.entities[0], w.savedOrder(1)
	e.AttackTarget, e.HasAttackTarget = 2, true
	before := *o
	w.normaliseSavedTargetOrders()
	if !reflect.DeepEqual(*o, before) || !e.HasAttackTarget || e.AttackTarget != 2 {
		t.Fatal("a present native victim with no archive key was treated as missing")
	}
}

func TestTargetDeathPreservesAnIndependentPendingVictim(t *testing.T) {
	w := targetOrderWorld(t)
	e, o := &w.entities[0], w.savedOrder(1)
	e.AttackTarget, e.HasAttackTarget, e.AttackPhase, e.AttackCountdown = 2, true, AttackCharging, 3
	e.PendingAttackTarget, e.HasPendingAttackTarget = 3, true
	binary.LittleEndian.PutUint32(o.Raw[0xc:], 3)
	w.entities[1].setCurrentHealth(-10)
	w.clearInvalidTargetReferences(2)
	w.normaliseSavedTargetOrders()
	w.takePendingAttacks()
	if !e.HasAttackTarget || e.AttackTarget != 3 || e.HasPendingAttackTarget || e.ActorState != actorStateEngage || o.State != uint32(actorStateEngage) || o.Raw[8] != 5 {
		t.Fatalf("death of the active victim discarded the independent replacement: entity=%+v order=%+v", e, o)
	}
}

func TestMissingTargetStandReleasesHeldBodyWords(t *testing.T) {
	w := targetOrderWorld(t)
	w.entities[0].HeldOrder = HeldOrder{Kind: HeldOrderBody, Target: 2, Phase: 5, Countdown: 9, Complete: 1}
	w.entities[1].setCurrentHealth(-20)
	w.entities[1].Decay, w.entities[1].Dwell = DecayBones, 0
	w.normaliseSavedTargetOrders()
	if w.entities[0].HeldOrder != (HeldOrder{}) {
		t.Fatal("stand left a live actor's deferred body action to be written by SAVE")
	}
	requireTargetStand(t, w)
}

func TestMissingCastTargetCannotRecreateItsTargetRow(t *testing.T) {
	for _, row := range []byte{0, 5, 8} {
		for _, pending := range []bool{false, true} {
			t.Run(strconv.Itoa(int(row))+"/"+strconv.FormatBool(pending), func(t *testing.T) {
				w := targetOrderWorld(t)
				w.savedOrder(1).Raw[8] = row
				if pending {
					w.entities[0].PendingOrder = PendingOrder{Kind: PendingActorCast, Target: 99, Spell: controlSpiritSpellID, RowAdmitted: true}
				} else {
					w.entities[0].CastWait = 3
					w.bookCasts = []bookCast{{Caster: 1, Target: 99, Spell: controlSpiritSpellID, Phase: bookCharging, Remaining: 3, Retained: true}}
				}
				w.normaliseSavedTargetOrders()
				w.syncCurrentActionCarriers()
				requireTargetStand(t, w)
				if len(w.bookCasts) != 0 || w.entities[0].CastWait != 0 || w.entities[0].PendingOrder.Kind == PendingActorCast {
					t.Fatal("missing-target cast retained a producer that can write row 8 with a null endpoint")
				}
			})
		}
	}
}

func TestMissingCastTargetPreservesTheIndependentValidProducer(t *testing.T) {
	for _, row := range []byte{0, 5, 8} {
		for _, missingPending := range []bool{false, true} {
			t.Run(strconv.Itoa(int(row))+"/"+strconv.FormatBool(missingPending), func(t *testing.T) {
				w := targetOrderWorld(t)
				w.savedOrder(1).Raw[8] = row
				w.entities[0].CastWait, w.entities[0].Mana = 3, 77
				book := bookCast{Caster: 1, Target: 2, Spell: controlSpiritSpellID, Phase: bookCharging, Remaining: 3, Retained: true, Paid: true}
				pending := PendingOrder{Kind: PendingActorCast, Target: 2, Spell: controlSpiritSpellID}
				if missingPending {
					pending.Target = 99
				} else {
					book.Target = 99
				}
				w.bookCasts, w.entities[0].PendingOrder = []bookCast{book}, pending
				w.normaliseSavedTargetOrders()
				w.syncCurrentActionCarriers()
				if w.entities[0].CastWait != 3 || w.entities[0].Mana != 77 {
					t.Fatal("removing the missing endpoint changed the valid producer's recovery or payment")
				}
				if missingPending {
					if len(w.bookCasts) != 1 || w.bookCasts[0] != book || w.entities[0].PendingOrder != (PendingOrder{}) {
						t.Fatal("removing the missing pending endpoint changed the paid active book cast")
					}
				} else if len(w.bookCasts) != 0 || w.entities[0].PendingOrder != pending {
					t.Fatal("removing the missing active endpoint discarded the independent pending cast")
				}
			})
		}
	}
}

func TestMissingScrollTargetRefundsReservationAcrossStartedStates(t *testing.T) {
	for _, row := range []byte{0, 5, 8} {
		for _, started := range []bool{false, true} {
			t.Run(strconv.Itoa(int(row))+"/"+strconv.FormatBool(started), func(t *testing.T) {
				w := scrollFixtureWorld(t, 2, 3)
				bindOperationsPack(t, w, 0)
				child := w.savedObjects.Effects[0]
				if !w.beginScroll(0, 0, 2, 0, 0, false) {
					t.Fatal("scroll fixture did not reserve a real pack item")
				}
				c := &w.scrollCasts[0]
				c.Target, c.Started = 99, started
				if started {
					c.Remaining = 3
				}
				reserved, handle := c.Item.ObjectID, c.Reservation
				if reserved == 0 || handle == 0 || !w.savedObjects.HasRoot(reserved, SavedObjectOwner{Kind: SavedOwnerSession, SessionHandle: handle}) {
					t.Fatal("scroll fixture lacks its detached session reservation")
				}
				e := &w.entities[0]
				e.ActorState = actorStateEngage
				o := w.ensureSavedOrder(0)
				o.State, o.Raw[8], o.Raw[0x68] = uint32(actorStateEngage), row, 0x7d
				w.normaliseSavedTargetOrders()
				w.syncCurrentActionCarriers()
				if len(w.ScrollCasts()) != 0 || !w.savedObjects.item(reserved).Retired || len(w.savedObjects.Locations(reserved)) != 0 {
					t.Fatal("missing-target cleanup retained the detached scroll or its session root")
				}
				stock, _ := w.CarriedStacks(1)
				if len(stock) != 1 || stock[0].ObjectID != 10 || stock[0].Count != 3 || w.entities[1].HP != 100 || w.savedObjects.Effects[0] != child {
					t.Fatalf("scroll disposition changed pack ownership or applied a release: started=%t stock=%+v target HP=%d", started, stock, w.entities[1].HP)
				}
				requireTargetStand(t, w)
				form, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var cold World
				if err := cold.UnmarshalBinary(form); err != nil || cold.Hash() != w.Hash() {
					t.Fatal("refunded ownership changed across cold native LOAD", err)
				}
			})
		}
	}
}

func TestMissingScrollTargetFailedRefundPreservesStateAndRefusesSave(t *testing.T) {
	for _, row := range []byte{0, 5, 8} {
		for _, started := range []bool{false, true} {
			t.Run(strconv.Itoa(int(row))+"/"+strconv.FormatBool(started), func(t *testing.T) {
				w := scrollFixtureWorld(t, 2, 1)
				bindOperationsPack(t, w, 0)
				if !w.beginScroll(0, 0, 2, 0, 0, false) {
					t.Fatal("scroll fixture did not reserve its pack item")
				}
				e := &w.entities[0]
				e.ActorLoad, e.Capacity = ActorLoad{Present: true}, 100
				pack := w.savedObjects.container(SavedObjectOwner{Kind: SavedOwnerActorPack, Entity: 1})
				pack.Present, pack.InsertIndex, pack.Accumulator = false, 0, 0
				if _, err := w.MarshalBinary(); err != nil {
					t.Fatal("invalid no-container control fixture", err)
				}
				c := &w.scrollCasts[0]
				c.Target, c.Started = 99, started
				if started {
					c.Remaining = 3
				}
				e.ActorState = actorStateEngage
				o := w.ensureSavedOrder(0)
				o.State, o.Raw[8] = uint32(actorStateEngage), row
				hash, actor, order := w.Hash(), *e, *o
				casts, objects := w.ScrollCasts(), w.savedObjects.Clone()
				if w.cancelScroll(0) || w.Hash() != hash {
					t.Fatal("no-container fixture did not refuse its refund atomically")
				}
				w.normaliseSavedTargetOrders()
				if w.Hash() != hash || !reflect.DeepEqual(w.entities[0], actor) || !reflect.DeepEqual(*w.savedOrder(1), order) || !reflect.DeepEqual(w.ScrollCasts(), casts) || !reflect.DeepEqual(w.savedObjects, objects) {
					t.Fatal("failed missing-target refund changed actor, order or reservation")
				}
				storage := bytes.Repeat([]byte{0x5a}, 4096)
				before := bytes.Clone(storage)
				written, err := w.MarshalBinaryInto(storage[:3])
				if written != nil || err == nil || err.Error() != "sim: actor 1 retains an unresolved cast target" || len(err.Error()) > 128 {
					t.Fatalf("unrefunded missing-target cast did not return its bounded SAVE refusal: %v", err)
				}
				if !bytes.Equal(storage, before) || w.Hash() != hash {
					t.Fatal("refused SAVE changed caller storage or world state")
				}
			})
		}
	}
}

func TestOriginalActorActionsNormalizeMissingTargetWithoutSupplement(t *testing.T) {
	for _, key := range []uint32{0, 0xdeadbeef, 2} {
		t.Run(strconv.FormatUint(uint64(key), 16), func(t *testing.T) {
			w := targetOrderWorld(t)
			w.entities[0].ActorState = actorStateGuard
			o := w.savedOrder(1)
			o.Authored, o.Raw[9] = false, 1
			binary.LittleEndian.PutUint32(o.Raw[0xc:], key)
			before := *o
			action := OriginalActorAction{Entity: 1, ActorState: actorStateEngage, PostX: 5, PostY: 4}
			if key == 2 {
				action.HasTarget, action.Target = true, 2
			}
			if err := w.ImportOriginalActorActions([]OriginalActorAction{action}); err != nil {
				t.Fatal(err)
			}
			if key == 2 {
				e := w.entities[0]
				if !e.HasAttackTarget || e.AttackTarget != 2 || e.ActorState != actorStateEngage || e.PostX != 5 || e.PostY != 4 || !reflect.DeepEqual(*w.savedOrder(1), before) {
					t.Fatal("ordinary action import changed an intact source engagement")
				}
				return
			}
			requireTargetStand(t, w)
			if w.savedOrder(1).Raw[9] != before.Raw[9] {
				t.Fatal("ordinary source normalization changed the retained progress byte")
			}
		})
	}
}

func TestMissingCastTargetKeepsAnIndependentPhysicalRequest(t *testing.T) {
	for _, pending := range []bool{false, true} {
		w := targetOrderWorld(t)
		w.savedOrder(1).Raw[8] = 8
		e := &w.entities[0]
		if pending {
			e.PendingAttackTarget, e.HasPendingAttackTarget = 2, true
		} else {
			e.AttackTarget, e.HasAttackTarget = 2, true
		}
		w.bookCasts = []bookCast{{Caster: 1, Target: 99, Spell: controlSpiritSpellID, Phase: bookCharging, Remaining: 3, Retained: true}}
		w.normaliseSavedTargetOrders()
		id, _, present := w.entities[0].RequestedAttackTarget()
		if !present || id != 2 || len(w.bookCasts) != 0 || w.savedOrder(1).Raw[8] != 5 {
			t.Fatal("missing cast lost the independent physical request", pending)
		}
	}
}
