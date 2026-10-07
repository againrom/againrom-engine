// Command weaponspellcheck reproduces the round-3 review's own weapon-spell
// measurement method against a lawful install (developer tool,
// docs/1001-spell-effects/round3-weapon.md): it lifts one placed actor out
// of its own mission world, sets it beside a synthetic dummy target on open
// ground, and orders an attack for a fixed number of advances — once as
// shipped and once with the weapon's own spell cleared (the control) —
// reporting the target's own final health, how many effects stand attached
// to it, and how many weapon-borne casts the probe observed.
//
// Its output is a measurement over converted game data and must never be
// committed (golden rule 1, AGENTS.md); the asset root comes from -assets
// or AGAINROM_ASSETS and this file names no install path (golden rule 3).
//
// Usage:
//
//	weaponspellcheck [-assets <root>] [-mission 130] [-entity 55] [-ticks 240]
//	weaponspellcheck [-assets <root>] -scan 90,111,120,130,131,150,151
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"againrom/pkg/game"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func main() {
	assets := flag.String("assets", "", "game install root (defaults to AGAINROM_ASSETS)")
	mission := flag.Int("mission", 130, "campaign mission the actor is placed in")
	entity := flag.Uint64("entity", 55, "the placed actor's own entity id in that mission")
	ticks := flag.Int("ticks", 240, "advances to run the isolated probe for")
	scan := flag.String("scan", "", "comma-separated missions to sweep for weapon-spell carriers instead of probing one")
	flag.Parse()

	root := game.ResolveAssetRoot(*assets, os.Getenv("AGAINROM_ASSETS"))
	if root == "" {
		fmt.Fprintln(os.Stderr, "weaponspellcheck: no asset root: pass -assets or set AGAINROM_ASSETS")
		os.Exit(1)
	}
	var err error
	if *scan != "" {
		err = sweep(root, *scan)
	} else {
		err = run(root, *mission, sim.EntityID(*entity), *ticks)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "weaponspellcheck:", err)
		os.Exit(1)
	}
}

// sweep lists every placed actor carrying a weapon spell across the named
// missions, with the arm its own mana pool selects.
//
// IT EXISTS BECAUSE A SHIPPED ROW DOES NOT NAME ITS ARM. Whether a weapon-borne
// release is the caster's replacement or the fighter's rider is decided by the
// CARRIER's mana pool, not by the item, so counting item rows in a data file
// cannot say which draw path a shipped carrier reaches. This reads the placed
// actors themselves.
func sweep(root, missions string) error {
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	table, err := game.LoadTable(archives.Containers)
	if err != nil {
		return err
	}
	fmt.Println("mission entity  spell level  mana      arm")
	for _, field := range strings.Split(missions, ",") {
		m, err := strconv.Atoi(strings.TrimSpace(field))
		if err != nil {
			return fmt.Errorf("mission %q: %w", field, err)
		}
		started, err := game.StartMission(archives.Containers, m, table, mapload.DifficultyNormal, nil)
		if err != nil {
			return fmt.Errorf("mission %d: %w", m, err)
		}
		for _, e := range started.World.Entities() {
			if e.WeaponSpell == 0 {
				continue
			}
			arm := "rider (fighter)"
			if e.MaxMana > 0 {
				arm = "replacement (caster)"
			}
			fmt.Printf("%7d %6d %6d %5d  %4d/%-4d %s\n",
				m, e.ID, e.WeaponSpell, e.WeaponSpellLevel, e.Mana, e.MaxMana, arm)
		}
	}
	return nil
}

func run(root string, mission int, entityID sim.EntityID, ticks int) error {
	archives, err := game.OpenArchives(root)
	if err != nil {
		return err
	}
	table, err := game.LoadTable(archives.Containers)
	if err != nil {
		return err
	}
	started, err := game.StartMission(archives.Containers, mission, table, mapload.DifficultyNormal, nil)
	if err != nil {
		return err
	}
	spells := mapload.SpellRules(table)

	var subject sim.Entity
	found := false
	for _, e := range started.World.Entities() {
		if e.ID == entityID {
			subject, found = e, true
			break
		}
	}
	if !found {
		return fmt.Errorf("mission %d holds no entity %d", mission, entityID)
	}

	fmt.Printf("mission %d entity %d: class %d weapon-spell %d level %d mana %d/%d reach %d damage %d-%d always-hits %v hp %d/%d\n",
		mission, entityID, subject.Class, subject.WeaponSpell, subject.WeaponSpellLevel,
		subject.Mana, subject.MaxMana, subject.Reach, subject.DamageBase, subject.DamageBase+subject.DamageSpread,
		subject.AlwaysHits, subject.HP, subject.MaxHP)

	shipped, err := probe(subject, spells, ticks)
	if err != nil {
		return err
	}
	fmt.Printf("as shipped (staff spell %-4d)                  victim hp=%-6d attacker hp=%-6d attachedEffects=%-3d casts=%d\n",
		subject.WeaponSpell, shipped.hp, shipped.attackerHP, shipped.attached, shipped.casts)

	control := subject
	control.WeaponSpell, control.WeaponSpellLevel = 0, 0
	cleared, err := probe(control, spells, ticks)
	if err != nil {
		return err
	}
	fmt.Printf("control: same actor with WeaponSpell cleared   victim hp=%-6d attacker hp=%-6d attachedEffects=%-3d casts=%d\n",
		cleared.hp, cleared.attackerHP, cleared.attached, cleared.casts)
	return nil
}

