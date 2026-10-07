package game

import (
	"encoding/binary"
	"maps"
	"reflect"
	"slices"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
)

// These are synthetic INPUT values, not decoder-generated expectations.
// The acceptance expectation is subsequently read only from the emitted bytes.
func players1154Fixture(t *testing.T, equalPlayers, aliasDiaries bool) (*FrontEnd, []byte) {
	t.Helper()
	f := newGroupFront(t, -1)
	doc, err := sav.DecodeDocumentData(newGroupLiteral(t, f, true))
	if err != nil {
		t.Fatal(err)
	}
	diary := func(seed uint32) sav.DocumentRecordData {
		dwords := binary.LittleEndian.AppendUint32(nil, 0)
		dwords = binary.LittleEndian.AppendUint32(dwords, seed)
		dwords = binary.LittleEndian.AppendUint32(dwords, 9)
		words := binary.LittleEndian.AppendUint16(nil, 1024)
		words = binary.LittleEndian.AppendUint16(words, uint16(1024-seed))
		words = binary.LittleEndian.AppendUint16(words, 17) // deliberately independent of Count.
		return sav.DocumentRecordData{Class: "Diary", Values: []sav.DocumentValueData{{Name: "D2C", Value: 0x1234}}, Counts: []sav.DocumentCountData{{Name: "Journal", Count: 3}, {Name: "JournalWords", Count: 3}}, Raw: []sav.DocumentRawData{{Name: "Journal", Bytes: dwords}, {Name: "JournalWords", Bytes: words}}}
	}
	var firstPlayer uint16
	var firstDiary uint16
	objects := len(doc.Objects)
	for i := 0; i < objects; i++ {
		record := &doc.Objects[i]
		if record.Class == "Player" {
			seed := uint32(i + 2)
			if equalPlayers {
				// A null source key is legal and is not an archive identity.
				// Equal nonzero keys would correctly fail the owner-key guard.
				newGroupSetValue1115(t, record, "This", 0)
				seed = 2
			}
			for j := range record.Raw {
				if record.Raw[j].Name == "PRaw32" {
					for k := range record.Raw[j].Bytes {
						record.Raw[j].Bytes[k] = byte(seed + uint32(k))
					}
				}
			}
			for j := range record.Inline {
				if record.Inline[j].Name == "Diary" {
					record.Inline[j].Record = diary(seed)
				}
			}
			// Retain an explicit nonzero actor-derived F58 owner value too.
			newGroupSetValue1115(t, record, "F58", seed+23)
			if equalPlayers {
				if firstPlayer == 0 {
					firstPlayer = uint16(i + 1)
				} else {
					record.Values = slices.Clone(doc.Objects[firstPlayer-1].Values)
					record.Texts = slices.Clone(doc.Objects[firstPlayer-1].Texts)
				}
			}
		} else if record.Class == "Human" {
			index := firstDiary
			if !aliasDiaries || index == 0 {
				index = uint16(len(doc.Objects) + 1)
				doc.Objects = append(doc.Objects, diary(7))
				if firstDiary == 0 {
					firstDiary = index
				}
			}
			for j := range doc.Objects[i].RefSlots {
				if doc.Objects[i].RefSlots[j].Name == "Diary" {
					doc.Objects[i].RefSlots[j].Objects = []uint16{index}
				}
			}
		}
	}
	doc, _, err = sav.ReindexDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	return f, raw
}

func players1154Inputs(t *testing.T, raw []byte) (players1154Source, map[uint16]uint16, sav.DocumentData) {
	t.Helper()
	f, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	want, err := players1154Expected(f)
	if err != nil {
		t.Fatal(err)
	}
	join, err := players1154Origins(raw)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(want.roots) != 6 || len(want.players) != 2 || want.roots[1] != 0 || want.roots[4] != 0 || want.roots[0] != want.roots[2] || want.roots[3] != want.roots[5] || want.roots[0] == want.roots[3] {
		t.Fatal("literal null/repeat/distinct roots changed", want.roots)
	}
	if d := players1154DocumentDifferences(want, join, &doc); len(d) != 0 {
		t.Fatal("unmodified retained baseline", d)
	}
	return want, join, doc
}

