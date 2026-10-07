//go:build sessioncorpusaudit

package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

const townReturnInputSHA = "60267c82072c77446ab9b34913318e89eab8f70e49f3510ae64aaaf423819bd6"

type townReturnWire struct {
	Mission                         uint32
	Money                           uint32
	Outcome                         byte
	PlayerOff, MoneyOff, OutcomeOff int
	HasWorld                        bool
	Won, Lost                       uint32
	Campaign                        sav.CampaignProjection
}

type townReturnMember struct {
	ID   string
	Worn [sim.EquipSlots]uint16
	Pack []uint16
}

type townReturnSample struct {
	Chapter, Selected, Gold int
	Done20, AtSquare        bool
	Available               []int
	Documents               []Document
	Members                 []HeadlessMemberSnapshot
	Items                   []townReturnMember
	Campaign                sav.CampaignProjection
}

type townReturnProof struct {
	Stage, SHA, WorldSHA string
	WorldHash            uint64
	Town, Next           townReturnSample
}

func townReturnOutput(t *testing.T, leaf string) string {
	t.Helper()
	root := os.Getenv("AGAINROM_TOWN_RETURN_OUTPUT")
	if root == "" {
		root = t.TempDir()
		t.Setenv("AGAINROM_TOWN_RETURN_OUTPUT", root)
	}
	if !filepath.IsAbs(root) {
		t.Fatal("explicit output directory required")
	}
	return filepath.Join(root, leaf)
}

func townReturnJSON(t *testing.T, leaf string, value any) {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(townReturnOutput(t, leaf), raw, 0600); err != nil {
		t.Fatal(err)
	}
	lossless, err := townReturnEncode(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(townReturnOutput(t, leaf+".bytes.json"), lossless, 0600); err != nil {
		t.Fatal(err)
	}
}

func townReturnRead(t *testing.T, raw []byte) townReturnWire {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal("not an ordinary SAV", err)
	}
	objects, err := f.DocumentObjectLocations()
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Body) < 9 || f.Body[8] == 255 {
		t.Fatal("unsupported raw head CString")
	}
	missionOff := 9 + int(f.Body[8]) + 44
	if missionOff > len(f.Body)-4 {
		t.Fatal("short raw head")
	}
	out := townReturnWire{Mission: binary.LittleEndian.Uint32(f.Body[missionOff:]), HasWorld: f.World != nil}
	humans := 0
	for _, object := range objects {
		if object.Class != "Player" {
			continue
		}
		at := object.Off
		if at < 0 || at >= len(f.Body) || f.Body[at] == 255 {
			t.Fatal("unsupported Player CString")
		}
		at += 1 + int(f.Body[at])
		if at > len(f.Body)-26 {
			t.Fatal("short Player fields")
		}
		if uint32(binary.LittleEndian.Uint16(f.Body[at:])) != binary.LittleEndian.Uint32(f.Body[at+2:]) {
			t.Fatal("raw Player slot anchors differ")
		}
		if binary.LittleEndian.Uint32(f.Body[at+15:]) != 0 {
			continue
		}
		humans++
		out.PlayerOff, out.MoneyOff, out.OutcomeOff = object.Off, at+21, at+25
		out.Money = binary.LittleEndian.Uint32(f.Body[at+21:]) ^ 0x5c073f4d
		out.Outcome = f.Body[at+25]
	}
	if humans != 1 {
		t.Fatalf("raw human Player population=%d", humans)
	}
	if out.HasWorld {
		at := f.World.SessionOff
		if at < 0 || at > len(f.Body)-4374 {
			t.Fatal("invalid session bounds")
		}
		out.Won = binary.LittleEndian.Uint32(f.Body[at+4362:])
		out.Lost = binary.LittleEndian.Uint32(f.Body[at+4370:])
	}
	var present bool
	out.Campaign, present, err = f.Campaign()
	if err != nil || !present {
		t.Fatal("full Campaign decode", present, err)
	}
	return out
}

