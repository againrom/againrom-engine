package ui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"againrom/pkg/render/terrain"
)

type AnimationState struct {
	Count                  uint32
	RemainderUS            int
	PhaseUS, PhasePeriodUS int
}

func (s *AnimationState) UnmarshalJSON(raw []byte) error {
	var fields struct {
		Count                               *uint32
		RemainderUS, PhaseUS, PhasePeriodUS *int
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&fields); err != nil {
		return err
	}
	if fields.Count == nil || fields.RemainderUS == nil || fields.PhaseUS == nil || fields.PhasePeriodUS == nil {
		return fmt.Errorf("saved animation clock requires four numeric fields")
	}
	value := AnimationState{*fields.Count, *fields.RemainderUS, *fields.PhaseUS, *fields.PhasePeriodUS}
	if err := ValidateAnimationState(value); err != nil {
		return err
	}
	*s = value
	return nil
}

func ValidateAnimationState(s AnimationState) error {
	if s.RemainderUS < 0 || s.RemainderUS >= terrain.CadencePeriod(terrain.CadenceRungMin) {
		return fmt.Errorf("saved animation remainder exceeds the cadence range")
	}
	if s.PhaseUS < 0 || s.PhaseUS >= terrain.CadencePeriod(terrain.CadenceRungMin) || s.PhasePeriodUS < 0 || s.PhasePeriodUS == 0 && s.PhaseUS != 0 || s.PhasePeriodUS != 0 && s.PhasePeriodUS != terrain.CadencePeriod(terrain.CadenceRung(s.PhasePeriodUS)) {
		return fmt.Errorf("saved crossing phase exceeds the cadence range")
	}
	return nil
}

func (v *Viewer) SaveAnimation() AnimationState {
	return AnimationState{Count: v.anim.Count(), RemainderUS: v.anim.Remainder(), PhaseUS: v.phaseUS, PhasePeriodUS: v.phasePeriodUS}
}

func (v *Viewer) RestoreAnimation(s AnimationState) error {
	if err := ValidateAnimationState(s); err != nil {
		return err
	}
	v.anim.Restore(s.Count, s.RemainderUS)
	v.SetPhase(s.PhaseUS, s.PhasePeriodUS)
	v.last = time.Time{}
	v.animationRestored = true
	return nil
}
