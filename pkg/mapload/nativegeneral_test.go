package mapload

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

func TestNativeFreshGeneralResistanceHasIndependentCurrentProducer(t *testing.T) {
	unit := nativeInitialBase(nil)
	if !unit.DefenceByteKnown(16) || unit.Defence[16] != 0 || unit.DefenceKnown != 1<<16 {
		t.Fatal("fresh Unit General policy missing or fabricated other defence bytes", unit)
	}
	var skills [data.SkillSlots]int32
	human := nativeInitialBase(&skills)
	worn := [sim.EquipSlots]sim.ItemInstance{}
	human = nativeInitialModifier(human, &worn, nil, 17, true, true)
	if !human.DefenceByteKnown(16) || human.Defence[16] != human.Modifier[58] {
		t.Fatal("fresh Human General lacks cleared defence plus modifier producer")
	}
}