func townReturnCapture(t *testing.T, f *FrontEnd) townReturnSample {
	t.Helper()
	snapshot, _, err := f.Snapshot(false)
	if err != nil || !snapshot.CampaignState {
		t.Fatal("town campaign view", err)
	}
	out := townReturnSample{Chapter: f.Town.Chapter(), Selected: f.Town.selectedMission(), Gold: f.Town.Gold(),
		Done20: f.Town.Done(20), AtSquare: f.townUI.AtTownSquare(), Available: f.Town.Available(), Documents: f.Town.Documents(),
		Members: f.HeadlessSnapshot(ui.ScreenTown).Members, Campaign: snapshot.Campaign}
	for _, member := range f.Carried {
		row := townReturnMember{ID: member.ID}
		for i, item := range mapload.MemberItemEquipment(member, f.Table) {
			row.Worn[i] = item.Code
		}
		for _, item := range mapload.MemberCarriedItems(member, f.Table) {
			row.Pack = append(row.Pack, item.Code)
		}
		out.Items = append(out.Items, row)
	}
	return out
}

func townReturnWorld(t *testing.T, f *FrontEnd) (uint64, string) {
	t.Helper()
	raw, err := f.live.world.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return f.live.world.Hash(), fmt.Sprintf("%x", sha256.Sum256(raw))
}

func townReturnMissionSave(t *testing.T, f *FrontEnd) string {
	t.Helper()
	dir := townReturnOutput(t, "mission-save")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	hash, worldSHA := townReturnWorld(t, f)
	prepared, err := f.SaveDialogSeams(SaveStore{Dir: dir}, OriginalStore{}).Prepare(ui.SaveRequest{OnMap: true, Directory: dir, Name: "Victory mission", Format: ui.SaveSAV})
	if err != nil {
		t.Fatal("ordinary mission SAVE prepare", err)
	}
	paths, err := prepared.Commit(false)
	if err != nil || len(paths) != 1 || !IsOriginal(filepath.Base(paths[0])) {
		t.Fatal("ordinary mission SAVE commit", paths, err)
	}
	if gotHash, gotSHA := townReturnWorld(t, f); gotHash != hash || gotSHA != worldSHA {
		t.Fatal("mission SAVE mutated World")
	}
	raw, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	wire := townReturnRead(t, raw)
	townReturnJSON(t, "mission-wire.json", wire)
	if wire.Mission != 20 || !wire.HasWorld || wire.Money != 600 || wire.Outcome != 1 || wire.Won != 1 || wire.Lost != 0 {
		t.Fatalf("mission raw values: %+v", wire)
	}
	proof := townReturnProof{Stage: "mission", SHA: fmt.Sprintf("%x", sha256.Sum256(raw)), WorldSHA: worldSHA, WorldHash: hash}
	townReturnWriteProof(t, paths[0], proof)
	return paths[0]
}

func townReturnStrings(value reflect.Value, decode bool) (reflect.Value, error) {
	out := reflect.New(value.Type()).Elem()
	switch value.Kind() {
	case reflect.String:
		if decode {
			raw, err := base64.StdEncoding.Strict().DecodeString(value.String())
			if err != nil {
				return reflect.Value{}, err
			}
			out.SetString(string(raw))
		} else {
			out.SetString(base64.StdEncoding.EncodeToString([]byte(value.String())))
		}
	case reflect.Pointer:
		if !value.IsNil() {
			child, err := townReturnStrings(value.Elem(), decode)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Set(reflect.New(value.Type().Elem()))
			out.Elem().Set(child)
		}
	case reflect.Struct:
		for i := 0; i < value.NumField(); i++ {
			if value.Type().Field(i).PkgPath != "" {
				return reflect.Value{}, fmt.Errorf("unexported proof field %s", value.Type().Field(i).Name)
			}
			child, err := townReturnStrings(value.Field(i), decode)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Field(i).Set(child)
		}
	case reflect.Slice, reflect.Array:
		if value.Kind() == reflect.Slice {
			if value.IsNil() {
				return out, nil
			}
			out.Set(reflect.MakeSlice(value.Type(), value.Len(), value.Len()))
		}
		for i := 0; i < value.Len(); i++ {
			child, err := townReturnStrings(value.Index(i), decode)
			if err != nil {
				return reflect.Value{}, err
			}
			out.Index(i).Set(child)
		}
	case reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		out.Set(value)
	default:
		return reflect.Value{}, fmt.Errorf("unsupported proof field type %s", value.Type())
	}
	return out, nil
}

func townReturnEncode(value any) ([]byte, error) {
	encoded, err := townReturnStrings(reflect.ValueOf(value), false)
	if err != nil {
		return nil, err
	}
	return json.MarshalIndent(encoded.Interface(), "", "  ")
}

