package mapload_test

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
)

func TestAPlacedPersonsSkillIsHisRowsRestoredLevel(t *testing.T) {
	e := loadHuman(t, humanTable(nil, nil, nil, nil), mapload.DifficultyNormal)

	if e.Skill[data.SkillBludgen] != srcHumanSkill {
		t.Errorf("entity carries Bludgen at level %d, want the row's own %d",
			e.Skill[data.SkillBludgen], srcHumanSkill)
	}
	if want := data.SkillXPFor(srcHumanSkill); e.SkillXP[data.SkillBludgen] != want {
		t.Errorf("Bludgen's experience is %d, want S(%d) = %d",
			e.SkillXP[data.SkillBludgen], srcHumanSkill, want)
	}
	for slot := range e.Skill {
		if int32(slot) == data.SkillBludgen {
			continue
		}
		if e.Skill[slot] != 0 || e.SkillXP[slot] != 0 {
			t.Errorf("slot %d carries level %d and experience %d, want 0 and 0 — only Bludgen was trained",
				slot, e.Skill[slot], e.SkillXP[slot])
		}
	}
}

func TestAPlacedPersonsSkillIsClampedBeforeItsExperienceIsComputed(t *testing.T) {
	const outOfRangeSkill = 150
	const bladeSlot = 10 + int(data.SkillBlade) // humandef.go's own case 10..15

	m := &alm.Map{Width: 10, Height: 10,
		Units: []alm.Unit{{X: 5 << 8, Y: 5 << 8, ClassID: 9}}}
	table := &mapload.Table{
		Humans: defCollection{
			{},
			{name: "h9", params: defRow(map[int]int32{slotHumanType: 9, bladeSlot: outOfRangeSkill})},
		},
	}
	w, err := mapload.FromALMWith(m, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatalf("FromALMWith: %v", err)
	}
	e := w.Entities()[0]

	if e.Skill[data.SkillBlade] != data.SkillCap {
		t.Fatalf("entity carries Blade at level %d, want the clamped %d", e.Skill[data.SkillBlade], data.SkillCap)
	}
	want := data.SkillXPFor(data.SkillCap)
	if wrong := data.SkillXPFor(outOfRangeSkill); want == wrong {
		t.Fatalf("S(%d) and S(%d) coincide — this fixture cannot tell the clamped answer from the raw one",
			data.SkillCap, outOfRangeSkill)
	}
	if e.SkillXP[data.SkillBlade] != want {
		t.Errorf("Blade's experience is %d, want S(%d) = %d — the stored, clamped level, not the row's raw %d",
			e.SkillXP[data.SkillBlade], data.SkillCap, want, outOfRangeSkill)
	}
	for slot := range e.Skill {
		if int32(slot) == data.SkillBlade {
			continue
		}
		if e.Skill[slot] != 0 || e.SkillXP[slot] != 0 {
			t.Errorf("slot %d carries level %d and experience %d, want 0 and 0 — only Blade was named",
				slot, e.Skill[slot], e.SkillXP[slot])
		}
	}
}

func TestAStartedPartyMembersSkillIsHisHerosLevel(t *testing.T) {
	m := startMap(t, 40, 40, mapload.Cell{X: 17, Y: 20})
	hero := data.Hero{Body: 25, Reaction: 25, Mind: 25, Spirit: 25}
	hero.Skill[data.SkillPike] = 42

	w, _ := mustStart(t, m, []mapload.PartyMember{{Class: 100, Hero: hero}})
	e := lastEntity(t, w)

	if e.Skill != hero.Skill {
		t.Errorf("member minted with levels %v, want his own %v", e.Skill, hero.Skill)
	}
	for slot, lvl := range hero.Skill {
		if want := data.SkillXPFor(lvl); e.SkillXP[slot] != want {
			t.Errorf("slot %d holds experience %d, want S(%d) = %d", slot, e.SkillXP[slot], lvl, want)
		}
	}
}
