package audio

// Channel identifies independently adjustable playback. Movie audio retains
// its existing master-only policy because its tracks are not split here.
type Channel uint8

const (
	MusicChannel Channel = iota
	EffectsChannel
	SpeechChannel
	ChannelCount
)

type ChannelVolumes [ChannelCount]int

func FullChannelVolumes() ChannelVolumes { return ChannelVolumes{MasterUnit, MasterUnit, MasterUnit} }

func (v ChannelVolumes) Valid() bool {
	for _, volume := range v {
		if volume < 0 || volume > MasterUnit {
			return false
		}
	}
	return true
}

// Settings combines the local master and one channel's percentage. The
// original volume arithmetic is not established by these authored percentages.
func (v ChannelVolumes) Settings(channel Channel, master Settings) Settings {
	if channel >= ChannelCount {
		return Settings{Muted: true}
	}
	master.Master = max(0, min(MasterUnit, master.Master)) * max(0, min(MasterUnit, v[channel])) / MasterUnit
	return master
}
