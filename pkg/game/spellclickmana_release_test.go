package game

import (
	"encoding/json"
	"os"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

type spellClickManaWitness struct {
	Caster, Target uint32
	Mana, Landing  int32
	Ticks          int
	TargetHP       int32
	// Queued names the spell a click queued behind the saved wind-up; zero
	// when none was queued.
	Queued uint16
}

// spellClickFixture is the delivery source with a party mage that knows
// installed spells 1 and 13, standing five cells from a hostile unit.
func spellClickFixture(t *testing.T) (*FrontEnd, sim.Entity, sim.Entity) {
	t.Helper()
	f := spellWitnessSource(t)
	for _, id := range []int{1, 13} {
		if r := f.live.world.Spells()[id-1]; r.Delivery == 0 || r.ManaCost <= 0 {
			t.Fatalf("installed spell %d does not pay at admission: %+v", id, r)
		}
	}
	var mage, foe sim.Entity
	for _, e := range f.live.world.Entities() {
		if mage.ID == 0 && e.Owner == sim.SelfSlot && e.Alive() && e.MaxMana > 0 && e.Book.HasInstances() && e.SourceBinding.RuntimeID != 0 {
			mage = e
		}
		if foe.ID == 0 && e.Alive() && e.HP > 30 && e.SourceBinding.RuntimeID != 0 && f.live.world.Relations().Hostile(sim.SelfSlot, e.Owner) {
			foe = e
		}
	}
	if mage.ID == 0 || foe.ID == 0 {
		t.Fatal("fixture lacks a party mage or a hostile unit")
	}
	book := mage.Book
	for _, id := range []int{1, 13} {
		book.Slots[id-1] = sim.BookSpell{Range: 10, ManaCost: uint16(f.live.world.Spells()[id-1].ManaCost)}
	}
	if err := f.live.world.ImportOriginalActorSpellbooks([]sim.OriginalActorSpellbook{{ID: mage.ID, KnownSpells: mage.KnownSpells | 1<<1 | 1<<13, Book: book}}); err != nil {
		t.Fatal(err)
	}
	sprayRowWalk(t, f.live, mage.ID, 13, 40)
	// A cast ordered mid-step waits for the step's end; these tests order it at once.
	for range 40 {
		if e := releaseEntity(t, f.live, mage.ID); e.Transit == 0 {
			break
		}
		f.live.tick()
	}
	if err := f.live.world.HeadlessPlace(foe.ID, 18, 40); err != nil {
		t.Fatal(err)
	}
	mage = releaseEntity(t, f.live, mage.ID)
	if mage.Mana < 3*f.live.world.Spells()[12].ManaCost {
		t.Fatalf("party mage mana %d too low", mage.Mana)
	}
	return f, mage, releaseEntity(t, f.live, foe.ID)
}

func runningBook(f *FrontEnd, caster sim.EntityID) (sim.BookContinuation, bool) {
	for _, b := range f.live.world.Actions().Books {
		if b.Caster == caster {
			return b, true
		}
	}
	return sim.BookContinuation{}, false
}

// spellClickContinuation loads a SAV written during the wind-up, clicks the
// same cast again on each of the first clicks ticks, and requires the mana and
// the landing of uninterrupted play, which regenerates mana. A queued cast must
// start once the first has released.
func spellClickContinuation(t *testing.T, path string, want spellClickManaWitness, clicks int) {
	t.Helper()
	f := loadAreaContinuation(t, path)
	caster := worldEntityByRuntimeID(t, f, want.Caster)
	if caster.Mana != want.Mana {
		t.Fatalf("LOAD changed mana %d, want %d", caster.Mana, want.Mana)
	}
	if books, _, _ := f.live.world.NativeCastContinuations(); books != 1 {
		t.Fatalf("LOAD lost the wind-up: %d book casts", books)
	}
	if want.Queued != 0 && caster.PendingOrder.Spell != want.Queued {
		t.Fatalf("LOAD lost the queued cast: %+v", caster.PendingOrder)
	}
	target := worldEntityByRuntimeID(t, f, want.Target)
	queued := false
	for tick := 1; tick <= want.Ticks; tick++ {
		if tick <= clicks {
			f.live.attackOrCast(uint32(caster.ID), uint32(target.ID), 1, 0, 0, false)
		}
		f.live.tick()
		if b, ok := runningBook(f, caster.ID); ok && want.Queued != 0 && b.Spell == want.Queued {
			queued = true
		}
		if got := worldEntityByRuntimeID(t, f, want.Target).HP; tick < want.Ticks && got != target.HP {
			t.Fatalf("landing at tick %d, want %d", tick, want.Ticks)
		}
	}
	if got := worldEntityByRuntimeID(t, f, want.Target).HP; got != want.TargetHP {
		t.Fatalf("target hp %d, want %d", got, want.TargetHP)
	}
	if want.Queued != 0 {
		for tick := 0; tick < 64 && !queued; tick++ {
			f.live.tick()
			if b, ok := runningBook(f, caster.ID); ok && b.Spell == want.Queued {
				queued = true
			}
		}
		if queued {
			return
		}
		t.Fatal("the queued cast never started after LOAD")
	}
	if got := worldEntityByRuntimeID(t, f, want.Caster).Mana; got != want.Landing {
		t.Fatalf("%d repeated clicks: mana %d at the release, want %d as uninterrupted play", clicks, got, want.Landing)
	}
}

// TestReleaseSpellClicksSpendOnceAcrossSAVE clicks one installed book spell
// through the map's input queue and saves during its wind-up. Uninterrupted
// play gives the expected landing; a LOAD in this process with three more
// clicks, and a LOAD in a fresh process with one, must match it.
func TestReleaseSpellClicksSpendOnceAcrossSAVE(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	if path := os.Getenv("AGAINROM_SPELL_CLICK_INPUT"); path != "" {
		var want spellClickManaWitness
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		spellClickContinuation(t, path, want, 1)
		t.Logf("cold: mana %d after LOAD and %d at the release on tick %d", want.Mana, want.Landing, want.Ticks)
		return
	}
	f, mage, foe := spellClickFixture(t)
	cost := f.live.world.Spells()[0].ManaCost
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
	f.live.tick()
	paid := releaseEntity(t, f.live, mage.ID).Mana
	if books, _, _ := f.live.world.NativeCastContinuations(); books != 1 || mage.Mana-paid != cost {
		t.Fatalf("one click: %d book casts, spent %d, want one wind-up and %d", books, mage.Mana-paid, cost)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	want := spellClickManaWitness{Caster: mage.SourceBinding.RuntimeID, Target: foe.SourceBinding.RuntimeID, Mana: paid}
	for tick := 1; tick <= 256; tick++ {
		f.live.tick()
		if hp := releaseEntity(t, f.live, foe.ID).HP; hp != foe.HP {
			want.Ticks, want.TargetHP = tick, hp
			break
		}
	}
	want.Landing = releaseEntity(t, f.live, mage.ID).Mana
	if want.Ticks == 0 || want.Landing < paid {
		t.Fatalf("live release: landing tick %d, mana %d after %d at SAVE", want.Ticks, want.Landing, paid)
	}
	proof, _ := json.MarshalIndent(want, "", "  ")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, "spellclickmana", proof)
	spellClickContinuation(t, path, want, 3)
	runSpellWitnessChild(t, path, "AGAINROM_SPELL_CLICK_INPUT")
}

// TestReleaseSpellClickDuringAPaidWindUpRunsNext clicks a second installed
// spell while the first one's paid wind-up runs. The first cast keeps its
// clock and its single payment and lands; the second order then starts and
// pays once. A SAVE between the two clicks' effects keeps the queued order
// through a fresh-process LOAD.
func TestReleaseSpellClickDuringAPaidWindUpRunsNext(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	if path := os.Getenv("AGAINROM_SPELL_QUEUE_INPUT"); path != "" {
		var want spellClickManaWitness
		proof, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(proof, &want); err != nil {
			t.Fatal(err)
		}
		spellClickContinuation(t, path, want, 0)
		t.Logf("cold: the queued spell %d started after the release on tick %d", want.Queued, want.Ticks)
		return
	}
	f, mage, foe := spellClickFixture(t)
	first, second := f.live.world.Spells()[0].ManaCost, f.live.world.Spells()[12].ManaCost
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
	f.live.tick()
	started, _ := runningBook(f, mage.ID)
	paid := releaseEntity(t, f.live, mage.ID).Mana
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 13, 0, 0, false)
	f.live.tick()
	b, ok := runningBook(f, mage.ID)
	m := releaseEntity(t, f.live, mage.ID)
	if mage.Mana-paid != first || !ok || b.Spell != 1 || !b.Paid || b.Remaining+1 != started.Remaining || m.Mana < paid || m.PendingOrder.Spell != 13 {
		t.Fatalf("queued click: spent %d of %d, cast %+v (was %+v), mana %d after %d, pending %+v", mage.Mana-paid, first, b, started, m.Mana, paid, m.PendingOrder)
	}
	path := saveCorpseMission(t, f, t.TempDir())
	want := spellClickManaWitness{Caster: mage.SourceBinding.RuntimeID, Target: foe.SourceBinding.RuntimeID, Mana: m.Mana, Queued: 13}
	// One clock from the SAVE: the first cast lands, and the queued one starts
	// after the first one's release and pays its cost on that tick.
	startTick, spent, landed := 0, int32(0), false
	for tick := 1; tick <= 256 && !landed; tick++ {
		before := releaseEntity(t, f.live, mage.ID).Mana
		f.live.tick()
		if b, ok := runningBook(f, mage.ID); startTick == 0 && ok && b.Spell == 13 {
			startTick, spent = tick, before-releaseEntity(t, f.live, mage.ID).Mana
		}
		hp := releaseEntity(t, f.live, foe.ID).HP
		if want.Ticks == 0 && hp != foe.HP {
			want.Ticks, want.TargetHP = tick, hp
		}
		landed = startTick != 0 && want.Ticks != 0 && hp < want.TargetHP
	}
	if want.Ticks == 0 || startTick == 0 || !landed {
		t.Fatalf("first landing tick %d, queued start tick %d, second landed %v", want.Ticks, startTick, landed)
	}
	if spent < second-1 || spent > second {
		t.Fatalf("the queued cast spent %d, want its cost %d once", spent, second)
	}
	proof, _ := json.MarshalIndent(want, "", "  ")
	if err := os.WriteFile(path+".json", proof, 0600); err != nil {
		t.Fatal(err)
	}
	emitSpellWitness(t, path, "spellclickqueue", proof)
	spellClickContinuation(t, path, want, 0)
	runSpellWitnessChild(t, path, "AGAINROM_SPELL_QUEUE_INPUT")
}