type probeResult struct {
	hp         int32
	attackerHP int32
	attached   int
	casts      int
}

// probe places attacker (re-numbered 1, re-positioned onto open ground) far
// enough from a synthetic dummy target that the attack order's own approach
// (combat.go's approach/closedOn) walks it in and stops at ITS OWN natural
// engagement distance — melee Reach for a fighter — rather than this
// probe guessing one. That matters for a large-Reach siege unit (Boulder
// Thrower, Reach 20): an AREA weapon-borne row lands AT THE TARGET'S OWN
// CELL, so an attacker placed inside its own blast radius would take
// self-splash damage a unit actually fighting from its own stand-off
// distance never would, and the two numbers are reported separately so the
// difference is visible rather than silently folded into the target's own.
//
// Every transient order/cycle field the lifted actor carried in its own
// mission is cleared first: a leftover AttackTarget, cast wind-up or route
// naming an entity this 2-actor world does not hold is a shape the
// constructor refuses outright, and the probe measures the row's own
// arithmetic, not whatever the actor happened to be doing when it was
// lifted out.
func probe(attacker sim.Entity, spells []sim.SpellRule, ticks int) (probeResult, error) {
	attacker.ID, attacker.X, attacker.Y = 1, 5, 5
	attacker.HasTarget, attacker.TargetX, attacker.TargetY, attacker.Stall = false, 0, 0, 0
	attacker.HasAttackTarget, attacker.AttackTarget = false, 0
	attacker.AttackPhase, attacker.AttackCountdown = sim.AttackReady, 0
	attacker.CastWait = 0
	attacker.OffMap = false
	// SelfSlot is the player's own roster slot: the mission's own AI group
	// engagement (engage.go, armSwarm and its kin) re-decides every OTHER
	// owner's attack order on its own schedule and overrides a directly
	// issued one within a few ticks, which would make this probe measure
	// the AI's own targeting instead of the weapon-spell release. A lifted
	// actor is put under the player's own slot so the attack order this
	// probe issues is the only thing deciding its target.
	attacker.Owner = sim.SelfSlot
	if attacker.Reach == 0 {
		attacker.Reach = 1
	}
	// The starting separation is the actor's own Reach plus a walk-in
	// margin, so the approach logic — not a guessed adjacency — decides the
	// final engagement distance, exactly as it would for the same actor
	// fighting inside its own mission.
	gap := int32(attacker.Reach) + 3
	bounds := sim.Bounds{Width: gap + 20, Height: 20}
	target := sim.Entity{ID: 2, X: 5 + gap, Y: 5, Domain: attacker.Domain, HP: 100000, MaxHP: 100000, DyingTime: 100000}

	w, err := sim.NewStockedSpelledWorld(1, bounds, sim.ModeCanonical,
		sim.Terrain{}, []sim.Entity{attacker, target}, nil, sim.Relations{}, nil, nil, spells)
	if err != nil {
		return probeResult{}, fmt.Errorf("probe world: %w", err)
	}

	sim.Step(w, []sim.Command{sim.Attack(1, target.ID)})
	casts := 0
	for i := 0; i < ticks; i++ {
		casts += len(sim.StepObserved(w, nil))
	}

	var result probeResult
	for _, e := range w.Entities() {
		if e.ID == target.ID {
			result.hp = e.HP
		}
		if e.ID == attacker.ID {
			result.attackerHP = e.HP
		}
	}
	for _, e := range w.ActiveEffects() {
		if e.Target == target.ID {
			result.attached++
		}
	}
	result.casts = casts
	return result, nil
}
