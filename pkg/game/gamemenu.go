package game

import (
	"sort"

	"againrom/pkg/audio"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// gameMenuContext projects live session state into the pause menu without
// giving the client a simulation handle. It is called at menu open, so a
// script diplomacy change made after map construction is visible immediately.
func gameMenuContext(mw *mapWorld, campaign bool, objective string) ui.GameMenuContext {
	ctx := ui.GameMenuContext{Campaign: campaign, CampaignVictory: campaign, Objective: objective}
	if mw == nil || mw.world == nil {
		return ctx
	}
	ctx.Objective = mw.secondGameObjectives(objective)
	ctx.VictoryAvailable = campaign && mw.mission != nil &&
		mw.mission.delayedVictory && !mw.mission.victoryTaken
	owners := make(map[uint32]struct{})
	for _, entity := range mw.world.Entities() {
		if entity.Owner != 0 && entity.Owner != sim.SelfSlot {
			owners[entity.Owner] = struct{}{}
		}
	}
	slots := make([]uint32, 0, len(owners))
	for slot := range owners {
		slots = append(slots, slot)
	}
	sort.Slice(slots, func(i, j int) bool { return slots[i] < slots[j] })
	relations := mw.world.Relations()
	for _, slot := range slots {
		state := "NEUTRAL"
		switch {
		case relations.Hostile(sim.SelfSlot, slot):
			state = "HOSTILE"
		case relations.Locked(sim.SelfSlot, slot):
			state = "ALLIED"
		}
		ctx.Relations = append(ctx.Relations, ui.GameMenuRelation{Slot: slot, State: state})
	}
	return ctx
}

type gameMenuAudioSetter interface {
	SetSettings(audio.Settings)
}

func (f *FrontEnd) gameMenuSound() (enabled bool, volume int, available bool) {
	if f == nil || (f.SoundPlayer == nil && f.SpeechPlayer == nil && f.MusicPlayer == nil && f.AmbientPlayer == nil && f.CutsceneAudioPlayer == nil) {
		return false, 0, false
	}
	return f.Sound.Enabled, f.Sound.Volume, true
}

func (f *FrontEnd) setGameMenuSound(enabled bool, volume int) error {
	if f == nil {
		return nil
	}
	if volume < 0 {
		volume = 0
	}
	if volume > audio.MasterUnit {
		volume = audio.MasterUnit
	}
	sound := SoundOptions{Enabled: enabled, Volume: volume}
	if f.Options.Path != "" {
		if err := f.Options.SetSoundOptions(sound); err != nil {
			return err
		}
	}
	f.Sound = sound
	f.applySoundSettings()
	return nil
}
