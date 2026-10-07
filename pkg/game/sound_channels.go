package game

import (
	"fmt"
	"strconv"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

var soundChannelKeys = [...]string{"MusicVolume", "EffectsVolume", "SpeechVolume"}
var soundChannelVolumes = audio.FullChannelVolumes()

// SetSoundChannelVolumes supplies startup preferences before devices open,
// just like SetSoundOptions supplies the master settings.
func SetSoundChannelVolumes(volumes audio.ChannelVolumes) {
	if volumes.Valid() {
		soundChannelVolumes = volumes
	}
}

func (s OptionsStore) SoundChannelVolumes() (audio.ChannelVolumes, error) {
	volumes := audio.FullChannelVolumes()
	values, err := s.readAll()
	if err != nil {
		return volumes, err
	}
	for channel, key := range soundChannelKeys {
		volume, err := strconv.Atoi(values[key])
		if err == nil && volume >= 0 && volume <= audio.MasterUnit {
			volumes[channel] = volume
		}
	}
	return volumes, nil
}

func (s OptionsStore) SetSoundChannelVolumes(volumes audio.ChannelVolumes) error {
	if !volumes.Valid() {
		return fmt.Errorf("sound channel volume outside [0,%d]", audio.MasterUnit)
	}
	values, err := s.readAll()
	if err != nil {
		return err
	}
	for channel, key := range soundChannelKeys {
		values[key] = strconv.Itoa(volumes[channel])
	}
	return s.writeAll(values)
}

func (f *FrontEnd) setSoundChannel(channel audio.Channel, volume int) error {
	if channel >= audio.ChannelCount || volume < 0 || volume > audio.MasterUnit {
		return fmt.Errorf("invalid sound channel %d volume %d", channel, volume)
	}
	volumes := f.SoundChannels
	volumes[channel] = volume
	if f.Options.Path != "" {
		if err := f.Options.SetSoundChannelVolumes(volumes); err != nil {
			return err
		}
	}
	f.SoundChannels = volumes
	f.applySoundSettings()
	return nil
}

func (f *FrontEnd) applySoundSettings() {
	master := audio.Settings{Master: f.Sound.Volume, Muted: !f.Sound.Enabled}
	if setter, ok := f.SoundPlayer.(gameMenuAudioSetter); ok {
		setter.SetSettings(f.SoundChannels.Settings(audio.EffectsChannel, master))
	}
	if setter, ok := f.SpeechPlayer.(gameMenuAudioSetter); ok {
		setter.SetSettings(f.SoundChannels.Settings(audio.SpeechChannel, master))
	}
	if f.MusicPlayer != nil {
		f.MusicPlayer.SetSettings(f.SoundChannels.Settings(audio.MusicChannel, master))
	}
	if f.AmbientPlayer != nil {
		f.AmbientPlayer.SetSettings(f.SoundChannels.Settings(audio.EffectsChannel, master))
	}
	if f.CutsceneAudioPlayer != nil {
		f.CutsceneAudioPlayer.SetSettings(master)
	}
}

func (f *FrontEnd) wireSoundOptions(a *ui.App) {
	controls := ui.SoundOptionControls{
		Read: func() audio.ChannelVolumes { return f.SoundChannels }, Write: f.setSoundChannel,
		ReadAcknowledgments: func() bool { return !f.acknowledgmentsOff }, WriteAcknowledgments: f.setAcknowledgments,
		Words: ui.DefaultSoundOptionWords(),
	}
	if f.Archives != nil {
		table := LoadTextTable(f.Archives.Containers, DialogsTextPath)
		for i, target := range map[int]*string{7: &controls.Words.Title, 0: &controls.Words.OK,
			16: &controls.Words.Labels[audio.MusicChannel], 17: &controls.Words.Labels[audio.EffectsChannel],
			18: &controls.Words.Labels[audio.SpeechChannel], 165: &controls.Words.Acknowledgments,
			143: &controls.Words.Tracks, 9: &controls.Words.RandomOrder, 12: &controls.Words.Play, 14: &controls.Words.Stop} {
			if word, ok := table.At(i); ok {
				*target = word
			}
		}
	}
	f.wireMusicPreferences(&controls)
	a.SetSoundOptionControls(controls)
}
