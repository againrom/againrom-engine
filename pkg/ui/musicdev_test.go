package ui

import (
	"testing"

	"againrom/pkg/audio"
)

func TestMusicVolumeMuteVolumeAndReenable(t *testing.T) {
	cases := []struct {
		settings audio.Settings
		want     float64
	}{
		{audio.Settings{Master: audio.MasterUnit, Muted: true}, 0},
		{audio.Settings{Master: 0}, 0},
		{audio.Settings{Master: audio.MasterUnit / 2}, 0.5},
		{audio.Settings{Master: audio.MasterUnit}, 1},
		{audio.Settings{Master: audio.MasterUnit * 2}, 1},
	}
	for _, tc := range cases {
		if got := musicVolume(tc.settings); got != tc.want {
			t.Errorf("musicVolume(%+v) = %v, want %v", tc.settings, got, tc.want)
		}
	}
	if got := musicVolume(audio.Settings{Master: audio.MasterUnit, Muted: false}); got != 1 {
		t.Fatalf("re-enabled full volume = %v, want 1", got)
	}
}
