// Package rules holds the game parameters a mod may change, as one immutable
// value with the original game's numbers as defaults.
//
// A Rules value is built once, before any world exists, and handed to every
// reader (the simulation, the derived-statistics graph, the save writer). It
// has no setters, owns its tables privately and returns only copies, so a
// reader cannot change what another reader sees. The zero Rules is the
// original game's rules, which keeps every constructor that never names a
// Rules on the unmodded path.
//
// The package imports nothing of this module: the simulation and the data
// package both depend on it, and neither may depend on a mod interpreter.
package rules

import (
	"fmt"
	"math"
	"math/big"
)

// ROM1SkillCap is the highest skill level of the original game.
const ROM1SkillCap int32 = 100

// EffectiveSkillBound is the highest effective skill level, the trained base
// plus the item and spell bonus. It is independent of the training cap. Three
// readers bound it: a skill level is a signed 16-bit word, the character
// readouts hold three digits, and the damage byte of a derived attack block
// takes a fifth of the level on top of the weapon, so 255 adds at most 51 to
// it against the 20 of level 100.
const EffectiveSkillBound int32 = 255

// Declared range of the skill cap. The upper bound is the highest level whose
// experience S(n) = ftol((1.1^n - 1) * 1000) fits the signed 32-bit experience
// the game stores per slot: S(152) = 1957437581 fits and S(153) overflows.
const (
	MinSkillCap int32 = 1
	MaxSkillCap int32 = 150
)

// rom1SkillXP is S(n) for n = 0..100 as the original game's curve gives it.
var rom1SkillXP = [101]int32{
	0, 100, 210, 331, 464,
	610, 771, 948, 1143, 1357,
	1593, 1853, 2138, 2452, 2797,
	3177, 3594, 4054, 4559, 5115,
	5727, 6400, 7140, 7954, 8849,
	9834, 10918, 12109, 13420, 14863,
	16449, 18194, 20113, 22225, 24547,
	27102, 29912, 33003, 36404, 40144,
	44259, 48785, 53763, 59240, 65264,
	71890, 79179, 87197, 96017, 105718,
	116390, 128129, 141042, 155247, 170871,
	188059, 206965, 227761, 250637, 275801,
	303481, 333929, 367422, 404265, 444791,
	489370, 538407, 592348, 651683, 716951,
	788746, 867721, 954593, 1050153, 1155268,
	1270895, 1398084, 1537993, 1691892, 1861182,
	2047400, 2252240, 2477564, 2725420, 2998062,
	3297969, 3627865, 3990752, 4389927, 4829020,
	5312022, 5843324, 6427757, 7070633, 7777796,
	8555676, 9411343, 10352578, 11387935, 12526829,
	13779612,
}

// Params are the parameters a mod can set. Their zero values are not valid
// settings; Defaults returns the original game's.
type Params struct {
	// SkillCap is the highest skill level, within [MinSkillCap, MaxSkillCap].
	SkillCap int32
}

// Defaults are the original game's parameters.
func Defaults() Params { return Params{SkillCap: ROM1SkillCap} }

// Rules is the immutable value built from Params.
type Rules struct {
	skillCap int32
	skillXP  []int32 // S(0)..S(skillCap); nil in the zero value
	spell    *spellTables
}

// Default returns the original game's rules.
func Default() Rules {
	r, err := New(Defaults())
	if err != nil {
		panic(err)
	}
	return r
}

// New validates p and builds the rules it describes. The error names the
// parameter and its declared range.
func New(p Params) (Rules, error) {
	if p.SkillCap < MinSkillCap || p.SkillCap > MaxSkillCap {
		return Rules{}, fmt.Errorf("skill_cap %d is outside the declared range %d..%d", p.SkillCap, MinSkillCap, MaxSkillCap)
	}
	table := make([]int32, p.SkillCap+1)
	for n := range table {
		table[n] = skillXPAt(int32(n))
	}
	return Rules{skillCap: p.SkillCap, skillXP: table}, nil
}

