package game

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// residueCarrierGaps names each part a carrier row lacks.
func residueCarrierGaps(c residueCarrier) []string {
	var gaps []string
	if c.name == "" {
		gaps = append(gaps, "name")
	}
	if c.capture == nil {
		gaps = append(gaps, "capture")
	}
	if c.validate == nil {
		gaps = append(gaps, "validation")
	}
	if c.actors.rows == nil {
		gaps = append(gaps, "actor binding check")
	}
	if c.restore.snapshot == nil || c.restore.saved == nil {
		gaps = append(gaps, "restore")
	}
	saved := len(c.sav.keys) != 0 && c.sav.write != nil
	if saved == (c.sav.unsaved != "") || len(c.sav.keys) != 0 && c.sav.write == nil {
		gaps = append(gaps, "SAV key")
	}
	if len(c.fields) == 0 && !slices.ContainsFunc(c.sav.keys, func(k string) bool { return strings.HasPrefix(k, "Residue.") }) {
		gaps = append(gaps, "fields")
	}
	return gaps
}

// residueTableGaps checks a carrier table against SnapshotResidue and the SAV
// supplement: every residue field has exactly one carrier, every row has its
// five parts, and every SAV key is one field of currentActionData.
func residueTableGaps(table []residueCarrier) []string {
	var gaps []string
	owner := map[string]string{}
	for _, c := range table {
		for _, gap := range residueCarrierGaps(c) {
			gaps = append(gaps, fmt.Sprintf("carrier %q lacks its %s", c.name, gap))
		}
		for _, field := range c.fields {
			if _, ok := reflect.TypeFor[SnapshotResidue]().FieldByName(field); !ok || field == "keyed" {
				gaps = append(gaps, fmt.Sprintf("carrier %q names no residue field %s", c.name, field))
			}
			if prior, ok := owner["field "+field]; ok {
				gaps = append(gaps, fmt.Sprintf("residue field %s has carriers %q and %q", field, prior, c.name))
			}
			owner["field "+field] = c.name
		}
		for _, key := range c.sav.keys {
			if _, ok := reflect.TypeFor[currentActionData]().FieldByName(key); !ok && !strings.HasPrefix(key, "Residue.") || key == "Residue" {
				gaps = append(gaps, fmt.Sprintf("carrier %q names no SAV key %s", c.name, key))
			}
			if prior, ok := owner["key "+key]; ok {
				gaps = append(gaps, fmt.Sprintf("SAV key %s has carriers %q and %q", key, prior, c.name))
			}
			owner["key "+key] = c.name
		}
	}
	for field := range reflect.TypeFor[SnapshotResidue]().Fields() {
		if _, ok := owner["field "+field.Name]; !ok && field.Name != "keyed" {
			gaps = append(gaps, fmt.Sprintf("residue field %s has no carrier", field.Name))
		}
	}
	return gaps
}

func TestEveryResidueFieldHasOneCompleteCarrier(t *testing.T) {
	if gaps := residueTableGaps(residueCarriers); len(gaps) != 0 {
		t.Fatal(strings.Join(gaps, "\n"))
	}
	// The check itself fails a field without a row and a row without a part.
	for _, part := range []string{"field", "capture", "validation", "actor binding check", "restore", "SAV key"} {
		table := slices.Clone(residueCarriers)
		c := &table[0]
		want := "lacks its " + part
		switch part {
		case "field":
			table = table[1:]
			want = "SpellBolts has no carrier"
		case "capture":
			c.capture = nil
		case "validation":
			c.validate = nil
		case "actor binding check":
			c.actors.rows = nil
		case "restore":
			c.restore.saved = nil
		case "SAV key":
			c.sav.write = nil
		}
		gaps := residueTableGaps(table)
		if !slices.ContainsFunc(gaps, func(g string) bool { return strings.Contains(g, want) }) {
			t.Fatalf("a table without %s passed: %v", part, gaps)
		}
	}
}

