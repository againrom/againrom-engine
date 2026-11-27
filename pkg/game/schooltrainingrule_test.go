package game

import (
	"slices"
	"strings"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/rules"
	"againrom/pkg/ui"
)

// TestSchoolTrainingRoutesShareOneRule trains the same slot from the same
// maintained base level by each of the school's three routes: a member with a
// source actor, a member restored from an original city save, and a plain
// member. Each charges rules.SchoolPrice of that base, displays the same price
// before the step, and leaves rules.SchoolTrainedXP of the next level in the
// slot.
func TestSchoolTrainingRoutesShareOneRule(t *testing.T) {
	const slot, base = 1, int32(9)
	price, err := rules.SchoolPrice(base)
	if err != nil {
		t.Fatal(err)
	}
	xp, err := rules.SchoolTrainedXP(base + 1)
	if err != nil {
		t.Fatal(err)
	}
	routes := []struct {
		name    string
		prepare func(*mapload.PartyMember)
		check   func(mapload.PartyMember) bool
	}{
		{"source actor", func(*mapload.PartyMember) {}, func(m mapload.PartyMember) bool { return mapload.HasSourceActor(m) }},
		{"original city Human", func(m *mapload.PartyMember) {
			carry := *m.Carry
			carry.LiveLoad = nil
			m.Carry = &carry
		}, func(m mapload.PartyMember) bool {
			_, ok := m.OriginalHumanState()
			return ok && !mapload.HasSourceActor(m)
		}},
		{"plain member", func(m *mapload.PartyMember) {
			carry := *m.Carry
			carry.LiveLoad = nil
			m.Carry = &carry
			m.OriginalHuman = nil
			m.Hero.Skill[slot] = base
		}, func(m mapload.PartyMember) bool {
			return m.OriginalHuman == nil && !mapload.HasSourceActor(m)
		}},
	}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			f := currentTrainingCity(t)
			for i := range f.Carried {
				if f.Carried[i].ID == "hero" {
					route.prepare(&f.Carried[i])
				}
			}
			before := trainingPartyMember(t, f, "hero")
			if !route.check(before) {
				t.Fatalf("the fixture does not take the %s route", route.name)
			}
			if got := memberSchoolPrice(before, slot); got != int(price) {
				t.Fatalf("displayed price %d, want rules.SchoolPrice(%d) = %d", got, base, price)
			}
			gold := f.Town.Gold()
			s := f.TownScreen().(*townScreen)
			s.room, s.schoolCell = roomSchool, slot-1
			for i := range f.Carried {
				if f.Carried[i].ID == "hero" {
					s.shopMember = i
				}
			}
			press := ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}
			if msg := s.TownSurfaceClick(press, false).Msg; !strings.HasPrefix(msg, "trained ") {
				t.Fatal(msg)
			}
			after := trainingPartyMember(t, f, "hero")
			if spent := gold - f.Town.Gold(); spent != int(price) {
				t.Fatalf("charged %d, want rules.SchoolPrice(%d) = %d", spent, base, price)
			}
			if got := after.Carry.SkillXP[slot]; got != xp {
				t.Fatalf("slot experience %d, want rules.SchoolTrainedXP(%d) = %d", got, base+1, xp)
			}
			if got := after.Hero.Skill[slot] - before.Hero.Skill[slot]; got != 1 {
				t.Fatalf("level moved by %d, want 1", got)
			}
		})
	}
}

