package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"againrom/pkg/base"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

type secondTownSample struct {
	Campaign      *currentSecondCampaign
	Party         [][]byte
	Gold, Offered int
	Difficulty    mapload.Difficulty
	Quick         [4]uint32
	Fame          SnapshotFame
	Header        string
	Rows          []ui.TownRow
	Frame         [32]byte
}

func secondTownSampleNow(t *testing.T, f *FrontEnd, app *ui.App) secondTownSample {
	t.Helper()
	screen := f.TownScreen()
	picture, note, err := app.HeadlessFrame()
	if err != nil || note != "" {
		t.Fatal("composed first-town frame", err, note)
	}
	sample := secondTownSample{Campaign: captureSecondCampaign(f.Town.second), Gold: f.Town.gold, Offered: f.Offered,
		Difficulty: f.Difficulty, Quick: f.quickSpells, Fame: cloneFame(f.fame), Header: screen.Header(), Rows: screen.Rows(), Frame: sha256.Sum256(picture.Pix)}
	for _, p := range mapload.CloneParty(f.Carried) {
		p.WornItems = mapload.MemberItemEquipment(p, f.Table)
		p.CarriedItems = mapload.MemberCarriedItems(p, f.Table)
		p.OriginalHuman = nil
		var raw bytes.Buffer
		if err := gob.NewEncoder(&raw).Encode(p); err != nil {
			t.Fatal(err)
		}
		sample.Party = append(sample.Party, raw.Bytes())
	}
	return sample
}

func secondTownAssertSample(t *testing.T, want, got secondTownSample) {
	t.Helper()
	if !reflect.DeepEqual(want, got) {
		for i := range want.Party {
			if i >= len(got.Party) || bytes.Equal(want.Party[i], got.Party[i]) {
				continue
			}
			var before, after mapload.PartyMember
			_ = gob.NewDecoder(bytes.NewReader(want.Party[i])).Decode(&before)
			_ = gob.NewDecoder(bytes.NewReader(got.Party[i])).Decode(&after)
			a, b := reflect.ValueOf(before), reflect.ValueOf(after)
			for field := 0; field < a.NumField(); field++ {
				if !reflect.DeepEqual(a.Field(field).Interface(), b.Field(field).Interface()) {
					t.Logf("party input %s before=%v after=%v", a.Type().Field(field).Name, a.Field(field).Interface(), b.Field(field).Interface())
				}
			}
		}
		t.Fatalf("town continuation: campaign=%v party=%v gold=%d/%d offered=%d/%d difficulty=%d/%d quick=%v fame=%v header=%q/%q rows=%v frame=%v", reflect.DeepEqual(want.Campaign, got.Campaign), reflect.DeepEqual(want.Party, got.Party), want.Gold, got.Gold, want.Offered, got.Offered, want.Difficulty, got.Difficulty, want.Quick == got.Quick, reflect.DeepEqual(want.Fame, got.Fame), want.Header, got.Header, reflect.DeepEqual(want.Rows, got.Rows), want.Frame == got.Frame)
	}
}

func secondTownNew(t *testing.T, inn, unlocked bool) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	app := f.App("first town save")
	app.Layout(1024, 768)
	if err := app.HeadlessActivate("new game"); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatal("ordinary NEW GAME first town", err, app.Screen())
	}
	if inn || unlocked {
		if err := app.HeadlessActivate("TAVERN"); err != nil {
			t.Fatal(err)
		}
	}
	if unlocked {
		secondTownTalk(t, app)
		if !inn {
			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
		}
	}
	return f, app
}

func secondTownTalk(t *testing.T, app *ui.App) {
	t.Helper()
	if err := app.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	for page := 0; page < 64; page++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			return
		}
	}
	t.Fatal("initial TALK exceeded 64 pages")
}

