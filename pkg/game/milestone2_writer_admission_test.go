package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"againrom/internal/cityfixture"
	"againrom/pkg/formats/sav"
)

func milestone2WriterCarriers(mf milestone2File) (uint32, bool, error) {
	store, _ := mf.f.StateStore()
	found := false
	if store != nil && store.Root != nil {
		for _, section := range store.Root.Children {
			if strings.EqualFold(section.Name, "CurrentState") {
				for _, leaf := range section.Children {
					found = found || strings.EqualFold(leaf.Name, "AgainromActions")
				}
			}
		}
	}
	if !found {
		return 0, false, nil
	}
	doc, err := sav.DecodeDocumentData(mf.raw)
	if err != nil {
		return 0, false, err
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		return 0, false, err
	}
	if a == nil {
		return 0, false, fmt.Errorf("native action leaf has no checked supplement")
	}
	return a.WriterCarriers, true, nil
}

func milestone2CarrierCorpus(t *testing.T, carrier uint32, fn func(*testing.T, milestone2File, *FrontEnd)) int {
	t.Helper()
	return milestone2Corpus(t, func(t *testing.T, mf milestone2File, fe *FrontEnd) {
		if milestone2CarrierApplies(t, mf, carrier) {
			fn(t, mf, fe)
			return
		}
		milestone2WriterRoundTrip(t, os.Getenv("AGAINROM_ASSETS"), mf, false)
	})
}

func milestone2CarrierApplies(t *testing.T, mf milestone2File, carrier uint32) bool {
	t.Helper()
	bits, written, err := milestone2WriterCarriers(mf)
	if err != nil {
		t.Fatalf("writer capability admission: %v", err)
	}
	if !written || bits&carrier != 0 || os.Getenv("AGAINROM_M2_ORIGINAL_CONTROL") == "1" {
		return true
	}
	t.Logf("M2-RULE Againrom round-trip: WriterCarriers=%#x, required=%#x", bits, carrier)
	return false
}

func milestone2DocumentBytes(t *testing.T, raw []byte) []byte {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	p, present, err := f.Campaign()
	if err != nil || !present || p.PayloadError != "" {
		t.Fatalf("campaign documents: present=%t err=%v payload=%s", present, err, p.PayloadError)
	}
	var out bytes.Buffer
	binary.Write(&out, binary.LittleEndian, uint32(len(p.Documents)))
	for _, d := range p.Documents {
		out.Write([]byte{byte(d.Value), byte(d.Value >> 8), byte(d.Value >> 16), byte(d.Value >> 24),
			byte(d.Kind), byte(d.Kind >> 8), byte(d.Kind >> 16), byte(d.Kind >> 24)})
	}
	if p.Payload != nil {
		pairs, err := sav.EncodeDocPayload(p.Payload)
		if err != nil {
			t.Fatal(err)
		}
		for _, pair := range pairs {
			binary.Write(&out, binary.LittleEndian, uint32(1))
			binary.Write(&out, binary.LittleEndian, pair|0x80000000)
		}
	}
	return out.Bytes()
}

