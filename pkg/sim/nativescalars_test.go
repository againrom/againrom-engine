package sim

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

func nativeScalarLiteralForm(base []byte, rows []NativeActorBasisRecord) []byte {
	out := bytes.Clone(base)
	previous, start := out[0], len(out)
	out[0] = 114
	out = binary.LittleEndian.AppendUint32(out, uint32(len(rows)))
	for _, row := range rows {
		flag := byte(0)
		if row.Basis.ScalarsPresent {
			flag |= 1
		}
		if row.Basis.BlockPresent {
			flag |= 2
		}
		out = binary.LittleEndian.AppendUint32(out, uint32(row.ID))
		out = append(out, flag)
		out = binary.LittleEndian.AppendUint32(out, row.Basis.ScalarKnown)
		for _, value := range row.Basis.Scalars {
			out = binary.LittleEndian.AppendUint32(out, value)
		}
		out = binary.LittleEndian.AppendUint16(out, row.Basis.BlockKnown)
		out = append(out, row.Basis.Block[:]...)
	}
	out = binary.LittleEndian.AppendUint32(out, uint32(len(out)-start))
	return append(out, previous, 'N', 'S', 'C', '1')
}

func TestNativeScalarsLiteralFormOldABIAndCurrentAction(t *testing.T) {
	slots := []int{ScalarT0C, ScalarT08High, ScalarT18, ScalarT1C, ScalarReference, ScalarU4B, ScalarU4C, ScalarU6C, ScalarU8E, ScalarU60, ScalarU61, ScalarUA0, ScalarUA4, ScalarU130, ScalarU136, ScalarU138, ScalarU148, ScalarU144, ScalarU50, ScalarU54, ScalarU58}
	if len(slots) != 21 || ScalarCount != 21 {
		t.Fatal("native scalar slot count changed")
	}
	for index, slot := range slots {
		if index != slot {
			t.Fatal("native scalar slot order changed", index, slot)
		}
	}
	w := nativeBasisWorld(t)
	oldBasis := (NativeActorBasis{}).WithBase([24]byte{2, 7}).WithModifier([64]byte{19}).WithBody(40)
	oldBasis.AttackPresent, oldBasis.AttackKnown, oldBasis.Attack[22] = true, 1<<22, 0x91
	w.entities[0].NativeBasis = oldBasis
	old, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	basis := oldBasis
	basis.ScalarsPresent, basis.ScalarKnown = true, 1<<ScalarT0C|1<<ScalarReference|1<<ScalarU130
	for index := range basis.Scalars {
		basis.Scalars[index] = 0x81000000 + uint32(index)
	}
	basis.Scalars[ScalarT0C], basis.Scalars[ScalarReference], basis.Scalars[ScalarU130] = 0, 0xfedcba98, 0x12345678
	basis.BlockPresent, basis.BlockKnown, basis.Block = true, 1<<0|1<<9, [10]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	zero := NativeActorBasis{ScalarsPresent: true}
	rows := []NativeActorBasisRecord{{ID: 0, Basis: basis}, {ID: 3, Basis: zero}}
	if err := w.RestoreNativeActorBases(rows); err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	expected := nativeScalarLiteralForm(old, rows)
	if !bytes.Equal(raw, expected) || binary.LittleEndian.Uint32(raw[len(raw)-9:]) != 4+2*105 {
		t.Fatal("native scalar section changed the literal form or old ABI")
	}
	if err := CheckSaveForm(raw); err != nil {
		t.Fatal("native scalar form refused", err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || cold.entities[0].NativeBasis != basis || cold.entities[1].NativeBasis != zero {
		t.Fatal("native scalar cold LOAD lost current values, masks or residue", err)
	}
	plain := nativeBasisWorld(t)
	actions := w.Actions()
	if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != basis || plain.RestoreActions(actions, nil) != nil || plain.entities[0].NativeBasis != basis {
		t.Fatal("current action lost native scalar history")
	}
	if err := cold.UnmarshalBinary(old); err != nil || cold.entities[0].NativeBasis != oldBasis || cold.entities[1].NativeBasis.HasValues() {
		t.Fatal("old form invented scalars or replaced old basis", err)
	}
	again, err := cold.MarshalBinary()
	if err != nil || !bytes.Equal(again, old) {
		t.Fatal("scalar absence changed the historical ABI", err)
	}
}

func TestNativeScalarsValuesMasksAndPresenceAffectHash(t *testing.T) {
	w := nativeBasisWorld(t)
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: (1 << 21) - 1, BlockPresent: true, BlockKnown: (1 << 10) - 1}
	w.entities[0].NativeBasis = basis
	before := w.Hash()
	for slot := 0; slot < 21; slot++ {
		changed := basis
		changed.ScalarKnown &^= 1 << slot
		w.entities[0].NativeBasis = changed
		if w.Hash() == before {
			t.Fatal("known scalar bit missing from hash", slot)
		}
		changed.Scalars[slot] = 0xfedcba98
		unknown := w.Hash()
		w.entities[0].NativeBasis = changed
		if w.Hash() == unknown {
			t.Fatal("unknown scalar residue missing from hash", slot)
		}
	}
	for index := 0; index < 10; index++ {
		changed := basis
		changed.Block[index] = byte(index + 1)
		w.entities[0].NativeBasis = changed
		if w.Hash() == before {
			t.Fatal("Block byte missing from hash", index)
		}
		changed = basis
		changed.BlockKnown &^= 1 << index
		w.entities[0].NativeBasis = changed
		if w.Hash() == before {
			t.Fatal("Block known bit missing from hash", index)
		}
	}
	for _, changed := range []NativeActorBasis{{ScalarsPresent: true}, {BlockPresent: true}, {}} {
		w.entities[0].NativeBasis = changed
		if w.Hash() == before {
			t.Fatal("scalar or Block availability missing from hash")
		}
	}
}

func TestNativeScalarsMalformedFormAndOwnerBatchAreAtomic(t *testing.T) {
	w := nativeBasisWorld(t)
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: (1 << 21) - 1, BlockPresent: true, BlockKnown: 1}
	w.entities[0].NativeBasis, w.entities[1].NativeBasis = basis, basis
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	before := w.Hash()
	start := len(raw) - 9 - (4 + 2*105)
	for _, mutate := range []func([]byte){
		func(b []byte) { b[len(b)-1] = '2' },
		func(b []byte) { b[len(b)-5] = 114 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[len(b)-9:], ^uint32(0)) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 0) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start:], 65536) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4:], 99) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+4+105:], 0) },
		func(b []byte) { b[start+8] = 0 },
		func(b []byte) { b[start+8] = 4 },
		func(b []byte) { b[start+8] = 1 },
		func(b []byte) { b[start+8] = 2 },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+9:], 1<<21) },
		func(b []byte) { binary.LittleEndian.PutUint16(b[start+97:], 1<<10) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+13+4*ScalarT0C:], 256) },
		func(b []byte) { binary.LittleEndian.PutUint32(b[start+13+4*ScalarT08High:], 65536) },
	} {
		bad := bytes.Clone(raw)
		mutate(bad)
		if w.UnmarshalBinary(bad) == nil || w.Hash() != before {
			t.Fatal("malformed scalar decode accepted or changed current state")
		}
	}
	if w.UnmarshalBinary(raw[:len(raw)-1]) == nil || w.Hash() != before {
		t.Fatal("truncated scalar form accepted or changed current state")
	}
	bad := basis
	bad.ScalarsPresent = false
	for _, rows := range [][]NativeActorBasisRecord{{{ID: 0, Basis: basis}, {ID: 99, Basis: basis}}, {{ID: 3, Basis: basis}, {ID: 0, Basis: basis}}, {{ID: 0, Basis: bad}}} {
		if w.RestoreNativeActorBases(rows) == nil || w.Hash() != before {
			t.Fatal("invalid scalar owner batch accepted or changed state")
		}
	}
	actions := w.Actions()
	actions.Actors[0].Current.NativeBasis = &bad
	if w.RestoreActions(actions, nil) == nil || w.Hash() != before {
		t.Fatal("invalid scalar action accepted or changed state")
	}
	source := nativeBasisWorld(t)
	source.entities[0].ActorLoad.Source.Class = 2
	entities := source.Entities()
	if source.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 0, Basis: basis}}) == nil || !reflect.DeepEqual(source.Entities(), entities) {
		t.Fatal("source actor acquired native scalar history")
	}
}

