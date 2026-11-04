package game

import (
	"bytes"
	"strings"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
	"againrom/pkg/mapedit"
	"againrom/pkg/ui"
)

// composedCompanionEditor is unitIdentityEditor with npc record 23 composed
// from the starting hero (a fighter of the other sex), Humans rows of the
// composed server ids 26 (male fighter) and 27 (female fighter), and unit 5
// on record 23.
func composedCompanionEditor(t *testing.T) (*mapInspectionEdits, []byte) {
	t.Helper()
	s, _ := unitIdentityEditor(t)
	x := s.inspector
	humans := x.table.Humans.(dbCollection)
	for _, r := range []struct {
		name string
		sid  int32
	}{{"Fighter man", 26}, {"Fighter woman", 27}} {
		p := figureRow(3, 1, 0)
		p[24] = r.sid
		humans = append(humans, dbEntry{name: r.name, params: p})
	}
	x.table.Humans = humans
	section := func(id string, flags string, def int32) synth.RegNode {
		n := synth.RegNode{Name: "npc" + id, Kind: 0x01, Children: []synth.RegNode{{Name: "DataBinID", Kind: 0x02, Int: def}}}
		if flags != "" {
			n.Children = append(n.Children, synth.RegNode{Name: "Flags", Kind: 0x00, Str: flags})
		}
		return n
	}
	npcBytes := synth.Reg(0x11, []synth.RegNode{section("0", "", 1234), section("7", "", 1234), section("8", "", 4321),
		section("9", "", 9999), section("23", "Hero,!Mage,!MySex,Start", 26)})
	r, err := reg.Parse(npcBytes)
	if err != nil {
		t.Fatal(err)
	}
	catalog := data.LoadNPCCatalog(r)
	x.table.NPCCatalog, x.table.NPC = catalog, catalog.Defs
	x.archives.Containers.Overlay("scenario/npc.reg", func() ([]byte, error) { return npcBytes, nil })
	unit, err := mapedit.New(synth.ALM(synth.ALMOptions{Width: 40, Height: 40, Units: []synth.ALMUnit{
		{X: 16 << 8, Y: 8 << 8, ClassID: 4, ClassSubID: 23, Flags: 1, UnitID: 12}}}))
	if err != nil {
		t.Fatal(err)
	}
	rec, _ := unit.UnitRecord(0)
	n, err := s.model.PlaceUnit(rec)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.model.SetUnitField(n, "owner", 1); err != nil {
		t.Fatal(err)
	}
	x.recordOptions, x.identities = nil, nil
	raw := s.model.Bytes()
	doc, err := x.inspect(raw, "composed.alm")
	if err != nil {
		t.Fatal(err)
	}
	return doc.Edits.(*mapInspectionEdits), raw
}

// A placement on an npc record the game composes from the starting hero is
// that named character: the popup, the label and the choice name the record
// and, for each kind of starting hero, the Humans row a start gives it;
// applying it unchanged writes byte-equal, and another unit can be made it.
func TestEditorUnitIdentityNamesAComposedCompanion(t *testing.T) {
	s, raw := composedCompanionEditor(t)
	x := s.inspector
	const unit = 5
	want := identityValue(identityNamed, 23)
	f := identityField(t, s, unit)
	if f.Value != want {
		t.Errorf("the composed companion shows identity %#x, want %#x", f.Value, want)
	}
	offered := false
	for _, c := range f.Choices {
		offered = offered || c.Value == want
	}
	if !offered {
		t.Error("the identity choice does not offer composed record 23")
	}
	text := f.Meaning
	for _, l := range f.Lines {
		text += "\n" + l.Text
	}
	if strings.Contains(text, x.word("editor.unit.identity.choose")) {
		t.Errorf("the popup asks to choose who a composed companion is: %q", text)
	}
	for _, w := range []struct{ hero, row string }{
		{"editor.message.hero.manfighter", "Fighter woman"}, {"editor.message.hero.womanfighter", "Fighter man"},
		{"editor.message.hero.manmage", "Fighter woman"}, {"editor.message.hero.womanmage", "Fighter man"},
	} {
		if !strings.Contains(text, x.word(w.hero)+": "+w.row) {
			t.Errorf("the popup does not say a %s hero gets %q: %q", x.word(w.hero), w.row, text)
		}
	}
	fields := s.Fields(ui.KindUnit, unit)
	sp := fields[fieldIndex(t, fields, "speaker")]
	speaker := ""
	for _, l := range sp.Lines {
		speaker += l.Text
	}
	if strings.Contains(speaker, x.word("editor.speaker.why.noplacement")) {
		t.Errorf("the speaker line says the game makes no unit: %q", speaker)
	}
	d, err := s.project(s.model)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range d.Records {
		if r.Kind != ui.KindUnit || r.Index != unit {
			continue
		}
		if !strings.HasPrefix(r.Name, x.word("editor.unit.kind.named")) || !strings.Contains(r.Name, x.word("editor.unit.identity.byhero")+" (record 23)") {
			t.Errorf("the composed companion's label is %q", r.Name)
		}
	}
	if _, err := s.SetField(ui.KindUnit, unit, "identity", f.Value); err != nil || s.CanUndo() || !bytes.Equal(s.model.Bytes(), raw) {
		t.Fatalf("applying the composed companion unchanged edited the map: %v", err)
	}
	if _, err := s.SetField(ui.KindUnit, 1, "identity", want); err != nil {
		t.Fatalf("making the wolf composed record 23: %v", err)
	}
	if got := identityField(t, s, 1).Value; got != want {
		t.Errorf("the wolf shows %#x after it was made record 23", got)
	}
	if _, err := s.Undo(); err != nil || !bytes.Equal(s.model.Bytes(), raw) {
		t.Errorf("one Undo does not restore the wolf: %v", err)
	}
}