func milestone2WriterRoundTrip(t *testing.T, assets string, mf milestone2File, changed bool) []string {
	t.Helper()
	if !mf.present {
		milestone2WriterTownRoundTrip(t, assets, mf)
		return nil
	}
	c := savGateCase{name: mf.rel, raw: mf.raw}
	f, _, refusal, town := savGateOpen(t, assets, "writer capability round-trip", c, mf.raw)
	if refusal != "" || town {
		t.Fatalf("Againrom LOAD: town=%t refusal=%s", town, refusal)
	}
	documents := milestone2DocumentBytes(t, mf.raw)
	ownedDocuments := f.Town.Documents()
	ownedPayload := f.Town.DocPayload()
	initial := f.live.world.Hash()
	initialTick := f.live.world.Tick()
	var changes []string
	if changed {
		store, _ := mf.f.StateStore()
		leaves := writerCensusLeaves(store)
		var view [2]int32
		_, xerr := fmt.Sscan(leaves["/View/X"], &view[0])
		_, yerr := fmt.Sscan(leaves["/View/Y"], &view[1])
		changes = writerCensusChange(f, view, xerr == nil && yerr == nil).changes
	}
	for cycle := 0; cycle < 2; cycle++ {
		f.live.world.PruneGroupsForSave() // SAVE prunes first; the rest of it changes nothing
		before := f.live.world.Hash()
		savedTick := f.live.world.Tick()
		g, _, written, refusal := savGateSaveLoad(t, assets, c, f)
		if refusal != "" {
			t.Fatal("Againrom SAVE/cold LOAD:", refusal)
		}
		if after := g.live.world.Hash(); after != before {
			t.Fatalf("Againrom cycle %d World hash saved=%016x loaded=%016x", cycle, before, after)
		}
		if got := milestone2DocumentBytes(t, written); !bytes.Equal(got, documents) {
			t.Fatalf("Againrom cycle %d document bytes changed: %x -> %x", cycle, documents, got)
		}
		if !reflect.DeepEqual(g.Town.Documents(), ownedDocuments) || !reflect.DeepEqual(g.Town.DocPayload(), ownedPayload) {
			t.Fatalf("Againrom cycle %d cold LOAD lost owned document state", cycle)
		}
		_, againrom, err := milestone2WriterCarriers(milestone2File{raw: written, f: mustMilestone2Open(t, written), present: true})
		if err != nil || !againrom {
			t.Fatalf("current writer supplement: admitted=%t err=%v", againrom, err)
		}
		firstDifference := 0
		for tick := 1; tick <= 60; tick++ {
			f.LiveAdvance(1)
			g.LiveAdvance(1)
			if f.live.world.Tick() != savedTick+uint64(tick) || g.live.world.Tick() != savedTick+uint64(tick) {
				t.Fatalf("Againrom cycle %d next tick %d did not advance both Worlds", cycle, tick)
			}
			if f.live.world.Hash() != g.live.world.Hash() && firstDifference == 0 {
				firstDifference = tick
			}
		}
		t.Logf("M2-WRITER cycle=%d ticks=%d initial=%016x saved=%016x cold=%016x next60-live=%016x next60-cold=%016x first-next-difference=%d documents=%d bytes", cycle, savedTick-initialTick, initial, before, before, f.live.world.Hash(), g.live.world.Hash(), firstDifference, len(documents))
	}
	return changes
}

func milestone2WriterTownRoundTrip(t *testing.T, assets string, mf milestone2File) {
	t.Helper()
	open := func(raw []byte) *FrontEnd {
		f, err := decodedInstallFront(assets)
		if err != nil {
			t.Fatal(err)
		}
		cleanupFrontAudio(t, f)
		headlessCensusMusic(t, f)
		f.SetDeterministicFrames(true)
		_, town, err := f.RestoreOriginal(raw)
		if err != nil || !town {
			t.Fatalf("Againrom town LOAD: town=%t err=%v", town, err)
		}
		return f
	}
	f := open(mf.raw)
	before, _, err := currentRoundTripSnapshot(f, false)
	if err != nil {
		t.Fatal(err)
	}
	written, refusal := savGateSave(t, f, false)
	if refusal != "" {
		t.Fatal(refusal)
	}
	g := open(written)
	after, _, err := currentRoundTripSnapshot(g, false)
	if err != nil {
		t.Fatal(err)
	}
	bv, av := reflect.ValueOf(before), reflect.ValueOf(after)
	for i := 0; i < bv.NumField(); i++ {
		field := bv.Type().Field(i)
		if field.IsExported() && !savGateTownProvenance[field.Name] {
			for path, difference := range savGateDiff(field.Name, bv.Field(i), av.Field(i)) {
				t.Errorf("Againrom town %s: %s", path, difference)
			}
		}
	}
	if !bytes.Equal(milestone2DocumentBytes(t, mf.raw), milestone2DocumentBytes(t, written)) ||
		!reflect.DeepEqual(f.Town.Documents(), g.Town.Documents()) || !reflect.DeepEqual(f.Town.DocPayload(), g.Town.DocPayload()) {
		t.Fatal("Againrom town lost owned document bytes/state")
	}
	t.Log("M2-WRITER town: no World; town state and owned document bytes survived SAVE/cold LOAD")
}