func TestNativeScalarsSurviveActualRemovalAndRetainedDead(t *testing.T) {
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarT18, BlockPresent: true, BlockKnown: 1 << 9, Block: [10]byte{0: 0x95, 9: 0x63}}
	basis.Scalars[ScalarT18], basis.Scalars[ScalarReference] = 0, 0xfedcba98
	for _, retained := range []bool{false, true} {
		var w *World
		id := EntityID(0)
		if retained {
			w, id = deadWorld(t), 1
			if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(id, 3, -40)}); err != nil {
				t.Fatal(err)
			}
		} else {
			w = spWorld(t, 9, nil, spEnt(id, 10, 11), spEnt(8, 1, 1))
		}
		if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: id, Basis: basis}}); err != nil {
			t.Fatal(err)
		}
		i := indexOfEntity(w.entities, id)
		w.entities[i].HP, w.entities[i].Decay = decayGoneHP-1, decayLast
		Step(w, nil)
		if _, exists := w.Entity(id); exists || !reflect.DeepEqual(w.RemovedNativeActorBases(), []NativeActorBasisRecord{{ID: id, Basis: basis}}) {
			t.Fatal("actual native removal lost scalar-only history", retained)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.RemovedNativeActorBases(), w.RemovedNativeActorBases()) {
			t.Fatal("removed scalar cold LOAD lost exact owner/history", retained, err)
		}
		Step(&cold, nil)
		if _, err := cold.MarshalBinary(); err != nil {
			t.Fatal("next tick cannot save removed scalar history", err)
		}
	}
}

