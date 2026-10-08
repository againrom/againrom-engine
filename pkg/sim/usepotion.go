package sim

// usePotion shares the base Item application boundary, not the spellbook:
// ITEM-EQUIP-006 refuses health <= 0 before applying any ordered effect.
// Unsupported payloads are refused as a whole before either item or actor
// changes. ITEM-USE-113 consumes capped and class-inapplicable uses too.
func (w *World) usePotion(i, index int) bool {
	if i < 0 || i >= len(w.entities) || w.entities[i].HP <= 0 || w.entities[i].OffMap || index < 0 || index >= len(w.carried[i]) {
		return false
	}
	item := w.carried[i][index].Instance()
	if !UsablePotion(item) {
		return false
	}
	if !w.sourceMutationReady(i) {
		return false
	}
	if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class == 2 {
		n := w.sourceMutationCopy(i)
		if !n.applyPotion(i, index, item) || !n.savedMutationValid() {
			return false
		}
		*w = n
	} else if !w.applyPotion(i, index, item) {
		return false
	}
	w.clearFelled(i)
	return true
}

func (w *World) applyPotion(i, index int, item ItemInstance) bool {
	for _, effect := range item.Effects {
		e := &w.entities[i]
		if effect.Mode&3 != 0 {
			if e.ActorLoad.Source.Class == 2 {
				if !w.attachSourcePotionEffect(i, effect) {
					return false
				}
			} else {
				w.attachPotionEffect(e.ID, effect)
			}
			continue
		}
		amount := int32(effect.Operand)
		if e.ActorLoad.Source.Class == 2 {
			if !w.sourcePotion(i, effect.Kind, amount) {
				return false
			}
			continue
		}
		switch effect.Kind {
		case 2, 3, 4, 5:
			slot := map[uint8]int{2: 0, 4: 1, 3: 2, 5: 3}[effect.Kind]
			if amount > e.PotionHeadroom[slot] {
				amount = e.PotionHeadroom[slot]
			}
			if amount > 100-e.PotionStats[slot] {
				amount = 100 - e.PotionStats[slot]
			}
			e.PotionStats[slot] += amount
			e.PotionHeadroom[slot] -= amount
			if slot == 0 && e.NativeBasis.BodyPresent && e.NativeBasis.BodyKnown {
				e.NativeBasis.Body += uint16(amount)
			}
		case 6:
			e.setCurrentHealth(potionPool(e.HP, e.MaxHP, amount))
		case 9:
			if isMage(*e) {
				e.setCurrentMana(potionPool(e.Mana, e.MaxMana, amount))
			}
		}
	}
	return w.consumeCarriedUnit(i, index)
}

// UseCarriedPotion is the same atomic command operation without advancing a
// clock. The town projection uses it on an isolated, typed actor value.
func (w *World) UseCarriedPotion(id EntityID, index int) bool {
	return w.usePotion(indexOfEntity(w.entities, id), index)
}

// UsablePotion bounds the implemented general-item vocabulary before mutation.
// All thirteen shipped Potion rows are covered. Unknown custom effects refuse
// the whole item instead of consuming a partially applied payload (DIV-582).
func UsablePotion(item ItemInstance) bool {
	if item.Kind != 3 || item.Price < 0 || len(item.Effects) == 0 || len(item.Effects) > 64 {
		return false
	}
	for _, e := range item.Effects {
		if e.Mode&3 != 0 {
			if (e.Mode != 1 && e.Mode != 2) || (e.Kind != 8 && e.Kind != 11 && e.Kind != 16) || uint16(e.Operand>>16) == 0 {
				return false
			}
		} else if (e.Mode != 0 && e.Mode != 8) || (e.Kind != 6 && e.Kind != 9 && (e.Kind < 2 || e.Kind > 5)) {
			return false
		} else if e.Kind >= 2 && e.Kind <= 5 && (int32(e.Operand) < 0 || e.Operand > 100) {
			return false
		}
	}
	return true
}

func (w *World) SetPotionHeadroom(id EntityID, headroom [4]int32) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 {
		return false
	}
	for _, n := range headroom {
		if n < 0 || n > 100 {
			return false
		}
	}
	w.entities[i].PotionHeadroom = headroom
	return true
}

