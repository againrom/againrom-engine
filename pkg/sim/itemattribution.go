package sim

// sourceHasDefinition is the simulation's canonical counterpart of the
// original source-definition pointer. A nonzero TypeID is a resolved actor
// definition; zero is the unresolved/plain constructor state.
func sourceHasDefinition(e Entity) bool { return e.TypeID != 0 }

// resolveDamageAttribution is the shared damage resolver's pre-envelope
// writer. Missing source or definition retains history. A resolved source with
// no owner clears only the pointer. An owned source becomes the credit; a mage
// carries the damage kind while a fighter writes zero.
func (w *World) resolveDamageAttribution(ci, ti int, damageKind int8) {
	if ci < 0 || ci >= len(w.entities) || ti < 0 || ti >= len(w.entities) {
		return
	}
	source := &w.entities[ci]
	target := &w.entities[ti]
	if !sourceHasDefinition(*source) {
		return
	}
	if source.Owner == 0 {
		target.KillCreditSource = 0
		target.HasKillCredit = false
		return
	}
	target.KillCreditSource = source.ID
	target.HasKillCredit = true
	if skillMage(*source) {
		target.KillCreditSpell = damageKind
	} else {
		target.KillCreditSpell = 0
	}
}

// pointAttribution is the non-Defensive PointEffect tail. Drain Life builds no
// point envelope and bypasses it. Definition loss clears prior credit; an
// ownerless but still defined caster retains it.
func (w *World) pointAttribution(ci, ti int, rule SpellRule) {
	defensive := rule.Defensive
	if rule.bookInstance {
		defensive = rule.bookDefensive == 1
	}
	if rule.ID == 11 || defensive || ci < 0 || ci >= len(w.entities) || ti < 0 || ti >= len(w.entities) {
		return
	}
	source := &w.entities[ci]
	target := &w.entities[ti]
	if rule.HealHostile && source.Owner != target.Owner && w.hostileTo(*source, *target) {
		return
	}
	if !sourceHasDefinition(*source) {
		target.KillCreditSource = 0
		target.HasKillCredit = false
		return
	}
	if source.Owner == 0 {
		return
	}
	target.KillCreditSource = source.ID
	target.HasKillCredit = true
	target.KillCreditSpell = int8(rule.ID)
}

// areaAttribution is the distinct AreaEffect tail. Wall of Earth and Light do
// not reach it. Every canonical movement domain maps to the original nonzero
// byte range (our zero value is ground, not the original null domain), so the
// unsigned nonzero gate admits all three defined Domain values. Unlike point,
// either definition or owner loss clears prior credit.
func (w *World) areaAttribution(ci, ti int, rule SpellRule) {
	if rule.ID == 19 || rule.ID == 12 || ci < 0 || ci >= len(w.entities) || ti < 0 || ti >= len(w.entities) {
		return
	}
	if uint8(w.entities[ti].Domain)+1 == 0 { // unsigned gate; custom 0xff wraps to zero
		return
	}
	source := &w.entities[ci]
	target := &w.entities[ti]
	if !sourceHasDefinition(*source) || source.Owner == 0 {
		target.KillCreditSource = 0
		target.HasKillCredit = false
		return
	}
	target.KillCreditSource = source.ID
	target.HasKillCredit = true
	target.KillCreditSpell = int8(rule.ID)
}

// processNewKillCredits is the delayed death processor. beforeHP is captured
// at the head of the tick. A scan before decay catches direct deaths before a
// zero-dwell flying body can disappear; one after decay catches a downed
// actor's 0 -> -1 crossing. Deleting a consumed id makes the pair one logical
// pass, after every point/area tail and the fighter's physical award.
func (w *World) processNewKillCredits(beforeHP map[EntityID]int32) {
	for i := range w.entities {
		before, tracked := beforeHP[w.entities[i].ID]
		if tracked && before >= 0 && w.entities[i].Dead() {
			w.processKillCredit(i)
			delete(beforeHP, w.entities[i].ID)
		}
	}
}

func (w *World) processKillCredit(ti int) {
	target := &w.entities[ti]
	if target.Owner == 0 || !target.HasKillCredit {
		return
	}
	ci := indexOfEntity(w.entities, target.KillCreditSource)
	if ci < 0 {
		return
	}
	named := int32(0)
	if skillMage(w.entities[ci]) {
		if target.KillCreditSpell == 0 {
			return
		}
		id := int32(target.KillCreditSpell)
		if id <= 0 {
			return
		}
		rule, ok := w.findSpell(uint32(id))
		if !ok {
			return
		}
		named = int32(rule.School)
	}
	w.awardSkill(ci, named, int64(target.ExperienceValue())/2, ti)
}
