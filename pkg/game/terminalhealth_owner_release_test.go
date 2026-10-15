package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/pkg/formats/sav"
)

type terminalHealthOwnerActor struct {
	class    string
	mapID    uint16
	identity uint32
}

type terminalHealthOwnerWords struct {
	terminalHealthWords
	identity uint32
}

func terminalHealthOwnerWire(t *testing.T, raw []byte) map[terminalHealthOwnerActor]terminalHealthOwnerWords {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	words, err := unitScalarExpected(file, raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[terminalHealthOwnerActor]terminalHealthOwnerWords{}
	seenMap := map[uint16]bool{}
	for _, actor := range words.records {
		mapID, identity := uint16(actor.values["T08"]), actor.values["Identity"]
		key := terminalHealthOwnerActor{class: actor.class, mapID: mapID}
		if mapID == 0 {
			if actor.class == "Unit" || identity == 0 {
				t.Fatal("owner source has no nonzero authored Unit ID", actor.class, identity)
			}
			key.identity = identity
		} else {
			if seenMap[mapID] {
				t.Fatal("owner source repeats an authored actor ID", mapID)
			}
			seenMap[mapID] = true
		}
		if _, duplicate := out[key]; duplicate {
			t.Fatal("owner source repeats an actor key", key)
		}
		out[key] = terminalHealthOwnerWords{terminalHealthWords{int16(actor.values["Health"]), uint16(actor.values["HealthMax"]), uint8(actor.values["Stage"])}, identity}
	}
	return out
}

func TestReleaseTerminalHealthMaximumOwnerSAV(t *testing.T) {
	dir := os.Getenv("AGAINROM_DEAD_HEALTH_OWNER_DIR")
	if dir == "" {
		t.Skip("set AGAINROM_DEAD_HEALTH_OWNER_DIR to the two owner health sources")
	}
	for _, input := range []struct {
		name, sha      string
		stage4, stage5 int
	}{
		{"game0031-original-resave.sav", "0716d61448a92ee517b043a52501f9e774a7ec41c0b534cd1a81af5d89e71cba", 0, 24},
		{"againrom-quick-save-3.sav", "c5bea6eca0fc62d799657441ebe977858200eff179c2eca5f370ce61f8fb63c5", 24, 0},
	} {
		t.Run(input.name, func(t *testing.T) {
			f := releaseFront(t)
			raw, err := os.ReadFile(filepath.Join(dir, input.name))
			if err != nil {
				t.Fatal(err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != input.sha {
				t.Fatal("owner source SHA differs", got)
			}
			want := terminalHealthOwnerWire(t, raw)
			live, stage4, stage5, zeroMap := 0, 0, 0, 0
			for key, actor := range want {
				if key.mapID == 0 {
					zeroMap++
				}
				switch {
				case actor.stage == 4:
					stage4++
				case actor.stage == 5:
					stage5++
				case actor.health > 0 && actor.stage <= 1:
					live++
				}
			}
			if len(want) != 48 || live != 24 || stage4 != input.stage4 || stage5 != input.stage5 {
				t.Fatal("owner source actor population changed", len(want), live, stage4, stage5)
			}
			app, _ := openOriginalSAVApp(t, f, raw, input.name)
			for cycle := range 2 {
				hash, tick := f.live.world.Hash(), f.live.world.Tick()
				terminal := f.live.world.CurrentTerminalActors()
				_, _, written := menuSAVE(t, f, app, OriginalStore{})
				if f.live.world.Hash() != hash || f.live.world.Tick() != tick {
					t.Fatal("ordinary SAVE changed the frozen live World")
				}
				got := terminalHealthOwnerWire(t, written)
				if len(got) != len(want) {
					t.Fatalf("cycle %d ordinary actor population %d, source %d", cycle, len(got), len(want))
				}
				identities := 0
				for key, expected := range want {
					actual, present := got[key]
					if !present {
						t.Errorf("cycle %d source actor %+v disappeared", cycle, key)
						continue
					}
					if actual.terminalHealthWords != expected.terminalHealthWords {
						t.Errorf("cycle %d actor %+v Health/HealthMax/Stage=%+v, source %+v", cycle, key, actual.terminalHealthWords, expected.terminalHealthWords)
					}
					if key.mapID != 0 && actual.identity != expected.identity {
						identities++
					}
				}
				cold := releaseFront(t)
				coldApp, _ := openOriginalSAVApp(t, cold, written, "owner-health-current.sav")
				if !reflect.DeepEqual(terminal, cold.live.world.CurrentTerminalActors()) {
					t.Fatalf("cycle %d cold LOAD changed terminal World IDs, health, maxima or placement", cycle)
				}
				// Native wire identities change; authored map IDs bind actors.
				t.Logf("cycle %d authored actor identities reallocated=%d; ordinary Health/HealthMax/Stage and terminal World rows retained", cycle, identities)
				f, app = cold, coldApp
			}
			t.Logf("source=%s sha=%s raw actor records=%d live=%d stage4=%d stage5=%d zero-map non-Unit=%d; two ordinary menu SAVE/cold App LOAD cycles", input.name, input.sha, len(want), live, stage4, stage5, zeroMap)
		})
	}
}
