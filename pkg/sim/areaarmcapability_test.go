package sim

import "testing"

func armProbe() []uint16 {
	out := []uint16{ArmNone, armSummon, 0x8000}
	for a := uint16(0); a <= 64; a++ {
		out = append(out, a)
	}
	return out
}

func TestAreaArmCapabilitiesKeepEachSiteAnswer(t *testing.T) {
	type overlay struct {
		slot          int
		second, drawn bool
	}
	for _, a := range armProbe() {
		r := areaRow(a)
		for name, c := range map[string][2]bool{
			"blocks":        {areaArmBlocks(a), a == 19},
			"inert":         {r.inert, a == 19},
			"inert export":  {AreaArmInert(a), a == 19},
			"unattributed":  {r.unattributed, a == 19 || a == 12},
			"uncapped":      {r.uncapped, a == 3},
			"sacrifice":     {r.sacrifice, a == 4},
			"footprint":     {r.footprint, a == 2},
			"onMiss":        {r.onMiss, a == 2},
			"blow":          {r.blow, a == 17},
			"poison":        {r.poison, a == 8},
			"secondLasting": {r.secondLasting, a == 8},
			"scorches":      {r.scorches, a == 2 || a == 3},
			"ambient":       {AreaArmAmbient(a), a == 3},
			"magnitude":     {MagnitudeArm(a), todayMagnitudeArm(a)},
			"duration":      {DurationArm(a), todayDurationArm(a)},
		} {
			if c[0] != c[1] {
				t.Errorf("arm %d %s = %v, today %v", a, name, c[0], c[1])
			}
		}
		want := map[uint16]overlay{3: {0, false, true}, 19: {1, false, true}, 7: {2, true, true}, 8: {3, true, true}}[a]
		if slot, second, drawn := AreaArmOverlay(a); (overlay{slot, second, drawn}) != want {
			t.Errorf("arm %d overlay = %d %v %v, today %v", a, slot, second, drawn, want)
		}
		if got, want := int32(areaRow(SecondGameSpellArm(a)).layer), todayROM2Layer(a); got != want {
			t.Errorf("second-game spell %d layer = %d, today %d", a, got, want)
		}
		if got, want := creatureAimOf(uint32(a)), todayCreatureAim(uint32(a)); got != want {
			t.Errorf("creature aim %d = %d, today %d", a, got, want)
		}
		for _, second := range []bool{false, true} {
			rule := SpellRule{ID: a, Second: second}
			for _, power := range []int32{-50, -1, 0, 1, 7, 14, 15, 29, 30, 44, 45, 90, 255, 1000} {
				for _, mag := range []int32{-7, 0, 3, 10} {
					got := mag
					if i := int(rule.arm()); i < len(pointMagnitudes) && pointMagnitudes[i] != nil {
						got = pointMagnitudes[i](rule, mag, power)
					}
					if want := todayPointMagnitude(rule, mag, power); got != want {
						t.Fatalf("arm %d second=%v power %d mag %d: magnitude %d, today %d", a, second, power, mag, got, want)
					}
				}
				rule.SpellDuration, rule.EffectDuration = 4, 9
				if got, want := spellPointDurationUnder(Rules{}, rule, power), todayPointDuration(Rules{}, rule, power); got != want {
					t.Fatalf("arm %d second=%v power %d: duration %d, today %d", a, second, power, got, want)
				}
			}
		}
	}
	for _, id := range []uint32{65, 1 << 16, 1<<32 - 1} {
		if creatureAimOf(id) != creatureAimNone {
			t.Errorf("creature aim %d is not none", id)
		}
	}
}

func todayMagnitudeArm(id uint16) bool {
	switch id {
	case 5, 7, 8, 10, 12, 16, 17, 18, 22, 23, 24, 27, 28:
		return true
	}
	return false
}

func todayDurationArm(id uint16) bool {
	switch id {
	case 5, 10, 15, 16, 18, 20, 22, 23, 24, 27, 28:
		return true
	}
	return false
}

func todayROM2Layer(spell uint16) int32 {
	l := int32(-1)
	switch spell {
	case 3:
		l = 0
	case 6:
		l = 2
	case 17:
		l = 3
	case 15:
		l = 4
	case 14:
		l = 5
	}
	return l
}

func todayCreatureAim(id uint32) creatureAim {
	switch id {
	case 1, 11, 13, 14, 20, 27, 28:
		return creatureAimVictim
	case 5, 6, 10, 15, 16, 18, 22, 23, 24:
		return creatureAimCaster
	case 2, 3, 7, 8, 12, 17, 19, 21:
		return creatureAimVictimCell
	case 9, 26:
		return creatureAimStepCell
	}
	return creatureAimNone
}

func todayPointMagnitude(rule SpellRule, mag, power int32) int32 {
	switch rule.arm() {
	case 5, 10, 16, 22:
		mag = power / 2
	case 18:
		mag = power/10 + 3
	case 24:
		mag = power/15 + 1
	case 7, 28:
		mag = -(power/15 + 1)
	case 8:
		if rule.Second {
			mag = mag * (power + 45) / 45
		} else {
			mag = PoisonMagnitude(mag, power)
		}
	case 12:
		mag = power/30 + 1
	case 17:
		mag = -1 - power/30
	}
	return mag
}

func todayPointDuration(r Rules, rule SpellRule, power int32) uint16 {
	switch rule.arm() {
	case 15, 5, 10, 16, 18, 20, 22, 23, 24, 27, 28:
		return spellLastingTicksUnder(r, rule, power)
	case 8:
		if rule.Second {
			return spellLastingTicksUnder(r, rule, power)
		}
	}
	return rule.EffectDuration
}
