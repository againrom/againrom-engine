package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

type currentItemValue struct {
	Code, Kind uint16
	Count      uint32
	Price      int32
	Weight     int16
	Effects    []sim.ItemEffect
}

type currentItemOwner struct {
	Runtime uint32
	Pack    []currentItemValue
	Worn    [sim.EquipSlots]currentItemValue
}

type currentObjectProof struct {
	Owners       []currentItemOwner
	Retired      []uint32
	NextOwner    uint32
	NextCode     uint16
	NextCount    uint32
	NextPrice    int32
	NextWeight   int16
	After        []currentItemOwner
	InputSHA256  string
	Crossing     uint32
	NextPosition [2]int32
	Held         []sim.DeadActorState
	Bodies       []currentBodyValue
}

type currentBodyValue struct {
	MapID uint16
	State sim.DeadActorState
}

func currentBodies(w *sim.World) []currentBodyValue {
	held := make(map[sim.EntityID]bool)
	var out []currentBodyValue
	for _, body := range w.OriginalDeadActors() {
		held[body.ID] = true
		out = append(out, currentBodyValue{body.Source.MapUnitID, body.Current})
	}
	for _, e := range w.Entities() {
		if !held[e.ID] && !e.Alive() && e.Decay >= 1 {
			out = append(out, currentBodyValue{e.MapUnitID, sim.DeadActorState{RuntimeID: e.SourceBinding.RuntimeID, Cell: uint16(e.X) | uint16(e.Y)<<8, FineX: 128, FineY: 128, Stage: uint8(e.Decay), HP: int16(e.HP), Timer: int8(e.Dwell)}})
		}
	}
	sortCurrentBodies(out)
	return out
}

func sortCurrentBodies(bodies []currentBodyValue) {
	slices.SortFunc(bodies, func(a, b currentBodyValue) int {
		if a.MapID != b.MapID {
			return int(a.MapID) - int(b.MapID)
		}
		if a.State.RuntimeID < b.State.RuntimeID {
			return -1
		}
		if a.State.RuntimeID > b.State.RuntimeID {
			return 1
		}
		return int(a.State.Cell) - int(b.State.Cell)
	})
}

func currentObjectOwners(t *testing.T, f *FrontEnd) []currentItemOwner {
	t.Helper()
	value := func(item sim.ItemStack) currentItemValue {
		return currentItemValue{Code: item.Code, Kind: uint16(item.Kind), Count: item.Count, Price: item.Price, Weight: item.Weight, Effects: item.Effects}
	}
	var out []currentItemOwner
	for _, actor := range f.live.world.Entities() {
		if actor.SourceBinding.Class == 0 || !actor.Alive() && actor.Decay >= 2 {
			continue
		}
		row := currentItemOwner{Runtime: actor.SourceBinding.RuntimeID}
		pack, _ := f.live.world.CarriedStacks(actor.ID)
		for _, item := range pack {
			for remaining := item.Count; remaining != 0; {
				part := item
				part.Count = min(remaining, uint32(65535))
				row.Pack = append(row.Pack, value(part))
				remaining -= part.Count
			}
		}
		worn, _ := f.live.world.EquippedItems(actor.ID)
		for i, item := range worn {
			if !item.Empty() {
				row.Worn[i] = value(sim.StackItem(item, 1))
			}
		}
		out = append(out, row)
	}
	slices.SortFunc(out, func(a, b currentItemOwner) int { return int(a.Runtime) - int(b.Runtime) })
	return out
}

