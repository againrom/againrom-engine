package game

import (
	"bytes"
	"testing"

	"againrom/pkg/sim"
)

// Installed actors, equipment, map, UI dispatch and native LOAD remain real.
// Only hero placement and revealed presentation fog are fixture controls. The
// sack is made by dropping one actual carried/equipped item, not by patching
// the pickup result or actor state.
func TestReleasePickup1117ClickKeyAndNativeBoundaries(t *testing.T) {
	for _, route := range []string{"click-walk", "underfoot-F"} {
		t.Run(route, func(t *testing.T) {
			f, app, mw, hero := openReleaseGroundMission(t, "1117 "+route)
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			livePlaceAndWalk(t, mw, hero, 20, 59)
			stock, _ := mw.world.CarriedStacks(hero)
			drop := sim.Command{Kind: sim.KindDropCarried, Entity: hero, X: 22, Y: 60}
			if len(stock) == 0 {
				worn, _ := mw.world.EquippedItems(hero)
				found := false
				for slot, item := range worn {
					if item.Code != 0 {
						drop.Kind, drop.Spell, found = sim.KindDropWorn, uint16(slot+1), true
						break
					}
				}
				if !found {
					t.Fatal("installed hero has no item to drop")
				}
			}
			sim.Step(mw.world, []sim.Command{drop})
			mw.rearm()
			sack, ok := mw.sackAt(22, 60)
			if !ok || len(sack.Items) != 1 {
				t.Fatalf("ordinary drop did not create one-item sack: %+v", sack)
			}
			beforeItems, _ := mw.world.Carried(hero)
			for i := range mw.fog.visible {
				mw.fog.visible[i], mw.fog.explored[i] = 1, 1
			}
			mw.push()
			mw.view.Camera().CenterOn(21*32, 60*32)
			if err := app.HeadlessSelectEntity(uint32(hero)); err != nil {
				t.Fatal(err)
			}
			if route == "click-walk" {
				entryPointer1084(t, app, 22, 60)
				if !mw.pickup.set || mw.pickup.id != hero || mw.pickup.x != 22 || mw.pickup.y != 60 {
					t.Fatalf("map pointer did not arm ordinary pickup: %+v", mw.pickup)
				}
				if err := app.HeadlessKey("0"); err != nil {
					t.Fatal(err)
				}
				for n := 0; n < 400; n++ {
					if _, exists := mw.sackAt(22, 60); !exists {
						break
					}
					if err := entryStep1084(app); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				liveWalk(t, mw, hero, 22, 60)
				mw.push()
				if _, _, up := f.LiveNotice(); up {
					if err := app.HeadlessActivate("notice"); err != nil {
						t.Fatal(err)
					}
				}
				if err := app.HeadlessKey("f"); err != nil {
					t.Fatal(err)
				}
			}
			e, _ := mw.entity(hero)
			if e.ActorState != 2 || e.HasTarget || e.HasAttackTarget || e.X != 22 || e.Y != 60 {
				t.Fatalf("%s transfer boundary: state%d target%v attack%v cell%d,%d", route, e.ActorState, e.HasTarget, e.HasAttackTarget, e.X, e.Y)
			}
			if _, exists := mw.sackAt(22, 60); exists {
				t.Fatal("successful transfer kept sack")
			}
			afterItems, _ := mw.world.Carried(hero)
			if len(afterItems) != len(beforeItems)+1 {
				t.Fatalf("pickup count before%d after%d", len(beforeItems), len(afterItems))
			}
			transferTick, transfer := mw.world.Tick(), pickupForm1117(t, mw.world)
			fresh, freshApp := cellAppSaveFresh1106(t, f, app, SaveStore{Dir: t.TempDir()}, route+" transfer")
			if !bytes.Equal(transfer, pickupForm1117(t, fresh.live.world)) {
				t.Fatal("menu SAVE/fresh LOAD advanced transfer")
			}
			// This is the ordinary mapWorld tick supplied to the App, not a
			// marker mutation or a test-only completion call.
			mw.tick()
			fresh.live.tick()
			e, _ = mw.entity(hero)
			if e.ActorState != 0x0c || mw.world.Tick() != transferTick+1 || !bytes.Equal(pickupForm1117(t, mw.world), pickupForm1117(t, fresh.live.world)) {
				t.Fatalf("next gameplay tick did not complete identically: state%d tick%d", e.ActorState, mw.world.Tick())
			}
			completion := pickupForm1117(t, fresh.live.world)
			loaded, _ := cellAppSaveFresh1106(t, fresh, freshApp, SaveStore{Dir: t.TempDir()}, route+" completion")
			if !bytes.Equal(completion, pickupForm1117(t, loaded.live.world)) {
				t.Fatal("completion SAVE/LOAD advanced world")
			}
			for n := 0; n < 33; n++ {
				mw.tick()
				loaded.live.tick()
				if mw.world.Hash() != loaded.live.world.Hash() || !bytes.Equal(pickupForm1117(t, mw.world), pickupForm1117(t, loaded.live.world)) {
					t.Fatalf("successor%d differs after completion LOAD", n+1)
				}
			}
			t.Logf("%s: one item transferred at tick%d; state2->12 at tick%d; ordinary menu SAVE/fresh LOAD at both boundaries; 33 successor ticks exact", route, transferTick, transferTick+1)
		})
	}
}
