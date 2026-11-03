package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func moverBytesDigest(m []byte) string {
	s := sha256.Sum256(m)
	return fmt.Sprintf("%x", s[:8])
}

// documentMoverDigests names the Mover of every actor record of a SAV
// document, in record order.
func documentMoverDigests(t *testing.T, got map[string]string, prefix string, doc *sav.DocumentData) {
	t.Helper()
	n := 0
	for i := range doc.Objects {
		r := &doc.Objects[i]
		for _, field := range r.Raw {
			if field.Name == "U154" {
				got[fmt.Sprintf("%s/%d/%s", prefix, n, r.Class)] = moverBytesDigest(field.Bytes)
				n++
			}
		}
	}
	got[prefix+"/count"] = fmt.Sprint(n)
}

// menuSavedDocument SAVEs f's mission through the game menu into a new store
// and decodes the written SAV.
func menuSavedDocument(t *testing.T, f *FrontEnd, app *ui.App, originals string) (*sav.DocumentData, SaveStore, string) {
	t.Helper()
	store := SaveStore{Dir: t.TempDir()}
	save, list, load := f.SaveSeams(store, OriginalStore{Dir: originals}, nil)
	app.SetSaveSeams(save, list, load)
	name := crossingMenuSave(t, app, store)
	raw, err := os.ReadFile(filepath.Join(store.Dir, name))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	return &doc, store, name
}

