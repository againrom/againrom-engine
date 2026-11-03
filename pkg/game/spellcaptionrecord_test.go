package game

import (
	"fmt"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// spellCaptionProbes are the characteristics every caption block is asked
// with: powers across each formula's steps and the int32 extremes, then mod
// magnitudes held at one power.
func spellCaptionProbes() []sim.SpellCharacteristics {
	var out []sim.SpellCharacteristics
	for _, p := range []int32{-2147483648, -100, -1, 0, 1, 14, 15, 19, 20, 29, 30, 99, 100, 101, 199, 200, 255, 1000, 536870912, 2147483647} {
		out = append(out, sim.SpellCharacteristics{Power: p})
	}
	for _, m := range []int32{-7, 0, 3, 99, 100, 150} {
		out = append(out, sim.SpellCharacteristics{Power: 30, Magnitude: m, HasMagnitude: true})
	}
	return out
}

// spellCaptionAnswers is every caption block's answer for one row over the
// probes, "label=value" for each block that holds one.
func spellCaptionAnswers(rule sim.SpellRule) string {
	var parts []string
	for _, c := range spellCaptionProbes() {
		var held []string
		for _, block := range spellCaptions {
			if v, ok := block.value(rule, c); ok {
				held = append(held, fmt.Sprintf("%d=%d", block.label, v))
			}
		}
		parts = append(parts, strings.Join(held, "+"))
	}
	if strings.Join(parts, "") == "" {
		return ""
	}
	return strings.Join(parts, " ")
}

// spellCaptionRecord is each arm's caption answers as the popup gave them
// before the arm table held them, the same in both games: the key is the arm
// a row runs.
var spellCaptionRecord = map[uint16]string{
	5:  "183=-1073741824 183=-50 183=0 183=0 183=0 183=7 183=7 183=9 183=10 183=14 183=15 183=49 183=50 183=50 183=99 183=100 183=100 183=100 183=100 183=100 183=-7 183=0 183=3 183=99 183=100 183=100",
	7:  "182=143165575 182=5 182=-1 182=-1 182=-1 182=-1 182=-2 182=-2 182=-2 182=-2 182=-3 182=-7 182=-7 182=-7 182=-14 182=-14 182=-18 182=-67 182=-35791395 182=-143165577 182=-7 182=0 182=3 182=99 182=100 182=150",
	10: "183=-1073741824 183=-50 183=0 183=0 183=0 183=7 183=7 183=9 183=10 183=14 183=15 183=49 183=50 183=50 183=99 183=100 183=100 183=100 183=100 183=100 183=-7 183=0 183=3 183=99 183=100 183=100",
	12: "184=71582787 184=2 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-2 184=-4 184=-4 184=-4 184=-7 184=-7 184=-9 184=-34 184=-17895698 184=-71582789 184=7 184=0 184=-3 184=-99 184=-100 184=-150",
	14: "186=0 186=0 186=2 186=2 186=2 186=2 186=2 186=2 186=3 186=3 186=3 186=6 186=7 186=7 186=7 186=7 186=7 186=7 186=7 186=7 186=3 186=3 186=3 186=3 186=3 186=3",
	16: "183=-1073741824 183=-50 183=0 183=0 183=0 183=7 183=7 183=9 183=10 183=14 183=15 183=49 183=50 183=50 183=99 183=100 183=100 183=100 183=100 183=100 183=-7 183=0 183=3 183=99 183=100 183=100",
	17: "184=71582787 184=2 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-1 184=-2 184=-4 184=-4 184=-4 184=-7 184=-7 184=-9 184=-34 184=-17895698 184=-71582789 184=7 184=0 184=-3 184=-99 184=-100 184=-150",
	18: "217=-214748361 217=-7 217=3 217=3 217=3 217=4 217=4 217=4 217=5 217=5 217=6 217=12 217=13 217=13 217=22 217=23 217=28 217=103 217=53687094 217=214748367 217=-7 217=0 217=3 217=99 217=100 217=150",
	22: "183=-1073741824 183=-50 183=0 183=0 183=0 183=7 183=7 183=9 183=10 183=14 183=15 183=49 183=50 183=50 183=99 183=100 183=100 183=100 183=100 183=100 183=-7 183=0 183=3 183=99 183=100 183=100",
	23: "185=-1717986898 185=-60 185=20 185=20 185=20 185=31 185=32 185=35 185=36 185=43 185=44 185=99 185=100 185=100 185=179 185=180 185=224 185=820 185=429496749 185=1717986937 185=-7 185=0 185=3 185=99 185=100 185=150",
	24: "182=-143165575 182=-5 182=1 182=1 182=1 182=1 182=2 182=2 182=2 182=2 182=3 182=7 182=7 182=7 182=14 182=14 182=18 182=67 182=35791395 182=143165577 182=-7 182=0 182=3 182=99 182=100 182=150",
	27: "187=-1717986898 187=-60 187=20 187=20 187=20 187=31 187=32 187=35 187=36 187=43 187=44 187=99 187=100 187=100 187=179 187=180 187=224 187=820 187=429496749 187=1717986937 187=-7 187=0 187=3 187=99 187=100 187=150",
	28: "182=143165575 182=5 182=-1 182=-1 182=-1 182=-1 182=-2 182=-2 182=-2 182=-2 182=-3 182=-7 182=-7 182=-7 182=-14 182=-14 182=-18 182=-67 182=-35791395 182=-143165577 182=-7 182=0 182=3 182=99 182=100 182=150",
}

func spellCaptionProbeIDs() []uint16 {
	out := []uint16{sim.ArmNone, 0xfffe, 0x8000}
	for id := uint16(0); id <= 64; id++ {
		out = append(out, id)
	}
	return out
}

// TestSpellCaptionsKeepTheirRecord asks every caption block about every
// probed row id in both games: a row states the caption of the arm it runs.
func TestSpellCaptionsKeepTheirRecord(t *testing.T) {
	for _, id := range spellCaptionProbeIDs() {
		for _, second := range []bool{false, true} {
			rule := sim.SpellRule{ID: id, Arm: id, Second: second}
			if second {
				rule.Arm = sim.SecondGameSpellArm(id)
			}
			if got, want := spellCaptionAnswers(rule), spellCaptionRecord[rule.Arm]; got != want {
				t.Errorf("id %d second=%v: captions %q, record %q", id, second, got, want)
			}
		}
	}
	if len(spellCaptionRecord) == 0 {
		t.Fatal("the record is empty")
	}
}
