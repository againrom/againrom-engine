package game

import (
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// TestReleaseEveryReachableMercenaryCarriesItsOwnRotationSpeed is the census
// TestReleaseEveryReachableMercenaryWeaponReachesStatsAndDoll already runs
// (taverninspection_release_test.go), read for RotationSpeed instead of for
// weapons: every Humans template a live tavern can actually construct, once,
// over all four level bands the installed campaign reaches, PLUS both siege
// types (adversarial return round 3, second P2 instance: this test's own
// first cut excluded typ <= 2 with the comment "siege units, not Humans
// rows" -- true about the collection, and the reason the population never
// saw buildSiegeSquad's missing HiredRotationSpeed at all).
//
// A Humans template's expected value is data.NewHumanDef(template,
// ...).RotationSpeed, read by data.FindHumanByName off the row's own name --
// never through data.FindHumanByType, which is the many-to-one lookup round
// 2's own P2 was the defect in. A siege type's expected value is read off its
// own Units row by name (Catapult, Ballista), the collection buildSiegeSquad
// itself resolves against; a Units TypeID is absent from Humans entirely, so
// there is no by-TypeID comparison to make for these two.
func TestReleaseEveryReachableMercenaryCarriesItsOwnRotationSpeed(t *testing.T) {
	f := releaseFront(t)
	seen := make(map[string]bool)
	seenSiege := make(map[int]bool)
	var templates, siegeTemplates, differFromByTypeID int
	for _, mission := range f.Campaign.Value().Main {
		chapter := f.Campaign.Value().Chapters[mission]
		if len(chapter.Mercenaries) == 0 {
			continue
		}
		town := NewTown(f.Campaign.Value())
		for _, prior := range f.Campaign.Value().Main {
			if prior >= mission {
				break
			}
			if _, accepted := town.Won(prior); !accepted {
				t.Fatalf("could not advance census town through main mission %d", prior)
			}
		}
		if town.Chapter() != mission {
			continue // pre-town stretch, as in the weapon census above
		}
		f.Town = town
		s := f.TownScreen().(*townScreen)
		s.room = roomTavern
		for _, typ := range chapter.Mercenaries {
			if typ <= 2 {
				if seenSiege[typ] {
					continue
				}
				seenSiege[typ] = true
				siegeTemplates++

				name := "Catapult"
				if typ == 2 {
					name = "Ballista"
				}
				idx := data.NotFound
				for i := 1; i < f.Table.Units.Len(); i++ {
					if f.Table.Units.EntryName(i) == name {
						idx = i
						break
					}
				}
				if idx == data.NotFound {
					t.Fatalf("mission %d type %d: Units row %q is absent", mission, typ, name)
				}
				def, err := data.NewUnitDef(name, f.Table.Units.EntryParams(idx))
				if err != nil {
					t.Fatalf("mission %d type %d %q: %v", mission, typ, name, err)
				}
				ownRotation := def.RotationSpeed

				members, ok := s.buildMercenarySquad(typ, 1)
				if !ok || len(members) != 1 {
					t.Fatalf("mission %d type %d production template is unavailable", mission, typ)
				}
				member := members[0]
				if member.HiredRotationSpeed != ownRotation {
					t.Errorf("mission %d type %d %s: HiredRotationSpeed = %d, want %d (the Units row's own)",
						mission, typ, name, member.HiredRotationSpeed, ownRotation)
				}
				loadout := mapload.PartyLoadout(member, f.Table)
				if loadout.RotationSpeed != ownRotation {
					t.Errorf("mission %d type %d %s: PartyLoadout.RotationSpeed = %d, want %d",
						mission, typ, name, loadout.RotationSpeed, ownRotation)
				}
				continue
			}
			template := fmt.Sprintf("NPC%02d_%d", typ, mercenaryLevel(mission))
			if seen[template] {
				continue
			}
			seen[template] = true
			templates++

			row := data.FindHumanByName(f.Table.Humans, template)
			if row == data.NotFound {
				t.Fatalf("mission %d type %d template %s is absent", mission, typ, template)
			}
			def, err := data.NewHumanDef(template, f.Table.Humans.EntryParams(row))
			if err != nil {
				t.Fatalf("mission %d type %d template %s: %v", mission, typ, template, err)
			}
			ownRotation := def.RotationSpeed

			var byTypeRotation int32 = -1
			if i := data.FindHumanByType(f.Table.Humans, def.TypeID); i != data.NotFound {
				if d2, err := data.NewHumanDef(f.Table.Humans.EntryName(i), f.Table.Humans.EntryParams(i)); err == nil {
					byTypeRotation = d2.RotationSpeed
				}
			}
			if byTypeRotation != ownRotation {
				differFromByTypeID++
			}

			members, ok := s.buildMercenarySquad(typ, 1)
			if !ok || len(members) != 1 {
				t.Fatalf("mission %d type %d production template is unavailable", mission, typ)
			}
			member := members[0]
			if member.HiredRotationSpeed != ownRotation {
				t.Errorf("mission %d type %d template %s: HiredRotationSpeed = %d, want %d (the row's own); by-TypeID alone answers %d",
					mission, typ, template, member.HiredRotationSpeed, ownRotation, byTypeRotation)
			}
			loadout := mapload.PartyLoadout(member, f.Table)
			if loadout.RotationSpeed != ownRotation {
				t.Errorf("mission %d type %d template %s: PartyLoadout.RotationSpeed = %d, want %d",
					mission, typ, template, loadout.RotationSpeed, ownRotation)
			}
		}
	}
	if templates != 33 {
		t.Fatalf("reachable Humans template population = %d, want 33 (taverninspection_release_test.go's own count)", templates)
	}
	if siegeTemplates != 2 {
		t.Fatalf("reachable siege template population = %d, want 2 (Catapult, Ballista)", siegeTemplates)
	}
	t.Logf("checked %d Humans templates (%d differ from a by-TypeID lookup) and %d siege templates",
		templates, differFromByTypeID, siegeTemplates)
}

// TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed opens a real
// shipped mission and reads the live world's ghost template, the same value
// a Control Spirit cast copies onto a raised actor (pkg/sim/spell.go's
// raisedGhost). The expected value is read independently, straight off the
// installed Units table's own "Ghost" row (mapload.ghostTemplate's own
// constant, mirrored here since the census tool and this package both name
// it the same way and neither may import the other's unexported symbol).
func TestReleaseRaisedGhostCarriesTheInstalledUnitsRowRotationSpeed(t *testing.T) {
	f := releaseFront(t)
	idx := data.NotFound
	for i := 1; i < f.Table.Units.Len(); i++ {
		if f.Table.Units.EntryName(i) == "Ghost" {
			idx = i
			break
		}
	}
	if idx == data.NotFound {
		t.Fatal("installed Units table has no Ghost row")
	}
	def, err := data.NewUnitDef("Ghost", f.Table.Units.EntryParams(idx))
	if err != nil {
		t.Fatalf("Ghost row: %v", err)
	}
	if def.RotationSpeed == 0 {
		t.Fatal("installed Ghost row itself carries RotationSpeed 0; this test cannot discriminate on this install")
	}

	if _, _, _, _, _, _, _, _, _, _, err := f.MissionOpenerWith(10, f.NextParty())(); err != nil {
		t.Fatalf("mission 10 open: %v", err)
	}
	w, ok := f.LiveWorld()
	if !ok {
		t.Fatal("mission 10 opener left no live world")
	}
	if got := w.Ghost().RotationSpeed; got != def.RotationSpeed {
		t.Fatalf("live world Ghost().RotationSpeed = %d, want %d (installed Units row's own value)",
			got, def.RotationSpeed)
	}
}
