package sim

// CorpseState is a detached actor identity, owner and decay-stage observation.
// It excludes the large combat and source-load state carried by Entity.
type CorpseState struct {
	ID    EntityID
	Owner uint32
	Stage DecayStage
}

// AppendCorpseStates appends the current actor population in ascending ID
// order. It reuses caller-owned storage and never changes the world.
func (w *World) AppendCorpseStates(dst []CorpseState) []CorpseState {
	for i := range w.entities {
		e := &w.entities[i]
		dst = append(dst, CorpseState{ID: e.ID, Owner: e.Owner, Stage: e.Decay})
	}
	return dst
}
