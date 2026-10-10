package game

import (
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func companionReportAudience(base EventAudience, mission, event int, defs *data.NPCDefs, faces map[int32]data.NPCFace, cast speakerCast) EventAudience {
	rec, exists := faces[23]
	predicate := data.NPCTokens(data.NPCTokenHero | data.NPCTokenNotMage | data.NPCTokenNotMySex)
	if mission != 70 || event != 2 || !exists || rec.Kind != data.NPCNoPicture || !rec.Archetype || rec.Start || rec.Tokens != predicate || !cast.hasPlayer {
		return base
	}
	mage, female, composed := defs.ComposedArchetype(23, cast.playerDir.Mage(), cast.playerDir.Female())
	if !composed || mage || female == cast.playerDir.Female() {
		return base
	}
	actor, alive := cast.resolve(rec)
	if !alive {
		return base
	}
	generic := base.Speaker
	base.Speaker = func(npc int) (bool, bool, bool) {
		if npc == 23 {
			return actor.fig.Dir.Female(), actor.fig.Dir.Mage(), true
		}
		if generic != nil {
			return generic(npc)
		}
		return false, false, false
	}
	return base
}

func (mw *mapWorld) eventAudience(event int) EventAudience {
	m := mw.mission
	if audience, ok := m.campaign().eventAudience(mw); ok {
		return audience
	}
	cast := speakerCast{actors: mw.speakerActors, alive: mw.entityAlive}
	cast.playerDir, cast.hasPlayer = mw.playerFigureDir()
	var defs *data.NPCDefs
	if m.table != nil {
		defs = m.table.NPC
	}
	return companionReportAudience(m.audience, m.number, event, defs, mw.npcFaces, cast)
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

func scenarioEventAudience(base EventAudience, ms *Mission, event int, table *mapload.Table, faces map[int32]data.NPCFace) EventAudience {
	if ms.Number != 70 || event != 2 || ms.World == nil || table == nil {
		return base
	}
	cast := speakerCast{
		actors: missionSpeakers(ms.Map, table, ms.Start.Roster, ms.Party, ms.Start.IDs,
			placedEntities(ms.Map, ms.World.Entities(), ms.Start.Roster, ms.savedDocument)),
		alive: func(id sim.EntityID) bool {
			actor, exists := ms.World.Entity(id)
			return exists && actor.Alive()
		},
	}
	for _, member := range ms.Party {
		if primaryPlayerHero(member) {
			cast.playerDir, _ = memberFigure(member)
			cast.hasPlayer = true
			break
		}
	}
	return companionReportAudience(base, ms.Number, event, table.NPC, faces, cast)
}
