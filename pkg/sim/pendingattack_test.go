package sim

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func pendingPair(t *testing.T) *World {
	t.Helper()
	w := lcPlayerWorld(t, lcVictim(3, 2, 24, 21), lcVictim(4, 2, 23, 21))
	Step(w, []Command{Attack(1, 3)})
	e := laEnt(t, w, 1)
	if e.AttackTarget != 2 || !e.HasPendingAttackTarget || e.PendingAttackTarget != 3 {
		t.Fatalf("missing active/requested pair: %+v", e)
	}
	return w
}

func TestPendingAttackLastAcceptedCommandWins(t *testing.T) {
	for _, tc := range []struct {
		name     string
		commands []Command
		pending  EntityID
		present  bool
	}{
		{"repeat requested", []Command{Attack(1, 3), Attack(1, 3)}, 3, true},
		{"return active", []Command{Attack(1, 2)}, 0, false},
		{"third target", []Command{Attack(1, 4)}, 4, true},
		{"invalid target preserves accepted", []Command{Attack(1, 99)}, 3, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := pendingPair(t)
			control := worldRoundTripForTest(t, w)
			Step(control, nil)
			Step(w, tc.commands)
			e, c := laEnt(t, w, 1), laEnt(t, control, 1)
			if e.AttackTarget != 2 || e.AttackPhase != c.AttackPhase || e.AttackCountdown != c.AttackCountdown || e.HasPendingAttackTarget != tc.present || e.PendingAttackTarget != tc.pending {
				t.Fatalf("last command changed cycle or requested target: %+v", e)
			}
			for range 180 {
				Step(w, nil)
			}
			if tc.present && laEnt(t, w, tc.pending).HP >= 200 {
				t.Fatal("accepted requested victim was never struck")
			}
			if !tc.present && laEnt(t, w, 3).HP != 200 {
				t.Fatal("return to active revived the old request")
			}
		})
	}
}

func TestPendingAttackLaterOrderCancelsRequestedVictim(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command Command
	}{
		{"move", MoveTo(1, CellPoint{X: 17, Y: 20})},
		{"Defend", GroupDefend(1, 4, 1)},
		{"Patrol", GroupPatrolTo(1, CellPoint{X: 17, Y: 20}, 1)},
		{"use", UseStructure(1, 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := pendingPair(t)
			if tc.name == "use" {
				w.structures = []Structure{{ID: 0, Col: 14, Row: 20, MaxHealth: 1, Width: 1, Height: 1, Kind: 28, Attach: 1, Blocking: 1}}
				w.rebuildStructureSlots()
			}
			Step(w, []Command{tc.command})
			if e := laEnt(t, w, 1); e.HasPendingAttackTarget || e.AttackTarget != 2 || e.AttackPhase == AttackReady {
				t.Fatalf("later order lost active cycle or retained request: %+v", e)
			}
			if _, err := w.MarshalBinary(); err != nil {
				t.Fatal("later command left incompatible state", err)
			}
			for range 140 {
				Step(w, nil)
				if laEnt(t, w, 1).HasPendingAttackTarget {
					t.Fatal("cancelled request revived")
				}
			}
		})
	}
}

func TestPendingAttackVetoCancelsAcceptedRequest(t *testing.T) {
	w := pendingPair(t)
	w.entities[indexOfEntity(w.entities, 1)].Reach = 1
	w.entities[indexOfEntity(w.entities, 4)].Domain = DomainAir
	if !w.targetVetoed(indexOfEntity(w.entities, 1), indexOfEntity(w.entities, 4)) {
		t.Fatal("fixture did not reach preference veto")
	}
	Step(w, []Command{Attack(1, 4)})
	e := laEnt(t, w, 1)
	if e.HasPendingAttackTarget || e.HasAttackTarget || e.AttackPhase != AttackReady || e.ActorState != actorStateAcquire {
		t.Fatal("veto revived the accepted request or retained the replaced primary action")
	}
	if _, err := w.MarshalBinary(); err != nil {
		t.Fatal(err)
	}
}

