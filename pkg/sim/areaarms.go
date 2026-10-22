package sim

// areaArm is one area arm's properties; every site that asks for one reads
// this table, a row id converted with spellArm first. An absent arm has none.
type areaArm struct {
	arm uint16
	// layer is the arm's slot among the cell record's six spell-effect
	// pointers (MAGIC-MAPLAYER-040, CLOUDOWNER-155), or noAreaLayer.
	layer int8
	// blocks: an occupied slot of this layer blocks its cell (TERR-STRUCT-078).
	blocks bool
	// clears: layer arms whose cells a landing takes; yieldsTo: one it skips.
	clears   []uint16
	yieldsTo uint16
	// stages and stage: the staged program's count and offsets (MAGIC-RING-048).
	stages int
	stage  func(w *World, e cellEffect, stage int) [][2]int32
	// draws: the program samples its cells from the world's stream, so the
	// painted cells cannot be produced again and a SAVE carries them.
	draws bool
}

const noAreaLayer int8 = -1

var areaArms = [...]areaArm{
	{arm: 2, layer: noAreaLayer, clears: []uint16{7, 8}},
	{arm: 3, layer: 0, clears: []uint16{3, 7, 8}},
	{arm: 4, layer: noAreaLayer, stages: 2, stage: fireSacrificeStage},
	{arm: 7, layer: 1, clears: []uint16{7, 3}},
	{arm: 8, layer: 2, clears: []uint16{8}, yieldsTo: 3},
	{arm: 9, layer: noAreaLayer, stages: 6, stage: acidStreamStage},
	{arm: 12, layer: 4, clears: []uint16{12, 17}},
	{arm: 17, layer: 5, clears: []uint16{12, 17}},
	{arm: 19, layer: 3, blocks: true, clears: []uint16{19}},
	{arm: 21, layer: noAreaLayer, stages: 32, stage: meteorStormStage, draws: true},
}

// areaLayerSpells is the arm of each cell record layer slot.
var areaLayerSpells = func() (out [6]uint16) {
	for _, a := range areaArms {
		if a.layer != noAreaLayer {
			out[a.layer] = a.arm
		}
	}
	return out
}()

func areaArmOf(arm uint16) (areaArm, bool) {
	for _, a := range areaArms {
		if a.arm == arm {
			return a, true
		}
	}
	return areaArm{layer: noAreaLayer}, false
}

// AreaArmLayer is the cell record slot the arm's cloud takes, if it takes one.
func AreaArmLayer(arm uint16) (uint8, bool) {
	a, _ := areaArmOf(arm)
	return uint8(a.layer), a.layer != noAreaLayer
}

// AreaArmStaged reports an arm that runs a staged program.
func AreaArmStaged(arm uint16) bool { return ringStageCount(arm) != 0 }

func layerSpell(arm uint16) bool {
	a, _ := areaArmOf(arm)
	return a.layer != noAreaLayer
}

func areaArmBlocks(arm uint16) bool {
	a, _ := areaArmOf(arm)
	return a.blocks
}

// areaArmClearsLayers reports an arm whose landing clears some layer.
func areaArmClearsLayers(arm uint16) bool {
	a, _ := areaArmOf(arm)
	return len(a.clears) != 0
}

// areaLayerConflict: a landing of incoming takes cells from layer arm old.
func areaLayerConflict(incoming, old uint16) bool {
	a, _ := areaArmOf(incoming)
	for _, arm := range a.clears {
		if arm == old {
			return true
		}
	}
	return false
}

// areaArmYieldsTo is the layer arm whose cells a landing of arm skips.
func areaArmYieldsTo(arm uint16) (uint16, bool) {
	a, _ := areaArmOf(arm)
	return a.yieldsTo, a.yieldsTo != 0
}

// areaBlockingLayer is the one layer slot whose occupancy blocks a cell.
func areaBlockingLayer() (layer int, arm uint16) {
	for _, a := range areaArms {
		if a.blocks {
			return int(a.layer), a.arm
		}
	}
	return -1, 0
}

// ringStageDraws reports a staged program that draws its cells.
func ringStageDraws(arm uint16) bool {
	a, _ := areaArmOf(arm)
	return a.draws
}

func ringStageCount(arm uint16) int {
	a, _ := areaArmOf(arm)
	return a.stages
}

// Fire Sacrifice: two fixed shells; orientation is ignored.
func fireSacrificeStage(_ *World, _ cellEffect, stage int) [][2]int32 {
	switch stage {
	case 0:
		return [][2]int32{{-1, 1}, {-1, 0}, {-1, -1}, {0, 1}, {0, -1}, {1, 1}, {1, 0}, {1, -1}}
	case 1:
		return [][2]int32{{-2, 1}, {-2, 0}, {-2, -1}, {-1, 2}, {0, 2}, {1, 2}, {-1, -2}, {0, -2}, {1, -2}, {2, 1}, {2, 0}, {2, -1}}
	}
	return nil
}

// Acid Stream: alternating straight and diagonal wedges.
func acidStreamStage(_ *World, e cellEffect, stage int) [][2]int32 {
	var offsets [][2]int32
	if stage < 0 || stage >= 6 {
		return nil
	}
	if e.Direction&1 == 0 {
		if stage < 5 {
			for x := -stage; x <= stage; x++ {
				offsets = append(offsets, [2]int32{int32(x), int32(stage)})
			}
		}
	} else {
		for i := 0; i <= stage; i++ {
			offsets = append(offsets, [2]int32{int32(stage - i), int32(i)})
		}
	}
	for i := range offsets {
		dx, dy := offsets[i][0], offsets[i][1]
		switch e.Direction & 7 {
		case 0:
			offsets[i] = [2]int32{dx, -dy}
		case 1:
			offsets[i] = [2]int32{dx, -dy}
		case 2:
			offsets[i] = [2]int32{dy, dx}
		case 3:
			offsets[i] = [2]int32{dx, dy}
		case 4:
			offsets[i] = [2]int32{-dx, dy}
		case 5:
			offsets[i] = [2]int32{-dx, dy}
		case 6:
			offsets[i] = [2]int32{-dy, dx}
		case 7:
			offsets[i] = [2]int32{-dx, -dy}
		}
	}
	return offsets
}

const meteorStormRadius = 2
const meteorStormSpan = 5

func meteorStormFootprint() [][2]int32 {
	var offsets [][2]int32
	for x := int32(-meteorStormRadius); x <= meteorStormSpan-meteorStormRadius; x++ {
		for y := int32(-meteorStormRadius); y <= meteorStormSpan-meteorStormRadius; y++ {
			offsets = append(offsets, [2]int32{x, y})
		}
	}
	return offsets
}

// Meteor Storm: one independently sampled cell per stage.
func meteorStormStage(w *World, _ cellEffect, stage int) [][2]int32 {
	if stage >= 0 && stage < 32 {
		return [][2]int32{{w.rng.uniform(meteorStormSpan) - meteorStormRadius, w.rng.uniform(meteorStormSpan) - meteorStormRadius}}
	}
	return nil
}
