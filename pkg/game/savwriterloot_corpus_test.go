//go:build sessioncorpusaudit

package game

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type lootObservation struct {
	Checkpoint                    string
	Revision                      string
	Purse                         uint32
	Actors                        []lootActorObservation
	Sacks                         []sim.Sack
	ItemWeights                   []sim.ItemWeight
	Objects                       *sim.SavedObjects
	ConstructionInputs            map[string]lootTableInput
	ConstructionTablePresent      bool
	EarlyConstructionInputs       map[string]lootTableInput
	EarlyConstructionRoutePresent bool
	EarlyConstructionTablePresent bool
	ConstructionTableRoutesEqual  bool
	EffectSelections              []lootEffectSelection
	SourceCapture                 writerSourceReference
}

type lootEffectSelection struct {
	Owner                  sim.SavedObjectOwner
	Cell                   [2]int32
	ItemIndex, EffectIndex int
	ItemObjectID           sim.SavedObjectID
	EffectObjectID         sim.SavedObjectID
	Branch, Policy         string
	Value                  sim.ItemEffect
	RegistryItemPresent    bool
	RegistryEffect         *sim.SavedEffectObject
}

type lootActorObservation struct {
	Entity sim.Entity
	Pack   []sim.ItemStack
	Worn   [sim.EquipSlots]sim.ItemInstance
}

func observeWriterLoot(t *testing.T, f *FrontEnd, checkpoint string) lootObservation {
	t.Helper()
	o := lootObservation{Checkpoint: checkpoint, Revision: os.Getenv("AGAINROM_WITNESS_REVISION"), Purse: f.live.world.Purse(sim.SelfSlot), Sacks: f.live.world.Sacks(), ItemWeights: f.live.world.ItemWeights()}
	o.Objects = f.live.world.SavedObjects()
	captured, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	o.SourceCapture = writeWriterSourceCapture(t, os.Getenv("AGAINROM_LOOT_WITNESS_OUT"), "source-"+checkpoint+".json", observeWriterSources(f, captured, checkpoint))
	o.ConstructionTablePresent, o.ConstructionInputs = observeLootConstruction(f.Table)
	if f.live.mission != nil && f.live.mission.state != nil {
		o.EarlyConstructionRoutePresent = true
		table := f.live.mission.state.Start.ConstructionTable
		o.EarlyConstructionTablePresent, o.EarlyConstructionInputs = observeLootConstruction(table)
		o.ConstructionTableRoutesEqual = table == f.Table
	}
	for _, e := range f.live.world.Entities() {
		pack, _ := f.live.world.CarriedStacks(e.ID)
		worn, _ := f.live.world.EquippedItems(e.ID)
		o.Actors = append(o.Actors, lootActorObservation{e, pack, worn})
	}
	addEffects := func(owner sim.SavedObjectOwner, cell [2]int32, itemIndex int, item sim.ItemInstance) {
		for index, value := range item.Effects {
			selection := lootEffectSelection{Owner: owner, Cell: cell, ItemIndex: itemIndex, EffectIndex: index, ItemObjectID: item.ObjectID, Value: value,
				Branch: "unkeyed constructor", Policy: "generatedDocumentBuilder.item -> sim.ConstructSavedItem; E0C initialization is engine policy; Token allocator is separate"}
			if item.ObjectID != 0 {
				selection.Branch, selection.Policy = "keyed registry", "projectCurrentItemGraphMode -> savedCurrentEffectRecord; exact ordered Item.Effects edge"
				if o.Objects != nil {
					if row, present := o.Objects.Item(item.ObjectID); present {
						selection.RegistryItemPresent = true
						if index < len(row.Effects) {
							selection.EffectObjectID = row.Effects[index]
							for _, effect := range o.Objects.Effects {
								if effect.ID == selection.EffectObjectID {
									copy := effect
									selection.RegistryEffect = &copy
									break
								}
							}
						}
					}
				}
			}
			o.EffectSelections = append(o.EffectSelections, selection)
		}
	}
	for _, actor := range o.Actors {
		for index, stack := range actor.Pack {
			addEffects(sim.SavedObjectOwner{Kind: sim.SavedOwnerActorPack, Entity: actor.Entity.ID}, [2]int32{}, index, stack.Instance())
		}
		for slot, item := range actor.Worn {
			addEffects(sim.SavedObjectOwner{Kind: sim.SavedOwnerActorWorn, Entity: actor.Entity.ID, Slot: uint32(slot + 1)}, [2]int32{}, slot, item)
		}
	}
	for _, sack := range o.Sacks {
		for index, item := range currentSackItemInstances(sack) {
			addEffects(sim.SavedObjectOwner{Kind: sim.SavedOwnerSack, Object: sack.ObjectID}, [2]int32{sack.X, sack.Y}, index, item)
		}
	}
	return o
}

