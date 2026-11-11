package game

import (
	"image/color"
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/sim"
)

func spawnPortraitTable() *mapload.Table {
	rows := dbCollection{{}}
	for i, name := range []string{"Squirrel.2", "Turtle.2", "Foot_Animated.2", "F_Zombie.1", "Single"} {
		p := make([]int32, 56)
		for j := range p {
			p[j] = -1
		}
		p[29], p[30], p[55] = int32(74+i), 2, int32(5000+i)
		rows = append(rows, dbEntry{name: name, params: p})
	}
	edition := base.GameROM2.Edition()
	return &mapload.Table{Units: rows, UnitKeys: mapload.ServerUnitKeys, Edition: &edition}
}

func spawnPortraitActor(t *testing.T, table *mapload.Table, name string) sim.Entity {
	t.Helper()
	e, _, _, err := mapload.CheatActor(name, false, table, mapload.DifficultyNormal)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func spawnPortraitMission(t *testing.T, table *mapload.Table, actors []sim.Entity, script *sim.Script, rules []sim.SpellRule, state *SnapshotSAVDocument, manifest *SnapshotActorManifest) *mapWorld {
	t.Helper()
	w, err := sim.NewSpelledWorld(17, sim.Bounds{Width: 24, Height: 24}, sim.ModeCanonical, nil, actors, script, rules)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	m := &alm.Map{Width: 24, Height: 24, Tiles: make([]uint16, 24*24), Altitudes: make([]uint8, 24*24)}
	src := missionSource{
		"graphics/infowindow/creature.bmp":  synthBMP(2, 2, color.RGBA{R: 1, A: 255}),
		"graphics/infowindow/creature2.bmp": synthBMP(2, 2, color.RGBA{R: 2, A: 255}),
		"graphics/infowindow/single.bmp":    synthBMP(2, 2, color.RGBA{R: 7, A: 255}),
	}
	units := &terrain.UnitSet{Classes: make(map[int32]*terrain.UnitClass)}
	for typ := int32(74); typ <= 78; typ++ {
		art := tierArtClass(2)
		art.Portrait = "creature"
		if typ == 78 {
			art.Portrait = "single"
		}
		units.Classes[typ] = art
	}
	ms := &Mission{Map: m, World: w, savedDocument: state, ActorManifest: manifest}
	before := w.Hash()
	mw := openMission(ms, table, units, worldFixtureViewer(t, m), src, nil, nil)
	if w.Hash() != before {
		t.Fatal("opening the portrait changed simulation state")
	}
	return mw
}

func checkSpawnPortrait(t *testing.T, mw *mapWorld, id sim.EntityID, want uint8) {
	t.Helper()
	pic := mw.inspectionUnitPicture(uint32(id))
	if pic == nil || pic.RGBAAt(0, 0).R != want {
		e, _ := mw.world.Entity(id)
		t.Errorf("actor %d class %d type %d tier %d: picture %v, want red mark %d", id, e.Class, e.TypeID, mw.tiers[id], pic, want)
	}
}

func TestRuntimeCreaturePortraitRoutes(t *testing.T) {
	table := spawnPortraitTable()
	for _, route := range []string{"summon", "animate", "cheat", "script return", "reinforcement"} {
		t.Run(route, func(t *testing.T) {
			script, err := sim.NewROM2Script(nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			caster := sim.NewActor(sim.ActorDefinition{Class: 35, TypeID: 35, HP: 100, MaxHP: 100, Mind: 40, Mana: 500, MaxMana: 500,
				KnownSpells: 1<<25 | 1<<22, TokenSize: 1, ScanRange: 19}, sim.ActorPlacement{ID: 1, X: 12, Y: 12, Owner: sim.SelfSlot})
			caster.Skill[5] = 25
			actors := []sim.Entity{caster}
			var rules []sim.SpellRule
			switch route {
			case "summon":
				rules = []sim.SpellRule{{ID: 25, School: 5, ManaCost: 100}}
			case "animate":
				rules = []sim.SpellRule{{ID: 22, ManaCost: 2, TargetsUnit: true, MaxRange: 10}}
				corpse := sim.NewActor(sim.ActorDefinition{Class: 35, TypeID: 35, HP: -11, MaxHP: 40, TokenSize: 1}, sim.ActorPlacement{ID: 2, X: 14, Y: 14})
				corpse.Decay = 2
				actors = append(actors, corpse)
			case "script return", "reinforcement":
				e := spawnPortraitActor(t, table, "Squirrel.2")
				e.ID, e.X, e.Y, e.OffMap, e.Group = 2, 14, 14, true, 7
				actors = append(actors, e)
				op := sim.ScriptInstantReturnToMap
				if route == "reinforcement" {
					op = sim.ScriptInstantGroupOnMap
				}
				trigger := sim.ScriptTrigger{Once: true}
				for i := range trigger.Instants {
					trigger.Instants[i] = sim.ScriptNone
				}
				trigger.Instants[0] = 0
				script, err = sim.NewROM2Script(nil, []sim.ScriptInstant{{Op: op, Unit: 2, HasUnit: true, Group: 7, HasGroup: true}},
					[]sim.ScriptTrigger{trigger})
				if err != nil {
					t.Fatal(err)
				}
			}
			sim.AssignSecondGameArms(rules)
			var manifest *SnapshotActorManifest
			if route == "script return" || route == "reinforcement" {
				manifest = &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: 2, Name: "Squirrel.2"}}}
			}
			mw := spawnPortraitMission(t, table, actors, script, rules, nil, manifest)
			id := sim.EntityID(2)
			switch route {
			case "summon", "animate":
				var creatures []sim.SpellCreature
				for _, name := range []string{"Squirrel.2", "Turtle.2", "Foot_Animated.2", "F_Zombie.1"} {
					creatures = append(creatures, sim.SpellCreature{Name: name, Actor: spawnPortraitActor(t, table, name)})
				}
				mw.world.DeclareSpellCreatures(creatures)
				spell, target := sim.SpellID(25), sim.EntityID(1)
				if route == "animate" {
					spell, target, id = 22, 2, 3
				}
				mw.pending = append(mw.pending, sim.Cast(1, target, spell))
				for n := 0; n < 255; n++ {
					mw.tick()
					if _, ok := mw.world.Entity(id); ok {
						break
					}
				}
			case "cheat":
				id, err = mw.world.CheatSummon(spawnPortraitActor(t, table, "Squirrel.2"), nil, [sim.EquipSlots]sim.ItemInstance{}, 12, 12)
				if err != nil {
					t.Fatal(err)
				}
				mw.rememberCreatedActor(id, "Squirrel.2")
			default:
				for n := 0; n < 100; n++ {
					mw.tick()
					if e, ok := mw.world.Entity(id); ok && !e.OffMap {
						break
					}
				}
			}
			e, ok := mw.world.Entity(id)
			if !ok || e.OffMap || e.SourceBinding.Face != 2 {
				t.Fatalf("route did not produce the row's on-map creature: present=%v offMap=%v face=%d", ok, e.OffMap, e.SourceBinding.Face)
			}
			checkSpawnPortrait(t, mw, id, 2)
		})
	}
}

