package game

import (
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/herokind"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// definitionName is the display name's definition stage for party member p:
// its companion name, else its mod name, else its class row, else fallback.
func (in *InstallResources) definitionName(p mapload.PartyMember, classIndex int, words *ui.Words, fallback string) string {
	if p.Membership().Is(mapload.CompanionMember) {
		return in.localizedNPCName(companionNameRow(p, in.Table), fallback)
	}
	if name, ok := in.modCharacterName(p.Name); ok {
		return name
	}
	// A second-game hire's ID names its NPC line.
	if m := p.Membership(); m.Is(mapload.CampaignMember) && m.Identity <= hireNameLines {
		if name, ok := ui.NPCName(m.Identity, words); ok {
			return name
		}
	}
	// An ordinary Human carries a table key such as M_Brigand3 as its name.
	if name, ok := ui.DisplayName(ui.InspectionUnit, "", classIndex, words); ok {
		return name
	}
	return fallback
}

// hireNameLines is how many hire IDs name an NPC line; the later IDs show
// their type label.
const hireNameLines = 7

// npcWord is the NPC word of entity id's placement, where the game keeps one.
func (mw *mapWorld) npcWord(id sim.EntityID) (uint16, bool) {
	if mw.mission == nil {
		return 0, false
	}
	word, ok := mw.mission.npcKeys[id]
	return word, ok && word != 0
}

// companionNameRow is companion p's row of the installed NPC names.
func companionNameRow(p mapload.PartyMember, table *mapload.Table) int {
	if !mapload.TableEdition(table).ComposedCompanion(p.CompanionID()) {
		return p.CompanionID() - 1
	}
	selector := -1
	for i, base := range data.ChargenBaseNames() {
		if p.Name == base || strings.HasPrefix(p.Name, base+"_") {
			selector = i
			break
		}
	}
	if selector < 0 {
		selector = herokind.Of(p.Mage, data.FigureDir(p.FigureDir).Female()).Index()
	}
	return 20 + selector
}

// drawnName is the display name's entity stage: the name the map draws for e
// with class c (its composed body when it has one), class row nameIndex.
// definitionCharName reports that the character's own name is only its
// definition row's, which the actor does not show.
func (mw *mapWorld) drawnName(e *sim.Entity, c *terrain.UnitClass, words *ui.Words) (name string, nameIndex int, definitionCharName bool) {
	if c != nil {
		name = c.Name
	}
	saved, hasSaved := mw.actorNames[e.ID]
	if hasSaved {
		name = saved
	}
	ordinary := (e.SourceBinding.ActorClass() == 1 || !sim.InPersistBand(e.TypeID)) &&
		mw.missionPartyMember(e.ID) == nil
	definitionName := ""
	if ordinary && mw.mission != nil {
		definitionName = sourceActorDefinitionName(mw.mission.table, e.SourceBinding)
	}
	sourceClassName := definitionName != "" && saved == definitionName
	char := mw.chars[e.ID]
	if ordinary && definitionName != "" && char.Name == definitionName {
		char.Name, definitionCharName = "", true
	}
	if char.Name != "" {
		name = char.Name
	}
	nameIndex = int(e.TypeID)
	nativeClassName := e.SourceBinding.Class == 0 && e.ActorLoad.Source.Class == 0 &&
		e.NativeBasis.HasValues() && c != nil && saved == c.Name
	if !ordinary && char.UnitNameIndex != 0 {
		nameIndex = char.UnitNameIndex
	}
	if (ordinary || char.Known) && char.Name == "" && (saved == "" || nativeClassName || sourceClassName) {
		if ordinary {
			name = ""
		}
		if className, ok := ui.DisplayName(ui.InspectionUnit, "", nameIndex, words); ok {
			name = className
		}
	}
	if member := mw.missionPartyMember(e.ID); member != nil && member.Membership().Is(mapload.CampaignMember) &&
		(char.Name == "" || char.Name == member.Name) && (saved == "" || saved == member.Name) {
		in := InstallResources{Table: mw.mission.table}
		name = in.definitionName(*member, nameIndex, words, "")
		definitionCharName = char.Name == member.Name
	}
	// A placed unit holding no own name, only its type label, takes the line
	// its NPC word names when that line is not empty.
	if ordinary {
		label, _ := ui.DisplayName(ui.InspectionUnit, "", nameIndex, words)
		if word, ok := mw.npcWord(e.ID); ok && (name == "" || name == label) {
			if line, ok := ui.NPCName(int(word), words); ok {
				name = line
			}
		}
	}
	// The map's own name for the placed unit is its last stage.
	if authored, ok := mw.authoredName(e); ok {
		name = authored
	}
	return name, nameIndex, definitionCharName
}

// shownName is the name the selection card shows for entity id, the name a
// message line gives the character it is about; "" when the card shows none.
func (mw *mapWorld) shownName(id sim.EntityID) string {
	e, ok := mw.world.Entity(id)
	if !ok {
		return ""
	}
	var classes map[int32]*terrain.UnitClass
	if mw.units != nil {
		classes = mw.units.Classes
	}
	c := classes[e.Class]
	if body := mw.art[e.ID]; body != nil {
		c = body
	}
	words := mw.drawnNameWords()
	name, nameIndex, _ := mw.drawnName(&e, c, words)
	name, _ = ui.DisplayName(ui.InspectionUnit, name, nameIndex, words)
	return name
}

// drawnNameWords is the word table the name stages read, nil without a view.
func (mw *mapWorld) drawnNameWords() *ui.Words {
	if mw.view == nil {
		return nil
	}
	words := mw.view.Words()
	return &words
}

func sourceActorDefinitionName(table *mapload.Table, binding sim.SourceBinding) string {
	if table == nil {
		return ""
	}
	var collection data.Collection
	switch binding.ActorClass() {
	case 1:
		collection = table.Units
	case 2:
		collection = table.Humans
	default:
		return ""
	}
	row := int(binding.DefinitionRow())
	if collection == nil || row >= collection.Len() {
		return ""
	}
	if binding.ActorClass() == 1 && row == 0 {
		if _, err := data.NewUnitDef(collection.EntryName(row), collection.EntryParams(row)); err != nil {
			row = data.FindUnit(collection, int32(uint8(binding.TypeID)), int32(binding.Face))
			if row == data.NotFound {
				for candidate := 1; candidate < collection.Len(); candidate++ {
					if collection.EntryName(candidate) != "Ghost" {
						continue
					}
					definition, err := data.NewUnitDef(collection.EntryName(candidate), collection.EntryParams(candidate))
					if err == nil && definition.TypeID == int32(uint8(binding.TypeID)) {
						row = candidate
					}
					break
				}
			}
			if row == data.NotFound {
				return ""
			}
			if _, err := data.NewUnitDef(collection.EntryName(row), collection.EntryParams(row)); err != nil {
				return ""
			}
		}
	}
	return collection.EntryName(row)
}
