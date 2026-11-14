package game

import (
	"strings"

	"againrom/pkg/ui"
)

// MissionKeys is the player's layout of the game with mission key profile profile.
func (s OptionsStore) MissionKeys(profile string) (ui.PlayerMissionKeys, error) {
	m, err := s.readAll()
	if err != nil {
		return ui.PlayerMissionKeys{}, err
	}
	return ui.PlayerMissionKeysFrom(profile, m), nil
}

// SetMissionKeys replaces the game's layout; the other game's stays.
func (s OptionsStore) SetMissionKeys(profile string, keys ui.PlayerMissionKeys) error {
	m, err := s.readAll()
	if err != nil {
		return err
	}
	prefix := ui.MissionKeyPrefix(profile)
	for k := range m {
		if strings.HasPrefix(k, prefix) {
			delete(m, k)
		}
	}
	for k, v := range keys.Entries(profile) {
		m[k] = v
	}
	return s.writeAll(m)
}

func (p *PersistenceContext) wireMissionKeys(a *ui.App, profile string) {
	keys, _ := p.Options.MissionKeys(profile)
	a.SetPlayerMissionKeys(keys, func(next ui.PlayerMissionKeys) error {
		if p.Options.Path == "" {
			return nil
		}
		return p.Options.SetMissionKeys(profile, next)
	})
}
