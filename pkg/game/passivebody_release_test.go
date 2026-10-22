package game

import (
	"os"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// deadRootContents reads the worn and carried items of every dead root a SAV
// writes, in root order; a root without items is an empty row.
func deadRootContents(t *testing.T, raw []byte) []sim.PassiveBody {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := make([]sim.PassiveBody, 0, len(doc.DeadActors))
	for _, root := range doc.DeadActors {
		out = append(out, passiveRecordContents(&doc, root, true))
	}
	return out
}

func countPassiveRoots(rows []sim.PassiveBody, code uint16) int {
	n := 0
	for _, row := range rows {
		if row.Worn[0].Code == code {
			n++
		}
	}
	return n
}

// passiveSaveReload writes an ordinary SAV and opens it in a fresh front end.
// The state of play is the saved one, and the passive contents equal it in
// items.
func passiveSaveReload(t *testing.T, f *FrontEnd, label string) ([]byte, *FrontEnd) {
	t.Helper()
	before, _ := f.LiveWorld()
	raw, err := os.ReadFile(saveCorpseMission(t, f, t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	next := releaseFront(t)
	open, town, err := next.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("%s: LOAD: town=%t %v", label, town, err)
	}
	if err := next.App(label).OpenMission(open); err != nil {
		t.Fatalf("%s: %v", label, err)
	}
	after, _ := next.LiveWorld()
	if after.Hash() != before.Hash() || after.Tick() != before.Tick() || !reflect.DeepEqual(after.CurrentTerminalActors(), before.CurrentTerminalActors()) {
		logCurrentCarrierDiff(t, before, after)
		t.Fatalf("%s: LOAD hash %016x tick %d, saved %016x tick %d", label, after.Hash(), after.Tick(), before.Hash(), before.Tick())
	}
	if len(after.Entities()) != len(before.Entities()) || after.Purse(sim.SelfSlot) != before.Purse(sim.SelfSlot) || !reflect.DeepEqual(after.Sacks(), before.Sacks()) {
		t.Fatalf("%s: passive contents became live entities, gold or loot", label)
	}
	want, got := before.PassiveBodies(), after.PassiveBodies()
	if len(got) != len(want) {
		t.Fatalf("%s: passive bodies %d, saved %d", label, len(got), len(want))
	}
	for i := range want {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("%s: passive body %d after LOAD %+v body %+v, saved %+v body %+v", label, want[i].ID, got[i], got[i].Body, want[i], want[i].Body)
		}
	}
	return raw, next
}

// A departed actor without a map unit or a party entry keeps its body and
// the items it still wore when it left: SAVE writes them on its dead record,
// a cold LOAD keeps them passive, and the next SAVE writes the same items.
func TestReleaseDepartedUnplacedActorContentsSurviveSAV(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	if err := f.App("departed contents").OpenMission(f.MissionOpenerWith(20, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	// No shipped ground actor departs with items: death drops every worn and
	// carried item except an innate weapon, and in the installed missions only
	// flying movers, which leave no body, hold one. The witness is therefore
	// constructed: a ground native actor of the map without its map unit, given
	// a flyer's innate weapon, and after its loot has dropped one stack from a
	// hero's pack.
	var model sim.Entity
	var worn [sim.EquipSlots]sim.ItemInstance
	for _, e := range w.Entities() {
		held, _ := w.EquippedItems(e.ID)
		switch {
		case worn[0].Empty() && held[0].InnateWeapon():
			worn[0] = held[0]
		case model.MapUnitID == 0 && e.Alive() && e.Owner != sim.SelfSlot && e.MapUnitID != 0 && e.SourceBinding.Class == 0 &&
			e.Domain == sim.DomainGround && !e.Humanoid:
			model = e
		}
	}
	var pack sim.ItemStack
	for _, member := range f.live.mission.ids {
		if carried, _ := w.CarriedStacks(member); pack.Code == 0 && len(carried) != 0 {
			pack = carried[len(carried)-1]
			pack.ObjectID, pack.NativeRecord = 0, nil
		}
	}
	if model.MapUnitID == 0 || worn[0].Empty() || pack.Code == 0 {
		t.Fatal("mission 20 has no ground native actor, innate weapon or hero pack item")
	}
	worn[0].ObjectID, worn[0].NativeRecord = 0, nil
	template := model
	template.MapUnitID = 0
	template.ActorLoad.Source.Class = 0
	id, err := w.CheatSummon(template, nil, worn, model.X, model.Y)
	if err != nil {
		t.Fatal(err)
	}
	if e, ok := w.Entity(id); !ok || e.MapUnitID != 0 || e.SourceBinding.Class != 0 {
		t.Fatalf("witness actor %d is not an unplaced native actor: %+v", id, e)
	}
	for _, member := range f.live.mission.ids {
		if member == id {
			t.Fatal("witness actor is a party member")
		}
	}
	f.LiveKill(uint32(id))
	for done := 0; ; done += 10 {
		if e, ok := f.live.world.Entity(id); ok && e.Decay >= sim.DecayBones {
			break
		}
		if done > 24000 {
			t.Fatal("witness body did not reach bones")
		}
		f.LiveAdvance(10)
	}
	if !f.live.world.CheatAddItem(id, pack) {
		t.Fatal("witness body refused the pack item")
	}
	killUntilTerminal(t, f, id, 24000)
	w = f.live.world
	for _, row := range w.CurrentTerminalActors() {
		if row.ID == id && (row.LeavesNoBody() || row.MapUnitID != 0) {
			t.Fatalf("witness terminal row %+v", row)
		}
	}
	kept := w.PassiveBodies()
	held := false
	for _, b := range kept {
		held = held || b.ID == id && b.Body != nil && b.Body.TypeID == model.TypeID && b.Worn[0].Code == worn[0].Code &&
			len(b.Carried) == 1 && b.Carried[0].Code == pack.Code && b.Carried[0].Count == pack.Count
	}
	if !held {
		t.Fatalf("departure kept no passive body for %d: %+v", id, kept)
	}
	first, cold := passiveSaveReload(t, f, "departed contents first SAV")
	second, again := passiveSaveReload(t, cold, "departed contents second SAV")
	written := deadRootContents(t, first)
	if n := countPassiveRoots(written, worn[0].Code); n == 0 {
		t.Fatalf("first SAV wrote no dead record with the departed actor's weapon %#x", worn[0].Code)
	}
	if !reflect.DeepEqual(deadRootContents(t, second), written) {
		t.Fatal("second SAV changed the dead records' items")
	}
	if !reflect.DeepEqual(again.live.world.PassiveBodies(), cold.live.world.PassiveBodies()) {
		t.Fatal("second cold LOAD changed the passive bodies")
	}
	t.Logf("departed native %d type %d weapon %#x; dead roots %d, with this weapon %d", id, model.TypeID, worn[0].Code, len(written), countPassiveRoots(written, worn[0].Code))
}
