package game

import (
	"crypto/sha256"
	"fmt"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

const mission90WeaponResistanceSHA256 = "1af4b4ff040d48f4293020101d5042092328f2794254f438cfd8253aa6a006e1"

func TestReleaseMission90BladeResistanceChangesOnePhysicalBlow(t *testing.T) {
	f := releaseFront(t)
	addr, _ := MissionMap(90)
	raw, err := f.Archives.Containers.ReadFile(addr)
	if err != nil {
		t.Fatalf("read %s: %v", addr, err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != mission90WeaponResistanceSHA256 {
		t.Fatalf("%s SHA256 = %s, want the shipped EN/RU witness %s", addr, got, mission90WeaponResistanceSHA256)
	}

	mission, err := StartMission(f.Archives.Containers, 90, f.Table, openDifficulty, nil)
	if err != nil {
		t.Fatalf("StartMission(90): %v", err)
	}
	attacker := entityByMapUnitID(t, mission.World, 42)
	target := entityByMapUnitID(t, mission.World, 290)

	if attacker.ID != 1 || attacker.X != 106 || attacker.Y != 14 || attacker.Class != 5 ||
		attacker.HP != 261 || attacker.MaxHP != 261 || attacker.DamageBase != 21 ||
		attacker.DamageSpread != 9 || attacker.ToHit != 202 || attacker.Reach != 1 ||
		attacker.AlwaysHits || attacker.WeaponSpell != 0 || attacker.XPSlot != uint8(data.SkillBlade) {
		t.Fatalf("mission-90 u42 attacker = %+v; want F_KnightLeader2 at 106,14 with the loaded Blade combat row", attacker)
	}
	if target.ID != 151 || target.X != 102 || target.Y != 14 || target.Class != 26 ||
		target.HP != 250 || target.MaxHP != 250 || target.Defence != 150 || target.Absorption != 7 ||
		target.Resistance != ([5]uint8{40, 0, 20, 60, 80}) {
		t.Fatalf("mission-90 u290 target = %+v; want Catapult at 102,14 with the shipped resistance row", target)
	}

	worn, ok := mission.World.Equipped(attacker.ID)
	if !ok || data.ItemCode(worn[0]).Name() != "0101106" {
		t.Fatalf("mission-90 u42 equipment = %v, ok=%v; want slot-1 item 0101106", worn, ok)
	}
	weapon, err := data.WeaponFromCode(data.ItemCode(worn[0]), f.Table.Shapes, f.Table.Materials, f.Table.Weapons)
	if err != nil {
		t.Fatalf("resolve mission-90 u42 weapon: %v", err)
	}
	kind, ok := data.AttackTypeFromCode(data.ItemCode(worn[0]), f.Table.Weapons)
	if weapon.Name != "Uncommon Bronze Two Handed Sword" || !ok || kind != data.SkillBlade {
		t.Fatalf("mission-90 u42 weapon = %q kind (%d,%v), want Uncommon Bronze Two Handed Sword / Blade",
			weapon.Name, kind, ok)
	}

	attackerItems, attackerStocked := mission.World.EquippedItems(attacker.ID)
	targetItems, targetStocked := mission.World.EquippedItems(target.ID)
	if !attackerStocked || !targetStocked || targetItems[0].Empty() {
		t.Fatalf("mission-90 witness stocks: attacker=%v target=%v held=%v", attackerStocked, targetStocked, targetItems[0])
	}
	stocks := []sim.Stock{{ID: 1, EquippedItems: attackerItems}, {ID: 2, EquippedItems: targetItems}}
	resistedTick, resistedDamage, controlTick, controlDamage := oneLoadedPhysicalBlow(t, attacker, target, stocks, mapload.SpellRules(f.Table))
	if resistedTick != controlTick {
		t.Fatalf("first successful blow landed at tick %d with resistance and %d without; resistance must draw no RNG",
			resistedTick, controlTick)
	}
	if controlDamage < 14 || controlDamage > 23 {
		t.Fatalf("control blow dealt %d, want loaded 21..30 less absorption 7 = 14..23", controlDamage)
	}
	want := (controlDamage*60 + 75) / 100
	if resistedDamage != want || resistedDamage >= controlDamage {
		t.Fatalf("40%% Blade resistance changed damage %d to %d, want (%d*60+75)/100 = %d and lower",
			controlDamage, resistedDamage, controlDamage, want)
	}
	t.Logf("%s: u42 Blade blow at tick %d dealt %d without resistance and %d against u290's 40%% Blade resistance",
		addr, resistedTick, controlDamage, resistedDamage)
}

func entityByMapUnitID(t *testing.T, w *sim.World, id uint16) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.MapUnitID == id {
			return e
		}
	}
	t.Fatalf("mission world has no entity for map UnitID %d", id)
	return sim.Entity{}
}

func oneLoadedPhysicalBlow(t *testing.T, loadedAttacker, loadedTarget sim.Entity,
	stocks []sim.Stock, spells []sim.SpellRule) (resistedTick int, resistedDamage int32, controlTick int, controlDamage int32) {
	t.Helper()
	attacker := cleanResistanceWitnessEntity(loadedAttacker, 1, 5, 5)
	attacker.Owner = sim.SelfSlot
	target := cleanResistanceWitnessEntity(loadedTarget, 2, 6, 5)
	target.Owner = 0
	controlTarget := target
	controlTarget.Resistance = [5]uint8{}

	makeWorld := func(victim sim.Entity) *sim.World {
		t.Helper()
		w, err := sim.NewStockedSpelledWorld(1, sim.Bounds{Width: 20, Height: 20},
			sim.ModeCanonical, sim.Terrain{}, []sim.Entity{attacker, victim}, nil,
			sim.Relations{}, nil, stocks, spells)
		if err != nil {
			t.Fatalf("build isolated mission-90 blow world: %v", err)
		}
		return w
	}
	resisted, control := makeWorld(target), makeWorld(controlTarget)
	startHP := target.HP
	for tick := 1; tick <= 4096; tick++ {
		var cmds []sim.Command
		if tick == 1 {
			cmds = []sim.Command{{Kind: sim.KindAttack, Entity: attacker.ID, X: int32(target.ID)}}
		}
		sim.Step(resisted, cmds)
		sim.Step(control, cmds)
		resistedHP := entityByID(t, resisted, target.ID).HP
		controlHP := entityByID(t, control, target.ID).HP
		resistedChanged, controlChanged := resistedHP != startHP, controlHP != startHP
		if resistedChanged != controlChanged {
			t.Fatalf("at tick %d only one lockstep world changed target HP: resisted=%d control=%d start=%d",
				tick, resistedHP, controlHP, startHP)
		}
		if resistedChanged {
			return tick, startHP - resistedHP, tick, startHP - controlHP
		}
	}
	t.Fatal("mission-90 attacker landed no blow in 4096 ticks")
	return 0, 0, 0, 0
}

func cleanResistanceWitnessEntity(e sim.Entity, id sim.EntityID, x, y int32) sim.Entity {
	e.ID, e.X, e.Y = id, x, y
	e.HasTarget, e.TargetX, e.TargetY, e.Stall = false, 0, 0, 0
	e.Transit, e.TransitTotal, e.GroupSpeed = 0, 0, 0
	e.HasAttackTarget, e.AttackTarget = false, 0
	e.AttackPhase, e.AttackCountdown = sim.AttackReady, 0
	e.CastWait, e.OffMap = 0, false
	e.HasEscortTarget, e.EscortTarget, e.EscortRange = false, 0, 0
	return e
}

func entityByID(t *testing.T, w *sim.World, id sim.EntityID) sim.Entity {
	t.Helper()
	for _, e := range w.Entities() {
		if e.ID == id {
			return e
		}
	}
	t.Fatalf("world has no entity %d", id)
	return sim.Entity{}
}
