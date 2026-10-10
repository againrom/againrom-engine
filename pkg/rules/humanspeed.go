package rules

// HumanOverloadFloor is the lowest speed the overload penalty leaves.
const HumanOverloadFloor int32 = 6

// HumanSpeed is the Human derive's speed word from its unencumbered base and
// its signed speed modifier (SAV-1116, HERO-SPEED-008, MOVE-RATE-053). When
// load reaches capacity it subtracts load/capacity and floors at
// HumanOverloadFloor, then adds the modifier as a word. A negative sum keeps
// the speed and clears the modifier. ok is false only for an overloaded
// actor with zero capacity, the division the derive cannot perform.
//
// Load and capacity are compared and divided in 32 bits; no claim states the
// width of the original's compare. A stored Human passes its signed words.
func HumanSpeed(base, modifier int16, load, capacity int32) (speed, kept int16, ok bool) {
	v := int32(base)
	if load >= capacity {
		if capacity == 0 {
			return base, modifier, false
		}
		v = max(v-load/capacity, HumanOverloadFloor)
	}
	speed = int16(uint16(v) + uint16(modifier))
	if speed < 0 {
		modifier = 0
	}
	return speed, modifier, true
}

// NativeHumanSpeed is HumanSpeed for a native Human, which holds the
// unencumbered sum speed with the signed modifier inside it. A speed at or
// below zero has no rate and is returned whole, and so is an actor with no
// stated capacity.
func NativeHumanSpeed(speed, modifier, load, capacity int32) (word, kept int32) {
	if speed <= 0 || capacity <= 0 {
		return speed, modifier
	}
	w, m, _ := HumanSpeed(int16(speed-modifier), int16(modifier), load, capacity)
	return int32(w), int32(m)
}
