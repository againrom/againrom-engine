package game

import (
	"strings"
	"testing"

	"againrom/pkg/mapload"
)

// The tavern's Catapult and Ballista each hold a weapon whose attached spell is
// Fire_Ball, so both reach the weapon-rider path (MAGIC-247). The weapon is
// resolved through the production hire and loadout path on the installed
// Units rows; the EN rows hold powers 70 and 40.
func TestReleaseSiegeEnginesCarryAFireBallWeaponSpell(t *testing.T) {
	f := releaseFront(t)
	f.Town = NewTown(f.Campaign.Value())
	s := f.TownScreen().(*townScreen)
	s.room = roomTavern
	for typ, name := range map[int]string{1: "Catapult", 2: "Ballista"} {
		members, ok := s.buildMercenarySquad(typ, 1)
		if !ok || len(members) != 1 {
			t.Fatalf("%s: the production template is unavailable", name)
		}
		hire, err := mapload.SiegeHireActor(members[0], f.Table, 0)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		spell, power, ok := hire.Worn[0].CastSpell()
		if !ok || spell == 0 || int(spell) >= f.Table.Spells.Len() {
			t.Fatalf("%s: the worn weapon carries no spell (id %d)", name, spell)
		}
		got := strings.ReplaceAll(f.Table.Spells.EntryName(int(spell)), " ", "_")
		if got != "Fire_Ball" || power <= 0 {
			t.Errorf("%s: weapon spell %q power %d, want Fire_Ball with a power", name, got, power)
		}
		t.Logf("%s: %s %d", name, got, power)
	}
}
