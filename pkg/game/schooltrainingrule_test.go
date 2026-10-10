package game

import (
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
