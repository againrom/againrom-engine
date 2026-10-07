package game

import (
	"fmt"
	"os"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func applyOriginalActorProfiles(ms *Mission, source []sav.ActorCurrent, report *OriginalSaveResume) error {
	if ms == nil || ms.Map == nil || ms.World == nil {
		return fmt.Errorf("original current profile: mission has no map/world")
	}
	party := make(map[sim.EntityID]bool, len(ms.Start.IDs))
	for _, id := range ms.Start.IDs {
		party[id] = true
	}
	byID := make(map[uint16][]sim.Entity)
	for _, e := range ms.World.Entities() {
		if e.MapUnitID != 0 {
			byID[e.MapUnitID] = append(byID[e.MapUnitID], e)
		}
	}
	seen := make(map[uint16]bool)
	var batch []sim.OriginalActorProfile
	for _, p := range source {
		var e sim.Entity
		if ms.actorRegistry != nil {
			bound, err := registryTarget(ms, p.Off)
			if err != nil {
				return err
			}
			if bound == nil {
				report.ProfilesExcluded++
				continue
			}
			e = *bound
		} else {
			if p.MapUnitID != 0 && originalPartyCarriesMapUnit(ms.Party, p.MapUnitID) {
				report.ProfilesParty++
				continue
			}
			if p.Stage != 0 || int16(p.HP) <= 0 || p.MapUnitID == 0 || int(p.Cell&255) >= int(ms.Map.Width) || int(p.Cell>>8) >= int(ms.Map.Height) {
				report.ProfilesExcluded++
				continue
			}
			if seen[p.MapUnitID] {
				return fmt.Errorf("original current profile: ambiguous MapUnitID %d at %d", p.MapUnitID, p.Off)
			}
			seen[p.MapUnitID] = true
			targets := byID[p.MapUnitID]
			if len(targets) == 0 {
				report.ProfilesUnmatched++
				continue
			}
			if len(targets) != 1 {
				return fmt.Errorf("original current profile: ambiguous MapUnitID %d at %d", p.MapUnitID, p.Off)
			}
			e = targets[0]
			if party[e.ID] {
				report.ProfilesParty++
				continue
			}
			if !e.Alive() || e.OffMap {
				report.ProfilesExcluded++
				continue
			}
		}
		var basis *sim.CurrentProfileBasis
		if ms.actorRegistry != nil {
			binding, _ := ms.actorRegistry.actor(p.Off)
			if current, present := ms.actorRegistry.profiles[binding.Source.Identity]; present {
				basis = &current
			}
		}
		if (basis == nil || *basis == sim.ProfileOriginalCurrent) && p.ManaPeriod == 0 && p.MaxMana != 0 {
			return fmt.Errorf("original current profile: MapUnitID %d has nonzero mana maximum with zero regeneration divisor", p.MapUnitID)
		}
		secondary := sim.SecondaryDamage{Base: p.ElementalBase, Spread: p.ElementalSpread}
		if secondary.Base != 0 || secondary.Spread != 0 {
			if p.ElementalKind < 1 || p.ElementalKind > 5 {
				return fmt.Errorf("original current profile: MapUnitID %d unsupported elemental selector %d", p.MapUnitID, p.ElementalKind)
			}
			// HERO-DMG2-029: raw selectors 1..5 name C4,CA,C8,C6,CC,
			// not the canonical C4,C6,C8,CA,CC protection-array order.
			// data.ElementalSelectorOrder is this same permutation, shared
			// with data.HumanState.Derived's own current-carry path.
			secondary.Selector = data.ElementalSelectorOrder[p.ElementalKind-1]
		}
		batch = append(batch, sim.OriginalActorProfile{ID: e.ID, ProfileBasis: basis, Reaction: p.Reaction, Mind: p.Mind, Spirit: p.Spirit,
			ToHit: p.ToHit, Defence: p.Defence, Absorption: p.Absorption, DamageBase: p.DamageBase, DamageSpread: p.DamageSpread, XPSlot: p.Active,
			SecondBase: p.SecondBase, SecondSpread: p.SecondSpread, SecondaryDamage: secondary, Protection: p.Protection, Resistance: p.Resistance,
			HealthPeriod: p.HealthPeriod, ManaPeriod: p.ManaPeriod, HealthRegeneration: p.HealthRegeneration, ManaRegeneration: p.ManaRegeneration,
			HealthHundredths: p.HealthHundredths, ManaHundredths: p.ManaHundredths})
	}
	if err := ms.World.ImportOriginalActorProfiles(batch); err != nil {
		return err
	}
	report.ProfilesRestored = len(batch)
	return nil
}

// The source-current authority is intentionally retired by a legal native
// rebuild, not silently overwritten by the cached ALM Hero (DIV-738). This
// diagnostic is presentation-only; the durable transition lives in the world.
func reportOriginalProfileRetirement(w *sim.World, before sim.Entity) {
	if before.CurrentProfileBasis != sim.ProfileOriginalCurrent {
		return
	}
	for _, after := range w.Entities() {
		if after.ID == before.ID && after.CurrentProfileBasis == sim.ProfileNativeRetired {
			fmt.Fprintf(os.Stderr, "original current profile: actor %d now uses native recompute (DIV-738)\n", before.ID)
			return
		}
	}
}