// TestMoverBytesRecordGame records the 180 Mover bytes every pkg/game writer
// writes for each actor kind it serves (a generated mission's SAVE, a SAVE
// after a LOAD of an original SAV and after its walk, the dead construct, the
// actor value pass, the siege hire, the town carrier and the city Human) and
// the World hash after a few hundred steps.
func TestMoverBytesRecordGame(t *testing.T) {
	got := map[string]string{}

	// A generated mission's SAVE, and again after 300 steps.
	f := cellStateFront(t)
	app := f.App("mover record generated")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	doc0, _, _ := menuSavedDocument(t, f, app, "")
	documentMoverDigests(t, got, "generated", doc0)
	for i := 0; i < 300; i++ {
		sim.Step(f.live.world, nil)
	}
	got["generated/hash"] = fmt.Sprintf("%016x", f.live.world.Hash())
	doc1, _, _ := menuSavedDocument(t, f, app, "")
	documentMoverDigests(t, got, "generated-stepped", doc1)

	// An original SAV's LOAD, a step east, and the SAVE.
	o := newGroupFront(t, -1)
	oapp := o.App("mover record original")
	originals, first := t.TempDir(), SaveStore{Dir: t.TempDir()}
	if err := os.WriteFile(filepath.Join(originals, "game1115.sav"), newGroupLiteral(t, o, false), 0600); err != nil {
		t.Fatal(err)
	}
	save, list, load := o.SaveSeams(first, OriginalStore{Dir: originals}, nil)
	oapp.SetSaveSeams(save, list, load)
	groundAppLoad(t, oapp, list, "game1115.sav")
	doc2, _, _ := menuSavedDocument(t, o, oapp, originals)
	documentMoverDigests(t, got, "original-loaded", doc2)
	strideStartEast(t, o)
	doc3, store, name := menuSavedDocument(t, o, oapp, originals)
	documentMoverDigests(t, got, "original-stride", doc3)
	for i := 0; i < 300; i++ {
		sim.Step(o.live.world, nil)
	}
	got["original/hash"] = fmt.Sprintf("%016x", o.live.world.Hash())
	doc4, _, _ := menuSavedDocument(t, o, oapp, originals)
	documentMoverDigests(t, got, "original-stepped", doc4)

	// The engine SAV's LOAD in a fresh front, and its SAVE.
	back := newGroupFront(t, -1)
	bapp := back.App("mover record engine load")
	bsave, blist, bload := back.SaveSeams(store, OriginalStore{}, nil)
	bapp.SetSaveSeams(bsave, blist, bload)
	groundAppLoad(t, bapp, blist, name)
	doc5, _, _ := menuSavedDocument(t, back, bapp, "")
	documentMoverDigests(t, got, "engine-loaded", doc5)
	for i := 0; i < 300; i++ {
		sim.Step(back.live.world, nil)
	}
	got["engine/hash"] = fmt.Sprintf("%016x", back.live.world.Hash())
	doc6, _, _ := menuSavedDocument(t, back, bapp, "")
	documentMoverDigests(t, got, "engine-stepped", doc6)

	// The actor value pass on a fresh and a kept record.
	doc, binding, world := actorProjectionFixture(t, "Human")
	e, _ := world.Entity(binding.EntityID)
	for _, fresh := range []bool{true, false} {
		record := doc.Objects[binding.ObjectIndex-1]
		for i := range record.Raw {
			if record.Raw[i].Name == "U154" {
				b := make([]byte, 180)
				for j := range b {
					b[j] = byte(j*3 + 1)
				}
				if fresh {
					b[5] = 0
				}
				record.Raw[i].Bytes = b
			}
		}
		next, err := savedActorValueRecord(record, e, fresh)
		if err != nil {
			t.Fatal(err)
		}
		mover, _ := savedActorRaw(&next, "U154", 180)
		got[fmt.Sprintf("value/%v", fresh)] = moverBytesDigest(mover)
	}

	// The dead construct's radius.
	dead := sav.DocumentRecordData{Raw: []sav.DocumentRawData{{Name: "U154", Bytes: make([]byte, 180)}}}
	if err := (&generatedDocumentBuilder{}).constructedSearchRadius(&dead, world); err != nil {
		t.Fatal(err)
	}
	got["dead"] = moverBytesDigest(dead.Raw[0].Bytes)

	// The siege hire.
	table := &mapload.Table{Units: dbCollection{{}, unitsRow("Catapult", map[int]int32{4: 70, 0x20: 2})}}
	for _, member := range []mapload.PartyMember{{Name: "Catapult", Class: unitsArmKey, MercenaryType: 1}, {Name: "Ballista", Class: unitsArmKey, MercenaryType: 2}} {
		u, err := nativeCitySiegeUnit(5, 7, member, table)
		if err != nil {
			got["siege/"+member.Name] = "error " + err.Error()
			continue
		}
		got["siege/"+member.Name] = moverBytesDigest(u.Raw154)
	}

	// The city Human: the constructor's zero record, its movement mask and
	// its rate.
	unit := sav.CityUnitData{Token: make([]byte, 37), Scalar1: make([]byte, 19), Raw154: make([]byte, 180)}
	for _, tbl := range []*mapload.Table{nil, {Humans: dbCollection{{}, {name: "partial", params: cityMoverRecordParams(3)}}}} {
		unit.Token[16] = 1
		if err := nativeCityInitializeHumanMovement(&unit, tbl); err != nil {
			t.Fatal(err)
		}
		got[fmt.Sprintf("city/movement/%v", tbl != nil)] = moverBytesDigest(unit.Raw154)
	}
	human := sav.CityUnitData{Token: make([]byte, 37), Scalar1: make([]byte, 19), Scalar2: make([]byte, 55), Raw154: make([]byte, 180),
		RawA6: make([]byte, 24), RawBE: make([]byte, 22), Raw114: make([]byte, 24), RawD4: make([]byte, 64), XP: make([]byte, 24)}
	if err := nativeCityApplyHumanState(&human, mapload.PartyMember{Name: "h", FigureFace: 3}, data.HumanState{MoverSpeed: 23}); err != nil {
		got["city/human"] = "error " + err.Error()
	} else {
		got["city/human"] = moverBytesDigest(human.Raw154)
	}

	// The town carrier: from a mission (built, kept, imported) and from a LOAD.
	tw, err := sim.NewWorld(3, sim.Bounds{Width: 16, Height: 16}, sim.ModeCanonical, nil, []sim.Entity{
		{ID: 1, X: 4, Y: 5, HP: 9, MaxHP: 9, Owner: sim.SelfSlot, TypeID: 1, Speed: 1, TokenSize: 1, Facing: 0x33, DesiredFacing: 0x44, RotationSpeed: 9},
		{ID: 2, X: 6, Y: 7, HP: 9, MaxHP: 9, Owner: 2, TypeID: 1, Speed: 1, TokenSize: 1, Facing: 0x55, DesiredFacing: 0x55, RotationSpeed: 11},
	})
	if err != nil {
		t.Fatal(err)
	}
	tw.ConstructMissionMovers([]sim.EntityID{1})
	var imported sim.SavedActorMotion
	imported.Entity, imported.Current = 2, true
	for i := range imported.Mover {
		imported.Mover[i] = byte(i + 2)
	}
	for _, c := range []struct {
		name    string
		motions map[sim.EntityID]sim.SavedActorMotion
		present bool
	}{{"built", nil, false}, {"present", nil, true}, {"imported", map[sim.EntityID]sim.SavedActorMotion{2: imported}, true}} {
		for _, te := range tw.Entities() {
			k, err := keptFromMission(tw, te, c.motions, c.present)
			if err != nil {
				t.Fatal(err)
			}
			got[fmt.Sprintf("town/%s/%d", c.name, te.ID)] = fmt.Sprintf("%v/%s", k.MoverPresent, moverBytesDigest(k.Mover[:]))
			u := townActorFixture()
			if err := writeTownActor(u, k); err != nil {
				t.Fatal(err)
			}
			got[fmt.Sprintf("town-write/%s/%d", c.name, te.ID)] = moverBytesDigest(u.Raw154)
		}
	}
	for name, m := range map[string]func([]byte){
		"zero":       func(b []byte) {},
		"mask":       func(b []byte) { b[5], b[10] = 0x44, 9 },
		"siege":      func(b []byte) { b[5], b[8], b[9], b[10] = 0x41, 5, 255, 3 },
		"radius":     func(b []byte) { b[5], b[8], b[9] = 0x41, 5, 0 },
		"facing":     func(b []byte) { b[0], b[5] = 0x40, 0x41 },
		"route-pair": func(b []byte) { b[8], b[9] = 0, 255 },
	} {
		b := make([]byte, 180)
		m(b)
		k := keptFromCharacter(sav.Character{Town: sav.TownActorState{Mover: b, State: 1}})
		if k == nil {
			got["town-load/"+name] = "nil"
			continue
		}
		got["town-load/"+name] = fmt.Sprintf("%v/%s", k.MoverPresent, moverBytesDigest(k.Mover[:]))
		var a [180]byte
		copy(a[:], b)
		var kept mapload.KeptActor
		keepMover(&kept, a)
		got["town-keep/"+name] = fmt.Sprintf("%v/%s", kept.MoverPresent, moverBytesDigest(kept.Mover[:]))
	}

	checkGameMoverRecords(t, got, gameMoverRecords)
}

