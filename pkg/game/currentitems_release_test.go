package game

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

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
	dir := os.Getenv("AGAINROM_OBJECT_AGS_DIR")
	if dir == "" {
		dir = filepath.Join(filepath.Dir(os.Getenv("AGAINROM_ASSETS")), "..", "engine", "saves")
	}
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
			path := filepath.Join(dir, input.name+".ags")
			raw, err := os.ReadFile(path)
			digest := sha256.Sum256(raw)
			if err != nil || hex.EncodeToString(digest[:]) != input.hash {
				t.Fatal("unchanged native corpus input missing or changed", path, err)
			}
			defer func() {
				after, err := os.ReadFile(path)
				if err != nil || sha256.Sum256(after) != digest {
					t.Fatal("original AGS input changed", err)
				}
			}()
			s, _, err := DecodeSave(raw)
			if err != nil {
				t.Fatal(err)
			}
			f := releaseFront(t)
			f.Options = OptionsStore{}
			f.SetDeterministicFrames(true)
			open, town, err := f.Restore(s)
			if err != nil || town {
				t.Fatal("native input LOAD", town, err)
			}
			if err := f.App("current objects").OpenMission(open); err != nil {
				t.Fatal(err)
			}
			proof := currentObjectProof{Owners: currentObjectOwners(t, f), InputSHA256: input.hash, Bodies: currentBodies(f.live.world)}
			held := make(map[sim.EntityID]bool)
			for _, d := range f.live.world.OriginalDeadActors() {
				held[d.ID] = true
				proof.Held = append(proof.Held, d.Current)
			}
			current, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range current.SavedDocument.Actors {
				if a.Retired && !held[a.EntityID] {
					id, _ := savedStructureValue(&current.SavedDocument.Document.Objects[a.ObjectIndex-1], "RuntimeID")
					proof.Retired = append(proof.Retired, id)
				}
			}
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
