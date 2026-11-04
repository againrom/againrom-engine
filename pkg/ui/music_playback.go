package ui

import "againrom/pkg/audio"

// MusicPreferences are process presentation preferences, independent of
// channel/master gain and of the deterministic mission state.
type MusicPreferences struct {
	Enabled, RandomOrder bool
}

func DefaultMusicPreferences() MusicPreferences { return MusicPreferences{true, true} }

func (m *MusicController) Candidates() []string {
	if m == nil || !m.set {
		return nil
	}
	return m.requestTracks(m.request)
}

func (m *MusicController) Playing() string {
	if m == nil || !m.active || m.position < 0 || m.position >= len(m.order) {
		return ""
	}
	return m.order[m.position]
}

// SetPreferences applies music off and Random Order without starting a new
// list. A held stop reselects its entry; Play owns manual selection.
func (m *MusicController) SetPreferences(p MusicPreferences) {
	if m == nil {
		return
	}
	current, changed := m.Playing(), m.preferences.RandomOrder != p.RandomOrder
	if !p.Enabled && m.active {
		if m.desc != nil && m.desc.Fade != nil {
			if m.fade == nil {
				m.beginFade()
			} else {
				m.pause()
			}
		} else if m.device != nil {
			m.device.Stop()
			m.active = false
		}
	}
	m.preferences = p
	if changed {
		m.hold = false
	}
	if changed && m.set && (m.desc == nil || m.desc.List == MusicListShuffled) {
		m.order, m.position = m.shuffle(m.Candidates()), 0
		for i, name := range m.order {
			if name == current {
				m.position = i
				break
			}
		}
	}
}

// Play accepts an index in the visible candidate bank, not its permutation.
// A missing manual selection stays silent rather than substituting a track.
func (m *MusicController) Play(index int) bool {
	names := m.Candidates()
	if m == nil || index < 0 || index >= len(names) || !m.preferences.Enabled {
		return false
	}
	name := names[index]
	m.resetVolume()
	if m.Playing() == name {
		return true
	}
	if m.active && m.device != nil {
		m.stopSelection()
	}
	m.active = false
	return m.selectIndex(index, true)
}

func (m *MusicController) stopSelection() {
	if m.hold {
		m.pause()
	} else if m.device != nil {
		m.device.Stop()
	}
}

func (m *MusicController) selectIndex(index int, playing bool) bool {
	names := m.Candidates()
	if index < 0 || index >= len(names) {
		return false
	}
	for i, name := range m.order {
		if name == names[index] {
			m.position = i
			return m.selectName(name, playing)
		}
	}
	return false
}

func (m *MusicController) selectName(name string, playing bool) bool {
	if m.device == nil || m.source == nil {
		return false
	}
	track, ok := m.source.Track(name)
	if !ok || track.Rate != audio.DeviceRate || len(track.StereoPCM) < 4 || len(track.StereoPCM)%4 != 0 {
		return false
	}
	if d, ok := m.device.(interface{ SelectMusic(audio.Track, bool) }); ok {
		d.SelectMusic(track, playing)
	} else {
		m.device.Start(track)
		if !playing {
			if d, ok := m.device.(MusicPauser); ok {
				d.Pause()
			} else {
				m.device.Stop()
			}
		}
	}
	m.active, m.paused, m.resumeStarts = playing, !playing, !playing
	return true
}