// CopyPotionEffects carries already-applied metadata through a loader rebuild
// that copied the source's entities. It never applies a potion a second time.
func (w *World) CopyPotionEffects(source *World) {
	if source == nil {
		return
	}
	for _, e := range source.attached {
		if e.Spell == 0 && indexOfEntity(w.entities, e.Target) >= 0 {
			k, exists := effectIndex(w.attached, e.Target, 0)
			if exists {
				w.attached[k] = e
				continue
			}
			w.attached = append(w.attached, attachedEffect{})
			copy(w.attached[k+1:], w.attached[k:])
			w.attached[k] = e
		}
	}
}

// All non-spell potions share Token id zero. Replacement preserves the old
// characteristic and mode, even when the new potion names a different one
// (MAGIC-CONSUME-143). No spell cast or spell-book mark is produced here.
func (w *World) attachPotionEffect(id EntityID, incoming ItemEffect) {
	kind := map[uint8]EffectKind{8: EffectHealthRegeneration, 11: EffectManaRegeneration, 16: EffectAbsorption}[incoming.Kind]
	mode, magnitude, duration := EffectMode(incoming.Mode), int32(int16(incoming.Operand)), uint16(incoming.Operand>>16)
	k, exists := effectIndex(w.attached, id, 0)
	if exists {
		if mode == EffectContinuous {
			w.attached[k].Remaining = duration
			return
		}
		kind, mode = w.attached[k].Kind, w.attached[k].Mode
		w.removeAttachedAt(k)
		k, _ = effectIndex(w.attached, id, 0)
	}
	effect := attachedEffect{Target: id, Kind: kind, Mode: mode, Magnitude: magnitude, Remaining: duration}
	w.attached = append(w.attached, attachedEffect{})
	copy(w.attached[k+1:], w.attached[k:])
	w.attached[k] = effect
	if ti := indexOfEntity(w.entities, id); ti >= 0 {
		// Keep the nominal magnitude even for a class-inapplicable mana
		// effect: a later id-zero replacement still retains its kind.
		if landed, _ := w.applyEffectDelta(ti, kind, magnitude); kind == EffectAbsorption && mode != EffectContinuous {
			w.attached[k].Magnitude = landed
		}
	}
}

// RestorePotionEffect applies a carried town/mission-boundary timer to a fresh
// actor exactly once. Native world decoding never calls this constructor.
func (w *World) RestorePotionEffect(id EntityID, effect ActiveEffect) bool {
	itemEffect, ok := potionAttachment(effect)
	if !ok || indexOfEntity(w.entities, id) < 0 {
		return false
	}
	if !w.sourceDeriveReady(indexOfEntity(w.entities, id)) {
		return false
	}
	if i := indexOfEntity(w.entities, id); w.entities[i].ActorLoad.Source.Class == 2 {
		n := w.sourceMutationCopy(i)
		if !n.attachSourcePotionEffect(i, itemEffect) {
			return false
		}
		*w = n
		return true
	}
	w.attachPotionEffect(id, itemEffect)
	return true
}

func (w *World) RestoreAppliedPotionEffect(id EntityID, effect ActiveEffect) bool {
	if _, ok := potionAttachment(effect); !ok || indexOfEntity(w.entities, id) < 0 {
		return false
	}
	k, exists := effectIndex(w.attached, id, 0)
	if exists {
		return false
	}
	w.attached = append(w.attached, attachedEffect{})
	copy(w.attached[k+1:], w.attached[k:])
	w.attached[k] = attachedEffect{Target: id, Kind: effect.Kind, Mode: effect.Mode,
		Magnitude: effect.Magnitude, Remaining: effect.Remaining}
	return true
}

// RestoreCarriedPotionEffect restores the new actor's effective delta while
// retaining the raw history in which this same timed attachment already landed.
func (w *World) RestoreCarriedPotionEffect(id EntityID, effect ActiveEffect) bool {
	i := indexOfEntity(w.entities, id)
	if i < 0 || w.entities[i].ActorLoad.Source.Class != 0 {
		return false
	}
	if _, exists := effectIndex(w.attached, id, 0); exists {
		return false
	}
	basis := w.entities[i].NativeBasis
	if !w.RestorePotionEffect(id, effect) {
		return false
	}
	w.entities[i].NativeBasis = basis
	return true
}

