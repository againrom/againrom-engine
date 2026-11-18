package game

import (
	"image"
	"math/rand"
	"os"
	"strings"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func scrollOfSpell(t *testing.T, f *FrontEnd, name string) sim.ItemInstance {
	t.Helper()
	var seen []string
	for _, item := range shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1090))) {
		if got, _, ok := sim.ScrollSpell(item.Instance()); ok {
			spell := f.Table.Spells.EntryName(int(got))
			seen = append(seen, spell)
			if strings.EqualFold(spell, name) {
				return item.Instance()
			}
		}
	}
	rows := f.Table.MagicItems
	for i := 1; i < rows.Len(); i++ {
		item := mapload.ItemInstanceFromCode(uint16(0x0e00|i), f.Table)
		if got, _, ok := sim.ScrollSpell(item); ok {
			spell := f.Table.Spells.EntryName(int(got))
			seen = append(seen, spell+"/"+rows.EntryName(i))
			if strings.EqualFold(spell, name) {
				return item
			}
		}
	}
	t.Fatalf("installed content has no scroll of %q among %q", name, seen)
	return sim.ItemInstance{}
}

// scrollRouteMission opens the installed game's first playable mission with
// the party's first member carrying only item, and returns the front.
func scrollRouteMission(t *testing.T, item func(*FrontEnd) sim.ItemInstance) (*FrontEnd, *ui.App) {
	t.Helper()
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS")
	}
	archives, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	if archives.Game() != base.GameROM2 {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		scroll := item(f)
		party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
		party[0].KnownSpells = 0
		party[0].CarriedItems = []sim.ItemInstance{scroll}
		party[0].Carried = []uint16{scroll.Code}
		f.Carried = party
		a := f.App("scroll inventory cast")
		a.Layout(1024, 768)
		if err := a.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		return f, a
	}
	f := secondGameFront(t)
	scroll := item(f)
	a := f.App("scroll inventory cast")
	a.Layout(1024, 768)
	c := newSecondCampaign()
	c.bank[769] = 1
	c.available = append(c.available, partyPlace{kind: placeMission, id: 42})
	if f.Town == nil {
		f.Town = NewTown(Campaign{})
	}
	f.Town.second = c
	party := MissionParty(f.StartWeapon.Value(), f.Bodies, f.Table)
	party[0].KnownSpells = 0
	party[0].CarriedItems = []sim.ItemInstance{scroll}
	party[0].Carried = []uint16{scroll.Code}
	f.Carried = party
	screen := f.TownScreen().(*townScreen)
	a.SetTown(screen)
	destinationTownCampaign{screen}.leave()
	if err := a.OpenMission(f.MissionOpener(42)); err != nil || f.live == nil {
		t.Fatalf("mission 42: %v %s", err, a.HeadlessMessage())
	}
	return f, a
}

func carriedUnits(mw *mapWorld, id sim.EntityID) int {
	stacks, _ := mw.world.CarriedStacks(id)
	n := 0
	for _, s := range stacks {
		n += int(s.Count)
	}
	return n
}

// TestInventoryScrollRoutesReachTheCast drives a carried scroll through the
// pack's double click and through a key bound to it, then clicks the map: the
// scroll is armed, the click makes the order, the scroll is used once and the
// spell takes effect.
func TestInventoryScrollRoutesReachTheCast(t *testing.T) {
	for _, spell := range []struct {
		name string
		self bool
	}{{"Teleport", false}, {"Fire Ball", false}, {"Shield", true}} {
		for _, route := range []string{"doubleclick", "boundkey"} {
			t.Run(spell.name+"/"+route, func(t *testing.T) {
				f, a := scrollRouteMission(t, func(f *FrontEnd) sim.ItemInstance { return scrollOfSpell(t, f, spell.name) })
				mw := f.live
				id := mw.mission.ids[0]
				caster := releaseEntity(t, mw, id)
				planes := mapload.Planes(mw.mission.state.Map, f.Table)
				var dest image.Point
				for radius := int32(4); radius <= 6 && dest == (image.Point{}); radius++ {
					if x, y, ok := creatureRingCell(mw.world, planes, mw.mission.state.Map.Width, mw.mission.state.Map.Height, caster.X, caster.Y, radius); ok {
						dest = image.Pt(int(x), int(y))
					}
				}
				if dest == (image.Point{}) {
					t.Fatal("no free destination four to six cells away")
				}
				openMissionPack(t, a)
				for n := 0; n < 32; n++ {
					if err := stepConsumableApp(a); err != nil {
						t.Fatal(err)
					}
				}
				if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
					t.Fatal(err)
				}
				mw.view.Camera().CenterOn(float64(caster.X*32), float64(caster.Y*32))
				px, py, err := a.HeadlessPackCellPoint(0)
				if err != nil {
					t.Fatal(err)
				}
				switch route {
				case "doubleclick":
					for range 2 {
						for _, edge := range []string{"press", "release"} {
							if err := a.HeadlessPointer(edge, px, py); err != nil {
								t.Fatal(err)
							}
						}
					}
				case "boundkey":
					if err := a.HeadlessPointer("hover", px, py); err != nil {
						t.Fatal(err)
					}
					for _, key := range []string{"shift-f4", "f4"} {
						if err := a.HeadlessKey(key); err != nil {
							t.Fatal(err)
						}
					}
				}
				if err := a.HeadlessStep(); err != nil {
					t.Fatal(err)
				}
				for i := range mw.fog.visible {
					mw.fog.visible[i], mw.fog.explored[i] = 1, 1
				}
				mw.push()
				var x, y int
				if spell.self {
					x, y, err = a.HeadlessEntityPoint(uint32(id))
				} else {
					x, y, err = a.HeadlessCellPoint(dest.X, dest.Y)
				}
				if err != nil {
					t.Fatal(err)
				}
				if err := a.HeadlessPointer("hover", x, y); err != nil {
					t.Fatal(err)
				}
				cursor, _ := a.HeadlessMapCursor()
				for _, edge := range []string{"press", "release"} {
					if err := a.HeadlessPointer(edge, x, y); err != nil {
						t.Fatal(err)
					}
				}
				var queued []uint8
				for _, c := range mw.pending {
					queued = append(queued, c.Kind)
				}
				for range 400 {
					mw.tick()
				}
				e := releaseEntity(t, mw, id)
				t.Logf("cursor %q orders %v units %d at %d,%d from %d,%d dest %v", cursor, queued, carriedUnits(mw, id), e.X, e.Y, caster.X, caster.Y, dest)
				if len(queued) != 1 || queued[0] != sim.KindUseScrollAt && queued[0] != sim.KindUseScroll {
					t.Fatalf("the click did not order the scroll: %v", queued)
				}
				if carriedUnits(mw, id) != 0 {
					t.Fatalf("scroll not used once: %d left", carriedUnits(mw, id))
				}
				if spell.name == "Teleport" && (int(e.X) != dest.X || int(e.Y) != dest.Y) {
					t.Fatalf("teleport did not land on %v: at %d,%d", dest, e.X, e.Y)
				}
			})
		}
	}
}
