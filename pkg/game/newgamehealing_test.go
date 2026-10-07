package game

import (
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestNewGameHealingUsesIncomingParty(t *testing.T) {
	for _, tc := range []struct {
		name    string
		old     []uint32
		profile int
		want    uint32
	}{
		{"fresh default", []uint32{0}, -1, 50},
		{"unrelated inconsistent source", []uint32{0, 100}, -1, 50},
		{"explicit no", []uint32{0}, 0, 100},
		{"explicit standard", []uint32{0}, 1, 50},
		{"explicit often", []uint32{100}, 2, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := missionFrontEnd(t)
			f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
			if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpener(10)(); err != nil {
				t.Fatal(err)
			}
			f.Town = NewTown(Campaign{})
			f.Town.gold = 123
			f.Carried = healingSourceParty(tc.old...)
			oldTown, oldLive := f.Town, f.live
			oldParty, oldHash := mapload.CloneParty(f.Carried), f.live.world.Hash()
			unchanged := func() {
				t.Helper()
				if f.Town != oldTown || f.live != oldLive || f.Town.Gold() != 123 ||
					!reflect.DeepEqual(f.Carried, oldParty) || oldLive.world.Hash() != oldHash {
					t.Fatal("new-game preparation changed the previous session")
				}
			}
			result := ui.ChargenResult{Name: "New party", Stats: []int{25, 25, 25, 25}}
			if _, err := f.prepareNewGame(20, result); err == nil {
				t.Fatal("missing mission did not fail")
			}
			unchanged()
			validOptions := f.Options
			f.Options.Path = t.TempDir()
			if _, err := f.prepareNewGame(10, result); err == nil {
				t.Fatal("unreadable options did not fail")
			}
			unchanged()
			f.Options = validOptions
			if tc.profile >= 0 {
				if err := f.Options.setGameOption(ui.GameOptionAutoHealing, tc.profile); err != nil {
					t.Fatal(err)
				}
			}
			open, err := f.prepareNewGame(10, result)
			unchanged()
			if err != nil {
				t.Fatal("old party must not govern a new campaign", err)
			}
			if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
				t.Fatal(err)
			}
			if got, present := f.live.world.AutoHealing(sim.SelfSlot); !present || got != tc.want {
				t.Fatalf("new party inherited old session policy: got %d/%v, want %d", got, present, tc.want)
			}
			if f.live == oldLive || f.Town == oldTown || len(f.Carried) != 0 {
				t.Fatal("new campaign was not adopted")
			}
		})
	}
}

func TestMapPickerHealingIgnoresCampaignParty(t *testing.T) {
	f := missionFrontEnd(t)
	f.Carried = healingSourceParty(0, 100)
	f.Maps = []MapEntry{{Source: "10.alm", FromArchive: true}}
	if _, _, _, _, _, _, _, _, _, _, err := f.loadMap(0); err != nil {
		t.Fatal("plain map must not consult campaign party policy", err)
	}
	if got, present := f.live.world.AutoHealing(sim.SelfSlot); !present || got != 50 {
		t.Fatal("plain map did not use default policy", got, present)
	}
}

func healingSourceParty(percentages ...uint32) []mapload.PartyMember {
	party := make([]mapload.PartyMember, len(percentages))
	for i, percent := range percentages {
		party[i].Carry = &mapload.Carry{LiveLoad: &sim.ActorLoadSnapshot{}}
		party[i].Carry.LiveLoad.Inventory.Source = sim.SourceActor{Class: 2, HasOwner: true, ManaReservePercent: percent}
	}
	return party
}