func requireCurrentObjectSAV(t *testing.T, path string, want currentObjectProof) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil || doc.World == nil {
		t.Fatal("ordinary SAVE lost the mission", err)
	}
	for _, record := range doc.Objects {
		if record.Class == "Unit" || record.Class == "Human" || record.Class == "Humanoid" {
			id, _ := savedStructureValue(&record, "RuntimeID")
			if slices.Contains(want.Retired, id) {
				t.Fatal("explicitly retired actor was written", id)
			}
		}
	}
	if want.Bodies != nil {
		file, err := sav.Open(raw)
		if err != nil {
			t.Fatal(err)
		}
		dead, err := file.DeadActors()
		if err != nil {
			t.Fatal(err)
		}
		var bodies []currentBodyValue
		for _, d := range dead {
			bodies = append(bodies, currentBodyValue{d.MapUnitID, sim.DeadActorState{RuntimeID: d.RuntimeID, Cell: d.Cell, FineX: d.FineX, FineY: d.FineY, Stage: d.Stage, HP: d.HP, Timer: d.Timer}})
		}
		sortCurrentBodies(bodies)
		if !reflect.DeepEqual(bodies, want.Bodies) {
			t.Fatal("raw SAV current body population or values differ")
		}
	}
}

func mutateCurrentObject(t *testing.T, f *FrontEnd, want *currentObjectProof) {
	t.Helper()
	actor := world1170Entity(t, f, want.NextOwner)
	pack, _ := f.live.world.CarriedStacks(actor.ID)
	index := -1
	for i, item := range pack {
		if item.Code == want.NextCode && min(item.Count, uint32(65535)) == want.NextCount && item.Price == want.NextPrice && item.Weight == want.NextWeight {
			index = i
			break
		}
	}
	if index < 0 {
		t.Fatal("next exact item mutation has no slot")
	}
	f.live.pending = append(f.live.pending, sim.DropCarried(actor.ID, sim.ItemSlot(index), sim.CellPoint{X: actor.X, Y: actor.Y}))
	f.live.tick()
	if want.Crossing != 0 {
		want.NextPosition = world1170Position(f.live.world, world1170Entity(t, f, want.Crossing))
	}
	ground := groundAt(f.live.world.Sacks(), actor.X, actor.Y)
	if ground == nil {
		t.Fatal("ordinary drop did not create ground holdings")
	}
	liveTakeAt(t, f.live, actor.ID, actor.X, actor.Y)
}

