package game

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestCurrentHumanBasisUsesArithmeticProfile(t *testing.T) {
	p := mapload.PartyMember{Name: "mixed profile", Mage: true, Profile: data.Profile{Fighter: true, HealthColumn: true}, Hero: data.Hero{Body: 20, Reaction: 25, Mind: 30, Spirit: 35}}
	table := &mapload.Table{}
	d, hp, mp := mapload.PartyDisplayWithTable(p, table)
	unit := sav.CityUnitData{Token: make([]byte, 37)}
	h, err := nativeCityHumanFromDerived(p, table, unit, d, hp, mp)
	if err != nil {
		t.Fatal("drawable mage classification changed current arithmetic", err)
	}
	if !h.Fighter || int32(h.HealthMax) != d.HealthMax {
		t.Fatal("current arithmetic profile changed", h.Fighter, h.HealthMax, d.HealthMax)
	}
}

func TestCurrentDocumentCloneReindexesBindingsAfterRootOrderChanges(t *testing.T) {
	doc, binding, _ := actorProjectionFixture(t, "Human")
	player := &doc.Objects[doc.Players[0]-1]
	player.Groups[0], player.Groups[1] = player.Groups[1], player.Groups[0]
	want := doc.Objects[binding.ObjectIndex-1]
	state := &SnapshotSAVDocument{Version: snapshotSAVDocumentVersion, Document: &doc, Actors: []SnapshotSAVActor{binding}}
	if _, err := sav.CloneDocumentData(doc); err == nil {
		t.Fatal("fixture did not change first-encounter order")
	}
	copied, err := cloneSavedDocument(state)
	if err != nil {
		t.Fatal("valid current graph cannot be captured", err)
	}
	got := copied.Document.Objects[copied.Actors[0].ObjectIndex-1]
	if copied.Actors[0].EntityID != binding.EntityID || got.Class != want.Class || !reflect.DeepEqual(got.Values, want.Values) || !reflect.DeepEqual(got.Raw, want.Raw) || !reflect.DeepEqual(got.Texts, want.Texts) {
		t.Fatal("reindex changed actor identity or ordinary values")
	}
	groups := copied.Document.Objects[copied.Document.Players[0]-1].Groups
	first, _ := savedObjectRefs(&groups[0], "Actors")
	second, _ := savedObjectRefs(&groups[1], "Actors")
	if len(first) != 2 || len(second) != 3 || first[1] != copied.Actors[0].ObjectIndex || second[0] != first[1] || second[1] != 0 || second[2] != first[1] {
		t.Fatal("reindex flattened actor aliases or root order", first, second)
	}
	if _, err := sav.EncodeDocumentData(*copied.Document); err != nil {
		t.Fatal("captured graph is not encodable", err)
	}
	if state.Actors[0] != binding || !reflect.DeepEqual(doc.Objects[binding.ObjectIndex-1], want) {
		t.Fatal("capture changed its input")
	}
	player.Groups[0].RefSlots[0].Objects = []uint16{65535}
	if _, err := cloneSavedDocument(state); err == nil {
		t.Fatal("invalid current reference was repaired")
	}
}

func TestCurrentWorldCompletesAnExistingActorGraph(t *testing.T) {
	f := cellStateFront(t)
	campaign := saveCampaign()
	campaign.Side = []int{11, 12}
	f.Campaign = resolved(campaign, nil)
	units := f.Table.Units.(dbCollection)
	f.Table.Units = dbCollection{units[1], units[1]}
	f.Town = NewTown(f.Campaign.Value())
	if err := f.App("current actor completion").OpenMission(f.MissionOpenerWith(10, nil)); err != nil {
		t.Fatal(err)
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	original, err := sim.NewWorld(11, f.live.world.Bounds(), sim.ModeCanonical, nil, []sim.Entity{{ID: 41, X: 15, Y: 15, HP: 29, MaxHP: 29, Owner: sim.SelfSlot, TypeID: 1, Speed: 1, TokenSize: 1}})
	if err != nil {
		t.Fatal(err)
	}
	first, err := f.materializeCurrentWorld(s, original)
	if err != nil {
		t.Fatal(err)
	}
	s.SavedDocument = first
	before, err := sav.EncodeDocumentData(*first.Document)
	if err != nil {
		t.Fatal(err)
	}
	entities := original.Entities()
	id, ok := original.NextEntityID()
	if !ok {
		t.Fatal("fixture exhausted actor identities")
	}
	entities = append(entities, sim.Entity{ID: id, X: 18, Y: 18, HP: 37, MaxHP: 37, Owner: sim.SelfSlot, TypeID: 1, Speed: 1, TokenSize: 1})
	w, err := sim.NewStockedSpelledWorld(11, f.live.world.Bounds(), sim.ModeCanonical, f.live.world.CurrentPolicy().Terrain, entities, nil, original.Relations(), nil, original.Stock(), original.Spells())
	if err != nil {
		t.Fatal(err)
	}
	s.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	after, err := f.materializeCurrentWorld(s, w)
	if err != nil {
		t.Fatal("retained graph blocked a later current actor", err)
	}
	if len(after.Actors) != len(entities) {
		t.Fatal("current actor population incomplete", len(after.Actors), len(entities))
	}
	unchanged, err := sav.EncodeDocumentData(*first.Document)
	if err != nil || !bytes.Equal(before, unchanged) {
		t.Fatal("completion mutated the retained graph", err)
	}
	raw, err := f.ExportCurrentSave(s, "later actor")
	if err != nil {
		t.Fatal("completed current graph cannot SAVE", err)
	}
	cold := cellStateFront(t)
	cold.Campaign = resolved(campaign, nil)
	cold.Table.Units = dbCollection{units[1], units[1]}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("completed actor graph cannot LOAD", town, err)
	}
	if err := cold.App("completed actor LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	if len(cold.live.world.Entities()) != 2 {
		t.Fatal("cold LOAD did not retain current actor population")
	}
	for _, e := range cold.live.world.Entities() {
		if e.X == 18 && e.Y == 18 && e.HP == 37 {
			return
		}
	}
	t.Fatal("cold LOAD lost the later current actor")
}

