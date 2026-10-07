package sav

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	cityTestPlayerKey   = uint32(0x20000100)
	cityTestReniestaKey = uint32(0x20000110)
	cityTestDanathKey   = uint32(0x20000120)
)

func cityTestToken(identity, owner uint32, defRow byte) []byte {
	b := make([]byte, 37)
	b[4], b[5] = 0x80, 0x80
	binary.LittleEndian.PutUint32(b[12:16], identity>>4)
	b[16] = defRow
	binary.LittleEndian.PutUint16(b[17:19], 0x21)
	binary.LittleEndian.PutUint32(b[29:33], identity)
	binary.LittleEndian.PutUint32(b[33:37], owner)
	return b
}

func cityTestHuman(index uint16, identity uint32, name string, defRow byte, body uint16) *cityObject {
	scalar2 := make([]byte, 55)
	binary.LittleEndian.PutUint16(scalar2[0:2], body)
	binary.LittleEndian.PutUint16(scalar2[2:4], body+1)
	binary.LittleEndian.PutUint16(scalar2[16:18], 100)
	binary.LittleEndian.PutUint16(scalar2[18:20], 100)
	binary.LittleEndian.PutUint16(scalar2[22:24], 50)
	binary.LittleEndian.PutUint16(scalar2[24:26], 50)
	rawA6 := make([]byte, 24)
	xp := make([]byte, 24)
	for i := 0; i < CharacterSkillSlots; i++ {
		binary.LittleEndian.PutUint16(rawA6[2+2*i:], uint16(i+1))
		binary.LittleEndian.PutUint32(xp[4*i:], uint32(100+i))
	}
	binary.LittleEndian.PutUint32(scalar2[35:39], 615)
	return &cityObject{sourceIndex: index, class: "Human", unit: &cityUnit{
		token: cityTestToken(identity, cityTestPlayerKey, defRow),
		rawA6: rawA6, rawBE: make([]byte, 22), raw114: make([]byte, 24),
		rawD4: make([]byte, 64), raw154: make([]byte, 180), raw158: make([]byte, 148),
		scalar1: make([]byte, 19), name: name, scalar2: scalar2,
		scalarTail: make([]byte, 17), xp: xp, equipment: make([]*cityObject, 13),
	}}
}

func cityTestState(name string) *cityState {
	state := &cityState{rootKind: 17, directoryKinds: make(map[string]uint32, len(cityStateShape)), values: make(map[string]cityStateValue, 15)}
	for _, directory := range cityStateShape {
		state.directoryKinds["/"+directory.directory] = 1
		for _, child := range directory.children {
			path := "/" + directory.directory + "/" + child.name
			value := cityStateValue{kind: child.kind}
			if path == "/Character/Name" {
				value.bytes = append(append([]byte(nil), name...), 0)
			}
			if path == "/SpellBook/Shortcuts" {
				value.bytes = make([]byte, 16)
			}
			state.values[path] = value
		}
	}
	return state
}

func cityTestSource(t *testing.T) []byte {
	t.Helper()
	reniesta := cityTestHuman(4, cityTestReniestaKey, "Reniesta", 29, 31)
	danath := cityTestHuman(5, cityTestDanathKey, "Danath", 28, 41)
	fixed := make([]byte, 51)
	binary.LittleEndian.PutUint16(fixed[0:2], 1)
	binary.LittleEndian.PutUint32(fixed[2:6], 1)
	binary.LittleEndian.PutUint32(fixed[15:19], 0)
	binary.LittleEndian.PutUint32(fixed[21:25], 123^obfuscator)
	binary.LittleEndian.PutUint32(fixed[43:47], cityTestDanathKey)
	binary.LittleEndian.PutUint32(fixed[47:51], cityTestPlayerKey)
	player := &cityObject{sourceIndex: 2, class: "Player", player: &cityPlayer{
		name: "Player", fixed: fixed,
		groups: []cityGroup{{raw80: make([]byte, 80), actors: []*cityObject{reniesta, danath}, f44: cityTestPlayerKey}},
		raw32:  make([]byte, 32),
	}}
	document := &cityDocument{
		mapName: "31.alm", playerList: 1, players: []*cityObject{player}, marker: 0xbadface1,
		trailerState: make([]byte, 400), objects: map[uint16]*cityObject{2: player, 4: reniesta, 5: danath},
	}
	document.head[12] = 2 // difficulty
	bodyBytes, err := serializeCityDocument(document)
	if err != nil {
		t.Fatal(err)
	}
	stateBytes, err := serializeCityState(cityTestState("Danath"))
	if err != nil {
		t.Fatal(err)
	}
	campaign := &cityCampaign{}
	campaign.scalars[1] = 0x11111111
	campaign.scalars[6] = 0x66666666
	campaignBytes, err := serializeCityCampaign(campaign)
	if err != nil {
		t.Fatal(err)
	}
	blob := Compress(bodyBytes)
	out := make([]byte, headerLen)
	copy(out, Magic)
	binary.LittleEndian.PutUint32(out[4:8], uint32(headerLen+len(blob)))
	binary.LittleEndian.PutUint32(out[8:12], MinVersion)
	binary.LittleEndian.PutUint32(out[12:16], uint32(len(blob)))
	out = append(out, blob...)
	label := make([]byte, labelLen)
	copy(label, "source")
	out = append(out, label...)
	out = append(out, stateBytes...)
	out = append(out, campaignBytes...)
	return out
}

