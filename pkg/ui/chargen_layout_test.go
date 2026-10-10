package ui

import (
	"image"
	"os"
	"path/filepath"
	"sync"
)

// testGenerator is the first game's generator description, read from the
// game package's embedded file so the page tests and the game compose one
// layout.
var testGenerator = sync.OnceValue(func() *GeneratorDescription {
	b, err := os.ReadFile(filepath.Join("..", "game", "generators", "rom1.json"))
	if err != nil {
		panic(err)
	}
	d, err := DecodeGenerator(b)
	if err != nil {
		panic(err)
	}
	return d
})

// The page tests name the layout's controls through these values, read from
// the description.
var (
	testLayout               = testGenerator()
	preChoiceOrigin          = heroOrigins(testLayout)
	preLevelOrigin           = levelOrigins(testLayout)
	preMaskCode              = heroMasks(testLayout)
	preBackMaskCode          = uint8(testLayout.PreCreate.Back.Mask)
	preForwardMaskCode       = uint8(testLayout.PreCreate.Forward.Mask)
	preForwardOrigin         = testLayout.PreCreate.Forward.Art.At.Pt()
	detailedSkillOrigin      = skillOrigins(testLayout)
	detailedMaskCode         = skillMasks(testLayout)
	chargenColumnDestination = testLayout.Detail.ColumnRect.Rectangle()
	chargenColumnOffset      = chargenColumnDestination.Min
	chargenCardBox           = testLayout.Detail.Card.Rect.Rectangle()
	chargenNavBox            = testLayout.Detail.Nav.Rect.Rectangle()
	chargenDollBox           = testLayout.Detail.Doll.Rect.Rectangle()
	chargenMessageRect       = testLayout.Detail.Message.Rectangle()
	chargenStatValueBox      = rects(testLayout.Detail.Stats.Value)
	chargenStatPlusBox       = rects(testLayout.Detail.Stats.Plus)
	chargenStatMinusBox      = rects(testLayout.Detail.Stats.Minus)
	chargenRemainingBox      = testLayout.Detail.Stats.Pool.Rectangle()
	chargenDoubleClickWindow = ms(testLayout.PreCreate.Keys.DoubleClickMS)
	PreCreateTipRect         = testLayout.Tips.PreCreateRect.Rectangle()
	ChargenTipRect           = testLayout.Tips.DetailRect.Rectangle()
)

func preCreateNameTextOrigin() image.Point { return testLayout.PreCreate.Name.TextAt.Pt() }

func heroOrigins(l *GeneratorDescription) [4]image.Point {
	var out [4]image.Point
	for i, h := range l.PreCreate.Heroes {
		out[i] = h.Art[0].At.Pt()
	}
	return out
}

func levelOrigins(l *GeneratorDescription) [3]image.Point {
	var out [3]image.Point
	for i, v := range l.PreCreate.Levels {
		out[i] = v.Art[0].At.Pt()
	}
	return out
}

func heroMasks(l *GeneratorDescription) [4]uint8 {
	var out [4]uint8
	for i, h := range l.PreCreate.Heroes {
		out[i] = uint8(h.Mask)
	}
	return out
}

func skillOrigins(l *GeneratorDescription) [2][5]image.Point {
	var out [2][5]image.Point
	for c, class := range l.Detail.Classes {
		for i, s := range class.Skills {
			out[c][i] = s.At.Pt()
		}
	}
	return out
}

func skillMasks(l *GeneratorDescription) [2][5]uint8 {
	var out [2][5]uint8
	for c, class := range l.Detail.Classes {
		for i, s := range class.Skills {
			out[c][i] = uint8(s.Mask)
		}
	}
	return out
}

func rects(in []GeneratorRect) [4]image.Rectangle {
	var out [4]image.Rectangle
	for i, r := range in {
		out[i] = r.Rectangle()
	}
	return out
}