func TestNativeScalarObservedAggregateFollowsActualAward(t *testing.T) {
	for _, known := range []bool{false, true} {
		for _, raises := range []bool{false, true} {
			w := trainingWorld(t, 17, 0)
			e := &w.entities[0]
			e.NativeBasis.ScalarsPresent = true
			e.NativeBasis.Scalars[ScalarU130] = ^uint32(0) - 1
			if known {
				e.NativeBasis.ScalarKnown = 1 << ScalarU130
			}
			if !raises {
				e.SkillXP[3] = 0
			}
			before := *e
			if got := w.awardSkill(0, 0, 1_000_000, 1); got != raises {
				t.Fatal("fixture award raise differs", got, raises)
			}
			delta := e.SkillXP[3] - before.SkillXP[3]
			want := before.NativeBasis
			if known {
				want.Scalars[ScalarU130] += uint32(delta)
			}
			if delta <= 0 || e.NativeBasis != want {
				t.Fatal("observed aggregate did not follow only its actual wrapping award", known, raises, delta, e.NativeBasis, want)
			}
		}
	}
}

func TestNativeScalarMaintainedPartsKeepIndependentBits(t *testing.T) {
	w := nativeBasisWorld(t)
	e := &w.entities[0]
	e.ScanRange, e.OffMap = 7, true
	e.NativeClass = NativeClass{Present: true, Fighter: false}
	e.TypeID, e.XPValue = 0x2a, 59
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1<<ScalarUA4 | 1<<ScalarU4C | 1<<ScalarT1C | 1<<ScalarU6C}
	basis.Scalars[ScalarUA4], basis.Scalars[ScalarU4C] = 0x09ab, 0xa1
	basis.Scalars[ScalarT1C], basis.Scalars[ScalarU6C] = 17, 63
	e.NativeBasis = basis
	want := basis
	want.Scalars[ScalarUA4], want.Scalars[ScalarU4C] = 0x07ab, 0xad
	if NativeBasisNow(*e) != want || e.NativeBasis != basis {
		t.Fatal("current scalar composition lost independent bits or changed backing history")
	}
	actions := w.Actions()
	if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != want {
		t.Fatal("action carrier disagrees with maintained scalar composition")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
		t.Fatal("maintained scalar cold form differs", err)
	}
	start := len(raw) - 9 - (4 + 105)
	bad := bytes.Clone(raw)
	binary.LittleEndian.PutUint32(bad[start+13+4*ScalarUA4:], 0x08ab)
	before := cold.Hash()
	if cold.UnmarshalBinary(bad) == nil || cold.Hash() != before {
		t.Fatal("noncanonical maintained scalar form accepted or changed state")
	}
	e.NativeBasis.ScalarKnown &^= 1<<ScalarUA4 | 1<<ScalarU4C
	if NativeBasisNow(*e) != e.NativeBasis {
		t.Fatal("maintained fields promoted unknown scalar history")
	}
}

