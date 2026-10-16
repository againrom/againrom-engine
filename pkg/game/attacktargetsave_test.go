package game

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type attackTargetWireFault struct {
	Object   int
	Class    string
	Identity uint32
	Health   int16
	Row      byte
	Target   uint32
}

func attackTargetWireValue(r sav.DocumentRecordData, name string) uint32 {
	for _, v := range r.Values {
		if v.Name == name {
			return v.Value
		}
	}
	return 0
}

func attackTargetWireOrder(r sav.DocumentRecordData) []byte {
	for _, v := range r.Raw {
		if v.Name == "U158" {
			return v.Bytes
		}
	}
	return nil
}

func attackTargetWireCensus(doc sav.DocumentData) (living, reading int, faults []attackTargetWireFault) {
	keys := map[uint32]int{}
	for i, r := range doc.Objects {
		for _, v := range r.Values {
			if (v.Name == "Identity" || v.Name == "This") && v.Value != 0 {
				if prior, ok := keys[v.Value]; !ok {
					keys[v.Value] = i
				} else if prior != i {
					keys[v.Value] = -1
				}
			}
		}
	}
	for i, r := range doc.Objects {
		if r.Class != "Unit" && r.Class != "Human" && r.Class != "Humanoid" {
			continue
		}
		hp, maximum := int16(attackTargetWireValue(r, "Health")), int16(attackTargetWireValue(r, "HealthMax"))
		if hp < 0 || hp == 0 && maximum > 0 {
			continue
		}
		living++
		order := attackTargetWireOrder(r)
		if len(order) != 148 {
			continue
		}
		at := 0
		switch order[8] {
		case 5, 6:
			at = 0x0c
		case 8:
			at = 0x28
		default:
			continue
		}
		reading++
		target := binary.LittleEndian.Uint32(order[at:])
		index, resolved := keys[target]
		if !resolved || index < 0 {
			faults = append(faults, attackTargetWireFault{i, r.Class, attackTargetWireValue(r, "Identity"), hp, order[8], target})
		}
	}
	return living, reading, faults
}

func attackTargetSaveFront(t *testing.T) *FrontEnd {
	t.Helper()
	f := currentPoolFixtureFront(t, 91, 92)
	if err := f.App("attack target save").OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	actors := []sim.Entity{
		{ID: 1, Owner: sim.SelfSlot, Group: 1, X: 10, Y: 10, HP: 100, MaxHP: 100, DamageBase: 30, AlwaysHits: true, Reach: 1, AttackCharge: 40, AttackRelax: 2, DyingTime: 60},
		{ID: 2, Owner: 3, Group: 2, X: 11, Y: 10, HP: 20, MaxHP: 20, DyingTime: 60},
	}
	for i := range actors {
		actors[i].TokenSize, actors[i].TypeID, actors[i].Speed = 1, 1, 10
		actors[i].ScanRange = 5
		actors[i].HealthRegenPeriod, actors[i].ManaRegenPeriod, actors[i].Capacity = 1, 1, data.UnitCapacity()
	}
	var relations sim.Relations
	relations.Set(sim.SelfSlot, 3, 1)
	relations.Set(3, sim.SelfSlot, 1)
	w, err := sim.NewRelatedWorld(1, f.live.world.Bounds(), sim.ModeCanonical, sim.Terrain{}, actors, nil, relations)
	if err != nil {
		t.Fatal(err)
	}
	f.live.world, f.live.mission.state.World = w, w
	return f
}