func mustMilestone2Open(t *testing.T, raw []byte) *sav.File {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestMilestone2WriterOriginalAdmission(t *testing.T) {
	sessionCorpusAudit(t)
	raw, err := cityfixture.Original(false)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"saveorcsdontgo.sav", "unlisted.sav"} {
		mf := milestone2File{rel: name, raw: raw, f: mustMilestone2Open(t, raw), present: true}
		bits, written, err := milestone2WriterCarriers(mf)
		if err != nil || written || bits != 0 {
			t.Fatalf("original admitted to Againrom: bits=%#x written=%t err=%v", bits, written, err)
		}
		for bit := uint32(1); bit <= currentCarrierTerrainBlockPlane; bit <<= 1 {
			if !milestone2CarrierApplies(t, mf, bit) {
				t.Fatal("original carrier comparison weakened", bit)
			}
		}
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.ProjectDocumentPayload(&doc, &sav.DocPayload{Version: sav.PayloadVersion, Data: []byte("payload alone is not a writer capability")}); err != nil {
		t.Fatal(err)
	}
	raw, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if _, written, err := milestone2WriterCarriers(milestone2File{raw: raw, f: mustMilestone2Open(t, raw)}); err != nil || written {
		t.Fatalf("document payload alone admitted: %t %v", written, err)
	}
}

func TestMilestone2WriterCurrentRoundTrip(t *testing.T) {
	sessionCorpusAudit(t)
	assets := os.Getenv("AGAINROM_ASSETS")
	f, _, refusal, town := savGateOpen(t, assets, "current writer", savGateCase{mission: 10}, nil)
	if refusal != "" || town {
		t.Fatal(refusal, town)
	}
	payload := &sav.DocPayload{Version: sav.PayloadVersion, Data: []byte{0, 255, 128, 1, 3, 7, 31, 63}}
	if err := f.Town.SetDocPayload(payload); err != nil {
		t.Fatal(err)
	}
	raw, refusal := savGateSave(t, f, true)
	if refusal != "" {
		t.Fatal(refusal)
	}
	projection, present, err := mustMilestone2Open(t, raw).Campaign()
	if err != nil || !present || !reflect.DeepEqual(projection.Payload, payload) {
		t.Fatalf("current writer did not emit requested payload: %v %t %v", projection.Payload, present, err)
	}
	mf := milestone2File{rel: "current writer", raw: raw, f: mustMilestone2Open(t, raw), present: true}
	bits, written, err := milestone2WriterCarriers(mf)
	if err != nil || !written || bits != 0 {
		t.Fatalf("current writer bits=%#x admitted=%t err=%v", bits, written, err)
	}
	milestone2WriterRoundTrip(t, assets, mf, false)
	ms := f.live.mission
	if refusal := f.CampaignSession.carryMissionHome(f.townInstall(), 10, ms.party, f.live.world, ms.ids, nil); refusal != "" {
		t.Fatal(refusal)
	}
	f.arriveInTown()
	townRaw, refusal := savGateSave(t, f, false)
	if refusal != "" {
		t.Fatal(refusal)
	}
	townFile := milestone2File{rel: "current town writer", raw: townRaw, f: mustMilestone2Open(t, townRaw)}
	bits, written, err = milestone2WriterCarriers(townFile)
	if err != nil || !written || bits != 0 || townFile.f.World != nil || milestone2CarrierApplies(t, townFile, currentCarrierDeadRoots) {
		t.Fatalf("current town writer bits=%#x admitted=%t err=%v", bits, written, err)
	}
	milestone2WriterRoundTrip(t, assets, townFile, false)
}

func TestMilestone2WriterBitAdmission(t *testing.T) {
	sessionCorpusAudit(t)
	raw, err := cityfixture.Original(false)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	for bit := uint32(1); bit <= currentCarrierTerrainBlockPlane; bit <<= 1 {
		leaf, err := json.Marshal(currentActionData{Version: 1, WriterCarriers: bit})
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		mf := milestone2File{raw: raw, f: mustMilestone2Open(t, raw), present: true}
		for required := uint32(1); required <= currentCarrierTerrainBlockPlane; required <<= 1 {
			if got := milestone2CarrierApplies(t, mf, required); got != (required == bit) {
				t.Fatalf("writer bit=%#x required=%#x applies=%t", bit, required, got)
			}
		}
	}
	for _, leaf := range []string{`{"Version":2,"WriterCarriers":0}`, `{"Version":1,"WriterCarriers":2048}`, `{"Version":1,"WriterCarriers":0} {}`} {
		if err := sav.SetNativeActions(&doc.State, []byte(leaf)); err != nil {
			t.Fatal(err)
		}
		raw, err := sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		if _, written, err := milestone2WriterCarriers(milestone2File{raw: raw, f: mustMilestone2Open(t, raw)}); err == nil || written {
			t.Fatalf("malformed supplement admitted=%t err=%v", written, err)
		}
	}
	milestone2WriterRecordControls(t)
}

func milestone2WriterRecordControls(t *testing.T) {
	t.Helper()
	var source milestone2File
	matched := 0
	for _, mf := range writerCensusFiles(t) {
		if fmt.Sprintf("%x", sha256.Sum256(mf.raw)) == "549c4deae41ecef8153c0d336e27c5307a856d799834693053c9e92e4ff56c2a" {
			source = mf
			matched++
		}
	}
	if matched != 1 {
		t.Fatalf("writer record control input occurs %d times, want 1", matched)
	}
	doc, err := sav.DecodeDocumentData(source.raw)
	if err != nil {
		t.Fatal(err)
	}
	leaf, present, err := sav.NativeActions(doc.State)
	if err != nil || !present {
		t.Fatal("writer record control supplement", present, err)
	}
	leaf = bytes.TrimSpace(leaf)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(leaf, &fields); err != nil {
		t.Fatal(err)
	}
	for key := range fields {
		if strings.EqualFold(key, "WriterCarriers") {
			t.Fatal("historical control already contains a writer record")
		}
	}
	wantGroups, err := expectedNativeGroups(source.f)
	if err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(filepath.Join(os.Getenv("AGAINROM_SAVE_CORPUS"), "2026-08-02", "game0009.sav"))
	if err != nil {
		t.Fatal(err)
	}
	control := milestone2File{raw: original, f: mustMilestone2Open(t, original)}
	if _, admitted, err := milestone2WriterCarriers(control); err != nil || admitted {
		t.Fatal("population control must remain original", admitted, err)
	}
	cases := []writerCarrierRecordCase{
		{"canonical none", `"WriterCarriers":0`, ""},
		{"canonical Groups", `"WriterCarriers":16`, ""},
	}
	cases = append(cases, ambiguousWriterCarrierRecords()...)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			modified := append(bytes.Clone(leaf[:len(leaf)-1]), []byte(","+tc.members+"}")...)
			if err := sav.SetNativeActions(&doc.State, modified); err != nil {
				t.Fatal(err)
			}
			raw, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(err)
			}
			gotGroups, err := expectedNativeGroups(mustMilestone2Open(t, raw))
			if err != nil || !reflect.DeepEqual(wantGroups, gotGroups) {
				t.Fatal("writer record changed the native Group control", err)
			}
			dir := t.TempDir()
			for name, data := range map[string][]byte{"probe.sav": raw, "control.sav": original} {
				if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			cmd := exec.Command(os.Args[0], "-test.v", "-test.count=1", "-test.run=^TestMilestone2Groups$")
			cmd.Env = append(os.Environ(), "AGAINROM_SAVE_CORPUS="+dir, "AGAINROM_M2_ORIGINAL_CONTROL=")
			out, err := cmd.CombinedOutput()
			if !bytes.Contains(out, []byte("--- PASS: TestMilestone2Groups/control.sav")) {
				t.Fatalf("original population control did not pass:\n%s", out)
			}
			if tc.name == "canonical none" {
				if err != nil || !bytes.Contains(out, []byte("--- PASS: TestMilestone2Groups/probe.sav")) || !bytes.Contains(out, []byte("M2-WRITER cycle=0")) || !bytes.Contains(out, []byte("M2-WRITER cycle=1")) {
					t.Fatalf("zero-record loss control did not run both writer cycles:\n%s", out)
				}
			} else {
				if err == nil || !bytes.Contains(out, []byte("--- FAIL: TestMilestone2Groups/probe.sav")) || bytes.Contains(out, []byte("M2-RULE Againrom round-trip")) {
					t.Fatalf("required Group refusal was bypassed:\n%s", out)
				}
				reason := "current actions: invalid WriterCarriers record"
				if tc.name == "canonical Groups" {
					reason = "raw/live Group or Player identity population differs"
				}
				if !bytes.Contains(out, []byte(reason)) {
					t.Fatalf("Group instrument did not report the required cause %q:\n%s", reason, out)
				}
			}
			t.Logf("M2-RECORD %s: original control PASS; probe PASS=%t; native Groups unchanged; child output SHA256=%x", tc.name, err == nil, sha256.Sum256(out))
		})
	}
}

