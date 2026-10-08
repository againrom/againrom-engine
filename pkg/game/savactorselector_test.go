package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestCurrentCanonicalActorSelectorUsesLiteralTokenRow(t *testing.T) {
	for _, class := range []string{"Human", "Unit"} {
		for _, row := range []uint8{0, 1, 0x81} {
			t.Run(fmt.Sprintf("%s/row%d", class, row), func(t *testing.T) {
				doc, binding, initial := actorProjectionFixture(t, class)
				entity := initial.Entities()[0]
				entity.SourceBinding.TokenRow = row
				entity.SourceBinding.Face = row ^ 0x41
				if class == "Human" {
					entity.SourceBinding.TypeID = 33
					entity.ActorLoad.Source.TypeID = 33
					if entity.SourceBinding.DefinitionRow() != 5 {
						t.Fatal("fixture lost the distinct Human definition lookup row")
					}
				}
				retained := uint32(row ^ 0x80)
				if err := savedActorSetValue(&doc.Objects[binding.ObjectIndex-1], "T0C", retained); err != nil {
					t.Fatal(err)
				}
				world, err := sim.NewWorld(123, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{entity})
				if err != nil {
					t.Fatal(err)
				}
				hash := world.Hash()
				if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, world); err != nil {
					t.Fatal(err)
				}
				if got := actorProjectionValue(t, doc.Objects[binding.ObjectIndex-1], "T0C"); got != uint32(row) || world.Hash() != hash {
					t.Fatalf("current literal row%d retained%d wrote%d", row, retained, got)
				}
				if got := actorProjectionValue(t, doc.Objects[binding.ObjectIndex-1], "U4B"); got != uint32(entity.SourceBinding.Face) {
					t.Fatalf("current literal face%d wrote%d", entity.SourceBinding.Face, got)
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
					if actor.Cell == 0x0605 && actor.Class == class {
						matched++
						if file.Body[actor.Off+16] != row || actor.DefRow != row {
							t.Fatal("raw selector/cold literal row lost", file.Body[actor.Off+16], actor.DefRow)
						}
						if file.Body[actor.ControlOff+2] != entity.SourceBinding.Face || actor.Face != entity.SourceBinding.Face {
							t.Fatal("raw portrait selector/cold literal face lost", file.Body[actor.ControlOff+2], actor.Face)
						}
					}
				}
				if matched != 1 {
					t.Fatal("selector exact actor count", matched)
				}
			})
		}
	}
}

func TestAbsentCanonicalSelectorRetainsDocumentValue(t *testing.T) {
	doc, binding, world := actorProjectionFixture(t, "Human")
	entity := world.Entities()[0]
	entity.SourceBinding = sim.SourceBinding{}
	entity.ActorLoad.Source = sim.SourceActor{}
	record := doc.Objects[binding.ObjectIndex-1]
	if err := savedActorSetValue(&record, "T0C", 0x81); err != nil {
		t.Fatal(err)
	}
	if err := savedActorSetValue(&record, "U4B", 0x82); err != nil {
		t.Fatal(err)
	}
	current, err := savedActorValueRecord(record, entity, false)
	if err != nil || actorProjectionValue(t, current, "T0C") != 0x81 || actorProjectionValue(t, current, "U4B") != 0x82 {
		t.Fatal("absent canonical selector synthesized a lookup row", err)
	}
}

func TestCurrentCanonicalClassFlagsKeepIndependentOffMapBit(t *testing.T) {
	for _, class := range []string{"Human", "Unit"} {
		for _, flags := range []uint8{0, 2, 0x82} {
			for _, offMap := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/flags%d/off%t", class, flags, offMap), func(t *testing.T) {
					doc, binding, initial := actorProjectionFixture(t, class)
					entity := initial.Entities()[0]
					entity.SourceBinding.ClassFlags, entity.OffMap = flags, offMap
					record := doc.Objects[binding.ObjectIndex-1]
					if err := savedActorSetValue(&record, "U4C", uint32(flags^0x80)); err != nil {
						t.Fatal(err)
					}
					current, err := savedActorValueRecord(record, entity, false)
					want := uint32(flags) &^ sav.ActorOffMapFlag
					if offMap {
						want |= sav.ActorOffMapFlag
					}
					if err != nil || actorProjectionValue(t, current, "U4C") != want {
						t.Fatal("current literal flags/independent presence", want, err)
					}
					doc.Objects[binding.ObjectIndex-1] = current
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
						if actor.Cell == 0x0605 && actor.Class == class {
							matched++
							if uint32(file.Body[actor.ControlOff+3]) != want || uint32(actor.ClassSelector) != want {
								t.Fatal("raw/cold class flags differ from current", want)
							}
						}
					}
					if matched != 1 {
						t.Fatal("literal flags exact actor count", matched)
					}
					entity.SourceBinding = sim.SourceBinding{}
					entity.ActorLoad.Source = sim.SourceActor{}
					absent, err := savedActorValueRecord(record, entity, false)
					retained := uint32(flags^0x80) &^ sav.ActorOffMapFlag
					if offMap {
						retained |= sav.ActorOffMapFlag
					}
					if err != nil || actorProjectionValue(t, absent, "U4C") != retained {
						t.Fatal("absent canonical flags changed retained class bits", retained, err)
					}
				})
			}
		}
	}
}
