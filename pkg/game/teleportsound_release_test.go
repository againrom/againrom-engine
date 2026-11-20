package game

import (
	"image"
	"os"
	"strings"
	"testing"

	"againrom/pkg/audio"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// installClassRun is a class's attack run length and delay as the loader
// resolves them.
func installClassRun(c *data.UnitClass) (run int, delay int32) {
	for _, n := range c.AttackAnimTime {
		run += int(n)
	}
	return run, c.AttackDelay
}

// Every installed class the second game's Teleport caster arm can run: the
// sound slot exists, and a class with attack phases has a run its AttackDelay
// falls inside (R2-ENGINE-704).
func TestReleaseSecondTeleportCasterClassesSound(t *testing.T) {
	root := os.Getenv("AGAINROM_ASSETS")
	if root == "" {
		t.Skip("no AGAINROM_ASSETS: needs lawful install")
	}
	bank := OpenSounds(root)
	if bank == nil {
		t.Fatal("no sound bank")
	}
	_, ok := bank.entries[secondTeleportSound]
	if !ok {
		t.Errorf("sound slot %d is not in the sound registry", secondTeleportSound)
	}
	a, err := OpenArchives(root)
	if err != nil {
		t.Fatal(err)
	}
	classes, err := loadUnitClasses(a.Containers)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range classes.All() {
		run, delay := installClassRun(c)
		if c.AttackPhases > 0 && (run == 0 || int(delay) >= run) {
			t.Errorf("class %d has attack phases but its AttackDelay %d is outside its run of %d", c.ID, delay, run)
		}
	}
}

// requestCounter is a sound device that plays nothing and counts the requests
// of the Teleport cast animation hook.
type requestCounter struct{ requests int }

func (*requestCounter) Play(audio.Sample, audio.Placement) {}

func (c *requestCounter) RequestSample(_ audio.Sample, r audio.Request) audio.Voice {
	if r.Source == "book-cast-animation" {
		c.requests++
	}
	return nil
}

// A second-game hero of the installed mission uses an installed Teleport scroll
// at rest, and a second one while the first cast runs: the frontend sound route
// receives the Teleport sound once per cast.
func TestReleaseSecondHeroTeleportScrollSounds(t *testing.T) {
	for _, queued := range []bool{false, true} {
		f, _ := secondPersonsMission(t, 30)
		mw := f.live
		var hero sim.Entity
		for _, e := range mw.world.Entities() {
			if mw.missionPartyMember(e.ID) != nil && e.Owner == sim.SelfSlot {
				hero = e
				break
			}
		}
		if hero.ID == 0 {
			t.Fatal("no party hero in the mission")
		}
		var scroll sim.ItemInstance
		for row := 1; row < f.Table.MagicItems.Len() && row <= 255; row++ {
			if strings.ReplaceAll(f.Table.MagicItems.EntryName(row), "_", " ") != "Scroll Teleport" {
				continue
			}
			scroll = mapload.ItemInstanceFromCode(uint16(0x0e00|row), f.Table)
			spell, _, ok := sim.ScrollSpell(scroll)
			t.Logf("Scroll Teleport row %d: spell %d ok %v arm teleports %v effects %v", row, spell, ok, sim.SpellArmTeleports(mw.world.SpellArm(spell)), scroll.Effects)
		}
		if scroll.Code == 0 {
			t.Fatal("installed content has no Teleport scroll")
		}
		heard, requested := 0, 0
		player := &requestCounter{}
		mw.view.SetAudio(player, f.SoundBank)
		inspectionCentre(mw, int(hero.X), int(hero.Y))
		mw.semanticSound = func(source string, slot int, owner uint32, cell, fine image.Point) {
			if slot == secondTeleportSound {
				heard++
			}
			mw.view.PlayEffectAtFine(source, slot, owner, cell, fine)
		}
		casts := 1
		if queued {
			casts = 2
		}
		for i := 0; i < casts; i++ {
			if !mw.world.CheatAddItem(hero.ID, sim.StackItem(scroll, 1)) {
				t.Fatal("scroll not added")
			}
		}
		m := mw.mission.state.Map
		planes := mapload.Planes(m, f.Table)
		for i := 0; i < casts; i++ {
			x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, hero.X, hero.Y, int32(3+i))
			if !found {
				t.Fatal("no free destination")
			}
			items, _ := mw.world.CarriedItems(hero.ID)
			mw.useScroll(uint32(hero.ID), 0, itemCastKey(items[0]), 0, int(x), int(y), true)
			if i == 0 {
				mw.tick()
			}
		}
		for range 120 {
			mw.tick()
		}
		requested = player.requests
		if heard != casts || requested != casts {
			t.Errorf("queued=%v: Teleport sound heard %d times and requested %d times of the player, want %d", queued, heard, requested, casts)
		}
	}
}