// skillXPAt is the curve at one level: the original table where it has one,
// the same formula in exact integers past it, saturating at the largest
// experience a slot can hold.
func skillXPAt(level int32) int32 {
	switch {
	case level <= 0:
		return 0
	case int(level) < len(rom1SkillXP):
		return rom1SkillXP[level]
	}
	ten := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(level)), nil)
	eleven := new(big.Int).Exp(big.NewInt(11), big.NewInt(int64(level)), nil)
	v := eleven.Sub(eleven, ten)
	v.Mul(v, big.NewInt(1000))
	v.Quo(v, ten)
	if !v.IsInt64() || v.Int64() > math.MaxInt32 {
		return math.MaxInt32
	}
	return int32(v.Int64())
}

// Params returns the parameters the rules were built from.
func (r Rules) Params() Params { return Params{SkillCap: r.SkillCap()} }

// SkillCap is the highest trained skill level: the level the experience award
// stops at and the highest base a hero holds.
func (r Rules) SkillCap() int32 {
	if r.skillXP == nil {
		return ROM1SkillCap
	}
	return r.skillCap
}

// EffectiveSkillLimit is the highest effective skill level, the trained base
// plus the bonus of worn items and active spells. It never falls below the
// training cap.
func (r Rules) EffectiveSkillLimit() int32 { return max(EffectiveSkillBound, r.SkillCap()) }

// EffectiveSkill is the level a hero fights and casts at: the trained base,
// held to the training cap, plus the bonus, kept within [0, the effective
// limit]. The sum is a signed 16-bit word, as in the original, so a bonus that
// wraps the word reads negative and floors at zero. A zero trainingCap reads
// as the original game's.
func EffectiveSkill(base, bonus, trainingCap int32) int32 {
	if trainingCap == 0 {
		trainingCap = ROM1SkillCap
	}
	sum := int32(int16(min(base, trainingCap) + bonus))
	return max(0, min(sum, max(EffectiveSkillBound, trainingCap)))
}

// EffectiveSkill is the package function under these rules.
func (r Rules) EffectiveSkill(base, bonus int32) int32 {
	return EffectiveSkill(base, bonus, r.SkillCap())
}

// SkillXP is the experience at which a slot reaches level. A level below zero
// reads as zero and a level above the cap reads as the cap's experience.
func (r Rules) SkillXP(level int32) int32 {
	table := r.table()
	switch {
	case level < 0:
		return table[0]
	case int(level) >= len(table):
		return table[len(table)-1]
	}
	return table[level]
}

// SkillXPExtended is SkillXP without the clamp to the cap: past the cap it
// continues along the curve.
func (r Rules) SkillXPExtended(level int32) int32 {
	if level >= 0 && int(level) < len(r.table()) {
		return r.SkillXP(level)
	}
	return skillXPAt(level)
}

// SkillLevelFor is the largest level in [0, cap] whose experience is at most xp.
func (r Rules) SkillLevelFor(xp int32) int32 {
	table := r.table()
	for n := len(table) - 1; n >= 0; n-- {
		if table[n] <= xp {
			return int32(n)
		}
	}
	return 0
}

// ClampSkill bounds a level to [0, cap].
func (r Rules) ClampSkill(level int32) int32 {
	if level < 0 {
		return 0
	}
	if c := r.SkillCap(); level > c {
		return c
	}
	return level
}

// IsDefault reports whether the rules are the original game's.
func (r Rules) IsDefault() bool { return r.SkillCap() == ROM1SkillCap && r.spell == nil }

// Equal reports whether two rules are the same parameters.
func (r Rules) Equal(o Rules) bool {
	return r.Params() == o.Params() && spellTablesEqual(r.spell, o.spell)
}

func (r Rules) table() []int32 {
	if r.skillXP == nil {
		return defaultTable
	}
	return r.skillXP
}

var defaultTable = func() []int32 {
	t := make([]int32, len(rom1SkillXP))
	copy(t, rom1SkillXP[:])
	return t
}()
