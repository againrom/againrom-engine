package game

import (
	"strings"

	"againrom/pkg/formats/sav"
)

// SAV-HEAD-025: the original LOAD door prepends Scenario to this name.
func originalMapName(address string) string {
	if len(address) >= 9 && strings.EqualFold(address[:8], "scenario") && (address[8] == '/' || address[8] == '\\') {
		return address[9:]
	}
	return address
}

func reservedParticipantName(name string) bool {
	return strings.EqualFold(name, "Self") || strings.EqualFold(name, "Computer")
}

// HERO-CHARGEN-084: authored slot sentinels cannot be participant join names.
// SAV-POSTLOAD-220: a mission's placed primary actor must not be placed again.
func repairSavedParticipants(doc *sav.DocumentData) {
	for _, index := range doc.Players {
		if index == 0 {
			continue
		}
		player := &doc.Objects[index-1]
		participant, _ := savedStructureValue(player, "Participant")
		if participant != 0 {
			continue
		}
		for i := range player.Texts {
			name := &player.Texts[i]
			if name.Name != "Name" || !reservedParticipantName(name.Value) {
				continue
			}
			hero, _ := savedStructureValue(player, "Hero")
			for j := range doc.Objects {
				actor := &doc.Objects[j]
				key, _ := savedStructureValue(actor, "Identity")
				if actor.Class != "Human" || key == 0 || key != hero {
					continue
				}
				for _, text := range actor.Texts {
					if text.Name == "Name" && text.Value != "" && !reservedParticipantName(text.Value) {
						name.Value = text.Value
						if doc.World != nil {
							savedObjectSetValue(player, "F3D", 1)
						}
					}
				}
			}
		}
	}
}
