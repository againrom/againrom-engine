package sim

import "testing"

// armAnswers is one consumer question's answer per arm in each game: an arm
// absent from a map answers zero (false, none).
type armAnswers struct {
	first, second map[uint16]int
}

func bothGames(m map[uint16]int) armAnswers { return armAnswers{m, m} }

func secondOnly(m map[uint16]int) armAnswers { return armAnswers{map[uint16]int{}, m} }

func armSet(arms ...uint16) map[uint16]int {
	out := map[uint16]int{}
	for _, a := range arms {
		out[a] = 1
	}
	return out
}

func (a armAnswers) of(arm uint16, second bool) int {
	if second {
		return a.second[arm]
	}
	return a.first[arm]
}

// Durations laws and delivery forms as the record numbers them.
const (
	recordLawFastThree = 1
	recordLawSegmented = 2
	recordFormBolt     = 1
	recordFormFan      = 2
	recordRoleHeal     = 1
	recordRoleShield   = 2
	recordRoleHide     = 3
	recordLightBright  = 1
	recordLightDark    = 2
)

// spellArmRecord is each spell-arm question's answer as the consumers
// computed it before the arm table existed: one entry per question, read
// from the consumer's own comparison (the story document lists each file
// and line). TestSpellArmConsumersKeepTheirRecord asks the consumers that
// are functions; the arm table's test asks every row.
var spellArmRecord = map[string]armAnswers{
	"applies":          bothGames(armSet(11, 14, 15, 18, 20, 23, 24, 25, 26, 27, 28, armSummon)),
	"atActor":          bothGames(armSet(15, 18, 20, 23, 24, 25, 26, 27, 28, armSummon)),
	"drain":            bothGames(armSet(11)),
	"heals":            bothGames(armSet(6)),
	"prismatic":        bothGames(armSet(14)),
	"teleport":         bothGames(armSet(26)),
	"pointTarget":      bothGames(armSet(26)),
	"fastRange":        bothGames(armSet(26)),
	"manualOnly":       bothGames(armSet(26)),
	"controlSpirit":    bothGames(armSet(25)),
	"summon":           bothGames(armSet(armSummon)),
	"stoneCurse":       bothGames(armSet(20)),
	"invisibility":     bothGames(armSet(15)),
	"attachAward":      bothGames(armSet(20, 27, 28)),
	"ownEffect":        bothGames(map[uint16]int{15: int(EffectInvisible), 23: int(EffectBless), 27: int(EffectCurse)}),
	"ownScaled":        bothGames(armSet(23, 27)),
	"opposite":         bothGames(map[uint16]int{23: 27, 27: 23}),
	"magnitude":        bothGames(armSet(5, 7, 8, 10, 12, 16, 17, 18, 22, 24, 28)),
	"formulaMagnitude": bothGames(armSet(5, 7, 8, 10, 12, 16, 17, 18, 22, 23, 24, 27, 28)),
	"lastingDuration":  bothGames(armSet(5, 10, 15, 16, 18, 20, 22, 23, 24, 27, 28)),
	"durationLaw":      bothGames(map[uint16]int{15: recordLawFastThree, 20: recordLawSegmented}),
	"form":             bothGames(map[uint16]int{13: recordFormBolt, 14: recordFormFan}),
	"autoCastLevel":    bothGames(map[uint16]int{6: 1, 18: 1, 23: 2, 24: 2, 16: 3, 22: 3, 5: 3, 10: 3, 15: 3}),
	"autoCastOrder":    bothGames(map[uint16]int{6: 0, 18: 1, 23: 0, 24: 1, 16: 0, 22: 1, 5: 2, 10: 3, 15: 4}),
	"autoCastRole":     bothGames(map[uint16]int{6: recordRoleHeal, 18: recordRoleShield, 15: recordRoleHide}),
	"selfAimed":        bothGames(armSet(4)),
	"sparesMages":      bothGames(armSet(23)),
	"light":            bothGames(map[uint16]int{12: recordLightBright, 17: recordLightDark}),
	"drawnDuration":    secondOnly(armSet(20)),
	"blankCurse":       secondOnly(armSet(20)),
	"keepsLarger": secondOnly(armSet(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25,
		26, 27, 28, armSummon)),
	"powerDuration": secondOnly(armSet(15)),
}

func probeRule(arm uint16, second bool) SpellRule {
	return SpellRule{ID: arm, Arm: arm, Second: second}
}

func boolAnswer(b bool) int {
	if b {
		return 1
	}
	return 0
}

// consumerAnswers asks each consumer that is a function its question about
// one arm, keyed by the question it answers.
func consumerAnswers(arm uint16, second bool) map[string]int {
	rule := probeRule(arm, second)
	w := &World{}
	out := map[string]int{
		"applies":          boolAnswer(spellApplicable(rule)),
		"atActor":          boolAnswer(castsAtActor(rule)),
		"drain":            boolAnswer(spellFinishesBody(rule)),
		"prismatic":        boolAnswer(prismaticBodyPrimary(Entity{}, rule) && raysFlag(spellFlagAreaHitsHi, arm)),
		"teleport":         boolAnswer(w.presentedAtAdmission(rule)),
		"manualOnly":       boolAnswer(!autoCastable(rule)),
		"formulaMagnitude": boolAnswer(MagnitudeArm(arm)),
		"lastingDuration":  boolAnswer(DurationArm(arm)),
		"blankCurse":       boolAnswer(secondBlankCurse(rule)),
		"keepsLarger":      boolAnswer(keepsLargerOnRepeat(rule)),
	}
	ranged := rule
	ranged.MaxRange = 5
	out["fastRange"] = boolAnswer(spellRange(ranged, 30) == 15)
	if level := AutoCastLevelOfArm(arm); level != 0 {
		for i, l := range autoCastLevelList {
			if l == level {
				out["autoCastLevel"] = i + 1
			}
		}
	}
	if kind, _, _, _ := w.pointEffect(-1, rule, 30); kind != EffectNone {
		out["ownEffect"] = int(kind)
	}
	lasting := rule
	lasting.SpellDuration = 4
	switch got := spellLastingTicks(lasting, 30); {
	case got == segmentedTicks(30, 3, durationFast) && got != lastingTicks(30, 4, durationSlow):
		out["durationLaw"] = recordLawFastThree
	case got == segmentedTicks(30, 4, durationSlow) && got != lastingTicks(30, 4, durationSlow):
		out["durationLaw"] = recordLawSegmented
	}
	if second && spellLastingTicksUnder(Rules{}, lasting, 30) == 30<<4 {
		out["powerDuration"] = 1
	}
	if spellDeliveryTicks(rule, 0, 0, 0, 0) == 10 {
		out["form"] = -1
	}
	return out
}

// TestSpellArmConsumersKeepTheirRecord asks every consumer that is a
// function about every probed arm in both games: each answers its record.
func TestSpellArmConsumersKeepTheirRecord(t *testing.T) {
	for _, arm := range armProbe() {
		for _, second := range []bool{false, true} {
			for name, got := range consumerAnswers(arm, second) {
				want := spellArmRecord[name].of(arm, second)
				if name == "form" && want != 0 {
					want = -1
				}
				if got != want {
					t.Errorf("%s: arm %d second=%v answers %d, record %d", name, arm, second, got, want)
				}
			}
		}
	}
}

func TestSpellArmRecordNamesEveryConsumer(t *testing.T) {
	for name := range consumerAnswers(0, false) {
		if _, ok := spellArmRecord[name]; !ok {
			t.Errorf("consumer %s has no record", name)
		}
	}
	if len(spellArmRecord) == 0 {
		t.Fatal("the record is empty")
	}
}
