package game

import (
	"fmt"
	"image"
	"time"

	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Reader thresholds (ANIM-119).
const (
	commandVoiceDelay   = 3 * time.Second
	selectionVoiceDelay = 2 * time.Second
	// voiceDrawRange is the span of the original's generator, 0 to 32767.
	voiceDrawRange = 0x7fff
)

// voiceDraw returns one value of 0 to voiceDrawRange from the command-voice
// stream; in original mode that is the shared stream (DIV-2038).
type voiceDraw func() int

// newViewerVoiceDraw is replaced by tests to script the draws.
var newViewerVoiceDraw = newVoiceDraw

func newVoiceDraw(st *random.Stream) voiceDraw { return st.Raw }

// speakerOf chooses one living member with a voice bank from the first
// non-empty tier by the index draw*n/32767 (VIDEO-068). A draw of 32767 names
// the slot one past the last member and picks nobody (DIV-2040).
func (mw *mapWorld) speakerOf(ids []uint32, draw voiceDraw) (sim.Entity, string, bool) {
	var tiers [3][]sim.Entity
	for _, id := range ids {
		e, ok := mw.world.Entity(sim.EntityID(id))
		if !ok || e.HP <= 0 {
			continue
		}
		if tier := mw.voiceTier(e); tier >= 0 {
			tiers[tier] = append(tiers[tier], e)
		}
	}
	for _, members := range tiers {
		if len(members) == 0 {
			continue
		}
		at := draw() * len(members) / voiceDrawRange
		if at >= len(members) {
			return sim.Entity{}, "", false
		}
		return members[at], mw.voiceBank(members[at]), true
	}
	return sim.Entity{}, "", false
}

// commandRecording is the leaf a gesture asks of the speaker's bank.
func commandRecording(gesture ui.VoiceGesture, bank string, draw voiceDraw) string {
	switch gesture {
	case ui.VoiceGuard, ui.VoiceStandGround, ui.VoiceDefend:
		return "defend"
	case ui.VoiceRetreat:
		return "retreat"
	case ui.VoicePickup:
		return "idle"
	}
	r := draw() >> 13
	peasant := bank == "m_peasant" || bank == "f_peasant" // ANIM-119, DIV-1293
	if r < 3 || peasant {
		return fmt.Sprintf("command%d", r%3+1)
	}
	return "defend"
}

// unitReplies returns one viewer's command and selection replies. Each asks
// the chooser once, then one reader of the speaker's bank (ANIM-119); a
// speaker inside its threshold stays silent and no second speaker is tried.
func (a runtimeMissionAudio) unitReplies(mw *mapWorld, draw voiceDraw) (command func(ui.VoiceGesture, []uint32, time.Time), selection func([]uint32, time.Time)) {
	speak := func(speaker sim.Entity, bank, leaf, source string) {
		if sample, ok := a.in.SoundBank.namedSample(bank + "/" + leaf + ".wav"); ok {
			mw.view.PlayReplySample(a.rt.SpeechPlayer, sample, source, "voice:"+bank+"/"+leaf+".wav", speaker.Owner, image.Pt(int(speaker.X), int(speaker.Y)), mw.actorSoundFine(speaker))
		}
	}
	ready := func() bool {
		return !a.rt.acknowledgmentsOff && a.rt.SpeechPlayer != nil && a.in.SoundBank != nil
	}
	command = func(gesture ui.VoiceGesture, ids []uint32, now time.Time) {
		if !ready() {
			return
		}
		speaker, bank, ok := mw.speakerOf(ids, draw)
		if !ok {
			return
		}
		// A speaker drawn as moving is silent for move and swarm (VIDEO-067).
		if (gesture == ui.VoiceMove || gesture == ui.VoiceSwarm) && mw.drawnMoving[speaker.ID] {
			return
		}
		if !mw.view.ClaimVoice(uint32(speaker.ID), now, commandVoiceDelay) {
			return
		}
		speak(speaker, bank, commandRecording(gesture, bank, draw), "unit-command")
	}
	selection = func(ids []uint32, now time.Time) {
		if !ready() || len(ids) == 0 {
			return
		}
		// The primary object must be the player's own (VIDEO-069).
		if primary, ok := mw.world.Entity(sim.EntityID(ids[0])); !ok || primary.Owner != sim.SelfSlot {
			return
		}
		speaker, bank, ok := mw.speakerOf(ids, draw)
		if !ok || !mw.view.ClaimVoice(uint32(speaker.ID), now, selectionVoiceDelay) {
			return
		}
		speak(speaker, bank, fmt.Sprintf("select%d", draw()>>14+1), "unit-selection")
	}
	return command, selection
}
