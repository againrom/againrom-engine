package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

func TestBuyingAndWearingInTheShopMovesTheWorldSpriteToo(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Bodies = data.BodyList{data.BodyUnarmed, data.BodySwordsman, data.BodyAxeman2H}
	f.Carried[0].Body = string(data.BodyUnarmed)
	f.Carried[0].BodyDir = data.HeroDirHeroes

	// Row 2 of the list above, and armour material 8, whose block is the light
	// directory. Neither is worn yet.
	weapon := uint16(data.ComposeItemCode(0, 0, 0, 2))
	armour := uint16(data.ComposeItemCode(8, 0, 0, 1))

	s.shopWearInto(1, weapon)
	if got := f.Carried[0].Body; got != string(data.BodySwordsman) {
		t.Errorf("after wearing a row-2 weapon the drawn body is %q, want %q",
			got, data.BodySwordsman)
	}

	s.shopWearInto(data.HeroArmourSlot, armour)
	if got := f.Carried[0].BodyDir; got != data.HeroDirHeroesLight {
		t.Errorf("after wearing material-8 armour the body directory is %q, want %q",
			got, data.HeroDirHeroesLight)
	}
	// The weapon is still on, so the body name must not have fallen back.
	if got := f.Carried[0].Body; got != string(data.BodySwordsman) {
		t.Errorf("armour changed the drawn body to %q, want %q", got, data.BodySwordsman)
	}

	// AND THE WAY BACK. Taking the weapon off returns the bare-handed name, so
	// this fails against a build that writes the fields once and then latches.
	s.shopUnequipDoll(1)
	if got := f.Carried[0].Body; got != string(data.BodyUnarmed) {
		t.Errorf("after taking the weapon off the drawn body is %q, want %q",
			got, data.BodyUnarmed)
	}
}

// TestTheTownAndTheMapDeriveOneAppearanceFromOneEquipment states the property
// the test above is one case of: whatever the town leaves on the member,
// data.HeroAppearance over that member's own equipment must already agree with
// it, because that is exactly what the mission's own partyArt will read.
func TestTheTownAndTheMapDeriveOneAppearanceFromOneEquipment(t *testing.T) {
	f, s := shopRoom(t, nil)
	f.Bodies = data.BodyList{data.BodyUnarmed, data.BodySwordsman, data.BodyAxeman2H}
	s.shopWearInto(1, uint16(data.ComposeItemCode(0, 0, 0, 3)))
	s.shopWearInto(data.HeroArmourSlot, uint16(data.ComposeItemCode(2, 0, 0, 1)))

	member := f.Carried[0]
	eq := mapload.EquipmentFromParty(member)
	body, dir, class, matched := data.HeroAppearance(f.Bodies, eq, member.Mage, false)
	if !matched {
		t.Fatal("the fixture's own equipment resolves to no body at all")
	}
	if member.Body != string(body) || member.BodyDir != dir || member.Class != class {
		t.Errorf("town left (%q, %q, %d); the mission would draw (%q, %q, %d)",
			member.Body, member.BodyDir, member.Class, body, dir, class)
	}
}
