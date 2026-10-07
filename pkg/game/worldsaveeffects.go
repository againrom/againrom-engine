package game

import (
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// DIV-1185: unused Token fields use the existing constructor policy. Effect's
// complete serializer has no runtime caster field (SAV-EFFCHAIN-046).
func newEffectAttachment(e sim.ActiveEffect) (sav.DocumentRecordData, error) {
	kind := sim.OriginalEffectKind(e.Kind)
	if kind == 0 || e.Mode == 0 || e.Mode&^(sim.EffectDuration|sim.EffectContinuous|sim.EffectCharges) != 0 {
		return sav.DocumentRecordData{}, worldSaveUnsupportedf("actor %d effect %d has no complete Effect field constructor", e.Target, e.Spell)
	}
	row := sim.SavedEffectObject{E0C: uint8(e.Spell), Value: sim.ItemEffect{Kind: kind, Mode: uint8(e.Mode), Operand: uint32(uint16(e.Magnitude)) | uint32(e.Remaining)<<16}}
	back, err := originalActorEffect(row, e.Target)
	e.Caster, e.HasCaster = 0, false
	e.Magnitude = int32(int16(e.Magnitude))
	if err != nil || back != e {
		return sav.DocumentRecordData{}, worldSaveUnsupportedf("actor %d effect %d cannot reload its current fields", e.Target, e.Spell)
	}
	return savedCurrentEffectRecord(row), nil
}
