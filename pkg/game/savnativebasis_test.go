package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestNativeBasisKnownBytesReplaceDocumentWithoutStatInverse(t *testing.T) {
	for _, class := range []string{"Unit", "Humanoid", "Human"} {
		t.Run(class, func(t *testing.T) {
			doc, binding, source := actorProjectionFixture(t, class)
			e := source.Entities()[0]
			e.ActorLoad, e.SourceBinding, e.HumanMovement = sim.ActorLoad{}, sim.SourceBinding{}, sim.HumanMovement{}
			base := [24]byte{77, 0, 0x34, 0x12, 41, 0, 42, 0, 43, 0, 44, 0, 45, 0, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
			modifier := [64]byte{3, 4, 5, 6, 17, 0}
			modifier[40], modifier[41], modifier[46], modifier[47], modifier[58] = 91, 92, 93, 94, 95
			e.NativeBasis = (sim.NativeActorBasis{}).WithBase(base).WithModifier(modifier).WithBody(0)
			w, err := sim.NewWorld(123, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{e})
			if err != nil {
				t.Fatal(err)
			}
			if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, w); err != nil {
				t.Fatal(err)
			}
			r := doc.Objects[binding.ObjectIndex-1]
			if got := actorProjectionValue(t, r, "Body"); got != 0 {
				t.Fatalf("present current Body0 retained Document %d", got)
			}
			gotBase, _ := savedActorRaw(&r, "U114", 24)
			for _, at := range []int{0, 2, 4, 14, 22} {
				if binary.LittleEndian.Uint16(gotBase[at:]) != binary.LittleEndian.Uint16(base[at:]) {
					t.Fatalf("base word%d retained/derived: %x want%x", at, gotBase[at:at+2], base[at:at+2])
				}
			}
			gotModifier, _ := savedActorRaw(&r, "UD4", 64)
			for _, at := range []int{0, 4, 40, 46, 58} {
				if gotModifier[at] != modifier[at] {
					t.Fatalf("modifier byte%d=%d want%d", at, gotModifier[at], modifier[at])
				}
			}
		})
	}
}

func TestNativeBasisColdSAVAndRequiredOrdinaryByteLoss(t *testing.T) {
	f := castOrderFront(t)
	var base [24]byte
	base[0], base[2], base[22] = 71, 83, 95
	var modifier [64]byte
	modifier[4], modifier[40], modifier[58] = 7, 99, 17
	basis := (sim.NativeActorBasis{}).WithBase(base).WithModifier(modifier).WithBody(0)
	if err := f.live.world.RestoreNativeActorBases([]sim.NativeActorBasisRecord{{ID: 1, Basis: basis}}); err != nil {
		t.Fatal(err)
	}
	raw, doc, actions := saveCurrentEffect(t, f)
	cold := openCurrentEffectSave(t, f, raw)
	assertCurrentWorldEqual(t, f.live.world, cold.live.world, "native basis cold SAV")
	for tick := 0; tick < 3; tick++ {
		sim.Step(f.live.world, nil)
		sim.Step(cold.live.world, nil)
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatal("native basis continuation differs", tick)
		}
	}
	for _, control := range []struct {
		name string
		edit func(*sav.DocumentRecordData)
		want sim.NativeActorBasis
	}{
		{"Body", func(r *sav.DocumentRecordData) { savedObjectSetValue(r, "Body", 1) }, basis.WithBody(1)},
		{"Base", func(r *sav.DocumentRecordData) { b, _ := savedActorRaw(r, "U114", 24); b[22]++ }, func() sim.NativeActorBasis { b := basis; b.Base[22] = 96; return b }()},
		{"Modifier", func(r *sav.DocumentRecordData) { b, _ := savedActorRaw(r, "UD4", 64); b[40]++ }, func() sim.NativeActorBasis { b := basis; b.Modifier[40] = 100; return b }()},
	} {
		t.Run(control.name, func(t *testing.T) {
			changed, err := sav.CloneDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			for _, b := range actions.Bindings {
				if b.ID == 1 && !b.Structure && !b.Missing {
					control.edit(&changed.Objects[b.Object-1])
				}
			}
			encoded, err := sav.EncodeDocumentData(changed)
			if err != nil {
				t.Fatal(err)
			}
			lost := openCurrentEffectSave(t, f, encoded)
			if got := lost.live.world.Entities()[0].NativeBasis; got != control.want {
				t.Fatalf("ordinary required-field edit: got %+v want %+v", got, control.want)
			}
			_, next, nextActions := saveCurrentEffect(t, lost)
			var actor *sav.DocumentRecordData
			for _, b := range nextActions.Bindings {
				if b.ID == 1 && !b.Structure && !b.Missing {
					actor = &next.Objects[b.Object-1]
				}
			}
			if actor == nil {
				t.Fatal("next SAVE omitted native actor")
			}
			nextBase, err := savedActorRaw(actor, "U114", 24)
			if err != nil {
				t.Fatal(err)
			}
			nextModifier, err := savedActorRaw(actor, "UD4", 64)
			if err != nil {
				t.Fatal(err)
			}
			if actorProjectionValue(t, *actor, "Body") != uint32(control.want.Body) || nextBase[22] != control.want.Base[22] || nextModifier[40] != control.want.Modifier[40] {
				t.Fatal("next SAVE masked edited native basis")
			}
		})
	}
}

