package game

import "againrom/pkg/sim"

// entityRef addresses a copy so a test can ask a pointer-receiver predicate of
// an Entity a helper returned by value.
func entityRef(e sim.Entity) *sim.Entity { return &e }