func townReturnDecode(raw []byte) (townReturnProof, error) {
	var proof townReturnProof
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&proof); err != nil {
		return proof, err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return proof, fmt.Errorf("trailing proof content: %v", err)
	}
	decoded, err := townReturnStrings(reflect.ValueOf(proof), true)
	if err != nil {
		return proof, err
	}
	return decoded.Interface().(townReturnProof), nil
}

func townReturnWriteProof(t *testing.T, path string, proof townReturnProof) {
	t.Helper()
	raw, err := townReturnEncode(proof)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := townReturnDecode(raw)
	if err != nil || !reflect.DeepEqual(decoded, proof) {
		t.Fatal("proof transport changed full source state", err)
	}
	if err := os.WriteFile(path+".proof.bytes.json", raw, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestMilestone2TownReturnProofBytes(t *testing.T) {
	const raw = "\x80\x81\x9f\xa0\xe0\xf0\xff\x00ASCII"
	sample := townReturnSample{
		Chapter: -1, Selected: 131, Gold: 1100, Done20: true, AtSquare: true,
		Available: []int{}, Documents: nil,
		Members: []HeadlessMemberSnapshot{{
			ID: raw, Name: raw, Membership: "", XP: -1,
			Appearance: HeadlessAppearance{Body: raw, BodyDir: "\ufffd", FigureDir: "text"},
			Weapon:     &HeadlessWeaponSnapshot{Code: 65535, Name: raw, Defense: -1},
			Worn:       []HeadlessWornSnapshot{{Name: raw, Info: []string{raw, "", "\ufffd"}}, {Info: []string{}}, {Info: nil}},
			Skills:     []HeadlessSkillSnapshot{{Name: raw, XP: -1}},
		}, {ID: "second", Weapon: nil, Worn: []HeadlessWornSnapshot{}, Skills: nil}},
		Items:    []townReturnMember{{ID: raw, Pack: []uint16{}}, {ID: raw, Pack: nil}},
		Campaign: sav.CampaignProjection{Mercenaries: []uint16{}, InnNPC: nil, SelectedMission: ^uint32(0)},
	}
	proof := townReturnProof{Stage: "town", SHA: raw, WorldSHA: raw, WorldHash: ^uint64(0), Town: sample, Next: sample}
	legacy, err := json.Marshal(proof)
	if err != nil {
		t.Fatal(err)
	}
	var lossy townReturnProof
	if err := json.Unmarshal(legacy, &lossy); err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(lossy, proof) || lossy.Town.Members[0].Name == raw {
		t.Fatal("literal must expose the JSON raw-string loss")
	}
	encoded, err := townReturnEncode(proof)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := townReturnDecode(encoded)
	if err != nil || !reflect.DeepEqual(decoded, proof) {
		t.Fatal("lossless proof round trip", err)
	}
	if decoded.Town.Members[0].Name != raw || !bytes.Equal([]byte(decoded.Next.Members[0].Worn[0].Info[0]), []byte{0x80, 0x81, 0x9f, 0xa0, 0xe0, 0xf0, 0xff, 0, 'A', 'S', 'C', 'I', 'I'}) {
		t.Fatal("literal bytes changed")
	}
	if proof.Town.Members[0].Name != raw || proof.Town.Members[0].Weapon.Name != raw {
		t.Fatal("encoding mutated source strings")
	}
	decoded.Town.Members[0].Name = "changed"
	decoded.Next.Members[0].Worn[0].Info[0] = "changed"
	if proof.Town.Members[0].Name != raw || proof.Next.Members[0].Worn[0].Info[0] != raw {
		t.Fatal("decoded proof aliases source")
	}
	if _, err := townReturnDecode([]byte(`{"Stage":"!"}`)); err == nil {
		t.Fatal("malformed base64 accepted")
	}
	if _, err := townReturnDecode(append(encoded, []byte(`{}`)...)); err == nil {
		t.Fatal("trailing proof accepted")
	}
}

func townReturnSave(t *testing.T, f *FrontEnd, app *ui.App, stage string) string {
	t.Helper()
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
		t.Fatal("town SAVE requires settled square")
	}
	dir := townReturnOutput(t, stage)
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: dir}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	before := townReturnCapture(t, f)
	raw := cityRosterF2Save(t, app, store, "Returned town")
	if after := townReturnCapture(t, f); !reflect.DeepEqual(before, after) {
		t.Fatal("town SAVE mutated current state")
	}
	wire := townReturnRead(t, raw)
	townReturnJSON(t, stage+"-wire.json", wire)
	if wire.Mission != 0 || wire.HasWorld || wire.Money != uint32(before.Gold) || !reflect.DeepEqual(wire.Campaign, before.Campaign) {
		t.Fatalf("town raw fields/current campaign disagree: %+v", wire)
	}
	if app.Screen() != ui.ScreenGameMenu {
		t.Fatalf("town SAVE destination=%s, want game menu", app.Screen())
	}
	if err := app.HeadlessGameMenuAction("return"); err != nil {
		t.Fatal("Return to Game after town SAVE", err)
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
		t.Fatal("Return to Game did not restore settled town")
	}
	if resumed := townReturnCapture(t, f); !reflect.DeepEqual(before, resumed) {
		t.Fatal("Return to Game changed town state")
	}
	return filepath.Join(dir, "Returned town.sav")
}

