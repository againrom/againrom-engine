package sim

import (
	"encoding/json"
	"reflect"
	"slices"
	"testing"
)

func actionCopy(t *testing.T, a ActionContinuations) ActionContinuations {
	t.Helper()
	b, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var v ActionContinuations
	if err = json.Unmarshal(b, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCurrentActionComponentsUseCurrentOperands(t *testing.T) {
	w := scrollFixtureWorld(t, 12, 1)
	w.entities[0].HealthRegenPeriod, w.entities[0].ManaRegenPeriod = 0, 0
	w.entities[0].Withdraw, w.entities[0].Wimpy = 7, 3
	Step(w, []Command{UseScroll(1, 0, 2)})
	w.casts = append(w.casts, scriptCast{FromX: 3, FromY: 4, ToX: 5, ToY: 6, Spell: 1, Power: 17, AtUnit: true, Target: 999})
	a := actionCopy(t, w.Actions())
	if len(a.Scrolls) != 1 || len(a.Scripts) != 1 || a.Actors[0].ActionClock.Known {
		t.Fatal("missing reserved/moving/zero-regen cut")
	}
	data, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var cold World
	if err = cold.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	// Different donor action values cannot win over the explicitly supplied
	// current component. Terrain, actors, items and rules are separate owners.
	for i := range cold.entities {
		e := &cold.entities[i]
		e.X, e.Y = 7, 7
		e.Transit, e.TransitTotal = 0, 0
		e.clearStride()
		e.clearTurn()
		e.ActionClock = ActionClock{Known: true, End: 123}
		e.Withdraw, e.Wimpy = 0, 0
	}
	cold.bookCasts, cold.scrollCasts, cold.casts = nil, nil, nil
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(w.Actions(), cold.Actions()) {
		t.Fatal("current actions used donor operands")
	}
	for i := 0; i < 70; i++ {
		Step(w, nil)
		Step(&cold, nil)
		if w.Hash() != cold.Hash() {
			t.Fatalf("current action continuation differs at %d", i)
		}
	}
}

func TestCurrentActionAbsentMotionDoesNotCreateProvenance(t *testing.T) {
	for _, issue := range []string{"", "explicit superseded source motion"} {
		w := mustWorld(t, 91, Bounds{8, 8}, []Entity{{ID: 1, HP: 10, MaxHP: 10}})
		w.savedMotion = &savedActorMotionState{}
		if issue != "" {
			w.savedMotion.Motions = []SavedActorMotion{{Entity: 1, Issue: issue}}
		}
		a := actionCopy(t, w.Actions())
		cold := *w
		cold.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true}}}
		if err := cold.RestoreActions(a, nil); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(cold.Actions(), a) || cold.Hash() != w.Hash() {
			t.Fatal("ordinary load fabricated motion provenance", issue, cold.Actions())
		}
		if issue == "" && len(cold.savedMotion.Motions) != 0 {
			t.Fatal("absent motion retained an inactive synthetic record")
		}
	}
}

func TestCurrentActionImportedMotionKeepsExplicitIssue(t *testing.T) {
	w := mustWorld(t, 91, Bounds{8, 8}, []Entity{{ID: 1, HP: 10, MaxHP: 10}})
	w.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true, Issue: "original boundary speed callback is not executed"}}}
	a := actionCopy(t, w.Actions())
	cold := *w
	cold.savedMotion = &savedActorMotionState{Motions: []SavedActorMotion{{Entity: 1, Current: true}}}
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cold.Actions(), a) || cold.Hash() != w.Hash() {
		t.Fatal("ordinary motion import erased current explicit issue")
	}
}

