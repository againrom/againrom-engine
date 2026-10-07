package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// A placed person who is not a Hero-mode placement keeps his table type id, so the
// client draws him by the second arm: the sex is bit 7 of his face byte and the
// face is the byte's low seven bits (UNIT-PICT-035, ALM-CLS-054, PARTY-M20-031).
// The spawner writes the placement's secondary key into that byte and bit 2 of the
// placement's flag word into its bit 7, on the type-key arm always and on the
// definition-id arm when the secondary key is not zero (ALM-FLAGPATH-109).
//
// The world figure and the dialogue candidate are one figure by construction, so
// both are read here. The rows disagree with each placement on purpose: a woman
// row placed as a man, a man row placed as a woman.
func TestAPlacedPersonsFigureFollowsTheFaceByteHisSpawnerWrites(t *testing.T) {
	const (
		manFighter   = 101
		womanFighter = 102
		womanMage    = 103
		sexFlag      = 4
	)
	row := func(typeID, face, gender, serverID int32) []int32 {
		p := figureRow(typeID, face, gender)
		p[0x18] = serverID
		return p
	}
	table := &mapload.Table{Humans: dbCollection{
		{},
		{name: "ManFighter", params: row(3, 8, 0, manFighter)},
		{name: "WomanFighter", params: row(4, 8, 1, womanFighter)},
		{name: "WomanMage", params: row(0x18, 5, 1, womanMage)},
	}}
	for _, tc := range []struct {
		name string
		unit alm.Unit
		want figureID
	}{
		{"type-key placement of a woman row with the sex flag clear is a man",
			alm.Unit{ClassID: 4, ClassSubID: 5}, figureID{Dir: data.FigureDirManFighter, Face: 5}},
		{"type-key placement of a man row with the sex flag set is a woman",
			alm.Unit{ClassID: 3, ClassSubID: 5, Flags: sexFlag}, figureID{Dir: data.FigureDirWomanFighter, Face: 5}},
		{"definition-id placement with no secondary key keeps the row's woman",
			alm.Unit{ClassID: 4, DefID: womanFighter}, figureID{Dir: data.FigureDirWomanFighter, Face: 8}},
		{"definition-id placement of a woman row with secondary 3 is a man of face 3",
			alm.Unit{ClassID: 4, ClassSubID: 3, DefID: womanFighter}, figureID{Dir: data.FigureDirManFighter, Face: 3}},
		{"definition-id placement of a woman mage with the sex flag set is a woman of face 2",
			alm.Unit{ClassID: 0x18, ClassSubID: 2, DefID: womanMage, Flags: sexFlag}, figureID{Dir: data.FigureDirWomanMage, Face: 2}},
		{"definition-id placement of a woman mage with the flag clear is a man of face 2",
			alm.Unit{ClassID: 0x18, ClassSubID: 2, DefID: womanMage}, figureID{Dir: data.FigureDirManMage, Face: 2}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := &alm.Map{Units: []alm.Unit{tc.unit}}
			got, ok := entityFigures(m, table)[sim.EntityID(0)]
			if !ok || got != tc.want {
				t.Errorf("world figure = %+v present %t, want %+v", got, ok, tc.want)
			}
			speakers := entitySpeakers(m, table)
			if len(speakers) != 1 || speakers[0].fig.Dir != tc.want.Dir || speakers[0].fig.Face != tc.want.Face ||
				int(speakers[0].face) != tc.want.Face {
				t.Errorf("dialogue candidate = %+v, want figure %s/%d and face term %d", speakers, tc.want.Dir, tc.want.Face, tc.want.Face)
			}
		})
	}
}