func TestPendingAttackRepeatedSavedEngagementKeepsThePair(t *testing.T) {
	w := lcBuild(t, SelfSlot, true, orderNone, lcVictim(3, 2, 24, 21))
	Step(w, []Command{Attack(1, 3)})
	cold := worldRoundTripForTest(t, w)
	control := worldRoundTripForTest(t, w)
	control.entities[indexOfEntity(control.entities, 1)].clearPendingAttack()
	oldApplied, reissued := false, false
	for range 180 {
		Step(w, nil)
		Step(cold, nil)
		Step(control, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("saved engagement changed after cold LOAD")
		}
		if e := laEnt(t, w, 1); e.HasPendingAttackTarget {
			c := laEnt(t, control, 1)
			if e.AttackTarget != 2 || e.AttackPhase != c.AttackPhase || e.AttackCountdown != c.AttackCountdown {
				t.Fatal("state3 reissue changed the running cycle")
			}
			if w.savedOrder(1).Raw[8] == 5 {
				reissued = true
			}
		}
		if !oldApplied && laEnt(t, w, 2).HP != 200 {
			if laEnt(t, w, 2).HP != 195 || laEnt(t, w, 3).HP != 200 {
				t.Fatal("saved state3 replaced the loaded victim")
			}
			oldApplied = true
		}
		if laEnt(t, w, 3).HP != 200 {
			if !oldApplied || !reissued || laEnt(t, w, 2).HP != 195 || laEnt(t, w, 3).HP != 195 {
				t.Fatal("saved state3 did not take the requested victim after recovery")
			}
			return
		}
	}
	t.Fatal("repeated saved engagement lost the request")
}

func TestPendingAttackScriptSetterKeepsTheLoadedVictim(t *testing.T) {
	w := lcBuild(t, SelfSlot, false, orderNone, lcVictim(3, 2, 24, 21))
	w.runInstant(lcGroupOrder(lcGroup, 3, subCommandAttack))
	e := laEnt(t, w, 1)
	if e.AttackTarget != 2 || !e.HasPendingAttackTarget || e.PendingAttackTarget != 3 || e.AttackPhase != AttackCharging {
		t.Fatal("script explicit setter replaced the loaded victim")
	}
	for range 180 {
		Step(w, nil)
		if laEnt(t, w, 3).HP < 200 {
			if laEnt(t, w, 2).HP != 195 {
				t.Fatal("script attack lost or repeated the old strike")
			}
			return
		}
	}
	t.Fatal("script attack never reached the requested victim")
}

func TestPendingAttackBytesHashAndOldBytes(t *testing.T) {
	w := pendingPair(t)
	form := mustMarshal(t, w)
	o := headerLen + 3*int(w.bounds.Width*w.bounds.Height)
	if form[0] != 95 || form[o+24] != 2 || binary.LittleEndian.Uint32(form[o+12:]) != 3 || binary.LittleEndian.Uint32(form[o+16:]) != 0 || binary.LittleEndian.Uint32(form[o+44:]) != 2 {
		t.Fatal("pending pair did not use the local destination tag")
	}
	cold := worldRoundTripForTest(t, w)
	if cold.Hash() != w.Hash() || !bytes.Equal(form, mustMarshal(t, cold)) {
		t.Fatal("pending bytes did not round trip")
	}
	lost := worldRoundTripForTest(t, w)
	lost.entities[0].clearPendingAttack()
	if lost.Hash() == w.Hash() {
		t.Fatal("pending target does not discriminate the hash")
	}
	for range 160 {
		Step(w, nil)
		Step(cold, nil)
		Step(lost, nil)
		if cold.Hash() != w.Hash() {
			t.Fatal("cold pending continuation diverged")
		}
	}
	if laEnt(t, w, 3).HP >= 200 || laEnt(t, lost, 3).HP != 200 {
		t.Fatal("deleting only pending state did not break requested strike")
	}
	if got := mustMarshal(t, pinWorld(t)); !bytes.Equal(got, pinBytes) {
		t.Fatal("unqueued historical form changed")
	}
}

