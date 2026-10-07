package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// SAV may remint object IDs; aliases and null references must stay exact.
func canonicalSavedObjectIDs(v reflect.Value) {
	type reference struct {
		field reflect.Value
		id    sim.SavedObjectID
	}
	var references []reference
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if f := v.Field(i); f.CanSet() {
					visit(f)
				}
			}
		case reflect.Slice, reflect.Array:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		default:
			if v.CanSet() && v.Type() == reflect.TypeOf(sim.SavedObjectID(0)) {
				references = append(references, reference{v, sim.SavedObjectID(v.Uint())})
			}
		}
	}
	visit(v)
	ids := map[sim.SavedObjectID]sim.SavedObjectID{0: 0}
	for _, r := range references {
		next, found := ids[r.id]
		if !found {
			next = sim.SavedObjectID(len(ids))
			ids[r.id] = next
		}
		r.field.SetUint(uint64(next))
	}
}

func TestFixtureObjectIdentityComparisonPreservesAliases(t *testing.T) {
	party := func(ids [4]sim.SavedObjectID) []mapload.PartyMember {
		p := make([]mapload.PartyMember, 2)
		p[0].WornItems[0].ObjectID = ids[0]
		p[0].Carry = &mapload.Carry{ItemInstances: []sim.ItemInstance{{ObjectID: ids[1]}}}
		p[1].WornItems[0].ObjectID = ids[2]
		p[1].Carry = &mapload.Carry{ItemInstances: []sim.ItemInstance{{ObjectID: ids[3]}}}
		canonicalSavedObjectIDs(reflect.ValueOf(p))
		return p
	}
	want := party([4]sim.SavedObjectID{0, 42, 42, 17})
	for _, test := range []struct {
		name  string
		ids   [4]sim.SavedObjectID
		equal bool
	}{
		{"renumbered", [4]sim.SavedObjectID{0, 7, 7, 8}, true},
		{"merged objects", [4]sim.SavedObjectID{0, 7, 7, 7}, false},
		{"split alias", [4]sim.SavedObjectID{0, 7, 8, 9}, false},
		{"materialized null", [4]sim.SavedObjectID{6, 7, 7, 8}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := reflect.DeepEqual(want, party(test.ids)); got != test.equal {
				t.Fatalf("identity correspondence equal=%t, want %t", got, test.equal)
			}
		})
	}
	id := sim.SavedObjectID(42)
	shared := []*sim.SavedObjectID{&id, &id}
	canonicalSavedObjectIDs(reflect.ValueOf(shared))
	if id != 1 {
		t.Fatal("shared pointer was canonicalized twice", id)
	}
}