func TestOldRuntimeCreaturePortrait(t *testing.T) {
	for _, tc := range []struct {
		name  string
		row   uint16
		face  uint8
		first bool
		want  uint8
	}{
		{"old summon", 1, 0, false, 2},
		{"old animated corpse", 4, 0, false, 2},
		{"imported actor without manifest", 1, 0, false, 2},
		{"single picture", 5, 0, false, 7},
		{"explicit saved face", 1, 1, false, 1},
		{"first game summon", 1, 0, true, 1},
		{"first game saved face", 1, 1, true, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			table := spawnPortraitTable()
			if tc.first {
				table.UnitKeys = mapload.ClassUnitKeys
				table.Edition = nil
			}
			e := spawnPortraitActor(t, table, table.Units.EntryName(int(tc.row)))
			e.ID, e.X, e.Y, e.SourceBinding.Face = 9, 12, 12, tc.face
			if tc.name == "single picture" {
				e.Class = e.TypeID
			}
			state := &SnapshotSAVDocument{Document: &sav.DocumentData{Objects: []sav.DocumentRecordData{savedTierRecord(0, uint32(e.TypeID), uint32(tc.face))}},
				Actors: []SnapshotSAVActor{{EntityID: 9, ObjectIndex: 1}}}
			manifest := &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: 9, Name: table.Units.EntryName(int(tc.row)), Constructed: true}}}
			if tc.name == "imported actor without manifest" {
				e.SourceBinding.Class, e.SourceBinding.ArchiveIndex, manifest = 1, 1, nil
			}
			mw := spawnPortraitMission(t, table, []sim.Entity{e}, nil, nil, state, manifest)
			before := mw.world.Hash()
			checkSpawnPortrait(t, mw, e.ID, tc.want)
			if mw.world.Hash() != before || mw.world.Entities()[0].SourceBinding.Face != tc.face {
				t.Fatal("presentation repair changed the actor's saved state")
			}
		})
	}
}

