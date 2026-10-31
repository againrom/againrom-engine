package game

import (
	"testing"

	"againrom/pkg/sim"
)

func TestFormationSettingReadsTheCommandedMode(t *testing.T) {
	for _, tc := range []struct{ command, setting int }{{0, 0}, {1, 1}, {2, 2}, {3, 1}, {-1, 1}} {
		w := visualIdentityWorld(t, []sim.EntityID{1}, 1)
		sim.Step(w, []sim.Command{sim.SetPlayerParameter(sim.SelfSlot, sim.PlayerParameterFormation, int32(tc.command))})
		mw := &mapWorld{world: w}
		if got := mw.formationSetting(); got != tc.setting {
			t.Errorf("command %d: formationSetting = %d, want %d", tc.command, got, tc.setting)
		}
		if got, _ := w.CommandFormationMode(sim.SelfSlot); got != sim.FormationModeOfSetting(int32(tc.command)) {
			t.Errorf("command %d: mode %d", tc.command, got)
		}
	}
}
