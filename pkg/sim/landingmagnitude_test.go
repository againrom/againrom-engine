package sim

import (
	"fmt"
	"testing"
)

// siteMagnitudes is what each landing site passes on today for one unit: the
// non-timed sites give the class cut alone, the timed site the class cut
// bounded at the source basis.
func TestLandingMagnitudeTable(t *testing.T) {
	for _, second := range []bool{false, true} {
		for _, arm := range []uint16{12, 17} {
			rule := probeRule(arm, second)
			a := spellArmOf(rule.arm())
			for _, power := range []int32{30, 150, 400} {
				mag := a.magnitude(0, power)
				for _, humanoid := range []bool{false, true} {
					for _, source := range []uint8{0, 1} {
						w := &World{entities: []Entity{{ID: 1, Humanoid: humanoid, ScanRange: 3}}}
						w.entities[0].ActorLoad.Source.Class = source
						cut := a.sightCut(&w.entities[0], EffectScanRange, mag)
						bound := w.sourceSightCut(0, rule, EffectScanRange, cut)
						t.Logf("second=%v arm=%d power=%d humanoid=%v source=%d mag=%d nonTimed=%d timed=%d landing=%d",
							second, arm, power, humanoid, source, mag, cut, bound, w.landingMagnitude(0, rule, EffectScanRange, mag))
						if got := w.landingMagnitude(0, rule, EffectScanRange, mag); got != bound {
							t.Fatalf("%s: landingMagnitude %d, timed path %d", fmt.Sprint(rule.ID), got, bound)
						}
					}
				}
			}
		}
	}
}
