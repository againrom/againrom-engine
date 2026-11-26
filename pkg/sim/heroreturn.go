package sim

const KindReturnHero uint8 = 0x49

const returnPenaltyTenths = 9

func returnedExperience(xp int32) int32 {
	return int32(uint32(xp)*returnPenaltyTenths) / 10
}

func (e *Entity) fillPools() {
	e.setCurrentHealth(e.MaxHP)
	e.setCurrentMana(e.MaxMana)
}

func (w *World) FillReturnedPools(id EntityID) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || !w.entities[i].Alive() {
		return false
	}
	w.entities[i].fillPools()
	return true
}

func (w *World) returnHero(i int, at CellPoint) {
	body := w.entities[i]
	if body.Alive() || !body.CorpseDecayed || body.ActorLoad.Source.Class != 0 {
		return
	}
	x, y, ok := nearestFreeCell(at.X, at.Y)(w, i)
	if !ok {
		return
	}
	hero := body
	hero.clearDecay()
	hero.HeldOrder = HeldOrder{}
	hero.CommandGroup = 0
	hero.setCurrentDefence(hero.Defence << 1)
	trained := hero.TrainedSkills(w.nativeSkillBonuses(i))
	sum := uint32(0)
	for slot := int32(1); slot < skillSlots; slot++ {
		hero.SkillXP[slot] = returnedExperience(hero.SkillXP[slot])
		sum += uint32(hero.SkillXP[slot])
		level := w.game.skillAward.heldLevel(w.rules, hero.SkillXP[slot])
		trained[slot] = level
		hero.Skill[slot] = w.rules.EffectiveSkill(level, w.wornSkillBonus(i, slot))
		if hero.NativeBasis.BasePresent {
			at := 2 + 2*slot
			hero.NativeBasis.Base[at], hero.NativeBasis.Base[at+1] = byte(level), byte(level>>8)
			hero.NativeBasis.BaseKnown |= uint32(3) << at
		}
	}
	if hero.NativeBasis.ScalarIsKnown(ScalarU130) {
		hero.NativeBasis.Scalars[ScalarU130] = sum
	}
	hero.NativeTraining = NativeTraining{Present: true, Levels: trained}
	RefreshBook(w.rules, &hero, w.spells)
	hero.fillPools()
	pack, worn := cloneStacks(w.carried[i]), cloneEquipment(w.equipment[i])
	if !w.remove([]EntityID{body.ID}) {
		return
	}
	w.takePooledDead(body.ID)
	w.createActor(hero, pack, worn, x, y, nearestFreeCell(x, y), false, actorCreationProfile{reuse: body.ID, keepsID: true})
}