func TestAttackTargetDeathSaveHasResolvedOrders(t *testing.T) {
	for _, cause := range []string{"physical blow", "damage command"} {
		t.Run(cause, func(t *testing.T) {
			f := attackTargetSaveFront(t)
			w := f.live.world
			sim.Step(w, []sim.Command{sim.MoveTo(1, sim.CellPoint{X: 10, Y: 10})})
			sim.Step(w, []sim.Command{sim.Attack(1, 2)})
			for n := 0; n < 128; n++ {
				_, orders, _ := w.SavedGroups()
				admitted := false
				for _, order := range orders {
					admitted = admitted || order.Entity == 1 && order.Raw[8] == 5
				}
				if admitted {
					break
				}
				sim.Step(w, nil)
			}
			if e, _ := w.Entity(1); !e.HasAttackTarget || e.AttackTarget != 2 {
				t.Fatal("ordinary attack did not retain the living victim", e)
			}
			_, before, actions := saveCurrentEffect(t, f)
			order := attackTargetWireOrder(castOrderRecord(t, before, actions, 1))
			if len(order) != 148 || order[8] != 5 || binary.LittleEndian.Uint32(order[0x0c:]) == 0 {
				t.Fatal("control SAVE did not hold a resolved row-5 attack", order)
			}
			if cause == "damage command" {
				sim.Step(w, []sim.Command{sim.Damage(2, 100)})
			} else {
				for n := 0; n < 128; n++ {
					victim, present := w.Entity(2)
					if !present || !victim.Alive() {
						break
					}
					sim.Step(w, nil)
				}
			}
			victim, present := w.Entity(2)
			if !present || victim.Alive() {
				t.Fatal("production death path did not leave the dead victim", victim, present)
			}
			raw, doc, actions := saveCurrentEffect(t, f)
			order = attackTargetWireOrder(castOrderRecord(t, doc, actions, 1))
			t.Logf("death=%s victimHP=%d attackerRow=%d attackerTarget=%#x", cause, victim.HP, order[8], binary.LittleEndian.Uint32(order[0x0c:]))
			if _, _, faults := attackTargetWireCensus(doc); len(faults) != 0 {
				t.Fatalf("living actor SAVE holds an unresolved target: %+v", faults)
			}
			cold := openCurrentEffectSave(t, f, raw)
			if w.Hash() != cold.live.world.Hash() {
				t.Fatal("death SAVE changed the World on cold LOAD")
			}
			_, second, _ := saveCurrentEffect(t, cold)
			if _, _, faults := attackTargetWireCensus(second); len(faults) != 0 {
				t.Fatalf("cold LOAD and second SAVE restored an unresolved target: %+v", faults)
			}
		})
	}
}

func attackTargetHeadlessResave(t *testing.T, assets string, raw []byte, directory, name, label string) []byte {
	t.Helper()
	result, err := attackTargetHeadlessResaveResult(assets, raw, directory, name, label)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func attackTargetHeadlessResaveResult(assets string, raw []byte, directory, name, label string) ([]byte, error) {
	f, err := decodedInstallFront(assets)
	if err != nil {
		return nil, fmt.Errorf("FRONTEND: %w", err)
	}
	devices := []interface{ Stop() }{f.MusicPlayer, f.AmbientPlayer, f.CutsceneAudioPlayer}
	owner := ui.DeliveryOwner(f.SoundPlayer)
	defer func() {
		for _, device := range devices {
			if device != nil {
				device.Stop()
			}
		}
		if owner != nil && owner.Service != nil {
			owner.Service.Close()
		}
	}()
	f.MusicPlayer = fakePlayer{}
	f.SoundPlayer, f.speech.player, f.AmbientPlayer = fakePlayer{}, fakePlayer{}, fakePlayer{}
	f.CutsceneAudioPlayer = fakeCutsceneAudioPlayer{}
	f.SetDeterministicFrames(true)
	open, town, err := f.RestoreOriginal(raw)
	if err != nil {
		return nil, fmt.Errorf("LOAD: %w", err)
	}
	if !town {
		app := f.App("attack target witness")
		defer app.StopAudio()
		app.Layout(1024, 768)
		if err := app.OpenMission(open); err != nil {
			return nil, fmt.Errorf("OPEN: %w", err)
		}
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		return nil, fmt.Errorf("OUTPUT-DIR: %w", err)
	}
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: directory}, OriginalStore{}).Prepare(ui.SaveRequest{
		OnMap: !town, Directory: directory, Name: name, Label: label, Format: ui.SaveSAV,
	})
	if err != nil {
		return nil, fmt.Errorf("SAVE-PREPARE: %w", err)
	}
	if _, err := prepared.Commit(true); err != nil {
		return nil, fmt.Errorf("SAVE-COMMIT: %w", err)
	}
	result, err := ReadSaveFile(filepath.Join(directory, name+".sav"))
	if err != nil {
		return nil, fmt.Errorf("READ: %w", err)
	}
	return result, nil
}

