package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/town"
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

func TestTownMusicRoomRulesAndInheritance(t *testing.T) {
	for _, id := range []int{1, 2, 3} {
		d := secondTownDescription(id)
		for _, room := range []string{"square", "inn", "shop"} {
			if scene, mage := townRoomMusic(d, townMusicState{room: room, present: true, mage: true}); scene != ui.MusicTown || mage {
				t.Fatalf("town %d room %s: music (%v, %v)", id, room, scene, mage)
			}
		}
	}
	d := &town.Description{Rooms: []town.RoomSpec{{Name: "inn", MusicRule: &town.RoomMusicSpec{Scene: "shop"}}}}
	if scene, _ := townRoomMusic(d, townMusicState{room: "inn", present: true}); scene != ui.MusicShop {
		t.Fatalf("description rule ignored: music %v", scene)
	}
	if scene, mage := townRoomMusic(d, townMusicState{}); scene != ui.MusicCampaign || mage {
		t.Fatalf("outside town: music (%v, %v)", scene, mage)
	}
}
