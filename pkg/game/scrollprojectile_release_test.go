package game

import (
	"fmt"
	"image"
	"math/rand"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/sim"
)

// The scroll and the book must release the installed Fire Arrow picture from
// the same party mage to the same mission target. The observations are taken
// after each real world tick, when the map has already consumed CastEvent.
func TestReleaseScrollFireArrowDrawsBookProjectile(t *testing.T) {
	bookEvent, bookBolt := releaseFireArrowProjectile(t, false)
	scrollEvent, scrollBolt := releaseFireArrowProjectile(t, true)
	if bookBolt.picture != data.CastPicture(1) || scrollBolt.picture != bookBolt.picture ||
		scrollBolt.from != bookBolt.from || scrollBolt.to != bookBolt.to {
		t.Fatalf("book event=%+v bolt=%+v; scroll event=%+v bolt=%+v", bookEvent, bookBolt, scrollEvent, scrollBolt)
	}
}

func releaseFireArrowProjectile(t *testing.T, fromScroll bool) (sim.CastEvent, spellBolt) {
	t.Helper()
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	pool := shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1090)))
	var scroll sim.ItemInstance
	for _, candidate := range pool {
		if id, _, ok := sim.ScrollSpell(candidate.Instance()); ok && id == 1 {
			scroll = candidate.Instance()
			break
		}
	}
	if scroll.Code == 0 {
		t.Fatal("installed shelf did not construct a Fire Arrow scroll")
	}
	party := MissionPartyAs(true, f.StartWeapon.Value(), f.Bodies, f.Table)
	if len(party) != 1 || !party[0].Mage {
		t.Fatalf("mission party is not one mage: %+v", party)
	}
	party[0].CarriedItems = []sim.ItemInstance{scroll}
	party[0].Carried = []uint16{scroll.Code}
	fighter := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)[0]
	fighter.ID, fighter.Name, fighter.StartingHero = "fire-arrow-target", "Fire Arrow target", false
	party = append(party, fighter)
	f.Carried = party
	app := f.App("scroll-fire-arrow-projectile")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	casterID := mw.mission.ids[0]
	caster, ok := mw.entity(casterID)
	if !ok || caster.MaxMana == 0 {
		t.Fatalf("party mage missing in mission: %+v", caster)
	}
	rule, ok := mw.world.Spell(1)
	if !ok {
		t.Fatal("installed Fire Arrow rule missing")
	}
	if len(mw.mission.ids) != 2 {
		t.Fatalf("mission party IDs = %v, want two", mw.mission.ids)
	}
	target, ok := mw.entity(mw.mission.ids[1])
	if !ok || !target.Alive() {
		t.Fatalf("party target missing in mission: %+v", target)
	}
	if err := mw.world.HeadlessPlace(target.ID, caster.X+5, caster.Y); err != nil {
		t.Fatal(err)
	}
	target, _ = mw.entity(target.ID)
	if dx, dy := caster.X-target.X, caster.Y-target.Y; dx > int32(rule.MaxRange) || dx < -int32(rule.MaxRange) || dy > int32(rule.MaxRange) || dy < -int32(rule.MaxRange) || data.CastFlight(data.CastPicture(1), data.PictureCellUnits*int(max(dx, -dx, dy, -dy))) < 2 {
		t.Fatalf("target placed outside visible Fire Arrow flight: range=%d caster=(%d,%d) target=(%d,%d)", rule.MaxRange, caster.X, caster.Y, target.X, target.Y)
	}
	if fromScroll {
		items, ok := mw.world.CarriedItems(casterID)
		if !ok || len(items) == 0 {
			t.Fatalf("mage scroll pack: %+v", items)
		}
		if spell, _, ok := sim.ScrollSpell(items[0]); !ok || spell != 1 {
			t.Fatalf("first mage item is not the installed Fire Arrow scroll: %+v", items[0])
		}
		mw.useScroll(uint32(casterID), 0, fmt.Sprintf("%#v", items[0]), uint32(target.ID), 0, 0, false)
	} else {
		book := caster.Book
		if !book.HasInstances() {
			book.State = sim.BookPresent
		}
		book.Slots[0] = sim.BookSpell{Range: rule.MaxRange, ManaCost: uint16(rule.ManaCost)}
		if err := mw.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{
			ID: casterID, KnownSpells: caster.KnownSpells | 1<<1, Book: book,
		}}); err != nil {
			t.Fatal(err)
		}
		mw.castAt(uint32(casterID), uint32(target.ID), 1)
	}
	for tick := 1; tick <= 160; tick++ {
		for _, ev := range f.LiveAdvanceCasts(1) {
			if ev.Caster != casterID || ev.Spell != 1 || ev.Target != target.ID {
				continue
			}
			from, to := image.Pt(int(ev.FromX), int(ev.FromY)), image.Pt(int(ev.ToX), int(ev.ToY))
			for _, bolt := range mw.bolts {
				if bolt.picture == data.CastPicture(1) && bolt.from == from && bolt.to == to {
					return ev, bolt
				}
			}
			t.Fatalf("Fire Arrow released on tick %d but drew no source-to-target projectile: scroll=%t event=%+v bolts=%+v", tick, fromScroll, ev, mw.bolts)
		}
	}
	t.Fatalf("Fire Arrow did not release within 160 ticks: scroll=%t mage=%+v target=%+v", fromScroll, caster, target)
	return sim.CastEvent{}, spellBolt{}
}
