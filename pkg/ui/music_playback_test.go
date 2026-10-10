package ui

import (
	"slices"
	"testing"

	"againrom/pkg/random"
)

func TestMusic1189SequentialPlayStopAndRandomOrder(t *testing.T) {
	source, device := &recordingMusicSource{}, &recordingMusicDevice{}
	m := NewMusicController(source, device, random.NewStream(1189))
	m.SetPreferences(MusicPreferences{Enabled: true})
	m.SetScene(MusicMission, false)
	if !slices.Equal(m.Candidates(), missionMusicTracks[:]) || m.Playing() != "B00.wav" {
		t.Fatal("sequential bank/start", m.Candidates(), m.Playing())
	}
	if !m.Play(3) || m.Playing() != "B03.wav" {
		t.Fatal("selected track not playing")
	}
	before := len(source.loads)
	if !m.Play(3) || len(source.loads) != before {
		t.Fatal("same track restarted")
	}
	m.SetPreferences(MusicPreferences{Enabled: true, RandomOrder: true})
	if len(source.loads) != before || m.Playing() != "B03.wav" {
		t.Fatal("order change restarted track")
	}
	m.SetPreferences(MusicPreferences{})
	m.SetScene(MusicMission, false)
	device.ended = true
	m.Update()
	m.SetScene(MusicTown, false)
	if m.Playing() != "" || len(source.loads) != before {
		t.Fatal("stopped music resumed through scene/update")
	}
	m.SetPreferences(MusicPreferences{Enabled: true})
	if len(source.loads) != before {
		t.Fatal("enabling raced selected Play")
	}
	if !m.Play(0) || m.Playing() != "town.wav" {
		t.Fatal("Play cannot resume in new context")
	}
	m.SetScene(MusicMission, false)
	device.ended = true
	m.Update()
	if m.Playing() != "B01.wav" {
		t.Fatal("sequential EOF did not advance")
	}
	if m.Play(-1) || m.Play(12) {
		t.Fatal("invalid list index played")
	}
}

func TestMusic1189MissingManualSelectionDoesNotPlayAnotherTrack(t *testing.T) {
	source := &recordingMusicSource{missing: map[string]bool{"B03.wav": true}}
	m := NewMusicController(source, &recordingMusicDevice{}, random.NewStream(1189))
	m.SetPreferences(MusicPreferences{Enabled: true})
	m.SetScene(MusicMission, false)
	if m.Play(3) || m.Playing() != "" || source.loads[len(source.loads)-1] != "B03.wav" {
		t.Fatal("missing selected track silently substituted another")
	}
}
