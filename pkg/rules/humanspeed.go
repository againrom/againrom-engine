package rules

// HumanOverloadFloor is the lowest speed the overload penalty leaves.
const HumanOverloadFloor int32 = 6

// HumanSpeed is the Human derive's speed word from its unencumbered base and
// its signed speed modifier (SAV-1116, HERO-SPEED-008, MOVE-RATE-053). When
// load reaches capacity it subtracts load/capacity and floors at
// HumanOverloadFloor, then adds the modifier as a word. A negative sum keeps
// the speed and clears the modifier. ok is false only for an overloaded
// actor with zero capacity, the division the derive cannot perform.
func HumanSpeed(base, modifier, load, capacity int16) (speed, kept int16, ok bool) {
	v := int32(base)
	if load >= capacity {
		if capacity == 0 {
			return base, modifier, false
		}
		v = max(v-int32(load)/int32(capacity), HumanOverloadFloor)
	}
	speed = int16(uint16(v) + uint16(modifier))
	if speed < 0 {
		modifier = 0
	}
	return speed, modifier, true
}
