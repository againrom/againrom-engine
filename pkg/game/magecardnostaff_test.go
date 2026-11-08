package game

import (
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func cardText(s ui.PanelSubject) string {
	return strings.Join(ui.PanelStatement(ui.CompactPanelLayout(nil), s), "\n")
}

func TestSecondGameMageWithoutStaffDrawsNoDamageRow(t *testing.T) {
	table, weapon := staffTooltipTable(t, 3, "Fire Arrow", 10, 5, 15, 18, 0)
	edition := base.GameROM2.Edition()
	table.Edition = &edition
	for name, hero := range map[string]data.Hero{
		"strong": {Body: 60, Reaction: 22, Mind: 51, Spirit: 36},
		"weak":   {Body: 1, Reaction: 1, Mind: 51, Spirit: 36},
	} {
		member := mapload.PartyMember{Mage: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero}
		if got := cardText(partyPanelSubject(member, table)); strings.Contains(got, "DMG") {
			t.Errorf("mage without staff (%s) card = %q, want no DMG row", name, got)
		}
	}
	member := mapload.PartyMember{Mage: true, Profile: data.Profile{HealthColumn: true, ManaColumn: true},
		Hero: data.Hero{Body: 30, Reaction: 22, Mind: 51, Spirit: 36}, Weapon: weapon}
	member.Worn[0] = uint16(weapon.Code)
	if got := cardText(partyPanelSubject(member, table)); !strings.Contains(got, "DMG") {
		t.Errorf("mage with staff card = %q, want a DMG row", got)
	}
	fighter := mapload.PartyMember{Profile: data.Profile{HealthColumn: true},
		Hero: data.Hero{Body: 30, Reaction: 22, Mind: 10, Spirit: 10}}
	if got := cardText(partyPanelSubject(fighter, table)); !strings.Contains(got, "DMG") {
		t.Errorf("fighter with empty hands card = %q, want a DMG row", got)
	}
}
