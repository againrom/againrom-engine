package game

import (
	"fmt"
	"image"

	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

const (
	tavernCandleLoaded   = 10
	tavernCandleLoop     = 9
	tavernCauldronLoaded = 21
	tavernCauldronLoop   = 20
	tavernBreathLoaded   = 24
	tavernDrinkLoaded    = 40
)

type tavernInteriorFamilySpec struct {
	name          string
	pattern       string
	first, loaded int
}

var tavernInteriorFamilySpecs = [...]tavernInteriorFamilySpec{
	{name: "candle", pattern: "candle/t%04d.bmp", loaded: tavernCandleLoaded},
	{name: "cauldron", pattern: "cauldron/t%04d.bmp", loaded: tavernCauldronLoaded},
	{name: "breath", pattern: "tender/breath/br%04d.bmp", first: 1, loaded: tavernBreathLoaded},
	{name: "drink", pattern: "tender/drink/dr%04d.bmp", first: 1, loaded: tavernDrinkLoaded},
}

// loadTavernInteriorArt loads each accepted family atomically and the four
// families independently. A truncated candle set, for example, contributes
// no Candle frames but cannot remove the cauldron, tender, static room or
// controls. Candle and cauldron use a black key. Tender frames include the
// room background and must replace the static tender, including its shadows.
func loadTavernInteriorArt(src terrain.EntrySource) (ui.TavernInteriorArt, []error) {
	var out ui.TavernInteriorArt
	var problems []error
	for _, spec := range tavernInteriorFamilySpecs {
		frames, err := loadTavernInteriorFamily(src, spec)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		switch spec.name {
		case "candle":
			out.Candle = frames
		case "cauldron":
			out.Cauldron = frames
		case "breath":
			out.Breath = frames
		case "drink":
			out.Drink = frames
		}
	}
	return out, problems
}

func loadTavernInteriorFamily(src terrain.EntrySource, spec tavernInteriorFamilySpec) ([]image.Image, error) {
	frames := make([]image.Image, 0, spec.loaded)
	for i := 0; i < spec.loaded; i++ {
		path := townTavernArtPrefix + fmt.Sprintf(spec.pattern, spec.first+i)
		pic, err := readChargenBMP(src, path)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", spec.name, err)
		}
		b := pic.Bounds()
		if b.Dx() <= 0 || b.Dy() <= 0 || b.Dx() > 640 || b.Dy() > 480 {
			return nil, fmt.Errorf("%s: %s: invalid bounds %v", spec.name, path, b)
		}
		if spec.name == "candle" || spec.name == "cauldron" {
			frames = append(frames, keyBlack(pic))
		} else {
			frames = append(frames, pic)
		}
	}
	return frames, nil
}
