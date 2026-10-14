package game

import (
	"bytes"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func potionSheetStats1090(c ui.UnitCharacter) [4]int {
	return [4]int{c.Body, c.Reaction, c.Mind, c.Spirit}
}

func TestReleaseConsumable1090AppRestoredSheets(t *testing.T) {
	t.Run("four permanent attributes", func(t *testing.T) {
		f := releaseFront(t)
		f.SetDeterministicFrames(true)
		party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
		party[0].Carried, party[0].CarriedItems = nil, nil
		for _, code := range []uint16{0xe02, 0xe03, 0xe04, 0xe05} {
			party[0].CarriedItems = append(party[0].CarriedItems, mapload.ItemInstanceFromCode(code, f.Table))
			party[0].Carried = append(party[0].Carried, code)
		}
		f.Carried = party
		app := f.App("1090-restored-sheet")
		app.Layout(1024, 768)
		if err := app.OpenMission(f.MissionOpener(10)); err != nil {
			t.Fatal(err)
		}
		id := f.live.mission.ids[0]
		for n := 0; n < 32; n++ {
			if err := consumableStep1090(app); err != nil {
				t.Fatal(err)
			}
		}
		e, _ := f.live.entity(id)
		f.live.view.Camera().CenterOn(float64(e.X*32), float64(e.Y*32))
		if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(err)
		}
		want := potionSheetStats1090(f.live.chars[id])
		for slot := 0; slot < 4; slot++ {
			usePack1090(t, app, 0)
			for n := 0; n < 32; n++ {
				if err := consumableStep1090(app); err != nil {
					t.Fatal(err)
				}
				if e, _ := f.live.entity(id); e.PotionStats[slot] != 0 {
					break
				}
			}
			want[slot]++
			if got := potionSheetStats1090(f.live.chars[id]); got != want {
				t.Fatalf("use slot%d sheet=%v want%v", slot, got, want)
			}
		}
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		disk, err := EncodeSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		decoded, _, err := DecodeSave(disk)
		if err != nil {
			t.Fatal(err)
		}
		back := releaseFront(t)
		opener, town, err := back.Restore(decoded)
		if err != nil || town {
			t.Fatalf("restore town=%v: %v", town, err)
		}
		reopened := back.App("1090-first-restored-frame")
		reopened.Layout(1024, 768)
		if err := reopened.OpenMission(opener); err != nil {
			t.Fatal(err)
		}
		form, err := back.live.world.MarshalBinary()
		if err != nil || !bytes.Equal(form, snapshot.World) {
			t.Fatal("opening projection changed world bytes")
		}
		var control sim.World
		if err := control.UnmarshalBinary(form); err != nil {
			t.Fatal(err)
		}
		mapload.BindSourceDerive(&control)
		for tick := 0; tick <= 8; tick++ {
			found := false
			for _, drawn := range back.live.entityDraws() {
				if drawn.ID == uint32(id) {
					found = true
					if got := potionSheetStats1090(drawn.Char); got != want {
						t.Fatalf("restored tick%d sheet=%v want%v", tick, got, want)
					}
				}
			}
			if !found {
				t.Fatal("restored actor not projected")
			}
			sim.Step(&control, nil)
			sim.Step(back.live.world, nil)
			back.live.recomputeRaisedSkills()
			back.live.push()
			if control.Hash() != back.live.world.Hash() {
				t.Fatal("sheet refresh changed canonical continuation")
			}
		}
	})
}
