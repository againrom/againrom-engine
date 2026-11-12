package ui

import "github.com/hajimehoshi/ebiten/v2"

type missionKeyAction uint8

const (
	missionKeyPatrol missionKeyAction = iota
	missionKeyGrab
	missionKeyPickup
)

type missionKeyBinding struct {
	name   string
	key    ebiten.Key
	action missionKeyAction
	noCtrl bool
}

var missionKeyProfiles = map[string]missionKeyAction{"pickup-p": missionKeyPickup}

// SetMissionKeys selects the edition's pickup and patrol bindings.
func (a *App) SetMissionKeys(name string) { a.missionKeys = missionKeyProfiles[name] }

func missionBindings(p missionKeyAction) [2]missionKeyBinding {
	return [2]missionKeyBinding{
		{"f", ebiten.KeyF, missionKeyGrab, true},
		{"p", ebiten.KeyP, p, p == missionKeyPickup},
	}
}

func (in *appInput) readMissionKeys(p missionKeyAction, pressed func(ebiten.Key) bool, ctrl bool) {
	for _, b := range missionBindings(p) {
		if !pressed(b.key) || b.noCtrl && ctrl {
			continue
		}
		switch b.action {
		case missionKeyPatrol:
			in.Patrol = true
		case missionKeyGrab:
			in.Grab = true
		case missionKeyPickup:
			in.Grab, in.GrabReply = true, true
		}
	}
}

func (in *appInput) missionKeyNamed(p missionKeyAction, name string) {
	for _, b := range missionBindings(p) {
		if b.name == name {
			in.readMissionKeys(p, onlyKey(b.key), in.Viewer.Ctrl)
			return
		}
	}
}
