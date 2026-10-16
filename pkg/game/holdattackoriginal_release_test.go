package game

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"againrom/pkg/sim"
)

// holdAttackGroup returns the registry Group that binds id: its ID, order and
// member count.
func holdAttackGroup(w *sim.World, id sim.EntityID) (uint32, uint8, int) {
	groups, _, _ := w.SavedGroups()
	for _, g := range groups {
		for _, m := range g.Members {
			if m.Bound && m.Entity == id {
				return g.ID, g.AI[0x20], len(g.Members)
			}
		}
	}
	return 0, 0, 0
}

// holdAttackPair picks the first living party member at state 0xc in an
// order-3 Group and the hostile nearest to it.
func holdAttackPair(w *sim.World) (sim.EntityID, sim.EntityID, bool) {
	var hero sim.EntityID
	found := false
	for _, e := range w.Entities() {
		if e.Owner != sim.SelfSlot || !e.Alive() || e.OffMap || e.ActorState != 0xc {
			continue
		}
		if _, order, _ := holdAttackGroup(w, e.ID); order == 3 {
			hero, found = e.ID, true
			break
		}
	}
	if !found {
		return 0, 0, false
	}
	h, _ := w.Entity(hero)
	rel := w.Relations()
	var victim sim.EntityID
	best := int64(-1)
	for _, e := range w.Entities() {
		if !e.Alive() || e.OffMap || !rel.Hostile(sim.SelfSlot, e.Owner) {
			continue
		}
		d := int64(e.X-h.X)*int64(e.X-h.X) + int64(e.Y-h.Y)*int64(e.Y-h.Y)
		if best < 0 || d < best {
			best, victim = d, e.ID
		}
	}
	return hero, victim, best >= 0
}

// TestReleaseAttackOrderReleasesAnOriginalHoldMember: an original SAV loads a
// party member at state 0xc in an order-3 Group. The player's attack order
// builds a new Group at order 0 for it alone (AI-CMD-033, AI-CMD-054), the same
// as for a member Hold set in this engine. The source SAVs are read from
// AGAINROM_ORIGINAL_HOLD_ATTACK_DIR/<en|ru>; a language with no SAV there is
// skipped.
func TestReleaseAttackOrderReleasesAnOriginalHoldMember(t *testing.T) {
	root, src := os.Getenv("AGAINROM_ASSETS"), os.Getenv("AGAINROM_ORIGINAL_HOLD_ATTACK_DIR")
	if root == "" || src == "" {
		t.Skip("AGAINROM_ASSETS and AGAINROM_ORIGINAL_HOLD_ATTACK_DIR are required")
	}
	lang := map[string]string{"english": "en", "russian": "ru"}[InspectInstall(root).Language]
	names, _ := filepath.Glob(filepath.Join(src, lang, "*.sav"))
	sort.Strings(names)
	if lang == "" || len(names) == 0 {
		t.Skip("no original SAV for this install's language")
	}
	for _, path := range names {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, name), raw, 0o644); err != nil {
				t.Fatal(err)
			}
			_, _, live := placedSpeakerLoad(t, dir, name)
			w := live.world
			hero, victim, ok := holdAttackPair(w)
			if !ok {
				t.Fatal("the SAV holds no party member at state 0xc in an order-3 Group with a hostile")
			}
			heldID, _, heldN := holdAttackGroup(w, hero)
			live.pending = append(live.pending, sim.Attack(hero, victim))
			live.tick()
			gid, order, n := holdAttackGroup(w, hero)
			if gid == heldID || order != 0 || n != 1 {
				t.Fatalf("member %d after the attack order on %d: Group %d order %d members %d, want a new one-member Group at order 0 (held Group %d, %d members)",
					hero, victim, gid, order, n, heldID, heldN)
			}
			t.Logf("member %d attacks %d: held Group %d (%d members), now Group %d order 0; tick 1 hash %016x", hero, victim, heldID, heldN, gid, w.Hash())
			for k := 2; k <= 901; k++ {
				live.tick()
				if k == 601 || k == 901 {
					v, _ := w.Entity(victim)
					_, order, _ = holdAttackGroup(w, hero)
					t.Logf("tick %d: Group order %d, victim HP %d, hash %016x", k, order, v.HP, w.Hash())
				}
			}
		})
	}
}
