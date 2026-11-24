package game

import (
	"testing"

	"againrom/pkg/scriptprofile"
)

// TestVocabularyRulesReachTheirBodies guards that each operation vocabulary's
// rule values index a body in the game's tables, so no route reads a rule
// that has no body.
func TestVocabularyRulesReachTheirBodies(t *testing.T) {
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		v := scriptprofile.VocabularyOf(ops)
		kind := kindRuleBodies[v.Kinds]
		if kind.copyPage == nil || kind.parameterKind == nil {
			t.Errorf("ops %d: kind rule %d has no body", ops, v.Kinds)
		}
		if operationNameBodies[v.Naming] == nil {
			t.Errorf("ops %d: naming rule %d has no body", ops, v.Naming)
		}
	}
}
