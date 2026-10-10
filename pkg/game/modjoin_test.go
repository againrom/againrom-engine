package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

func joinFixtureFront(t *testing.T) *FrontEnd {
	t.Helper()
	c := Campaign{Main: []int{30, 40}, Chapters: map[int]Chapter{
		30: {Mission: 30, AddHero: []int{22}, Inn: []int{30, 31}, InnNPC: []int{22, 5}},
		40: {Mission: 40, AddHero: []int{23}, Inn: []int{40}, InnNPC: []int{7}},
		50: {Mission: 50, Inn: []int{50}, InnNPC: []int{8}},
	}}
	return &FrontEnd{InstallResources: InstallResources{Campaign: resolved(c, nil), Table: &mapload.Table{}}}
}

func joinOf(chapter, companion, talk int) mod.CompanionJoin {
	return mod.CompanionJoin{Mod: "m", File: mod.CompanionsFile, Line: 2, Key: "k", Companion: companion, Chapter: chapter, Building: mod.BuildingTavern, Talk: talk}
}

func TestSetModCompanionsRefusesWhatTheCampaignDoesNotBearOut(t *testing.T) {
	for name, c := range map[string]struct {
		join mod.CompanionJoin
		want string
	}{
		"unknown chapter":       {joinOf(60, 22, 22), "chapter 60 is not a chapter"},
		"map companion":         {joinOf(40, 25, 7), "companion 25 joins on a map, not in a town"},
		"no grant":              {joinOf(50, 22, 8), "the town of chapter 50 does not grant companion 22"},
		"conversation not held": {joinOf(30, 22, 99), "the tavern of chapter 30 has no conversation with record 99 (it has [22 5])"},
	} {
		f := joinFixtureFront(t)
		err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{c.join}})
		want := `mod "m": data/companions.toml:2: ` + c.want
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: %v, want %q", name, err, want)
		}
		if !f.Table.Mods.Companions.Empty() {
			t.Errorf("%s: a refused condition was recorded", name)
		}
	}
	f := joinFixtureFront(t)
	if err := f.SetModCompanions(mod.CompanionData{}); err != nil || f.townInstall().holdsCompanion(30, 22) {
		t.Fatalf("empty data: %v", err)
	}
	if err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{joinOf(30, 22, 22)}}); err != nil {
		t.Fatal(err)
	}
	if !f.townInstall().holdsCompanion(30, 22) || f.townInstall().holdsCompanion(40, 22) || f.townInstall().holdsCompanion(30, 25) {
		t.Fatal("holdsCompanion names a grant the condition does not")
	}
	// SetMods leaves the recorded conditions in place whichever call comes first.
	if err := f.SetMods(f.Table.Rules, mod.Set{Base: "rom1-en"}, false); err != nil || !f.townInstall().holdsCompanion(30, 22) {
		t.Fatalf("SetMods dropped the conditions: %v", err)
	}
}

// The town companion is whatever the registry's AddHero arrays name: the
// fixture's chapter 40 grants 23, so a join naming 23 is a town join and 25,
// which no AddHero names, is not.
func TestTownCompanionIsTheRegistryAddHero(t *testing.T) {
	f := joinFixtureFront(t)
	c := f.Campaign.Value()
	if !c.townGrants(22) || !c.townGrants(23) || c.townGrants(25) || c.townGrants(0) {
		t.Fatal("townGrants does not follow the AddHero arrays")
	}
	if err := f.SetModCompanions(mod.CompanionData{Joins: []mod.CompanionJoin{joinOf(40, 23, 7)}}); err != nil {
		t.Fatalf("a join naming the chapter-40 grant was refused: %v", err)
	}
	if !f.townInstall().holdsCompanion(40, 23) {
		t.Fatal("the chapter-40 grant is not held")
	}
	var none Campaign
	if none.townGrants(22) {
		t.Fatal("a campaign with no AddHero grants a town companion")
	}
	// A city document grafts the companion the registry grants, and no other.
	if !nativeCityGraftableMember(mapload.PartyMember{CompanionNPC: 23}, c) ||
		nativeCityGraftableMember(mapload.PartyMember{CompanionNPC: 23}, none) ||
		nativeCityGraftableMember(mapload.PartyMember{CompanionNPC: 25}, c) {
		t.Fatal("the graftable companion does not follow the AddHero arrays")
	}
}

func TestAHeldGrantStaysPendingUntilItsOwnTake(t *testing.T) {
	keep := func(npc int) bool { return npc == 22 }
	c := joinFixtureFront(t).Campaign.Value()

	native := NewTown(c)
	native.Arrive()
	if got := native.takeAddHeroesExcept(30, keep); len(got) != 0 || !reflect.DeepEqual(native.pendingNativeHeroGrants(30), []int{22}) {
		t.Fatalf("native: kept grant handed out %v, pending %v", got, native.pendingNativeHeroGrants(30))
	}
	if native.takeAddHero(30, 25) {
		t.Fatal("native: took a grant that is not pending")
	}
	if !native.takeAddHero(30, 22) || native.takeAddHero(30, 22) || len(native.pendingNativeHeroGrants(30)) != 0 {
		t.Fatal("native: the pending grant was not consumed exactly once")
	}
	unheld := NewTown(c)
	unheld.Arrive()
	if got := unheld.takeAddHeroesExcept(30, nil); !reflect.DeepEqual(got, []int{22}) || len(unheld.pendingNativeHeroGrants(30)) != 0 {
		t.Fatalf("native without a condition: %v", got)
	}

	cc, projection := restoredCampaignFixture()
	projection.Main.AddHero = []uint16{22, 25}
	progress, err := campaignProgressFromSAV(cc, projection)
	if err != nil {
		t.Fatal(err)
	}
	town := newTownFromCampaignProgress(cc, progress)
	if got := town.takeAddHeroesExcept(50, keep); !reflect.DeepEqual(got, []int{25}) || !reflect.DeepEqual(progress.projection().Main.AddHero, []uint16{22}) {
		t.Fatalf("progress: handed out %v, the record holds %v", got, progress.projection().Main.AddHero)
	}
	if town.takeAddHero(50, 25) || !town.takeAddHero(50, 22) || len(progress.projection().Main.AddHero) != 0 || town.takeAddHero(50, 22) {
		t.Fatalf("progress: the held grant was not consumed exactly once: %v", progress.projection().Main.AddHero)
	}
}
