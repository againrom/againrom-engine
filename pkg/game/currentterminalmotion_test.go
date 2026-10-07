package game

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func currentRemovedMotionFront(t *testing.T) *FrontEnd {
	t.Helper()
	f, _ := itemMutationOpen1115(t, 1, false)
	actor := newGroupActors1115(t, f.live.world)[newGroupA]
	f.live.pending = append(f.live.pending, sim.Damage(actor.ID, actor.HP+1000))
	for tick := 0; tick < 100 && slices.ContainsFunc(f.live.world.Entities(), func(e sim.Entity) bool { return e.ID == actor.ID }); tick++ {
		f.live.tick()
	}
	if slices.ContainsFunc(f.live.world.Entities(), func(e sim.Entity) bool { return e.ID == actor.ID }) {
		t.Fatal("actual actor removal did not complete")
	}
	dead := f.live.world.OriginalDeadActors()
	if len(dead) != 1 || dead[0].ID != actor.ID || dead[0].Source.Identity != actor.SourceBinding.Identity || dead[0].Current.Stage != 5 {
		t.Fatalf("source-bound removal lost its exact terminal row: %+v", dead)
	}
	return f
}

func assertCurrentBoundTerminalMotion(t *testing.T, doc sav.DocumentData, a *currentActionData, world *sim.World) {
	t.Helper()
	dead := world.OriginalDeadActors()[0]
	var object uint16
	for _, root := range doc.DeadActors {
		if root == 0 {
			continue
		}
		identity, _ := savedStructureValue(&doc.Objects[root-1], "Identity")
		if identity == dead.Source.Identity {
			object = root
		}
	}
	if object == 0 {
		t.Fatal("terminal source identity has no ordinary dead root")
	}
	stage, _ := savedStructureValue(&doc.Objects[object-1], "Stage")
	if stage != 5 || dead.Current.Stage != 5 {
		t.Fatal("terminal source root changed stage", stage, dead.Current.Stage)
	}
	if len(a.TerminalMotions) != 1 || a.TerminalMotions[0].Entity != dead.ID ||
		a.TerminalMotions[0].Detached != nil || a.TerminalMotions[0].FrozenFine != nil {
		t.Fatalf("retained source motion is not bound to its dead root: %+v", a.TerminalMotions)
	}
	for _, b := range a.Bindings {
		if b.ID == dead.ID && !b.Structure && !b.Missing && b.Object == object {
			return
		}
	}
	t.Fatal("terminal source motion lacks its exact ordinary binding")
}

func TestCurrentBoundTerminalMotionKeepsTwoOrdinaryCyclesAndCellEdits(t *testing.T) {
	for _, edit := range []bool{false, true} {
		name := "unchanged"
		if edit {
			name = "ordinary-ground"
		}
		t.Run(name, func(t *testing.T) {
			f := currentRemovedMotionFront(t)
			source := f.live.world
			var site uint16
			for cycle := 0; cycle < 2; cycle++ {
				if cycle != 0 {
					source = f.live.world
				}
				doc, a := currentRootSAVDocument(t, f)
				assertCurrentBoundTerminalMotion(t, doc, a, f.live.world)
				if len(a.TerminalMotions[0].Slots) != 0 {
					t.Fatal("terminal motion retained typed actor cell sites")
				}
				if cycle == 0 {
					dead := source.OriginalDeadActors()[0]
					foundBody := false
					for _, root := range doc.DeadActors {
						r := &doc.Objects[root-1]
						identity, _ := savedStructureValue(r, "Identity")
						if identity != dead.Source.Identity {
							continue
						}
						position, err := savedMotionRaw(r, "Block12", 12)
						if err != nil {
							t.Fatal(err)
						}
						site, foundBody = binary.LittleEndian.Uint16(position), true
					}
					foundCell := false
					for _, cell := range doc.World.Cells {
						if cell.Cell == site {
							foundCell = true
							if cell.GroundActor != 0 {
								t.Fatal("former terminal body cell still has a ground actor")
							}
						}
					}
					if !foundBody || !foundCell {
						t.Fatal("control lacks the former body's ordinary cell")
					}
				}
				leaf, _, _ := sav.NativeActions(doc.State)
				if edit && cycle == 0 {
					for i := range doc.World.Cells {
						if doc.World.Cells[i].Cell == site {
							doc.World.Cells[i].GroundActor = 0xfefedcba
						}
					}
					unchanged, _, _ := sav.NativeActions(doc.State)
					if !bytes.Equal(leaf, unchanged) {
						t.Fatal("ordinary edit changed motion policy")
					}
				}
				cold := loadCurrentRootSAV(t, doc, func(t *testing.T) *FrontEnd { return itemMutationFront1115(t, 1) })
				if !edit && cold.live.world.Hash() != source.Hash() {
					t.Fatal("cold LOAD changed exact terminal World")
				}
				if edit {
					_, cells, _, _ := cold.live.world.SavedActorMotions()
					found := false
					for _, c := range cells {
						if c.Cell == site {
							found = c.Ground.Key == 0xfefedcba && !c.Ground.Bound
						}
					}
					if !found {
						t.Fatal("ordinary key edit did not supersede the old typed site")
					}
					for _, c := range cold.live.world.SavedCellRecords() {
						if c.Cell == site && (c.Ground.Key != 0xfefedcba || c.Ground.Bound) {
							t.Fatal("record Ground ignored ordinary edit")
						}
					}
					if cycle != 0 && cold.live.world.Hash() != f.live.world.Hash() {
						t.Fatal("edited terminal state changed at second cold LOAD")
					}
				}
				for range 20 {
					sim.Step(source, nil)
					sim.Step(cold.live.world, nil)
					if (!edit || cycle != 0) && source.Hash() != cold.live.world.Hash() {
						t.Fatal("next terminal ticks differ")
					}
				}
				f = cold
			}
		})
	}
}

func TestCurrentBoundTerminalMotionMalformedPolicyIsAtomic(t *testing.T) {
	for _, name := range []string{"unexpected-detached", "duplicate", "empty-issue", "duplicate-object-binding"} {
		t.Run(name, func(t *testing.T) {
			f := currentRemovedMotionFront(t)
			doc, a := currentRootSAVDocument(t, f)
			assertCurrentBoundTerminalMotion(t, doc, a, f.live.world)
			switch name {
			case "unexpected-detached":
				a.TerminalMotions[0].Detached = &currentDetachedMotion{}
			case "duplicate":
				a.TerminalMotions = append(a.TerminalMotions, a.TerminalMotions[0])
			case "empty-issue":
				a.TerminalMotions[0].Issue = ""
			case "duplicate-object-binding":
				var other uint16
				for _, b := range a.Bindings {
					if b.ID != a.TerminalMotions[0].Entity && !b.Missing && !b.Structure {
						other = b.Object
						break
					}
				}
				if other == 0 {
					t.Fatal("control lacks another ordinary actor")
				}
				for i := range a.Bindings {
					if a.Bindings[i].ID == a.TerminalMotions[0].Entity && !a.Bindings[i].Structure {
						a.Bindings[i].Object = other
					}
				}
			}
			leaf, _ := json.Marshal(a)
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			hash := f.live.world.Hash()
			if _, _, err := f.RestoreOriginal(raw); err == nil || f.live.world.Hash() != hash {
				t.Fatal("malformed terminal policy was accepted or changed live World", err)
			}
		})
	}
}
