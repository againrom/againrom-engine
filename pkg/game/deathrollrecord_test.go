package game

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"againrom/internal/synth"
)

// firstDeathRollRow is a first-game Units row of 55 cells: health 20, the
// type and face keys, gold columns 38..40 and the item columns 41..44 that
// the first game's data.bin also holds (an orc's 30, an ogre's 40).
func firstDeathRollRow(typeID, face, chance, least, span, item, itemMin, itemMax, mask int32) []int32 {
	p := make([]int32, 55)
	for i := range p {
		p[i] = -1
	}
	p[4], p[29], p[30] = 20, typeID, face
	p[38], p[39], p[40] = chance, least, span
	p[41], p[42], p[43], p[44] = item, itemMin, itemMax, mask
	return p
}

// firstDeathRollFront is a first-game mission 10 with four creatures: an
// orc and an ogre whose rows hold item columns, a goblin at type 64 and a
// troll with no item chance, each at its own cell.
func firstDeathRollFront(t *testing.T) *FrontEnd {
	t.Helper()
	units := []synth.ALMUnit{
		{X: 12<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x50, ClassSubID: 1, UnitID: 31},
		{X: 15<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x42, ClassSubID: 1, UnitID: 32},
		{X: 18<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x40, ClassSubID: 1, UnitID: 33},
		{X: 21<<8 | 0x80, Y: 16<<8 | 0x80, ClassID: 0x44, ClassSubID: 1, UnitID: 34},
	}
	f := poolFixtureFrontMap(t, synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: units}))
	f.Table = actorRegistryTable()
	f.Table.Units = dbCollection{{},
		{name: "Orc_Sword", params: firstDeathRollRow(0x50, 1, 80, 30, 90, 30, 60, 180, -1)},
		{name: "Ogre", params: firstDeathRollRow(0x42, 1, 90, 50, 250, 40, 100, 500, 0x21c7ffff)},
		{name: "Goblin_Pike", params: firstDeathRollRow(0x40, 1, 60, 10, 30, 0, -1, -1, -1)},
		{name: "Troll", params: firstDeathRollRow(0x44, 1, 90, 200, 300, -1, -1, -1, 20)},
	}
	f.Humans = fourBaseHumans()
	f.Table.Humans = append(fourBaseHumans(), dbEntry{name: "actor runtime", params: humansParams(50, 0, 5)})
	f.Campaign = resolved(Campaign{Main: []int{10}, Offered: []int{10}, Chapters: map[int]Chapter{10: {Mission: 10}}}, nil)
	return f
}

// firstDeathRollRecord opens the mission four times; round r kills every
// creature but map unit 30+r (round 0 spares none), ticks until every body
// has dropped, and answers the sacks the deaths left with their gold, the
// World hash and the SAV bytes' digest.
func firstDeathRollRecord(t *testing.T) string {
	t.Helper()
	var out []string
	for round := range 4 {
		f := firstDeathRollFront(t)
		if err := f.App("first death roll").OpenMission(f.MissionOpener(10)); err != nil || f.live == nil {
			t.Fatal("mission 10", err)
		}
		w := f.live.world
		for _, e := range w.Entities() {
			if e.MapUnitID >= 31 && e.MapUnitID <= 34 && int(e.MapUnitID) != 30+round {
				if err := w.HeadlessDamage(e.ID, e.HP+20); err != nil {
					t.Fatal(err)
				}
			}
		}
		for range 60 {
			f.live.tick()
		}
		for _, s := range w.Sacks() {
			out = append(out, fmt.Sprintf("round %d sack (%d,%d) gold %d items %v", round, s.X, s.Y, s.Gold, s.Items))
		}
		snapshot, label, err := f.Snapshot(true)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := f.ExportCurrentSave(snapshot, label)
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, fmt.Sprintf("round %d hash %x sav %x", round, w.Hash(), sha256.Sum256(raw)))
	}
	return strings.Join(out, "\n")
}

// firstDeathRollWant is firstDeathRollRecord at the engine before the
// second game's death item roll: the first game's gold, sacks, World and
// SAV bytes after the deaths.
const firstDeathRollWant = `round 0 sack (12,16) gold 68 items []
round 0 sack (15,16) gold 270 items []
round 0 sack (21,16) gold 434 items []
round 0 hash 74ead3e9cda5c042 sav a78bbb0707bf01fd455305ad1445c0dc50243a19789d9a5b7a0a0c635227f099
round 1 sack (15,16) gold 155 items []
round 1 sack (21,16) gold 464 items []
round 1 hash 295abe8a5dce4009 sav 935ff147863badd58014d562cb7feef40fec73cf5a38daca861f7db5cc2cd7a8
round 2 sack (12,16) gold 68 items []
round 2 sack (21,16) gold 464 items []
round 2 hash fbbe75eb1e361e3d sav 24babaf165f3972f85c2a4c464244e5a9bf5ddbc92ddc770b2c09ab666a1b943
round 3 sack (12,16) gold 68 items []
round 3 sack (15,16) gold 270 items []
round 3 sack (21,16) gold 434 items []
round 3 hash 59677c9a7bba312e sav 5ca30b1f10e399bf1ece61c80675db2d950e162d996923a6a02a770457506840`

// The first game's death roll keeps its gold, its sacks, its World hash and
// its SAV bytes: four rounds of four deaths, an orc and an ogre whose rows
// hold item columns among them, equal the record. Fails when the first
// game draws an item, an item roll, or carries the item columns.
func TestFirstGameDeathRollKeepsTheRecord(t *testing.T) {
	got := firstDeathRollRecord(t)
	if got != firstDeathRollWant {
		t.Fatalf("first-game death record:\n%s\nwant:\n%s", got, firstDeathRollWant)
	}
}
