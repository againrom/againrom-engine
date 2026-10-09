package game

import (
	"testing"

	"againrom/pkg/formats/sav"
)

func TestGeneratedMissionSAVOffMapActorHasNoPublicationRecipient(t *testing.T) {
	f := releaseFront(t)
	app := f.App("generated mission off-map actor")
	if err := app.OpenMission(f.MissionOpenerWith(141, MissionParty(nil, nil, nil))); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(113)
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range doc.Objects {
		record := &doc.Objects[i]
		if record.Class != "Unit" {
			continue
		}
		mapID, err := savedStructureValue(record, "T08")
		if err != nil || mapID != 106 {
			continue
		}
		found = true
		flags, err := savedStructureValue(record, "U4C")
		if err != nil || flags&8 == 0 {
			t.Fatalf("map unit 106 has U4C=%#x, err=%v; want off-map bit", flags, err)
		}
		mask, err := savedStructureValue(record, "T18")
		if err != nil || mask != 0 {
			t.Fatalf("off-map map unit 106 has T18=%d, err=%v; want 0", mask, err)
		}
	}
	if !found {
		t.Fatal("generated mission SAV lacks map unit 106")
	}
}

// An actor's recipient mask is written by the SAV-1093/SAV-678 rule from the
// current off-map state; a loaded record's other mask is not carried (DIV-2503).
func TestGeneratedMissionSAVRetainsLoadedOffMapRecipientMask(t *testing.T) {
	f := releaseFront(t)
	if err := f.App("off-map mask source").OpenMission(f.MissionOpenerWith(141, MissionParty(nil, nil, nil))); err != nil {
		t.Fatal(err)
	}
	f.LiveAdvance(113)
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		id, _ := savedStructureValue(r, "T08")
		if id == 106 {
			flags, _ := savedStructureValue(r, "U4C")
			if flags&sav.ActorOffMapFlag == 0 {
				t.Fatal("source map unit 106 is not off-map")
			}
			mustSetValue(r, "T18", 2)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("source map unit 106 absent")
	}
	wire, err = sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	loaded := releaseFront(t)
	open, town, err := loaded.RestoreOriginal(wire)
	if err != nil || town {
		t.Fatalf("source LOAD town=%t: %v", town, err)
	}
	if err := loaded.App("off-map mask source resave").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	snapshot, label, err = loaded.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	wire, err = loaded.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err = sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if r.Class != "Unit" {
			continue
		}
		id, _ := savedStructureValue(r, "T08")
		if id == 106 {
			flags, _ := savedStructureValue(r, "U4C")
			mask, err := savedStructureValue(r, "T18")
			if err != nil || flags&sav.ActorOffMapFlag == 0 || mask != 0 {
				t.Fatalf("loaded off-map map unit 106 has U4C=%#x T18=%d, err=%v; want the off-map mask 0", flags, mask, err)
			}
			return
		}
	}
	t.Fatal("resaved map unit 106 absent")
}
