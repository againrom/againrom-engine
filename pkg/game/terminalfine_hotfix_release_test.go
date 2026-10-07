//go:build sessioncorpusaudit

package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

func TestMilestone2PlayedTerminalFrozenFineSurvivesCurrentSAV(t *testing.T) {
	for _, tc := range []struct {
		name, hash   string
		identity     uint32
		fineX, fineY uint8
	}{
		{"2026-08-27/EXP-0261-owner-runs/game9999.sav", "d8e6f1d21a7d5f98d8e21d4514faa3741f4793cbf39eebbc57061db58fc84a74", 0x2b58590, 192, 64},
		{"2027-09-07/game0036.sav", "dd45e761d806ffe5d0ad2104d3dd21cd30f5985dcd5695ac47bdf2ce5348b530", 0x2ff6990, 160, 96},
		{"2027-09-07/game0043.sav", "f030af255c18fe6f8efd5dabd0cddf0396fbab2f8cb6ff9323a19f0c85e47aaf", 0x301e930, 112, 144},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
			if corpus == "" {
				t.Fatal("AGAINROM_SAVE_CORPUS must name owner saves")
			}
			raw, err := os.ReadFile(filepath.Join(corpus, filepath.FromSlash(tc.name)))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != tc.hash {
				t.Fatalf("owner SAV SHA256=%s, want %s", got, tc.hash)
			}
			assets := os.Getenv("AGAINROM_ASSETS")
			if assets == "" {
				t.Fatal("AGAINROM_ASSETS must name a lawful install")
			}
			c := savGateCase{population: "corpus played", name: tc.name + " played", raw: raw, ticks: 30, kill: true}
			f, _, refusal, town := savGateOpen(t, assets, "terminal fine play", c, raw)
			if refusal != "" || town {
				t.Fatalf("source mission LOAD: town=%t refusal=%s", town, refusal)
			}
			savGatePlay(t, f, c)
			world := f.live.world
			motions, _, _, present := world.SavedActorMotions()
			if !present {
				t.Fatal("played World has no saved motion carrier")
			}
			byID := map[sim.EntityID]sim.SavedActorMotion{}
			for _, motion := range motions {
				byID[motion.Entity] = motion
			}
			type frozenFine struct {
				actor    sim.EntityID
				position sim.SavedActorPosition
				body     sim.DeadActorState
				terrain  uint32
			}
			byIdentity := map[uint32]frozenFine{}
			byActor := map[sim.EntityID]frozenFine{}
			for _, dead := range world.OriginalDeadActors() {
				motion, present := byID[dead.ID]
				_, held := world.Entity(dead.ID)
				if !present || held || dead.Current.Stage != 5 || motion.Issue == "" ||
					motion.Position.FineX == dead.Current.FineX && motion.Position.FineY == dead.Current.FineY {
					continue
				}
				want := frozenFine{dead.ID, motion.Position, dead.Current, dead.Source.TerrainKey}
				byIdentity[dead.Source.Identity] = want
				byActor[dead.ID] = want
				t.Logf("bound terminal actor=%d identity=%#x frozen fine=(%d,%d) body fine=(%d,%d)",
					dead.ID, dead.Source.Identity, motion.Position.FineX, motion.Position.FineY, dead.Current.FineX, dead.Current.FineY)
			}
			if len(byIdentity) == 0 {
				t.Fatal("played route has no bound terminal actor with distinct frozen fine")
			}
			if want, ok := byIdentity[tc.identity]; len(byIdentity) != 1 || !ok || want.actor != 0 ||
				want.position.FineX != tc.fineX || want.position.FineY != tc.fineY ||
				want.body.FineX != 128 || want.body.FineY != 128 {
				t.Fatalf("played source bound terminal identity %#x: %+v", tc.identity, byIdentity)
			}
			written, refusal := savGateSave(t, f, true)
			if refusal != "" {
				t.Fatal(refusal)
			}
			doc, err := sav.DecodeDocumentData(written)
			if err != nil {
				t.Fatal(err)
			}
			actions, err := readCurrentActions(&doc)
			if err != nil || actions == nil {
				t.Fatal("SAV has no current actions", err)
			}
			fineRows := map[sim.EntityID]bool{}
			for _, row := range actions.TerminalMotions {
				want, relevant := byActor[row.Entity]
				if !relevant {
					if row.FrozenFine != nil {
						t.Fatalf("unaffected terminal actor %d gained frozen fine", row.Entity)
					}
					continue
				}
				if row.Detached != nil || row.FrozenFine == nil ||
					row.FrozenFine.X != want.position.FineX || row.FrozenFine.Y != want.position.FineY {
					t.Fatalf("actor %d frozen fine extension = %+v, want (%d,%d)",
						row.Entity, row.FrozenFine, want.position.FineX, want.position.FineY)
				}
				fineRows[row.Entity] = true
			}
			if len(fineRows) != len(byActor) {
				t.Fatalf("SAV contains %d frozen fine rows, want %d", len(fineRows), len(byActor))
			}
			for _, root := range doc.DeadActors {
				if root == 0 {
					continue
				}
				record := &doc.Objects[root-1]
				identity, _ := savedStructureValue(record, "Identity")
				want, relevant := byIdentity[identity]
				if !relevant {
					continue
				}
				position, err := savedMotionRaw(record, "Block12", 12)
				if err != nil {
					t.Fatal(err)
				}
				if cell := binary.LittleEndian.Uint16(position); cell != want.body.Cell ||
					binary.LittleEndian.Uint16(position[2:]) != want.body.Cell ||
					position[4] != want.body.FineX || position[5] != want.body.FineY ||
					binary.LittleEndian.Uint16(position[6:]) != want.position.Residue ||
					binary.LittleEndian.Uint32(position[8:]) != want.terrain {
					t.Fatalf("actor %d ordinary body position %x, want current cell %#04x fine (%d,%d)",
						want.actor, position, want.body.Cell, want.body.FineX, want.body.FineY)
				}
				delete(byIdentity, identity)
			}
			if len(byIdentity) != 0 {
				t.Fatalf("SAV omitted %d bound terminal roots", len(byIdentity))
			}
			cold, _, refusal, town := savGateOpen(t, assets, "terminal fine reload", c, written)
			if refusal != "" || town {
				t.Fatalf("cold mission LOAD: town=%t refusal=%s", town, refusal)
			}
			loadedMotions, _, _, _ := cold.live.world.SavedActorMotions()
			for id, want := range byActor {
				found := false
				for _, motion := range loadedMotions {
					if motion.Entity == id {
						if motion.Position != want.position {
							t.Fatalf("actor %d loaded frozen Position %+v, want %+v", id, motion.Position, want.position)
						}
						found = true
					}
				}
				if !found {
					t.Fatalf("cold LOAD omitted terminal motion %d", id)
				}
			}
			if world.Hash() != cold.live.world.Hash() {
				t.Fatalf("GAME/SAV/LOAD World hash %016x -> %016x", world.Hash(), cold.live.world.Hash())
			}
			f.LiveAdvance(1)
			cold.LiveAdvance(1)
			if world.Hash() != cold.live.world.Hash() {
				t.Fatalf("post LOAD tick World hash %016x -> %016x", world.Hash(), cold.live.world.Hash())
			}
		})
	}
}
