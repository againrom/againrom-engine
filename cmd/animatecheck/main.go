package main

import (
	"flag"
	"fmt"
	"os"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func main() {
	root := flag.String("assets", "", "read-only second-game install root")
	flag.Parse()
	if err := run(*root); err != nil {
		fmt.Fprintln(os.Stderr, "animatecheck:", err)
		os.Exit(1)
	}
}

func run(root string) error {
	if root == "" {
		return fmt.Errorf("pass -assets")
	}
	a, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	d, err := game.LoadDefinitionsFor(a.Containers, a.Game())
	if err != nil {
		return err
	}
	var creatures []sim.SpellCreature
	for _, name := range sim.RaisedCreatureNames() {
		e, pack, worn, err := mapload.CheatActor(name, false, d.Table, mapload.DifficultyNormal)
		if err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
		creatures = append(creatures, sim.SpellCreature{Name: name, Actor: e, Pack: pack, Worn: worn})
		fmt.Printf("row %s type=%d body=%d reaction=%d mind=%d spirit=%d hp=%d speed=%d hit=%d defence=%d damage=%d..%d absorption=%d protection=%v resistance=%v\n", name, e.TypeID, body(e), e.Reaction, e.Mind, e.Spirit, e.MaxHP, e.Speed, e.ToHit, e.Defence, e.DamageBase, e.DamageBase+e.DamageSpread, e.Absorption, e.Protection, e.Resistance)
	}
	fmt.Println("synthetic corpse: body=65 reaction=47 mind=39 spirit=31 maxHP=122 hit=73 defence=81 cell=16,16; caster owner=7")
	for kind := sim.DecayStage(2); kind <= 4; kind++ {
		script, err := sim.NewROM2Script(nil, nil, nil)
		if err != nil {
			return err
		}
		caster := sim.NewActor(sim.ActorDefinition{HP: 100, MaxHP: 100, TypeID: 35, Mana: 10000, MaxMana: 10000, Mind: 40, KnownSpells: 1 << 22, TokenSize: 1, ScanRange: 19}, sim.ActorPlacement{ID: 1, X: 14, Y: 14, Owner: 7})
		corpse := sim.NewActor(sim.ActorDefinition{TypeID: 35, HP: -25, MaxHP: 122, NativeBasis: sim.NativeActorBasis{}.WithBody(65), Reaction: 47, Mind: 39, Spirit: 31, ToHit: 73, Defence: 81, TokenSize: 1}, sim.ActorPlacement{ID: 2, X: 16, Y: 16})
		corpse.Decay = kind
		corpse.HP = map[sim.DecayStage]int32{2: -11, 3: -21, 4: -41}[kind]
		w, err := sim.NewStructuredWorld(53, sim.Bounds{Width: 40, Height: 40}, sim.ModeCanonical, sim.Terrain{}, []sim.Entity{caster, corpse}, script, sim.Relations{}, nil, nil, mapload.SpellRules(d.Table), sim.GhostTemplate{}, nil)
		if err != nil {
			return err
		}
		mapload.BindSourceDerive(w)
		w.DeclareSpellCreatures(creatures)
		commands := []sim.Command{sim.Cast(1, 2, 22)}
		name := ""
		for n := 0; n < 120; n++ {
			report := sim.StepReported(w, commands)
			commands = nil
			for _, created := range report.CreatedActors {
				name = created.Name
			}
			if _, ok := w.Entity(3); ok {
				break
			}
		}
		e, ok := w.Entity(3)
		if !ok {
			return fmt.Errorf("kind %d: no raised actor, admission %q", kind, w.BookSpellRefusal(1, 2, 22))
		}
		fmt.Printf("kind=%d name=%s type=%d body=%d reaction=%d mind=%d spirit=%d hp=%d/%d hit=%d defence=%d speed=%d damage=%d..%d absorption=%d protection=%v resistance=%v cell=%d,%d owner=%d group=%d suppressed=%v\n", kind, name, e.TypeID, body(e), e.Reaction, e.Mind, e.Spirit, e.HP, e.MaxHP, e.ToHit, e.Defence, e.Speed, e.DamageBase, e.DamageBase+e.DamageSpread, e.Absorption, e.Protection, e.Resistance, e.X, e.Y, e.Owner, e.Group, e.SuppressRewards)
	}
	return nil
}

func body(e sim.Entity) uint16 {
	if e.ActorLoad.Source.Class != 0 {
		return e.ActorLoad.Source.Stats[0]
	}
	return e.NativeBasis.Body
}
