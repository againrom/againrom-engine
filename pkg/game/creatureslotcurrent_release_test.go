package game

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/sim"
)

func TestReleaseCurrentCreatureSlotWordsFollowWorldAfterLOAD(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	for field := 0; field < 6; field++ {
		t.Run(fmt.Sprintf("word%d", field), func(t *testing.T) {
			_, raw := groundCorpusFile(t, "2026-09-24/game0002-bigsack.sav", "")
			input := t.TempDir()
			if err := os.WriteFile(filepath.Join(input, "source.sav"), raw, 0600); err != nil {
				t.Fatal(err)
			}
			f, app := castOrderSession(t, input)
			var actor sim.Entity
			for _, e := range f.live.world.Entities() {
				if e.Alive() && e.CreatureSpells[0].ID != 0 && e.Owner != sim.SelfSlot {
					actor = e
					break
				}
			}
			if actor.ID == 0 {
				t.Fatal("original input has no living creature slot holder")
			}
			loaded := append([]byte(nil), creatureWireOrder(t, raw, actor.MapUnitID)[0x78:0x90]...)
			slots := actor.CreatureSpells
			if field < 3 {
				slots[field].ID ^= 1
			} else {
				slots[field-3].Threshold ^= 1
			}
			setCurrentCreatureSlotWords(t, f.live.world, actor, slots)
			want := append([]byte(nil), loaded...)
			binary.LittleEndian.PutUint32(want[field*4:], binary.LittleEndian.Uint32(loaded[field*4:])^1)
			dir, saved := castOrderF2Save(t, f, app, "changed creature word")
			assert := func(raw []byte) {
				t.Helper()
				got := creatureWireOrder(t, raw, actor.MapUnitID)[0x78:0x90]
				if !bytes.Equal(got, want) {
					t.Fatalf("slot words %x, want %x; loaded %x", got, want, loaded)
				}
			}
			assert(saved)
			g, coldApp := castOrderSession(t, dir)
			assertCurrentWorldEqual(t, f.live.world, g.live.world, "changed original creature slots")
			_, second := castOrderF2Save(t, g, coldApp, "changed creature word again")
			assert(second)
			for tick := 0; tick < 3; tick++ {
				f.live.tick()
				g.live.tick()
				assertCurrentWorldEqual(t, f.live.world, g.live.world, "changed original creature slots next tick")
			}
		})
	}
}
