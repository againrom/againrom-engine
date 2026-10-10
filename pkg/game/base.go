package game

import (
	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// Base is the profile the front end's install was detected as. A front end
// assembled by hand has none, which states no limit.
func (f *FrontEnd) Base() base.Match {
	if f == nil || f.Archives == nil {
		return base.Match{}
	}
	return f.Archives.Base
}

// NewGameMission is the mission a new game opens: the base profile's own
// first mission where it names one, otherwise the installed campaign's first
// mission, otherwise the profile default.
func (f *FrontEnd) NewGameMission() int {
	p := f.Base().Profile
	if p.Limits.FirstMission > 0 {
		return p.Limits.FirstMission
	}
	if f != nil {
		if n, ok := f.Campaign.Value().FirstMission(); ok {
			return n
		}
	}
	return p.Mission()
}

// DirectNewGame is the new game of a base that ships no character generation:
// mission n opened on a fresh town with the default party at normal difficulty.
// Obtaining the opener commits nothing, as NewGameOpener's own.
func (f *FrontEnd) DirectNewGame(n int) ui.MapOpener {
	return func() (*ui.Viewer, ui.MapTick, ui.MapOrder, ui.MapCadence, ui.MapAffect, ui.MapAdvance, ui.MapAttack, ui.MapGrab, ui.MapStance, ui.MapMarch, error) {
		open, err := f.prepareNewGameWith(n, 0, MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table))
		if err != nil {
			return nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, err
		}
		return open()
	}
}

// profile is the profile the install was detected as; no install is the zero
// profile, the first game's.
func (in *InstallResources) profile() base.Profile {
	if in == nil || in.Archives == nil {
		return base.Profile{}
	}
	return in.Archives.Base.Profile
}

// directNewGame reports whether NEW GAME opens the profile's first mission
// without generation.
func (f *FrontEnd) directNewGame() bool {
	return f.Base().Profile.Limits.NoCharacterGeneration
}

// BaseLines are the lines -check prints about the base: the detected profile
// and each limit the profile states.
func (f *FrontEnd) BaseLines() []string {
	m := f.Base()
	if !m.Profile.Known() {
		return nil
	}
	lines := []string{"againrom: base " + m.String()}
	l := m.Profile.Limits
	if l.NoCharacterGeneration {
		lines = append(lines, "againrom: base limit: "+f.campaign().newGameLimit(m.Profile))
	}
	if l.OriginalSaveRefusal != "" {
		lines = append(lines, "againrom: base limit: loading an original save is refused: "+l.OriginalSaveRefusal)
	}
	for _, n := range l.Notes {
		lines = append(lines, "againrom: base limit: "+n)
	}
	return lines
}
