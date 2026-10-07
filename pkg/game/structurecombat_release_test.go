package game

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Installed-data arena, not a claim that a campaign hero can acquire the
// Flame Thrower. Actor placement/population are fixtures; equipment folding,
// structure/terrain, input, HP/ruin presentation and SaveStore are production.
func TestReleaseStructurePhysicalAttack(t *testing.T) {
	for _, damaging := range []bool{false, true} {
		name := "native Short Bow zero damage"
		if damaging {
			name = "equipped installed Flame Thrower"
		}
		t.Run(name, func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := f.ChargenParty(ui.ChargenResult{Name: "Structure witness", Choices: []int{0, 0, 4}, Stats: []int{30, 30, 30, 30}})
			if damaging {
				p := f.Table.Weapons.EntryParams(25)
				if len(p) < 17 || p[5] != 11 || p[6] != 10 || p[7] != 80 || p[11] != 8 {
					t.Fatalf("weapon row changed: %v", p)
				}
				weapon, err := data.WeaponFromCode(data.ComposeItemCode(0, 1, 1, 25), f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
				if err != nil {
					t.Fatal(err)
				}
				party[0].Weapon, party[0].Worn[0] = &weapon, uint16(weapon.Code)
				party[0].WornItems[0] = mapload.ItemInstanceFromCode(uint16(weapon.Code), f.Table)
			}
			a := f.App("1085 structure combat")
			a.Layout(1024, 768)
			if err := a.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			live := f.live
			actor, ok := live.entity(live.mission.ids[0])
			if !ok {
				t.Fatal("no party actor")
			}
			if damaging {
				// Independent table calculation: material 0, shape 1, damage
				// multiplier slot 4; round base first, then subtract from maximum.
				scale := f.Table.Materials.EntryDoubles(0)[4] * f.Table.Shapes.EntryDoubles(1)[4]
				base := uint8(int(10*scale + 0.5))
				spread := uint8(int(80*scale - float64(base) + 0.5))
				if base != 2 || spread != 17 || actor.SecondaryDamage.Base != base || actor.SecondaryDamage.Spread != spread || actor.Reach != 8 {
					t.Fatalf("weapon did not fold: %+v", actor.SecondaryDamage)
				}
			} else if actor.SecondaryDamage != (sim.SecondaryDamage{}) {
				t.Fatalf("Short Bow invented damage: %+v", actor.SecondaryDamage)
			}
			sid, col, row := 4, int32(33), int32(49)
			if damaging {
				sid, col, row = 7, 53, 10
			}
			target := live.world.Structures()[sid]
			if int(target.ID) != sid || target.Col != col || target.Row != row || target.Field42 != 1000 || target.MaxHealth != 1000 || target.Width != 3 || target.Height != 3 {
				t.Fatalf("structure changed: %+v", target)
			}
			actor.X, actor.Y = target.Col-12, target.Row
			if !damaging {
				actor.X = target.Col - 6
			}
			actor.TargetX, actor.TargetY, actor.HasTarget = 0, 0, false
			actor.Transit, actor.TransitTotal = 0, 0
			worn, _ := live.world.EquippedItems(actor.ID)
			for slot := range worn {
				worn[slot].ObjectID = 0
			}
			m := live.mission.state.Map
			arena, err := sim.NewStructuredWorld(1085, live.world.Bounds(), sim.ModeCanonical,
				sim.Terrain{Block: mapload.PassabilityWith(m, f.Table), Cost: mapload.Cost(m), Height: mapload.Height(m)},
				[]sim.Entity{actor}, nil, live.world.Relations(), nil,
				[]sim.Stock{{ID: actor.ID, EquippedItems: worn, ItemInstances: party[0].CarriedItems}},
				mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
			if err != nil {
				t.Fatal(err)
			}
			codes := make([]uint16, 0)
			for _, item := range worn {
				if item.Code != 0 {
					codes = append(codes, item.Code)
				}
			}
			for _, item := range party[0].CarriedItems {
				codes = append(codes, item.Code)
			}
			mapload.DeclareCodeWeights(arena, f.Table, codes)
			mapload.BindSourceDerive(arena)
			live.world = arena
			candidate := *live.mission.state
			candidate.World, candidate.savedDocument = arena, nil
			candidate.ActorManifest = &SnapshotActorManifest{Version: actorManifestVersion}
			if live.mission.state.ActorManifest != nil {
				for _, row := range live.mission.state.ActorManifest.Actors {
					if row.ID == actor.ID {
						candidate.ActorManifest.Actors = append(candidate.ActorManifest.Actors, row)
					}
				}
			}
			live.mission.state = &candidate
			live.commanded[actor.ID] = true
			for i := range live.fog.visible {
				live.fog.visible[i], live.fog.explored[i] = 1, 1
			}
			live.push()
			inspectionCentre(live, int(actor.X), int(actor.Y))
			if err := a.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
				t.Fatal(err)
			}
			inspectionCentre(live, int(target.Col), int(target.Row))
			ref := ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(sid)}
			x, y, err := live.view.InspectionPoint(ref)
			if err != nil {
				t.Fatal(err)
			}
			inv, before := live.invSubject, live.world.Hash()
			if err := a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			if live.world.Hash() != before || len(live.pending) != 0 {
				t.Fatal("hover mutated world")
			}
			cardBefore, err := a.HeadlessMissionCard()
			if err != nil {
				t.Fatal(err)
			}
			beforePixels := append([]byte(nil), cardBefore.Pix...)
			if entries, ruined := live.view.StructureRuinFrames(uint32(sid)); entries == 0 || ruined != 0 {
				t.Fatalf("initial structure frames=%d/%d", ruined, entries)
			}
			if err := a.HeadlessKey("attack"); err != nil {
				t.Fatal(err)
			}
			x, y, err = live.view.InspectionPoint(ref)
			if err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("press", x, y); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessPointer("release", x, y); err != nil {
				t.Fatal(err)
			}
			if len(live.pending) != 1 || live.pending[0].Kind != sim.KindAttackStructure || live.pending[0].Entity != actor.ID || live.pending[0].X != int32(sid) {
				t.Fatalf("pointer queued %+v", live.pending)
			}
			if selected, ok := live.view.SelectedUnit(); !ok || selected != uint32(actor.ID) || !reflect.DeepEqual(inv, live.invSubject) {
				t.Fatal("attack changed selection/inventory")
			}
			live.tick()
			approach, strike := false, false
			hits, lastHP := 0, uint16(1000)
			for n := 0; n < 6000; n++ {
				e, _ := live.entity(actor.ID)
				if !approach && e.HasTarget {
					structureSaveContinuation(t, f, actor.ID, sim.EntityID(sid))
					approach = true
				}
				if !strike && e.AttackPhase == sim.AttackCharging && e.AttackCountdown > 0 {
					structureSaveContinuation(t, f, actor.ID, sim.EntityID(sid))
					strike = true
				}
				live.tick()
				hp := live.world.Structures()[sid].Field42
				if hp != lastHP {
					hits++
					lastHP = hp
				}
				if int16(hp) <= 0 || (!damaging && strike && n > 250) {
					break
				}
			}
			if !approach || !strike {
				t.Fatalf("approach=%v windup=%v", approach, strike)
			}
			for i := range live.fog.visible {
				live.fog.visible[i], live.fog.explored[i] = 1, 1
			}
			live.push()
			releaseHoverInspection(t, a, live, ref)
			panel, ok := live.view.InspectionPanel()
			if !ok || panel.HP != int(int16(lastHP)) || panel.MaxHP != 1000 {
				t.Fatalf("live HP panel=%+v", panel)
			}
			cardAfter, err := a.HeadlessMissionCard()
			if err != nil {
				t.Fatal(err)
			}
			if unchanged := bytes.Equal(beforePixels, cardAfter.Pix); unchanged == damaging {
				t.Fatalf("live HP card pixels unchanged=%v damaging=%v", unchanged, damaging)
			}
			if damaging {
				if int16(lastHP) > 0 || hits < 2 {
					t.Fatalf("not destroyed: hp%d hits%d", int16(lastHP), hits)
				}
				if e, _ := live.entity(actor.ID); e.HasAttackTarget {
					t.Fatal("attack survived ruin")
				}
				entries, ruined := live.view.StructureRuinFrames(uint32(sid))
				if entries == 0 || ruined != entries {
					t.Fatalf("ruin frames=%d/%d", ruined, entries)
				}
			} else if lastHP != 1000 || hits != 0 {
				t.Fatalf("Short Bow damaged structure: hp%d hits%d", lastHP, hits)
			}
			t.Logf("weapon=%s third=%d+U[0,%d] target%d hp1000->%d hits%d approach+charge saves continued exactly", data.ItemCode(worn[0].Code).Name(), actor.SecondaryDamage.Base, actor.SecondaryDamage.Spread, sid, int16(lastHP), hits)
		})
	}
}

func structureSaveContinuation(t *testing.T, f *FrontEnd, actor, target sim.EntityID) {
	t.Helper()
	dir := t.TempDir()
	save, _, _ := nativeContinuationSeams1170(t, f, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(1000, 0) })
	name, err := save(true)
	if err != nil {
		t.Fatal(err)
	}
	restored := releaseFront(t)
	_, _, load := agsSaveSeams(restored, SaveStore{Dir: dir}, OriginalStore{}, nil)
	open, town, err := load(name)
	if err != nil || town || open == nil {
		t.Fatalf("restore: town=%v err=%v", town, err)
	}
	if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
		t.Fatal(err)
	}
	if restored.live.world.Hash() != f.live.world.Hash() {
		t.Fatal("save/load changed attack world")
	}
	e, _ := restored.live.entity(actor)
	if !e.HasAttackTarget || e.AttackTargetKind != sim.AttackTargetStructure || e.AttackTarget != target {
		t.Fatalf("lost saved target: %+v", e)
	}
	for n := 0; n < 30; n++ {
		f.live.tick()
		restored.live.tick()
		if restored.live.world.Hash() != f.live.world.Hash() {
			t.Fatalf("save/load continuation diverged at tick%d", n)
		}
	}
}
