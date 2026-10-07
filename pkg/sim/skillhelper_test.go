package sim

// skillXPFor is the original game's curve, for tests that build a character
// at a level.
func skillXPFor(level int32) int32 { return Rules{}.SkillXP(level) }
