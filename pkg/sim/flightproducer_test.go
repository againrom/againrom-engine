package sim

import (
	"fmt"
	"reflect"
	"testing"
)

func producerInputs(delay int32) FlightInputs {
	return FlightInputs{
		Pictures: map[int32]FlightPicture{
			1:  {Present: true, Phases: 1},
			10: {Present: true, Phases: 4, LifeDivisor: 200},
			34: {Present: true, Phases: 5, CellLife: 5},
			36: {Present: true, Phases: 13, LifeTicks: 13, CellLife: 13},
		},
		Classes: map[int32]FlightClass{14: {Projectile: 1, ShootDelay: delay, TileSize: 1}},
	}
}

func TestFlightProducerOrdinaryRangedOrders(t *testing.T) {
	for _, delay := range []int32{0, 2, 6} {
		for _, structure := range []bool{false, true} {
			t.Run(fmt.Sprintf("delay%d/structure%v", delay, structure), func(t *testing.T) {
				w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
					[]Entity{
						{ID: 1, Class: 14, X: 2, Y: 4, HP: 100, MaxHP: 100, Owner: 1, Reach: 6,
							AttackCharge: 4, AttackRelax: 12, DamageBase: 7, AlwaysHits: true},
						{ID: 2, Class: 1, X: 6, Y: 4, HP: 100, MaxHP: 100, Owner: 2},
					}, nil, nil)
				if err != nil {
					t.Fatal(err)
				}
				w.SetFlightInputs(producerInputs(delay))
				command, key := Attack(1, 2), int32(2)
				if structure {
					w.DeclareStructures([]Structure{{ID: 40, Col: 6, Row: 4, Width: 1, Height: 1, Field42: 100, MaxHealth: 100, Attach: 1}})
					command, key = AttackStructure(1, 40), 40
				}
				first, retired := 0, false
				for tick := 1; tick <= 15; tick++ {
					var commands []Command
					if tick == 1 {
						commands = []Command{command}
					}
					StepReported(w, commands)
					p, present := savedProjectileOf(w, 0)
					if present && first == 0 {
						first = tick
						if p.ID != 0 || p.Picture != 1 || p.ActionTarget != key || p.ActionX != 1664 || p.ActionY != 1152 || p.ActionPhase < 1 {
							t.Fatalf("ordinary order built %+v", p)
						}
					}
					if first != 0 && !present {
						retired = true
					}
					if w.SavedProjectiles().FreeIndex > 1 {
						t.Fatal("one physical run released a duplicate record")
					}
				}
				if want := max(int(delay), 1); first != want || !retired {
					t.Fatalf("first record tick %d, want %d; retired=%v", first, want, retired)
				}
				if !structure {
					target, _ := w.Entity(2)
					if target.HP >= 100 {
						t.Fatal("the ordinary blow never damaged its target")
					}
				}
			})
		}
	}
}

func TestFlightProducerLateShotContinuesAcrossWorldDecode(t *testing.T) {
	w := shotWorld(t, Entity{ID: 1, Class: 14, X: 2, Y: 4, HP: 100, MaxHP: 100, Owner: 1,
		Reach: 6, AttackCharge: 4, AttackRelax: 12}, Entity{ID: 2, X: 6, Y: 4, HP: 100, MaxHP: 100, Owner: 2})
	w.SetFlightInputs(producerInputs(6))
	Step(w, []Command{Attack(1, 2)})
	for range 3 {
		Step(w, nil)
	}
	if w.SavedProjectiles().FreeIndex != 0 {
		t.Fatal("the delayed shot was built before its release tick")
	}
	back := shotWorld(t)
	back.SetFlightInputs(producerInputs(6))
	if err := back.UnmarshalBinary(mustMarshal(t, w)); err != nil {
		t.Fatal(err)
	}
	seen := false
	for tick := 0; tick < 12; tick++ {
		Step(w, nil)
		Step(back, nil)
		if !reflect.DeepEqual(w.SavedProjectiles(), back.SavedProjectiles()) || w.Hash() != back.Hash() {
			t.Fatalf("tick %d after decode changed delayed-shot continuation", tick+1)
		}
		seen = seen || len(w.SavedProjectiles().Items) != 0
	}
	if !seen || w.SavedProjectiles().FreeIndex != 1 {
		t.Fatal("the decoded physical run never released exactly one delayed shot")
	}
}