func attackTargetCopiedInput(t *testing.T, source, directory string) []byte {
	t.Helper()
	if !filepath.IsAbs(directory) {
		t.Fatal("witness directory must be absolute", directory)
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(directory, "input.sav")
	if err := os.WriteFile(path, raw, 0600); err != nil {
		t.Fatal(err)
	}
	copy, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return copy
}

func TestAttackTargetOwnerSave(t *testing.T) {
	source, output, assets := os.Getenv("AGAINROM_ATTACK_TARGET_OWNER_SAVE"), os.Getenv("AGAINROM_ATTACK_TARGET_OWNER_OUTPUT"), os.Getenv("AGAINROM_ASSETS")
	if source == "" || output == "" || assets == "" {
		t.Skip("AGAINROM_ASSETS, AGAINROM_ATTACK_TARGET_OWNER_SAVE and AGAINROM_ATTACK_TARGET_OWNER_OUTPUT are required")
	}
	output = filepath.Join(output, filepath.Base(assets))
	raw := attackTargetCopiedInput(t, source, filepath.Join(output, "input"))
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	const identity uint32 = 0x51000410
	actor := -1
	for i, r := range doc.Objects {
		if attackTargetWireValue(r, "Identity") == identity {
			actor = i
		}
	}
	if actor != 22 {
		t.Fatalf("owner object identity %#x is object[%d], want object[22]", identity, actor)
	}
	order := attackTargetWireOrder(doc.Objects[actor])
	if len(order) != 148 || order[8] != 5 || binary.LittleEndian.Uint32(order[0x0c:]) != 0 {
		t.Fatal("owner control no longer has the reported targetless row-5 attack")
	}
	prior, err := readCurrentActions(&doc)
	if err != nil || prior == nil {
		t.Fatal("owner save has no current actor bindings", err)
	}
	var entity sim.EntityID
	bound := false
	for _, b := range prior.Bindings {
		if !b.Missing && !b.Structure && int(b.Object) == actor+1 {
			entity, bound = b.ID, true
		}
	}
	if !bound {
		t.Fatal("owner object[22] has no current actor binding")
	}
	beforeLiving, beforeReading, before := attackTargetWireCensus(doc)
	written := attackTargetHeadlessResave(t, assets, raw, output, "game0020", "110 attack target fixed")
	next, err := sav.DecodeDocumentData(written)
	if err != nil {
		t.Fatal(err)
	}
	afterLiving, afterReading, after := attackTargetWireCensus(next)
	current, err := readCurrentActions(&next)
	if err != nil || current == nil {
		t.Fatal("resave has no current actor bindings", err)
	}
	newActor := castOrderRecord(t, next, current, entity)
	newOrder := attackTargetWireOrder(newActor)
	if len(newOrder) != 148 {
		t.Fatal("owner resave has no complete ordinary order")
	}
	t.Logf("OWNER source=%s sha256=%x object[22] identity=%#x actor=%d row=%d target=%#x living=%d reading=%d unresolved=%d", source, sha256.Sum256(raw), identity, entity, order[8], binary.LittleEndian.Uint32(order[0x0c:]), beforeLiving, beforeReading, len(before))
	t.Logf("OWNER result=%s sha256=%x formerObject[22] identity=%#x actor=%d row=%d target=%#x living=%d reading=%d unresolved=%d", filepath.Join(output, "game0020.sav"), sha256.Sum256(written), attackTargetWireValue(newActor, "Identity"), entity, newOrder[8], binary.LittleEndian.Uint32(newOrder[0x0c:]), afterLiving, afterReading, len(after))
	if len(after) != 0 {
		t.Fatalf("owner resave still holds unresolved living target orders: %+v", after)
	}
	if len(next.Label) == 0 || strings.TrimRight(string(next.Label), "\x00") != "110 attack target fixed" {
		t.Fatalf("owner chooser label=%q", next.Label)
	}
}

func TestAttackTargetWireCensusDistinguishesActorAndCastOperands(t *testing.T) {
	for _, row := range []byte{5, 6, 8} {
		for _, target := range []uint32{0, 23, 999} {
			order := make([]byte, 148)
			order[8] = row
			at := 0x0c
			if row == 8 {
				at = 0x28
			}
			binary.LittleEndian.PutUint32(order[at:], target)
			doc := sav.DocumentData{Objects: []sav.DocumentRecordData{
				{Class: "Human", Values: []sav.DocumentValueData{{Name: "Health", Value: 10}, {Name: "HealthMax", Value: 10}, {Name: "Identity", Value: 11}}, Raw: []sav.DocumentRawData{{Name: "U158", Bytes: order}}},
				{Class: "Unit", Values: []sav.DocumentValueData{{Name: "Health", Value: 0xffff}, {Name: "Identity", Value: 23}}},
			}}
			living, reading, faults := attackTargetWireCensus(doc)
			want := 1
			if target == 23 {
				want = 0
			}
			if living != 1 || reading != 1 || len(faults) != want {
				t.Fatal(fmt.Sprintf("row=%d target=%d living=%d reading=%d faults=%+v", row, target, living, reading, faults))
			}
		}
	}
}
