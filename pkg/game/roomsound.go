package game

import (
	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

func (t *townScreen) soundRoom() townRoom {
	if t.room == roomTalk {
		switch t.dialogueBuilding {
		case TownTavern:
			return roomTavern
		case TownShop:
			return roomShop
		case TownSchool:
			return roomSchool
		}
	}
	return t.room
}

func (t *townScreen) roomSoundPlayer(group audio.Channel) audio.Player {
	if t == nil || t.sess == nil {
		return nil
	}
	room := t.soundRoom()
	if t.roomAudio != nil && (t.audioRoom != room || t.roomAudio.Owner() != ui.DeliveryOwner(t.sound.soundDevice())) {
		t.destroyRoomAudio()
	}
	if t.roomAudio == nil {
		t.roomAudio = ui.NewAudioScope(t.sound.soundDevice())
		t.audioRoom = room
	}
	if t.roomAudio != nil {
		return t.roomAudio.Player(group)
	}
	if group == audio.SpeechChannel {
		return t.sound.speechDevice()
	}
	return t.sound.soundDevice()
}

func (t *townScreen) destroyRoomAudio() {
	if t != nil {
		t.roomAudio.Destroy()
		t.roomAudio = nil
		t.audioRoom = roomSquare
	}
}