func fixturePartyAtInventoryBoundary(t *testing.T, snapshot game.Snapshot, world *sim.World, sourceTemplate bool) ([]mapload.PartyMember, int, int) {
	t.Helper()
	party := mapload.CloneParty(snapshot.Party)
	if len(party) != len(snapshot.CurrentPartyIDs) {
		t.Fatal("inventory boundary lacks exact party actor bindings")
	}
	materialized := 0
	objects := map[sim.SavedObjectID]bool{}
	bind := func(path string, template, live sim.ItemInstance) sim.ItemInstance {
		t.Helper()
		if live.ObjectID != 0 {
			objects[live.ObjectID] = true
		}
		if template.ObjectID != live.ObjectID {
			if !sourceTemplate || template.ObjectID != 0 || live.ObjectID == 0 {
				t.Fatalf("%s: snapshot handle %d differs from live inventory %d", path, template.ObjectID, live.ObjectID)
			}
			materialized++
			template.ObjectID = live.ObjectID
		}
		if !reflect.DeepEqual(template, live) {
			t.Fatalf("%s: template item values differ from live inventory: %+v / %+v", path, template, live)
		}
		return template
	}
	for i := range party {
		member := &party[i]
		worn, wornOK := world.EquippedItems(snapshot.CurrentPartyIDs[i])
		items, itemsOK := world.CarriedItems(snapshot.CurrentPartyIDs[i])
		stacks, stacksOK := world.CarriedStacks(snapshot.CurrentPartyIDs[i])
		if !wornOK || !itemsOK || !stacksOK || member.Carry == nil ||
			len(member.CarriedItems) != len(items) || len(member.Carry.ItemInstances) != len(items) || len(member.Carry.OrderedStacks) != len(stacks) {
			t.Fatalf("member %s inventory shape differs from its live actor", member.ID)
		}
		for j, item := range worn {
			member.WornItems[j] = bind(fmt.Sprintf("party[%d].WornItems[%d]", i, j), member.WornItems[j], item)
			member.Carry.EquippedItems[j] = bind(fmt.Sprintf("party[%d].Carry.EquippedItems[%d]", i, j), member.Carry.EquippedItems[j], item)
		}
		for j, item := range items {
			member.CarriedItems[j] = bind(fmt.Sprintf("party[%d].CarriedItems[%d]", i, j), member.CarriedItems[j], item)
			member.Carry.ItemInstances[j] = bind(fmt.Sprintf("party[%d].Carry.ItemInstances[%d]", i, j), member.Carry.ItemInstances[j], item)
		}
		for j, stack := range stacks {
			cached := &member.Carry.OrderedStacks[j]
			if cached.Count != stack.Count {
				t.Fatalf("party[%d].Carry.OrderedStacks[%d] count differs: %d / %d", i, j, cached.Count, stack.Count)
			}
			cached.ObjectID = bind(fmt.Sprintf("party[%d].Carry.OrderedStacks[%d]", i, j), cached.Instance(), stack.Instance()).ObjectID
		}
	}
	return party, materialized, len(objects)
}

// decodeFixture observes the state installed by a fresh SAV load.
func decodeFixture(t *testing.T, assets string, raw []byte) (game.Snapshot, string) {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatalf("fixture does not open as SAV: %v", err)
	}
	f, err := game.NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil {
		t.Fatalf("fixture does not restore: %v", err)
	}
	if !town {
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			t.Fatalf("fixture opener: %v", err)
		}
	}
	snap, _, err := f.Snapshot(!town)
	if err != nil {
		t.Fatalf("fixture snapshot: %v", err)
	}
	return snap, string(file.Label)
}

func TestEndpointFixtureMovesOnlyTheEscortAndDoesNotDecideOrStep(t *testing.T) {
	entities := []sim.Entity{
		{ID: 1, MapUnitID: 136, X: 3, Y: 4, HP: 10, MaxHP: 10},
		{ID: 2, MapUnitID: 20, X: 5, Y: 6, HP: 20, MaxHP: 20},
	}
	w, err := sim.NewWorld(1, sim.Bounds{Width: 144, Height: 144}, sim.ModeCanonical, nil, entities)
	if err != nil {
		t.Fatal(err)
	}
	before := w.Entities()
	if _, err := placeEscort(w); err != nil {
		t.Fatal(err)
	}
	after := w.Entities()
	if w.Tick() != 0 || w.Outcome() != sim.OutcomeUndecided || !reflect.DeepEqual(before[1], after[1]) {
		t.Fatal("fixture advanced/decided the world or changed another actor")
	}
	if after[0].X < 110 || after[0].X > 113 || after[0].Y < 131 || after[0].Y > 134 || after[0].HP != before[0].HP {
		t.Fatalf("escort=%+v", after[0])
	}
	missing, err := sim.NewWorld(1, sim.Bounds{Width: 144, Height: 144}, sim.ModeCanonical, nil, entities[1:])
	if err != nil {
		t.Fatal(err)
	}
	if _, err := placeEscort(missing); err == nil {
		t.Fatal("missing escort accepted")
	}
}

