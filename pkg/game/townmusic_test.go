package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestTownMusicFollowsVisibleRoomAndSelectedSchoolClass(t *testing.T) {
	front := &FrontEnd{CampaignSession: CampaignSession{Carried: []mapload.PartyMember{{Mage: false}, {Mage: true}}}}
	screen := front.bindTown(&townScreen{})

	cases := []struct {
		name      string
		room      townRoom
		building  TownBuilding
		member    int
		wantScene ui.MusicScene
		wantMage  bool
	}{
		{"square", roomSquare, TownTavern, 0, ui.MusicTown, false},
		{"gates", roomGates, TownTavern, 0, ui.MusicCampaign, false},
		{"shop", roomShop, TownTavern, 0, ui.MusicShop, false},
		{"tavern", roomTavern, TownTavern, 0, ui.MusicTavern, false},
		{"school fighter", roomSchool, TownTavern, 0, ui.MusicSchool, false},
		{"school mage", roomSchool, TownTavern, 1, ui.MusicSchool, true},
		{"shop dialogue", roomTalk, TownShop, 0, ui.MusicShop, false},
		{"tavern dialogue", roomTalk, TownTavern, 0, ui.MusicTavern, false},
		{"school dialogue", roomTalk, TownSchool, 1, ui.MusicSchool, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			screen.room, screen.dialogueBuilding, screen.shopMember = tc.room, tc.building, tc.member
			gotScene, gotMage := screen.TownMusic()
			if gotScene != tc.wantScene || gotMage != tc.wantMage {
				t.Fatalf("TownMusic() = (%v, %v), want (%v, %v)", gotScene, gotMage, tc.wantScene, tc.wantMage)
			}
		})
	}
}
