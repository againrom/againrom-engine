package game

import (
	"fmt"
	"io"

	"againrom/pkg/ui"
)

func (f *FrontEnd) witnessMusicPlayback(output string, report io.Writer) error {
	prefs, err := f.Options.MusicPreferences()
	if err != nil {
		return err
	}
	a := f.App("music playback witness")
	defer a.StopAudio()
	a.SetCutscenes(nil)
	if !prefs.Enabled {
		_, playing := a.MusicPlaybackState()
		if playing != "" {
			return fmt.Errorf("music witness: stopped profile started %s", playing)
		}
		if _, gains, ok := ui.AudioDeviceState(f.MusicPlayer); ok && len(gains) != 0 {
			return fmt.Errorf("music witness: stopped startup created a player")
		}
	}
	fmt.Fprintf(report, "music: startup enabled=%v random=%v\n", prefs.Enabled, prefs.RandomOrder)
	if err := a.OpenMission(f.MissionOpener(111)); err != nil {
		return err
	}
	a.Layout(640, 480)
	if err := a.HeadlessKey("escape"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("sound-options"); err != nil {
		return err
	}
	before := f.live.world.Hash()
	check := func(want string) error {
		candidates, playing := a.MusicPlaybackState()
		if len(candidates) != 12 || playing != want {
			return fmt.Errorf("music witness: candidates=%v playing=%q want=%q", candidates, playing, want)
		}
		if _, gains, ok := ui.AudioDeviceState(f.MusicPlayer); ok {
			count := 0
			if want != "" {
				count = 1
			}
			if len(gains) != count {
				return fmt.Errorf("music witness: device players=%d want=%d", len(gains), count)
			}
		}
		return nil
	}
	_, prior := a.MusicPlaybackState()
	// The fourth visible row is selected by the actual pointer hit test.
	row := ui.SoundTrackPoint(3)
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, row.X, row.Y); err != nil {
			return err
		}
	}
	if err := check(prior); err != nil {
		return err
	}
	play := ui.SoundButtonPoint("play")
	for _, edge := range []string{"press", "release"} {
		if err := a.HeadlessPointer(edge, play.X, play.Y); err != nil {
			return err
		}
	}
	if err := check("B03.wav"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("music-tracks"); err != nil {
		return err
	}
	for _, key := range []string{"end", "up"} {
		if err := a.HeadlessKey(key); err != nil {
			return err
		}
	}
	if err := a.HeadlessGameMenuAction("music-play"); err != nil {
		return err
	}
	if err := check("B10.wav"); err != nil {
		return err
	}
	if err := a.HeadlessGameMenuAction("music-random"); err != nil {
		return err
	}
	if err := check("B10.wav"); err != nil {
		return err
	}
	if output != "" {
		if err := writeMediaFrame(output, "music-tracks", a.GameMenuPanel()); err != nil {
			return err
		}
	}
	if err := a.HeadlessGameMenuAction("music-stop"); err != nil {
		return err
	}
	if err := check(""); err != nil {
		return err
	}
	stored, err := f.Options.MusicPreferences()
	if err != nil || stored.Enabled || stored.RandomOrder == prefs.RandomOrder {
		return fmt.Errorf("music witness: persisted preferences=%+v: %v", stored, err)
	}
	if f.live.world.Hash() != before {
		return fmt.Errorf("music witness: menu changed simulation")
	}
	for _, action := range []string{"page-return", "return"} {
		if err := a.HeadlessGameMenuAction(action); err != nil {
			return err
		}
	}
	if err := a.HeadlessStep(); err != nil {
		return err
	}
	if err := check(""); err != nil {
		return err
	}
	fmt.Fprintln(report, "music: 12 installed tracks; pointer B03 and keyboard B10 played; selection and Random preserve current playback; Stop survives menu exit and persists")
	return nil
}
