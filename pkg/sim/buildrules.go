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
	// Path is the install's [Path Finding] route values; nil runs the shipped
	// ones (ShippedPathFinding). Like the rest it is input: it is not in the
	// world hash or the save, so a save loads under the values of the install
	// that opens it.
	Path *PathFinding
	// Scan is the install's [Scanning] value; nil runs the shipped one
	// (ShippedScanning). Input like Path.
	Scan *Scanning
}

// The bounds the loaded values keep: the sight tables hold shifts 1 to 8 (the
// seed shifts a 1/256 cell sight by 8 minus the shift).
const (
	MinSpeedMultiplier = 1
	MaxSpeedMultiplier = 255
	MinScanShift       = 1
	MaxScanShift       = 8
)

// Scanning is the [Scanning] value of the install's map.reg that the AI sight
// reads. The fog keeps the compiled shift (TERR-FOG-088).
type Scanning struct {
	// ScanShift is the fixed-point precision of the AI sight predicate.
	ScanShift int32
}

// ShippedScanning is the value both installs ship and the original's default
// when the key is missing.
func ShippedScanning() Scanning { return Scanning{ScanShift: sightShift} }

// scanShift is the AI sight shift the world runs under; a value outside the
// bounds (the loader refuses one) runs the shipped shift.
func (w *World) scanShift() int32 {
	if w.build.Scan == nil || w.build.Scan.ScanShift < MinScanShift || w.build.Scan.ScanShift > MaxScanShift {
		return sightShift
	}
	return w.build.Scan.ScanShift
}

// PathFinding is the route values of the install's map.reg section [Path
// Finding] that the walk and the search read. A key the file lacks holds its
// shipped value.
type PathFinding struct {
	// StaticScanAhead and DynamicScanAhead are the budget slacks of the far
	// and the near search.
	StaticScanAhead, DynamicScanAhead int32
	// StaticRefreshRate and DynamicRefreshRate are the counts above which a
	// stored route and a listed near step are searched again.
	StaticRefreshRate, DynamicRefreshRate int32
	// SpeedMultiplier multiplies a ground mover's speed in the rate law.
	SpeedMultiplier int32
	// DynamicByStaticLookup is the head distance past which the near search
	// aims at the head; StaticIsntNeeded the route length up to which it aims
	// at the route end.
	DynamicByStaticLookup, StaticIsntNeeded int32
}

// ShippedPathFinding is the values both installs ship and the original's
// defaults when a key is missing.
func ShippedPathFinding() PathFinding {
	return PathFinding{StaticScanAhead: 5, DynamicScanAhead: 3, StaticRefreshRate: 16, DynamicRefreshRate: 32, SpeedMultiplier: shippedSpeedMultiplier, DynamicByStaticLookup: 3, StaticIsntNeeded: 5}
}

// pathFinding is the route values the world runs under.
func (w *World) pathFinding() PathFinding {
	if w.build.Path == nil {
		return ShippedPathFinding()
	}
	return *w.build.Path
}

// SetBuildRules sets the build's rules (base.Profile, through the table).
func (w *World) SetBuildRules(r BuildRules) { w.build = r }

// BuildRules is the build's rules (SetBuildRules).
func (w *World) BuildRules() BuildRules { return w.build }