func TestNativeScalarCurrentClockKeepsUnknownHistory(t *testing.T) {
	for _, tc := range []struct {
		name                 string
		clockKnown, rawKnown bool
		end, raw, want       uint32
	}{
		{"known zero", true, true, 0, 77, 0},
		{"known nonzero", true, true, 17, 0, 17},
		{"unknown clock", false, true, 0, 77, 77},
		{"unknown scalar", true, false, 17, 77, 77},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := nativeBasisWorld(t)
			e := &w.entities[0]
			e.ActionClock = ActionClock{Known: tc.clockKnown, End: tc.end}
			e.NativeBasis = NativeActorBasis{ScalarsPresent: true}
			e.NativeBasis.Scalars[ScalarU138] = tc.raw
			if tc.rawKnown {
				e.NativeBasis.ScalarKnown = 1 << ScalarU138
			}
			before, want := e.NativeBasis, e.NativeBasis
			want.Scalars[ScalarU138] = tc.want
			if NativeBasisNow(*e) != want || e.NativeBasis != before {
				t.Fatal("current clock changed unknown history, masks or backing values")
			}
			actions := w.Actions()
			if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != want {
				t.Fatal("action carrier lost current known clock priority")
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
				t.Fatal("current clock scalar cold form differs", err)
			}
		})
	}
}

func TestNativeScalarCreatureExperienceKeepsOtherHistory(t *testing.T) {
	for _, tc := range []struct {
		name      string
		humanoid  bool
		typeID    int32
		xp        int32
		known     bool
		raw, want uint32
	}{
		{"creature", false, 0x1a, 12, true, 0, 12},
		{"known zero", false, 0x2a, 0, true, 77, 0},
		{"signed value", false, 0x2a, -1, true, 0, ^uint32(0)},
		{"human", true, 0x2a, 12, true, 31, 31},
		{"noncreature", false, 0x19, 12, true, 31, 31},
		{"unknown scalar", false, 0x2a, 12, false, 31, 31},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := nativeBasisWorld(t)
			e := &w.entities[0]
			e.Humanoid, e.TypeID, e.XPValue = tc.humanoid, tc.typeID, tc.xp
			basis := NativeActorBasis{ScalarsPresent: true}
			basis.Scalars[ScalarT1C], basis.Scalars[ScalarReference] = tc.raw, 0xfedcba98
			if tc.known {
				basis.ScalarKnown = 1 << ScalarT1C
			}
			e.NativeBasis = basis
			want := basis
			want.Scalars[ScalarT1C] = tc.want
			if NativeBasisNow(*e) != want || e.NativeBasis != basis {
				t.Fatal("creature experience changed masks, independent history or backing values")
			}
			actions := w.Actions()
			if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != want {
				t.Fatal("action carrier lost current creature experience priority")
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
				t.Fatal("creature experience scalar cold form differs", err)
			}
			e.NativeBasis = NativeActorBasis{}
			if NativeBasisNow(*e).HasValues() {
				t.Fatal("creature experience invented an absent carrier")
			}
		})
	}
}

func TestNativeScalarCreatureExperienceSurvivesActualRemoval(t *testing.T) {
	w := nativeBasisWorld(t)
	e := &w.entities[0]
	e.Humanoid, e.TypeID, e.XPValue = false, 0x2a, 12
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1 << ScalarT1C}
	basis.Scalars[ScalarReference] = 0xfedcba98
	e.NativeBasis = basis
	e.HP, e.Decay = decayGoneHP-1, decayLast
	Step(w, nil)
	want := basis
	want.Scalars[ScalarT1C] = 12
	if _, live := w.Entity(0); live || !reflect.DeepEqual(w.RemovedNativeActorBases(), []NativeActorBasisRecord{{ID: 0, Basis: want}}) {
		t.Fatal("actual removal lost the current creature experience or independent history")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.RemovedNativeActorBases(), w.RemovedNativeActorBases()) {
		t.Fatal("removed creature experience scalar cold form differs", err)
	}
}

