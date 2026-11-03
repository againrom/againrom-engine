package game

import (
	"testing"

	"againrom/pkg/base"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func TestAnimateCorpsePickUsesTheGamesArmTable(t *testing.T) {
	for _, tc := range []struct {
		game base.Game
		row  uint32
		want bool
	}{{base.GameROM1, 25, true}, {base.GameROM1, 22, false}, {base.GameROM2, 22, true}, {base.GameROM2, 25, false}} {
		if got := filesOf(tc.game).spellTakesBody(tc.row); got != tc.want {
			t.Fatalf("%s row %d: takes body %v", tc.game, tc.row, got)
		}
	}
}

func TestCurrentSaveKeepsAnimatedCorpseAcrossColdLoad(t *testing.T) {
	f, app, screen, _ := secondBagFrontOf(t, nil, secondMissionContent{})
	secondBagEnter(t, f, app, screen)
	snapshot, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	rules := []sim.SpellRule{{ID: 22, ManaCost: 2, TargetsUnit: true, MaxRange: 10}}
	sim.AssignSecondGameArms(rules)
	script, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	caster := sim.NewActor(sim.ActorDefinition{Class: 35, TypeID: 35, HP: 100, MaxHP: 100, Mana: 100, MaxMana: 100, Mind: 40, KnownSpells: 1 << 22, TokenSize: 1, ScanRange: 19}, sim.ActorPlacement{ID: 101, X: 14, Y: 14, Owner: sim.SelfSlot})
	corpse := sim.NewActor(sim.ActorDefinition{Class: 35, TypeID: 35, HP: -11, MaxHP: 122, Reaction: 46, Mind: 38, Spirit: 30, ToHit: 72, Defence: 80, TokenSize: 1}, sim.ActorPlacement{ID: 102, X: 16, Y: 16})
	corpse.Decay = 2
	actors := append(f.live.world.Entities(), caster, corpse)
	var stock []sim.Stock
	for _, e := range f.live.world.Entities() {
		pack, _ := f.live.world.CarriedStacks(e.ID)
		items, _ := f.live.world.CarriedItems(e.ID)
		worn, _ := f.live.world.EquippedItems(e.ID)
		for i := range pack {
			pack[i].ObjectID = 0
		}
		for i := range items {
			items[i].ObjectID = 0
		}
		for i := range worn {
			worn[i].ObjectID = 0
		}
		stock = append(stock, sim.Stock{ID: e.ID, OrderedStacks: pack, ItemInstances: items, EquippedItems: worn})
	}
	w, err := sim.NewStructuredWorld(53, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, script, sim.Relations{}, nil, stock, rules, sim.GhostTemplate{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(w)
	template := sim.NewActor(sim.ActorDefinition{Class: 35, TypeID: 35, HP: 24, MaxHP: 24, TokenSize: 1, Speed: 7}, sim.ActorPlacement{})
	w.DeclareSpellCreatures([]sim.SpellCreature{{Name: "F_Zombie.1", Actor: template}})
	commands := []sim.Command{sim.Cast(101, 102, 22)}
	for n := 0; n < 120; n++ {
		sim.Step(w, commands)
		commands = nil
		if _, ok := w.Entity(103); ok {
			break
		}
	}
	raised, ok := w.Entity(103)
	if !ok || raised.HP != 122 || raised.MaxHP != 122 || !raised.SuppressRewards || raised.SuppressCorpseLoot {
		t.Fatalf("Animate Dead: raised=%v hp=%d/%d mark=%v actors=%d admission=%q", ok, raised.HP, raised.MaxHP, raised.SuppressRewards, len(w.Entities()), w.BookSpellRefusal(101, 102, 22))
	}
	snapshot.SavedDocument = nil
	snapshot.Residue.VisualIdentities, snapshot.Residue.VisualNext = nil, 0
	snapshot.World, err = w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, "animated corpse")
	if err != nil {
		t.Fatal(err)
	}
	cold := secondMissionFront(t)
	cold.Table = f.Table
	opener, town, err := cold.RestoreOriginal(raw)
	if err == nil && !town {
		err = cold.App("animated corpse cold LOAD").OpenMission(opener)
	}
	if err != nil || town {
		t.Fatal("cold animated corpse LOAD", town, err)
	}
	got, ok := cold.live.world.Entity(103)
	if !ok || got.HP != raised.HP || got.MaxHP != raised.MaxHP || got.Reaction != raised.Reaction || got.Mind != raised.Mind || got.Spirit != raised.Spirit || got.ToHit != raised.ToHit || got.Defence != raised.Defence || got.Owner != raised.Owner || got.Group != raised.Group || got.X != raised.X || got.Y != raised.Y || !got.SuppressRewards || got.SuppressCorpseLoot != raised.SuppressCorpseLoot {
		t.Fatalf("cold SAV changed raised actor %d: health %d/%d, owner %d, cell %d,%d", got.ID, got.HP, got.MaxHP, got.Owner, got.X, got.Y)
	}
}
