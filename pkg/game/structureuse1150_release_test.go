package game

import (
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"againrom/internal/firstgame"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// Real installed structures, potion rows, pointer dispatch and cold SaveStore.
// Only the actor position/population are controlled; no original-runtime claim.
func TestReleaseStructureUse1150(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	placements := []usablePlacement{{90, 27, 16}, {90, 38, 15}, {91, 9, 28}, {101, 11, 29}}
	for _, p := range shippedUsablePlacements(t, releaseFront(t)) {
		if p.kind != 15 && p.kind != 16 && p.kind != 28 && p.kind != 29 {
			placements = append(placements, p)
		}
	}
	for _, tc := range placements {
		t.Run(fmt.Sprintf("mission%d-kind%d", tc.mission, tc.kind), func(t *testing.T) {
			f := releaseFront(t)
			f.SetDeterministicFrames(true)
			party := []mapload.PartyMember{{ID: "hero", PlayerCharacter: true, StartingHero: true, Mage: true,
				Profile: data.Profile{HealthColumn: true, ManaColumn: true}, Hero: data.Hero{Body: 60, Reaction: 60, Mind: 100, Spirit: 100},
				Saved: &mapload.Saved{Cell: mapload.Cell{X: 29, Y: 50}, HP: 10, MaxHP: 500, Mana: 20, MaxMana: 500}}}
			a := f.App("structure use1150")
			a.Layout(1024, 768)
			if err := a.OpenMission(f.MissionOpenerWith(tc.mission, party)); err != nil {
				t.Fatal(err)
			}
			if err := a.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			live := f.live
			target := live.world.Structures()[tc.id]
			if target.Kind != tc.kind || !target.Usable() || (tc.kind < 28 && target.UseAmount != 100) {
				t.Fatalf("installed metadata %+v", target)
			}
			class := f.Structures.Classes[tc.kind]
			t.Logf("structure=%+v indestructible=%v frames=%d grid=%d", target, class.Indestructible, len(class.Frames), class.GridCells())
			actor, ok := live.entity(live.mission.ids[0])
			if !ok {
				t.Fatal("missing hero")
			}
			actor.X, actor.Y = target.Col-5, target.Row
			actor.TargetX, actor.TargetY, actor.HasTarget = 0, 0, false
			actor.Transit, actor.TransitTotal = 0, 0
			actor.HealthRegenPeriod, actor.ManaRegenPeriod = 0, 0
			m := live.mission.state.Map
			refs := campaignScriptPartyRefs(m, f.Table, party, func(i int) sim.EntityID { return live.mission.ids[i] }, nil)
			refs.Structures = mapload.ScriptStructures(m)
			currentScript, _, err := mapload.CompileScript(m, refs)
			if err != nil {
				t.Fatal(err)
			}
			// The witness World is a mission World: it holds the installed cell
			// planes and its Buildings' records, as mission entry builds them.
			world, err := sim.NewStructuredWorld(1150, live.world.Bounds(), sim.ModeCanonical,
				mapload.Planes(m, f.Table),
				[]sim.Entity{actor}, currentScript, live.world.Relations(), nil, nil, mapload.SpellRules(f.Table), sim.GhostTemplate{}, live.world.Structures())
			if err != nil {
				t.Fatal(err)
			}
			if err := world.ImportOriginalCellTails(live.world.CellTails()); err != nil {
				t.Fatal(err)
			}
			witness := *live.mission.state
			witness.World = world
			if err = constructCurrentBuildings(&witness, f.Archives.Containers); err != nil {
				t.Fatal(err)
			}
			if err = world.DeclareItemWeights(live.world.ItemWeights()); err != nil {
				t.Fatal(err)
			}
			live.world = world
			live.commanded[actor.ID] = true
			for i := range live.fog.visible {
				live.fog.visible[i], live.fog.explored[i] = 1, 1
			}
			live.push()
			inspectionCentre(live, int(target.Col), int(target.Row))
			if err = a.HeadlessSelectEntity(uint32(actor.ID)); err != nil {
				t.Fatal(err)
			}
			live.push()
			x, y, err := live.view.InspectionPoint(ui.InspectionSubject{Kind: ui.InspectionStructure, ID: uint32(target.ID)})
			if err != nil {
				t.Fatal(err)
			}
			if err = a.HeadlessPointer("hover", x, y); err != nil {
				t.Fatal(err)
			}
			usableHoverProof(t, f, a, tc.kind, uint32(target.ID))
			for _, edge := range []string{"press", "release"} {
				if err = a.HeadlessPointer(edge, x, y); err != nil {
					t.Fatal(err)
				}
			}
			if len(live.pending) != 1 || live.pending[0].Kind != sim.KindUseStructure || live.pending[0].X != int32(target.ID) {
				t.Fatalf("pointer queued %+v", live.pending)
			}
			live.tick()
			if len(world.StructureUses()) != 1 {
				t.Fatal("click did not retain pending approach")
			}
			dir := t.TempDir()
			save, _, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return time.Unix(1150, 0) })
			name, err := save(true)
			if err != nil {
				t.Fatal(err)
			}
			cold := releaseFront(t)
			cold.SetDeterministicFrames(true)
			_, _, load := cold.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
			open, town, err := load(localOriginalSaveToken(name))
			if err != nil || town {
				t.Fatal("cold load", town, err)
			}
			ca := cold.App("cold structure use1150")
			ca.Layout(1024, 768)
			if err = ca.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if world.Hash() != cold.live.world.Hash() {
				x, _ := world.MarshalBinary()
				y, _ := cold.live.world.MarshalBinary()
				for at := 0; at < len(x) && at < len(y); at++ {
					if x[at] != y[at] {
						t.Logf("first world byte delta at%d live%x cold%x lengths%d/%d", at, x[max(0, at-8):min(len(x), at+16)], y[max(0, at-8):min(len(y), at+16)], len(x), len(y))
						break
					}
				}
				t.Fatalf("cold approach changed %x/%x", world.Hash(), cold.live.world.Hash())
			}
			for tick := 0; tick < 1000 && len(world.StructureUses()) > 0; tick++ {
				live.tick()
				cold.live.tick()
				if world.Hash() != cold.live.world.Hash() {
					t.Fatal("cold continuation changed", tick)
				}
			}
			if len(world.StructureUses()) != 0 {
				t.Fatal("never reached use target")
			}
			got, _ := live.entity(actor.ID)
			state := world.Structures()[tc.id]
			want := uint16(0)
			if target.Field42 == 0 {
				want = 1
			}
			if (tc.kind == 28 || tc.kind == 29) && state.Field42 != want {
				t.Fatal("lever did not toggle", state.Field42)
			}
			if tc.kind == 15 && (got.HP != actor.HP+100 || state.Field42 != 2) {
				t.Fatalf("healing=%d->%d charges=%d", actor.HP, got.HP, state.Field42)
			}
			if tc.kind == 16 && (got.Mana != actor.Mana+100 || state.Field42 != 2) {
				t.Fatalf("mana=%d->%d charges=%d", actor.Mana, got.Mana, state.Field42)
			}
			if tc.kind == 28 || tc.kind == 29 {
				entries, ruins := live.view.StructureRuinFrames(uint32(target.ID))
				if entries == 0 || state.Field42 == 0 && ruins == 0 {
					t.Fatalf("lever art not switched %d/%d", entries, ruins)
				}
			}
			t.Logf("completed at tick%d HP=%d mana=%d state=%d", world.Tick(), got.HP, got.Mana, state.Field42)
			name, err = save(true)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := (SaveStore{Dir: dir}).Read(name)
			if err != nil {
				t.Fatal(err)
			}
			file, err := sav.Open(raw)
			if err != nil {
				t.Fatal(err)
			}
			buildings, _, err := file.Buildings()
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for _, b := range buildings {
				if int32(b.Col) == state.Col && int32(b.Row) == state.Row && uint16(b.Kind) == tc.kind {
					found = true
					if b.Health != state.Field42 || b.MaxHealth != state.MaxHealth {
						t.Fatal("SAV producer lost current pair", b)
					}
					pair := file.Body[b.Off+60 : b.Off+64]
					if binary.LittleEndian.Uint16(pair) != state.Field42 || binary.LittleEndian.Uint16(pair[2:]) != state.MaxHealth {
						t.Fatalf("SAV body60..63=%x", pair)
					}
					t.Logf("SAV class%d structure%d body60..63=%x", tc.kind, tc.id, pair)
				}
			}
			if !found {
				t.Fatal("missing current SAV structure")
			}
			coldBuilding(t, f)
			open, town, err = load(localOriginalSaveToken(name))
			if err != nil || town {
				t.Fatal("completed use cold load", town, err)
			}
			if err = ca.OpenMission(open); err != nil {
				t.Fatal(err)
			}
			if world.Hash() != cold.live.world.Hash() {
				t.Fatalf("completed use hash %x != cold %x", world.Hash(), cold.live.world.Hash())
			}
			t.Logf("completed use cold SAV hash=%016x", world.Hash())
		})
	}
}