func TestMilestone2WriterLossControls(t *testing.T) {
	sessionCorpusAudit(t)
	if os.Getenv("AGAINROM_M2_ORIGINAL_CONTROL") == "1" {
		t.Fatal("loss-control child must select only byte instruments")
	}
	checked := 0
	for _, mf := range writerCensusFiles(t) {
		if fmt.Sprintf("%x", sha256.Sum256(mf.raw)) != "549c4deae41ecef8153c0d336e27c5307a856d799834693053c9e92e4ff56c2a" {
			continue
		}
		bits, written, err := milestone2WriterCarriers(mf)
		if err != nil {
			t.Fatal(err)
		}
		if !mf.present || !written || bits != 0 {
			t.Fatalf("old writer loss-control input: World=%t written=%t bits=%#x", mf.present, written, bits)
		}
		checked++
		t.Run(mf.rel, func(t *testing.T) {
			w, stage, err := writerCensusWriteOne(t, os.Getenv("AGAINROM_ASSETS"), mf)
			if err != nil {
				t.Fatal(stage, err)
			}
			mismatch, _, _, err := writerCensusCheck(mf.f, mf.raw, w, w.raw)
			if err != nil || len(mismatch[writerCensusMissingKey]) != 2 {
				t.Fatalf("original Sack loss control: mismatch=%v err=%v", mismatch, err)
			}
			t.Logf("M2-LOSS Sacks: %v", mismatch[writerCensusMissingKey])
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "renamed.sav"), mf.raw, 0600); err != nil {
				t.Fatal(err)
			}
			// A World with cell planes keeps the Building roster this file
			// lists, because its cell records name that roster, so the
			// Buildings instrument finds the file's Buildings and passes; its
			// own controls prove it finds a changed roster.
			instruments := []string{"CellRecords", "DeadRoots", "DyingOwnerActors", "Groups", "MoverRoutes", "Players", "Sacks", "SessionBlock", "SpellEffects", "TerrainBlockPlane"}
			cmd := exec.Command(os.Args[0], "-test.v", "-test.count=1", "-test.run=^TestMilestone2(Buildings|"+strings.Join(instruments, "|")+")$/renamed.sav$")
			cmd.Env = append(os.Environ(), "AGAINROM_SAVE_CORPUS="+dir, "AGAINROM_M2_ORIGINAL_CONTROL=1")
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatal("original-carrier controls all passed")
			}
			if !bytes.Contains(out, []byte("--- PASS: TestMilestone2Buildings/renamed.sav")) {
				t.Fatalf("the Buildings instrument differs from the roster the file lists:\n%s", out)
			}
			for _, name := range instruments {
				marker := "--- FAIL: TestMilestone2" + name + "/renamed.sav"
				if !bytes.Contains(out, []byte(marker)) {
					t.Fatalf("loss control %s did not catch file mismatch:\n%s", name, out)
				}
				t.Log("M2-LOSS", marker)
			}
		})
	}
	if checked != 1 {
		t.Fatalf("old writer loss-control input occurs %d times, want 1", checked)
	}
}
