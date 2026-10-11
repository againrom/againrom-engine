package ui

import (
	"image"
	"strconv"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/random"
)

func settingsContractOptionsApp(t *testing.T) *App {
	t.Helper()
	var seam *cadenceSeam
	a := newTestApp(t, appRows(1), cadenceLoader(t, &seam))
	a.Layout(640, 480)
	a.SetMapCadencePreference(defaultCadenceRung(), nil)
	values := GameOptionValues{}
	a.SetGameOptionControls(GameOptionControls{
		Read: func(bool) GameOptionValues { return values },
		Write: func(_ bool, option GameOption, value int) error {
			values[option] = value
			return nil
		},
		Words: GameOptionWords{Title: "Game Options", OK: "OK", Cancel: "Cancel", Speed: "Game Speed"},
	})
	a.flow.menuFont, a.flow.screen = gameMenuTestFont(), ScreenPicker
	for _, key := range []string{"enter", "escape"} {
		if err := a.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	return a
}
func TestGameOptionsPositiveSpeedPositionsUseTheMillisecondTable(t *testing.T) {
	periods := [...]int{125000, 100000, 83000, 71000, 62000, 50000, 41000, 35000, 31000}
	for index, period := range periods {
		t.Run(strconv.Itoa(index+1), func(t *testing.T) {
			a := settingsContractOptionsApp(t)
			if a.flow.gameOptions.draft.speed != 5 || a.flow.rung != 7 {
				t.Fatal("default speed changed", a.flow.gameOptions.draft.speed, a.flow.rung)
			}
			var persisted []int
			a.SetMapCadencePreference(7, func(rung int) { persisted = append(persisted, rung) })
			var calls []cadenceCall
			a.flow.cadence = func(periodUS int, stopped, unpaced, reset bool) {
				calls = append(calls, cadenceCall{periodUS: periodUS, stopped: stopped, unpaced: unpaced, reset: reset})
			}
			level := index + 1
			steps, action := level-5, "speed-up"
			if steps < 0 {
				steps, action = -steps, "speed-down"
			}
			for range steps {
				if err := a.HeadlessGameMenuAction(action); err != nil {
					t.Fatal(err)
				}
			}
			if a.flow.gameOptions.draft.speed != level {
				t.Fatal("speed controls did not reach the requested level", a.flow.gameOptions.draft.speed, level)
			}
			if a.flow.rung != 7 || len(persisted) != 0 || len(calls) != 0 {
				t.Fatal("draft changed the active or persisted cadence", a.flow.rung, persisted, calls)
			}
			if err := a.HeadlessGameMenuAction("page-return"); err != nil {
				t.Fatal(err)
			}
			if a.flow.rung != index+3 || a.MapCadencePreference() != index+3 || a.flow.viewer.anim.Period() != period {
				t.Fatal("OK selected the wrong cadence", a.flow.rung, a.MapCadencePreference(), a.flow.viewer.anim.Period(), period)
			}
			if index == 4 {
				if len(persisted) != 0 {
					t.Fatal("unchanged default was persisted", persisted)
				}
			} else if len(persisted) != 1 || persisted[0] != index+3 {
				t.Fatal("OK did not persist the selected rung once", persisted)
			}
			if len(calls) > 0 {
				last := calls[len(calls)-1]
				if last.periodUS != period || !last.stopped || last.unpaced {
					t.Fatal("paused driver received the wrong cadence", last, period)
				}
			} else if index != 4 {
				t.Fatal("changed speed did not reach the driver")
			}
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
			if len(calls) == 0 {
				t.Fatal("closing the menu did not reach the cadence driver")
			}
			last := calls[len(calls)-1]
			if last.periodUS != period || last.stopped || last.unpaced {
				t.Fatal("resumed driver received the wrong cadence", last, period)
			}
		})
	}
}

func TestGameOptionsZeroSpeedPausesOnlyTheMissionAndPreservesTheDefault(t *testing.T) {
	a := settingsContractOptionsApp(t)
	if a.flow.gameOptions.draft.speed != 5 || a.flow.rung != 7 || a.flow.viewer.anim.Period() != 62000 {
		t.Fatal("default speed changed", a.flow.gameOptions.draft.speed, a.flow.rung, a.flow.viewer.anim.Period())
	}
	var persisted []int
	a.SetMapCadencePreference(7, func(rung int) { persisted = append(persisted, rung) })
	for range 5 {
		if err := a.HeadlessGameMenuAction("speed-down"); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.gameOptions.draft.speed != 0 || a.flow.stopped || len(persisted) != 0 {
		t.Fatal("mission zero applied before OK", a.flow.gameOptions.draft.speed, a.flow.stopped, persisted)
	}
	if err := a.HeadlessGameMenuAction("page-return"); err != nil {
		t.Fatal(err)
	}
	if !a.flow.stopped || a.flow.rung != 7 || a.MapCadencePreference() != 7 || len(persisted) != 0 {
		t.Fatal("mission zero changed the resume speed", a.flow.stopped, a.flow.rung, a.MapCadencePreference(), persisted)
	}
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if a.flow.gameOptions.draft.speed != 0 {
		t.Fatal("mission pause was not shown on reopening", a.flow.gameOptions.draft.speed)
	}
	if err := a.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	a.flow.menuBack = ScreenTown
	if err := a.HeadlessGameMenuAction("game-options"); err != nil {
		t.Fatal(err)
	}
	if a.flow.gameOptions.draft.speed != 5 {
		t.Fatal("town copied the mission pause", a.flow.gameOptions.draft.speed)
	}
	for range 5 {
		if err := a.HeadlessGameMenuAction("speed-down"); err != nil {
			t.Fatal(err)
		}
	}
	if a.flow.gameOptions.draft.speed != 1 || a.MapCadencePreference() != 7 || len(persisted) != 0 {
		t.Fatal("town offered active pause or applied its draft", a.flow.gameOptions.draft.speed, a.MapCadencePreference(), persisted)
	}
}

func TestSoundOptionsEscapeDiscardsLocalChoicesAndRetainsImmediateChanges(t *testing.T) {
	for _, action := range []string{"music-play", "music-stop"} {
		t.Run(action, func(t *testing.T) {
			a := NewApp("sound escape", appAssets(t), appRows(1), nil)
			a.Layout(640, 480)
			a.flow.viewer, a.flow.screen, a.flow.menuFont = fiViewer(t), ScreenMap, gameMenuTestFont()
			a.SetGameMenuSettings(nil, nil, func() (bool, int, bool) { return true, 100, true }, nil)
			volumes := audio.FullChannelVolumes()
			acknowledgments, acknowledgmentWrites := true, 0
			playback := MusicPreferences{Enabled: true}
			a.SetSoundOptionControls(SoundOptionControls{
				Words: DefaultSoundOptionWords(),
				Read:  func() audio.ChannelVolumes { return volumes },
				Write: func(channel audio.Channel, value int) error {
					volumes[channel] = value
					return nil
				},
				ReadAcknowledgments: func() bool { return acknowledgments },
				WriteAcknowledgments: func(value bool) error {
					acknowledgments = value
					acknowledgmentWrites++
					return nil
				},
				ReadPlayback: func() MusicPreferences { return playback },
				WritePlayback: func(value MusicPreferences) error {
					playback = value
					return nil
				},
			})
			source, device := &recordingMusicSource{}, &recordingMusicDevice{}
			a.SetMusic(firstMusic, source, device, random.NewStream(1))
			defer a.StopMusic()
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
				t.Fatal(err)
			}
			oldList := a.flow.soundOptions.list
			row := soundTrackBox(a.flow.menuFont).Row(3)
			point := image.Pt(row.Min.X+60, row.Min.Y+row.Dy()/2)
			for _, edge := range []string{"press", "release"} {
				if err := a.HeadlessPointer(edge, point.X, point.Y); err != nil {
					t.Fatal(err)
				}
			}
			if oldList.Selection() != 3 || a.music.Playing() != "B00.wav" {
				t.Fatal("track choice changed playback", oldList.Selection(), a.music.Playing())
			}
			wantVolumes := audio.ChannelVolumes{25, 50, 75}
			for channel, volume := range wantVolumes {
				point := SoundSliderPoint(audio.Channel(channel), volume)
				for _, edge := range []string{"press", "release"} {
					if err := a.HeadlessPointer(edge, point.X, point.Y); err != nil {
						t.Fatal(err)
					}
				}
			}
			for _, option := range []string{"acknowledgments", "music-random", "music-play"} {
				if err := a.HeadlessGameMenuAction(option); err != nil {
					t.Fatal(err)
				}
			}
			if a.flow.soundOptions.acknowledgments || !acknowledgments || acknowledgmentWrites != 0 ||
				volumes != wantVolumes || !playback.RandomOrder || !playback.Enabled || a.music.Playing() != "B03.wav" {
				t.Fatal("staged or immediate action did not take effect", a.flow.soundOptions.acknowledgments, acknowledgments, acknowledgmentWrites, volumes, playback, a.music.Playing())
			}
			wantPlaying := "B03.wav"
			if action == "music-stop" {
				if err := a.HeadlessGameMenuAction(action); err != nil {
					t.Fatal(err)
				}
				wantPlaying = ""
			}
			loads, events, requests := len(source.loads), len(device.events), len(a.MusicRequestLog())
			if err := a.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if a.flow.menuPage != gameMenuRoot || !a.flow.viewer.menuUp {
				t.Fatal("Escape lost the menu's modal ownership")
			}
			if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
				t.Fatal(err)
			}
			if a.flow.soundOptions.list == oldList || a.flow.soundOptions.list.Selection() != 0 ||
				!a.flow.soundOptions.acknowledgments || !acknowledgments || acknowledgmentWrites != 0 {
				t.Fatal("Escape retained a local choice", a.flow.soundOptions.list.Selection(), a.flow.soundOptions.acknowledgments, acknowledgments, acknowledgmentWrites)
			}
			if volumes != wantVolumes || !playback.RandomOrder || playback.Enabled != (action == "music-play") || a.music.Playing() != wantPlaying {
				t.Fatal("Escape reverted an immediate action", volumes, playback, a.music.Playing(), wantPlaying)
			}
			if len(source.loads) != loads || len(device.events) != events || len(a.MusicRequestLog()) != requests {
				t.Fatal("Escape or reopening issued playback", source.loads, device.events, a.MusicRequestLog())
			}
		})
	}
}
