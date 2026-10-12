package sim

// BlockedBy reports whether a mover of domain d cannot enter a cell whose
// derived passability byte is plane: the same mask the world applies.
func (d Domain) BlockedBy(plane byte) bool { return plane&d.blocks() != 0 }
