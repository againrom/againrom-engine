package game

import "againrom/pkg/sim"

// dollChange is what a gesture did to a figure's worn slots, as sound.
type dollChange uint8

const (
	dollUnchanged dollChange = iota
	dollPutOn
	dollTakeOff
)

// dollMixedRule decides a gesture that both fills and empties slots.
type dollMixedRule uint8

const (
	// dollAnyGain: a slot that gained or changed an item decides put-on,
	// whatever else came off.
	dollAnyGain dollMixedRule = iota
	// dollNetCount: fewer worn items after the gesture decides take-off; more,
	// or the same number in other slots, decides put-on.
	dollNetCount
)

// dollSoundSource is where a surface's sound comes from: a room event of the
// shop's sound table, or a named sfx.res member. Empty stays silent.
type dollSoundSource struct{ event, member string }

// dollSoundProfile is one surface's sound decision data: its rule for a
// mixed gesture and the source of the put-on and take-off sounds.
type dollSoundProfile struct {
	mixed   dollMixedRule
	on, off dollSoundSource
}

// shopDollSounds is the shop's profile for both games: the room events the
// town sound tables bind (the first game binds none and stays silent).
var shopDollSounds = dollSoundProfile{
	mixed: dollNetCount,
	on:    dollSoundSource{event: "doll-on"},
	off:   dollSoundSource{event: "doll-off"},
}

// decideDollChange is the one decision over a gesture's worn slots before
// and after it.
func decideDollChange(before, after [sim.EquipSlots]sim.ItemInstance, rule dollMixedRule) dollChange {
	var gained, lost bool
	var had, has int
	for i := range before {
		if !before[i].Empty() {
			had++
		}
		if !after[i].Empty() {
			has++
		}
		if itemStateEqual(before[i], after[i]) {
			continue
		}
		if after[i].Empty() {
			lost = true
		} else {
			gained = true
		}
	}
	switch rule {
	case dollNetCount:
		switch {
		case has < had:
			return dollTakeOff
		case has > had || gained || lost:
			return dollPutOn
		}
	default:
		switch {
		case gained:
			return dollPutOn
		case lost:
			return dollTakeOff
		}
	}
	return dollUnchanged
}

// play requests the sound the profile states for the change: the room event
// through playEvent, the named member through playMember.
func (p dollSoundProfile) play(change dollChange, playEvent, playMember func(string)) {
	var src dollSoundSource
	switch change {
	case dollPutOn:
		src = p.on
	case dollTakeOff:
		src = p.off
	default:
		return
	}
	if src.event != "" {
		playEvent(src.event)
	}
	if src.member != "" {
		playMember(src.member)
	}
}
