package game

import (
	"encoding/json"
	"strings"
	"testing"

	"againrom/pkg/base"
)

// Every object of the second game's town 1 description that states a value
// carries a cite, every cite list holds a second-game claim, and no
// first-game claim is cited.
func TestRom2TownDescriptionEveryValueIsCited(t *testing.T) {
	checkDescriptionCites(t, rom2Town1JSON)
	var doc any
	if err := json.Unmarshal(rom2Town1JSON, &doc); err != nil {
		t.Fatal(err)
	}
	lists := 0
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case []any:
			for _, e := range x {
				walk(path, e)
			}
		case map[string]any:
			for k, e := range x {
				walk(path+"."+k, e)
			}
			list, ok := x["cite"].([]any)
			if !ok {
				return
			}
			lists++
			second := false
			for _, c := range list {
				id, _ := c.(string)
				switch {
				case strings.HasPrefix(id, "R2-"):
					second = true
				case strings.HasPrefix(id, "DIV-"):
				default:
					t.Errorf("%s cites %q, not a second-game claim", path, id)
				}
			}
			if !second {
				t.Errorf("%s cites no second-game claim", path)
			}
		}
	}
	walk("$", doc)
	if lists == 0 {
		t.Fatal("no cite list walked")
	}
	t.Logf("%d cite lists", lists)
}

// The description decodes under the second game's vocabulary, its square
// lists TAVERN then GATES by hotspot row, and its music is the track the
// game's music description names for town 1.
func TestRom2TownDescriptionShape(t *testing.T) {
	d := townDescriptions[secondTownDescriptions[1]]
	if d == nil {
		t.Fatal("town 1 has no description")
	}
	var rows []string
	for _, h := range d.Hotspots {
		if h.Row != nil {
			for len(rows) <= *h.Row {
				rows = append(rows, "")
			}
			rows[*h.Row] = secondTownRowText[h.Name]
		}
	}
	if strings.Join(rows, ",") != "TAVERN,GATES" {
		t.Errorf("square rows %v", rows)
	}
	p, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	if track, ok := GameMusic(p).TownTrack(1); !ok || track != d.Music.Track {
		t.Errorf("music %q, game music names %q", d.Music.Track, track)
	}
	for name := range secondTownRooms {
		if name != d.Square.Name && d.Room(name) == nil {
			t.Errorf("campaign room %q is not in the description", name)
		}
	}
	if r := d.Room("inn"); r == nil || r.Page != "inn" || r.Seats == nil || r.Seats.Count() != 18 || len(r.Buttons) != 3 {
		t.Fatal("the inn is not an inn page with 18 seats and 3 buttons")
	}
	if RoomDescription(p) != nil {
		t.Error("the second game reads a room description")
	}
}