func TestPlayers1154RetainedOmissionAndAliasControls(t *testing.T) {
	_, raw := players1154Fixture(t, false, false)
	want, join, doc := players1154Inputs(t, raw)
	left, right := want.players[want.roots[0]], want.players[want.roots[3]]
	if left.values["Slot"] != right.values["Slot"] || slices.Equal(left.tail, right.tail) || reflect.DeepEqual(left.diary.entries(), right.diary.entries()) {
		t.Fatal("same-Slot distinct-tail/Diary subject absent")
	}
	player := join[left.index]
	mutate := func(t *testing.T, change func(*sav.DocumentData, map[uint16]uint16)) {
		t.Helper()
		copy, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		mapping := maps.Clone(join)
		change(&copy, mapping)
		if d := players1154DocumentDifferences(want, mapping, &copy); len(d) == 0 {
			t.Fatal("acceptance missed omission/alias mutation")
		}
	}
	for _, field := range doc.Objects[player-1].Values {
		t.Run("omit scalar "+field.Name, func(t *testing.T) {
			mutate(t, func(d *sav.DocumentData, _ map[uint16]uint16) {
				r := &d.Objects[player-1]
				r.Values = slices.DeleteFunc(r.Values, func(v sav.DocumentValueData) bool { return v.Name == field.Name })
			})
		})
	}
	cases := map[string]func(*sav.DocumentData, map[uint16]uint16){
		"omit Name": func(d *sav.DocumentData, _ map[uint16]uint16) { d.Objects[player-1].Texts = nil },
		"omit Raw10": func(d *sav.DocumentData, _ map[uint16]uint16) {
			r := &d.Objects[player-1]
			r.Raw = slices.DeleteFunc(r.Raw, func(v sav.DocumentRawData) bool { return v.Name == "Raw10" })
		},
		"omit tail": func(d *sav.DocumentData, _ map[uint16]uint16) {
			r := &d.Objects[player-1]
			r.Raw = slices.DeleteFunc(r.Raw, func(v sav.DocumentRawData) bool { return v.Name == "PRaw32" })
		},
		"tail byte31": func(d *sav.DocumentData, _ map[uint16]uint16) {
			for i := range d.Objects[player-1].Raw {
				r := &d.Objects[player-1].Raw[i]
				if r.Name == "PRaw32" {
					r.Bytes[31] ^= 1
				}
			}
		},
		"omit inline Diary":          func(d *sav.DocumentData, _ map[uint16]uint16) { d.Objects[player-1].Inline = nil },
		"omit root null":             func(d *sav.DocumentData, _ map[uint16]uint16) { d.Players = slices.Delete(d.Players, 1, 2) },
		"alias root to other Player": func(d *sav.DocumentData, _ map[uint16]uint16) { d.Players[2] = d.Players[3] },
		"omit origin":                func(_ *sav.DocumentData, j map[uint16]uint16) { delete(j, left.index) },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) { mutate(t, change) })
	}
	for _, name := range []string{"D2C", "Journal count", "JournalWords count", "Journal bytes", "JournalWords bytes"} {
		t.Run("omit inline Diary "+name, func(t *testing.T) {
			mutate(t, func(d *sav.DocumentData, _ map[uint16]uint16) {
				r := &d.Objects[player-1].Inline[0].Record
				switch name {
				case "D2C":
					r.Values = nil
				case "Journal count":
					r.Counts = slices.DeleteFunc(r.Counts, func(c sav.DocumentCountData) bool { return c.Name == "Journal" })
				case "JournalWords count":
					r.Counts = slices.DeleteFunc(r.Counts, func(c sav.DocumentCountData) bool { return c.Name == "JournalWords" })
				case "Journal bytes":
					r.Raw = slices.DeleteFunc(r.Raw, func(c sav.DocumentRawData) bool { return c.Name == "Journal" })
				case "JournalWords bytes":
					r.Raw = slices.DeleteFunc(r.Raw, func(c sav.DocumentRawData) bool { return c.Name == "JournalWords" })
				}
			})
		})
	}
}

