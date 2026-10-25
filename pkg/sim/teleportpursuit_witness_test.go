package sim

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// The Teleport pursuit witness. pkg/game's TestReleaseTeleportPursuitWitness
// drives the EN mission-10 SAV and writes each recorded tick's world form to
// AGAINROM_TELEPORT_OUT/<case>.worlds; this reader decodes them and prints,
// per pursuer, the fields only this package can read: the order record, the
// remembered attacker cell, the pursuit state and route end, and the group's
// sight of the hero.

const tpWitnessHero = EntityID(35)

type tpEntry struct {
	tick uint64
	w    *World
}

func tpEntries(t *testing.T, path string) []tpEntry {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var out []tpEntry
	for len(raw) >= 12 {
		tick := binary.LittleEndian.Uint64(raw)
		n := int(binary.LittleEndian.Uint32(raw[8:]))
		if 12+n > len(raw) {
			t.Fatalf("%s: truncated entry at tick %d", path, tick)
		}
		w := &World{}
		if err := w.UnmarshalBinary(raw[12 : 12+n]); err != nil {
			t.Fatalf("%s tick %d: %v", path, tick, err)
		}
		out = append(out, tpEntry{tick, w})
		raw = raw[12+n:]
	}
	return out
}

// tpGroupSees reports whether the hero is among a saved group's candidates
// (sight, remembered attacker cells, hostility), read without ageing any
// notice, and whether the stored circle keeps it.
func (w *World) tpGroupSees(g *SavedGroup, hero int) (sees, kept bool) {
	members, _ := w.savedLiving(g)
	if len(members) == 0 {
		return false, false
	}
	ag := aiGroup{group: g.ID, owner: w.entities[members[0]].Owner, members: members}
	cands := w.groupCandidates(ag, false)
	sees = containsIndex(cands, hero)
	kept = containsIndex(w.clipCircle(g.AI[0x20], w.storedCircle(g, members), cands), hero)
	return sees, kept
}

func tpCell(c uint16) string {
	if c == 0 {
		return "-"
	}
	return fmt.Sprintf("%d,%d", c&255, c>>8)
}

func (w *World) tpLines(hero int, watch map[uint32]bool) []string {
	h := w.entities[hero]
	var out []string
	for gi := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[gi]
		involved := watch[g.ID]
		for _, m := range g.Members {
			if i := indexOfEntity(w.entities, m.Entity); i >= 0 && w.entities[i].HasAttackTarget && w.entities[i].AttackTarget == h.ID {
				involved = true
			}
		}
		if !involved {
			continue
		}
		sees, kept := w.tpGroupSees(g, hero)
		for _, m := range g.Members {
			i := indexOfEntity(w.entities, m.Entity)
			if i < 0 || !w.entities[i].Alive() {
				continue
			}
			e := w.entities[i]
			o := w.savedOrder(e.ID)
			kind, dest, post, assigned := "-", "-", "-", uint32(0)
			if o != nil {
				kind = fmt.Sprintf("%d", o.Raw[8])
				dest = tpCell(binary.LittleEndian.Uint16(o.Raw[10:]))
				post = tpCell(binary.LittleEndian.Uint16(o.Raw[0:]))
				assigned = binary.LittleEndian.Uint32(o.Raw[0x20:])
			}
			n := w.attackNoticeAt(i)
			victim := "-"
			if e.HasAttackTarget {
				victim = fmt.Sprintf("%d", e.AttackTarget)
			}
			end := "-"
			if r := w.routes[i]; len(r) > 0 {
				end = fmt.Sprintf("%d,%d(%d)", r[len(r)-1].x, r[len(r)-1].y, len(r))
			}
			walk := "-"
			if e.HasTarget {
				walk = fmt.Sprintf("%d,%d", e.TargetX, e.TargetY)
			}
			p := e.Pursuit
			pursuit := "-"
			if p.Held {
				pursuit = fmt.Sprintf("end %d,%d aim %d,%d n%d p%d", p.EndX, p.EndY, p.AimX, p.AimY, p.Count, p.Passes)
			}
			out = append(out, fmt.Sprintf("  g%d ord%d act%d sees %v kept %v | %d at %d,%d hp %d kind %s dest %s post %s asg %#x victim %s phase %d walk %s pursuit [%s] route %s notice %s/%d dHero %d dPost %d",
				g.ID, g.AI[0x20], g.AI[0x45], sees, kept, e.ID, e.X, e.Y, e.HP, kind, dest, post, assigned, victim, e.AttackPhase, walk, pursuit, end,
				tpCell(n.Cell), n.Scans, cellOf(&e).chebyshevTo(cellOf(&h)), cellOf(&e).chebyshevTo(cell{x: e.PostX, y: e.PostY})))
		}
	}
	return out
}

// TestTeleportPursuitWitnessRead prints every case file the game-side witness
// wrote. It asserts nothing about the original.
func TestTeleportPursuitWitnessRead(t *testing.T) {
	dir := os.Getenv("AGAINROM_TELEPORT_OUT")
	if dir == "" {
		t.Skip("set AGAINROM_TELEPORT_OUT to the game-side Teleport witness output")
	}
	files, _ := filepath.Glob(filepath.Join(dir, "*.worlds"))
	sort.Strings(files)
	if len(files) == 0 {
		t.Skip("no case files")
	}
	watch := map[uint32]bool{}
	for _, s := range strings.Split(os.Getenv("AGAINROM_TELEPORT_GROUPS"), ",") {
		var id uint32
		if _, err := fmt.Sscan(s, &id); err == nil {
			watch[id] = true
		}
	}
	for _, path := range files {
		name := strings.TrimSuffix(filepath.Base(path), ".worlds")
		var log strings.Builder
		for k, en := range tpEntries(t, path) {
			if k == 0 {
				w := en.w
				if hi := indexOfEntity(w.entities, tpWitnessHero); hi >= 0 {
					h := w.entities[hi]
					rule, ok := w.bookSpell(h, 26)
					var level int32
					if rule.School < skillSlots {
						level = h.Skill[rule.School]
					}
					power := spellPowerUnder(w.rules, rule, level, h.Mind)
					fmt.Fprintf(&log, "tick %d slot range %d teleport rule ok %v id %d school %d level %d mind %d power %d maxrange %d book %v cast range %d record range %d cost %d\n",
						en.tick, h.Book.Slots[25].Range, ok, rule.ID, rule.School, level, h.Mind, power, rule.MaxRange, rule.bookInstance,
						spellRangeOf(w.rules, rule, power, rangeCast), spellRangeOf(w.rules, rule, power, rangeRecord), rule.ManaCost)
				}
			}
			w := en.w
			hero := indexOfEntity(w.entities, tpWitnessHero)
			if hero < 0 {
				fmt.Fprintf(&log, "tick %d hero absent\n", en.tick)
				continue
			}
			h := w.entities[hero]
			fmt.Fprintf(&log, "tick %d hero %d,%d hp %d mana %d\n", en.tick, h.X, h.Y, h.HP, h.Mana)
			for _, line := range w.tpLines(hero, watch) {
				fmt.Fprintln(&log, line)
			}
		}
		if err := os.WriteFile(filepath.Join(dir, name+".trace"), []byte(log.String()), 0600); err != nil {
			t.Fatal(err)
		}
	}
}