func TestLootConstructionCapturePreservesAbsenceNamesAndInputOwnership(t *testing.T) {
	if present, inputs := observeLootConstruction(nil); present || inputs != nil {
		t.Fatal("absent constructor table was fabricated")
	}
	params := []int32{17, 23}
	doubles := []float64{1.5, 2.5}
	table := &mapload.Table{Weapons: &fixtureCollection{names: []string{"named weapon"}, params: [][]int32{params}},
		Shapes: fixtureScale{names: []string{"named shape"}, doubles: [][]float64{doubles}}}
	present, inputs := observeLootConstruction(table)
	if !present || inputs["Armors"].Present || !inputs["Weapons"].Present || !inputs["Shapes"].Present ||
		inputs["Weapons"].Rows[0].Name != "named weapon" || inputs["Shapes"].Rows[0].Name != "named shape" {
		t.Fatal("effective table presence or resolver names lost", inputs)
	}
	params[0], doubles[0] = 99, 99
	if inputs["Weapons"].Rows[0].Params[0] != 17 || inputs["Shapes"].Rows[0].Doubles[0] != 1.5 {
		t.Fatal("captured input aliases a later table mutation")
	}
}

func lootHoldings(f *FrontEnd) map[string]uint64 {
	counts := map[string]uint64{}
	add := func(item sim.ItemInstance, count uint32) {
		if item.Empty() {
			return
		}
		if len(item.Effects) == 0 {
			item.Effects = nil
		}
		// A merge retains its destination object (ITEM-MERGE-129). This count
		// checks quantities and values; the pickup check below follows that join.
		item.ObjectID = 0
		key, _ := json.Marshal(item)
		counts[string(key)] += uint64(count)
	}
	for _, e := range f.live.world.Entities() {
		if !e.Alive() {
			continue
		}
		pack, _ := f.live.world.CarriedStacks(e.ID)
		worn, _ := f.live.world.EquippedItems(e.ID)
		for _, item := range pack {
			add(item.Instance(), item.Count)
		}
		for _, item := range worn {
			add(item, 1)
		}
	}
	for _, sack := range f.live.world.Sacks() {
		for _, item := range currentSackItemInstances(sack) {
			add(item, 1)
		}
	}
	return counts
}

