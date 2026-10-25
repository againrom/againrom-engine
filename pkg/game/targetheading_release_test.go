package game

import (
	"crypto/sha256"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// TestReleaseAttackersTurnToTheOriginalsHeading places installed creatures on
// mission 20's open ground and orders one to attack through the production
// step: a ranged creature shooting at a two-cell body two east and one north,
// and a two-cell melee creature striking the hero one east and one north of
// its anchor. Each faces along footprint centres (AI-443, AI-444), where the
// anchor bearing named the diagonal 32. The facing held when the charge loads
// survives SAVE and a cold LOAD.
func TestReleaseAttackersTurnToTheOriginalsHeading(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	ranged, big, names := targetHeadingAttackers(t, releaseFront(t))
	for _, tc := range []struct {
		name             string
		attacker, victim sim.Entity
		dx, dy           int32
		want             uint8
	}{
		{"ranged", ranged, big, 2, -1, 64},
		{"melee", big, sim.Entity{}, 1, -1, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			arena := openLoadedCycleArena(t, 1)
			f, live := arena.front, arena.live
			hero := arena.get(arena.heroes[0])
			attacker := cleanResistanceWitnessEntity(tc.attacker, 0, arena.cx-4, arena.cy-2)
			attacker.PostX, attacker.PostY = attacker.X, attacker.Y
			attacker.Owner = 2
			targetHeadingBind(&attacker)
			hero.HP, hero.MaxHP = 1_000_000, 1_000_000
			victim := &hero
			entities := []sim.Entity{attacker, hero}
			if tc.victim.MaxHP > 0 {
				body := cleanResistanceWitnessEntity(tc.victim, 1, 0, 0)
				body.Owner, body.HP, body.MaxHP = sim.SelfSlot, 1_000_000, 1_000_000
				targetHeadingBind(&body)
				hero.X, hero.Y = arena.cx+5, arena.cy+5
				entities = []sim.Entity{attacker, body, hero}
				victim = &entities[1]
			}
			victim.X, victim.Y = attacker.X+tc.dx, attacker.Y+tc.dy
			victim.PostX, victim.PostY = victim.X, victim.Y
			if victim == &hero {
				entities[1] = hero
			} else {
				entities[2] = hero
			}
			rel := live.world.Relations()
			rel.Set(2, sim.SelfSlot, 1)
			rel.Set(sim.SelfSlot, 2, 2)
			m := live.mission.state.Map
			terrain := sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)}
			worn, _ := live.world.EquippedItems(hero.ID)
			for slot := range worn {
				worn[slot].ObjectID = 0
			}
			stocks := []sim.Stock{{ID: hero.ID, EquippedItems: worn}}
			w, err := sim.NewStructuredWorld(2020, live.world.Bounds(), sim.ModeCanonical, terrain, entities,
				nil, rel, nil, stocks, mapload.SpellRules(f.Table), sim.GhostTemplate{}, nil)
			if err != nil {
				t.Fatal(err)
			}
			mapload.BindSourceDerive(w)
			live.world = w
			candidate := *live.mission.state
			candidate.World, candidate.savedDocument = w, nil
			candidate.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
			for _, e := range entities[:len(entities)-1] {
				candidate.ActorManifest.Actors = append(candidate.ActorManifest.Actors,
					SnapshotActor{ID: e.ID, Name: names[e.TypeID], Constructed: true})
			}
			if live.mission.state.ActorManifest != nil {
				for _, row := range live.mission.state.ActorManifest.Actors {
					if row.ID == hero.ID {
						candidate.ActorManifest.Actors = append(candidate.ActorManifest.Actors, row)
					}
				}
			}
			live.mission.state = &candidate
			live.push()

			cmds := []sim.Command{sim.Attack(attacker.ID, victim.ID)}
			var got sim.Entity
			for n := 0; n < 256; n++ {
				sim.Step(w, cmds)
				cmds = nil
				got = entityByID(t, w, attacker.ID)
				if got.AttackPhase != sim.AttackReady {
					break
				}
			}
			snapshot, label, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := f.ExportCurrentSave(snapshot, label)
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("%s: type %d size %d reach %d at (%d,%d), victim type %d size %d at (%d,%d): tick %d facing %d desired %d phase %d; World hash %016x; SAV %d bytes sha256 %x",
				tc.name, got.TypeID, got.TokenSize, got.Reach, got.X, got.Y, victim.TypeID, victim.TokenSize, victim.X, victim.Y, w.Tick(), got.Facing, got.DesiredFacing,
				got.AttackPhase, w.Hash(), len(raw), sha256.Sum256(raw))
			if got.X != attacker.X || got.Y != attacker.Y || got.AttackPhase == sim.AttackReady {
				t.Fatalf("the attacker moved or loaded no charge: %+v", got)
			}
			if got.Facing != tc.want {
				t.Fatalf("the attacker faces %d, want %d", got.Facing, tc.want)
			}
			cold, _ := coldMission(t, raw)
			found := false
			for _, e := range cold.live.world.Entities() {
				if e.TypeID == got.TypeID && e.X == got.X && e.Y == got.Y && e.Owner != sim.SelfSlot {
					found = true
					if e.Facing != tc.want {
						t.Fatalf("cold LOAD facing %d, want %d", e.Facing, tc.want)
					}
				}
			}
			if !found {
				t.Fatal("cold LOAD holds no attacker at its cell")
			}
		})
	}
}

// targetHeadingAttackers is the first installed ground creature that shoots
// from one cell, and the first melee ground creature larger than one cell.
func targetHeadingAttackers(t *testing.T, f *FrontEnd) (ranged, big sim.Entity, names map[int32]string) {
	t.Helper()
	names = map[int32]string{}
	var haveRanged, haveBig bool
	for row := 1; f.Table.Units != nil && row < f.Table.Units.Len() && row <= 255; row++ {
		name := f.Table.Units.EntryName(row)
		if name == "" {
			continue
		}
		e, _, _, err := mapload.CheatActor(name, false, f.Table, mapload.DifficultyNormal)
		if err != nil || e.MaxHP <= 0 || e.Domain != sim.DomainGround || e.Speed <= 0 || e.DamageBase <= 0 {
			continue
		}
		if !haveRanged && e.Reach >= 2 && e.TokenSize <= 1 {
			ranged, haveRanged = e, true
			names[e.TypeID] = name
		}
		if !haveBig && e.Reach == 1 && e.TokenSize >= 2 {
			big, haveBig = e, true
			names[e.TypeID] = name
		}
	}
	if !haveRanged || !haveBig {
		t.Fatalf("installed creatures: ranged %v, larger melee %v", haveRanged, haveBig)
	}
	return ranged, big, names
}

// targetHeadingBind gives a constructed creature the generated source key a
// summon gives it.
func targetHeadingBind(e *sim.Entity) {
	e.ActorLoad.Source.HasOwner = true
	b := &e.SourceBinding
	b.ArchiveIndex, b.GroupIndex, b.GroupSelector, b.GroupOwnerKey = 0, 0, 0, 0
	b.GroupOwnerSlot, b.GroupOwnerResolved = 0, false
	b.Class, b.Identity = sim.GeneratedUnitBinding, 0x50000000+uint32(e.ID)*16
	b.RuntimeID = b.Identity >> 4
}
