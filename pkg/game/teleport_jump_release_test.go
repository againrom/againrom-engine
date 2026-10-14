package game

import (
	"crypto/sha256"
	"fmt"
	"image"
	"math/rand"
	"os"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func teleportJumpPushed(t *testing.T, mw *mapWorld, id sim.EntityID) ui.MapEntity {
	t.Helper()
	for _, entity := range mw.soundEntities {
		if entity.ID == uint32(id) {
			return entity
		}
	}
	t.Fatalf("push has no entity %d", id)
	return ui.MapEntity{}
}

func teleportJumpGeometry(t *testing.T, mw *mapWorld, entity ui.MapEntity, phase int) map[string][6]float64 {
	t.Helper()
	if entity.Art == nil || entity.Frame == nil {
		t.Fatal("installed witness actor has no sprite", entity.ID)
	}
	mw.view.SetEntities([]ui.MapEntity{entity})
	defer mw.view.SetEntities(mw.soundEntities)
	mw.view.SetPhase(phase, mw.clock.Period())
	draws, err := mw.view.HeadlessArtDraws()
	if err != nil {
		t.Fatal(err)
	}
	out := make(map[string][6]float64)
	for _, draw := range draws {
		if draw.Frame != entity.Frame || draw.Kind != "sprite" && draw.Kind != "shadow" {
			continue
		}
		if _, duplicate := out[draw.Kind]; duplicate {
			t.Fatal("isolated actor has duplicate draw submissions", entity.ID, draw.Kind)
		}
		var geometry [6]float64
		for row := range 2 {
			for column := range 3 {
				geometry[row*3+column] = draw.Geometry.Element(row, column)
			}
		}
		out[draw.Kind] = geometry
	}
	if _, found := out["sprite"]; !found {
		t.Fatal("production painter omitted installed actor", entity.ID, entity.Cell)
	}
	return out
}

func teleportJumpItem(t *testing.T, f *FrontEnd, scroll bool) sim.ItemInstance {
	t.Helper()
	if scroll {
		for _, item := range shopScrollPool(f.Table, 100000, rand.New(rand.NewSource(1090))) {
			if spell, _, ok := sim.ScrollSpell(item.Instance()); ok && spell == 26 {
				return item.Instance()
			}
		}
	} else {
		m := releaseMissionMap(t, f, 80)
		mission, err := StartMissionFrom(m, "scenario80.alm", 80, f.Table, mapload.DifficultyNormal, creatureWitnessParty())
		if err != nil {
			t.Fatal(err)
		}
		for index, unit := range m.Units {
			if unit.UnitID != 143 {
				continue
			}
			items, _ := mission.World.CarriedItems(sim.EntityID(index))
			for _, item := range items {
				if spell, ok := item.BookSpell(); ok && spell == 26 {
					return item
				}
			}
		}
	}
	t.Fatal("installed content has no Teleport item", scroll)
	return sim.ItemInstance{}
}

func teleportJumpFront(t *testing.T, mission int, scroll bool) (*FrontEnd, *ui.App) {
	t.Helper()
	return teleportJumpFrontOn(t, stoneStillFront(t), mission, scroll)
}

// teleportJumpFrontOn opens the Teleport witness mission on front f.
func teleportJumpFrontOn(t *testing.T, f *FrontEnd, mission int, scroll bool) (*FrontEnd, *ui.App) {
	t.Helper()
	item := teleportJumpItem(t, f, scroll)
	party := hasteParty()
	party[0].Hero = data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100}
	party[0].Hero.Skill[5] = 100
	party[0].KnownSpells = 0
	party[0].CarriedItems = []sim.ItemInstance{item}
	party[0].Carried = []uint16{item.Code}
	if mission != 101 {
		for index := range party {
			party[index].Saved = nil
		}
	}
	a := f.App("Teleport jump witness")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(mission, party)); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	mw.world.SetRawSessionHead([48]byte{16: 0x80, 17: 0x96, 18: 0x98, 32: 0x10, 33: 0x27})
	if !scroll {
		mw.enqueueEquip(0)
		if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindReadBook {
			t.Fatal("Teleport book use did not queue reading", mw.pending)
		}
		mw.tick()
		caster := releaseEntity(t, mw, mw.mission.ids[0])
		if caster.KnownSpells&(1<<26) == 0 {
			t.Fatal("installed book did not teach Teleport")
		}
	}
	return f, a
}

func teleportJumpLogHash(t *testing.T, mw *mapWorld, name string) {
	t.Helper()
	t.Logf("teleport-state case=%s tick=%d hash=%016x", name, mw.world.Tick(), mw.world.Hash())
}