func TestNativeBasisWireAnchorRejectsMissingAndExtraRecords(t *testing.T) {
	doc, binding, _ := actorProjectionFixture(t, "Human")
	a := currentActionData{Bindings: []currentActionBinding{{ID: 7, Object: binding.ObjectIndex}}, Actions: sim.ActionContinuations{Actors: []sim.ActorContinuation{{Entity: 7, Current: &sim.ActorCurrentContinuation{NativeBasis: new(sim.NativeActorBasis)}}}}}
	*a.Actions.Actors[0].Current.NativeBasis = (sim.NativeActorBasis{}).WithBody(0)
	if err := matchCurrentNativeBasis(&doc, &a); err == nil {
		t.Fatal("missing raw anchor accepted")
	}
	if err := captureCurrentNativeBasis(&doc, &a); err != nil {
		t.Fatal(err)
	}
	a.NativeBasisWires = append(a.NativeBasisWires, a.NativeBasisWires[0])
	if err := matchCurrentNativeBasis(&doc, &a); err == nil {
		t.Fatal("duplicate raw anchor accepted")
	}
}

func TestNativeBasisHeldWireTracksKnownOrdinaryBytes(t *testing.T) {
	doc, binding, _ := actorProjectionFixture(t, "Human")
	doc.DeadActors = []uint16{binding.ObjectIndex}
	basis := sim.NativeActorBasis{BasePresent: true, BaseKnown: 1 << 22,
		ModifierPresent: true, ModifierKnown: 1 << 40, BodyPresent: true, BodyKnown: true}
	basis.Base[0], basis.Base[22], basis.Modifier[40] = 17, 23, 29
	a := currentActionData{Bindings: []currentActionBinding{{ID: 7, Object: binding.ObjectIndex}},
		Held: []sim.ActorContinuation{{Entity: 7, Current: &sim.ActorCurrentContinuation{NativeBasis: &basis}}}}
	if err := captureCurrentNativeBasis(&doc, &a); err != nil {
		t.Fatal(err)
	}
	if len(a.NativeBasisWires) != 1 || a.NativeBasisWires[0].Entity != 7 {
		t.Fatal("held current basis has no ordinary DeadActors root anchor")
	}
	if err := matchCurrentNativeBasis(&doc, &a); err != nil || basis.Base[22] != 23 || basis.Modifier[40] != 29 || basis.Body != 0 {
		t.Fatal("unchanged held wire replaced independent current history", err)
	}
	r := &doc.Objects[binding.ObjectIndex-1]
	base, _ := savedActorRaw(r, "U114", 24)
	base[0], base[22] = 91, 97
	modifier, _ := savedActorRaw(r, "UD4", 64)
	modifier[40] = 101
	mustSetValue(r, "Body", 3)
	if err := matchCurrentNativeBasis(&doc, &a); err != nil {
		t.Fatal(err)
	}
	if basis.Base[0] != 17 || basis.Base[22] != 97 || basis.Modifier[40] != 101 || basis.Body != 3 {
		t.Fatal("held ordinary edit did not replace exactly the represented known bytes", basis)
	}
	legacy := a
	legacy.HeldNativeBasisWires, legacy.NativeBasisWires = false, nil
	if err := matchCurrentNativeBasis(&doc, &legacy); err != nil {
		t.Fatal("older Held basis without its new anchor marker was refused", err)
	}
	missing := a
	missing.NativeBasisWires = nil
	if err := matchCurrentNativeBasis(&doc, &missing); err == nil {
		t.Fatal("new Held anchor marker admitted absent ordinary wire anchors")
	}
	empty := currentActionData{HeldNativeBasisWires: true}
	if err := matchCurrentNativeBasis(&doc, &empty); err == nil {
		t.Fatal("Held anchor marker without a held basis was accepted")
	}
	a.Actions.Actors = append(a.Actions.Actors, a.Held[0])
	if err := captureCurrentNativeBasis(&doc, &a); err == nil {
		t.Fatal("one actor's basis was accepted in both current and held populations")
	}
}

