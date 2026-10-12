package game

import "againrom/pkg/ui"

// The first game's tip rectangles, read from its tip table.
var (
	preCreateTipRect = rom1TipRow("pre-create-enter", "", "").Rectangle()
	chargenTipRect   = rom1TipRow("detail-enter", "", "fighter").Rectangle()
	missionTipRect   = rom1TipRow("dialogue-close", "", "").Rectangle()
)

// rom1TipRow is the first game's row for event; a missing row stops the test
// binary, as a table that does not decode stops the game.
func rom1TipRow(event, room, class string) *ui.TipRow {
	r := tipTables["rom1"].Row(event, room, class)
	if r == nil {
		panic("the first game's tip table has no " + event + " row for " + room + class)
	}
	return r
}

// rom1RoomTip is the first game's tip row of room.
func rom1RoomTip(room townRoom) *ui.TipRow { return rom1TipRow("room-enter", townRoomName(room), "") }