func TestReleaseTeleportBookAndScrollDrawAtDestination(t *testing.T) {
	for _, name := range []string{"book", "scroll"} {
		t.Run(name, func(t *testing.T) {
			scroll := name == "scroll"
			f, a := teleportJumpFront(t, 101, scroll)
			mw := f.live
			casterID, walkerID := mw.mission.ids[0], mw.mission.ids[1]
			caster := releaseEntity(t, mw, casterID)
			from := image.Pt(int(caster.X), int(caster.Y))
			m := mw.mission.state.Map
			planes := mapload.Planes(m, f.Table)
			var destination image.Point
			for radius := int32(4); radius <= 6 && destination == (image.Point{}); radius++ {
				x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, caster.X, caster.Y, radius)
				if found {
					destination = image.Pt(int(x), int(y))
				}
			}
			if destination == (image.Point{}) {
				t.Fatal("installed mission has no free Teleport destination four to six cells away")
			}
			inspectionCentre(mw, from.X, from.Y)
			mw.view.Camera().SetZoom(1)
			before := teleportJumpPushed(t, mw, casterID)
			if before.Cell != from {
				t.Fatal("pre-cast pushed actor differs from simulation", before.Cell, from)
			}
			teleportJumpLogHash(t, mw, name)
			if scroll {
				items, _ := mw.world.CarriedItems(casterID)
				mw.useScroll(uint32(casterID), 0, fmt.Sprintf("%#v", items[0]), 0, destination.X, destination.Y, true)
				if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindUseScrollAt {
					t.Fatal("Teleport scroll did not queue its cell cast", mw.pending)
				}
			} else {
				mw.attackOrCast(uint32(casterID), 0, 26, destination.X, destination.Y, true)
				if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCastAt {
					t.Fatal("Teleport book did not queue its cell cast", mw.pending)
				}
			}
			hashOnly := os.Getenv("AGAINROM_TELEPORT_HASH_ONLY") != ""
			arrived := false
			for range 256 {
				old := teleportJumpPushed(t, mw, casterID)
				oldDistance := mw.walk[casterID].dist
				mw.tick()
				teleportJumpLogHash(t, mw, name)
				actor := releaseEntity(t, mw, casterID)
				if actor.X == caster.X && actor.Y == caster.Y {
					continue
				}
				arrived = true
				arrival := teleportJumpPushed(t, mw, casterID)
				if old.Cell != from || arrival.Cell != destination {
					t.Fatal("Teleport did not jump from the prior pushed cell to its destination", old.Cell, arrival.Cell, from, destination)
				}
				full := teleportJumpGeometry(t, mw, arrival, mw.clock.Period())
				for _, phase := range []int{0, mw.clock.Period() / 2} {
					got := teleportJumpGeometry(t, mw, arrival, phase)
					if !hashOnly {
						for kind, geometry := range full {
							if got[kind] != geometry {
								t.Errorf("Teleport %s phase%d geometry%v differs from destination geometry%v", kind, phase, got[kind], geometry)
							}
						}
					}
				}
				if !hashOnly && (arrival.Step != (image.Point{}) || arrival.FinePosition || mw.drawnMoving[casterID] || mw.walk[casterID].dist > oldDistance) {
					t.Errorf("Teleport arrival is walking: step%v fine%t moving%t distance%d->%d", arrival.Step, arrival.FinePosition, mw.drawnMoving[casterID], oldDistance, mw.walk[casterID].dist)
				}
				t.Logf("Teleport %s tick%d prior%v destination%v Step%v moving%t distance%d->%d sprite%v shadow%v", name, mw.world.Tick(), old.Cell, arrival.Cell, arrival.Step, mw.drawnMoving[casterID], oldDistance, mw.walk[casterID].dist, full["sprite"], full["shadow"])
				break
			}
			if !arrived {
				t.Fatal("Teleport did not relocate within 256 production ticks", releaseEntity(t, mw, casterID), mw.world.BookSpellCellRefusal(casterID, int32(destination.X), int32(destination.Y), 26))
			}
			walker := releaseEntity(t, mw, walkerID)
			walkX, walkY, found := creatureRingCell(mw.world, planes, m.Width, m.Height, walker.X, walker.Y, 1)
			if !found {
				t.Fatal("installed walking control has no free adjacent cell")
			}
			mw.enqueue(uint32(walkerID), int(walkX), int(walkY))
			walked := false
			for range 256 {
				mw.tick()
				teleportJumpLogHash(t, mw, name)
				e := releaseEntity(t, mw, walkerID)
				if e.X == walker.X && e.Y == walker.Y {
					continue
				}
				walked = true
				draw := teleportJumpPushed(t, mw, walkerID)
				inspectionCentre(mw, int(e.X), int(e.Y))
				mid := teleportJumpGeometry(t, mw, draw, mw.clock.Period()/2)
				full := teleportJumpGeometry(t, mw, draw, mw.clock.Period())
				if draw.Step == (image.Point{}) || max(absCell(int32(draw.Step.X)), absCell(int32(draw.Step.Y))) != 1 || mid["sprite"] == full["sprite"] {
					t.Fatal("one-cell walking lost its mid-tick shift", draw.Step, mid["sprite"], full["sprite"])
				}
				t.Logf("walking control tick%d Cell%v Step%v mid%v full%v", mw.world.Tick(), draw.Cell, draw.Step, mid["sprite"], full["sprite"])
				break
			}
			if !walked {
				t.Fatal("walking control did not cross a cell within 256 production ticks")
			}
			_, _, raw := menuSAVE(t, f, a, OriginalStore{})
			t.Logf("teleport-SAV case=%s tick=%d hash=%016x bytes=%d sha256=%x", name, mw.world.Tick(), mw.world.Hash(), len(raw), sha256.Sum256(raw))
		})
	}
}

