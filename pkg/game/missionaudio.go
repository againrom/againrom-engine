package game

import (
	"againrom/pkg/random"
	"againrom/pkg/ui"
)

// missionAudio is everything a mission screen's construction asks of sound.
// The mission-entry sequence calls these four operations and names no player,
// bank, channel or seed, so a different audio service is a different
// implementation of this interface and no change to the sequence.
//
// A nil player or bank behind an implementation is a lawful silent state: no
// operation reports a failure.
type missionAudio interface {
	// attach hands a freshly decoded viewer the scoped players and banks it
	// plays through. It runs before any step that can fail, so release has
	// something to drop on a refused entry.
	attach(v *ui.Viewer)
	// release drops the audio a viewer attached, for a viewer that is not
	// going to be adopted or has been replaced.
	release(v *ui.Viewer)
	// wireReplies connects the unit command and selection acknowledgments of a
	// built world to its viewer.
	wireReplies(mw *mapWorld)
	// wireEffects connects the world's swing, spell and semantic effect
	// sounds to its viewer.
	wireEffects(mw *mapWorld, v *ui.Viewer)
}

// runtimeMissionAudio is the production audio service: the process's opened
// players and the install's sound bank, read at the moment each operation
// runs so a toggle made while a mission is open still takes effect.
type runtimeMissionAudio struct {
	rt *RuntimeServices
	in *InstallResources
}

// runtimeAudio is the production missionAudio over this front end's own
// services and install.
func (f *FrontEnd) runtimeAudio() runtimeMissionAudio {
	return runtimeMissionAudio{rt: &f.RuntimeServices, in: &f.InstallResources}
}

func (a runtimeMissionAudio) attach(v *ui.Viewer) {
	v.SetScopedAudio(a.rt.SoundPlayer, a.rt.SpeechPlayer, a.in.SoundBank, a.rt.AmbientPlayer, a.rt.randomService().Stream(random.AmbientBirds))
}

func (a runtimeMissionAudio) release(v *ui.Viewer) {
	v.DestroyAudio()
}

func (a runtimeMissionAudio) wireReplies(mw *mapWorld) {
	command, selection := a.unitReplies(mw, newViewerVoiceDraw(a.rt.randomService().Stream(random.CommandVoice)))
	mw.view.SetCommandAcknowledgment(command)
	mw.view.SetSelectionAcknowledgment(selection)
}

func (a runtimeMissionAudio) wireEffects(mw *mapWorld, v *ui.Viewer) {
	mw.setSwingSound(a.in.SoundClasses, v.PlaySlotAt)
	mw.semanticSound = v.PlayEffectAtFine
}

// releaseWorldAudio drops the audio of a map that is no longer live. A nil map
// or one with no viewer holds none.
func releaseWorldAudio(a missionAudio, mw *mapWorld) {
	if mw != nil && mw.view != nil {
		a.release(mw.view)
	}
}
