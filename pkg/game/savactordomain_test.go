package game

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentActorDomainReplacesRetainedScalarThroughExactBinding(t *testing.T) {
	for _, class := range []string{"Human", "Unit"} {
		for _, native := range []bool{false, true} {
			for _, domain := range []sim.Domain{sim.DomainGround, sim.DomainGhost, sim.DomainAir} {
				t.Run(fmt.Sprintf("%s/native%t/domain%d", class, native, domain), func(t *testing.T) {
					doc, binding, initial := actorProjectionFixture(t, class)
					record := &doc.Objects[binding.ObjectIndex-1]
					retained := uint32(domain+1)%3 + 1
					if err := savedActorSetValue(record, "U4A", retained); err != nil {
						t.Fatal(err)
					}
					before, err := sav.CloneDocumentData(doc)
					if err != nil {
						t.Fatal(err)
					}
					entity := initial.Entities()[0]
					entity.Domain = domain
					if native {
						entity.SourceBinding = sim.SourceBinding{}
						entity.ActorLoad.Source = sim.SourceActor{}
					}
					world, err := sim.NewWorld(123, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{entity})
					if err != nil {
						t.Fatal(err)
					}
					hash := world.Hash()
					if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, world); err != nil {
						t.Fatal(err)
					}
					current := &doc.Objects[binding.ObjectIndex-1]
					if got := actorProjectionValue(t, *current, "U4A"); got != uint32(domain)+1 {
						t.Fatalf("retained domain%d replaced live domain%d: wire%d", retained, domain, got)
					}
					mover := bytes.Clone(savedRecordRawForTest(t, *current, "U154"))
					priorMover := savedRecordRawForTest(t, before.Objects[binding.ObjectIndex-1], "U154")
					mover[0], mover[10] = priorMover[0], priorMover[10]
					if hash != world.Hash() || !bytes.Equal(mover, priorMover) {
						t.Fatal("domain scalar changed independent current World or mover")
					}
					for i := range doc.Objects {
						if i != int(binding.ObjectIndex)-1 && !reflect.DeepEqual(doc.Objects[i], before.Objects[i]) {
							t.Fatal("domain projector changed an unbound record", i)
						}
					}
					raw, err := sav.EncodeDocumentData(doc)
					if err != nil {
						t.Fatal(err)
					}
					file, err := sav.Open(raw)
					if err != nil {
						t.Fatal(err)
					}
					graph, err := file.ActorGraph()
					if err != nil {
						t.Fatal(err)
					}
					matched := 0
					for _, actor := range graph.Actors {
						if actor.Cell != 0x0605 || actor.Class != class {
							continue
						}
						matched++
						wire := file.Body[actor.ControlOff+1]
						cold, err := mapload.SourceActorDomain(wire)
						if err != nil || cold != domain || wire != uint8(domain)+1 {
							t.Fatal("raw byte/cold domain differs from current", wire, cold, err)
						}
					}
					if matched != 1 {
						t.Fatal("exact bound fixture actor count", matched)
					}
				})
			}
		}
	}
}