// ProjectPotionEffect applies the same attachment arithmetic to a value only.
// A town sheet has no simulation clock or actor to mutate; its carried timer
// remains untouched. Never use this on an already-applied live world actor.
func ProjectPotionEffect(base Entity, effect ActiveEffect) (Entity, bool) {
	if _, ok := potionAttachment(effect); !ok {
		return base, false
	}
	projected, _, ok := effectLanding(base, effect.Kind, effect.Magnitude)
	return projected, ok
}

func potionAttachment(effect ActiveEffect) (ItemEffect, bool) {
	if effect.Spell != 0 || effect.Remaining == 0 || effect.Magnitude < -32768 || effect.Magnitude > 32767 {
		return ItemEffect{}, false
	}
	var kind uint8
	switch effect.Kind {
	case EffectHealthRegeneration:
		kind = 8
	case EffectManaRegeneration:
		kind = 11
	case EffectAbsorption:
		kind = 16
	default:
		return ItemEffect{}, false
	}
	item := ItemInstance{Kind: 3, Effects: []ItemEffect{{Kind: kind, Mode: uint8(effect.Mode), Operand: uint32(uint16(effect.Magnitude)) | uint32(effect.Remaining)<<16}}}
	return item.Effects[0], UsablePotion(item)
}

// Do the addition wide before the existing pool ceiling. Corrupt/custom
// payloads cannot wrap a positive restoration into negative health.
func potionPool(current, maximum, amount int32) int32 {
	v := int64(current) + int64(amount)
	if v > int64(maximum) {
		v = int64(maximum)
	}
	if v < -1<<31 {
		v = -1 << 31
	}
	return int32(v)
}

func (w *World) consumeCarriedUnit(i, index int) bool {
	live := w
	if w.savedObjects != nil || w.entities[i].ActorLoad.Source.Class != 0 {
		n := w.sourceMutationCopy(i)
		w = &n
	}
	before := w.beginLoadMutation(i)
	item, ok := w.takeCarriedObject(i, index, false)
	if !ok {
		return false
	}
	if item.ObjectID != 0 && w.savedObjects.Dispose(item.ObjectID) != nil {
		return false
	}
	if !w.finishLoadMutation(i, before) || !w.savedMutationValid() {
		return false
	}
	if w != live {
		*live = *w
	}
	return true
}

// The source arm follows the same id-zero replacement law, but a failed
// derive must propagate to the whole item transaction instead of leaving a
// deleted old attachment or a partly applied ordered effect list.
func (w *World) attachSourcePotionEffect(i int, incoming ItemEffect) bool {
	id := w.entities[i].ID
	kind := map[uint8]EffectKind{8: EffectHealthRegeneration, 11: EffectManaRegeneration, 16: EffectAbsorption}[incoming.Kind]
	mode, magnitude, duration := EffectMode(incoming.Mode), int32(int16(incoming.Operand)), uint16(incoming.Operand>>16)
	k, exists := effectIndex(w.attached, id, 0)
	if exists {
		if mode == EffectContinuous {
			w.attached[k].Remaining = duration
			return true
		}
		old := w.attached[k]
		kind, mode = old.Kind, old.Mode
		if old.Mode != EffectContinuous {
			if _, ok := w.sourceEffect(i, old.Kind, -old.Magnitude); !ok {
				return false
			}
		}
		w.attached = append(w.attached[:k], w.attached[k+1:]...)
	}
	landed, ok := w.sourceEffect(i, kind, magnitude)
	if !ok {
		return false
	}
	if kind == EffectAbsorption && mode != EffectContinuous {
		magnitude = landed
	}
	k, _ = effectIndex(w.attached, id, 0)
	w.attached = append(w.attached, attachedEffect{})
	copy(w.attached[k+1:], w.attached[k:])
	w.attached[k] = attachedEffect{Target: id, Kind: kind, Mode: mode, Magnitude: magnitude, Remaining: duration}
	return true
}
