package ui

import (
	"testing"

	"againrom/pkg/scriptprofile"
)

// TestFieldRuleReachesItsBody guards that each operation vocabulary's field
// rule indexes a complete body in the node table.
func TestFieldRuleReachesItsBody(t *testing.T) {
	for _, ops := range []scriptprofile.Ops{scriptprofile.OpsFirst, scriptprofile.OpsSecond} {
		rule := scriptprofile.VocabularyOf(ops).Fields
		b := nodeRuleBodies[rule]
		if b.fields == nil || b.listed == nil || b.notice == nil {
			t.Errorf("ops %d: field rule %d has an incomplete body", ops, rule)
		}
	}
}