func townReturnNext(t *testing.T, f *FrontEnd, app *ui.App) townReturnSample {
	t.Helper()
	if err := app.HeadlessActivate("SHOP"); err != nil {
		t.Fatal("next action SHOP", err)
	}
	for n := 0; n < 128 && f.townUI.room == roomTalk; n++ {
		if err := app.HeadlessActivate("dialogue"); err != nil {
			t.Fatal("shop entry dialogue", err)
		}
	}
	if f.townUI.room != roomShop {
		t.Fatal("shop entry did not settle")
	}
	screen := f.townUI
	screen.CloseTip()
	if screen.shopMemberIndex() != 0 || len(f.Carried) == 0 {
		t.Fatal("next action first-member binding")
	}
	before := townReturnCapture(t, f)
	code := before.Items[0].Worn[0]
	if code == 0 {
		t.Fatal("next action has no equipped weapon")
	}
	x, y, err := app.HeadlessShopPoint("doll", 1)
	if err != nil {
		t.Fatal(err)
	}
	tx, ty, err := app.HeadlessShopPoint("pack", 0)
	if err != nil {
		t.Fatal(err)
	}
	portraitInputDrag(t, app, image.Pt(x, y), image.Pt(tx, ty))
	changed := townReturnCapture(t, f)
	count := func(items []uint16) int {
		n := 0
		for _, item := range items {
			if item == code {
				n++
			}
		}
		return n
	}
	if changed.Items[0].Worn[0] != 0 || count(changed.Items[0].Pack) != count(before.Items[0].Pack)+1 || changed.Gold != before.Gold {
		t.Fatal("next pointer action failed to move weapon to pack without payment")
	}
	for n := 0; n < 8 && !f.townUI.AtTownSquare(); n++ {
		if err := app.HeadlessKey("escape"); err != nil {
			t.Fatal(err)
		}
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
		t.Fatal("next action did not return to town square")
	}
	return townReturnCapture(t, f)
}

