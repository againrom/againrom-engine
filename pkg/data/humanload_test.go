package data

import "testing"

func TestHumanSaleEqualQuotientsPreserveAllOtherFields(t *testing.T) {
	for _, v := range []struct {
		name                   string
		load, weight, capacity uint16
		inventory              int32
		want                   uint16
	}{
		{"ordinary", 90, 40, 411, 14, 47},
		{"negative-odd", 0, 40, 411, -15, 33},
		{"negative-load", 65530, 65534, 411, -15, 65527},
		{"negative-capacity", 90, 40, 65125, 14, 47},
		{"threshold", 32000, 60000, 32001, 64000, 32000},
		{"before-threshold", 32039, 40, 32040, 63999, 32039},
		{"word-wrap", 65530, 40000, 32767, 52000, 464},
	} {
		t.Run(v.name, func(t *testing.T) {
			h := trainingHuman()
			h.Load, h.Weight, h.Capacity, h.InventoryWeight = v.load, v.weight, v.capacity, v.inventory
			want := h
			want.Load = v.want
			n, derived, err := h.RefreshInventoryLoad()
			if err != nil || derived || n != want {
				t.Fatalf("got=%+v derived=%t err=%v want=%+v", n, derived, err, want)
			}
		})
	}
}

func TestHumanSaleChangedQuotientDerivesWithoutTraining(t *testing.T) {
	h := trainingHuman()
	h.Load, h.Capacity = 800, 400
	n, derived, err := h.RefreshInventoryLoad()
	if err != nil || !derived {
		t.Fatal(derived, err)
	}
	// trunc(82+log(1.46)/log(1.1)*2)=89; trunc(89*(1.1^41/100+1))+4=137.
	if n.Load != 490 || n.Speed != 20 || n.Capacity != 414 || n.Attack.Skill[1] != 11 || n.Attack.SecondBase != 6 || n.Attack.SecondSpread != 7 || n.HealthMax != 137 || n.Health != 137 {
		t.Fatalf("conditional derive result %+v", n)
	}
	if n.Base != h.Base || n.SkillXP != h.SkillXP || n.Experience != h.Experience || n.Modifier != h.Modifier || n.Attack.Active != h.Attack.Active || n.Attack.Tail != h.Attack.Tail {
		t.Fatal("sale trained or rebuilt the modifier")
	}
}

func TestHumanSaleFaultsReturnUnchangedState(t *testing.T) {
	for _, zero := range []bool{true, false} {
		h := trainingHuman()
		h.Load, h.Capacity = 800, 400
		if zero {
			h.Capacity = 0
		} else {
			h.Experience = 0x80000000
		}
		n, derived, err := h.RefreshInventoryLoad()
		if err == nil || derived || n != h {
			t.Fatal("fault changed input", n, derived, err)
		}
	}
}
