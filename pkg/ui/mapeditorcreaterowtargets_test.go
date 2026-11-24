package ui

import "testing"

// TestCreateRowsAreNotRecordTargets pins that the helpers which pick a record
// skip every create row of the table.
func TestCreateRowsAreNotRecordTargets(t *testing.T) {
	e := &MapEditor{doc: &InspectionDocument{Script: &InspectionScriptModel{Targets: map[uint32][]InspectionScriptTarget{
		ScriptGlobalParameter: {{Record: -1, NewChoice: NewGlobalChoice}, {Record: 0, Value: 5}},
	}}}}
	if got := e.firstTarget(ScriptGlobalParameter, 9); got != 5 {
		t.Fatalf("first target %d, want the record 5 past the create row", got)
	}
}
