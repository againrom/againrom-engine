package game

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// milestone2WriterCarrier is one carrier the writer claims: the suffix of its
// TestMilestone2 instrument, the per-file comparison that instrument runs and
// the corruptions of the written carrier it must detect.
type milestone2WriterCarrier struct {
	name   string
	check  func(*testing.T, milestone2File, *FrontEnd) (milestone2CarrierFileResult, error)
	losses []milestone2CarrierLoss
}

type milestone2CarrierLoss struct {
	name string
	edit func(*sav.DocumentData)
}

// milestone2WriterCarriersChecked holds every bit of currentWriterCarriers.
var milestone2WriterCarriersChecked = map[uint32]milestone2WriterCarrier{
	currentCarrierTerrainBlockPlane: {name: "TerrainBlockPlane", check: milestone2TerrainFile, losses: []milestone2CarrierLoss{
		// SAV-BLOCK-011: a row's dynamic byte is above 0x0f and its cell lies in the sweep window.
		{"dynamic-gate", func(d *sav.DocumentData) { d.World.Blocks[len(d.World.Blocks)/2].Dyn = 0x0f }},
		{"sweep-window", func(d *sav.DocumentData) {
			d.World.Blocks = append([]sav.BlockRecord{{Cell: 0x0100, Dyn: 0x30, Static: 0x20}}, d.World.Blocks...)
		}},
	}},
}

func milestone2WriterCarrierBits() []uint32 {
	var out []uint32
	for bit := uint32(1); bit&currentCarrierAll != 0; bit <<= 1 {
		if currentWriterCarriers&bit != 0 {
			out = append(out, bit)
		}
	}
	return out
}

func milestone2WriterCarrierNames() string {
	var names []string
	for _, bit := range milestone2WriterCarrierBits() {
		names = append(names, milestone2WriterCarriersChecked[bit].name)
	}
	return strings.Join(names, ",")
}

// milestone2WriterCarrierCheck runs each claimed carrier's per-file
// comparison on one written file; a town file has no World and is not checked.
func milestone2WriterCarrierCheck(t *testing.T, assets, rel string, raw []byte) (int, []string) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		return 0, []string{"written file: " + err.Error()}
	}
	if f.World == nil {
		return 0, nil
	}
	fe, err := decodedInstallFront(assets)
	if err != nil {
		t.Fatalf("NewFrontEnd(%q): %v", assets, err)
	}
	cleanupFrontAudio(t, fe)
	fe.SetDeterministicFrames(true)
	mf := milestone2File{rel: rel, raw: raw, body: f.Body, f: f, present: true}
	records := 0
	var differences []string
	for _, bit := range milestone2WriterCarrierBits() {
		c := milestone2WriterCarriersChecked[bit]
		r, err := c.check(t, mf, fe)
		if err != nil {
			differences = append(differences, c.name+": LOAD refused: "+err.Error())
			continue
		}
		records += r.records
		for _, d := range r.differences {
			differences = append(differences, c.name+": "+d)
		}
	}
	return records, differences
}

// milestone2WriterCarrierChild runs the named instrument tests over the files
// in dir. The bits each file carries select the instrument.
func milestone2WriterCarrierChild(dir string, names []string) ([]byte, error) {
	cmd := exec.Command(os.Args[0], "-test.v", "-test.count=1", "-test.run=^TestMilestone2("+strings.Join(names, "|")+")$")
	cmd.Env = append(os.Environ(), "AGAINROM_SAVE_CORPUS="+dir, "AGAINROM_M2_ORIGINAL_CONTROL=")
	return cmd.CombinedOutput()
}

