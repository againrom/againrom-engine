package sim

import (
	"reflect"
	"testing"
)

// spellArmCapabilities reads each recorded question off a table row; rule is
// the probed row of that arm in one game.
var spellArmCapabilities = map[string]func(row spellArm, rule SpellRule) int{
	"applies":          func(r spellArm, _ SpellRule) int { return boolAnswer(r.applies) },
	"atActor":          func(r spellArm, _ SpellRule) int { return boolAnswer(r.atActor) },
	"drain":            func(r spellArm, _ SpellRule) int { return boolAnswer(r.drain) },
	"heals":            func(r spellArm, _ SpellRule) int { return boolAnswer(r.heals) },
	"prismatic":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.prismatic) },
	"teleport":         func(r spellArm, _ SpellRule) int { return boolAnswer(r.teleport) },
	"pointTarget":      func(r spellArm, _ SpellRule) int { return boolAnswer(r.pointTarget) },
	"fastRange":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.fastRange) },
	"manualOnly":       func(r spellArm, _ SpellRule) int { return boolAnswer(r.manualOnly) },
	"controlSpirit":    func(r spellArm, _ SpellRule) int { return boolAnswer(r.controlSpirit) },
	"summon":           func(r spellArm, _ SpellRule) int { return boolAnswer(r.summon) },
	"stoneCurse":       func(r spellArm, _ SpellRule) int { return boolAnswer(r.stoneCurse) },
	"invisibility":     func(r spellArm, _ SpellRule) int { return boolAnswer(r.invisibility) },
	"attachAward":      func(r spellArm, _ SpellRule) int { return boolAnswer(r.attachAward) },
	"selfAimed":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.selfAimed) },
	"ownEffect":        func(r spellArm, _ SpellRule) int { return int(r.own.kind) },
	"ownScaled":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.own.scaled) },
	"opposite":         func(r spellArm, _ SpellRule) int { return int(r.opposite) },
	"magnitude":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.magnitude != nil) },
	"formulaMagnitude": func(r spellArm, _ SpellRule) int { return boolAnswer(r.formulaMagnitude) },
	"lastingDuration":  func(r spellArm, _ SpellRule) int { return boolAnswer(r.lastingDuration) },
	"durationLaw": func(r spellArm, _ SpellRule) int {
		return map[durationLaw]int{lawFastThree: recordLawFastThree, lawSegmented: recordLawSegmented}[r.durationLaw]
	},
	"form": func(r spellArm, _ SpellRule) int {
		return map[spellForm]int{formBolt: recordFormBolt, formFan: recordFormFan}[r.form]
	},
	"autoCastLevel": func(r spellArm, _ SpellRule) int { return int(r.autoCast.level) },
	"autoCastOrder": func(r spellArm, _ SpellRule) int { return int(r.autoCast.order) },
	"autoCastRole": func(r spellArm, _ SpellRule) int {
		return map[autoCastRole]int{autoCastHealer: recordRoleHeal, autoCastShielder: recordRoleShield, autoCastHider: recordRoleHide}[r.autoCast.role]
	},
	"light": func(r spellArm, _ SpellRule) int {
		return map[SpellLighting]int{SpellLightBright: recordLightBright, SpellLightDark: recordLightDark}[r.light]
	},
	"drawnDuration": func(_ spellArm, rule SpellRule) int { return boolAnswer(rule.secondArm().drawnDuration) },
	"blankCurse":    func(_ spellArm, rule SpellRule) int { return boolAnswer(rule.secondArm().blankCurse) },
	"keepsLarger":   func(_ spellArm, rule SpellRule) int { return boolAnswer(rule.secondArm().keepsLarger) },
	"powerDuration": func(_ spellArm, rule SpellRule) int { return boolAnswer(rule.secondArm().powerDuration) },
	"creatureAim":   func(r spellArm, _ SpellRule) int { return int(r.creatureAim) },
	"burst":         func(r spellArm, _ SpellRule) int { return boolAnswer(r.burst) },
	"speedFlight":   func(r spellArm, _ SpellRule) int { return boolAnswer(r.speedFlight) },
	"healID":        func(r spellArm, _ SpellRule) int { return boolAnswer(r.heals && uint32(r.arm) == uint32(healingArm)) },
	"caption": func(r spellArm, _ SpellRule) int {
		return int(r.caption.kind)*2 + boolAnswer(r.caption.negate)
	},
}