// TestReleaseMoveClickDuringAPaidWindUpWalksAfterTheCast orders a walk while
// a paid wind-up runs: the caster stands until the cast releases, the cast
// lands, and the walk follows.
func TestReleaseMoveClickDuringAPaidWindUpWalksAfterTheCast(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	f, mage, foe := spellClickFixture(t)
	f.live.attackOrCast(uint32(mage.ID), uint32(foe.ID), 1, 0, 0, false)
	f.live.tick()
	paid := releaseEntity(t, f.live, mage.ID).Mana
	f.live.enqueue(uint32(mage.ID), 13, 44)
	released := false
	for tick := 0; tick < 256 && releaseEntity(t, f.live, foe.ID).HP == foe.HP; tick++ {
		f.live.tick()
		e := releaseEntity(t, f.live, mage.ID)
		if _, casting := runningBook(f, mage.ID); casting {
			if e.X != 13 || e.Y != 40 || e.Mana < paid {
				t.Fatalf("tick %d: the caster left its wind-up at (%d,%d) or spent again (mana %d after %d)", tick, e.X, e.Y, e.Mana, paid)
			}
		} else {
			released = true
		}
	}
	if !released || releaseEntity(t, f.live, foe.ID).HP == foe.HP {
		t.Fatal("the cast did not release and land")
	}
	for tick := 0; tick < 256; tick++ {
		if e := releaseEntity(t, f.live, mage.ID); e.X == 13 && e.Y == 44 {
			return
		}
		f.live.tick()
	}
	e := releaseEntity(t, f.live, mage.ID)
	t.Fatalf("the walk after the cast stopped at (%d,%d)", e.X, e.Y)
}