func TestNativeScalarCurrentMotionOrderAndTerminalSnapshots(t *testing.T) {
	for _, tc := range []struct {
		name          string
		set           func(*World)
		state, action uint32
	}{
		{"absent motion plane idle", func(*World) {}, 11, 0},
		{"present motion plane without row", func(w *World) { w.savedMotion = &savedActorMotionState{} }, 11, 1},
		{"absent motion plane crossing", func(w *World) {
			e := &w.entities[0]
			e.X, e.Transit, e.TransitTotal = 1, 15, 16
			e.Stride = NativeStride{Present: true, FromX: 0, FromY: 0, ToX: 1, ToY: 0, Rate: 16, StepX: 16, Direction: 2}
		}, 11, 1},
		{"current imported motion", func(w *World) {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Current: true, Position: SavedActorPosition{FineX: 128, FineY: 128}, ActorAction: 17}}}
		}, 11, 17},
		{"superseded motion", func(w *World) {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Issue: "native order supersedes imported motion", ActorAction: 0}}}
		}, 11, 0},
		{"current order", func(w *World) { w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{{Entity: 0, State: 10}}} }, 10, 0},
		{"native patrol", func(w *World) { w.entities[0].ActorState, w.entities[0].PatrolTailX = actorStatePatrol, 2 }, 10, 0},
		{"terminal", func(w *World) { w.entities[0].HP, w.entities[0].Decay = -10, DecayBones }, 16, 16},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, known := range []uint32{1<<ScalarU50 | 1<<ScalarU54, 1 << ScalarU50, 0} {
				w := nativeBasisWorld(t)
				basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: known, BlockPresent: true, BlockKnown: 1 << 8, Block: [10]byte{2: 37, 8: 59}}
				basis.Scalars[ScalarU50], basis.Scalars[ScalarU54], basis.Scalars[ScalarU58] = 11, 1, 0xfedcba98
				w.entities[0].NativeBasis = basis
				tc.set(w)
				want := basis
				if basis.ScalarIsKnown(ScalarU50) {
					want.Scalars[ScalarU50] = tc.state
				}
				if basis.ScalarIsKnown(ScalarU54) {
					want.Scalars[ScalarU54] = tc.action
				}
				before := w.entities[0]
				e, ok := w.Entity(0)
				if !ok || e.NativeBasis != want || w.Entities()[0].NativeBasis != want || NativeBasisNow(e) != want {
					t.Fatal("current snapshot lost motion/order/terminal authority or independent history", known, e.NativeBasis, want)
				}
				if w.entities[0] != before {
					t.Fatal("current snapshot changed live backing state")
				}
				a := w.Actions()
				if a.Actors[0].Current.NativeBasis == nil || *a.Actors[0].Current.NativeBasis != want {
					t.Fatal("current action disagrees with entity snapshot")
				}
				raw, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var cold World
				if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
					t.Fatal("current motion/order/terminal scalar cold form differs", err)
				}
				w.entities[0].NativeBasis = NativeActorBasis{}
				if e, _ := w.Entity(0); e.NativeBasis.HasValues() {
					t.Fatal("current carrier invented an absent native basis")
				}
			}
		})
	}
}

