package sim

import (
	"reflect"
	"sort"
	"testing"
)

// scriptedDraws answers each bounded draw from a queue kept per width and
// records the width of every call, in call order.
type scriptedDraws struct {
	answers map[int32][]int32
	widths  []int32
}

func (s *scriptedDraws) draw(n int32) int32 {
	s.widths = append(s.widths, n)
	q := s.answers[n]
	if len(q) == 0 {
		return 0
	}
	s.answers[n] = q[1:]
	if q[0] > n {
		return n
	}
	return q[0]
}

// blowDrawCase is one blow: attacker 1 (to-hit 50, base 10, spread 6 unless
// changed) on target 2 (defence 100, absorption 2, health 100 of 100).
type blowDrawCase struct {
	name                     string
	bless, curse, targetCurs int32 // magnitudes; 0 attaches nothing
	base, spread             int32
	second, secondSpread     uint8
	magic                    SecondaryDamage
	defence, absorption      int32
	answers                  map[int32][]int32
	second2, first           []int32 // widths in call order, second and first game
	hp2, hp1                 int32   // target health after the blow
}

// Widths: Bless and Curse 100, the spread 6, the roll 200 (answer r + 100),
// the second pair 3, the magic part 4.
var blowDrawCases = []blowDrawCase{
	{name: "blessed attacker on a cursed target, Bless takes effect", bless: 50, targetCurs: 100,
		answers: map[int32][]int32{100: {10}, 200: {160}},
		second2: []int32{100, 200}, first: []int32{6, 100, 200}, hp2: 86, hp1: 86},
	{name: "blessed attacker on a cursed target, Bless fails", bless: 50, targetCurs: 100,
		answers: map[int32][]int32{100: {80}, 6: {3}, 200: {160}},
		second2: []int32{100, 6, 200}, first: []int32{6, 100, 200}, hp2: 89, hp1: 89},
	{name: "blessed and cursed attacker, both take effect", bless: 50, curse: 50,
		answers: map[int32][]int32{100: {10, 10}, 200: {160}},
		second2: []int32{100, 100, 200}, first: []int32{6, 100, 200}, hp2: 86, hp1: 86},
	{name: "cursed attacker, Curse takes effect", curse: 50,
		answers: map[int32][]int32{100: {10}, 200: {160}},
		second2: []int32{100, 200}, first: []int32{6, 100, 200}, hp2: 92, hp1: 92},
	{name: "neither effect, spread above 0",
		answers: map[int32][]int32{6: {3}, 200: {160}},
		second2: []int32{6, 200}, first: []int32{6, 200}, hp2: 89, hp1: 89},
	{name: "neither effect, spread 0", spread: -1,
		answers: map[int32][]int32{200: {160}},
		second2: []int32{200}, first: []int32{0, 200}, hp2: 92, hp1: 92},
	{name: "a miss draws Bless and the spread before the roll, the second pair after", bless: 50,
		second: 4, secondSpread: 3, magic: SecondaryDamage{Base: 5, Spread: 4},
		answers: map[int32][]int32{100: {80}, 6: {3}, 200: {100}, 3: {1}, 4: {2}},
		second2: []int32{100, 6, 200, 3}, first: []int32{6, 100, 200, 3}, hp2: 95, hp1: 95},
	{name: "a roll of 90 or more against a defence the sum does not reach", defence: 1000,
		magic:   SecondaryDamage{Base: 5, Spread: 4},
		answers: map[int32][]int32{6: {3}, 200: {195}, 4: {2}},
		second2: []int32{6, 200}, first: []int32{6, 200, 4}, hp2: 89, hp1: 82},
	{name: "a roll below -90",
		answers: map[int32][]int32{6: {3}, 200: {5}},
		second2: []int32{6, 200}, first: []int32{6, 200}, hp2: 100, hp1: 100},
	{name: "second pair and magic part admitted by to-hit plus the roll",
		second: 4, secondSpread: 3, magic: SecondaryDamage{Base: 5, Spread: 4},
		answers: map[int32][]int32{6: {3}, 200: {160}, 3: {1}, 4: {2}},
		second2: []int32{6, 200, 3, 4}, first: []int32{6, 200, 3, 4}, hp2: 77, hp1: 77},
	{name: "second pair and magic part on a miss: the magic part is not admitted",
		second: 4, secondSpread: 3, magic: SecondaryDamage{Base: 5, Spread: 4},
		answers: map[int32][]int32{6: {3}, 200: {100}, 3: {1}, 4: {2}},
		second2: []int32{6, 200, 3}, first: []int32{6, 200, 3}, hp2: 95, hp1: 95},
	{name: "an empty physical pair admits the magic part on a miss", base: -1, spread: -1,
		magic:   SecondaryDamage{Base: 5, Spread: 4},
		answers: map[int32][]int32{200: {100}, 4: {2}},
		second2: []int32{200, 4}, first: []int32{0, 200, 4}, hp2: 93, hp1: 93},
	{name: "a negative absorption gives an empty physical part nothing", base: -1, spread: -1, absorption: -3,
		answers: map[int32][]int32{200: {160}},
		second2: []int32{200}, first: []int32{0, 200}, hp2: 100, hp1: 97},
}