func TestPendingAttackMalformedBytesAreAtomic(t *testing.T) {
	w := pendingPair(t)
	good := mustMarshal(t, w)
	o := headerLen + 3*int(w.bounds.Width*w.bounds.Height)
	for _, control := range []string{"unknown tag", "second operand", "absent victim", "self victim", "same victim", "stall", "ready residue", "structure kind"} {
		t.Run(control, func(t *testing.T) {
			bad := bytes.Clone(good)
			switch control {
			case "unknown tag":
				bad[o+24] = 4
			case "second operand":
				bad[o+16] = 1
			case "absent victim":
				binary.LittleEndian.PutUint32(bad[o+12:], 99)
			case "self victim":
				binary.LittleEndian.PutUint32(bad[o+12:], 1)
			case "same victim":
				binary.LittleEndian.PutUint32(bad[o+12:], 2)
			case "stall":
				bad[o+25] = 1
			case "ready residue":
				bad[o+48] = 0
			case "structure kind":
				bad[o+24] = 3
			}
			before := w.Hash()
			if err := w.UnmarshalBinary(bad); err == nil || before != w.Hash() || !bytes.Equal(good, mustMarshal(t, w)) {
				t.Fatal("malformed pending bytes accepted or changed receiver", err)
			}
		})
	}
}

func TestPendingAttackBorrowedOperandsExcludeStructureUse(t *testing.T) {
	w := pendingPair(t)
	w.structures = []Structure{{ID: 0, Col: 14, Row: 20, MaxHealth: 1, Width: 1, Height: 1, Kind: 28, Attach: 1, Blocking: 1}}
	w.rebuildStructureSlots()
	Step(w, []Command{UseStructure(1, 0)})
	good := mustMarshal(t, w)
	bad := bytes.Clone(good)
	o := headerLen + 3*int(w.bounds.Width*w.bounds.Height)
	bad[o+24] = 2
	binary.LittleEndian.PutUint32(bad[o+12:], 3)
	binary.LittleEndian.PutUint32(bad[o+16:], 0)
	if err := w.UnmarshalBinary(bad); err == nil || !bytes.Equal(good, mustMarshal(t, w)) {
		t.Fatal("pending tag silently discarded structure use", err)
	}
	w.entities[indexOfEntity(w.entities, 1)].PendingAttackTarget = 3
	w.entities[indexOfEntity(w.entities, 1)].HasPendingAttackTarget = true
	if _, err := w.MarshalBinary(); err == nil {
		t.Fatal("producer accepted pending attack with destination/use")
	}
}

