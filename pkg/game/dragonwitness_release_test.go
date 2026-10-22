package game

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"againrom/pkg/sim"
)

// TestReleaseDragonWitnessFromLoad runs owner SAVs (AGAINROM_DRAGON_SAVES)
// without input and reports each dragon's orders, casts and damage dealt.
func TestReleaseDragonWitnessFromLoad(t *testing.T) {
	saves := os.Getenv("AGAINROM_DRAGON_SAVES")
	if saves == "" {
		t.Skip("set AGAINROM_DRAGON_SAVES to comma-separated owner SAV paths")
	}
	ticks := 3000
	if s := os.Getenv("AGAINROM_DRAGON_TICKS"); s != "" {
		n, err := strconv.Atoi(s)
		if err != nil || n <= 0 {
			t.Fatalf("AGAINROM_DRAGON_TICKS %q", s)
		}
		ticks = n
	}
	var out strings.Builder
	for _, path := range strings.Split(saves, ",") {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		f := releaseFront(t)
		openOriginalSAVApp(t, f, raw, filepath.Base(path))
		if dragonWitness(&out, f, filepath.Base(path), ticks) <= 0 {
			t.Errorf("%s: no dragon damaged the party in %d ticks", path, ticks)
		}
	}
	if dst := os.Getenv("AGAINROM_DRAGON_WITNESS_OUT"); dst != "" {
		if err := os.WriteFile(dst, []byte(out.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	t.Log("\n" + out.String())
}

type dragonState struct {
	x, y, tx, ty   int32
	hasTarget      bool
	victim         sim.EntityID
	hasVictim      bool
	pending        sim.EntityID
	hasPending     bool
	state          uint8
	phase          sim.AttackPhase
	idle           bool
	admitted       uint16
	hp, mana       int32
	transit        uint16
	stall          uint8
	refusals       string
	castBusy, seen bool
}

func dragonWitness(out *strings.Builder, f *FrontEnd, name string, ticks int) (total int32) {
	mw := f.live
	w := mw.world
	fmt.Fprintf(out, "== %s: tick %d, %d ticks\n", name, w.Tick(), ticks)
	primary, ok := mw.primaryPartyID()
	if !ok {
		fmt.Fprintf(out, "no primary party member\n")
		return 0
	}
	hero, _ := w.Entity(primary)
	var party, dragons []sim.EntityID
	for _, e := range w.Entities() {
		if !e.Alive() || e.OffMap {
			continue
		}
		if e.Owner == hero.Owner {
			party = append(party, e.ID)
			fmt.Fprintf(out, "party %d %q at %d,%d hp %d/%d domain %d\n", e.ID, mw.shownName(e.ID), e.X, e.Y, e.HP, e.MaxHP, e.Domain)
			continue
		}
		if strings.Contains(strings.ToLower(mw.shownName(e.ID)), "dragon") || strings.Contains(strings.ToLower(mw.shownName(e.ID)), "дракон") || e.TypeID == 71 {
			dragons = append(dragons, e.ID)
			fmt.Fprintf(out, "dragon %d %q type %d class %d owner %d group %d at %d,%d hp %d/%d mana %d/%d mind %d book %#x state %d skills %v domain %d reach %d scan %d state %d dmg %d+%d tohit %d charge %d relax %d withdraw %d wimpy %d\n",
				e.ID, mw.shownName(e.ID), e.TypeID, e.Class, e.Owner, e.Group, e.X, e.Y, e.HP, e.MaxHP, e.Mana, e.MaxMana, e.Mind,
				e.KnownSpells, e.Book.State, e.Skill, e.Domain, e.Reach, e.ScanRange, e.ActorState, e.DamageBase, e.DamageSpread, e.ToHit,
				e.AttackCharge, e.AttackRelax, e.Withdraw, e.Wimpy)
		}
	}
	if len(dragons) == 0 {
		fmt.Fprintf(out, "no dragon on the map\n")
		return 0
	}
	for _, rule := range w.Spells() {
		for _, id := range dragons {
			if e, _ := w.Entity(id); rule.ID <= 31 && e.KnownSpells&(1<<rule.ID) != 0 {
				fmt.Fprintf(out, "rule %d school %d mana %d range %d dmg %d..%d unit %v damaging %v defensive %v restorative %v area %v radius %d effect %d mode %d delivery %d arm %d\n",
					rule.ID, rule.School, rule.ManaCost, rule.MaxRange, rule.DamageMin, rule.DamageMax, rule.TargetsUnit, rule.Damaging,
					rule.Defensive, rule.Restorative, rule.Area, rule.Radius, rule.EffectKind, rule.EffectMode, rule.Delivery, rule.Arm)
				break
			}
		}
	}
	prev := make(map[sim.EntityID]dragonState)
	dealt := make(map[[2]sim.EntityID]int32)
	casts := make(map[sim.EntityID]map[uint16]int)
	for n := 0; n < ticks; n++ {
		var report sim.Report
		mw.tickStep(nil, n == ticks-1, func(r sim.Report) { report = r })
		tick := w.Tick()
		for _, c := range report.Casts {
			if !slices.Contains(dragons, c.Caster) {
				if slices.Contains(dragons, c.Target) {
					fmt.Fprintf(out, "t%d cast at dragon by %d spell %d weapon %v\n", tick, c.Caster, c.Spell, c.Weapon)
				}
				continue
			}
			if casts[c.Caster] == nil {
				casts[c.Caster] = make(map[uint16]int)
			}
			casts[c.Caster][c.Spell]++
			fmt.Fprintf(out, "t%d dragon %d CAST spell %d target %d atcell %v to %d,%d weapon %v\n", tick, c.Caster, c.Spell, c.Target, c.AtCell, c.ToX, c.ToY, c.Weapon)
		}
		for _, d := range report.Damages {
			if slices.Contains(dragons, d.Target) {
				v, _ := w.Entity(d.Target)
				fmt.Fprintf(out, "t%d DAMAGE dragon %d hp %d->%d credit %v:%d spell %d\n", tick, d.Target, d.BeforeHP, d.AfterHP, v.HasKillCredit, v.KillCreditSource, v.KillCreditSpell)
			}
			if !slices.Contains(party, d.Target) {
				continue
			}
			v, _ := w.Entity(d.Target)
			src := "unknown"
			if v.HasKillCredit {
				src = fmt.Sprint(v.KillCreditSource)
				if slices.Contains(dragons, v.KillCreditSource) {
					dealt[[2]sim.EntityID{v.KillCreditSource, d.Target}] += d.BeforeHP - d.AfterHP
				}
			}
			fmt.Fprintf(out, "t%d DAMAGE party %d hp %d->%d credit %s spell %d\n", tick, d.Target, d.BeforeHP, d.AfterHP, src, v.KillCreditSpell)
		}
		for _, id := range dragons {
			e, ok := w.Entity(id)
			if !ok {
				continue
			}
			if !e.Alive() {
				if p := prev[id]; p.seen && p.hp > 0 {
					fmt.Fprintf(out, "t%d dragon %d DIED\n", tick, id)
					p.hp = 0
					prev[id] = p
				}
				continue
			}
			s := dragonState{x: e.X, y: e.Y, tx: e.TargetX, ty: e.TargetY, hasTarget: e.HasTarget,
				victim: e.AttackTarget, hasVictim: e.HasAttackTarget, pending: e.PendingAttackTarget,
				hasPending: e.HasPendingAttackTarget, state: e.ActorState, phase: e.AttackPhase, idle: e.PursuitIdle,
				admitted: e.AdmittedBookSpell, hp: e.HP, mana: e.Mana, transit: e.Transit, stall: e.Stall, seen: true}
			if e.HasAttackTarget || e.HasPendingAttackTarget {
				victim := e.AttackTarget
				if !e.HasAttackTarget {
					victim = e.PendingAttackTarget
				}
				var parts []string
				for spell := uint32(1); spell <= 28; spell++ {
					if e.KnownSpells&(1<<spell) == 0 {
						continue
					}
					v, _ := w.Entity(victim)
					r := w.BookSpellRefusal(id, victim, spell)
					rc := w.BookSpellCellRefusal(id, v.X, v.Y, spell)
					parts = append(parts, fmt.Sprintf("%d:[%s|cell %s]", spell, r, rc))
				}
				s.refusals = strings.Join(parts, " ")
			}
			p := prev[id]
			if !p.seen || p.x != s.x || p.y != s.y || p.hasTarget != s.hasTarget || p.tx != s.tx || p.ty != s.ty ||
				p.victim != s.victim || p.hasVictim != s.hasVictim || p.pending != s.pending || p.hasPending != s.hasPending ||
				p.state != s.state || p.phase != s.phase || p.idle != s.idle || p.admitted != s.admitted || p.hp != s.hp ||
				p.stall != s.stall {
				v, _ := w.Entity(s.victim)
				fmt.Fprintf(out, "t%d dragon %d at %d,%d transit %d stall %d move %v->%d,%d state %d victim %v:%d (at %d,%d dist %d) pending %v:%d phase %d cd %d idle %v admitted %d hp %d mana %d\n",
					tick, id, s.x, s.y, s.transit, s.stall, s.hasTarget, s.tx, s.ty, s.state, s.hasVictim, s.victim, v.X, v.Y,
					max(v.X-e.X, e.X-v.X, v.Y-e.Y, e.Y-v.Y), s.hasPending, s.pending, s.phase, e.AttackCountdown, s.idle, s.admitted, s.hp, s.mana)
			}
			if s.refusals != p.refusals {
				fmt.Fprintf(out, "t%d dragon %d admission %s\n", tick, id, s.refusals)
			}
			prev[id] = s
		}
	}
	fmt.Fprintf(out, "-- %s summary at tick %d\n", name, w.Tick())
	for _, id := range dragons {
		e, _ := w.Entity(id)
		fmt.Fprintf(out, "dragon %d hp %d at %d,%d casts %v\n", id, e.HP, e.X, e.Y, casts[id])
		for _, pid := range party {
			fmt.Fprintf(out, "  dealt to %d: %d\n", pid, dealt[[2]sim.EntityID{id, pid}])
			total += dealt[[2]sim.EntityID{id, pid}]
		}
	}
	for _, pid := range party {
		e, _ := w.Entity(pid)
		fmt.Fprintf(out, "party %d hp %d/%d at %d,%d\n", pid, e.HP, e.MaxHP, e.X, e.Y)
	}
	return total
}
