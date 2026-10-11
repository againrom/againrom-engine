package game

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/base"
)

// Every object of the second game's town descriptions that states a value
// carries a cite, every cite list holds a second-game claim, and no
// first-game claim is cited.
func TestRom2TownDescriptionEveryValueIsCited(t *testing.T) {
	for name, data := range map[string][]byte{"town 1": rom2Town1JSON, "town 2": rom2Town2JSON, "town 3": rom2Town3JSON} {
		t.Run(name, func(t *testing.T) { checkSecondTownCites(t, data) })
	}
}

func checkSecondTownCites(t *testing.T, data []byte) {
	checkDescriptionCites(t, data)
	var doc any
	if err := json.Unmarshal(data, &doc); err != nil {
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

// Town 2 decodes under the second game's vocabulary: its square lists
// TAVERN, GATES and the four persons by hotspot row, each person opens its
// stage-keyed section, its music is the track the game's music description
// names for town 2, and its inn is the same page kind as town 1's.
func TestRom2TownTwoDescriptionShape(t *testing.T) {
	d := townDescriptions[secondTownDescriptions[2]]
	if d == nil {
		t.Fatal("town 2 has no description")
	}
	var rows []string
	keys := map[string]string{}
	for _, h := range d.Hotspots {
		if h.Row != nil {
			for len(rows) <= *h.Row {
				rows = append(rows, "")
			}
			rows[*h.Row] = secondTownRowText[h.Name]
		}
		for _, op := range h.Click {
			if op.Dialogue != "" {
				keys[h.Name] = op.Dialogue + " " + op.Value
			}
		}
	}
	if strings.Join(rows, ",") != "TAVERN,GATES,GIRL 1,GIRL 2,GUARD,DERVISH" {
		t.Errorf("square rows %v", rows)
	}
	for name, want := range map[string]string{"girl1": "kaargwoman%d stage", "girl2": "kaargwoman%d stage",
		"guard": "kaargguard%d stage", "dervish": "kaargman%d stage"} {
		if keys[name] != want {
			t.Errorf("%s opens %q, want %q", name, keys[name], want)
		}
	}
	p, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	if track, ok := GameMusic(p).TownTrack(2); !ok || track != d.Music.Track {
		t.Errorf("music %q, game music names %q", d.Music.Track, track)
	}
	one := townDescriptions[secondTownDescriptions[1]].Room("inn")
	r := d.Room("inn")
	if r == nil || r.Page != one.Page || r.Seats == nil || len(r.Buttons) != len(one.Buttons) {
		t.Fatal("the inn is not town 1's inn page kind")
	}
	seats, oneSeats := *r.Seats, *one.Seats
	seats.Cite, oneSeats.Cite = nil, nil
	if !reflect.DeepEqual(seats, oneSeats) {
		t.Errorf("seats %+v, town 1 %+v", seats, oneSeats)
	}
	for i, b := range r.Buttons {
		if b.Name != one.Buttons[i].Name || b.Rect != one.Buttons[i].Rect {
			t.Errorf("button %d %s %v, town 1 %s %v", i, b.Name, b.Rect, one.Buttons[i].Name, one.Buttons[i].Rect)
		}
	}
	c := newSecondCampaign()
	c.bank[768] = 30
	if v := secondTownValues["stage"](c); v != 30 {
		t.Errorf("stage value %d, want bank slot 768", v)
	}
}

// Town 3 decodes under the second game's vocabulary: its square lists
// TAVERN, GATES and the two keepers by hotspot row, each keeper opens its
// stage-keyed section and shows no tip (R2-ENGINE-255), its music is the
// track the game's music description names for town 3, and its inn is the
// same page kind as town 1's (R2-ENGINE-258).
func TestRom2TownThreeDescriptionShape(t *testing.T) {
	d := townDescriptions[secondTownDescriptions[3]]
	if d == nil {
		t.Fatal("town 3 has no description")
	}
	var rows []string
	keys := map[string]string{}
	for _, h := range d.Hotspots {
		if h.Row != nil {
			for len(rows) <= *h.Row {
				rows = append(rows, "")
			}
			rows[*h.Row] = secondTownRowText[h.Name]
		}
		for _, op := range h.Click {
			if op.Dialogue != "" {
				keys[h.Name] = op.Dialogue + " " + op.Value
			}
		}
		_, tip := h.TipSlot()
		if keeper := strings.HasSuffix(h.Name, "keeper"); tip == keeper {
			t.Errorf("%s shows a tip %v", h.Name, tip)
		}
	}
	if strings.Join(rows, ",") != "TAVERN,GATES,INNKEEPER,SHOPKEEPER" {
		t.Errorf("square rows %v", rows)
	}
	for name, want := range map[string]string{"innkeeper": "druidinnkeeper%d stage", "shopkeeper": "druidshopkeeper%d stage"} {
		if keys[name] != want {
			t.Errorf("%s opens %q, want %q", name, keys[name], want)
		}
	}
	p, ok := base.Find(base.ROM2EN)
	if !ok {
		t.Fatal("missing ROM2 profile")
	}
	if track, ok := GameMusic(p).TownTrack(3); !ok || track != d.Music.Track {
		t.Errorf("music %q, game music names %q", d.Music.Track, track)
	}
	one := townDescriptions[secondTownDescriptions[1]].Room("inn")
	r := d.Room("inn")
	if r == nil || r.Page != one.Page || r.Seats == nil || len(r.Buttons) != len(one.Buttons) {
		t.Fatal("the inn is not town 1's inn page kind")
	}
	seats, oneSeats := *r.Seats, *one.Seats
	seats.Cite, oneSeats.Cite = nil, nil
	if !reflect.DeepEqual(seats, oneSeats) {
		t.Errorf("seats %+v, town 1 %+v", seats, oneSeats)
	}
	for i, b := range r.Buttons {
		if b.Name != one.Buttons[i].Name || b.Rect != one.Buttons[i].Rect {
			t.Errorf("button %d %s %v, town 1 %s %v", i, b.Name, b.Rect, one.Buttons[i].Name, one.Buttons[i].Rect)
		}
	}
}
