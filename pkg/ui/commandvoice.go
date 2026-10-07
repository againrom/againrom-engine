package ui

import (
	"image"
	"slices"
	"time"

	"againrom/pkg/audio"
)

// VoiceGesture names the map gesture whose reply is asked for (VIDEO-067). A
// cast order has none because it asks for no reply.
type VoiceGesture uint8

const (
	VoiceMove VoiceGesture = iota + 1
	VoiceAttack
	VoiceSwarm
	VoicePatrol
	VoiceTown
	VoiceGuard
	VoiceStandGround
	VoiceDefend
	VoiceRetreat
	VoicePickup
)

// SetCommandAcknowledgment supplies optional presentation after command
// delivery: the gesture and the units it reached, once per gesture.
func (v *Viewer) SetCommandAcknowledgment(reply func(VoiceGesture, []uint32, time.Time)) {
	v.commandAcknowledgment = reply
}

// replyStance asks the command reply for a guard or stand ground gesture.
func (v *Viewer) replyStance(g VoiceGesture, ids []uint32, now time.Time) {
	if len(ids) != 0 && v.commandAcknowledgment != nil {
		v.commandAcknowledgment(g, ids, now)
	}
}

// replyRetreat asks the command reply for a retreat gesture, recorded where
// the panel or key is read, before a clock was at hand.
func (v *Viewer) replyRetreat(now time.Time) {
	ids := v.retreatSpoken
	v.retreatSpoken = nil
	if len(ids) != 0 && v.commandAcknowledgment != nil {
		v.commandAcknowledgment(VoiceRetreat, ids, now)
	}
}

// SetSelectionAcknowledgment supplies optional presentation after a player
// selection form: a map click, a marquee, the E key or a group recall. The
// reply receives the units the form selected. Only a plain click and a plain
// marquee are such forms (VIDEO-069).
func (v *Viewer) SetSelectionAcknowledgment(reply func([]uint32, time.Time)) {
	v.selectionAcknowledgment = reply
}

// noteSelected records the units a selection form put into the selection. A
// form that selected nothing leaves an earlier form's units of the frame.
func (v *Viewer) noteSelected(ids []uint32) {
	if len(ids) != 0 {
		v.selectionPicked = ids
	}
}

// replySelection hands the frame's selected units to the selection reply once,
// keeping only those the frame still holds selected: a cancel or a deselection
// later in the same frame silences them.
func (v *Viewer) replySelection(now time.Time) {
	picked := v.selectionPicked
	v.selectionPicked = nil
	var held []uint32
	for _, id := range picked {
		if slices.Contains(v.sel, id) {
			held = append(held, id)
		}
	}
	if len(held) != 0 && v.selectionAcknowledgment != nil {
		v.selectionAcknowledgment(held, now)
	}
}

// PlaySpeechSample uses the map listener and fog policy with a distinct speech
// player. The sample was resolved by the game layer from its installed bank.
func (v *Viewer) PlaySpeechSample(player audio.Player, sample audio.Sample, owner uint32, cell image.Point) {
	v.PlayReplySample(player, sample, "unit-reply", "provided-speech-sample", owner, cell, cell.Mul(256).Add(image.Pt(128, 128)))
}

func (v *Viewer) PlayReplySample(player audio.Player, sample audio.Sample, source, selector string, owner uint32, cell, fine image.Point) {
	if DeliveryOwner(player) != nil && DeliveryOwner(player) == DeliveryOwner(v.speechPlayer) {
		player = v.speechPlayer
	}
	v.placeAndPlay(player, owner, cell, fine, source, selector, audio.SpeechChannel, func() (audio.Sample, bool) { return sample, true })
}
