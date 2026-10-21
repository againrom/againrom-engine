package sim

import "testing"

func TestAutoCastTargetAdmission(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name   string
		target Entity
	}{
		{"living", Entity{HP: 100, MaxHP: 100}},
		{"wounded", Entity{HP: 20, MaxHP: 100}},
		{"fallen", Entity{HP: 0, MaxHP: 100}},
		{"finishable", Entity{HP: -9, MaxHP: 100}},
		{"bones", Entity{HP: -10, MaxHP: 100, Decay: DecayBones}},
		{"no_health", Entity{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if !autoCastTargetAdmitted(tc.target) {
				t.Fatal("on-map target refused before its policy's health checks")
			}
			tc.target.OffMap = true
			if autoCastTargetAdmitted(tc.target) {
				t.Fatal("off-map target admitted")
			}
		})
	}
}
