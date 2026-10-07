package game

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

func TestReleaseItemOwners1115TownAndNextMissionEquipment(t *testing.T) {
	f := releaseFront(t)
	source := os.Getenv("AGAINROM_SAVE_666")
	if source == "" {
		t.Skip("AGAINROM_SAVE_666 is not set")
	}
	scenario, err := ReadHeadlessScenario(filepath.Join("..", "..", "scenarios", "1005-doll-and-shop.json"))
	if err != nil {
		t.Fatal(err)
	}
	scenario.OriginalSaves, scenario.Saves = filepath.Dir(source), t.TempDir()
	f.SetDeterministicFrames(true)
	app := f.App("item ownership across town and next mission")
	app.SetSaveSeams(f.SaveSeams(SaveStore{Dir: scenario.Saves}, OriginalStore{Dir: scenario.OriginalSaves}, nil))
	// Mission 30 builds its item registry from the town's party, rather than
	// the completed mission-20 World. Locate the gates and SAVE by their input.
	steps := scenario.Steps
	openAt, saveAt := -1, -1
	for i, step := range steps {
		if step.Command == "activate" && step.Target == "walk out to mission 30" {
			if openAt >= 0 {
				t.Fatal("scenario repeats mission-30 gates")
			}
			openAt = i
		}
		if step.Command == "save" {
			if saveAt >= 0 {
				t.Fatal("scenario repeats SAVE")
			}
			saveAt = i
		}
	}
	if openAt < 1 || saveAt <= openAt+1 || saveAt+1 >= len(steps) || steps[saveAt+1].Command != "load" {
		t.Fatal("scenario requires mission-30 gates before SAVE and immediate LOAD")
	}
	part := func(from, to int) {
		t.Helper()
		run := scenario
		run.Steps = steps[from:to]
		if err := RunHeadlessScenario(f, app, run, io.Discard, io.Discard); err != nil {
			t.Fatal(err)
		}
		if f.live == nil {
			t.Fatalf("no live mission after step %d", to)
		}
	}
	part(0, openAt)
	completed := f.live.world.SavedObjects()
	if completed == nil {
		t.Fatal("completed mission-20 World has no registry")
	}
	part(openAt, openAt+1)
	if f.live.mission.number != 30 {
		t.Fatalf("gates opened mission %d, want 30", f.live.mission.number)
	}
	opened := f.live.world.SavedObjects()
	if opened == nil || reflect.DeepEqual(opened, completed) {
		t.Fatal("next authored mission unexpectedly reused the completed World's registry")
	}
	var party, rows []string
	for _, id := range f.live.mission.ids {
		carried, _ := f.live.world.CarriedStacks(id)
		for _, s := range carried {
			party = append(party, fmt.Sprintf("%d:%d:%d", s.ObjectID, s.Code, s.Count))
		}
		worn, _ := f.live.world.EquippedItems(id)
		for _, v := range worn {
			if v.ObjectID != 0 {
				party = append(party, fmt.Sprintf("%d:%d:1", v.ObjectID, v.Code))
			}
		}
	}
	for _, row := range opened.Items {
		rows = append(rows, fmt.Sprintf("%d:%d:%d", row.ID, row.Value.Code, row.Value.Count))
	}
	slices.Sort(party)
	slices.Sort(rows)
	if !slices.Equal(party, rows) {
		t.Fatalf("mission 30 registry items %v, party carries and wears %v", rows, party)
	}
	part(openAt+1, saveAt)
	if reflect.DeepEqual(f.live.world.SavedObjects(), completed) {
		t.Fatal("mission 30 holds the completed World's registry before its SAVE")
	}
	scenario.Steps = steps[saveAt:]
	if err := RunHeadlessScenario(f, app, scenario, io.Discard, io.Discard); err != nil {
		if f.live != nil {
			t.Logf("saved registry present=%t", f.live.world.SavedObjects() != nil)
			for _, id := range f.live.mission.ids {
				held, _ := f.live.world.EquippedItems(id)
				for _, e := range f.live.world.Entities() {
					if e.ID == id {
						t.Logf("party %d SourceBindingClass=%d ActorLoadSourceClass=%d heldWeapon=%+v", id, e.SourceBinding.Class, e.ActorLoad.Source.Class, held[0])
					}
				}
			}
		}
		t.Fatal(err)
	}
	if f.live == nil {
		t.Fatal("no mission is open after the in-mission LOAD")
	}
	cold, _, written := itemObjectCheckpoint(t, f, app)
	codes := make(map[uint16]int)
	for _, row := range f.live.world.SavedObjects().Items {
		codes[row.Value.Code]++
	}
	var worn sim.SavedObjectID
	for _, id := range f.live.mission.ids {
		held, _ := f.live.world.EquippedItems(id)
		for _, v := range held {
			if worn == 0 && v.ObjectID != 0 && codes[v.Code] == 1 {
				worn = v.ObjectID
			}
		}
	}
	if worn == 0 {
		t.Fatal("no worn Item with a unique code for the loss control")
	}
	requireAlteredItemWeight(t, f.live.world, written, worn)
	for range 20 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatal("next mission equipment continuation differs after ordinary SAVE/LOAD")
		}
	}
}