func TestCurrentReservedScrollKeepsSharedChildAndRefund(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	r := bindOperationsPack(t, w, 0)
	r.Effects[0].ExternalReferences = 1
	Step(w, []Command{UseScroll(1, 0, 2)})
	a := actionCopy(t, w.Actions())
	if len(a.Reservations.Items) != 1 || len(a.Reservations.Effects) != 1 {
		t.Fatal("bound reservation absent")
	}
	data, _ := w.MarshalBinary()
	var cold World
	if err := cold.UnmarshalBinary(data); err != nil {
		t.Fatal(err)
	}
	cold.scrollCasts = nil
	cold.savedObjects.Items = slices.DeleteFunc(cold.savedObjects.Items, func(v SavedItemObject) bool { return v.ID == a.Reservations.Items[0].ID })
	cold.savedObjects.ItemRoots = slices.DeleteFunc(cold.savedObjects.ItemRoots, func(v SavedItemRoot) bool { return v.Owner.Kind == SavedOwnerSession })
	child := a.Reservations.Effects[0].ID
	if err := cold.RestoreActions(a, map[SavedObjectID]SavedObjectID{child: child}); err != nil {
		t.Fatal(err)
	}
	refund := roundTripSharedItems(t, &cold)
	Step(refund, []Command{GroupStance(1, int32(OrderStandGround), 0)})
	p, _ := refund.CarriedStacks(1)
	if len(refund.ScrollCasts()) != 0 || len(p) != 1 || p[0].Count != 1 || p[0].ObjectID == 0 || refund.savedObjects.Items[len(refund.savedObjects.Items)-1].Effects[0] != child {
		t.Fatal("refund lost exact ownership or shared child")
	}
	for range 20 {
		Step(w, nil)
		Step(&cold, nil)
	}
	if len(cold.ScrollCasts()) != 0 || len(cold.savedObjects.Effects) != 1 || cold.savedObjects.Effects[0].ID != child || cold.savedObjects.Effects[0].Retired || cold.entities[1].HP != w.entities[1].HP {
		t.Fatal("completed reservation lost its surviving shared child")
	}
}

func TestCurrentActionMalformedImportIsAtomic(t *testing.T) {
	w := scrollFixtureWorld(t, 2, 1)
	Step(w, []Command{UseScroll(1, 0, 2)})
	before := w.Hash()
	for _, which := range []string{"actor", "turn", "clock", "scroll", "target"} {
		a := actionCopy(t, w.Actions())
		switch which {
		case "actor":
			a.Actors = append(a.Actors, a.Actors[0])
		case "turn":
			a.Actors[0].TurnRemaining = 255
		case "clock":
			a.Actors[0].ActionClock = ActionClock{End: 1}
		case "scroll":
			a.Scrolls[0].Caster = 999
		case "target":
			a.Actors[0].AttackTargetKind = 255
		}
		if err := w.RestoreActions(a, nil); err == nil {
			t.Fatalf("accepted %s", which)
		}
		if w.Hash() != before {
			t.Fatalf("partial import after %s", which)
		}
	}
}

func restoreActionCut(t *testing.T, w *World) *World {
	t.Helper()
	a := actionCopy(t, w.Actions())
	var cold World
	if err := cold.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	// Deliberately erase the donor's action fields before the component import.
	for i := range cold.entities {
		e := &cold.entities[i]
		e.clearStride()
		e.clearTurn()
		e.Transit, e.TransitTotal = 0, 0
		e.HasTarget = false
		e.AttackCountdown = 0
		e.ActionClock = ActionClock{Known: true, End: 99}
	}
	cold.bookCasts, cold.casts = nil, nil
	if err := cold.RestoreActions(a, nil); err != nil {
		t.Fatal(err)
	}
	return &cold
}

