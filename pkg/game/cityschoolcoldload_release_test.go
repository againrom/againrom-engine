package game

import (
	"testing"

	"againrom/pkg/ui"
)

func TestReleaseCitySchoolTrainsNativeMageAfterColdLoad(t *testing.T) {
	store := SaveStore{Dir: t.TempDir()}
	f, app := saveDialogArrivedTown(t, store.Dir)
	t.Cleanup(app.StopAudio)
	raw := cityRosterF2Save(t, app, store, "school-seed")
	cold, coldApp, coldStore := cityPotionLoadApp(t, raw)
	idx := cityMageIndex(t, cold)
	member := cold.Carried[idx]
	if !member.Mage || f.Carried[idx].ID != member.ID {
		t.Fatal("the loaded city lost its mage", member.ID)
	}
	gold, slot, price := cold.Town.Gold(), 0, 0
	for i := 1; i <= 5; i++ {
		if p := memberSchoolPrice(member, i); p > 0 && p <= gold && (slot == 0 || p < price) {
			slot, price = i, p
		}
	}
	if slot == 0 {
		t.Fatal("no affordable mage school slot")
	}
	s := cold.TownScreen().(*townScreen)
	s.room, s.schoolCell, s.shopMember = roomSchool, slot+4, idx
	if err := coldApp.HeadlessActivate(cold.Words.SchoolTrain); err != nil {
		t.Fatal(err)
	}
	trained := trainingPartyMember(t, cold, member.ID)
	if cold.Town.Gold() != gold-price || trained.Hero.Skill[slot] != member.Hero.Skill[slot]+1 {
		t.Fatalf("Train after cold LOAD: gold %d -> %d (price %d), skill %d -> %d", gold, cold.Town.Gold(), price, member.Hero.Skill[slot], trained.Hero.Skill[slot])
	}
	if coldApp.Screen() != ui.ScreenTown {
		t.Fatal("Train left the city", coldApp.Screen())
	}
	again := cityRosterF2Save(t, coldApp, coldStore, "school-trained")
	back, _, _ := cityPotionLoadApp(t, again)
	if got := trainingPartyMember(t, back, member.ID); got.Hero.Skill[slot] != trained.Hero.Skill[slot] || back.Town.Gold() != cold.Town.Gold() {
		t.Fatalf("second cold LOAD skill %d gold %d, want %d and %d", got.Hero.Skill[slot], back.Town.Gold(), trained.Hero.Skill[slot], cold.Town.Gold())
	}
}