func TestReleaseCurrentObjectSAVContinuation(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	if path := os.Getenv("AGAINROM_OBJECT_SAV_INPUT"); path != "" {
		var proof currentObjectProof
		data, err := os.ReadFile(path + ".json")
		if err != nil || json.Unmarshal(data, &proof) != nil {
			t.Fatal("missing object expectations", err)
		}
		f := loadAreaContinuation(t, path)
		if !reflect.DeepEqual(currentBodies(f.live.world), proof.Bodies) {
			t.Fatal("cold body population or values changed")
		}
		var held []sim.DeadActorState
		for _, d := range f.live.world.OriginalDeadActors() {
			held = append(held, d.Current)
		}
		if len(held) < len(proof.Held) || !reflect.DeepEqual(held[:len(proof.Held)], proof.Held) {
			t.Fatal("held corpse state changed")
		}
		if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.Owners) {
			t.Fatal("cold LOAD changed current values/counts/owner/order", firstObjectDifference(proof.Owners, got))
		}
		wantPosition := proof.NextPosition
		mutateCurrentObject(t, f, &proof)
		if got := currentObjectOwners(t, f); !reflect.DeepEqual(got, proof.After) {
			t.Fatal("cold next item mutation differs", firstObjectDifference(proof.After, got))
		}
		if proof.Crossing != 0 && proof.NextPosition != wantPosition {
			t.Fatal("held crossing changed after current item SAVE", proof.NextPosition, wantPosition)
		}
		second := saveCorpseMission(t, f, t.TempDir())
		proof.Bodies = currentBodies(f.live.world)
		requireCurrentObjectSAV(t, second, proof)
		cold := loadAreaContinuation(t, second)
		if got := currentObjectOwners(t, cold); !reflect.DeepEqual(got, proof.After) {
			t.Fatal("second SAV lost next item mutation", firstObjectDifference(proof.After, got))
		}
		return
	}
	_ = releaseFront(t)
	for _, input := range []struct {
		name, hash string
		retired    int
	}{
		{"112323232", "522d5cd01dea60a6d64637884f0427b58888c329be3458259c14d17bce5afe35", 14},
		{"122323", "903efb70bf0e042dc90a3b6b3bbd9ce36aca83534dbc662d4b12c678d0477702", 14},
		{"123", "957f66d791c83261931ac31489988414e14189b124cb2ccc2afa9d4502e9cde3", 11},
		{"save1234", "199a94ce8024f784cd2cd0945d64081806b1e0f6823e75e72616181ebe01aa9c", 11},
	} {
		t.Run(input.name, func(t *testing.T) {
			fixture := input.name
			if fixture == "112323232" {
				fixture = "mission111-portrait-input"
			}
			_, raw := ownerCurrentSAV(t, fixture)
			expectedSource := ownerSourceProof(t, fixture)
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			open, town, err := f.RestoreOriginal(raw)
			if err != nil || town {
				t.Fatal("native input LOAD", town, err)
			}
			if err := f.App("current objects").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			proof := currentObjectProof{Owners: currentObjectOwners(t, f), InputSHA256: input.hash, Bodies: currentBodies(f.live.world)}
			if !reflect.DeepEqual(expectedSource.Owners, proof.Owners) || !reflect.DeepEqual(expectedSource.Bodies, proof.Bodies) {
				t.Fatal("SAV fixture lost independent source owner/body state")
			}

			for _, d := range f.live.world.OriginalDeadActors() {
				proof.Held = append(proof.Held, d.Current)
			}
			if !reflect.DeepEqual(expectedSource.Held, proof.Held) {
				t.Fatal("SAV fixture lost independent source held corpse state")
			}
			proof.Retired = append([]uint32(nil), expectedSource.Retired...)
			if len(proof.Retired) != input.retired {
				t.Fatal("retired input population differs", len(proof.Retired))
			}
			for _, a := range f.live.world.Entities() {
				if a.Transit > 0 {
					proof.Crossing = a.SourceBinding.RuntimeID
				}
				pack, _ := f.live.world.CarriedStacks(a.ID)
				if a.Owner == sim.SelfSlot && a.Alive() && a.SourceBinding.RuntimeID != 0 && len(pack) > 0 && a.Transit == 0 {
					proof.NextOwner, proof.NextCode, proof.NextCount = a.SourceBinding.RuntimeID, pack[0].Code, pack[0].Count
					proof.NextPrice, proof.NextWeight = pack[0].Price, pack[0].Weight
				}
			}
			if proof.NextOwner == 0 {
				t.Fatal("next-mutation owner missing")
			}
			seed := saveCorpseMission(t, f, t.TempDir())
			requireCurrentObjectSAV(t, seed, proof)
			mutateCurrentObject(t, f, &proof)
			proof.After = currentObjectOwners(t, f)
			data, _ := json.MarshalIndent(proof, "", "  ")
			if err := os.WriteFile(seed+".json", data, 0600); err != nil {
				t.Fatal(err)
			}
			emitSpellWitness(t, seed, "objects-"+input.name, data)
			runSpellWitnessChild(t, seed, "AGAINROM_OBJECT_SAV_INPUT")
			t.Logf("unchanged %s: %d retired; ordinary mission SAVE, cold values/order, drop/pickup and second SAV", input.name, input.retired)
		})
	}
}

func firstObjectDifference(want, got []currentItemOwner) string {
	for i := range min(len(want), len(got)) {
		if !reflect.DeepEqual(want[i], got[i]) {
			return fmt.Sprintf("owner slot%d want %+v got %+v", i, want[i], got[i])
		}
	}
	return fmt.Sprintf("owner count %d/%d", len(want), len(got))
}