func TestResidueMissionStateCoversEveryField(t *testing.T) {
	if (SnapshotResidue{}).hasMissionState() {
		t.Fatal("an empty residue holds mission state")
	}
	empty := SnapshotResidue{Swing: map[uint32]int{}, Phase: map[uint32]uint8{}, PendingMessages: []int32{}}
	if empty.hasMissionState() {
		t.Fatal("empty containers hold mission state")
	}
	v := reflect.ValueOf(&SnapshotResidue{}).Elem()
	for i := range v.NumField() {
		r := SnapshotResidue{}
		field := reflect.ValueOf(&r).Elem().Field(i)
		field = reflect.NewAt(field.Type(), field.Addr().UnsafePointer()).Elem()
		switch field.Kind() {
		case reflect.Slice:
			field.Set(reflect.MakeSlice(field.Type(), 1, 1))
		case reflect.Map:
			field.Set(reflect.MakeMap(field.Type()))
			field.SetMapIndex(reflect.Zero(field.Type().Key()), reflect.Zero(field.Type().Elem()))
		case reflect.Pointer:
			field.Set(reflect.New(field.Type().Elem()))
		case reflect.Bool:
			field.SetBool(true)
		default:
			field.Set(reflect.ValueOf(1).Convert(field.Type()))
		}
		if !r.hasMissionState() {
			t.Fatalf("residue field %s is not mission state", v.Type().Field(i).Name)
		}
	}
}

// residueProbeRow is the row of a synthetic carrier the table gains in a test.
type residueProbeRow struct {
	Entity sim.EntityID
	Value  int
}

func residueProbeRows(a *currentActionData) ([]residueProbeRow, error) {
	raw, ok := a.Residue["probe"]
	if !ok {
		return nil, nil
	}
	var rows []residueProbeRow
	return rows, json.Unmarshal(raw, &rows)
}

// registerResidueProbe adds one synthetic carrier row and nothing else. live
// is the probe's state per map; seen counts each part the sites invoked.
func registerResidueProbe(t *testing.T) (live map[*mapWorld][]residueProbeRow, seen map[string]int) {
	t.Helper()
	live, seen = map[*mapWorld][]residueProbeRow{}, map[string]int{}
	probe := residueCarrier{
		name: "probe",
		capture: func(mw *mapWorld, r *SnapshotResidue) {
			seen["capture"]++
			if rows, ok := live[mw]; ok {
				if r.keyed == nil {
					r.keyed = map[string]any{}
				}
				r.keyed["probe"] = slices.Clone(rows)
			}
		},
		validate: func(a *currentActionData) error {
			seen["validate"]++
			rows, err := residueProbeRows(a)
			for _, row := range rows {
				if row.Value < 0 {
					return fmt.Errorf("invalid probe value")
				}
			}
			return err
		},
		actors: residueActors{
			rows: func(a *currentActionData) []sim.EntityID {
				seen["actors"]++
				rows, _ := residueProbeRows(a)
				return residueRowActors(rows, func(r residueProbeRow) sim.EntityID { return r.Entity })
			},
			unbound: func(sim.EntityID) error { return fmt.Errorf("probe actor lacks a binding") },
		},
		restore: residueRestore{
			snapshot: func(mw *mapWorld, r SnapshotResidue) {
				seen["snapshot restore"]++
				if rows, ok := r.keyed["probe"].([]residueProbeRow); ok {
					live[mw] = slices.Clone(rows)
				}
			},
			saved: func(l residueLoad) error {
				seen["saved restore"]++
				rows, err := residueProbeRows(l.a)
				if err != nil || rows == nil {
					return err
				}
				for i := range rows {
					id, ok := l.ids[rows[i].Entity]
					if !ok {
						return fmt.Errorf("probe actor is absent")
					}
					rows[i].Entity = id
				}
				live[l.mw] = rows
				return nil
			},
		},
		sav: residueSAVKey{keys: []string{"Residue.probe"}, write: func(p residueProjection) error {
			seen["write"]++
			rows, ok := p.r.keyed["probe"]
			if !ok {
				return nil
			}
			raw, err := json.Marshal(rows)
			if p.a.Residue == nil {
				p.a.Residue = map[string]json.RawMessage{}
			}
			p.a.Residue["probe"] = raw
			return err
		}},
	}
	table := residueCarriers
	residueCarriers = append(slices.Clone(table), probe)
	t.Cleanup(func() { residueCarriers = table })
	return live, seen
}

