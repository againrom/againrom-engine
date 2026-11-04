package game

import (
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

type dialogueMusicSource struct{ loads []string }

func (s *dialogueMusicSource) Track(name string) (audio.Track, bool) {
	s.loads = append(s.loads, name)
	return audio.Track{Rate: audio.DeviceRate, StereoPCM: make([]byte, 16)}, true
}

type dialogueFadeDevice struct {
	secondTestMusic
	volume  int
	volumes []int
}

func (d *dialogueFadeDevice) MusicVolume() int                           { return d.volume }
func (d *dialogueFadeDevice) SetMusicVolume(v int)                       { d.volume = v; d.volumes = append(d.volumes, v) }
func (d *dialogueFadeDevice) ResetMusicVolume()                          { d.volume = 0 }
func (d *dialogueFadeDevice) MusicTimes() (time.Duration, time.Duration) { return 10 * time.Second, -1 }

func TestDialogueTunePageDrivesProfilePlayerAndPlainPageDoesNothing(t *testing.T) {
	for name, g := range talkGames() {
		t.Run(name, func(t *testing.T) {
			s, _ := openFakeTalk(g, g.profile, "<part=1 tune=3>\r\nMusic\r\n<part=2>\r\nWords\r\n")
			game := base.GameROM1
			if name == "second" {
				game = base.GameROM2
			}
			source, device := &dialogueMusicSource{}, &dialogueFadeDevice{}
			m := ui.NewMusicController(GameMusic(base.Profile{Game: game}), source, device, nil)
			m.SetScene(ui.MusicMission, false)
			n, ok := s.TakeMusicTune()
			if !ok || n != 3 {
				t.Fatal("accepted town page did not queue tune", n, ok)
			}
			accepted := m.Tune(n)
			if accepted != (name == "second") {
				t.Fatal("profile chose the wrong tune policy")
			}
			if m.Playing() != "B00.wav" {
				t.Fatal("tune replaced the entry before fading")
			}
			now := time.Now()
			for i := 1; i <= 100; i++ {
				m.UpdateAt(now.Add(time.Duration(i) * 100 * time.Millisecond))
			}
			if name == "second" {
				if m.Playing() != "B03.wav" || len(device.volumes) < 2 || device.volumes[0] != -367 {
					t.Fatal("dialogue did not drive the claimed fade to index 3", m.Playing(), device.volumes)
				}
				device.ended = true
				m.UpdateAt(now.Add(11 * time.Second))
				if m.Playing() != "B03.wav" {
					t.Fatal("dialogue hold was not set")
				}
			} else if m.Playing() != "B00.wav" || len(source.loads) != 1 || len(device.volumes) != 0 {
				t.Fatal("first game's existing behaviour changed")
			}
			before := len(source.loads)
			s.AdvanceTownDialogue()
			if _, ok := s.TakeMusicTune(); ok {
				t.Fatal("page without tune queued music")
			}
			m.UpdateAt(now.Add(12 * time.Second))
			if len(source.loads) != before {
				t.Fatal("plain page changed music")
			}
		})
	}
}

func TestDialogueTuneMissionOpenAndAdvanceUseTheSharedReader(t *testing.T) {
	m, v := missionDialogue(t, "<part=1 tune=2>\r\nOne\r\n<part=2>\r\nTwo\r\n<part=3 tune=4>\r\nThree\r\n", nil)
	if n, ok := v.TakeMusicTune(); !ok || n != 2 {
		t.Fatal("mission open lost tune", n, ok)
	}
	if _, ok := v.TakeMusicTune(); ok {
		t.Fatal("cue repeated")
	}
	m.advanceNotice()
	if _, ok := v.TakeMusicTune(); ok {
		t.Fatal("plain mission page changed music")
	}
	m.advanceNotice()
	if n, ok := v.TakeMusicTune(); !ok || n != 4 {
		t.Fatal("mission advance lost tune", n, ok)
	}
	for _, tag := range []string{"tune=-1", "tune=no", "tune=9999999999999", "iamfemale tune=3"} {
		r := readDialoguePart([]byte("<part=1 "+tag+">\r\nWords"), 1, EventAudience{tags: secondEventTags{}}, nil)
		if r.musicTune() != -1 {
			t.Fatal("invalid or rejected tag produced tune", tag)
		}
	}
}
