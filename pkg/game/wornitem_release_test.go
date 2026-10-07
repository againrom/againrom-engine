package game

import (
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// wornItemShield is the shield row at the installed definitions that carries
// the campaign's "Сила +1" enchantment in the merchants' mission.
const wornItemShield = uint16(0x9247)

// wornItemOpen opens mission 101 with a hero who carries the shield, given
// the effect it holds, and selects him.
func wornItemOpen(t *testing.T, effect sim.ItemEffect) (*FrontEnd, *ui.App, sim.EntityID) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	member := &party[0]
	worn := mapload.MemberItemEquipment(*member, f.Table)
	carried := mapload.MemberCarriedItems(*member, f.Table)
	item := mapload.ItemInstanceFromCode(wornItemShield, f.Table)
	item.Effects = []sim.ItemEffect{effect}
	carried = append(carried, item)
	syncMissionPartyStock(member, worn, carried)
	f.Carried = party
	app := f.App("worn item")
	app.Layout(1024, 768)
	if err := app.OpenMission(f.MissionOpener(101)); err != nil {
		t.Fatal(err)
	}
	return f, app, shieldWitnessSelect(t, f, app)
}

func wornItemWear(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) {
	t.Helper()
	stacks, _ := f.live.world.CarriedStacks(id)
	cell := -1
	for i, s := range stacks {
		if s.Code == wornItemShield {
			cell = i
		}
	}
	if cell < 0 {
		t.Fatalf("the pack holds no shield: %+v", stacks)
	}
	usePackCell(t, app, cell)
	for n := 0; n < 4; n++ {
		if err := stepConsumableApp(app); err != nil {
			t.Fatal(err)
		}
	}
}

// Wearing a shield that adds Body moves the Body the character panel states,
// together with the maximum health and to-hit it drives.
func TestReleaseWornBodyItemRaisesThePanelBody(t *testing.T) {
	f, app, id := wornItemOpen(t, sim.ItemEffect{Kind: 2, Operand: 1})
	before, _ := f.live.entity(id)
	bodyBefore := f.live.chars[id].Body
	wornItemWear(t, f, app, id)
	after, _ := f.live.entity(id)
	if got := f.live.chars[id].Body; got != bodyBefore+1 {
		t.Fatalf("panel Body %d after the Body +1 shield, want %d", got, bodyBefore+1)
	}
	if after.MaxHP <= before.MaxHP || after.ToHit <= before.ToHit {
		t.Fatalf("max health %d to %d, to-hit %d to %d: the derived numbers did not follow Body", before.MaxHP, after.MaxHP, before.ToHit, after.ToHit)
	}
}