func TestSchoolTrainingRoutesArmTrainEvent(t *testing.T) {
	const slot, base = 1, int32(9)
	price, err := rules.SchoolPrice(base)
	if err != nil {
		t.Fatal(err)
	}
	routes := []struct {
		name    string
		prepare func(*FrontEnd, *mapload.PartyMember)
		check   func(*FrontEnd, mapload.PartyMember) bool
	}{
		{"source actor", func(*FrontEnd, *mapload.PartyMember) {}, func(f *FrontEnd, m mapload.PartyMember) bool {
			return f.originalCity != nil && mapload.HasSourceActor(m)
		}},
		{"source actor without original city", func(f *FrontEnd, _ *mapload.PartyMember) {
			f.originalCity = nil
			f.Town.cityObjects = nil
		}, func(f *FrontEnd, m mapload.PartyMember) bool {
			return f.originalCity == nil && mapload.HasSourceActor(m)
		}},
		{"original city Human", func(_ *FrontEnd, m *mapload.PartyMember) {
			carry := *m.Carry
			carry.LiveLoad = nil
			m.Carry = &carry
		}, func(f *FrontEnd, m mapload.PartyMember) bool {
			_, ok := m.OriginalHumanState()
			return f.originalCity != nil && ok && !mapload.HasSourceActor(m)
		}},
		{"plain member", func(_ *FrontEnd, m *mapload.PartyMember) {
			carry := *m.Carry
			carry.LiveLoad = nil
			m.Carry = &carry
			m.OriginalHuman = nil
			m.Hero.Skill[slot] = base
		}, func(_ *FrontEnd, m mapload.PartyMember) bool {
			return m.OriginalHuman == nil && !mapload.HasSourceActor(m)
		}},
	}
	press := ui.TownSurfaceControl{Kind: ui.TownSurfaceControlButton, Index: 0}
	for _, route := range routes {
		t.Run(route.name, func(t *testing.T) {
			f := currentTrainingCity(t)
			rec := addRoomSounds(f)
			heroIdx := -1
			for i := range f.Carried {
				if f.Carried[i].ID == "hero" {
					route.prepare(f, &f.Carried[i])
					heroIdx = i
				}
			}
			before := trainingPartyMember(t, f, "hero")
			if heroIdx < 0 || !route.check(f, before) {
				t.Fatalf("the fixture does not take the %s route", route.name)
			}
			s := f.TownScreen().(*townScreen)
			s.room, s.shopMember, s.schoolCell = roomSchool, heroIdx, slot-1
			s.enterSchoolPage()

			f.Town.gold = int(price) - 1
			soundBase := len(rec.requests)
			if act := s.TownSurfaceClick(press, false); act.Msg != "" {
				t.Fatalf("short button click = %q, want silent refusal", act.Msg)
			}
			if got := schoolDiamondState(s); got != (diamondState{}) {
				t.Fatalf("short button click armed diamond: %+v", got)
			}
			if keys := rec.keys(soundBase); len(keys) != 0 {
				t.Fatalf("short button click requested sounds %q", keys)
			}
			if msg := s.trainHeroSkill(slot); !strings.HasPrefix(msg, "training costs ") {
				t.Fatalf("short direct training = %q, want cost refusal", msg)
			}
			if got := schoolDiamondState(s); got != (diamondState{}) {
				t.Fatalf("short direct training armed diamond: %+v", got)
			}
			if f.Town.Gold() != int(price)-1 || trainingPartyMember(t, f, "hero").Hero.Skill[slot] != before.Hero.Skill[slot] {
				t.Fatal("short training mutated purse or skill")
			}

			f.Town.gold = int(price)
			soundBase = len(rec.requests)
			if msg := s.TownSurfaceClick(press, false).Msg; !strings.HasPrefix(msg, "trained ") {
				t.Fatalf("affordable training = %q", msg)
			}
			if got := schoolDiamondState(s); got != (diamondState{step: 1}) {
				t.Fatalf("affordable training diamond = %+v, want armed step 1", got)
			}
			if keys := rec.keys(soundBase); !slices.Contains(keys, "school-training:town/school/teach.wav") {
				t.Fatalf("affordable training requested %q, want teach sound", keys)
			}
			if f.Town.Gold() != 0 || trainingPartyMember(t, f, "hero").Hero.Skill[slot] != before.Hero.Skill[slot]+1 {
				t.Fatal("affordable training did not charge price or advance skill")
			}
		})
	}
}