func TestSAVWriterLootContinuationWitness(t *testing.T) {
	dir := os.Getenv("AGAINROM_LOOT_WITNESS_OUT")
	if dir == "" {
		t.Skip("no explicit loot witness output directory")
	}
	const corpusInput = "2026-10-07/saveorcsdontgo.sav"
	f := censusMissionFront(t, corpusInput)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, raw []byte) {
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	observations := []lootObservation{observeWriterLoot(t, f, "loaded-before-mutation")}
	before, err := censusMissionSave(t, f)
	if err != nil {
		t.Fatal(err)
	}
	write("before.sav", before)
	var changes []string
	defer func() {
		data, marshalErr := json.MarshalIndent(struct {
			Revision     string
			KnowledgePin string
			Changes      []string
			Observations []lootObservation
		}{os.Getenv("AGAINROM_WITNESS_REVISION"), os.Getenv("AGAINROM_WITNESS_KNOWLEDGE"), changes, observations}, "", "  ")
		if marshalErr != nil {
			t.Error(marshalErr)
			return
		}
		write("observations.json", data)
	}()
	w := writerCensusChange(f, [2]int32{}, false)
	changes = w.changes
	observations = append(observations, observeWriterLoot(t, f, "changed-before-save"))
	raw, _, err := writerCensusSave(f, true, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	write("changed.sav", raw)
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	constructorFound := false
	for _, index := range doc.World.Sacks {
		r := &doc.Objects[index-1]
		position, err := savedActorRaw(r, "Block12", 12)
		if err != nil || position[2] != 70 || position[3] != 41 {
			continue
		}
		refs, _ := savedObjectRefs(r, "Contents")
		for _, ref := range refs {
			item := &doc.Objects[ref-1]
			code, _ := savedStructureValue(item, "F40")
			if code != 33045 {
				continue
			}
			constructorFound = true
			material, _ := savedStructureValue(item, "F46")
			operand, _ := savedStructureValue(item, "F48")
			if material != 8 || operand != 20 {
				t.Errorf("death loot constructor Material/F48 %d/%d, want 8/20", material, operand)
			}
		}
	}
	if !constructorFound {
		t.Fatal("death loot at 70,41 absent")
	}
	cold, _ := coldMission(t, raw)
	observations = append(observations, observeWriterLoot(t, cold, "cold-load"))
	coldRaw, err := censusMissionSave(t, cold)
	if err != nil {
		t.Fatal(err)
	}
	write("cold-resave.sav", coldRaw)
	if !reflect.DeepEqual(lootHoldings(f), lootHoldings(cold)) {
		t.Fatal("cold LOAD lost or duplicated current holdings")
	}
	for _, sack := range cold.live.world.Sacks() {
		for _, item := range sack.ItemInstances {
			if item.Code == 33045 {
				t.Logf("cold loot cell %d,%d: %+v", sack.X, sack.Y, item)
			}
		}
	}
	id, partyIndex := sim.EntityID(0), -1
	var bestHP int32
	for i, candidate := range cold.live.mission.ids {
		if e, ok := cold.live.world.Entity(candidate); ok && e.Alive() && e.HP > bestHP {
			id, partyIndex, bestHP = candidate, i, e.HP
		}
	}
	if partyIndex < 0 {
		t.Fatal("no living party member for next pickup")
	}
	counts := lootHoldings(cold)
	beforePack, _ := cold.live.world.CarriedStacks(id)
	var incoming sim.ItemInstance
	for _, sack := range cold.live.world.Sacks() {
		if sack.X == 71 && sack.Y == 33 {
			items := currentSackItemInstances(sack)
			if len(items) != 1 {
				t.Fatal("target Sack is not a single-item loss control")
			}
			incoming = items[0]
		}
	}
	if incoming.Empty() {
		t.Fatal("target Sack absent before pickup")
	}
	for _, cell := range [][2]int32{{71, 33}} {
		if err := cold.live.world.HeadlessPlace(id, cell[0], cell[1]); err != nil {
			t.Fatal(err)
		}
		cold.live.orderPickup(id, cell[0], cell[1])
		found := true
		for tick := 0; tick < 240 && found; tick++ {
			cold.LiveAdvance(1)
			found = false
			for _, sack := range cold.live.world.Sacks() {
				found = found || sack.X == cell[0] && sack.Y == cell[1]
			}
		}
		if found {
			e, _ := cold.live.world.Entity(id)
			x, y, present := cold.live.world.ActorFinePosition(id)
			t.Fatalf("cold pickup did not consume Sack at %v: actor %+v fine %d,%d/%v progress %d", cell, e, x, y, present, cold.live.world.ActorOrderProgress(id))
		}
	}
	observations = append(observations, observeWriterLoot(t, cold, "after-target-pickup"))
	if !reflect.DeepEqual(counts, lootHoldings(cold)) {
		after := lootHoldings(cold)
		for key, count := range counts {
			if after[key] != count {
				t.Logf("holding before %d after %d: %s", count, after[key], key)
			}
		}
		for key, count := range after {
			if counts[key] != count {
				t.Logf("holding after %d before %d: %s", count, counts[key], key)
			}
		}
		t.Fatal("pickup lost or duplicated current holdings")
	}
	pack, _ := cold.live.world.CarriedStacks(id)
	equipped := false
	for i, item := range pack {
		value := item.Instance()
		value.ObjectID = incoming.ObjectID
		if !sim.StackStateEqual(sim.StackItem(value, 1), sim.StackItem(incoming, 1)) {
			continue
		}
		var previous uint32
		for _, old := range beforePack {
			if sim.StackStateEqual(sim.StackItem(old.Instance(), 1), sim.StackItem(item.Instance(), 1)) {
				previous += old.Count
			}
		}
		if item.Count != previous+1 {
			continue
		}
		cold.live.switchInventorySubject(uint32(id))
		cold.live.enqueueEquip(i)
		cold.LiveAdvance(1)
		worn, _ := cold.live.world.EquippedItems(id)
		value = worn[0]
		value.ObjectID = item.ObjectID
		equipped = sim.StackStateEqual(sim.StackItem(value, 1), sim.StackItem(item.Instance(), 1))
		if !equipped {
			t.Logf("equip selected actor %d item %+v worn %+v", id, item, worn[0])
		}
		break
	}
	observations = append(observations, observeWriterLoot(t, cold, "after-weapon-equip"))
	if !equipped || !reflect.DeepEqual(counts, lootHoldings(cold)) {
		t.Fatal("next equip changed the weapon or holdings")
	}
	picked, err := censusMissionSave(t, cold)
	if err != nil {
		t.Fatal(err)
	}
	write("picked-equipped.sav", picked)
	next := releaseFront(t)
	next.SetDeterministicFrames(true)
	open, _, err := next.RestoreOriginal(picked)
	if err != nil || open == nil {
		t.Fatal(err)
	}
	app := next.App("loot-cold-next")
	app.Layout(1024, 768)
	if err := app.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	nextID := next.live.mission.ids[partyIndex]
	next.LiveAdvance(2)
	worn, _ := next.live.world.EquippedItems(nextID)
	if worn[0].Code != 33045 || !reflect.DeepEqual(counts, lootHoldings(next)) {
		t.Fatal("second cold LOAD resurrected loot or lost the equipped weapon")
	}
	observations = append(observations, observeWriterLoot(t, next, "second-cold-load-next-ticks"))
	finalRaw, err := censusMissionSave(t, next)
	if err != nil {
		t.Fatal(err)
	}
	write("cold-next-ticks.sav", finalRaw)
	t.Logf("witness preserved %s, changes %v", dir, w.changes)
}
