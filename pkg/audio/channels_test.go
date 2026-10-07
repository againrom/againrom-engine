package audio

import "testing"

func TestChannelVolumesCombineIndependentLevelsWithMaster(t *testing.T) {
	levels := ChannelVolumes{25, 50, 75}
	for channel, want := range []int{20, 40, 60} {
		for _, muted := range []bool{false, true} {
			got := levels.Settings(Channel(channel), Settings{Master: 80, Muted: muted})
			if got != (Settings{Master: want, Muted: muted}) {
				t.Fatal(channel, got)
			}
		}
	}
	if levels.Settings(ChannelCount, DefaultSettings) != (Settings{Muted: true}) {
		t.Fatal("invalid channel must be silent")
	}
	levels[MusicChannel] = -1
	if levels.Valid() || levels.Settings(MusicChannel, DefaultSettings).Master != 0 {
		t.Fatal("negative channel volume accepted")
	}
	levels[MusicChannel] = 101
	if levels.Valid() || levels.Settings(MusicChannel, Settings{Master: 101}).Master != 100 {
		t.Fatal("amplifying gain was not bounded")
	}
}