func TestPendingAttackVictimRemovalAndLethalOldStrike(t *testing.T) {
	t.Run("requested victim killed", func(t *testing.T) {
		killer := lcArcher(5, SelfSlot, 23, 21)
		killer.AttackCharge, killer.AttackRelax, killer.DamageBase = 1, 1, 250
		w := lcPlayerWorld(t, lcVictim(3, 2, 24, 21), killer)
		Step(w, []Command{Attack(1, 3), Attack(5, 3)})
		for range 20 {
			Step(w, nil)
			if !entityRef(laEnt(t, w, 3)).OrdinaryTargetable() {
				break
			}
		}
		if entityRef(laEnt(t, w, 3)).OrdinaryTargetable() {
			t.Fatal("requested victim did not reach the terminal floor")
		}
		if e := laEnt(t, w, 1); e.HasPendingAttackTarget || e.AttackTarget != 2 {
			t.Fatalf("dead requested victim replaced the active one: %+v", e)
		}
		if _, err := w.MarshalBinary(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("old strike reaches terminal floor", func(t *testing.T) {
		archer, first := lcArcher(1, SelfSlot, 20, 20), lcVictim(2, 2, 24, 20)
		archer.DamageBase, first.HP = 25, 1
		rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
		w := engWorld(t, rel, archer, first, lcVictim(3, 2, 24, 21))
		Step(w, []Command{Attack(1, 2)})
		Step(w, []Command{Attack(1, 3)})
		var killed bool
		for range 180 {
			Step(w, nil)
			if laEnt(t, w, 2).HP <= -10 {
				killed = true
			}
			if killed && laEnt(t, w, 3).HP < 200 {
				return
			}
		}
		t.Fatal("lethal active cleanup lost the requested victim")
	})
}

func TestPendingAttackDuringWeaponCastKeepsActiveInputs(t *testing.T) {
	caster := wpnCaster(1, 4, 4, 1, 30, 40, 30)
	caster.Owner, caster.Reach, caster.ScanRange = SelfSlot, 4, 6
	first, next := wpnEnt(2, 5, 4), wpnEnt(3, 4, 5)
	first.Owner, next.Owner = 2, 2
	w := spWorld(t, 17, []SpellRule{wpnRule(6, 6, 5)}, caster, first, next)
	w.relations = engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
	Step(w, []Command{Attack(1, 2)})
	if laEnt(t, w, 1).AttackPhase != AttackCasting {
		t.Fatal("weapon cast did not load")
	}
	control := worldRoundTripForTest(t, w)
	Step(control, nil)
	Step(w, []Command{Attack(1, 3)})
	e, old := laEnt(t, w, 1), laEnt(t, control, 1)
	if e.AttackTarget != 2 || !e.HasPendingAttackTarget || e.PendingAttackTarget != 3 || e.AttackPhase != old.AttackPhase || e.AttackCountdown != old.AttackCountdown || e.WeaponSpell != old.WeaponSpell || e.WeaponSpellLevel != old.WeaponSpellLevel || e.Mana != old.Mana {
		t.Fatal("explicit Attack changed a loaded weapon cast")
	}
	cold := worldRoundTripForTest(t, w)
	oldApplied := false
	for range 200 {
		Step(w, nil)
		Step(cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatal("weapon cast pending state changed after LOAD")
		}
		if !oldApplied && laEnt(t, w, 2).HP != 100 {
			if laEnt(t, w, 2).HP != 88 || laEnt(t, w, 3).HP != 100 {
				t.Fatal("loaded spell reached the wrong victim")
			}
			oldApplied = true
		}
		if laEnt(t, w, 3).HP != 100 {
			e := laEnt(t, w, 1)
			if !oldApplied || laEnt(t, w, 2).HP != 88 || laEnt(t, w, 3).HP != 88 || e.AttackTarget != 3 || e.HasPendingAttackTarget || e.WeaponSpell != 1 || e.WeaponSpellLevel != 30 || e.Mana != 100 {
				t.Fatal("requested attack lost its weapon mode or ran before the old spell")
			}
			return
		}
	}
	t.Fatal("weapon casts did not reach old and requested victims")
}

func TestPendingAttackZeroAndStructureNamespaces(t *testing.T) {
	for _, kind := range []AttackTargetKind{AttackTargetUnit, AttackTargetStructure} {
		archer := lcArcher(1, SelfSlot, 20, 20)
		rel := engRel(t, [3]uint32{SelfSlot, 2, relationHostile}, [3]uint32{2, SelfSlot, 2})
		structure := Structure{ID: 0, Col: 22, Row: 21, Width: 1, Height: 1, Field42: 100, MaxHealth: 100, Attach: 1, Blocking: 1}
		w, err := NewStructuredWorld(1, engBounds, ModeCanonical, Terrain{}, []Entity{archer, lcVictim(0, 2, 24, 21), lcVictim(2, 2, 24, 20)}, nil, rel, nil, nil, nil, GhostTemplate{}, []Structure{structure})
		if err != nil {
			t.Fatal(err)
		}
		Step(w, []Command{Attack(1, 2)})
		command := Attack(1, 0)
		if kind == AttackTargetStructure {
			command = AttackStructure(1, 0)
		}
		Step(w, []Command{command})
		cold := worldRoundTripForTest(t, w)
		e := laEnt(t, cold, 1)
		if !e.HasPendingAttackTarget || e.PendingAttackTarget != 0 || e.PendingAttackTargetKind != kind {
			t.Fatalf("zero pending handle lost its namespace: %+v", e)
		}
		a := cold.Actions()
		if err := a.RemapActors(func(id EntityID, structure bool) (EntityID, error) {
			if structure {
				return id + 100, nil
			}
			return id + 10, nil
		}); err != nil {
			t.Fatal(err)
		}
		var got ActorContinuation
		for _, v := range a.Actors {
			if v.Entity == 11 {
				got = v
			}
		}
		want := EntityID(10)
		if kind == AttackTargetStructure {
			want = 100
		}
		if !got.HasPendingAttackTarget || got.PendingAttackTarget != want || got.AttackTarget != 12 {
			t.Fatalf("typed remap conflated namespaces: %+v", got)
		}
	}
}
