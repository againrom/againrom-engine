package ui

import (
	"encoding/json"
	"image"
	"os"
	"path/filepath"

	"againrom/pkg/town"
)

// The square's and the rooms' tip rectangles, read from the first game's
// town description: the file the game decodes for its tips. The tests read
// these values rather than a copy beside the description.
var (
	townTipRects  = descriptionTipRects()
	TownTipRect   = townTipRects["square"]
	TavernTipRect = townTipRects["tavern"]
	SchoolTipRect = townTipRects["school"]
	shopTipRect   = townTipRects["shop"]
)

// descriptionTipRects is the tip rectangle the town description gives each
// view: "square" for the town square, and each room by its name. A
// description that cannot be read, or a view without a tip, stops the test
// binary, as a description that does not decode stops the game.
func descriptionTipRects() map[string]image.Rectangle {
	data, err := os.ReadFile(filepath.Join("..", "game", "towns", "rom1.json"))
	if err != nil {
		panic(err)
	}
	var d struct {
		Tip   town.TipSpec `json:"tip"`
		Rooms []struct {
			Name string        `json:"name"`
			Tip  *town.TipSpec `json:"tip"`
		} `json:"rooms"`
	}
	if err := json.Unmarshal(data, &d); err != nil {
		panic(err)
	}
	out := map[string]image.Rectangle{"square": d.Tip.Rect.Rectangle()}
	for _, r := range d.Rooms {
		if r.Tip != nil {
			out[r.Name] = r.Tip.Rect.Rectangle()
		}
	}
	for _, view := range []string{"square", "tavern", "shop", "school"} {
		if out[view].Empty() {
			panic("town description gives the " + view + " no tip rectangle")
		}
	}
	return out
}