func TestNativeScalarPhysicalAttackOverridesMotion(t *testing.T) {
	for _, tc := range []struct {
		name   string
		phase  AttackPhase
		set    func(*World)
		action uint32
	}{
		{"ready", AttackReady, func(*World) {}, 3},
		{"charging", AttackCharging, func(w *World) { w.entities[0].AttackCountdown = 2 }, 3},
		{"relaxing", AttackRelaxing, func(w *World) { w.entities[0].AttackCountdown = 2 }, 3},
		{"first boundary", AttackBoundaryOne, func(*World) {}, 3},
		{"second boundary", AttackBoundaryTwo, func(*World) {}, 0},
		{"first boundary with both idle carriers", AttackBoundaryOne, func(w *World) {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Current: true, Position: SavedActorPosition{FineX: 128, FineY: 128}}}}
			w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{{Entity: 0, State: 11}}}
		}, 0},
		{"first boundary with idle motion only", AttackBoundaryOne, func(w *World) {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Current: true, Position: SavedActorPosition{FineX: 128, FineY: 128}}}}
		}, 3},
		{"first boundary with nonidle order", AttackBoundaryOne, func(w *World) {
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Current: true, Position: SavedActorPosition{FineX: 128, FineY: 128}}}}
			order := SavedActorOrder{Entity: 0, State: 11}
			order.Raw[9] = 1
			w.savedGroups = &savedGroupState{Orders: []SavedActorOrder{order}}
		}, 3},
		{"released target", AttackReady, func(w *World) {
			w.entities[0].HasAttackTarget, w.entities[0].AttackTarget = false, 0
		}, 0},
		{"crossing", AttackReady, func(w *World) {
			e := &w.entities[0]
			e.X, e.Transit, e.TransitTotal = 1, 15, 16
			e.Stride = NativeStride{Present: true, FromX: 0, FromY: 0, ToX: 1, ToY: 0, Rate: 16, StepX: 16, Direction: 2}
		}, 1},
		{"turning", AttackReady, func(w *World) {
			e := &w.entities[0]
			e.RotationSpeed, e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 16, 32, 1, 1
		}, 0},
		{"casting phase without a held Spell", AttackCasting, func(w *World) {
			w.entities[0].AttackCountdown = 2
			w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 0, Current: true, Position: SavedActorPosition{FineX: 128, FineY: 128}, ActorAction: 13}}}
		}, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, known := range []uint32{1 << ScalarU54, 0} {
				w := nativeBasisWorld(t)
				e := &w.entities[0]
				e.AttackTarget, e.HasAttackTarget, e.AttackPhase = 3, true, tc.phase
				e.AttackCharge, e.AttackRelax = 5, 5
				basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: known}
				basis.Scalars[ScalarU54], basis.Scalars[ScalarU58] = 3, 0xfedcba98
				e.NativeBasis = basis
				tc.set(w)
				want := basis
				if known != 0 {
					want.Scalars[ScalarU54] = tc.action
				}
				before := w.entities[0]
				current, _ := w.Entity(0)
				if current.NativeBasis != want || w.Entities()[0].NativeBasis != want || NativeBasisNow(current) != want || w.entities[0] != before {
					t.Fatal("physical action lost current priority, unknown history, or backing immutability", known, current.NativeBasis, want)
				}
				if a := w.Actions(); a.Actors[0].Current.NativeBasis == nil || *a.Actors[0].Current.NativeBasis != want {
					t.Fatal("physical action continuation differs from current scalar")
				}
				raw, err := w.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				var cold World
				if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
					t.Fatal("physical action scalar cold form differs", err)
				}
			}
		})
	}
}

func TestNativeScalarRetainedTerminalOrderUsesExactOwner(t *testing.T) {
	w := deadWorld(t)
	if err := w.ImportOriginalDeadActors([]OriginalDeadActor{deadInput(1, 5, -10001)}); err != nil {
		t.Fatal(err)
	}
	basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: 1<<ScalarU50 | 1<<ScalarU54}
	basis.Scalars[ScalarU50], basis.Scalars[ScalarU54], basis.Scalars[ScalarU58] = 11, 1, 0xfedcba98
	if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 1, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	want := basis
	want.Scalars[ScalarU50], want.Scalars[ScalarU54] = 16, 16
	if !reflect.DeepEqual(w.RemovedNativeActorBases(), []NativeActorBasisRecord{{ID: 1, Basis: want}}) || w.removedNativeBases[0].Basis != basis {
		t.Fatal("retained terminal snapshot lost exact current order or changed backing history")
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.RemovedNativeActorBases(), w.RemovedNativeActorBases()) {
		t.Fatal("retained terminal scalar cold form differs", err)
	}
}

func TestNativeScalarCurrentTerminalOrderDoesNotRequireBoneStage(t *testing.T) {
	for _, stage := range []uint8{0, 1, 4} {
		for _, known := range []uint32{1<<ScalarU50 | 1<<ScalarU54, 1 << ScalarU50, 0} {
			w := nativeBasisWorld(t)
			if err := w.RestoreCurrentTerminalActors([]CurrentTerminalActor{{ID: 0, HP: -10001, Stage: stage}}); err != nil {
				t.Fatal(err)
			}
			basis := NativeActorBasis{ScalarsPresent: true, ScalarKnown: known}
			basis.Scalars[ScalarU50], basis.Scalars[ScalarU54], basis.Scalars[ScalarU58] = 11, 0, 0xfedcba98
			if err := w.RestoreNativeActorBases([]NativeActorBasisRecord{{ID: 0, Basis: basis}}); err != nil {
				t.Fatal(err)
			}
			want := basis
			if basis.ScalarIsKnown(ScalarU50) {
				want.Scalars[ScalarU50] = 16
			}
			if basis.ScalarIsKnown(ScalarU54) {
				want.Scalars[ScalarU54] = 16
			}
			if !reflect.DeepEqual(w.RemovedNativeActorBases(), []NativeActorBasisRecord{{ID: 0, Basis: want}}) || w.removedNativeBases[0].Basis != basis {
				t.Fatal("current terminal ownership lost known order priority or changed unknown history", stage, known)
			}
			raw, err := w.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			var cold World
			if err := cold.UnmarshalBinary(raw); err != nil || cold.Hash() != w.Hash() || !reflect.DeepEqual(cold.RemovedNativeActorBases(), w.RemovedNativeActorBases()) {
				t.Fatal("current terminal order scalar cold form differs", stage, known, err)
			}
		}
	}
}