func townReturnChild(t *testing.T, path string) {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := os.ReadFile(path + ".proof.bytes.json")
	if err != nil {
		t.Fatal(err)
	}
	proof, err := townReturnDecode(encoded)
	if err != nil || proof.SHA != fmt.Sprintf("%x", sha256.Sum256(raw)) {
		t.Fatal("invalid child input/proof", err)
	}
	f, app := missionHandoffLoad(t, "town-return-"+proof.Stage+"-cold", path)
	if proof.Stage == "town" {
		if app.Screen() != ui.ScreenTown || !f.townUI.AtTownSquare() {
			t.Fatal("cold town did not settle")
		}
		got := townReturnCapture(t, f)
		townReturnJSON(t, "cold-town.json", got)
		if !reflect.DeepEqual(got, proof.Town) {
			t.Errorf("cold town changed current fields or unmodified character presentation; see cold-town.json and proof")
		}
		for range 32 {
			if err := app.HeadlessStep(); err != nil {
				t.Fatal(err)
			}
		}
		if after := townReturnCapture(t, f); !reflect.DeepEqual(after, got) {
			t.Fatal("cold town idle repeated payout or changed state")
		}
		next := townReturnNext(t, f, app)
		townReturnJSON(t, "cold-next-action.json", next)
		if !reflect.DeepEqual(next, proof.Next) {
			t.Errorf("next action differs from source branch; see cold-next-action.json and proof")
		}
		townReturnSave(t, f, app, "next-town-save")
		if !t.Failed() {
			t.Logf("TOWN-COLD-PASS pid=%d", os.Getpid())
		}
		return
	}
	if proof.Stage != "mission" || app.Screen() != ui.ScreenMap {
		t.Fatal("invalid mission continuation")
	}
	hash, worldSHA := townReturnWorld(t, f)
	if hash != proof.WorldHash || worldSHA != proof.WorldSHA {
		t.Fatal("cold mission changed canonical World")
	}
	if won, lost := f.live.world.ScriptCounters(); f.live.world.Outcome() != sim.OutcomeWon || won != 1 || lost != 0 || f.live.world.Purse(sim.SelfSlot) != 600 {
		t.Fatal("saved victory did not restore")
	}
	for range 64 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if h, s := townReturnWorld(t, f); h != hash || s != worldSHA {
		t.Fatal("unacknowledged Victory changed World")
	}
	if _, kind, open := f.LiveNotice(); !open || kind != ui.NoticeSuccess {
		t.Fatal("Victory notice absent")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || !f.townUI.AtWorldMap() || !f.townUI.WorldMapView().Returning {
		t.Fatal("Victory did not start automatic return")
	}
	for n := 0; n < 4000 && !f.townUI.AtTownSquare(); n++ {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if !f.townUI.AtTownSquare() || !f.Town.Done(20) || f.Campaign.Value().Reward(20) != 500 || f.Town.Gold() != 1100 || len(f.Carried) != 2 {
		t.Fatal("automatic return/payout/roster differs")
	}
	town := townReturnCapture(t, f)
	for range 32 {
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if after := townReturnCapture(t, f); !reflect.DeepEqual(after, town) {
		t.Fatal("town arrival repeated payout or changed characters")
	}
	townReturnJSON(t, "source-town.json", town)
	townPath := townReturnSave(t, f, app, "town-save")
	next := townReturnNext(t, f, app)
	townReturnJSON(t, "source-next-action.json", next)
	townRaw, err := os.ReadFile(townPath)
	if err != nil {
		t.Fatal(err)
	}
	townReturnWriteProof(t, townPath, townReturnProof{Stage: "town", SHA: fmt.Sprintf("%x", sha256.Sum256(townRaw)), Town: town, Next: next})
	runSpellWitnessChild(t, townPath, "AGAINROM_TOWN_RETURN_CHILD")
	t.Logf("TOWN-MISSION-PASS pid=%d", os.Getpid())
}

func TestMilestone2TownReturnCurrentSAV(t *testing.T) {
	if os.Getenv("AGAINROM_ASSETS") == "" {
		t.Fatal("AGAINROM_ASSETS must name the explicit lawful install to resume through")
	}
	if path := os.Getenv("AGAINROM_TOWN_RETURN_CHILD"); path != "" {
		townReturnChild(t, path)
		return
	}
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Fatal("AGAINROM_SAVE_CORPUS must name owner saves")
	}
	raw, err := os.ReadFile(filepath.Join(corpus, "2026-08-02", "game0009.sav"))
	if err != nil || fmt.Sprintf("%x", sha256.Sum256(raw)) != townReturnInputSHA {
		t.Fatal("wrong saved-victory input", err)
	}
	wire := townReturnRead(t, raw)
	townReturnJSON(t, "owner-wire.json", wire)
	frozen, err := sav.Open(raw)
	if err != nil || len(frozen.Body) != 83732 || frozen.Body[119] != 1 || binary.LittleEndian.Uint32(frozen.Body[109:]) != 0 || binary.LittleEndian.Uint32(frozen.Body[81606:]) != 1 || binary.LittleEndian.Uint32(frozen.Body[81614:]) != 0 || frozen.Body[77651] != 1 {
		t.Fatal("frozen source anchors changed", err)
	}
	if wire.Mission != 20 || wire.Money != 600 || wire.Outcome != 1 || wire.Won != 1 || wire.Lost != 0 {
		t.Fatal("owner source raw fields differ")
	}
	dir := townReturnOutput(t, "input")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "game0009.sav")
	if err := os.WriteFile(path, slices.Clone(raw), 0600); err != nil {
		t.Fatal(err)
	}
	f, app := missionHandoffLoad(t, "town-return-owner", path)
	if app.Screen() != ui.ScreenMap {
		t.Fatal("owner mission did not open")
	}
	missionPath := townReturnMissionSave(t, f)
	runSpellWitnessChild(t, missionPath, "AGAINROM_TOWN_RETURN_CHILD")
	t.Logf("TOWN-PARENT-PASS pid=%d input-sha=%s", os.Getpid(), townReturnInputSHA)
}
