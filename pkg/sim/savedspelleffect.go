package sim

// SavedEffect holds ordinary Effect fields and the frozen direct-damage block.
type SavedEffect struct {
	Class string // "Effect" or "Effect_DirectDamage".

	E3C, E3D uint8
	E40      uint32
	E0C      uint8

	// DirectDamage is Effect_DirectDamage's own 24 raw bytes. Zero on a bare
	// Effect.
	DirectDamage [24]byte
}

// SavedSpellEffect retains the historical typed wire shape. SavedSpellGraph
// owns executable node identity; the game Document retains untouched Token fields.
type SavedSpellEffect struct {
	Class string // "SpellEffect", "PointEffect", "AreaEffect" or "SpellTransport".

	// SE40/SE41 are on every subtype.
	SE40, SE41 uint8

	// PointEffect's target key is bound separately to an EntityID (SAV-1010).
	PE48 *SavedEffect
	PE44 uint32

	// AreaEffect only.
	AE48 [4]byte
	AE4C uint16
	AE44 *SavedEffect

	// SpellTransport only.
	ST44 *SavedSpellEffect
	ST48 *SavedSpellEffect
	ST4C uint16
}

// SavedSpellEffects returns this world's carried top-level SpellEffect-list
// graph, in archive order. It is a fresh slice: mutating the result slice
// itself (appending, reassigning an element) cannot reach the world it came
// from. But unlike SavedCellRecords, whose fields carry no pointers, this one
// hands out live pointers: each PE48/AE44/ST44/ST48 still aliases the exact
// *SavedEffect/*SavedSpellEffect the world's own copy points to, so mutating
// what one of those pointers refers to reaches the world.
func (w *World) SavedSpellEffects() []SavedSpellEffect {
	out := make([]SavedSpellEffect, len(w.savedSpellEffects))
	copy(out, w.savedSpellEffects)
	return out
}

// SetSavedSpellEffects replaces the carried graph outright. Its production
// caller is applyOriginalSpellEffects (pkg/game/originalspelleffects.go) on
// LOAD, carrying the archive's own decoded graph in for the first time.
// Form85 gives the field a wire position (carriedresumebinary.go), so both
// staging round trips now carry it through UnmarshalBinary like every other
// field and neither calls this any more.
func (w *World) SetSavedSpellEffects(effects []SavedSpellEffect) {
	w.savedSpellEffects = append([]SavedSpellEffect(nil), effects...)
}