func TestASyntheticResidueCarrierIsOneRow(t *testing.T) {
	f := currentArchiveFixture(t)
	live, seen := registerResidueProbe(t)
	if gaps := residueTableGaps(residueCarriers); len(gaps) != 0 {
		t.Fatal(strings.Join(gaps, "\n"))
	}
	actor := f.live.world.Entities()[0].ID
	// An actor the World does not hold reaches SAV only through the probe's
	// own binding, which LOAD resolves as missing.
	missing := sim.EntityID(60000)
	want := []residueProbeRow{{actor, 7}, {missing, 9}}
	live[f.live] = slices.Clone(want)

	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Residue.keyed["probe"].([]residueProbeRow); !reflect.DeepEqual(got, want) || !s.Residue.hasMissionState() {
		t.Fatalf("probe not captured: %v", s.Residue.keyed)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	a, err := readCurrentActions(&doc)
	if err != nil {
		t.Fatal(err)
	}
	if rows, _ := residueProbeRows(a); !reflect.DeepEqual(rows, want) {
		t.Fatalf("probe not written: %v", rows)
	}
	if !slices.ContainsFunc(a.Bindings, func(b currentActionBinding) bool { return b.ID == missing && !b.Structure && b.Missing }) {
		t.Fatal("probe actor not bound by the projection")
	}

	cold := openCurrentArchive(t, raw)
	got := live[cold.live]
	if len(got) != 2 || got[0].Value != 7 || got[1] != want[1] {
		t.Fatalf("probe not restored from SAV: %v", got)
	}
	if e, ok := cold.live.world.Entity(got[0].Entity); !ok || e.SourceBinding != f.live.world.Entities()[0].SourceBinding {
		t.Fatal("probe actor not resolved into the reopened World")
	}
	other := currentArchiveFixture(t)
	other.live.applyResidue(s.Residue)
	if !reflect.DeepEqual(live[other.live], want) {
		t.Fatalf("probe not restored from a snapshot: %v", live[other.live])
	}
	for _, part := range []string{"capture", "validate", "actors", "snapshot restore", "saved restore", "write"} {
		if seen[part] == 0 {
			t.Fatalf("no site invoked the probe's %s", part)
		}
	}

	for _, change := range []string{"invalid", "unbound", "unknown"} {
		t.Run(change, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			rows, _ := residueProbeRows(a)
			wantErr := ""
			switch change {
			case "invalid":
				rows[0].Value, wantErr = -1, "invalid probe value"
			case "unbound":
				rows[0].Entity, wantErr = 4000000, "probe actor lacks a binding"
			case "unknown":
				a.Residue["other"], wantErr = json.RawMessage("[]"), `current residue carrier "other" is unknown`
			}
			a.Residue["probe"], _ = json.Marshal(rows)
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			if _, err := readCurrentActions(&doc); err == nil || err.Error() != wantErr {
				t.Fatalf("decode = %v, want %s", err, wantErr)
			}
		})
	}
}

