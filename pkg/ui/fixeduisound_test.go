package ui

import (
	"reflect"
	"testing"

	"againrom/pkg/audio"
)

func fixedUISoundBank() stubBank {
	b := stubBank{}
	for _, slot := range []UISoundSlot{
		UISoundCampaignPanel, UISoundCommonControl, UISoundBookToggle,
		UISoundCommandPanel, UISoundMissionComplete, UISoundMissionFailed,
		UISoundOptionsTest,
	} {
		b[int(slot)] = soundSample(int(slot))
	}
	return b
}

func playedUISlots(rec *recordingPlayer) []int {
	out := make([]int, len(rec.plays))
	for i, play := range rec.plays {
		out[i] = int(play.Sample.PCM[0])
	}
	return out
}

func TestFixedUISoundSelectorsResolveCenteredThroughTheExistingBank(t *testing.T) {
	rec := &recordingPlayer{}
	want := []int{1, 2, 7, 8, 14, 16, 100}
	for _, slot := range []UISoundSlot{
		UISoundCampaignPanel, UISoundCommonControl, UISoundBookToggle,
		UISoundCommandPanel, UISoundMissionComplete, UISoundMissionFailed,
		UISoundOptionsTest,
	} {
		playUISound(rec, fixedUISoundBank(), slot)
	}
	if got := playedUISlots(rec); !reflect.DeepEqual(got, want) {
		t.Fatalf("played slots = %v, want %v", got, want)
	}
	full := audio.Placement{Left: audio.GainUnit, Right: audio.GainUnit}
	for i, play := range rec.plays {
		if play.Placement != full {
			t.Errorf("play %d placement = %+v, want centred %+v", i, play.Placement, full)
		}
	}
}

func TestCommandDispatchAndBookChildMutationsUseTheirFixedSelectors(t *testing.T) {
	v := &Viewer{}
	rec := &recordingPlayer{}
	v.SetAudio(rec, fixedUISoundBank())

	// 0x40c is a dispatch cue: even a refused cell still made the dispatch.
	v.pressCommandPanelCell(commandCellAttack, false)
	// Child 2/3 cues follow the two mutations. Neither other HUD flag is one.
	v.toggleHudPanel(hudPanelPack)
	v.toggleHudPanel(hudPanelBook)
	v.toggleHudPanel(hudPanelDoll)
	v.toggleHudPanel(hudPanelWorn)

	if got, want := playedUISlots(rec), []int{8, 7, 7}; !reflect.DeepEqual(got, want) {
		t.Fatalf("played slots = %v, want %v", got, want)
	}
}

func TestCommonActivationPrecedesActionAndOptionsActionSevenTestsSword(t *testing.T) {
	a := newTestApp(t, appRows(1), okLoader(t))
	rec := &recordingPlayer{}
	a.SetAudio(rec, fixedUISoundBank())

	calledAfter := -1
	a.activatePicker(a.flow.picker, func() { calledAfter = len(rec.plays) })
	if calledAfter != 1 || !reflect.DeepEqual(playedUISlots(rec), []int{2}) {
		t.Fatalf("common activation: plays=%v, action observed %d prior plays", playedUISlots(rec), calledAfter)
	}

	rec.plays = nil
	a.flow.screen = ScreenGameMenu
	a.flow.menuSurface = gameMenuMission
	a.flow.menuSound = func() (bool, int, bool) { return true, 50, true }
	a.flow.setMenuSound = func(bool, int) error { return nil }
	a.flow.rebuildGameMenu(gameMenuSoundOptionsPage, 0)
	if !selectGameMenuAction(a.flow, gameMenuTestSound) {
		t.Fatal("sound-options page has no action 7 test row")
	}
	a.chooseGameMenu()
	if got, want := playedUISlots(rec), []int{2, 100}; !reflect.DeepEqual(got, want) {
		t.Fatalf("options test played %v, want common click then slot 100: %v", got, want)
	}
}
