package game

import (
	"fmt"
	"image"
	"testing"

	"againrom/pkg/ui"
)

// The three squares read their own sections (R2-ENGINE-307).
func TestSecondTipTableSquares(t *testing.T) {
	table := tipTables["rom2"]
	for town, section := range map[string]string{"rom2-town1": "tips1", "rom2-town2": "tips12", "rom2-town3": "tips11"} {
		if r := table.RoomRow(town, "square"); r == nil || r.Text.Section != section || r.Rectangle() != image.Rect(328, 0, 640, 200) {
			t.Fatalf("%s square row %+v", town, r)
		}
		if r := table.RoomRow(town, "inn"); r == nil || r.Text.Section != "tips2" {
			t.Fatalf("%s inn row %+v", town, r)
		}
		if r := table.RoomRow(town, "shop"); r == nil || r.Text.Section != "tips3" {
			t.Fatalf("%s shop row %+v", town, r)
		}
	}
	if tipTables["rom1"].RoomRow("rom2-town1", "square") != tipTables["rom1"].Row("room-enter", "square", "") {
		t.Fatal("a town without rows of its own must take the room's row")
	}
}

// A town names only a room-enter row, and one town and room take one row.
func TestTipTableTownRows(t *testing.T) {
	head := `{"tips": "t", "art": "a", "rows": [`
	row := `{"name": "%s", "event": "%s", "town": "t1", "room": "%s", "text": {"file": "f"}, "rect": [0, 0, 1, 1], "once": "%s", "pointer": "%s"}`
	for _, body := range []string{
		fmt.Sprintf(row, "a", "pre-create-enter", "", "page-enter", "generator"),
		fmt.Sprintf(row, "a", "room-enter", "square", "entry", "town") + "," + fmt.Sprintf(row, "b", "room-enter", "square", "entry", "town"),
	} {
		if _, err := ui.DecodeTipTable([]byte(head + body + "]}")); err == nil {
			t.Fatalf("accepted %s", body)
		}
	}
	if _, err := ui.DecodeTipTable([]byte(head + fmt.Sprintf(row, "a", "room-enter", "square", "entry", "town") + "]}")); err != nil {
		t.Fatal(err)
	}
}