// runBlowDrawCase resolves tc's blow once in the game the profile names and
// answers the widths drawn and the target's health.
func runBlowDrawCase(t *testing.T, tc blowDrawCase, second bool) ([]int32, int32) {
	t.Helper()
	a := cbEnt(1, 0, 0)
	a.ToHit, a.DamageBase, a.DamageSpread = 50, 10, 6
	if tc.base < 0 {
		a.DamageBase = 0
	}
	if tc.spread < 0 {
		a.DamageSpread = 0
	}
	a.SecondBase, a.SecondSpread, a.SecondaryDamage = tc.second, tc.secondSpread, tc.magic
	v := cbEnt(2, 1, 0)
	v.Defence, v.Absorption = 100, 2
	if tc.defence != 0 {
		v.Defence = tc.defence
	}
	if tc.absorption != 0 {
		v.Absorption = tc.absorption
	}
	w := cbWorld(t, 1, a, v)
	if second {
		w.game = newGameProfile(ScriptROM2)
	}
	attach := func(target EntityID, spell uint16, kind EffectKind, m int32) {
		if m != 0 {
			w.attached = append(w.attached, attachedEffect{Target: target, Spell: spell, Kind: kind, Mode: EffectDuration, Magnitude: m, Remaining: 100})
		}
	}
	attach(1, w.blessingSpell(), EffectBless, tc.bless)
	attach(1, w.cursingSpell(), EffectCurse, tc.curse)
	attach(2, w.cursingSpell(), EffectCurse, tc.targetCurs)
	sort.Slice(w.attached, func(i, j int) bool {
		x, y := w.attached[i], w.attached[j]
		return x.Target < y.Target || x.Target == y.Target && x.Spell < y.Spell
	})
	answers := map[int32][]int32{}
	for k, q := range tc.answers {
		answers[k] = append([]int32(nil), q...)
	}
	script := &scriptedDraws{answers: answers}
	w.rng.scripted = script

	w.resolveBlow(0, 1, nil)

	return script.widths, cbAt(t, w, 2).HP
}

// TestTheSecondGameBlowDrawsInTheOriginalOrder is R2-ENGINE-1149 at the one
// resolver: Bless, then Curse whatever Bless gave, then the spread only when
// neither took effect and it is above 0, then the roll, the second pair, and
// the magic part only when admitted (R2-ENGINE-1148). A spread of 0 makes no
// draw (R2-ENGINE-1152); absorption comes off a physical part above 0 only.
func TestTheSecondGameBlowDrawsInTheOriginalOrder(t *testing.T) {
	for _, tc := range blowDrawCases {
		t.Run(tc.name, func(t *testing.T) {
			widths, hp := runBlowDrawCase(t, tc, true)
			if !reflect.DeepEqual(widths, tc.second2) {
				t.Errorf("draw widths = %v, want %v", widths, tc.second2)
			}
			if hp != tc.hp2 {
				t.Errorf("target health = %d, want %d", hp, tc.hp2)
			}
		})
	}
}

// TestTheFirstGameBlowKeepsItsDraws pins the first game's draws for the same
// blows: the spread first, even at 0, then Bless, then Curse only when Bless
// failed, then the roll, the second pair and the magic part on any hit.
func TestTheFirstGameBlowKeepsItsDraws(t *testing.T) {
	for _, tc := range blowDrawCases {
		t.Run(tc.name, func(t *testing.T) {
			widths, hp := runBlowDrawCase(t, tc, false)
			if !reflect.DeepEqual(widths, tc.first) {
				t.Errorf("draw widths = %v, want %v", widths, tc.first)
			}
			if hp != tc.hp1 {
				t.Errorf("target health = %d, want %d", hp, tc.hp1)
			}
		})
	}
}