func TestReleaseTeleportUnitTargetDrawsAtDestination(t *testing.T) {
	f, _ := teleportJumpFront(t, 20, false)
	mw := f.live
	casterID := mw.mission.ids[0]
	m := mw.mission.state.Map
	planes := mapload.Planes(m, f.Table)
	var target sim.Entity
	for _, entity := range mw.world.Entities() {
		if !entity.Alive() || entity.OffMap || entity.Domain != sim.DomainAir || !entity.OrdinaryTargetable() || planes.Block[int(entity.Y)*m.Width+int(entity.X)] != 0 {
			continue
		}
		x, y, found := creatureRingCell(mw.world, planes, m.Width, m.Height, entity.X, entity.Y, 6)
		if !found {
			continue
		}
		if err := mw.world.HeadlessPlace(casterID, x, y); err != nil {
			t.Fatal(err)
		}
		if refusal := mw.world.BookSpellRefusal(casterID, entity.ID, 26); refusal != "" {
			continue
		}
		target = entity
		break
	}
	if target.Domain != sim.DomainAir {
		t.Fatal("mission20 has no aerial Teleport target above passable ground with an admitted six-cell cast")
	}
	mw.fog.refresh(mw.world, sim.SelfSlot)
	mw.push()
	caster := releaseEntity(t, mw, casterID)
	inspectionCentre(mw, int(caster.X), int(caster.Y))
	mw.view.Camera().SetZoom(1)
	mw.attackOrCast(uint32(casterID), uint32(target.ID), 26, int(target.X), int(target.Y), false)
	if len(mw.pending) != 1 || mw.pending[0].Kind != sim.KindCast {
		t.Fatal("unit Teleport did not queue its actor cast", mw.pending)
	}
	hashOnly := os.Getenv("AGAINROM_TELEPORT_HASH_ONLY") != ""
	teleportJumpLogHash(t, mw, "book-unit")
	// The cast is reported at its admission and the caster moves when the
	// countdown reaches zero (MAGIC-TELEPORT-371, MAGIC-TELEPORT-370), so the
	// arrival is the first tick after the report that moves the caster.
	arrived := false
	var admitted []sim.CastEvent
	for range 256 {
		old := teleportJumpPushed(t, mw, casterID)
		oldDistance := mw.walk[casterID].dist
		mw.tickWithCastSink(func(casts []sim.CastEvent) {
			for _, event := range casts {
				if event.Caster == casterID && event.Spell == 26 && event.Target == target.ID {
					admitted = append(admitted, event)
				}
			}
		})
		teleportJumpLogHash(t, mw, "book-unit")
		var events []sim.CastEvent
		if moved := teleportJumpPushed(t, mw, casterID); len(admitted) == 1 && moved.Cell != old.Cell {
			events = admitted
		}
		for _, event := range events {
			arrived = true
			arrival := teleportJumpPushed(t, mw, casterID)
			destination := image.Pt(int(event.ToX), int(event.ToY))
			delta := arrival.Cell.Sub(old.Cell)
			if arrival.Cell != destination || max(absCell(int32(delta.X)), absCell(int32(delta.Y))) <= 1 {
				t.Fatal("unit Teleport did not apply a multi-cell relocation to the aerial target", old.Cell, arrival.Cell, event)
			}
			inspectionCentre(mw, arrival.Cell.X, arrival.Cell.Y)
			full := teleportJumpGeometry(t, mw, arrival, mw.clock.Period())
			for _, phase := range []int{0, mw.clock.Period() / 2} {
				got := teleportJumpGeometry(t, mw, arrival, phase)
				for kind, geometry := range full {
					if !hashOnly && got[kind] != geometry {
						t.Errorf("unit Teleport %s phase%d geometry%v differs from destination%v", kind, phase, got[kind], geometry)
					}
				}
			}
			if !hashOnly && (arrival.Step != (image.Point{}) || arrival.FinePosition || mw.drawnMoving[casterID] || mw.walk[casterID].dist > oldDistance) {
				t.Errorf("unit Teleport arrival is walking: step%v fine%t moving%t distance%d->%d", arrival.Step, arrival.FinePosition, mw.drawnMoving[casterID], oldDistance, mw.walk[casterID].dist)
			}
			t.Logf("unit Teleport tick%d target%d prior%v destination%v Step%v moving%t distance%d->%d sprite%v shadow%v", mw.world.Tick(), target.ID, old.Cell, arrival.Cell, arrival.Step, mw.drawnMoving[casterID], oldDistance, mw.walk[casterID].dist, full["sprite"], full["shadow"])
		}
		if arrived {
			break
		}
	}
	if !arrived {
		t.Fatal("unit Teleport did not release within 256 production ticks", mw.world.BookSpellRefusal(casterID, target.ID, 26))
	}
	t.Logf("teleport-final case=book-unit tick=%d hash=%016x", mw.world.Tick(), mw.world.Hash())
}