func TestEndpointFixtureRefusesInputsAndExistingOutput(t *testing.T) {
	base := t.TempDir()
	assets, sourceDir, out := filepath.Join(base, "install"), filepath.Join(base, "source"), filepath.Join(base, "out")
	for _, path := range []string{assets, sourceDir, out, filepath.Join(assets, "child"), filepath.Join(sourceDir, "child")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	source := filepath.Join(sourceDir, "source.sav")
	if err := os.WriteFile(source, []byte("not the owner's save"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{assets, sourceDir, filepath.Join(assets, "child"), filepath.Join(sourceDir, "child")} {
		if err := validateOutput(assets, source, bad); err == nil {
			t.Errorf("protected output accepted: %s", bad)
		}
	}
	if err := prepare(assets, source, out, io.Discard); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Fatalf("wrong source accepted: %v", err)
	}
	if err := os.WriteFile(filepath.Join(out, "keep.txt"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := validateOutput(assets, source, out); err == nil {
		t.Fatal("nonempty output accepted")
	}
	if got, _ := os.ReadFile(filepath.Join(out, "keep.txt")); string(got) != "keep" {
		t.Fatal("existing output changed")
	}
	var stderr bytes.Buffer
	if code := run(nil, io.Discard, &stderr); code != 2 {
		t.Fatalf("missing arguments exit=%d", code)
	}
}

// This is a fresh installed mission with save666's imported party/purse, not
// original-world continuation. The exact shipped JSON files drive the App.
func TestReleaseLegacyMission20EndpointScenarios(t *testing.T) {
	assets, source := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_SAVE_666")
	if assets == "" || source == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_SAVE_666 are required")
	}
	original, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	game.SetSoundOptions(game.SoundOptions{})
	front, err := game.NewFrontEnd(assets)
	if err != nil {
		t.Fatal(err)
	}
	front.SetDeterministicFrames(true)
	sourceApp := front.App("legacy-source-control")
	sourceApp.Layout(1024, 768)
	open, _, err := front.RestoreOriginal(original)
	if err != nil {
		t.Fatal(err)
	}
	if err := sourceApp.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	imported, _, err := front.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	originalWorld, _ := front.LiveWorld()
	if won, lost := originalWorld.ScriptCounters(); !originalWorld.ScriptLatched(7) || originalWorld.Outcome() != sim.OutcomeWon || won != 1 || lost != 0 {
		t.Fatal("source lost its saved victory or spent completion latch")
	}
	for i := 0; i < 64; i++ {
		if err := sourceApp.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if _, kind, open := front.LiveNotice(); !open || kind != ui.NoticeSuccess || !originalWorld.ScriptLatched(7) {
		t.Fatal("original import lost Victory, replayed a spent notice or reset its completion latch")
	}
	if after, _ := originalWorld.MarshalBinary(); !bytes.Equal(imported.World, after) {
		t.Fatal("pending original Victory changed the canonical source state")
	}
	first, second := t.TempDir(), t.TempDir()
	if err := prepare(assets, source, first, io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := prepare(assets, source, second, io.Discard); err != nil {
		t.Fatal(err)
	}
	name := "game0000.sav"
	raw, err := os.ReadFile(filepath.Join(first, name))
	if err != nil {
		t.Fatal(err)
	}
	again, err := os.ReadFile(filepath.Join(second, name))
	if err != nil {
		t.Fatal(err)
	}
	firstSnapshot, firstLabel := decodeFixture(t, assets, raw)
	secondSnapshot, secondLabel := decodeFixture(t, assets, again)
	if firstLabel != secondLabel || !reflect.DeepEqual(firstSnapshot, secondSnapshot) {
		t.Fatalf("fixture state is not deterministic: labels=%q/%q", firstLabel, secondLabel)
	}
	snapshot, label := firstSnapshot, firstLabel
	if label != fixtureLabel {
		t.Fatalf("fixture envelope: label=%q", label)
	}
	var world sim.World
	if err := world.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	if world.Outcome() != sim.OutcomeUndecided || world.Tick() != 0 || world.ScriptLatched(7) {
		t.Fatal("fresh fixture decided/advanced the mission or inherited the source's spent completion latch")
	}
	for _, trigger := range world.Script().Triggers() {
		if world.ScriptLatched(trigger.Latch) {
			t.Fatalf("fresh fixture carries spent latch %d", trigger.Latch)
		}
	}
	if won, lost := world.ScriptCounters(); won != 0 || lost != 0 {
		t.Fatalf("fixture wrote outcome counters: won=%d lost=%d", won, lost)
	}
	importedParty, materialized, sourceObjects := fixturePartyAtInventoryBoundary(t, imported, originalWorld, true)
	snapshotParty, coldMaterialized, coldObjects := fixturePartyAtInventoryBoundary(t, snapshot, &world, false)
	if materialized == 0 || coldMaterialized != 0 || sourceObjects != coldObjects {
		t.Fatalf("entry template/live inventory boundary was not exercised: source=%d cold=%d objects=%d/%d", materialized, coldMaterialized, sourceObjects, coldObjects)
	}
	t.Logf("source entry-template inventory: %d null handle fields bound to %d distinct existing source World objects after exact item-value/stack-count checks; cold snapshot handles all match loaded World", materialized, sourceObjects)
	canonicalSavedObjectIDs(reflect.ValueOf(importedParty))
	canonicalSavedObjectIDs(reflect.ValueOf(snapshotParty))
	if !reflect.DeepEqual(importedParty, snapshotParty) || world.Purse(sim.SelfSlot) != 600 || snapshot.Gold != imported.Gold {
		logFixtureDifferences(t, "party", reflect.ValueOf(importedParty), reflect.ValueOf(snapshotParty))
		t.Logf("purse=%d snapshot gold=%d source gold=%d", world.Purse(sim.SelfSlot), snapshot.Gold, imported.Gold)
		t.Fatal("fresh fixture did not preserve the imported party and purse")
	}
	for _, name := range []string{"1013-world-map-one-click.json", "0163-mission-to-town.json"} {
		t.Run(name, func(t *testing.T) {
			scenario, err := game.ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", name))
			if err != nil {
				t.Fatal(err)
			}
			f, err := game.NewFrontEnd(assets)
			if err != nil {
				t.Fatal(err)
			}
			f.SetDeterministicFrames(true)
			app := f.App("legacy-endpoint-release")
			app.SetSaveSeams(f.SaveSeams(game.SaveStore{Dir: first}, game.OriginalStore{}, nil))
			var machine, trace bytes.Buffer
			if err := game.RunHeadlessScenario(f, app, scenario, &machine, &trace); err != nil {
				t.Fatalf("%v\n%s", err, trace.String())
			}
			// Require the sequence in the recorded production observations, not
			// merely a final town save or direct FinishMission call.
			decoder := json.NewDecoder(&machine)
			victory, returning, square, mission30 := false, false, false, false
			for decoder.More() {
				var event game.HeadlessEvent
				if err := decoder.Decode(&event); err != nil {
					t.Fatal(err)
				}
				state := event.State
				victory = victory || state.Notice == "victory"
				returning = returning || (victory && state.TownPlace == "world_map")
				square = square || (returning && state.TownPlace == "square")
				mission30 = mission30 || (square && state.Screen == ui.ScreenMap.String() && state.Mission == 30)
			}
			if !victory || !returning || !square || !mission30 {
				t.Fatalf("missing UI sequence: victory=%v return=%v square=%v mission30=%v", victory, returning, square, mission30)
			}
		})
	}
	unchanged, err := os.ReadFile(source)
	if err != nil || !bytes.Equal(original, unchanged) {
		t.Fatalf("source changed: %v", err)
	}
}

func logFixtureDifferences(t *testing.T, path string, want, got reflect.Value) {
	t.Helper()
	if reflect.DeepEqual(want.Interface(), got.Interface()) {
		return
	}
	switch want.Kind() {
	case reflect.Pointer:
		if !want.IsNil() && !got.IsNil() {
			logFixtureDifferences(t, path, want.Elem(), got.Elem())
			return
		}
	case reflect.Struct:
		for i := 0; i < want.NumField(); i++ {
			logFixtureDifferences(t, path+"."+want.Type().Field(i).Name, want.Field(i), got.Field(i))
		}
		return
	case reflect.Slice, reflect.Array:
		if want.Len() == got.Len() {
			for i := 0; i < want.Len(); i++ {
				logFixtureDifferences(t, fmt.Sprintf("%s[%d]", path, i), want.Index(i), got.Index(i))
			}
			return
		}
	}
	t.Logf("%s: %v -> %v", path, want.Interface(), got.Interface())
}
