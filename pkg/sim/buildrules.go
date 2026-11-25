package sim

// BuildRules are the rules one install build runs beside its game's
// profile: install-derived input, carried across a decode and never encoded.
// The zero value is a build that adds none.
type BuildRules struct {
	// HeroBodyRaise: the corpse walk sets a hero body below decayHeroHP and
	// not consumed to decayHeroHP, so the ladder never removes it
	// (R2-ENGINE-1825).
	HeroBodyRaise bool
	// DeathSackCap: a death bag merges nothing into a cell's sack holding
	// this many entries or more, and its items are lost while its gold is
	// still added; 0 is no bound (R2-ENGINE-1828).
	DeathSackCap int
}

// SetBuildRules sets the build's rules (base.Profile, through the table).
func (w *World) SetBuildRules(r BuildRules) { w.build = r }

// BuildRules is the build's rules (SetBuildRules).
func (w *World) BuildRules() BuildRules { return w.build }
