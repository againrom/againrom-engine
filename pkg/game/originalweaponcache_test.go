package game

import (
	"fmt"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestOriginalActorAdmissionReplacesTemplateWeaponSpell(t *testing.T) {
	for _, origin := range []sim.WeaponSpellSource{sim.WeaponSpellNone, sim.WeaponSpellItem, sim.WeaponSpellInnate, sim.WeaponSpellLegacy} {
		for _, saved := range []struct {
			name  string
			id    uint16
			power int32
		}{{"same", 2, 40}, {"changed", 1, 7}, {"removed", 0, 0}} {
			t.Run(fmt.Sprintf("source%d/%s", origin, saved.name), func(t *testing.T) {
				a, b := actorBookFixture(91), actorBookFixture(999)
				b.cell = 0x0706
				file, err := sav.Open(poolFixtureSave(a, b))
				if err != nil {
					t.Fatal(err)
				}
				graph, err := file.ActorGraph()
				if err != nil || len(graph.Actors) != 2 {
					t.Fatalf("source graph: %v %v", graph, err)
				}
				template := sim.Entity{ID: 1, X: 5, Y: 6, HP: 31, MaxHP: 31, WeaponSpellSource: origin}
				if origin != sim.WeaponSpellNone {
					template.WeaponSpell, template.WeaponSpellLevel = 2, 40
				}
				var equipment [sim.EquipSlots]sim.ItemInstance
				if origin == sim.WeaponSpellItem {
					equipment[0] = sim.ItemInstance{Code: 0xf11a, Kind: 2, Effects: []sim.ItemEffect{{Kind: 41, Operand: 40<<16 | 2}}}
				}
				unbound := sim.Entity{ID: 2, X: 8, Y: 8, HP: 31, MaxHP: 31, WeaponSpell: 3, WeaponSpellLevel: 9, WeaponSpellSource: sim.WeaponSpellInnate}
				world, err := sim.NewStockedWorld(1, sim.Bounds{Width: 32, Height: 32}, sim.ModeCanonical, sim.Terrain{},
					[]sim.Entity{template, unbound}, nil, sim.Relations{}, nil, []sim.Stock{{ID: 1, EquippedItems: equipment}})
				if err != nil {
					t.Fatal(err)
				}
				ms := &Mission{World: world, Start: mapload.Start{Roster: map[sim.EntityID]mapload.PartyMember{}}}
				registry := &originalActorRegistry{actors: []originalActorBinding{
					{Source: graph.Actors[0], ID: 1}, {Source: graph.Actors[1], ID: 3, New: true},
				}, groups: graph.Groups}
				if err := admitOriginalActorRegistry(ms, registry, actorRegistryTable()); err != nil {
					t.Fatal(err)
				}
				var worn [sim.EquipSlots]sim.ItemInstance
				if saved.id != 0 {
					worn[0] = sim.ItemInstance{Code: 0xf11a, Kind: 2, Effects: []sim.ItemEffect{{Kind: 41, Operand: uint32(saved.power)<<16 | uint32(saved.id)}}}
				}
				if err := world.ImportOriginalActorStock([]sim.OriginalActorStock{{ID: 1, Equipped: worn}, {ID: 3, Equipped: worn}}); err != nil {
					t.Fatal(err)
				}
				for _, e := range world.Entities() {
					if e.ID == 2 {
						if e.WeaponSpell != 3 || e.WeaponSpellLevel != 9 || e.WeaponSpellSource != sim.WeaponSpellInnate {
							t.Fatal("unbound native actor changed")
						}
						continue
					}
					wantSource := sim.WeaponSpellNone
					if saved.id != 0 {
						wantSource = sim.WeaponSpellItem
					}
					if e.WeaponSpell != saved.id || e.WeaponSpellLevel != saved.power || e.WeaponSpellSource != wantSource {
						t.Fatalf("actor %d saved weapon cache: %d/%d/%d", e.ID, e.WeaponSpell, e.WeaponSpellLevel, e.WeaponSpellSource)
					}
				}
				raw, err := world.MarshalBinary()
				if err != nil {
					t.Fatal(err)
				}
				fresh := new(sim.World)
				mapload.BindSourceDerive(fresh)
				if err := fresh.UnmarshalBinary(raw); err != nil {
					t.Fatal(err)
				}
				for range 64 {
					if world.Hash() != fresh.Hash() {
						t.Fatal("native continuation changed restored weapon")
					}
					sim.Step(world, nil)
					sim.Step(fresh, nil)
				}
			})
		}
	}
}