func TestCurrentActorProjectionHasNoSourceOrWidthAdmission(t *testing.T) {
	doc, binding, initial := actorProjectionFixture(t, "Human")
	e := initial.Entities()[0]
	e.SourceBinding = sim.SourceBinding{}
	e.HP, e.MaxHP = 70000, 80000
	w, err := sim.NewWorld(123, initial.Bounds(), sim.ModeCanonical, nil, []sim.Entity{e})
	if err != nil {
		t.Fatal(err)
	}
	if err := projectSavedActorValues(&doc, []SnapshotSAVActor{binding}, w); err != nil {
		t.Fatal(err)
	}
	health, err := savedStructureValue(&doc.Objects[binding.ObjectIndex-1], "Health")
	if err != nil || health != uint32(uint16(e.HP)) {
		t.Fatal("ordinary word does not carry the current low bits", health, err)
	}
	if len(e.Values().Widths) < 2 {
		t.Fatal("wide current pools have no explicit width residue")
	}
}

func TestCurrentActorTypeProjectsArithmeticAndRuntimeSeparately(t *testing.T) {
	for _, class := range []string{"Unit", "Human"} {
		t.Run(class, func(t *testing.T) {
			doc, binding, initial := actorProjectionFixture(t, class)
			e := initial.Entities()[0]
			e.TypeID, e.ActorLoad.Source.TypeID = 33, 34
			for _, sourceBound := range []bool{true, false} {
				if !sourceBound {
					e.SourceBinding = sim.SourceBinding{}
				}
				r, err := savedActorValueRecord(doc.Objects[binding.ObjectIndex-1], e, false)
				if err != nil {
					t.Fatal(err)
				}
				wire, err := savedStructureValue(&r, "T0E")
				if err != nil || wire != 34 || e.Values().RuntimeType == nil || e.Values().RuntimeType.Value != 33 {
					t.Fatal("arithmetic type or distinct gameplay type lost", sourceBound, wire, e.Values().RuntimeType, err)
				}
			}
			e.ActorLoad.Source = sim.SourceActor{}
			if source := currentActorSource(e, sim.SourceActor{Class: 1, TypeID: 7}); source.TypeID != 33 {
				t.Fatal("constructor replaced current native type", source.TypeID)
			}
		})
	}
}

func TestCurrentTerrainUsesOrdinaryBlocks(t *testing.T) {
	w, err := sim.NewWorld(5, sim.Bounds{Width: 2, Height: 2}, sim.ModeCanonical, []byte{3, 1, 2, 0}, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := sav.DocumentData{World: &sav.DocumentWorldData{Blocks: []sav.BlockRecord{{Cell: 1, Static: 0x20, Dyn: 0x60}}}}
	projectCurrentTerrain(&doc, w, false)
	if len(doc.World.Blocks) != 4 || doc.World.Blocks[0].Static != 3 || doc.World.Blocks[1].Static != 0x21 || doc.World.Blocks[1].Dyn != 0x61 || doc.World.Blocks[2].Cell != 256 || doc.World.Blocks[2].Static != 2 {
		t.Fatal("current native block projection lost values or unmodelled bits", doc.World.Blocks)
	}
}
