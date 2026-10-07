package game

import (
	"fmt"
	"strconv"

	"againrom/pkg/audio"
)

// These local master controls are authored preferences, not ROM1 registry
// names. Independent channel percentages are stored by sound_channels.go.
const (
	soundEnabledKey = "SoundEnabled"
	masterVolumeKey = "MasterVolume"
)

// SoundOptions overlays valid saved values on the caller's defaults. Reading
// never creates or repairs a file. A malformed field leaves its own default
// intact without discarding the other field.
func (s OptionsStore) SoundOptions(fallback SoundOptions) (SoundOptions, error) {
	m, err := s.readAll()
	if err != nil {
		return fallback, err
	}
	switch m[soundEnabledKey] {
	case "0":
		fallback.Enabled = false
	case "1":
		fallback.Enabled = true
	}
	if volume, err := strconv.Atoi(m[masterVolumeKey]); err == nil && volume >= 0 && volume <= audio.MasterUnit {
		fallback.Volume = volume
	}
	return fallback, nil
}

// SetSoundOptions writes both fields together and retains every other key.
// An unreadable existing store is an error, not permission to replace its
// other preferences with an empty map.
func (s OptionsStore) SetSoundOptions(sound SoundOptions) error {
	if sound.Volume < 0 || sound.Volume > audio.MasterUnit {
		return fmt.Errorf("master volume %d outside [0,%d]", sound.Volume, audio.MasterUnit)
	}
	m, err := s.readAll()
	if err != nil {
		return err
	}
	m[soundEnabledKey] = "0"
	if sound.Enabled {
		m[soundEnabledKey] = "1"
	}
	m[masterVolumeKey] = strconv.Itoa(sound.Volume)
	return s.writeAll(m)
}
