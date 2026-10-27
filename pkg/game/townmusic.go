package game

import (
	"againrom/pkg/town"
	"againrom/pkg/ui"
)

type townMusicState struct {
	room    string
	present bool
	mage    bool
}

var townMusicScenes = map[string]ui.MusicScene{
	"town": ui.MusicTown, "campaign": ui.MusicCampaign, "shop": ui.MusicShop,
	"tavern": ui.MusicTavern, "school": ui.MusicSchool,
}

func townRoomMusic(d *town.Description, state townMusicState) (ui.MusicScene, bool) {
	if !state.present {
		return ui.MusicCampaign, false
	}
	if d != nil {
		if r := d.Room(state.room); r != nil && r.MusicRule != nil {
			return townMusicScenes[r.MusicRule.Scene], r.MusicRule.By == "member-class" && state.mage
		}
	}
	return ui.MusicTown, false
}