func secondTownMutate(f *FrontEnd) {
	f.Carried[0].Name += string([]byte{' ', 0xe4, 0xa0, '7'})
	f.Carried[0].Hero.Body += 3
	f.Carried[0].Hero.Mind += 2
	f.Carried[0].KnownSpells = 1 << 2
	f.Carried[0].Book = sim.Spellbook{State: sim.BookPresent}
	f.Carried[0].Book.Slots[1] = sim.BookSpell{Range: 7, Defensive: 1, ManaCost: 11}
	f.Carried[0].SpellbookPresent, f.Carried[0].SpellbookRestored = true, true
	equipment := mapload.MemberItemEquipment(f.Carried[0], f.Table)
	if equipment[0].Code != 0 {
		f.Carried[0].Carried = append(f.Carried[0].Carried, equipment[0].Code)
		f.Carried[0].CarriedItems = append(f.Carried[0].CarriedItems, equipment[0])
	}
	f.Town.gold, f.Difficulty = 777, mapload.DifficultyHard
	f.quickSpells = [4]uint32{2, 0, 4, 0}
	f.Town.second.bank[0], f.Town.second.bank[997], f.Town.second.bank[754] = -27, 34567, 91
}

func secondTownNamedSave(t *testing.T, f *FrontEnd, app *ui.App, out, name string) []byte {
	t.Helper()
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatalf("ordinary first-town SAVE refusal: %v screen=%v bank_stage=%d", err, app.Screen(), f.Town.second.bank[768])
	}
	if err := app.HeadlessSaveEdit(out, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		_, fileErr := os.Stat(filepath.Join(out, name+".sav"))
		t.Fatalf("ordinary first-town SAVE producer refusal: %v file=%v", err, fileErr)
	}
	if app.Screen() != ui.ScreenGameMenu || app.HeadlessMessage() != f.Words.SaveAcknowledgement {
		t.Fatal("ordinary SAVE acknowledgement", app.Screen(), app.HeadlessMessage())
	}
	raw, err := os.ReadFile(filepath.Join(out, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal(err)
	}
	return raw
}

