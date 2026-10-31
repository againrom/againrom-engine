package game

import (
	"image"
	"time"

	"againrom/pkg/random"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Reader thresholds (ANIM-119, R2-ENGINE-719).
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

// unitVoiceRules are one game's command and selection voices, read by the one
// selector (unitVoice), chooser (speakerOf) and reader (commandRecording).
type unitVoiceRules struct {
	// dir is the sound-archive directory the voices lie under.
	dir string
	// voices are the voices by sex and arm; heroArm gives a hero-shaped
	// person who is no mage the hero arm.
	voices  [2][4]string
	heroArm bool
	// named are the voices of identity words that have their own, by
	// female + 2 × mage.
	named map[uint16][4]string
	// hires are the identity words, lo to hi, that put a unit no person
	// figure draws in the armed tier; zero puts none.
	hires [2]uint16
	// selection is the selection reply's recording, gestures each order
	// gesture's; a gesture the map has no entry for asks nobody.
	selection voiceRecording
	gestures  map[ui.VoiceGesture]voiceRecording
}

// voiceRecording is the recordings a reply plays one of.
type voiceRecording struct {
	// leaves are the recordings under the voice's directory, without .wav;
	// plain replace them for a voice of the arm no weapon shows.
	leaves, plain []string
	// drawn takes one draw r and plays leaf floor(r × n / 32768) of n.
	drawn bool
	// counted keeps the first n leaves, n the number the archive holds.
	counted bool
}

// commandLeaves are the first game's command reader (ANIM-119): a peasant
// bank never plays defend (DIV-1293).
var commandLeaves = voiceRecording{
	leaves: []string{"command1", "command2", "command3", "defend"},
	plain:  []string{"command1", "command2", "command3", "command1"},
	drawn:  true,
}

// firstGameVoices: VIDEO-067, VIDEO-069, VIDEO-070, ANIM-094, ANIM-119.
var firstGameVoices = unitVoiceRules{
	voices:    humanVoiceBanks,
	heroArm:   true,
	selection: voiceRecording{leaves: []string{"select1", "select2"}, drawn: true},
	gestures: map[ui.VoiceGesture]voiceRecording{
		ui.VoiceMove: commandLeaves, ui.VoiceAttack: commandLeaves, ui.VoiceSwarm: commandLeaves,
		ui.VoicePatrol: commandLeaves, ui.VoiceTown: commandLeaves,
		ui.VoiceGuard: {leaves: []string{"defend"}}, ui.VoiceStandGround: {leaves: []string{"defend"}},
		ui.VoiceDefend: {leaves: []string{"defend"}}, ui.VoiceRetreat: {leaves: []string{"retreat"}},
		ui.VoicePickup: {leaves: []string{"idle"}},
	},
}

// numberedVoice is a second-game gesture of up to four numbered files, as
// many as sfx.res holds (R2-ENGINE-718, R2-ENGINE-719).
func numberedVoice(gesture string) voiceRecording {
	return voiceRecording{leaves: []string{gesture + "/1", gesture + "/2", gesture + "/3", gesture + "/4"}, drawn: true, counted: true}
}

func sameVoice(name string) [4]string { return [4]string{name, name, name, name} }

// secondGameVoices: R2-ENGINE-717 (the selection and the twelve order
// senders; patrol, town and a cast have none), R2-ENGINE-718 (the voices by
// identity word), R2-ENGINE-719 (hires 1..20 in the armed tier), R2-ASSET-185;
// DIV-4347, DIV-4348.
var secondGameVoices = unitVoiceRules{
	dir:    "characters/",
	voices: [2][4]string{{"m_mage", "", "m_fighter", "m_peasant"}, {"f_mage", "", "f_fighter", "f_peasant"}},
	named: map[uint16][4]string{
		1: sameVoice("xbow"), 2: sameVoice("knight"), 3: sameVoice("druid"),
		4: sameVoice("troll"), 5: sameVoice("orc"), 6: sameVoice("mage"),
		21: {"aldor", "sagita", "hildarius", "galinel"}, 22: sameVoice("igles"), 23: sameVoice("dina"),
	},
	hires:     [2]uint16{1, 20},
	selection: numberedVoice("select"),
	gestures: map[ui.VoiceGesture]voiceRecording{
		ui.VoiceMove: numberedVoice("move"), ui.VoiceAttack: numberedVoice("attack"), ui.VoiceSwarm: numberedVoice("swarm"),
		ui.VoiceGuard: {leaves: []string{"defend/1"}}, ui.VoiceStandGround: {leaves: []string{"defend/1"}},
		ui.VoiceDefend: {leaves: []string{"defend/1"}}, ui.VoiceRetreat: {leaves: []string{"retreat/1"}},
		ui.VoicePickup: {leaves: []string{"pickup/1"}},
	},
}

// unitVoiceRules are the voices of the game the mission's table belongs to.
func (mw *mapWorld) unitVoiceRules() *unitVoiceRules { return &mw.files().unitVoices }

// speakerOf chooses one living member of the first non-empty tier by the
// index draw*n/32767 (VIDEO-068, R2-ENGINE-719). A draw of 32767 names the
// slot one past the last member and picks nobody (DIV-2040, DIV-4347).
func (mw *mapWorld) speakerOf(ids []uint32, draw voiceDraw) (sim.Entity, unitVoice, bool) {
	rules := mw.unitVoiceRules()
	var tiers [3][]sim.Entity
	var voices [3][]unitVoice
	for _, id := range ids {
		e, ok := mw.world.Entity(sim.EntityID(id))
		if !ok || !e.Living() {
			continue
		}
		if voice := mw.unitVoice(e, rules, true); voice.tier >= 0 {
			tiers[voice.tier] = append(tiers[voice.tier], e)
			voices[voice.tier] = append(voices[voice.tier], voice)
		}
	}
	for tier, members := range tiers {
		if len(members) == 0 {
			continue
		}
		at := draw() * len(members) / voiceDrawRange
		if at >= len(members) {
			return sim.Entity{}, unitVoice{}, false
		}
		return members[at], voices[tier][at], true
	}
	return sim.Entity{}, unitVoice{}, false
}

// commandRecording is the archive path of the recording rec plays in voice,
// "" for none. A counted recording with no file held plays nothing and draws
// nothing; a drawn one draws once even for one file (R2-ENGINE-719).
func (r *unitVoiceRules) commandRecording(rec voiceRecording, voice unitVoice, held func(string) bool, draw voiceDraw) string {
	leaves := rec.leaves
	if voice.plain && rec.plain != nil {
		leaves = rec.plain
	}
	path := func(leaf string) string { return r.dir + voice.name + "/" + leaf + ".wav" }
	if rec.counted {
		n := 0
		for _, leaf := range leaves {
			if held(path(leaf)) {
				n++
			}
		}
		leaves = leaves[:n]
	}
	if len(leaves) == 0 {
		return ""
	}
	at := 0
	if rec.drawn {
		at = draw() * len(leaves) / (voiceDrawRange + 1)
	}
	return path(leaves[at])
}

// unitReplies returns one viewer's command and selection replies. Each asks
// the chooser once, then one reader of the speaker's voice (ANIM-119,
// R2-ENGINE-719); a speaker inside its threshold stays silent and no second
// speaker is tried. The stamp is stored before the recording is looked up.
func (a runtimeMissionAudio) unitReplies(mw *mapWorld, draw voiceDraw) (command func(ui.VoiceGesture, []uint32, time.Time), selection func([]uint32, time.Time)) {
	speak := func(speaker sim.Entity, path, source string) {
		if sample, ok := a.in.SoundBank.namedSample(path); ok {
			mw.view.PlayReplySample(a.rt.speech.player, sample, source, "voice:"+path, speaker.Owner, image.Pt(int(speaker.X), int(speaker.Y)), mw.actorSoundFine(speaker))
		}
	}
	ready := func() bool {
		return !a.rt.speech.acknowledgmentsOff && a.rt.speech.player != nil && a.in.SoundBank != nil
	}
	held := func(path string) bool { return a.in.SoundBank.holdsNamed(path) }
	command = func(gesture ui.VoiceGesture, ids []uint32, now time.Time) {
		if !ready() {
			return
		}
		rules := mw.unitVoiceRules()
		rec, voiced := rules.gestures[gesture]
		if !voiced {
			return
		}
		speaker, voice, ok := mw.speakerOf(ids, draw)
		if !ok {
			return
		}
		// A speaker drawn as moving is silent for move and swarm (VIDEO-067,
		// R2-ENGINE-717, DIV-2042).
		if (gesture == ui.VoiceMove || gesture == ui.VoiceSwarm) && mw.drawnMoving[speaker.ID] {
			return
		}
		if !mw.view.ClaimVoice(uint32(speaker.ID), now, commandVoiceDelay) {
			return
		}
		if path := rules.commandRecording(rec, voice, held, draw); path != "" {
			speak(speaker, path, "unit-command")
		}
	}
	selection = func(ids []uint32, now time.Time) {
		if !ready() || len(ids) == 0 {
			return
		}
		// The primary object must be the player's own (VIDEO-069,
		// R2-ENGINE-717).
		if primary, ok := mw.world.Entity(sim.EntityID(ids[0])); !ok || primary.Owner != sim.SelfSlot {
			return
		}
		speaker, voice, ok := mw.speakerOf(ids, draw)
		if !ok || !mw.view.ClaimVoice(uint32(speaker.ID), now, selectionVoiceDelay) {
			return
		}
		rules := mw.unitVoiceRules()
		if path := rules.commandRecording(rules.selection, voice, held, draw); path != "" {
			speak(speaker, path, "unit-selection")
		}
	}
	return command, selection
}