func TestPlayers1154EqualValuedIdentityControls(t *testing.T) {
	_, raw := players1154Fixture(t, true, false)
	want, join, doc := players1154Inputs(t, raw)
	a, b := want.players[want.roots[0]], want.players[want.roots[3]]
	if !reflect.DeepEqual(a.values, b.values) || !slices.Equal(a.tail, b.tail) || !reflect.DeepEqual(a.diary.entries(), b.diary.entries()) {
		t.Fatal("equal-valued distinct Player subject absent")
	}
	t.Run("distinct equal Players cannot share origin", func(t *testing.T) {
		mapping := maps.Clone(join)
		mapping[b.index] = mapping[a.index]
		copy, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		for i, p := range copy.Players {
			if p == join[b.index] {
				copy.Players[i] = join[a.index]
			}
		}
		if d := players1154DocumentDifferences(want, mapping, &copy); len(d) == 0 {
			t.Fatal("equal-valued distinct Player collapse accepted")
		}
	})
	var tagged []diary1154Raw
	for _, d := range want.diaries {
		if d.location.ArchiveIndex != 0 {
			tagged = append(tagged, d)
		}
	}
	if len(tagged) != 2 || tagged[0].location.ArchiveIndex == tagged[1].location.ArchiveIndex || !slices.Equal(tagged[0].dwords, tagged[1].dwords) || !slices.Equal(tagged[0].words, tagged[1].words) {
		t.Fatal("equal-valued distinct tagged Diary subject absent")
	}
	t.Run("distinct equal Diaries cannot share origin", func(t *testing.T) {
		mapping := maps.Clone(join)
		mapping[tagged[1].location.ArchiveIndex] = mapping[tagged[0].location.ArchiveIndex]
		copy, err := sav.CloneDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		owner := &copy.Objects[join[tagged[1].location.OwnerArchiveIndex]-1]
		for i := range owner.RefSlots {
			if owner.RefSlots[i].Name == "Diary" {
				owner.RefSlots[i].Objects = []uint16{mapping[tagged[0].location.ArchiveIndex]}
			}
		}
		if d := players1154DocumentDifferences(want, mapping, &copy); len(d) == 0 {
			t.Fatal("equal-valued distinct Diary collapse accepted")
		}
	})
	t.Run("true Diary alias remains one archive object", func(t *testing.T) {
		_, raw := players1154Fixture(t, false, true)
		source, _, _ := players1154Inputs(t, raw)
		var indices []uint16
		for _, d := range source.diaries {
			if d.location.ArchiveIndex != 0 {
				indices = append(indices, d.location.ArchiveIndex)
			}
		}
		if len(indices) != 2 || indices[0] != indices[1] {
			t.Fatal("true shared Diary identity changed", indices)
		}
	})
}

func TestPlayers1154SnapshotOriginsAfterInsertion(t *testing.T) {
	for _, aliases := range []bool{false, true} {
		name := "distinct equal Diaries"
		if aliases {
			name = "shared Diary"
		}
		t.Run(name, func(t *testing.T) {
			_, raw := players1154Fixture(t, true, aliases)
			want, original, source := players1154Inputs(t, raw)
			// Equal Player fields include equal Participant selectors, so this
			// identity fixture is not a playable session. Supply explicit,
			// distinct stable bindings independently of those scalar values.
			baseline := &SnapshotSAVDocument{Document: &source, GroupBindings: &SnapshotSAVGroupBindings{}}
			for i, object := range source.Objects {
				switch object.Class {
				case "Player":
					baseline.GroupBindings.Players = append(baseline.GroupBindings.Players, SnapshotSAVGroupPlayerBinding{ID: uint32(i + 1), ObjectIndex: uint16(i + 1)})
				case "Human", "Unit", "Humanoid":
					baseline.Actors = append(baseline.Actors, SnapshotSAVActor{EntityID: sim.EntityID(i + 1), ObjectIndex: uint16(i + 1)})
				}
			}
			copy, err := sav.CloneDocumentData(source)
			if err != nil {
				t.Fatal(err)
			}
			current := &SnapshotSAVDocument{Document: &copy, Actors: slices.Clone(baseline.Actors), GroupBindings: &SnapshotSAVGroupBindings{Players: slices.Clone(baseline.GroupBindings.Players)}}
			// A literal new pack object changes canonical traversal without
			// changing any expected Player or Diary value. The permutation
			// builds the input fixture; the observer never receives it.
			item, err := sav.NewDocumentRecord("Item")
			if err != nil {
				t.Fatal(err)
			}
			newGroupSetValue1115(t, &item, "F40", 0x0e1e)
			newGroupSetValue1115(t, &item, "F42", 1)
			current.Document.Objects = append(current.Document.Objects, item)
			owner := &current.Document.Objects[current.Actors[0].ObjectIndex-1]
			refs, _ := savedObjectRefs(owner, "Inventory")
			savedObjectSetValue(owner, "HasInventory", 1)
			savedObjectSetValue(owner, "Inventory1C", 0)
			savedObjectSetValue(owner, "Inventory20", 0)
			savedObjectSetRefs(owner, "Inventory", append(slices.Clone(refs), uint16(len(current.Document.Objects))), true)
			doc, permutation, err := sav.ReindexDocumentData(*current.Document)
			if err != nil {
				t.Fatal(err)
			}
			current.Document = &doc
			for i := range current.Actors {
				current.Actors[i].ObjectIndex = permutation[current.Actors[i].ObjectIndex]
			}
			for i := range current.GroupBindings.Players {
				p := &current.GroupBindings.Players[i]
				p.ObjectIndex = permutation[p.ObjectIndex]
			}
			for i := range current.GroupBindings.Members {
				m := &current.GroupBindings.Members[i]
				m.ObjectIndex = permutation[m.ObjectIndex]
			}
			if slices.Equal(baseline.Document.Players, doc.Players) {
				t.Fatal("insertion did not move a Player subject")
			}
			if _, err := sav.EncodeDocumentData(doc); err != nil {
				t.Fatal("inserted graph is not encodable", err)
			}
			check := func() []string {
				join, err := players1154SnapshotOrigins(want, original, baseline, current)
				if err != nil {
					return []string{err.Error()}
				}
				return players1154DocumentDifferences(want, join, current.Document)
			}
			if d := check(); len(d) != 0 {
				t.Fatal("insertion changed Player/Diary correspondence", d)
			}
			if d := players1154DocumentDifferences(want, original, current.Document); len(d) == 0 {
				t.Fatal("stale-index negative control lost its subject")
			}
			players := slices.Clone(current.GroupBindings.Players)
			current.GroupBindings.Players[1].ObjectIndex = players[0].ObjectIndex
			if d := check(); len(d) == 0 {
				t.Fatal("distinct equal Players collapsed after reindex")
			}
			current.GroupBindings.Players = players
			if !aliases {
				var tagged []diary1154Raw
				for _, d := range want.diaries {
					if d.location.Off >= 0 && d.location.ArchiveIndex != 0 {
						tagged = append(tagged, d)
					}
				}
				if len(tagged) != 2 {
					t.Fatal("distinct tagged Diary subjects absent")
				}
				join, err := players1154SnapshotOrigins(want, original, baseline, current)
				if err != nil {
					t.Fatal(err)
				}
				owner := &current.Document.Objects[join[tagged[1].location.OwnerArchiveIndex]-1]
				savedObjectSetRefs(owner, "Diary", []uint16{join[tagged[0].location.ArchiveIndex]}, false)
				if d := check(); len(d) == 0 {
					t.Fatal("distinct equal Diaries collapsed after reindex")
				}
			}
		})
	}
}

