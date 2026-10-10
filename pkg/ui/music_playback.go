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

// SetPreferences never starts a stream. Play owns the chosen row; SetScene
// owns automatic context startup. A random-order change preserves the track
// already playing and its decoder position.
func (m *MusicController) SetPreferences(p MusicPreferences) {
	if m == nil {
		return
	}
	current, changed := m.Playing(), m.preferences.RandomOrder != p.RandomOrder
	m.preferences = p
	if !p.Enabled && m.active {
		if m.device != nil {
			m.device.Stop()
		}
		m.active = false
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
	if m.Playing() == name {
		return true
	}
	if m.active && m.device != nil {
		m.device.Stop()
	}
	m.active = false
	for i, candidate := range m.order {
		if candidate == name {
			m.position = i
			return m.startName(name)
		}
	}
	return false
}

func (m *MusicController) startName(name string) bool {
	if m.device == nil || m.source == nil || !m.preferences.Enabled {
		return false
	}
	track, ok := m.source.Track(name)
	if !ok || track.Rate != audio.DeviceRate || len(track.StereoPCM) < 4 || len(track.StereoPCM)%4 != 0 {
		return false
	}
	m.device.Start(track)
	m.active, m.paused, m.resumeStarts = true, false, false
	return true
}
