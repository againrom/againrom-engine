package sim

// HumanExperienceValue is ITEM-ARMFOLD-033's actor+0x1c value: one percent
// of the five class-skill experience counters. General is excluded.
func HumanExperienceValue(xp [skillSlots]int32) int32 {
	var total int64
	for i := 1; i < len(xp); i++ {
		total += int64(xp[i])
	}
	return int32(total / 100)
}

// ExperienceValue supplies the target value shared by hit, spell and kill
// awards. Humans derive it from live skill progress; flat Units retain their
// installed template value. Reading it here also repairs old native saves
// whose Human XPValue still contains the base constructor's zero.
func (e Entity) ExperienceValue() int32 {
	if e.Humanoid || e.ActorLoad.Source.Class == 2 {
		return HumanExperienceValue(e.SkillXP)
	}
	return e.XPValue
}