func TestRuntimePortraitWithoutAMatchingRow(t *testing.T) {
	for _, control := range []string{"no table rows", "row zero", "row outside table", "wrong row type", "no source binding", "explicit face without row"} {
		t.Run(control, func(t *testing.T) {
			table := spawnPortraitTable()
			e := spawnPortraitActor(t, table, "Squirrel.2")
			e.ID, e.X, e.Y, e.SourceBinding.Face = 9, 12, 12, 0
			want := uint8(1)
			switch control {
			case "no table rows":
				table.Units = nil
			case "row zero":
				e.SourceBinding.TokenRow = 0
			case "row outside table":
				e.SourceBinding.TokenRow = 65535
			case "wrong row type":
				e.SourceBinding.TokenRow = 2
			case "no source binding":
				e.SourceBinding, e.Class = sim.SourceBinding{}, e.TypeID
			case "explicit face without row":
				table.Units, e.SourceBinding.Face, want = nil, 2, 2
			}
			manifest := &SnapshotActorManifest{Version: actorManifestVersion, Actors: []SnapshotActor{{ID: e.ID, Name: "Squirrel.2"}}}
			mw := spawnPortraitMission(t, table, []sim.Entity{e}, nil, nil, nil, manifest)
			if control != "explicit face without row" && mw.tiers[e.ID] != 0 {
				t.Errorf("an unavailable row supplied tier %d", mw.tiers[e.ID])
			}
			checkSpawnPortrait(t, mw, e.ID, want)
		})
	}
}

func TestFirstGameSummonedPortraitIsUnchanged(t *testing.T) {
	table := spawnPortraitTable()
	table.UnitKeys, table.Edition = mapload.ClassUnitKeys, nil
	mw := spawnPortraitMission(t, table, nil, nil, nil, nil, nil)
	id, err := mw.world.CheatSummon(spawnPortraitActor(t, table, "Squirrel.2"), nil, [sim.EquipSlots]sim.ItemInstance{}, 12, 12)
	if err != nil {
		t.Fatal(err)
	}
	mw.rememberCreatedActor(id, "Squirrel.2")
	checkSpawnPortrait(t, mw, id, 2)
	e := spawnPortraitActor(t, table, "Squirrel.2")
	e.ID, e.Class = 9, 0
	mw = spawnPortraitMission(t, table, []sim.Entity{e}, nil, nil, nil, nil)
	if pic := mw.inspectionUnitPicture(uint32(e.ID)); pic != nil {
		t.Fatal("the first game's class key was replaced with the source type")
	}
}
