package game

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// groupCarrierWitness is what the Group carrier SAVE witness writes beside
// the SAV it made, when a witness directory is named.
type groupCarrierWitness struct {
	Install                   string
	Mission                   int
	Tick                      uint64
	Groups, Orders, Moved     int
	LiveHash, ColdHash        string
	RegistryBytes             int
	SuccessorTicks            int
	LossHash                  string
	LossOwnerKeys, LossGroups int
	ProjectedRuns             int
	Constructed               bool
}

// TestReleaseGroupCarrierSurvivesSAVEAndColdLOAD starts the Kadagan Post
// mission on a lawful install and orders the hero to attack the garrison's
// gate warrior until two members of the warrior's Group have moved, then SAVEs the
// mission as SAV and LOADs it into a fresh front end. The restored World hash
// and Group registry equal the live ones, and the next 60 ticks agree. The
// loss control drops the supplement's record that the World named its Group
// owners by slot alone: the restored registry then holds the writer's Player
// keys and the World hash differs.
func TestReleaseGroupCarrierSurvivesSAVEAndColdLOAD(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	const mission, anchor = 41, sim.EntityID(9)
	f := releaseFront(t)
	f.SetTipsOff(true)
	a := f.App("group carrier SAVE")
	a.SetCutscenes(nil)
	if err := a.OpenMission(f.MissionOpenerWith(mission, f.NextParty())); err != nil {
		t.Fatal(err)
	}
	w := f.live.world
	if groups, _, present := w.SavedGroups(); !present || len(groups) == 0 {
		t.Fatal("the mission World holds no Group registry")
	}
	first, ok := w.Entity(anchor)
	if !ok {
		t.Fatalf("this install's mission %d holds no entity %d, the warrior at the gate", mission, anchor)
	}
	var members []sim.EntityID
	start := map[sim.EntityID][2]int32{}
	for _, e := range w.Entities() {
		if e.Owner == first.Owner && e.Group == first.Group {
			members = append(members, e.ID)
			start[e.ID] = [2]int32{e.X, e.Y}
		}
	}
	hero := f.live.mission.state.Start.IDs[0]
	f.live.pending = []sim.Command{sim.Attack(hero, anchor)}
	moved := 0
	for i := 0; i < 4000 && moved < 2; i++ {
		if h, _ := w.Entity(hero); h.HP > 0 && h.HP < h.MaxHP/2 {
			if err := w.HeadlessHeal(hero); err != nil {
				t.Fatal(err)
			}
		}
		f.live.tick()
		w = f.live.world
		moved = 0
		for _, id := range members {
			if e, held := w.Entity(id); held && [2]int32{e.X, e.Y} != start[id] {
				moved++
			}
		}
	}
	if moved == 0 {
		t.Fatal("no member of the attacked Group moved")
	}
	s, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, "Group carrier")
	if err != nil {
		t.Fatal(err)
	}
	cold := groupCarrierColdLoad(t, raw)
	live := f.live.world
	groups, orders, _ := live.SavedGroups()
	coldGroups, coldOrders, present := cold.live.world.SavedGroups()
	if !present || !reflect.DeepEqual(groups, coldGroups) || !reflect.DeepEqual(orders, coldOrders) {
		t.Fatal("the restored Group registry differs from the live one")
	}
	liveBytes, err := live.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	coldBytes, err := cold.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(liveBytes, coldBytes) {
		assertCurrentWorldEqual(t, live, cold.live.world, "Group carrier cold LOAD")
		t.Fatal("the restored World bytes differ from the live ones")
	}
	witness := groupCarrierWitness{Install: filepath.Base(os.Getenv("AGAINROM_ASSETS")), Mission: mission, Tick: live.Tick(),
		Groups: len(groups), Orders: len(orders), Moved: moved, LiveHash: fmt.Sprintf("%016x", live.Hash()), ColdHash: fmt.Sprintf("%016x", cold.live.world.Hash())}
	if r, err := json.Marshal(groups); err == nil {
		witness.RegistryBytes = len(r)
	}
	for i := 0; i < 60; i++ {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("successor tick %d differs after the cold LOAD", i+1)
		}
	}
	witness.SuccessorTicks = 60

	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil || actions == nil {
		t.Fatal("the SAV holds no current actions", err)
	}
	witness.Constructed = actions.GroupsConstructed
	for _, row := range actions.Actions.Actors {
		if row.Order != nil {
			witness.ProjectedRuns += len(row.Order.Projected)
		}
	}
	for i := range actions.Groups {
		if actions.Groups[i].KeylessOwner != 0 {
			actions.Groups[i].KeylessOwner = 0
			witness.LossGroups++
		}
	}
	if witness.LossGroups == 0 {
		t.Fatal("no Group owner was held by slot alone")
	}
	payload, err := json.Marshal(actions)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, payload); err != nil {
		t.Fatal(err)
	}
	lossRaw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	loss := groupCarrierColdLoad(t, lossRaw)
	lossGroups, _, _ := loss.live.world.SavedGroups()
	for _, g := range lossGroups {
		if g.Owner.Class == 1 && g.Owner.Key != 0 {
			witness.LossOwnerKeys++
		}
	}
	witness.LossHash = fmt.Sprintf("%016x", loss.live.world.Hash())
	if witness.LossOwnerKeys == 0 || witness.LossHash == witness.ColdHash {
		t.Fatal("the loss control restored the live registry without its keyless owner record")
	}
	if dir := os.Getenv("AGAINROM_GROUP_CARRIER_WITNESS_DIR"); dir != "" {
		dir = filepath.Join(dir, witness.Install)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		report, _ := json.MarshalIndent(witness, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "witness.json"), report, 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "group-carrier.sav"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Logf("%+v", witness)
}

// groupCarrierColdLoad LOADs a SAV into a fresh front end and opens its
// mission.
func groupCarrierColdLoad(t *testing.T, raw []byte) *FrontEnd {
	t.Helper()
	cold := releaseFront(t)
	cold.SetTipsOff(true)
	open, town, err := cold.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatal("cold LOAD", town, err)
	}
	a := cold.App("group carrier cold LOAD")
	a.SetCutscenes(nil)
	if err := a.OpenMission(open); err != nil {
		t.Fatal(err)
	}
	return cold
}
