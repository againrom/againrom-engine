package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// placedPersonWorld is npcLineFixtureWorld with the actor rebuilt as a person
// placed from Humans row 1, named "NPC06_1", under the mods mods.
func placedPersonWorld(t *testing.T, mods mapload.ModContext, own string) (*mapWorld, sim.EntityID) {
	t.Helper()
	const id sim.EntityID = 5
	mw := npcLineFixtureWorld(t, id, 1, 0, "internal person")
	e, _ := mw.world.Entity(id)
	e.Humanoid = true
	e.SourceBinding = sim.SourceBinding{Class: 2, ArchiveIndex: 1, TokenRow: 1, TypeID: 1}
	w, err := sim.NewWorld(17, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), []sim.Entity{e})
	if err != nil {
		t.Fatal(err)
	}
	mw.world = w
	mw.actorNames = map[sim.EntityID]string{id: "NPC06_1"}
	mw.mission.table = &mapload.Table{Humans: dbCollection{{}, {name: "NPC06_1"}}, Mods: mods}
	if own != "" {
		mw.chars = map[sim.EntityID]ui.UnitCharacter{id: {Name: own, Known: true}}
	}
	return mw, id
}

func TestPlacedPersonShowsTheNameAModGivesItsRow(t *testing.T) {
	mods := mapload.ModContext{Characters: []mapload.ModCharacter{{Mod: "m", Rows: []string{"NPC06_1"}, Name: "Archer girl"}}}
	mw, id := placedPersonWorld(t, mods, "")
	if got := mw.shownName(id); got != "Archer girl" {
		t.Errorf("a placed person of a renamed row shows %q, want the mod's name", got)
	}
	mw, id = placedPersonWorld(t, mods, "Snoot")
	if got := mw.shownName(id); got != "Snoot" {
		t.Errorf("a placed person with its own name shows %q", got)
	}
	mw, id = placedPersonWorld(t, mapload.ModContext{}, "")
	if got := mw.shownName(id); got != "Fixture peasant" {
		t.Errorf("without a mod a placed person shows %q, want the class name", got)
	}
}

func TestSpeakerOfARenamedRowIsLabelledByTheModName(t *testing.T) {
	_, s, _ := nativeIdentityEditor(t)
	x := s.inspector
	x.table.NPCNames[7] = ""
	if label, _ := x.speakerLabel(8); label == "Archer girl" {
		t.Fatal("the label names the mod before the mod exists")
	}
	x.table.Mods = mapload.ModContext{Characters: []mapload.ModCharacter{{Mod: "m", Rows: []string{"Courier"}, Name: "Archer girl"}}}
	if label, ok := x.speakerLabel(8); !ok || label != "Archer girl" {
		t.Errorf("the speaker of a renamed row is labelled %q, %v", label, ok)
	}
	if label, _ := x.speakerLabel(7); label != "Guard native name" {
		t.Errorf("a speaker with its own name is labelled %q", label)
	}
}