// gameMoverRecords holds the bytes the writers wrote before they had one
// producer.
var gameMoverRecords = map[string]string{
	"city/human":                "82e07f18091bbc64",
	"city/movement/false":       "bec4161fa7d6f545",
	"city/movement/true":        "41e373c45acfb8f3",
	"dead":                      "51e788f1cd1354f9",
	"engine-loaded/0/Human":     "96207bd424be6a46",
	"engine-loaded/1/Unit":      "96d74a32369d0f73",
	"engine-loaded/2/Human":     "beb67d0d7b448afc",
	"engine-loaded/3/Human":     "96207bd424be6a46",
	"engine-loaded/count":       "4",
	"engine-stepped/0/Human":    "96207bd424be6a46",
	"engine-stepped/1/Unit":     "dd9090237ebc6bc7",
	"engine-stepped/2/Human":    "beb67d0d7b448afc",
	"engine-stepped/3/Human":    "96207bd424be6a46",
	"engine-stepped/count":      "4",
	"engine/hash":               "21bce19a2a9070e6",
	"generated-stepped/0/Human": "beb67d0d7b448afc",
	"generated-stepped/count":   "1",
	"generated/0/Human":         "beb67d0d7b448afc",
	"generated/count":           "1",
	"generated/hash":            "b23d099b030d1d17",
	"original-loaded/0/Unit":    "c022858d776c3966",
	"original-loaded/1/Human":   "c022858d776c3966",
	"original-loaded/2/Human":   "beb67d0d7b448afc",
	"original-loaded/3/Human":   "c022858d776c3966",
	"original-loaded/count":     "4",
	"original-stepped/0/Human":  "96207bd424be6a46",
	"original-stepped/1/Unit":   "dd9090237ebc6bc7",
	"original-stepped/2/Human":  "beb67d0d7b448afc",
	"original-stepped/3/Human":  "96207bd424be6a46",
	"original-stepped/count":    "4",
	"original-stride/0/Human":   "96207bd424be6a46",
	"original-stride/1/Unit":    "96d74a32369d0f73",
	"original-stride/2/Human":   "beb67d0d7b448afc",
	"original-stride/3/Human":   "96207bd424be6a46",
	"original-stride/count":     "4",
	"original/hash":             "21bce19a2a9070e6",
	"siege/Ballista":            "faab83a69d6923c5",
	"siege/Catapult":            "faab83a69d6923c5",
	"town-keep/facing":          "true/cc1e4868ad5576c9",
	"town-keep/mask":            "false/10b2a66888c58a54",
	"town-keep/radius":          "true/93fdb43845e15eac",
	"town-keep/route-pair":      "true/e3493edfff6cd526",
	"town-keep/siege":           "false/10b2a66888c58a54",
	"town-keep/zero":            "false/10b2a66888c58a54",
	"town-load/facing":          "true/cc1e4868ad5576c9",
	"town-load/mask":            "false/10b2a66888c58a54",
	"town-load/radius":          "true/93fdb43845e15eac",
	"town-load/route-pair":      "true/e3493edfff6cd526",
	"town-load/siege":           "false/10b2a66888c58a54",
	"town-load/zero":            "false/10b2a66888c58a54",
	"town-write/built/1":        "3e63718305aef311",
	"town-write/built/2":        "e422b25563dc6d4b",
	"town-write/imported/1":     "3e63718305aef311",
	"town-write/imported/2":     "38438b7ab945c895",
	"town-write/present/1":      "3e63718305aef311",
	"town-write/present/2":      "e422b25563dc6d4b",
	"town/built/1":              "true/077328326cf9bdf3",
	"town/built/2":              "true/41b7adad023d7225",
	"town/imported/1":           "true/077328326cf9bdf3",
	"town/imported/2":           "true/524bdf12bc33e7f2",
	"town/present/1":            "true/077328326cf9bdf3",
	"town/present/2":            "true/41b7adad023d7225",
	"value/false":               "ce65f491eb4f3874",
	"value/true":                "67fe05663a74dddb",
}

func cityMoverRecordParams(domain int32) []int32 {
	params := make([]int32, 23)
	for i := range params {
		params[i] = -1
	}
	params[21], params[22] = 3, domain
	return params
}

func checkGameMoverRecords(t *testing.T, got, want map[string]string) {
	t.Helper()
	bad := len(got) != len(want)
	for k, v := range got {
		if want[k] != v {
			bad = true
			t.Errorf("%s = %s, recorded %s", k, v, want[k])
		}
	}
	if bad {
		keys := make([]string, 0, len(got))
		for k := range got {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var b strings.Builder
		for _, k := range keys {
			fmt.Fprintf(&b, "\t%q: %q,\n", k, got[k])
		}
		t.Fatalf("records differ; now:\n%s", b.String())
	}
}