func TestCurrentBookAllPhasesAndTypedEndpoints(t *testing.T) {
	for _, mode := range []string{"unit", "self", "cell", "removed"} {
		t.Run(mode, func(t *testing.T) {
			w := manualCastWorld(t)
			cmd := spCast(1, 2, 1)
			if mode == "self" {
				cmd.X = 1
				cmd.Y = 6
				w.entities[0].KnownSpells = 1 << 6
				w.entities[0].HP = 30
				w.spells = []SpellRule{{ID: 6, ManaCost: 5, MaxRange: 10, TargetsUnit: true, Restorative: true, Defensive: true, DamageMin: 4, DamageMax: 4}}
			}
			if mode == "cell" {
				cmd = CastAt(1, 26, CellPoint{X: 6, Y: 3})
			}
			Step(w, []Command{cmd})
			if mode == "removed" {
				w.bookCasts[0].Target = 999
			}
			phases := map[bookPhase]bool{}
			for sample := 0; sample < 40; sample++ {
				for _, c := range w.bookCasts {
					phases[c.Phase] = true
				}
				cold := restoreActionCut(t, w)
				for n := 0; n < 45; n++ {
					Step(cold, nil)
				}
				control := worldRoundTripForTest(t, w)
				for n := 0; n < 45; n++ {
					Step(control, nil)
				}
				if control.Hash() != cold.Hash() {
					t.Fatalf("%s cut%d changed next cast", mode, sample)
				}
				Step(w, nil)
			}
			if !phases[bookCharging] {
				t.Fatalf("missing actual commanded charge: %v", phases)
			}
		})
	}
	for _, state := range []bookCast{
		{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookCharging, Remaining: 2, Retained: true, Paid: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookRelaxing, Remaining: 2, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryOne, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookBoundaryTwo, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Progress: 1, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Progress: 2, Complete: true, Retained: true},
		{Caster: 1, Target: 2, Spell: 1, Phase: bookPending, Progress: 3, Complete: true, Retained: true},
	} {
		w := cadenceFormWorld(t, state)
		w.entities[0].Mana = 0
		cold := restoreActionCut(t, w)
		if !reflect.DeepEqual(w.Actions(), cold.Actions()) {
			t.Fatal("retained book boundary changed before tick")
		}
		for n := 0; n < 40; n++ {
			if n == 8 {
				w.entities[0].Mana, cold.entities[0].Mana = 30, 30
			}
			a, b := StepObserved(w, nil), StepObserved(cold, nil)
			if !reflect.DeepEqual(a, b) || w.Hash() != cold.Hash() {
				t.Fatalf("retained phase%d progress%d next tick%d differs", state.Phase, state.Progress, n)
			}
		}
	}
	// Colliding integer labels in actor and structure namespaces must not join.
	a := ActionContinuations{Actors: []ActorContinuation{{Entity: 1, AttackTarget: 1, HasAttackTarget: true, AttackTargetKind: AttackTargetStructure, HasEscortTarget: true, EscortTarget: 1}}, Books: []BookContinuation{{Caster: 1, Target: 1}}, Scripts: []ScriptCast{{Target: 1, AtUnit: true}}, StructureUses: []StructureUse{{Entity: 1, Structure: 1}}}
	if err := a.RemapActors(func(id EntityID, s bool) (EntityID, error) {
		if s {
			return id + 100, nil
		}
		return id + 10, nil
	}); err != nil {
		t.Fatal(err)
	}
	if a.Actors[0].Entity != 11 || a.Actors[0].AttackTarget != 101 || a.Actors[0].EscortTarget != 11 || a.Books[0].Target != 11 || a.Scripts[0].Target != 11 || a.StructureUses[0] != (StructureUse{Entity: 11, Structure: 101}) {
		t.Fatal("typed action identities conflated")
	}
}

func TestCurrentActionMotionAndClockCuts(t *testing.T) {
	for _, mode := range []string{"accepted-rate", "turn", "unrated", "offmap", "zero-regen", "first-tick", "wrap"} {
		t.Run(mode, func(t *testing.T) {
			e := nativeStrideActorForTest()
			e.HealthRegenPeriod = 100
			e.HP = 50
			switch mode {
			case "accepted-rate":
				e.Speed = 1 // admitted rate16 remains authoritative
			case "turn":
				e.Transit, e.TransitTotal = 0, 0
				e.clearStride()
				e.RotationSpeed = 32
				e.DesiredFacing, e.TurnRemaining, e.TurnTotal = 192, 3, 4
			case "unrated":
				e.clearStride()
			case "offmap":
				e.Transit, e.TransitTotal = 0, 0
				e.clearStride()
				e.OffMap = true
			case "zero-regen":
				e.Transit, e.TransitTotal = 0, 0
				e.clearStride()
				e.HealthRegenPeriod = 0
			case "first-tick":
				e.Transit, e.TransitTotal = 0, 0
				e.clearStride()
			case "wrap":
				e.Transit, e.TransitTotal = 0, 0
				e.clearStride()
				e.ActionClock = ActionClock{Known: true, End: 0xfffffff0}
			}
			w := mustWorld(t, 1218, Bounds{20, 20}, []Entity{e})
			cold := restoreActionCut(t, w)
			if !reflect.DeepEqual(w.Actions(), cold.Actions()) {
				t.Fatal("component changed pre-tick action")
			}
			for tick := 0; tick < 165; tick++ {
				Step(w, nil)
				Step(cold, nil)
				if w.Hash() != cold.Hash() {
					t.Fatalf("cut differs at%d", tick)
				}
			}
			if mode == "zero-regen" && cold.entities[0].ActionClock.Known {
				t.Fatal("zero regen initialized idle clock")
			}
			if mode == "first-tick" && !cold.entities[0].ActionClock.Known {
				t.Fatal("first tick failed to initialize idle clock")
			}
		})
	}
}
