package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"againrom/pkg/sim"
)

type writerSourceReference struct {
	Route, File, SHA256 string
}

func writeWriterSourceCapture(t *testing.T, dir, name string, source map[string]any) writerSourceReference {
	t.Helper()
	if filepath.Base(name) != name {
		t.Fatal("source capture name must remain inside its output directory")
	}
	raw, err := json.Marshal(struct {
		Revision, KnowledgePin string
		Capture                map[string]any
	}{os.Getenv("AGAINROM_WITNESS_REVISION"), os.Getenv("AGAINROM_WITNESS_KNOWLEDGE"), source})
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) > 64<<20 {
		t.Fatal("source capture exceeds bounded atlas input")
	}
	file, err := os.OpenFile(filepath.Join(dir, name), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write(raw); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return writerSourceReference{source["Route"].(string), name, fmt.Sprintf("%x", sha256.Sum256(raw))}
}

func observeWriterSources(f *FrontEnd, captured Snapshot, route string) map[string]any {
	result := map[string]any{"Route": route, "Snapshot": captured, "Game": captured.game,
		"Second": captured.second, "CityGroups": captured.cityGroups, "WorldPresent": len(captured.World) != 0}
	result["ConstructionTablePresent"], result["ConstructionInputs"] = observeLootConstruction(f.Table)
	if f.Table != nil {
		result["ConstructionGame"] = f.Table.Game
	}
	if len(captured.World) == 0 || f.live == nil || f.live.world == nil {
		return result
	}
	w := f.live.world
	result["Bounds"], result["Ghost"] = w.Bounds(), w.Ghost()
	result["Hash"], result["Tick"] = fmt.Sprintf("%x", w.Hash()), w.Tick()
	result["RawSessionHead"], result["RawSessionMid"] = w.RawSessionHead(), w.RawSessionMid()
	clock, present := w.SessionClock()
	result["Clock"], result["ClockPresent"] = clock, present
	result["Registers"], result["Relations"], result["Outcome"] = w.ScriptRegisters(), w.Relations(), w.Outcome()
	won, lost := w.ScriptCounters()
	result["Won"], result["Lost"] = won, lost
	result["Policy"], result["Actions"] = w.CurrentPolicy(), w.Actions()
	result["Entities"], result["OriginalDead"], result["TerminalActors"] = w.Entities(), w.OriginalDeadActors(), w.CurrentTerminalActors()
	result["RemovedNativeBases"] = w.RemovedNativeActorBases()
	var actors []map[string]any
	for _, entity := range w.Entities() {
		pack, packPresent := w.CarriedStacks(entity.ID)
		worn, wornPresent := w.EquippedItems(entity.ID)
		actors = append(actors, map[string]any{"Entity": entity.ID, "SourceNow": entity.SourceNow(), "Pack": pack,
			"PackPresent": packPresent, "Worn": worn, "WornPresent": wornPresent})
	}
	result["ActorSources"] = actors
	groups, orders, present := w.SavedGroups()
	result["Groups"], result["Orders"], result["GroupsPresent"] = groups, orders, present
	players, present := w.SavedGroupPlayers()
	result["Players"], result["PlayersPresent"] = players, present
	participants, present := w.PlayerParticipants()
	result["Participants"], result["ParticipantsPresent"] = participants, present
	players, playersPresent := w.CurrentPlayers()
	result["CurrentPlayers"], result["CurrentPlayersPresent"] = players, playersPresent
	formations, present := w.SavedPlayerFormations()
	result["Formations"], result["FormationsPresent"] = formations, present
	result["GroupIssues"], result["Diaries"] = w.SavedGroupIssues(), w.SavedDiaries()
	motions, cells, blocks, present := w.SavedActorMotions()
	result["Motions"], result["MotionCells"], result["MotionBlocks"], result["MotionsPresent"] = motions, cells, blocks, present
	result["MotionIssues"] = w.SavedActorMotionIssues()
	structures, structureCells, present := w.SavedStructures()
	result["SavedStructures"], result["StructureCells"], result["StructuresPresent"] = structures, structureCells, present
	result["Structures"], result["Cells"], result["CellTails"] = w.Structures(), w.SavedCellRecords(), w.CellTails()
	planes, present := w.SavedCellPlanes()
	result["Planes"], result["PlanesPresent"] = planes, present
	result["Objects"], result["ItemWeights"], result["Sacks"] = w.SavedObjects(), w.ItemWeights(), w.Sacks()
	result["ActiveEffects"], result["CellEffects"] = w.ActiveEffects(), w.CellEffects()
	result["Spells"] = w.Spells()
	result["SpellGraph"], result["SpellEffects"], result["Projectiles"] = w.SavedSpellGraph(), w.SavedSpellEffects(), w.SavedProjectiles()
	result["WorldEffectDrivers"], result["WorldEffectOrder"] = w.SavedWorldEffectDrivers(), w.CurrentWorldEffectOrder()
	areas, err := w.NativeAreaSaveStates()
	result["NativeAreas"] = areas
	if err != nil {
		result["NativeAreasIssue"] = err.Error()
	}
	if f.live.mission != nil && f.live.mission.state != nil {
		ms := f.live.mission.state
		result["RetainedDocument"], result["MissionAddress"] = ms.savedDocument, ms.Address
		if ms.Map != nil {
			result["MapDescriptors"] = map[string]any{"Width": ms.Map.Width, "Height": ms.Map.Height,
				"Groups": slices.Clone(ms.Map.Groups), "Units": slices.Clone(ms.Map.Units)}
		}
	}
	return result
}

func TestWriterSourceObservationKeepsZeroHeadAndSeparateCurrentCarriers(t *testing.T) {
	f := castOrderFront(t)
	captured, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	before := f.live.world.Hash()
	observed := observeWriterSources(f, captured, "source control")
	if observed["WorldPresent"] != true || observed["RawSessionHead"] != ([48]byte{}) || observed["Tick"] != f.live.world.Tick() || f.live.world.Hash() != before {
		t.Fatal("observer changed current state or hid legal zero head")
	}
	entities := observed["Entities"].([]sim.Entity)
	if len(entities) != 2 || len(observed["ActorSources"].([]map[string]any)) != 2 {
		t.Fatal("exact actor source population lost")
	}
	notifySaveCapture([]func(string, Snapshot){nil, func(route string, s Snapshot) {
		if route != "source control" || len(s.World) == 0 {
			t.Fatal("capture callback lost exact snapshot")
		}
	}}, "source control", captured)
}

func TestWriterSourceCaptureFreezesValuesBeforeLaterMutation(t *testing.T) {
	dir := t.TempDir()
	values := []int{17}
	source := map[string]any{"Route": "source control", "Values": values}
	ref := writeWriterSourceCapture(t, dir, "capture.json", source)
	values[0] = 99
	raw, err := os.ReadFile(filepath.Join(dir, ref.File))
	if err != nil || ref.SHA256 != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("capture reference lost its exact bytes", err)
	}
	var captured struct {
		Capture struct{ Values []int }
	}
	if err := json.Unmarshal(raw, &captured); err != nil || len(captured.Capture.Values) != 1 || captured.Capture.Values[0] != 17 {
		t.Fatal("capture changed after the producer boundary", err, captured)
	}
}
