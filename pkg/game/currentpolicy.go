package game

import "againrom/pkg/sim"

// Input migration checks distinguish an explicitly saved runtime policy from
// a damaged historical native/document pair. SAVE reads World directly.
func currentNativePolicy(state *SnapshotSAVDocument, check func(*sim.CurrentWorldPolicy) bool) bool {
	if state == nil || state.Document == nil {
		return false
	}
	a, err := readCurrentActions(state.Document)
	return err == nil && a != nil && a.Policy != nil && check(a.Policy)
}