func secondTownCold(t *testing.T, out, name string, prepare ...func(*FrontEnd)) (*FrontEnd, *ui.App) {
	t.Helper()
	f := secondGameFront(t)
	for _, p := range prepare {
		p(f)
	}
	app := f.App("cold first town")
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	token := localOriginalSaveToken(name)
	label := ""
	for _, row := range list() {
		if row.Name == token {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("town SAV absent from ordinary LOAD list", token)
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || f.live != nil || f.Town.second == nil || f.Town.Open() || f.Shop != nil {
		t.Fatal("cold LOAD applied mission or ROM1 arrival", app.Screen(), f.live, f.Town.Open(), f.Shop)
	}
	return f, app
}

func secondTownShape(t *testing.T, raw []byte) {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	actions, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Head.Mission != 0 || doc.World != nil || actions.Session.Game != base.GameROM2 || actions.Session.Second == nil || actions.Program != nil || actions.Policy != nil || actions.Fog != nil || len(actions.Party) == 0 {
		t.Fatal("town SAV lacks explicit city shape")
	}
}

func secondTownEnter(t *testing.T, f *FrontEnd, app *ui.App) {
	t.Helper()
	c := f.Town.second
	bank := c.bank
	if c.room == secondTownSquare {
		if err := app.HeadlessActivate("TAVERN"); err != nil {
			t.Fatal(err)
		}
	}
	secondTownTalk(t, app)
	// The initial record stores slot 769 (R2-SESSION-110).
	bank[769] = 1
	if c.bank != bank || !reflect.DeepEqual(c.available, []secondLocation{{2, 1}, {1, 10}}) {
		t.Fatal("ordinary TALK changed bank or duplicated unlock")
	}
	secondTownTalk(t, app)
	if len(c.available) != 2 {
		t.Fatal("repeated ordinary TALK duplicated destination")
	}
	if err := app.HeadlessActivate("GATES"); err != nil {
		t.Fatal(err)
	}
	party := mapload.CloneParty(f.Carried)
	for _, target := range []string{"mission 10", "CANCEL"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if c.bank != bank || c.current != (secondLocation{}) || !reflect.DeepEqual(c.available, []secondLocation{{1, 10}}) || !reflect.DeepEqual(party, f.Carried) {
		t.Fatal("cancelled destination changed town inputs")
	}
	for _, target := range []string{"mission 10", "ENTER"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenMap || f.liveMission != 10 || f.live.mission.resumed {
		t.Fatal("ordinary initial departure did not start mission 10")
	}
	got, ok := f.live.world.ROM2ScenarioState()
	if !ok {
		t.Fatal("first mission bank absent")
	}
	for slot, want := range bank {
		if slot >= 752 && slot < 768 {
			want = 0
		}
		if got[slot] != want {
			t.Fatalf("ordinary entry bank[%d]=%d want%d", slot, got[slot], want)
		}
	}
	for range 20 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	dismissSecondMissionDialogue(t, app, f.live)
}

type secondTownProof struct {
	Sample      secondTownSample
	Mission     []secondSaveSample
	Actor       sim.EntityID
	X, Y, Ticks int
}

func TestReleaseSecondTownNamedSaveContinuation(t *testing.T) {
	secondGameRoot(t)
	if path := os.Getenv("AGAINROM_SECOND_TOWN_SAVE_INPUT"); path != "" {
		raw, err := os.ReadFile(path + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var proof secondTownProof
		if err := json.Unmarshal(raw, &proof); err != nil {
			t.Fatal(err)
		}
		f, app := secondTownCold(t, filepath.Dir(path), filepath.Base(path))
		secondTownAssertSample(t, proof.Sample, secondTownSampleNow(t, f, app))
		secondTownEnter(t, f, app)
		secondAssertSample(t, proof.Mission[0], secondSaveSampleNow(t, f, app))
		secondMissionMove(t, app, proof.Actor, proof.X, proof.Y)
		for range proof.Ticks {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		secondAssertSample(t, proof.Mission[1], secondSaveSampleNow(t, f, app))
		t.Log("fresh-process App LOAD independent of NewGame -> same TALK/GATES/cancel/retry/mission10/pointer action/ticks")
		return
	}
	out := secondMissionSaveDirectory(t)
	for _, inn := range []bool{false, true} {
		for _, unlocked := range []bool{false, true} {
			t.Run(fmt.Sprintf("inn=%v/unlocked=%v", inn, unlocked), func(t *testing.T) {
				dir := filepath.Join(out, fmt.Sprintf("%v-%v", inn, unlocked))
				if err := os.Mkdir(dir, 0700); err != nil {
					t.Fatal(err)
				}
				f, app := secondTownNew(t, inn, unlocked)
				secondTownMutate(f)
				before := secondTownSampleNow(t, f, app)
				raw := secondTownNamedSave(t, f, app, dir, "First town")
				secondTownShape(t, raw)
				secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
				cold, a := secondTownCold(t, dir, "First town.sav")
				secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))

				secondTownNamedSave(t, cold, a, dir, "Resaved town")
				again, b := secondTownCold(t, dir, "Resaved town.sav")
				secondTownAssertSample(t, before, secondTownSampleNow(t, again, b))
				secondTownEnter(t, f, app)
				secondTownEnter(t, cold, a)
				assertCurrentWorldEqual(t, f.live.world, cold.live.world, "first-town cold departure")
				first := secondSaveSampleNow(t, f, app)
				departed := secondSaveSampleNow(t, cold, a)
				secondAssertSample(t, first, departed)
				for i, pose := range first.Poses {
					if !bytes.Equal(pose.Name, departed.Poses[i].Name) || !bytes.Equal(pose.Art, departed.Poses[i].Art) {
						t.Fatal("source/cold actual actor bytes differ before JSON proof transport")
					}
				}
				t.Logf("independent source/cold raw Name/Art bytes equal for %d actor descriptors before JSON proof transport", len(first.Poses))
				id := f.live.mission.ids[0]
				e, _ := f.live.world.Entity(id)
				x, y := int(e.X)-1, int(e.Y)
				secondMissionMove(t, app, id, x, y)
				secondMissionMove(t, a, id, x, y)
				for range 8 {
					if err := app.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
					if err := a.HeadlessStep(); err != nil {
						t.Fatal(err)
					}
					assertCurrentWorldEqual(t, f.live.world, cold.live.world, "town continuation ordinary movement")
				}
				last := secondSaveSampleNow(t, f, app)
				secondAssertSample(t, last, secondSaveSampleNow(t, cold, a))
				if bytes.Equal(first.World, last.World) {
					t.Fatal("ordinary move/ticks changed no World")
				}
				proof, err := json.Marshal(secondTownProof{Sample: before, Mission: []secondSaveSample{first, last}, Actor: id, X: x, Y: y, Ticks: 8})
				if err != nil {
					t.Fatal(err)
				}
				path := filepath.Join(dir, "First town.sav")
				if err := os.WriteFile(path+".json", proof, 0600); err != nil {
					t.Fatal(err)
				}
				runSpellWitnessChild(t, path, "AGAINROM_SECOND_TOWN_SAVE_INPUT")

				t.Logf("ordinary square/inn locked/unlocked -> %d-byte sole city SAV -> cold first-frame -> resave", len(raw))
			})
		}
	}
}

func secondTownDismissAck(t *testing.T, app *ui.App) {
	t.Helper()
	if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenLoad {
		t.Fatal("ordinary LOAD window", err, app.Screen())
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() == ui.ScreenGameMenu {
		if err := app.HeadlessGameMenuAction("return"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || app.HeadlessMessage() != "" {
		t.Fatal("ordinary transient acknowledgement dismissal", app.Screen(), app.HeadlessMessage())
	}
}

func TestReleaseSecondTownSaveEntryPoints(t *testing.T) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondTownNew(t, true, true)
	secondTownMutate(f)
	f.Options = OptionsStore{Path: filepath.Join(out, "options.txt")}
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, func() time.Time { return now })
	defer app.FlushBackground()
	before := secondTownSampleNow(t, f, app)
	if err := app.HeadlessKey("f4"); err != nil {
		t.Fatal(err)
	}
	cold, a := secondTownCold(t, out, string(quickSaveBase(0))+".sav")
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	f.Town.gold++
	f.Town.second.bank[997]++
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	secondTownDismissAck(t, app)
	secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
	save, _, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		t.Fatal(err)
	}
	cold, a = secondTownCold(t, out, name)
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	snapshot, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := f.ExportCurrentSave(snapshot, "same producer")
	if err != nil {
		t.Fatal(err)
	}
	for _, contradiction := range []string{"World", "pending notices", "fog"} {
		bad := snapshot
		switch contradiction {
		case "World":
			bad.World = []byte{1}
		case "pending notices":
			bad.Residue.PendingMessages = []int32{2, 3}
		case "fog":
			bad.Residue.FogVisible = []byte{1}
		}
		if raw, err := f.ExportCurrentSave(bad, "contradictory town"); err == nil || len(raw) != 0 {
			t.Fatal("city producer discarded mission carrier", contradiction, err)
		}
	}
	for _, export := range []func(Snapshot, string) ([]byte, error){f.ExportCurrentWorldSave, f.ExportNativeCitySave, f.ExportOriginalSave} {
		raw, err := export(snapshot, "same producer")
		if err != nil || !bytes.Equal(reference, raw) {
			t.Fatal("city compatibility API differs", err)
		}
	}
	view, detached := f.detachedExporter(snapshot)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	f.Town.second.bank[997]++
	f.Carried[0].Name += " changed"
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	raw, err := view.ExportCurrentSave(detached, "same producer")
	if err != nil || !bytes.Equal(raw, reference) {
		t.Fatal("detached exporter read later live room/party/TALK/bank", err)
	}
	if _, _, err := f.RestoreOriginal(reference); err != nil {
		t.Fatal(err)
	}
	if _, open := f.TownScreen().(ui.TownDialogueScreen).TownDialogue(); open || f.Town.second.selected != (secondLocation{}) {
		t.Fatal("valid LOAD retained stale modal")
	}
	if err := app.HeadlessKey("f9"); err != nil {
		t.Fatal(err)
	}
	secondTownDismissAck(t, app)
	secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
	for _, target := range []string{"GATES", "mission 10"} {
		if err := app.HeadlessActivate(target); err != nil {
			t.Fatal(err)
		}
	}
	if f.Town.second.selected == (secondLocation{}) {
		t.Fatal("destination was not selected")
	}
	if err := app.HeadlessKey("f3"); err != nil || app.Screen() != ui.ScreenLoad {
		t.Fatal("ordinary LOAD from selected destination", err, app.Screen())
	}
	_, list, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	label := ""
	for _, row := range list() {
		if row.Name == localOriginalSaveToken(string(quickSaveBase(0))+".sav") {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("quick town absent from ordinary list")
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if f.Town.second.selected != (secondLocation{}) {
		t.Fatal("ordinary named LOAD retained stale selection")
	}
	secondTownDismissAck(t, app)
	secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	cold, a = secondTownCold(t, out, string(timedSaveBase(0))+".sav")
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	t.Log("F4/cold App LOAD; F9; SaveSeams; compat exporters; detached room/TALK/party/bank; timed city autosave/cold LOAD; stale modal/selection cleared")
}

func secondTownChangeJSON(t *testing.T, raw []byte, change func(map[string]any)) []byte {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := json.Marshal(a)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(leaf, &fields); err != nil {
		t.Fatal(err)
	}
	change(fields)
	leaf, err = json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func TestReleaseSecondTownSaveLossControls(t *testing.T) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondTownNew(t, false, false)
	secondTownMutate(f)
	raw := secondTownNamedSave(t, f, app, out, "Controls")
	for _, loss := range []string{"bank short", "bank long", "bank absent", "bank overflow", "duplicate", "foreign", "missing list", "missing current", "foreign current", "stage", "room", "missing room", "game absent", "game wrong", "game unknown", "party absent", "city absent", "base absent", "program", "policy", "fog", "mission gold", "return", "pending", "options", "visual next", "difficulty range", "difficulty conflict"} {
		t.Run(loss, func(t *testing.T) {
			changed := secondTownChangeJSON(t, raw, func(a map[string]any) {
				s := a["Session"].(map[string]any)
				c := s["Second"].(map[string]any)
				switch loss {
				case "bank short":
					c["Bank"] = c["Bank"].([]any)[:1023]
				case "bank long":
					c["Bank"] = append(c["Bank"].([]any), 0)
				case "bank absent":
					delete(c, "Bank")
				case "bank overflow":
					c["Bank"].([]any)[997] = float64(1 << 32)
				case "duplicate":
					c["Available"] = append(c["Available"].([]any), map[string]any{"Kind": 2, "ID": 1})
				case "foreign":
					c["Available"] = append(c["Available"].([]any), map[string]any{"Kind": 1, "ID": 20})
				case "missing list":
					delete(c, "Available")
				case "missing current":
					delete(c, "Current")
				case "foreign current":
					c["Current"] = map[string]any{"Kind": 2, "ID": 2}
				case "stage":
					c["Bank"].([]any)[768] = 20
				case "room":
					c["Room"] = 8
				case "missing room":
					delete(c, "Room")
				case "game absent":
					delete(s, "Game")
				case "game wrong":
					s["Game"] = "rom1"
				case "game unknown":
					s["Game"] = "unknown"
				case "party absent":
					delete(a, "Party")
				case "city absent":
					delete(a["Party"].([]any)[0].(map[string]any), "City")
				case "base absent":
					delete(a["Party"].([]any)[0].(map[string]any), "Base")
				case "program":
					a["Program"] = map[string]any{"Dialect": sim.ScriptROM2}
				case "policy":
					a["Policy"] = map[string]any{}
				case "fog":
					a["Fog"] = map[string]any{"Cols": 1, "Rows": 1, "Explored": []byte{0}, "Visible": []byte{0}}
				case "mission gold":
					s["MissionGold"] = 1
				case "return":
					a["WorldMapReturn"] = map[string]any{}
				case "pending":
					a["PendingMessages"] = []int{2, 3}
				case "options":
					a["Options"] = []map[string]any{{"ID": 1}}
				case "visual next":
					a["VisualNext"] = 1
				case "difficulty range":
					s["Difficulty"] = 4
				case "difficulty conflict":
					s["Difficulty"] = 1
				}
			})
			before := secondTownSampleNow(t, f, app)
			oldTown, oldLive := f.Town, f.live
			if _, _, err := f.RestoreOriginal(changed); err == nil {
				t.Fatal("invalid town continuation accepted")
			}
			if oldTown != f.Town || oldLive != f.live {
				t.Fatal("invalid town adopted before refusal")
			}
			secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
		})
	}
	for _, loss := range []string{"bank", "room", "availability"} {
		t.Run("detect "+loss, func(t *testing.T) {
			changed := secondChangeSave(t, raw, func(a *currentActionData) {
				switch loss {
				case "bank":
					a.Session.Second.Bank[997]++
				case "room":
					*a.Session.Second.Room = secondTownInn
				case "availability":
					a.Session.Second.Available = append(a.Session.Second.Available, currentSecondLocation{1, 10})
				}
			})
			name := loss + ".sav"
			if err := os.WriteFile(filepath.Join(out, name), changed, 0600); err != nil {
				t.Fatal(err)
			}
			cold, _ := secondTownCold(t, out, name)
			if reflect.DeepEqual(captureSecondCampaign(f.Town.second), captureSecondCampaign(cold.Town.second)) {
				t.Fatal("independent loss control invisible")
			}
		})
	}
	firstRoot := os.Getenv("AGAINROM_FIRST_ASSETS")
	if firstRoot == "" {
		t.Fatal("AGAINROM_FIRST_ASSETS required for town cross-game control")
	}
	first, err := NewFrontEnd(firstRoot)
	if err != nil {
		t.Fatal(err)
	}
	cleanupFrontAudio(t, first)
	first.SetDeterministicFrames(true)
	first.Options = OptionsStore{}
	a := first.App("first game city identity")
	a.Layout(1024, 768)
	if err := a.OpenMission(first.DirectNewGame(10)); err != nil {
		t.Fatal(err)
	}
	firstSnapshot, _, err := first.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	firstRaw, err := first.ExportCurrentSave(firstSnapshot, "first game")
	if err != nil {
		t.Fatal(err)
	}
	oldFirst, hash := first.live, first.live.world.Hash()
	if _, _, err := first.RestoreOriginal(raw); err == nil {
		t.Fatal("ROM2 city admitted into ROM1")
	}
	if first.live != oldFirst || first.live.world.Hash() != hash {
		t.Fatal("cross-base refusal changed ROM1")
	}
	before := secondTownSampleNow(t, f, app)
	oldTown := f.Town
	if _, _, err := f.RestoreOriginal(firstRaw); err == nil {
		t.Fatal("ROM1 admitted into ROM2 city")
	}
	if f.Town != oldTown {
		t.Fatal("cross-base refusal changed ROM2")
	}
	secondTownAssertSample(t, before, secondTownSampleNow(t, f, app))
	historical := secondChangeSave(t, firstRaw, func(a *currentActionData) { a.Session.Game = "" })
	if _, _, err := first.RestoreOriginal(historical); err != nil {
		t.Fatal("historical absent ROM1 marker", err)
	}
	t.Log("28 atomic malformed/loss/shape/game controls; independent bank/room/list controls; cross-base rejection; historical ROM1 marker default")
}

func TestReleaseSecondTownModalSaveBoundary(t *testing.T) {
	secondGameRoot(t)
	out := secondMissionSaveDirectory(t)
	f, app := secondTownNew(t, true, true)
	f.Options = OptionsStore{Path: filepath.Join(out, "options.txt")}
	now := time.Unix(100, 0)
	f.ConfigureSaveSeams(app, SaveStore{Dir: out}, OriginalStore{}, func() time.Time { return now })
	defer app.FlushBackground()
	quiet, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("TALK 517"); err != nil {
		t.Fatal(err)
	}
	c := f.Town.second
	payload, part := bytes.Clone(c.payload), c.part
	if len(c.available) != 2 || len(payload) == 0 {
		t.Fatal("already-unlocked TALK control absent")
	}
	if _, _, err := f.Snapshot(false); err == nil {
		t.Fatal("capture discarded open TALK")
	}
	blocked := quiet
	snapshotTown(f.Town, &blocked)
	for _, export := range []func(Snapshot, string) ([]byte, error){f.ExportCurrentSave, f.ExportCurrentWorldSave, f.ExportNativeCitySave, f.ExportOriginalSave} {
		if raw, err := export(blocked, "blocked"); err == nil || len(raw) != 0 {
			t.Fatal("producer discarded modal", err)
		}
	}
	save, _, _ := f.SaveSeams(SaveStore{Dir: out}, OriginalStore{}, nil)
	if _, err := save(false); err == nil {
		t.Fatal("SaveSeams admitted modal")
	}
	dialog := f.SaveDialogSeams(SaveStore{Dir: out}, OriginalStore{})
	if _, err := dialog.Prepare(ui.SaveRequest{Directory: out, Name: "Blocked", Format: ui.SaveSAV, OnMap: false}); err == nil {
		t.Fatal("named preparation admitted modal")
	}
	for _, key := range []string{"f2", "f4", "f9"} {
		if err := app.HeadlessKey(key); err != nil {
			t.Fatal(err)
		}
		if app.Screen() != ui.ScreenTown {
			t.Fatal("modal allowed SAVE/quick input", app.Screen())
		}
	}
	now = now.Add(5 * time.Minute)
	if err := app.HeadlessStep(); err != nil {
		t.Fatal(err)
	}
	app.FlushBackground()
	if !bytes.Equal(c.payload, payload) || c.part != part || !reflect.DeepEqual(c.available, []secondLocation{{2, 1}, {1, 10}}) {
		t.Fatal("refusal changed TALK modal/unlock")
	}
	if files, err := os.ReadDir(out); err != nil || len(files) != 0 {
		t.Fatal("modal refusal left partial files", files, err)
	}
	for page := 0; len(c.payload) != 0 && page < 64; page++ {
		if err := app.HeadlessActivate("notice"); err != nil {
			t.Fatal(err)
		}
	}
	if !f.Town.second.savePoint() {
		t.Fatal("closed TALK not saveable")
	}
	before := secondTownSampleNow(t, f, app)
	secondTownNamedSave(t, f, app, out, "Closed TALK")
	cold, a := secondTownCold(t, out, "Closed TALK.sav")
	secondTownAssertSample(t, before, secondTownSampleNow(t, cold, a))
	t.Log("already-unlocked open TALK refuses capture/sole producer/aliases/named preparation/seam/F2/F4/F9/timed without partial file or discarded modal; closed TALK SAV cold LOAD")
}