// TestMilestone2WriterCarrierInstruments writes the New Game save points and
// one town save through the player's SAVE route and runs each claimed
// carrier's instrument test on them. Each loss corrupts that carrier in a
// copy of a written file, which the instrument must fail beside the clean copy.
func TestMilestone2WriterCarrierInstruments(t *testing.T) {
	sessionCorpusAudit(t)
	if os.Getenv("AGAINROM_M2_ORIGINAL_CONTROL") == "1" {
		t.Fatal("the written bits, not the original control, must select the instruments")
	}
	for bit := uint32(1); bit&currentCarrierAll != 0; bit <<= 1 {
		if _, checked := milestone2WriterCarriersChecked[bit]; checked != (currentWriterCarriers&bit != 0) {
			t.Fatalf("carrier bit %#x: claimed=%t checked=%t", bit, currentWriterCarriers&bit != 0, checked)
		}
	}
	assets := os.Getenv("AGAINROM_ASSETS")
	dir := t.TempDir()
	written := map[string][]byte{}
	for i, c := range savGateNewGames() {
		f, _, refusal, _ := savGateOpen(t, assets, "writer carriers", c, nil)
		if refusal != "" {
			t.Fatal(c.name, refusal)
		}
		savGatePlay(t, f, c)
		raw, refusal := savGateSave(t, f, true)
		if refusal != "" {
			t.Fatal(c.name, refusal)
		}
		written[fmt.Sprintf("newgame%d.sav", i)] = raw
		if i == 0 {
			ms := f.live.mission
			if refusal := f.CampaignSession.carryMissionHome(f.townInstall(), c.mission, ms.party, f.live.world, ms.ids, nil); refusal != "" {
				t.Fatal(refusal)
			}
			f.arriveInTown()
			town, refusal := savGateSave(t, f, false)
			if refusal != "" {
				t.Fatal(refusal)
			}
			written["town.sav"] = town
		}
	}
	for name, raw := range written {
		bits, admitted, err := milestone2WriterCarriers(milestone2File{raw: raw, f: mustMilestone2Open(t, raw)})
		if err != nil || !admitted || bits != currentWriterCarriers {
			t.Fatalf("%s: WriterCarriers=%#x admitted=%t err=%v, want %#x", name, bits, admitted, err, currentWriterCarriers)
		}
		if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	var names []string
	for _, bit := range milestone2WriterCarrierBits() {
		names = append(names, milestone2WriterCarriersChecked[bit].name)
	}
	if len(names) == 0 {
		return
	}
	out, err := milestone2WriterCarrierChild(dir, names)
	if err != nil || bytes.Contains(out, []byte("M2-RULE Againrom round-trip")) {
		t.Fatalf("claimed carrier instruments on written files: %v\n%s", err, out)
	}
	for _, name := range names {
		for file := range written {
			if !bytes.Contains(out, []byte("--- PASS: TestMilestone2"+name+"/"+file)) {
				t.Fatalf("%s did not pass on %s:\n%s", name, file, out)
			}
		}
	}
	t.Logf("writer carriers: WriterCarriers=%#x; %s passed on %d New Game and town file(s) this build wrote", currentWriterCarriers, strings.Join(names, ","), len(written))
	source := written["newgame1.sav"]
	for _, bit := range milestone2WriterCarrierBits() {
		c := milestone2WriterCarriersChecked[bit]
		for _, loss := range c.losses {
			doc, err := sav.DecodeDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			loss.edit(&doc)
			corrupt, err := sav.EncodeDocumentData(doc)
			if err != nil {
				t.Fatal(loss.name, err)
			}
			lossDir := t.TempDir()
			for name, raw := range map[string][]byte{"clean.sav": source, loss.name + ".sav": corrupt} {
				if err := os.WriteFile(filepath.Join(lossDir, name), raw, 0o600); err != nil {
					t.Fatal(err)
				}
			}
			out, err := milestone2WriterCarrierChild(lossDir, []string{c.name})
			if err == nil || !bytes.Contains(out, []byte("--- PASS: TestMilestone2"+c.name+"/clean.sav")) ||
				!bytes.Contains(out, []byte("--- FAIL: TestMilestone2"+c.name+"/"+loss.name+".sav")) {
				t.Fatalf("%s loss %s not detected:\n%s", c.name, loss.name, out)
			}
			t.Logf("writer carriers: %s loss %s detected; clean copy passes", c.name, loss.name)
		}
	}
}
