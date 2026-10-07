package mapload_test

import (
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestScriptCompanionReferenceIsExplicitAndMayNameEntityZero(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{
		node("primary", 4, 1, par(4, 10001)),
		node("companion", 4, 2, par(4, 10002)),
		node("other slot", 4, 3, par(4, 10003)),
	}}
	bound, report := compile(t, src, mapload.ScriptRefs{Hero: 71, HasHero: true, Companion: 0, HasCompanion: true})
	checks := bound.Checks()
	if !checks[0].HasUnit || checks[0].Unit != 71 || !checks[1].HasUnit || checks[1].Unit != 0 || checks[2].HasUnit {
		t.Fatalf("explicit hero bindings: %+v", checks)
	}
	if len(report.Unresolved) != 1 || report.Unresolved[0].Value != 10003 {
		t.Fatalf("remaining references: %+v", report.Unresolved)
	}
	absent, _ := compile(t, src, mapload.ScriptRefs{Hero: 71, HasHero: true})
	if absent.Checks()[1].HasUnit {
		t.Fatal("absent companion aliased the primary")
	}
}

func TestScriptNamedRoleMayNameEntityZeroWithoutBindingOtherOrdinals(t *testing.T) {
	src := alm.Script{Conditions: []alm.ScriptNode{
		node("role", 4, 1, par(4, 10006)),
		node("unknown", 4, 2, par(4, 10005)),
		node("map unit", 4, 3, par(4, 77)),
	}}
	bound, report := compile(t, src, mapload.ScriptRefs{Roles: map[uint32]sim.EntityID{10006: 0, 77: 99}})
	checks := bound.Checks()
	if !checks[0].HasUnit || checks[0].Unit != 0 || checks[1].HasUnit || checks[2].HasUnit {
		t.Fatalf("named role escaped its reference band: %+v", checks)
	}
	if len(report.Unresolved) != 2 {
		t.Fatalf("unknown references=%+v", report.Unresolved)
	}
}
