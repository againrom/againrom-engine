package game

import (
	"encoding/binary"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

type hasteSaveFields struct {
	Speed, Modifier int16
	Mover           uint8
	Magnitude       int16
	Remaining       uint32
	Children        int
	MaskBit         bool
}

func hasteSaveRead(t *testing.T, doc sav.DocumentData, want int) hasteSaveFields {
	t.Helper()
	var out hasteSaveFields
	found := 0
	for i := range doc.Objects {
		a := &doc.Objects[i]
		if a.Class != "Human" {
			continue
		}
		speed, _ := savedStructureValue(a, "Speed")
		reaction, _ := savedStructureValue(a, "Reaction")
		if int(reaction) != want {
			continue
		}
		found++
		modifier, err := savedMotionRaw(a, "UD4", 64)
		if err != nil {
			t.Fatal(err)
		}
		mover, err := savedMotionRaw(a, "U154", 180)
		if err != nil {
			t.Fatal(err)
		}
		mask, _ := savedStructureValue(a, "U144")
		out.Speed, out.Modifier, out.Mover = int16(speed), int16(binary.LittleEndian.Uint16(modifier[4:])), mover[10]
		out.MaskBit = mask&(1<<24) != 0
		refs, _ := savedObjectRefs(a, "Effects")
		out.Children = len(refs)
		for _, r := range refs {
			child := &doc.Objects[r-1]
			if kind, _ := savedStructureValue(child, "E3C"); kind != 17 {
				continue
			}
			operand, _ := savedStructureValue(child, "E40")
			out.Magnitude, out.Remaining = int16(operand), operand>>16
		}
	}
	if found != 1 {
		t.Fatalf("Human records with Reaction %d: %d, want one", want, found)
	}
	return out
}

func hasteParty() []mapload.PartyMember {
	hero := data.Hero{Body: 60, Reaction: 60, Mind: 40, Spirit: 100}
	for i := range hero.Skill {
		hero.Skill[i] = 45
	}
	return []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
		Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: hero, KnownSpells: 1 << 24,
		Saved: &mapload.Saved{Cell: mapload.Cell{X: 53, Y: 54}, HP: 1000, MaxHP: 1000, Mana: 1000, MaxMana: 1000}},
		{ID: "companion", Profile: data.Profile{HealthColumn: true}, Hero: data.Hero{Body: 26, Reaction: 15},
			Saved: &mapload.Saved{Cell: mapload.Cell{X: 54, Y: 55}, HP: 1000, MaxHP: 1000}}}
}

func hasteActive(f *FrontEnd) (sim.ActiveEffect, bool) {
	for _, e := range f.live.world.ActiveEffects() {
		if e.Spell == 24 {
			return e, true
		}
	}
	return sim.ActiveEffect{}, false
}

// SAVE under Haste writes the speed modifier word the original subtracts at
// expiry; cold LOAD and expiry return speed to its base.
func TestReleaseHasteSaveKeepsSpeedModifierThroughColdLoadAndExpiry(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	if err := f.App("haste save").OpenMission(f.MissionOpenerWith(101, hasteParty())); err != nil {
		t.Fatal(err)
	}
	caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
	base := releaseEntity(t, f.live, target).Speed
	victim := releaseEntity(t, f.live, target)
	f.live.attackOrCast(uint32(caster), uint32(target), 24, int(victim.X), int(victim.Y), false)
	var effect sim.ActiveEffect
	for tick := 0; ; tick++ {
		if tick > 256 {
			t.Fatal("Haste never attached", f.live.world.BookSpellRefusal(caster, target, 24))
		}
		f.live.tick()
		if e, ok := hasteActive(f); ok {
			effect = e
			break
		}
	}
	if effect.Target != target || effect.Kind != sim.EffectSpeed || effect.Magnitude <= 0 {
		t.Fatal("unexpected Haste attachment", effect)
	}
	mag := int16(effect.Magnitude)
	hasted := releaseEntity(t, f.live, target).Speed
	if hasted != base+int32(mag) {
		t.Fatalf("live speed %d, want base %d + %d", hasted, base, mag)
	}

	store := SaveStore{Dir: t.TempDir()}
	name, doc := deadPatrolSave(t, f, store)
	saved := hasteSaveRead(t, doc, 15)
	t.Logf("SAVE under Haste: speed %d modifier %d mover %d child magnitude %d remaining %d", saved.Speed, saved.Modifier, saved.Mover, saved.Magnitude, saved.Remaining)
	if saved.Speed != int16(hasted) || saved.Magnitude != mag || !saved.MaskBit || saved.Children != 1 {
		t.Fatalf("effect state not written: %+v", saved)
	}
	if saved.Modifier != mag {
		t.Fatalf("speed modifier word %d, want the Haste magnitude %d; the original's expiry would leave speed %d, not base %d", saved.Modifier, mag, saved.Speed-mag, base)
	}
	if after := saved.Speed - saved.Modifier; int32(after) != base {
		t.Fatalf("speed minus modifier = %d, want base %d", after, base)
	}

	cold := loadLocalLegacySave(t, store, name)
	loaded, ok := hasteActive(cold)
	if !ok || loaded.Target != effect.Target && releaseEntity(t, cold.live, loaded.Target).Reaction != 15 {
		t.Fatal("cold LOAD lost the Haste attachment", loaded)
	}
	if got := releaseEntity(t, cold.live, loaded.Target).Speed; got != hasted {
		t.Fatalf("cold LOAD speed %d, want %d (effect applied once)", got, hasted)
	}
	_, again := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
	if resaved := hasteSaveRead(t, again, 15); resaved.Modifier != mag || resaved.Speed != int16(hasted) || resaved.Children != 1 {
		t.Fatalf("second SAVE under Haste from the loaded world: %+v, want modifier %d speed %d", resaved, mag, hasted)
	}
	for tick := 0; ; tick++ {
		if tick > 3000 {
			t.Fatal("Haste never expired")
		}
		f.live.tick()
		cold.live.tick()
		_, a := hasteActive(f)
		_, b := hasteActive(cold)
		if a != b {
			t.Fatal("live and loaded Haste expired on different ticks", tick)
		}
		if !a {
			break
		}
	}
	for _, front := range []*FrontEnd{f, cold} {
		id := target
		if front == cold {
			id = loaded.Target
		}
		if got := releaseEntity(t, front.live, id).Speed; got != base {
			t.Fatalf("speed after expiry %d, want base %d", got, base)
		}
	}
	_, after := deadPatrolSave(t, cold, SaveStore{Dir: t.TempDir()})
	final := hasteSaveRead(t, after, 15)
	t.Logf("SAVE after expiry from the loaded world: speed %d modifier %d children %d", final.Speed, final.Modifier, final.Children)
	if int32(final.Speed) != base || final.Modifier != 0 || final.Children != 0 {
		t.Fatalf("post-expiry SAVE: %+v, want speed %d modifier 0", final, base)
	}
}