// DIV-3348 bounds the replacement subjects; it does not promise legacy hash parity.
func ownerCurrentSAV(t *testing.T, name string) (string, []byte) {
	t.Helper()
	root := os.Getenv("AGAINROM_RESTORE_OWNER_FIXTURES")
	if root == "" {
		t.Skip("AGAINROM_RESTORE_OWNER_FIXTURES is not set")
	}
	locale := "en"
	if releaseFront(t).textSelector() == 1 {
		locale = "ru"
	}
	hashes := map[string][2]string{
		"122323":                    {"0ab5d2dd647454849eb4ed0dac744d2f84e4021b800da8495f4434b4fc56e2e9", "713272ab4bec4b2e510dfb3b65d2cac3e9bf2d2d4e2acc5df5f2f21d2cfc4da1"},
		"123":                       {"3993b79acf4c2f1045098150b9aa3e5f5d6de0f3a72154b10b034538b6b9ff00", "ceb21629266d2bcc2888ce4dfb835fe6c0d909098aff6b1d1745291ef6789bee"},
		"save1234":                  {"5bcc04684a13a559d80ccef1a319d4369dbd88ba9a8ed82f2e2b879d652a2bec", "911f483b9df66835427294ddf680eb60cf8964f0ff4622c8339bef89f1e0442f"},
		"mission111-enemy-dolls":    {"13bc40dad274f6c9fead13f1bb96e3c033d88d10cde11eefaeffa87966eff48a", "13bc40dad274f6c9fead13f1bb96e3c033d88d10cde11eefaeffa87966eff48a"},
		"mission110-portrait-input": {"d77f7e9109b79254135c2ec9fc66c863cfc1981210735676f653afdd7cf12ca1", "c72dc4cd3f9dc90a34376d6cc7c74d72f1a887410a957d598800ecf6fd7434c5"},
		"mission111-portrait-input": {"95ac3ebeb80f385db8a0390cb77aafaf4fe24dfb7462b9edbc39f30307369654", "d01a7a1847023a573ea9c8ca7a3473511ad35aebefeea98aa11ec8a225936bff"},
	}
	expected, ok := hashes[name]
	if !ok {
		t.Fatal("unknown owner SAV fixture", name)
	}
	at := 0
	if locale == "ru" {
		at = 1
	}
	path := filepath.Join(root, locale, name, "fixture.sav")
	raw, err := os.ReadFile(path)
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != expected[at] {
		t.Fatal("owner SAV fixture missing or changed", name, locale, err)
	}
	return path, raw
}

func ownerSourceProof(t *testing.T, name string) currentObjectProof {
	t.Helper()
	hashes := map[string]string{
		"122323":                    "59707db2605b7cf1cfd60d029c7f90d9fc1e0c1891aaa983834b238d6095b5d5",
		"123":                       "eb4c8e370095195dac144443f5ce9307e0147caf142336cde80246ce34f9b7ad",
		"save1234":                  "67c27f6993ed0735d8bf95f2d260aeab234b7e16c12053ba4c06eea5ce2aec09",
		"mission111-portrait-input": "a70d9549535a6fa889826952d3f5992a74fe186bc7f034eca082916c34606601",
	}
	expected, ok := hashes[name]
	if !ok {
		t.Fatal("unknown source expectation manifest", name)
	}
	root := os.Getenv("AGAINROM_RESTORE_OWNER_FIXTURES")
	if root == "" {
		t.Skip("AGAINROM_RESTORE_OWNER_FIXTURES is not set")
	}
	locale := "en"
	if releaseFront(t).textSelector() == 1 {
		locale = "ru"
	}
	raw, err := os.ReadFile(filepath.Join(root, locale, name, "source-object-proof.json"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != expected {
		t.Fatal("independent source expectation manifest missing or changed", name, err)
	}
	var proof currentObjectProof
	if err = json.Unmarshal(raw, &proof); err != nil {
		t.Fatal(err)
	}
	return proof
}