func cityTestUpdates(roster []CityCharacter) []CityCharacterUpdate {
	out := make([]CityCharacterUpdate, len(roster))
	for i, character := range roster {
		out[i] = CityCharacterUpdate{
			Identity: character.Identity, Name: character.Name, Stats: character.Stats,
			SkillLevels: character.SkillLevels, SkillXP: character.SkillXP, Experience: character.Experience,
		}
	}
	return out
}

// This is a structural serializer test over a synthetic object graph. Its
// changed Human values prove field routing only; they are not evidence that an
// arbitrary invented Human is safe in ROM1 (SAV-ORIGVALUE-399).
func TestCityProvenanceStructurallyRoutesIdentityKeyedRosterAndRemints(t *testing.T) {
	source := cityTestSource(t)
	file, err := Open(source)
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	roster := provenance.Roster()
	if len(roster) != 2 || roster[0].Name != "Reniesta" || roster[1].Name != "Danath" || roster[0].Hero || !roster[1].Hero {
		t.Fatalf("source roster = %#v", roster)
	}
	updates := cityTestUpdates(roster)
	updates[0].Name = "Reniesta live"
	updates[0].Stats[StatBody] = 101
	updates[0].SkillLevels[2] = 12
	updates[0].SkillXP[2] = 1200
	updates[0].Experience = 1700
	updates[1].Name = "Danath live"
	updates[1].Stats[StatBody] = 202
	updates[1].SkillLevels[2] = 22
	updates[1].SkillXP[2] = 2200
	updates[1].Experience = 2700
	updates[0], updates[1] = updates[1], updates[0] // live order differs from the source actor list

	wantUpdate := CityUpdate{Label: []byte("new city"), Money: 0xf1234567, Characters: updates}
	first, err := provenance.Marshal(wantUpdate)
	if err != nil {
		t.Fatal(err)
	}
	// Provenance is detached from every source byte region.
	file.Body[0] ^= 0xff
	file.Store[0] ^= 0xff
	file.TailRest[0] ^= 0xff
	second, err := provenance.Marshal(wantUpdate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same provenance and update produced different saves")
	}

	opened, err := Open(first)
	if err != nil {
		t.Fatal(err)
	}
	if string(opened.Label) != "new city" || opened.Players[0].Money != 0xf1234567 {
		t.Fatalf("label=%q money=%#x", opened.Label, opened.Players[0].Money)
	}
	authored, err := opened.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	got := authored.Roster()
	if len(got) != 2 || got[0].Name != "Reniesta live" || got[1].Name != "Danath live" {
		t.Fatalf("authored roster = %#v", got)
	}
	if got[0].Stats[StatBody] != 101 || got[1].Stats[StatBody] != 202 || got[0].SkillXP[2] != 1200 || got[1].SkillXP[2] != 2200 {
		t.Fatalf("authored live fields = %#v", got)
	}
	if got[0].Identity == cityTestReniestaKey || got[1].Identity == cityTestDanathKey {
		t.Fatalf("source identities were carried: %#x %#x", got[0].Identity, got[1].Identity)
	}
	state, err := parseCityState(opened.Store)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(state.values["/Character/Name"].bytes); got != "Danath live\x00" {
		t.Fatalf("state primary character = %q", got)
	}
}