func TestPlayers1154LiveSubsetAndOmissionControls(t *testing.T) {
	f, raw := players1154Fixture(t, false, false)
	want, join, _ := players1154Inputs(t, raw)
	ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	check := func(world *sim.World, state *SnapshotSAVDocument) []string {
		d, _ := players1154LiveDifferences(want, join, state, world)
		return d
	}
	// manaOwners is 3, not 2, and excluded is empty, not one entry: this
	// fixture's third F58 owner binds to the first retained map unit, whose
	// live EntityID is genuinely 0 (originalActorRegistry.go), not "unbound".
	// The comparison's entities map fix (comma-ok, not ==0) reclassified it
	// correctly; the omission path itself stays covered by the real corpus
	// (TestMilestone2Players reports genuine "no live actor binding" and
	// "N direct Player containers" exclusions from actual save files).
	if d, p := players1154LiveDifferences(want, join, ms.savedDocument, ms.World); len(d) != 0 || p.players != 2 || p.unavailable != 2 || p.playerDiaries != 1 || p.actorDiaries != 2 || p.manaOwners != 3 || len(p.excluded) != 0 {
		t.Fatal("supported live subset", d, p)
	}
	for _, name := range []string{"Player Diary", "actor Diary", "entry value", "length", "duplicate owner", "wrong owner"} {
		t.Run(name, func(t *testing.T) {
			diaries := ms.World.SavedDiaries()
			before := ms.World.SavedDiaries()
			defer ms.World.SetSavedDiaries(before)
			switch name {
			case "Player Diary":
				diaries = slices.DeleteFunc(diaries, func(d sim.SavedDiary) bool { return d.Owner.Player })
			case "actor Diary":
				diaries = slices.DeleteFunc(diaries, func(d sim.SavedDiary) bool { return !d.Owner.Player })
			case "entry value":
				diaries[0].Entries[0].Count++
			case "length":
				diaries[0].Length++
			case "duplicate owner":
				diaries = append(diaries, diaries[0])
			case "wrong owner":
				diaries[0].Owner = diaries[1].Owner
			}
			ms.World.SetSavedDiaries(diaries)
			if len(check(ms.World, ms.savedDocument)) == 0 {
				t.Fatal("live Diary mutation accepted")
			}
		})
	}
	t.Run("missing exact Player binding", func(t *testing.T) {
		state, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		state.GroupBindings.Players = state.GroupBindings.Players[:1]
		if len(check(ms.World, state)) == 0 {
			t.Fatal("missing Player accepted")
		}
	})
	t.Run("same Slot cannot supply two purses", func(t *testing.T) {
		state, err := cloneSavedDocument(ms.savedDocument)
		if err != nil {
			t.Fatal(err)
		}
		state.PlayerPurses.Players[0].Unavailable = ""
		if len(check(ms.World, state)) == 0 {
			t.Fatal("ambiguous purse ownership accepted")
		}
	})
}

