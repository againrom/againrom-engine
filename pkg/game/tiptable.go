package game

import (
	_ "embed"

	"againrom/pkg/base"
	"againrom/pkg/ui"
)

// The tip tables: every tip popup a game shows, one row each, read by the
// same code for both games. No other file of the engine names a tip's text,
// rectangle, raising event or once-only rule; the archtest tip-table guard
// holds that.

//go:embed tips/rom1.json
var rom1TipsJSON []byte

//go:embed tips/rom2.json
var rom2TipsJSON []byte

// tipTables are the decoded tables by the name an edition's Tips gives. One
// that does not decode is a build defect, so it stops the process at start.
var tipTables = map[string]*ui.TipTable{
	"rom1": mustDecodeTips(rom1TipsJSON),
	"rom2": mustDecodeTips(rom2TipsJSON),
}

func mustDecodeTips(data []byte) *ui.TipTable {
	t, err := ui.DecodeTipTable(data)
	if err != nil {
		panic(err)
	}
	return t
}

// TipTable is the tip table of the edition, nil when it names none. Callers
// must not change it.
func TipTable(e base.Edition) *ui.TipTable { return tipTables[e.Tips] }

// readTipText reads one row text from the install: the whole file, or one
// section of it; empty when the install has none.
func readTipText(src entrySource, t ui.TipText, code TextCode) string {
	text, _ := ReadShopTip(src, t.File, code)
	if t.Section == "" || text == "" {
		return text
	}
	return secondGameTextSection([]byte(text), t.Section)
}
