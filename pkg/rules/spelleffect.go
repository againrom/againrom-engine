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

type effectDriveRow struct {
	ids   []int32
	drive EffectDrive
}

// The first game's arms (ANIM-143, ANIM-149, ANIM-PHASECLOCK-028,
// MAGIC-BOLTSTILL-072, DIV-944) and the second game's (R2-ENGINE-447,
// DIV-3101). An identifier no row names takes the zero EffectDrive.
var effectDrives = [...][]effectDriveRow{
	SpellGameFirst: {
		{[]int32{34, 36}, EffectDrive{EffectPath, FrameRamp}},
		{[]int32{18, 24, 28, 40, 44, 48, 52, 54, 56, 62, 64}, EffectDrive{EffectFollow, FrameHalf}},
		{[]int32{20, 30}, EffectDrive{EffectFollow, FrameResolvedOne}},
		{[]int32{60}, EffectDrive{EffectStill, FrameCounterLess1}},
		{[]int32{51}, EffectDrive{EffectTravel, FrameCounter}},
	},
	SpellGameSecond: {
		{[]int32{28, 30}, EffectDrive{EffectPath, FrameRamp}},
		{[]int32{16, 20, 24, 34, 44, 46, 48, 50, 62, 64, 66}, EffectDrive{EffectFollow, FrameCounter}},
		{[]int32{56, 60}, EffectDrive{EffectFollow, FrameHalf}},
		{[]int32{54}, EffectDrive{EffectStill, FrameCounterLess1}},
		{[]int32{23}, EffectDrive{EffectStill, FrameCounter}},
	},
}

// EffectDriveOf is the update arm identifier id takes in game.
func EffectDriveOf(game SpellGame, id int32) EffectDrive {
	if int(game) >= len(effectDrives) {
		return EffectDrive{}
	}
	for _, row := range effectDrives[game] {
		for _, v := range row.ids {
			if v == id {
				return row.drive
			}
		}
	}
	return EffectDrive{}
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
