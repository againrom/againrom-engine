package data

// DamageKindResistance narrows the five Blade, Axe, Bludgeon, Pike and
// Shooting resistance values to the actor bytes the physical resolver reads.
//
// The store is a conversion, not a clamp. HERO-FOLD-033 and UNIT-WIDTH-016
// establish that the original writes the low byte and lets additive modifiers
// wrap modulo 256. Converting each int32 to uint8 gives the same result for
// zero, negative values, values above 255 and both int32 extremes.
func DamageKindResistance(values [5]int32) (out [5]uint8) {
	for i, value := range values {
		out[i] = uint8(value)
	}
	return out
}