// This release witness changes no Human field from the lawful original. It
// proves the source-faithful city slice over the explicitly selected lawful
// save corpus; synthetic fixtures never stand in for that provenance.
func TestCityProvenanceLawfulSourceFaithfulRelease(t *testing.T) {
	corpus := os.Getenv("AGAINROM_SAVE_CORPUS")
	if corpus == "" {
		t.Skip("no AGAINROM_SAVE_CORPUS: lawful original city provenance not requested")
	}
	var paths []string
	if err := filepath.WalkDir(corpus, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(path), ".sav") {
			paths = append(paths, path)
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	sort.Strings(paths)
	var (
		file       *File
		provenance *CityProvenance
		path       string
	)
	for _, candidate := range paths {
		raw, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		opened, err := Open(raw)
		if err != nil {
			continue
		}
		city, err := opened.CityProvenance()
		if err != nil {
			continue
		}
		file, provenance, path = opened, city, candidate
		break
	}
	if provenance == nil {
		t.Fatalf("%s has %d .sav files and no fully supported lawful no-world city", corpus, len(paths))
	}
	money := uint32(0)
	foundHuman := false
	for _, player := range file.Players {
		if player.Participant == 0 {
			money, foundHuman = player.Money, true
			break
		}
	}
	if !foundHuman {
		t.Fatal("supported lawful city has no human Player")
	}
	want := provenance.Roster()
	updates := cityTestUpdates(want)
	for left, right := 0, len(updates)-1; left < right; left, right = left+1, right-1 {
		updates[left], updates[right] = updates[right], updates[left]
	}
	raw, err := provenance.Marshal(CityUpdate{Label: []byte("Againrom city"), Money: money, Characters: updates})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	reminted, err := opened.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	got := reminted.Roster()
	if len(got) != len(want) {
		t.Fatalf("reminted roster=%d, source=%d", len(got), len(want))
	}
	for i := range want {
		want[i].Identity, got[i].Identity = 0, 0
		for j := range want[i].Spells {
			want[i].Spells[j].Key, got[i].Spells[j].Key = 0, 0
		}
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Fatalf("source-faithful character %d changed:\n got %#v\nwant %#v", i, got[i], want[i])
		}
	}
	digest := sha256.Sum256(raw)
	t.Logf("lawful city %s -> %d bytes SHA256 %s", filepath.Base(path), len(raw), fmt.Sprintf("%x", digest))
}

func TestCityProvenanceRejectsIncompleteOrAmbiguousRoster(t *testing.T) {
	file, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	valid := cityTestUpdates(provenance.Roster())
	for _, test := range []struct {
		name   string
		update CityUpdate
	}{
		{"missing", CityUpdate{Characters: valid[:1]}},
		{"duplicate", CityUpdate{Characters: []CityCharacterUpdate{valid[0], valid[0]}}},
		{"unknown", CityUpdate{Characters: []CityCharacterUpdate{valid[0], func() CityCharacterUpdate { v := valid[1]; v.Identity = 0xdeadbeef; return v }()}}},
		{"embedded label NUL", CityUpdate{Label: []byte{'a', 0, 'b'}, Characters: valid}},
		{"long label", CityUpdate{Label: bytes.Repeat([]byte{'x'}, labelLen), Characters: valid}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := provenance.Marshal(test.update); err == nil {
				t.Fatal("invalid city update was accepted")
			}
		})
	}
}

func TestCityCampaignOverlayPreservesUnknownScalarAndWritesScoreHistory(t *testing.T) {
	file, err := Open(cityTestSource(t))
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := file.CityProvenance()
	if err != nil {
		t.Fatal(err)
	}
	projection, ok, err := file.Campaign()
	if err != nil || !ok {
		t.Fatalf("Campaign() = ok=%t err=%v", ok, err)
	}
	projection.SelectedMission = 77
	projection.AutoGetMission = 88
	projection.LastMission = 99
	projection.FirstMapPoint = true
	projection.MissionTime = 1234
	projection.ScoreEvents = 0xfedcba98
	projection.MercenaryHired = []bool{true, false, true}
	projection.Markers = []CampaignMarker{{Value: 3, Picture: "marker.bmp", Field0: 4, Field1: 5}}
	raw, err := provenance.Marshal(CityUpdate{Label: []byte("campaign"), Characters: cityTestUpdates(provenance.Roster()), Campaign: &projection})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	full, err := parseCityCampaign(opened.TailRest)
	if err != nil {
		t.Fatal(err)
	}
	if full.scalars[1] != 0x11111111 || full.scalars[6] != 0xfedcba98 {
		t.Fatalf("unknown or score scalar = %#x %#x", full.scalars[1], full.scalars[6])
	}
	if full.scalars[0] != 77 || full.scalars[2] != 88 || full.scalars[3] != 99 || full.scalars[4] != 1 || full.scalars[5] != 1234 {
		t.Fatalf("named scalars = %#v", full.scalars)
	}
	if len(full.markers) != 1 || string(full.markers[0].text) != "marker.bmp\x00" {
		t.Fatalf("markers = %#v", full.markers)
	}
	projection.ScoreEventsKnown = false // old AGS did not carry this wire value
	projection.ScoreEvents = 0
	raw, err = provenance.Marshal(CityUpdate{Label: []byte("legacy"), Characters: cityTestUpdates(provenance.Roster()), Campaign: &projection})
	if err != nil {
		t.Fatal(err)
	}
	opened, err = Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	full, err = parseCityCampaign(opened.TailRest)
	if err != nil || full.scalars[6] != 0x66666666 {
		t.Fatal("legacy projection erased original score history", err)
	}
}

func TestCityStateRequiresOriginalShortcutShape(t *testing.T) {
	state := cityTestState("Danath")
	value := state.values["/SpellBook/Shortcuts"]
	value.bytes = make([]byte, 12)
	state.values["/SpellBook/Shortcuts"] = value
	if _, err := serializeCityState(state); err == nil || !strings.Contains(err.Error(), "want 16") {
		t.Fatalf("serializeCityState error = %v", err)
	}
}
