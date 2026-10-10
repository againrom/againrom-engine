package game

// The first game's tip rectangles, read from its generator description.
var (
	preCreateTipRect = generatorDescriptions["rom1"].Tips.PreCreateRect.Rectangle()
	chargenTipRect   = generatorDescriptions["rom1"].Tips.DetailRect.Rectangle()
)