func TestRemovedNativeBasisRequiresExactOrdinaryDeadRoot(t *testing.T) {
	doc, binding, _ := actorProjectionFixture(t, "Unit")
	mustSetValue(&doc.Objects[binding.ObjectIndex-1], "Identity", 0x51000070)
	doc.DeadActors = []uint16{binding.ObjectIndex}
	basis := (sim.NativeActorBasis{}).WithBody(0)
	a := currentActionData{Bindings: []currentActionBinding{{ID: 7, Object: binding.ObjectIndex}},
		Values:             map[sim.EntityID]sim.ActorValues{7: {CurrentTerminal: &sim.CurrentTerminalActor{ID: 7, Cell: 0x0304, HP: -10001, Stage: 5}}},
		RemovedNativeBases: []sim.NativeActorBasisRecord{{ID: 7, Basis: basis}}}
	if err := captureCurrentNativeBasis(&doc, &a); err != nil {
		t.Fatal(err)
	}
	if err := matchCurrentNativeBasis(&doc, &a); err != nil {
		t.Fatal(err)
	}
	noRoot := doc
	noRoot.DeadActors = nil
	if err := matchCurrentNativeBasis(&noRoot, &a); err == nil {
		t.Fatal("removed basis accepted an ordinary object outside DeadActors")
	}
	missingAnchor := a
	missingAnchor.NativeBasisWires = nil
	if err := matchCurrentNativeBasis(&doc, &missingAnchor); err == nil {
		t.Fatal("removed basis accepted an absent wire anchor")
	}
	repeated := a
	repeated.RemovedNativeBases = append(append([]sim.NativeActorBasisRecord{}, a.RemovedNativeBases...), a.RemovedNativeBases[0])
	if err := matchCurrentNativeBasis(&doc, &repeated); err == nil {
		t.Fatal("removed basis accepted a repeated current owner")
	}
	repeated = a
	repeated.HeldNativeBasisWires = true
	repeated.Held = []sim.ActorContinuation{{Entity: 7, Current: &sim.ActorCurrentContinuation{NativeBasis: &basis}}}
	if err := matchCurrentNativeBasis(&doc, &repeated); err == nil {
		t.Fatal("removed basis accepted a second held payload")
	}
	unbound := a
	unbound.Bindings = []currentActionBinding{{ID: 7, Missing: true}}
	if err := matchCurrentNativeBasis(&doc, &unbound); err == nil {
		t.Fatal("bound ordinary anchor silently became missing")
	}
	if err := captureCurrentNativeBasis(&doc, &unbound); err != nil {
		t.Fatal(err)
	}
	if err := matchCurrentNativeBasis(&doc, &unbound); err != nil {
		t.Fatal("legitimate departed-before-SAV history", err)
	}
	unbound.Values = nil
	if err := matchCurrentNativeBasis(&doc, &unbound); err == nil {
		t.Fatal("missing ordinary anchor has no pure terminal owner")
	}
}