// spellArmCaptionRecord is the caption each row id states, as kind*2 plus 1
// when the row's magnitude is negated: pkg/game's popup record fixes the
// numbers each states.
var spellArmCaptionRecord = map[string]armAnswers{
	"caption": bothGames(map[uint16]int{
		7: 2 * int(SpellCaptionSpeed), 24: 2 * int(SpellCaptionSpeed), 28: 2 * int(SpellCaptionSpeed),
		5: 2 * int(SpellCaptionProtection), 10: 2 * int(SpellCaptionProtection), 16: 2 * int(SpellCaptionProtection),
		22: 2 * int(SpellCaptionProtection),
		12: 2*int(SpellCaptionLight) + 1, 17: 2 * int(SpellCaptionLight),
		23: 2 * int(SpellCaptionBless), 27: 2 * int(SpellCaptionCurse), 14: 2 * int(SpellCaptionRays),
		18: 2 * int(SpellCaptionShield),
	}),
}

// spellArmRecordOf is a capability's record: the consumers', the World
// sites' or the captions'.
func spellArmRecordOf(name string) (armAnswers, bool) {
	for _, m := range []map[string]armAnswers{spellArmRecord, spellArmSiteRecord, spellArmCaptionRecord} {
		if r, ok := m[name]; ok {
			return r, true
		}
	}
	return armAnswers{}, false
}

// spellArmFieldQuestions names, per field of a row, the questions it answers.
var spellArmFieldQuestions = map[string][]string{
	"applies": {"applies"}, "atActor": {"atActor"}, "drain": {"drain"}, "heals": {"heals", "healID"},
	"creatureAim": {"creatureAim"}, "caption": {"caption"}, "burst": {"burst"}, "speedFlight": {"speedFlight"},
	"prismatic": {"prismatic"}, "teleport": {"teleport"}, "pointTarget": {"pointTarget"},
	"fastRange": {"fastRange"}, "manualOnly": {"manualOnly"}, "controlSpirit": {"controlSpirit"},
	"summon": {"summon"}, "stoneCurse": {"stoneCurse"}, "invisibility": {"invisibility"},
	"attachAward": {"attachAward"}, "selfAimed": {"selfAimed"}, "own": {"ownEffect", "ownScaled"},
	"opposite": {"opposite"}, "magnitude": {"magnitude"}, "formulaMagnitude": {"formulaMagnitude"},
	"lastingDuration": {"lastingDuration"}, "durationLaw": {"durationLaw"}, "form": {"form"},
	"autoCast": {"autoCastLevel", "autoCastOrder", "autoCastRole"}, "light": {"light"},
	"second": {"drawnDuration", "blankCurse", "keepsLarger", "powerDuration"},
}

// TestSpellArmTable asks each capability of every probed arm's row in both
// games, one subtest per capability: the row answers the record.
func TestSpellArmTable(t *testing.T) {
	for name, of := range spellArmCapabilities {
		t.Run(name, func(t *testing.T) {
			record, ok := spellArmRecordOf(name)
			if !ok {
				t.Fatalf("capability %s has no record", name)
			}
			for _, arm := range armProbe() {
				for _, second := range []bool{false, true} {
					if got, want := of(spellArmOf(arm), probeRule(arm, second)), record.of(arm, second); got != want {
						t.Errorf("arm %d second=%v answers %d, record %d", arm, second, got, want)
					}
				}
			}
		})
	}
}

// TestSpellArmTableAsksEveryField pins that each row field answers a
// recorded question, and each recorded question is a capability.
func TestSpellArmTableAsksEveryField(t *testing.T) {
	typ := reflect.TypeOf(spellArm{})
	for i := 0; i < typ.NumField(); i++ {
		name := typ.Field(i).Name
		if name == "arm" {
			continue
		}
		questions, ok := spellArmFieldQuestions[name]
		if !ok {
			t.Errorf("row field %s answers no recorded question", name)
		}
		for _, q := range questions {
			if _, ok := spellArmCapabilities[q]; !ok {
				t.Errorf("field %s: question %s is not a capability", name, q)
			}
		}
	}
	for _, m := range []map[string]armAnswers{spellArmRecord, spellArmSiteRecord, spellArmCaptionRecord} {
		for name := range m {
			if _, ok := spellArmCapabilities[name]; !ok {
				t.Errorf("recorded question %s is not a capability", name)
			}
		}
	}
}

// TestSpellArmTableRowsAreUnique pins one row per arm and the probe covering
// every row.
func TestSpellArmTableRowsAreUnique(t *testing.T) {
	seen := map[uint16]bool{}
	for _, r := range spellArms {
		if seen[r.arm] {
			t.Errorf("arm %d has two rows", r.arm)
		}
		seen[r.arm] = true
	}
	probed := map[uint16]bool{}
	for _, a := range armProbe() {
		probed[a] = true
	}
	for a := range seen {
		if !probed[a] {
			t.Errorf("arm %d has a row the probe does not ask", a)
		}
	}
}