func TestFlightProducerCellEntryClientArm(t *testing.T) {
	for _, spell := range []uint16{13, 14} {
		t.Run(fmt.Sprint(spell), func(t *testing.T) {
			w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
				[]Entity{{ID: 7, X: 2, Y: 3, HP: 1000, MaxHP: 1000, Domain: DomainGround, TokenSize: 1}}, nil,
				[]SpellRule{{ID: spell, School: 3, DamageMin: 20, DamageMax: 20, Damaging: true, TargetsUnit: true}})
			if err != nil {
				t.Fatal(err)
			}
			if err := w.DeclareCellTails([]CellTail{{X: 3, Y: 3, Bytes: [6]byte{byte(spell), 1, 9, 10, 11, 12}}}); err != nil {
				t.Fatal(err)
			}
			w.SetFlightInputs(producerInputs(2))
			Step(w, []Command{MoveTo(7, CellPoint{X: 3, Y: 3})})
			report := StepReported(w, nil)
			if len(report.ScriptCasts) != 1 {
				t.Fatalf("ordinary entry produced %d script casts", len(report.ScriptCasts))
			}
			items := w.SavedProjectiles().Items
			if len(items) != 1 {
				t.Fatalf("ordinary cell-entry cast produced %d records", len(items))
			}
			p := items[0]
			segments := int32(4)
			if spell == 14 {
				segments = 12
			}
			if p.Picture != int32(2*spell+8) || p.ActionPhase != 0 || p.ActionSegments != segments || p.Dir != 0 || p.ActionDir != 0 || p.X != 2432 || p.Y != 2688 {
				t.Fatalf("client-arm record %+v", p)
			}
			for range 20 {
				Step(w, nil)
			}
			if len(w.SavedProjectiles().Items) != 0 || w.SavedProjectiles().FreeIndex != 1 {
				t.Fatal("cell-entry client record was duplicated or never retired")
			}
		})
	}
}

func TestFlightProducerWeaponAndBookRelease(t *testing.T) {
	for _, family := range []string{"book", "staff", "weapon-rider"} {
		t.Run(family, func(t *testing.T) {
			caster := Entity{ID: 1, Class: 14, X: 2, Y: 4, HP: 100, MaxHP: 100, Owner: 1,
				Reach: 6, AttackCharge: 4, AttackRelax: 30, Mana: 100, MaxMana: 100,
				Mind: 30, KnownSpells: 1 << 1, AlwaysHits: true, DamageBase: 5, ScanRange: 12}
			command := Cast(1, 2, 1)
			if family != "book" {
				caster.WeaponSpell, caster.WeaponSpellLevel = 1, 30
				command = Attack(1, 2)
			}
			if family == "weapon-rider" {
				caster.Mana, caster.MaxMana = 0, 0
			}
			w, err := NewSpelledWorld(1, Bounds{Width: 16, Height: 16}, ModeCanonical, nil,
				[]Entity{caster, {ID: 2, X: 6, Y: 4, HP: 100, MaxHP: 100, Owner: 2}}, nil,
				[]SpellRule{{ID: 1, School: 1, ManaCost: 3, MaxRange: 8, TargetsUnit: true, Damaging: true, DamageMin: 4, DamageMax: 4}})
			if err != nil {
				t.Fatal(err)
			}
			w.SetFlightInputs(producerInputs(2))
			seen, events := false, 0
			for tick := 0; tick < 24; tick++ {
				var commands []Command
				if tick == 0 {
					commands = []Command{command}
				}
				report := StepReported(w, commands)
				for _, event := range report.Casts {
					events++
					if event.Weapon != (family != "book") || event.Rider != (family == "weapon-rider") {
						t.Fatalf("ordinary %s cast event %+v", family, event)
					}
				}
				for _, p := range w.SavedProjectiles().Items {
					want := int32(10)
					if family == "weapon-rider" {
						want = 1
					}
					if p.Picture != want {
						t.Fatalf("%s created picture %d, want %d", family, p.Picture, want)
					}
					seen = true
				}
			}
			victim, _ := w.Entity(2)
			if !seen || events != 1 || victim.HP >= 100 || w.SavedProjectiles().FreeIndex != 1 || len(w.SavedProjectiles().Items) != 0 {
				t.Fatalf("%s seen %v events %d HP %d records %+v", family, seen, events, victim.HP, w.SavedProjectiles())
			}
		})
	}
}

func TestFlightProducerMissingClassKeepsDefaultRun(t *testing.T) {
	actors := shotPair(4)
	actors[0].Class, actors[0].AttackCharge, actors[0].AttackRelax = 14, 4, 12
	plain, configured := shotWorld(t, actors...), shotWorld(t, actors...)
	configured.SetFlightInputs(FlightInputs{})
	for tick := 0; tick < 10; tick++ {
		var commands []Command
		if tick == 0 {
			commands = []Command{Attack(1, 2)}
		}
		Step(plain, commands)
		Step(configured, commands)
		actor, _ := configured.Entity(1)
		if actor.FlightRun != (FlightRun{}) || plain.Hash() != configured.Hash() {
			t.Fatalf("tick %d: absent projectile class introduced a canonical release clock %+v", tick+1, actor.FlightRun)
		}
	}
}
