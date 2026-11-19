package game

import (
	"slices"
	"testing"

	"againrom/pkg/sim"
)

// secondVictimPair is the two living actors of the computer that share a
// kind word, the weakest such pair in actor order.
func secondVictimPair(t *testing.T, w *sim.World, hero sim.Entity) (sim.Entity, sim.Entity) {
	t.Helper()
	all := w.Entities()
	var pair [2]sim.Entity
	best := int32(-1)
	for i, a := range all {
		if !a.Alive() || a.KindWord == 0 || a.Owner == hero.Owner || a.Humanoid {
			continue
		}
		for _, b := range all[i+1:] {
			if !b.Alive() || b.KindWord != a.KindWord || b.Owner != a.Owner || b.Humanoid {
				continue
			}
			if worst := max(a.MaxHP, b.MaxHP); best < 0 || worst < best {
				pair, best = [2]sim.Entity{a, b}, worst
			}
		}
	}
	if best < 0 {
		t.Fatal("no two creatures of the computer share a kind word")
	}
	return pair[0], pair[1]
}

// On installed data, the hero's kills of two creatures of one kind word in
// the campaign's first mission count as one kind (R2-ENGINE-1157); the list
// and every actor's kind word and main skill cross a SAVE and a cold LOAD.
func TestReleaseSecondGameRepeatedVictimCrossesSaveAndLoad(t *testing.T) {
	f := secondGameFront(t)
	f.Options = OptionsStore{}
	app := f.App("second-game victims")
	app.Layout(1024, 768)
	enterSecondCampaignMission(t, app)
	dismissSecondMissionDialogue(t, app, f.live)
	w := f.live.world
	hero := f.live.mission.ids[0]
	h, _ := w.Entity(hero)
	if h.KindWord != 21 || h.MainSkill == 0 {
		t.Fatalf("the hero holds kind %d main %d, want 21 and his chosen skill", h.KindWord, h.MainSkill)
	}
	a, b := secondVictimPair(t, w, h)
	t.Logf("victims %d and %d (types %d and %d) of kind %d", a.ID, b.ID, a.TypeID, b.TypeID, a.KindWord)
	victimKindKillActor(t, f, hero, a.ID)
	victimKindKillActor(t, f, hero, b.ID)
	h, _ = w.Entity(hero)
	list := h.Values().VictimKinds
	if !slices.Contains(list, sim.VictimKindCount{Kind: uint32(a.KindWord), Count: 1}) {
		t.Fatalf("the hero's list is %+v, want kind %d counted twice", list, a.KindWord)
	}
	dismissSecondMissionDialogue(t, app, f.live)
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatalf("%v: notice open %t kind %v", err, f.live.mission.open, f.live.mission.kind)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := secondGameFront(t)
	cold.Options = OptionsStore{}
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("LOAD town=%t: %v", town, err)
	}
	if err := cold.App("second-game victims LOAD").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	loaded, _ := cold.live.world.Entity(hero)
	if got := loaded.Values().VictimKinds; !slices.Equal(got, list) {
		t.Fatalf("the loaded hero's list is %+v, want %+v", got, list)
	}
	for _, e := range w.Entities() {
		c, ok := cold.live.world.Entity(e.ID)
		if ok && (c.KindWord != e.KindWord || c.MainSkill != e.MainSkill) {
			t.Errorf("actor %d loads kind %d main %d, saved %d and %d", e.ID, c.KindWord, c.MainSkill, e.KindWord, e.MainSkill)
		}
	}
}
