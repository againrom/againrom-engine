package ui

import (
	"image"
	"os"
	"path/filepath"
	"sync"
)

// testTips is the first game's tip table, read from the game package's
// embedded file so the tests and the game draw from one table.
var testTips = sync.OnceValue(func() *TipTable {
	b, err := os.ReadFile(filepath.Join("..", "game", "tips", "rom1.json"))
	if err != nil {
		panic(err)
	}
	t, err := DecodeTipTable(b)
	if err != nil {
		panic(err)
	}
	return t
})

// The tests name each popup's rectangle through these values, read from the
// table's rows.
var (
	TownTipRect      = testTipRect("room-enter", "square", "")
	TavernTipRect    = testTipRect("room-enter", "tavern", "")
	SchoolTipRect    = testTipRect("room-enter", "school", "")
	shopTipRect      = testTipRect("room-enter", "shop", "")
	PreCreateTipRect = testTipRect("pre-create-enter", "", "")
	ChargenTipRect   = testTipRect("detail-enter", "", "fighter")
	MissionTipRect   = testTipRect("dialogue-close", "", "")
)

// testTipRect is the rectangle of the table's row for event; a missing row
// stops the test binary, as a table that does not decode stops the game.
func testTipRect(event, room, class string) image.Rectangle {
	r := testTips().Row(event, room, class)
	if r == nil {
		panic("the tip table has no " + event + " row for " + room + class)
	}
	return r.Rectangle()
}

// testChargenTips are the table's generator rows carrying the given texts.
func testChargenTips(selects [3]string, fighter, mage, after string) ChargenTips {
	t := testTips()
	tip := func(event, class, text string) ChargenTip {
		return NewChargenTip(t, t.Row(event, "", class), text)
	}
	out := ChargenTips{
		Fighter: tip("detail-enter", "fighter", fighter),
		Mage:    tip("detail-enter", "mage", mage),
		After:   tip("first-skill-click", "", after),
	}
	for i, event := range PreCreateTipEvents {
		out.Select = append(out.Select, tip(event, "", selects[i]))
	}
	return out
}
