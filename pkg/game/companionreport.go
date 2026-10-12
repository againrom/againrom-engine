package game

// eventAudience is the audience an event text of this mission is read for: the
// campaign's own when it answers one, and otherwise the mission's, whose
// speaker half derives every hero speaker's sex from the starting hero.
func (mw *mapWorld) eventAudience(int) EventAudience {
	m := mw.mission
	if audience, ok := m.campaign().eventAudience(mw); ok {
		return audience
	}
	return m.audience
}

// secondGameAudience is the second game's speaker audience: a speaker is
// present while an on-map unit carries its key.
func (mw *mapWorld) secondGameAudience() EventAudience {
	m := mw.mission
	audience := m.audience
	audience.tags = m.campaign().eventTags()
	audience.NPCPresent = func(npc int) bool {
		if npc < 0 || npc > 65535 {
			return false
		}
		for id, key := range m.npcKeys {
			if int(key) == npc {
				if actor, exists := mw.world.Entity(id); exists && !actor.OffMap {
					return true
				}
			}
		}
		return false
	}
	return audience
}
