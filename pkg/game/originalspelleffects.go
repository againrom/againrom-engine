package game

import (
	"fmt"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// originalSpellEffects decodes the complete top-level SpellEffect-list graph
// before a mission is prepared, mirroring originalCellRecords' own shape for
// its table.
func originalSpellEffects(f *sav.File) ([]sav.SpellEffect, bool, error) {
	return f.SpellEffects()
}

// Both original LOAD doors first carry the typed fields. importOriginalWorldEffects
// then binds archive aliases, area drivers and PointEffect targets from the Document.
func applyOriginalSpellEffects(ms *Mission, effects []sav.SpellEffect, present bool, r *OriginalSaveResume) error {
	if !present {
		return nil
	}
	if ms == nil || ms.World == nil {
		return fmt.Errorf("original spell effects: mission has no world")
	}
	conv := newSpellEffectConverter()
	out := make([]sim.SavedSpellEffect, len(effects))
	for i, e := range effects {
		out[i] = conv.toSaved(e)
	}
	ms.World.SetSavedSpellEffects(out)
	if r != nil {
		r.SpellEffects, r.SpellEffectsApplied = len(effects), true
	}
	return nil
}

// This historical fixed-shape helper is retained for round-trip callers.
// Ordinary SAVE uses the current Document producer, including graph retirement.
func exportOriginalSpellEffects(f *sav.File, w *sim.World) error {
	if f == nil || f.World == nil {
		return fmt.Errorf("original spell effects export: this save has no world session")
	}
	if w == nil {
		return fmt.Errorf("original spell effects export: nil world")
	}
	_, present, err := f.SpellEffects()
	if err != nil {
		return err
	}
	if !present {
		return fmt.Errorf("original spell effects export: this save has no world session")
	}
	conv := newSavedSpellEffectConverter()
	saved := w.SavedSpellEffects()
	out := make([]sav.SpellEffect, len(saved))
	for i, s := range saved {
		out[i] = conv.toSav(s)
	}
	return f.SetSpellEffects(out)
}

// spellEffectConverter turns a sav.File's own decoded SpellEffect graph into
// its sim.SavedSpellEffect projection, converting each shared *sav.Effect or
// *sav.SpellEffect object exactly once — on its first encounter, in archive
// order. This preserves any sharing in its INPUT values, but File.SpellEffects
// currently allocates a distinct value at every archive reference site. It is
// therefore not an archive-identity guarantee (DIV-939).
type spellEffectConverter struct {
	effects map[*sav.Effect]*sim.SavedEffect
	nested  map[*sav.SpellEffect]*sim.SavedSpellEffect
}

func newSpellEffectConverter() *spellEffectConverter {
	return &spellEffectConverter{
		effects: map[*sav.Effect]*sim.SavedEffect{},
		nested:  map[*sav.SpellEffect]*sim.SavedSpellEffect{},
	}
}

func (c *spellEffectConverter) toSaved(e sav.SpellEffect) sim.SavedSpellEffect {
	return sim.SavedSpellEffect{
		Class: e.Class, SE40: e.SE40, SE41: e.SE41,
		PE48: c.effect(e.PE48), PE44: e.PE44,
		AE48: e.AE48, AE4C: e.AE4C, AE44: c.effect(e.AE44),
		ST44: c.nestedPtr(e.ST44), ST48: c.nestedPtr(e.ST48), ST4C: e.ST4C,
	}
}

func (c *spellEffectConverter) effect(e *sav.Effect) *sim.SavedEffect {
	if e == nil {
		return nil
	}
	if out, ok := c.effects[e]; ok {
		return out
	}
	out := &sim.SavedEffect{Class: e.Class, E3C: e.E3C, E3D: e.E3D, E40: e.E40, E0C: e.E0C,
		DirectDamage: e.DirectDamage}
	c.effects[e] = out
	return out
}

func (c *spellEffectConverter) nestedPtr(e *sav.SpellEffect) *sim.SavedSpellEffect {
	if e == nil {
		return nil
	}
	if out, ok := c.nested[e]; ok {
		return out
	}
	// Registered before recursing: a diamond-shaped reference (two records
	// both naming this same nested SpellEffect) must convert it once, and
	// registering first is what lets the second visit find it already done.
	out := &sim.SavedSpellEffect{}
	c.nested[e] = out
	*out = c.toSaved(*e)
	return out
}

// savedSpellEffectConverter is spellEffectConverter reversed, for
// exportOriginalSpellEffects. sav.SetSpellEffects itself only reads Class and
// each record's own scalar/presence fields (spelleffect.go), never the value
// struct's own Off, so leaving Off zero here is not a loss.
type savedSpellEffectConverter struct {
	effects map[*sim.SavedEffect]*sav.Effect
	nested  map[*sim.SavedSpellEffect]*sav.SpellEffect
}

func newSavedSpellEffectConverter() *savedSpellEffectConverter {
	return &savedSpellEffectConverter{
		effects: map[*sim.SavedEffect]*sav.Effect{},
		nested:  map[*sim.SavedSpellEffect]*sav.SpellEffect{},
	}
}

func (c *savedSpellEffectConverter) toSav(s sim.SavedSpellEffect) sav.SpellEffect {
	return sav.SpellEffect{
		Class: s.Class, SE40: s.SE40, SE41: s.SE41,
		PE48: c.effect(s.PE48), PE44: s.PE44,
		AE48: s.AE48, AE4C: s.AE4C, AE44: c.effect(s.AE44),
		ST44: c.nestedPtr(s.ST44), ST48: c.nestedPtr(s.ST48), ST4C: s.ST4C,
	}
}

func (c *savedSpellEffectConverter) effect(s *sim.SavedEffect) *sav.Effect {
	if s == nil {
		return nil
	}
	if out, ok := c.effects[s]; ok {
		return out
	}
	out := &sav.Effect{Class: s.Class, E3C: s.E3C, E3D: s.E3D, E40: s.E40, E0C: s.E0C,
		DirectDamage: s.DirectDamage}
	c.effects[s] = out
	return out
}

func (c *savedSpellEffectConverter) nestedPtr(s *sim.SavedSpellEffect) *sav.SpellEffect {
	if s == nil {
		return nil
	}
	if out, ok := c.nested[s]; ok {
		return out
	}
	out := &sav.SpellEffect{}
	c.nested[s] = out
	*out = c.toSav(*s)
	return out
}