// Each carrier whose rows name an actor the decode must bind refuses a row
// whose actor has no binding; a cast run and a visual identity are resolved by
// their restore instead.
func TestResidueCarriersRefuseAnUnboundActor(t *testing.T) {
	f := currentArchiveFixture(t)
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	const unbound = sim.EntityID(4000000)
	swing := 1
	cases := []struct {
		carrier, want string
		add           func(a *currentActionData)
	}{
		{"target marks", "current target mark actor lacks a binding", func(a *currentActionData) {
			a.TargetMarks = append(a.TargetMarks, SnapshotTargetMark{Entity: unbound, Element: 20<<16 | 1})
		}},
		{"attack clocks", "current animation actor lacks a binding", func(a *currentActionData) {
			a.Animation = append(a.Animation, currentAnimationClock{Entity: unbound, Swing: &swing})
		}},
		{"death ages", fmt.Sprintf("invalid current death age for actor %d", unbound), func(a *currentActionData) {
			a.DeathAges = append(a.DeathAges, currentDeathAge{Entity: unbound, Age: 1})
		}},
		{"departed characters", "current departed character lacks an actor binding", func(a *currentActionData) {
			a.DepartedCharacters = append(a.DepartedCharacters, uint32(unbound))
		}},
		{"cast runs", "", func(a *currentActionData) {
			a.Runs = append(a.Runs, SnapshotCastRun{Entity: unbound, Left: 1, Span: 1})
		}},
		{"visual identities", "", func(a *currentActionData) {
			a.VisualIdentities = append(a.VisualIdentities, SnapshotVisualIdentity{Entity: unbound, Label: unbound})
		}},
	}
	for _, c := range cases {
		t.Run(c.carrier, func(t *testing.T) {
			doc, err := sav.DecodeDocumentData(raw)
			if err != nil {
				t.Fatal(err)
			}
			a, err := readCurrentActions(&doc)
			if err != nil {
				t.Fatal(err)
			}
			c.add(a)
			leaf, err := json.Marshal(a)
			if err != nil {
				t.Fatal(err)
			}
			if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
				t.Fatal(err)
			}
			_, err = readCurrentActions(&doc)
			if c.want == "" && err != nil || c.want != "" && (err == nil || err.Error() != c.want) {
				t.Fatalf("decode = %v, want %q", err, c.want)
			}
		})
	}
}

// withResidueCarrier runs body over a copy of the table whose named carrier
// change altered.
func withResidueCarrier(t *testing.T, name string, change func(c *residueCarrier), body func()) {
	t.Helper()
	table := residueCarriers
	defer func() { residueCarriers = table }()
	residueCarriers = slices.Clone(table)
	i := slices.IndexFunc(residueCarriers, func(c residueCarrier) bool { return c.name == name })
	if i < 0 {
		t.Fatalf("no carrier %q", name)
	}
	change(&residueCarriers[i])
	body()
}

// Loss controls: a carrier row without its binding check admits an unbound
// actor, and one without its restore loses its rows on a cold LOAD.
func TestResidueCarrierLossControls(t *testing.T) {
	f := currentArchiveFixture(t)
	id := f.live.world.Entities()[0].ID
	f.live.castRun[id] = castRun{2, 3, &castRunIdentity{}}
	f.live.swing[id], f.live.phase[id] = 1, sim.AttackCharging
	s, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	unboundMark := func() error {
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		a, err := readCurrentActions(&doc)
		if err != nil {
			t.Fatal(err)
		}
		a.TargetMarks = append(a.TargetMarks, SnapshotTargetMark{Entity: 4000000, Element: 20<<16 | 1})
		leaf, err := json.Marshal(a)
		if err != nil {
			t.Fatal(err)
		}
		if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
			t.Fatal(err)
		}
		_, err = readCurrentActions(&doc)
		return err
	}
	if unboundMark() == nil {
		t.Fatal("an unbound target mark was admitted")
	}
	withResidueCarrier(t, "target marks", func(c *residueCarrier) { c.actors.unbound = nil }, func() {
		if unboundMark() != nil {
			t.Fatal("the refusal does not come from the target marks' binding check")
		}
	})
	if cold := openCurrentArchive(t, raw); len(cold.live.castRun) != 1 {
		t.Fatalf("cold LOAD restored %d cast runs, want 1", len(cold.live.castRun))
	}
	withResidueCarrier(t, "cast runs", func(c *residueCarrier) { c.restore.saved = residueNotLoaded }, func() {
		if cold := openCurrentArchive(t, raw); len(cold.live.castRun) != 0 {
			t.Fatal("the cast runs do not come from their carrier's restore")
		}
	})
}
