package rules

// SpellGame selects one game's spell effect rules. The zero value is the
// first game's.
type SpellGame uint8

const (
	SpellGameFirst SpellGame = iota
	SpellGameSecond
)

// EffectMotion is how one driver update moves a spell effect.
type EffectMotion uint8

const (
	// EffectTravel steps toward the aim point: the driver's default arm.
	EffectTravel EffectMotion = iota
	// EffectFollow stands at the target's point.
	EffectFollow
	// EffectStill writes no position.
	EffectStill
	// EffectPath writes no position and draws a point list.
	EffectPath
)

// EffectFrame is which frame an update writes.
type EffectFrame uint8

const (
	// FrameHalf is (counter/2) mod Phases, and 0 without phases.
	FrameHalf EffectFrame = iota
	// FrameCounter is the counter itself.
	FrameCounter
	// FrameCounterLess1 is the counter minus one.
	FrameCounterLess1
	// FrameRamp takes RampFrame for counters 1..13 and keeps the frame
	// otherwise.
	FrameRamp
	// FrameResolvedOne is 1 on an update that resolves its target and
	// FrameHalf otherwise.
	FrameResolvedOne
)

// EffectDrive is one identifier's update arm.
type EffectDrive struct {
	Motion EffectMotion
	Frame  EffectFrame
}

// TargetMark is the mark a projectile picture's resolved actor call leaves
// on its target.
type TargetMark uint8

const (
	TargetMarkNone TargetMark = iota
	TargetMarkHeal
	TargetMarkDrainLife
)

// PictureCapability is one projectile picture's row: its target mark, its
// update arm, Unbound, that its original-SAV continuation's picture-specific
// consumers are not run (DIV-1814), SoundPhase, the action phase whose
// update plays its spell sound (0 for none), and Deformation, the background
// deformation draw.
type PictureCapability struct {
	Picture     int32
	Mark        TargetMark
	Drive       EffectDrive
	Unbound     bool
	SoundPhase  int32
	Deformation bool
}

var (
	followHalf  = EffectDrive{EffectFollow, FrameHalf}
	followFirst = EffectDrive{EffectFollow, FrameResolvedOne}
	pathRamp    = EffectDrive{EffectPath, FrameRamp}
)

// The first game's rows (ANIM-143, ANIM-149, ANIM-PHASECLOCK-028,
// MAGIC-BOLTSTILL-072, DIV-944, DIV-1814) and the second game's
// (R2-ENGINE-447, DIV-3101). A picture no row names takes the zero row.
var pictureCapabilities = [...][]PictureCapability{
	SpellGameFirst: {
		{Picture: 7, Deformation: true},
		{Picture: 13, Unbound: true},
		{Picture: 18, Drive: followHalf, Unbound: true},
		{Picture: 20, Mark: TargetMarkHeal, Drive: followFirst, Unbound: true},
		{Picture: 24, Drive: followHalf, Unbound: true},
		{Picture: 28, Drive: followHalf, Unbound: true},
		{Picture: 30, Mark: TargetMarkDrainLife, Drive: followFirst, Unbound: true},
		{Picture: 34, Drive: pathRamp},
		{Picture: 36, Drive: pathRamp, Unbound: true},
		{Picture: 40, Drive: followHalf, Unbound: true},
		{Picture: 44, Drive: followHalf, Unbound: true},
		{Picture: 48, Drive: followHalf, Unbound: true},
		{Picture: 51, Drive: EffectDrive{EffectTravel, FrameCounter}, Unbound: true, SoundPhase: 8},
		{Picture: 52, Drive: followHalf, Unbound: true},
		{Picture: 54, Drive: followHalf, Unbound: true},
		{Picture: 56, Drive: followHalf, Unbound: true},
		{Picture: 60, Drive: EffectDrive{EffectStill, FrameCounterLess1}},
		{Picture: 62, Drive: followHalf, Unbound: true},
		{Picture: 64, Drive: followHalf, Unbound: true},
	},
	SpellGameSecond: {
		{Picture: 7, Deformation: true},
		{Picture: 16, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 20, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 23, Drive: EffectDrive{EffectStill, FrameCounter}},
		{Picture: 24, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 28, Drive: pathRamp},
		{Picture: 30, Drive: pathRamp},
		{Picture: 34, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 44, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 46, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 48, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 50, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 54, Drive: EffectDrive{EffectStill, FrameCounterLess1}},
		{Picture: 56, Drive: followHalf},
		{Picture: 60, Drive: followHalf},
		{Picture: 62, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 64, Drive: EffectDrive{EffectFollow, FrameCounter}},
		{Picture: 66, Drive: EffectDrive{EffectFollow, FrameCounter}},
	},
}

// PictureCapabilityOf is projectile picture's row in game.
func PictureCapabilityOf(game SpellGame, picture int32) PictureCapability {
	if int(game) < len(pictureCapabilities) {
		for _, row := range pictureCapabilities[game] {
			if row.Picture == picture {
				return row
			}
		}
	}
	return PictureCapability{Picture: picture}
}

// EffectDriveOf is the update arm identifier id takes in game.
func EffectDriveOf(game SpellGame, id int32) EffectDrive {
	return PictureCapabilityOf(game, id).Drive
}

// rampFrames is the frame FrameRamp writes for counters 1..13.
var rampFrames = [...]int32{4, 3, 2, 1, 0, 1, 2, 1, 0, 1, 2, 3, 4}

// RampFrame is FrameRamp's frame at counter, and false outside 1..13.
func RampFrame(counter int32) (int32, bool) {
	if counter < 1 || counter > int32(len(rampFrames)) {
		return 0, false
	}
	return rampFrames[counter-1], true
}

// EffectFrameAt is the frame drive writes at counter for a sheet of phases
// frames. It answers false for FrameRamp outside its counters, where the
// frame stays.
func EffectFrameAt(frame EffectFrame, counter int32, phases uint16, resolved bool) (int32, bool) {
	switch frame {
	case FrameRamp:
		return RampFrame(counter)
	case FrameCounter:
		return counter, true
	case FrameCounterLess1:
		return counter - 1, true
	case FrameResolvedOne:
		if resolved {
			return 1, true
		}
	}
	if phases == 0 {
		return 0, true
	}
	return (counter / 2) % int32(phases), true
}