func TestNativeScalarBlockFollowsActualStrideAndCompletion(t *testing.T) {
	w := mustWorld(t, 17, Bounds{20, 20}, []Entity{{ID: 7, X: 8, Y: 8, HP: 100, MaxHP: 100, Speed: 16, Facing: 96, DesiredFacing: 96, Owner: SelfSlot}})
	basis := NativeActorBasis{BlockPresent: true, BlockKnown: (1 << 10) - 1, Block: [10]byte{30, 31, 166, 166, 171, 172, 173, 174, 175, 176}}
	w.entities[0].NativeBasis = basis
	Step(w, []Command{MoveTo(7, CellPoint{X: 9, Y: 9})})
	stride := NativeStride{Present: true, FromX: 8, FromY: 8, ToX: 9, ToY: 9, Rate: 16, StepX: 11, StepY: 11, Direction: 3}
	for paid := uint16(1); paid <= 24; paid++ {
		e := w.entities[0]
		if e.Stride != stride || e.Transit != 24-paid || e.TransitTotal != 24 {
			t.Fatal("actual rated diagonal stride fixture differs", paid, e.Stride, e.Transit, e.TransitTotal)
		}
		want := basis
		if paid < 24 {
			position := 8*256 + 128 + 11*paid
			want.Block[0], want.Block[1], want.Block[2], want.Block[3] = byte(position>>8), byte(position>>8), byte(position), byte(position)
		} else {
			want.Block[0], want.Block[1], want.Block[2], want.Block[3] = 9, 9, 128, 128
		}
		if NativeBasisNow(e) != want || e.NativeBasis != basis {
			t.Fatal("actual stride changed backing history or failed to compose current Block", paid, NativeBasisNow(e), want)
		}
		actions := w.Actions()
		if actions.Actors[0].Current.NativeBasis == nil || *actions.Actors[0].Current.NativeBasis != want {
			t.Fatal("actual stride action carrier differs", paid)
		}
		raw, err := w.MarshalBinary()
		if err != nil {
			t.Fatal(err)
		}
		var cold World
		if err := cold.UnmarshalBinary(raw); err != nil || cold.entities[0].NativeBasis != want || cold.Hash() != w.Hash() {
			t.Fatal("actual stride scalar cold form differs", paid, err)
		}
		if paid < 24 {
			Step(w, nil)
		}
	}
}

func TestNativeScalarBlockKeepsPartialMasksAndImportedHistory(t *testing.T) {
	e := nativeStrideActorForTest()
	basis := NativeActorBasis{BlockPresent: true, BlockKnown: 1<<0 | 1<<2 | 1<<8, Block: [10]byte{30, 31, 166, 167, 171, 172, 173, 174, 175, 176}}
	e.NativeBasis = basis
	want := basis
	want.Block[0], want.Block[2] = 4, 144
	if NativeBasisNow(e) != want || e.NativeBasis != basis {
		t.Fatal("native stride changed unknown bytes, masks or unmodeled Block tail")
	}
	e.NativeBasis.BlockKnown = 0
	if NativeBasisNow(e) != e.NativeBasis {
		t.Fatal("native stride promoted unknown Block bytes")
	}
	e.NativeBasis = NativeActorBasis{}
	if NativeBasisNow(e).HasValues() {
		t.Fatal("native stride invented an absent Block")
	}
	e.NativeBasis, e.Stride = basis, NativeStride{}
	for _, transit := range []uint16{15, 0} {
		e.Transit = transit
		if NativeBasisNow(e) != basis {
			t.Fatal("actor without native stride replaced imported Block history", transit)
		}
	}
}