func TestReleaseCreatureTeleportUsesAdjacentCell(t *testing.T) {
	f := stoneStillFront(t)
	probe, _, creatures := creatureClassCases(t, f, creatureClassTele)
	a := f.App("creature Teleport adjacent witness")
	a.Layout(1024, 768)
	if err := a.OpenMission(f.MissionOpenerWith(creatureWitnessMission, creatureWitnessParty())); err != nil {
		t.Fatal(err)
	}
	mw := f.live
	source := mw.world
	heroID := mw.mission.ids[0]
	planes := mapload.Planes(probe, f.Table)
	tried := 0
	for _, creature := range creatures {
		x, y, found := creatureFreeCellNear(source, planes, probe.Width, probe.Height, creature.X, creature.Y)
		if !found || tried == 10 {
			continue
		}
		tried++
		mw.world = creatureWitnessWorld(t, source, probe, f.Table, heroID, x, y, creatureAlone(heroID, creature.ID))
		mw.fog.refresh(mw.world, sim.SelfSlot)
		mw.push()
		// The cast is reported at its admission; the creature moves when the
		// countdown reaches zero (MAGIC-TELEPORT-371, MAGIC-TELEPORT-370).
		casts := 0
		var admitted []sim.CastEvent
		for range 1500 {
			mw.tickWithCastSink(func(released []sim.CastEvent) {
				for _, event := range released {
					if event.Caster == creature.ID && event.Spell == 26 {
						admitted = append(admitted[:0], event)
						casts++
					}
				}
			})
			for _, event := range admitted {
				delta := image.Pt(int(event.ToX-event.FromX), int(event.ToY-event.FromY))
				if max(absCell(int32(delta.X)), absCell(int32(delta.Y))) != 1 {
					t.Fatal("installed creature Teleport is not aimed at an adjacent cell", event)
				}
				actor := releaseEntity(t, mw, creature.ID)
				if actor.X != event.ToX || actor.Y != event.ToY {
					continue
				}
				draw := teleportJumpPushed(t, mw, creature.ID)
				inspectionCentre(mw, int(actor.X), int(actor.Y))
				mid := teleportJumpGeometry(t, mw, draw, mw.clock.Period()/2)
				full := teleportJumpGeometry(t, mw, draw, mw.clock.Period())
				if draw.Step != delta || mid["sprite"] == full["sprite"] {
					t.Fatal("adjacent creature Teleport did not retain the one-cell displacement rule", draw.Step, delta, mid["sprite"], full["sprite"])
				}
				t.Logf("AI Teleport mission130 tick%d caster%d from(%d,%d) to%v Step%v moving%t mid%v full%v", mw.world.Tick(), creature.ID, event.FromX, event.FromY, draw.Cell, draw.Step, mw.drawnMoving[creature.ID], mid["sprite"], full["sprite"])
				return
			}
		}
		t.Logf("AI Teleport candidate%d casts%d after1500 hero%+v caster%+v", creature.ID, casts, releaseEntity(t, mw, heroID), releaseEntity(t, mw, creature.ID))
	}
	t.Fatalf("%d installed Teleport creatures did not relocate within 1500 production ticks each", tried)
}