func TestPlayers1154RawDiaryBounds(t *testing.T) {
	// Extended counts and an independent word-array length are raw grammar,
	// not a typed Diary's equal-length projection.
	b := binary.LittleEndian.AppendUint16(nil, 0xffff)
	b = binary.LittleEndian.AppendUint32(b, 1)
	b = binary.LittleEndian.AppendUint32(b, 7)
	b = binary.LittleEndian.AppendUint16(b, 2)
	b = binary.LittleEndian.AppendUint16(b, 1017)
	b = binary.LittleEndian.AppendUint16(b, 55)
	b = binary.LittleEndian.AppendUint32(b, 0xabcdef01)
	r := player1154Reader{body: b}
	d := r.diary(sav.DocumentDiaryLocation{Off: 0})
	if r.err != nil || len(d.dwords) != 4 || len(d.words) != 4 || d.self != 0xabcdef01 || d.end != len(b) {
		t.Fatal(d, r.err)
	}
	for n := 0; n < len(b); n++ {
		r := player1154Reader{body: b[:n]}
		r.diary(sav.DocumentDiaryLocation{Off: 0})
		if r.err == nil {
			t.Fatalf("accepted Diary truncation at %d", n)
		}
	}
	for _, count := range []uint32{1<<20 + 1, 0xffffffff} {
		b := binary.LittleEndian.AppendUint16(nil, 0xffff)
		b = binary.LittleEndian.AppendUint32(b, count)
		r := player1154Reader{body: b}
		r.diary(sav.DocumentDiaryLocation{Off: 0})
		if r.err == nil {
			t.Fatal("accepted oversized count", count)
		}
	}
}

func TestPlayers1154RawScopeAndF58Controls(t *testing.T) {
	f, raw := players1154Fixture(t, false, false)
	want, join, _ := players1154Inputs(t, raw)
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	groups, err := file.DocumentPlayerGroupLocations()
	if err != nil {
		t.Fatal(err)
	}
	objects, err := file.DocumentObjectLocations()
	if err != nil {
		t.Fatal(err)
	}
	byOff, byIndex := map[int]sav.DocumentObjectLocation{}, map[uint16]sav.DocumentObjectLocation{}
	playersByOff := map[int]uint16{}
	for _, loc := range objects {
		byOff[loc.Off], byIndex[loc.ArchiveIndex] = loc, loc
	}
	for index, p := range want.players {
		playersByOff[p.off] = index
	}
	t.Run("missing Group structure", func(t *testing.T) {
		copy := want
		copy.memberContainers = map[uint16][]int{}
		r := player1154Reader{body: file.Body}
		if err := copy.readGroups(&r, groups[1:], playersByOff, byOff, byIndex); err == nil {
			t.Fatal("missing Group location hid raw population")
		}
	})
	t.Run("missing actor reference structure", func(t *testing.T) {
		copy := want
		copy.memberContainers = map[uint16][]int{}
		locations := slices.Clone(groups)
		for i, g := range locations {
			if len(g.ActorRefOffs) > 0 {
				locations[i].ActorRefOffs = g.ActorRefOffs[1:]
				break
			}
		}
		r := player1154Reader{body: file.Body}
		if err := copy.readGroups(&r, locations, playersByOff, byOff, byIndex); err == nil {
			t.Fatal("missing actor tag location hid raw population")
		}
	})
	t.Run("raw F58 differs through exact owner despite equal Slot", func(t *testing.T) {
		ms, _, err := ResumeOriginalSave(f.Archives.Containers, raw, f.Table, f.Difficulty, nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		owner := want.players[want.roots[3]]
		at := owner.off + 1 + len(owner.name) + 39 // SAV-PLAYER-028's widths up to F58.
		binary.LittleEndian.PutUint32(file.Body[at:], owner.values["F58"]+1)
		changed, err := players1154Expected(file)
		if err != nil {
			t.Fatal(err)
		}
		if changed.players[owner.index].values["F58"] != owner.values["F58"]+1 {
			t.Fatal("raw F58 control missed source field")
		}
		differences, _ := players1154LiveDifferences(changed, join, ms.savedDocument, ms.World)
		found := false
		for _, d := range differences {
			if len(d) >= 3 && d[:3] == "F58" {
				found = true
			}
		}
		if !found {
			t.Fatal("acceptance missed source F58/live actor-basis difference", differences)
		}
	})
}
