package sav

import (
	"bytes"
	"reflect"
	"testing"

	"againrom/internal/synth"
)

// ---------------------------------------------------------------------------
// Group 1: the embedded state store.
// ---------------------------------------------------------------------------

// storeFixture is a small well-formed registry shaped like the shipped city
// state store: one scalar before the array under test in heap order (Speed),
// the array itself (Selection), and one heap-resident sibling after it
// (Shortcuts), so a length-changing edit can be shown to shift only what
// follows it.
func storeFixture() []byte {
	return synth.Reg(0x11, []synth.RegNode{
		{Name: "GameOptions", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Speed", Kind: 0x02, Int: 8},
		}},
		{Name: "Objects", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Selection", Kind: 0x06, Ints: []int32{1}},
		}},
		{Name: "SpellBook", Kind: 0x01, Children: []synth.RegNode{
			{Name: "Shortcuts", Kind: 0x06, Ints: []int32{-1, -1, -1, -1}},
		}},
	})
}

func storeFile(t *testing.T) (*File, []byte) {
	t.Helper()
	s := standard()
	s.tail = append(append([]byte(nil), storeFixture()...), campaignDWords(0)...)
	f := open(t, s)
	if f.Store == nil {
		t.Fatal("fixture tail did not frame as a state store")
	}
	return f, append([]byte(nil), f.Body...)
}

func TestSetStoreIntChangesOnlyTheStoreLeaf(t *testing.T) {
	f, bodyBefore := storeFile(t)
	labelBefore := append([]byte(nil), f.LabelRegion...)
	tailBefore := append([]byte(nil), f.TailRest...)

	if err := f.SetStoreInt("GameOptions", "Speed", 0); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Body, bodyBefore) {
		t.Fatal("a state-store edit disturbed the campaign body")
	}
	if !bytes.Equal(f.LabelRegion, labelBefore) || !bytes.Equal(f.TailRest, tailBefore) {
		t.Fatal("a state-store edit disturbed the label region or the campaign tail")
	}
	store, ok := f.StateStore()
	if !ok {
		t.Fatal("state store lost after edit")
	}
	if v, ok := store.GetInt("GameOptions", "Speed"); !ok || v != 0 {
		t.Fatalf("GameOptions/Speed reads %d, %v", v, ok)
	}
	if a, ok := store.GetIntArray("Objects", "Selection"); !ok || !reflect.DeepEqual(a, []int32{1}) {
		t.Fatalf("Objects/Selection reads %v, %v, want [1] unchanged", a, ok)
	}

	// Marshal/reopen: the same file the story's boundedness proof asks for.
	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	if v, ok := back.StateStore(); !ok {
		t.Fatal("reopened file lost its state store")
	} else if got, ok := v.GetInt("GameOptions", "Speed"); !ok || got != 0 {
		t.Fatalf("reopened GameOptions/Speed = %d, %v", got, ok)
	}
}

func TestSetStoreIntArrayReflowsOnlyTheHeapAfterIt(t *testing.T) {
	f, bodyBefore := storeFile(t)
	labelBefore := append([]byte(nil), f.LabelRegion...)
	tailBefore := append([]byte(nil), f.TailRest...)

	if err := f.SetStoreIntArray("Objects", "Selection", nil); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(f.Body, bodyBefore) {
		t.Fatal("a state-store edit disturbed the campaign body")
	}
	if !bytes.Equal(f.LabelRegion, labelBefore) || !bytes.Equal(f.TailRest, tailBefore) {
		t.Fatal("a state-store edit disturbed the label region or the campaign tail")
	}
	store, ok := f.StateStore()
	if !ok {
		t.Fatal("state store lost after edit")
	}
	if a, ok := store.GetIntArray("Objects", "Selection"); !ok || len(a) != 0 {
		t.Fatalf("Objects/Selection reads %v, %v, want empty", a, ok)
	}
	// The field before it in heap order is untouched at the decoded level...
	if v, ok := store.GetInt("GameOptions", "Speed"); !ok || v != 8 {
		t.Fatalf("GameOptions/Speed reads %d, %v, want 8 unchanged", v, ok)
	}
	// ...and so is the one after it, even though the heap reflowed under it.
	if sc, ok := store.GetIntArray("SpellBook", "Shortcuts"); !ok || !reflect.DeepEqual(sc, []int32{-1, -1, -1, -1}) {
		t.Fatalf("SpellBook/Shortcuts reads %v, %v, want [-1 -1 -1 -1] unchanged", sc, ok)
	}
	if len(f.Store) != len(storeFixture())-4 { // one fewer int32 in the heap
		t.Fatalf("Store is %d bytes, want %d fewer than the fixture's %d", len(f.Store), 4, len(storeFixture()))
	}

	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	bs, _ := back.StateStore()
	if sc, ok := bs.GetIntArray("SpellBook", "Shortcuts"); !ok || !reflect.DeepEqual(sc, []int32{-1, -1, -1, -1}) {
		t.Fatalf("reopened SpellBook/Shortcuts = %v, %v", sc, ok)
	}
}

func TestSetStoreEditsRefuseAMissingStoreOrWrongType(t *testing.T) {
	f := open(t, standard()) // the standard fixture's tail is unframed: no store
	if err := f.SetStoreInt("GameOptions", "Speed", 0); err == nil {
		t.Fatal("SetStoreInt on a file with no state store was accepted")
	}
	if err := f.SetStoreIntArray("Objects", "Selection", nil); err == nil {
		t.Fatal("SetStoreIntArray on a file with no state store was accepted")
	}
	f2, _ := storeFile(t)
	if err := f2.SetStoreInt("Objects", "Selection", 0); err == nil {
		t.Fatal("SetStoreInt on an IntArray leaf was accepted")
	}
	if err := f2.SetStoreIntArray("GameOptions", "Speed", nil); err == nil {
		t.Fatal("SetStoreIntArray on an Int leaf was accepted")
	}
}

// ---------------------------------------------------------------------------
// Group 2: Head.MapName (variable length; decoded-level bound).
// ---------------------------------------------------------------------------

func TestSetMapNameShiftsEveryOffsetAfterItAndNothingElse(t *testing.T) {
	for _, name := range []string{"", "a", "10.alm", "a-very-long-map-name.alm"} {
		t.Run(name, func(t *testing.T) {
			f := open(t, standard())
			labelBefore := append([]byte(nil), f.LabelRegion...)
			storeBefore := append([]byte(nil), f.Store...)
			tailBefore := append([]byte(nil), f.TailRest...)
			wantMission, wantDiff, wantPlayers := f.Head.Mission, f.Head.Difficulty, len(f.Players)
			wantMoney, wantOutcome := f.Players[0].Money, f.Players[1].Outcome

			if err := f.SetMapName(name); err != nil {
				t.Fatal(err)
			}
			if f.Head.MapName != name {
				t.Fatalf("MapName reads %q, want %q", f.Head.MapName, name)
			}
			// Everything the map name's length does not own is unchanged at
			// the DECODED level, even though its byte position moved.
			if f.Head.Mission != wantMission || f.Head.Difficulty != wantDiff || len(f.Players) != wantPlayers {
				t.Fatalf("head/players %+v", f.Head)
			}
			if f.Players[0].Money != wantMoney || f.Players[1].Outcome != wantOutcome {
				t.Fatalf("player fields moved: money=%d outcome=%d", f.Players[0].Money, f.Players[1].Outcome)
			}
			// The label region, the store and the tail sit AFTER the whole
			// compressed body, so a body-internal length change must not
			// touch any of them.
			if !bytes.Equal(f.LabelRegion, labelBefore) {
				t.Fatal("SetMapName disturbed the label region")
			}
			if !bytes.Equal(f.Store, storeBefore) || !bytes.Equal(f.TailRest, tailBefore) {
				t.Fatal("SetMapName disturbed the store or the campaign tail")
			}

			back, err := Open(f.Marshal())
			if err != nil {
				t.Fatal(err)
			}
			if back.Head.MapName != name || back.Head.Mission != wantMission {
				t.Fatalf("reopened head %+v", back.Head)
			}
		})
	}
}

func TestSetMapNameRefusesAnExtendedName(t *testing.T) {
	f := open(t, standard())
	long := make([]byte, 0xff)
	for i := range long {
		long[i] = 'x'
	}
	if err := f.SetMapName(string(long)); err == nil {
		t.Fatal("a 255-byte map name (the extended-CString marker) was accepted")
	}
}

// ---------------------------------------------------------------------------
// Groups 3 and 4: the campaign record.
// ---------------------------------------------------------------------------

func campaignFile(t *testing.T) (*File, CampaignProjection) {
	t.Helper()
	want, body := campaignProjectionFixture()
	s := standard()
	s.tail = append(append([]byte(nil), storeFixture()...), body...)
	f := open(t, s)
	if f.Store == nil {
		t.Fatal("fixture tail did not frame as a state store")
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("fixture campaign record did not parse: ok=%v err=%v", ok, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fixture campaign = %#v, want %#v", got, want)
	}
	return f, want
}

func TestSetCampaignScalarChangesOnlyThatScalar(t *testing.T) {
	f, want := campaignFile(t)
	bodyBefore := append([]byte(nil), f.Body...)
	storeBefore := append([]byte(nil), f.Store...)

	if err := f.SetCampaignScalar(2, 0); err != nil { // AutoGetMission
		t.Fatal(err)
	}
	if !bytes.Equal(f.Body, bodyBefore) || !bytes.Equal(f.Store, storeBefore) {
		t.Fatal("a campaign scalar edit disturbed the body or the store")
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after edit: ok=%v err=%v", ok, err)
	}
	if got.AutoGetMission != 0 {
		t.Fatalf("AutoGetMission = %d, want 0", got.AutoGetMission)
	}
	want.AutoGetMission = 0
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after edit = %#v, want %#v (every OTHER field unchanged)", got, want)
	}

	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	bc, _, err := back.Campaign()
	if err != nil || bc.AutoGetMission != 0 {
		t.Fatalf("reopened AutoGetMission = %d, err=%v", bc.AutoGetMission, err)
	}
}

func TestSetCampaignBaseDWordChangesOnlyThatDWord(t *testing.T) {
	f, want := campaignFile(t)
	if err := f.SetCampaignBaseDWord(1, 0); err != nil { // Main.MapObject
		t.Fatal(err)
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after edit: ok=%v err=%v", ok, err)
	}
	if got.Main.MapObject != 0 {
		t.Fatalf("Main.MapObject = %d, want 0", got.Main.MapObject)
	}
	want.Main.MapObject = 0
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after edit = %#v, want %#v", got, want)
	}
}

func TestSetCampaignArrayReflowsOnlyWhatFollowsIt(t *testing.T) {
	f, want := campaignFile(t)
	if err := f.SetCampaignArray(0, nil); err != nil { // Mercenaries -> empty
		t.Fatal(err)
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after edit: ok=%v err=%v", ok, err)
	}
	if len(got.Mercenaries) != 0 {
		t.Fatalf("Mercenaries = %v, want empty", got.Mercenaries)
	}
	want.Mercenaries = nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after edit = %#v, want %#v (every OTHER field's own decoded value unchanged)", got, want)
	}

	back, err := Open(f.Marshal())
	if err != nil {
		t.Fatal(err)
	}
	bc, _, err := back.Campaign()
	if err != nil || !reflect.DeepEqual(bc.PermanentMercenaries, want.PermanentMercenaries) {
		t.Fatalf("reopened PermanentMercenaries = %v, err=%v", bc.PermanentMercenaries, err)
	}
}

func TestSetCampaignDocumentsChangesOnlyDocuments(t *testing.T) {
	f, want := campaignFile(t)
	if err := f.SetCampaignDocuments(nil); err != nil {
		t.Fatal(err)
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after edit: ok=%v err=%v", ok, err)
	}
	if len(got.Documents) != 0 {
		t.Fatalf("Documents = %v, want empty", got.Documents)
	}
	want.Documents = nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after edit = %#v, want %#v", got, want)
	}
}

func TestSetCampaignChildrenChangesOnlyChildren(t *testing.T) {
	f, want := campaignFile(t)
	if err := f.SetCampaignChildren(nil); err != nil {
		t.Fatal(err)
	}
	got, ok, err := f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after edit: ok=%v err=%v", ok, err)
	}
	if len(got.Children) != 0 {
		t.Fatalf("Children = %v, want none", got.Children)
	}
	got.Children, want.Children = nil, nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after edit = %#v, want %#v", got, want)
	}

	// And the reverse direction — growing from none — is bounded the same
	// way, which the shipped corpus never exercises (its Children are
	// already empty). Every one of the seven fields is given its OWN
	// distinct value here, and every one is checked below: this is what
	// pins campaignRecordToBase's own stated purpose (city_campaign.go),
	// the one place that maps CampaignRecord onto its six fixed dword
	// slots. Checking only two of eight fields, as this test once did,
	// cannot catch two dword slots swapped with each other.
	if err := f.SetCampaignChildren([]CampaignRecord{
		{Mission: 9, MapObject: 8, Payment: 7, ShopMin: 6, ShopMax: 5, Announced: true, Age: 4},
	}); err != nil {
		t.Fatal(err)
	}
	got, ok, err = f.Campaign()
	if err != nil || !ok {
		t.Fatalf("campaign after growing edit: ok=%v err=%v", ok, err)
	}
	if len(got.Children) != 1 {
		t.Fatalf("Children = %#v, want exactly one", got.Children)
	}
	if child := got.Children[0]; child.Mission != 9 || child.MapObject != 8 || child.Payment != 7 ||
		child.ShopMin != 6 || child.ShopMax != 5 || !child.Announced || child.Age != 4 {
		t.Fatalf("Children[0] = %#v, want {Mission:9 MapObject:8 Payment:7 ShopMin:6 ShopMax:5 Announced:true Age:4}", child)
	}
	got.Children = nil
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaign after growing edit = %#v, want %#v plus the one child", got, want)
	}
}

func TestCampaignEditsRefuseOutOfRangeAndOverBoundInput(t *testing.T) {
	f, _ := campaignFile(t)
	if err := f.SetCampaignScalar(7, 0); err == nil {
		t.Fatal("scalar index 7 (out of range) was accepted")
	}
	if err := f.SetCampaignScalar(4, 2); err == nil {
		t.Fatal("FirstMapPoint flag 2 was accepted")
	}
	if err := f.SetCampaignBaseDWord(6, 0); err == nil {
		t.Fatal("base dword index 6 (out of range) was accepted")
	}
	if err := f.SetCampaignBaseDWord(5, 2); err == nil {
		t.Fatal("Announced latch 2 was accepted")
	}
	if err := f.SetCampaignArray(6, nil); err == nil {
		t.Fatal("array index 6 (out of range) was accepted")
	}
	if err := f.SetCampaignDocuments([]CampaignDocument{{Kind: 2}}); err == nil {
		t.Fatal("document kind 2 was accepted")
	}
	empty := open(t, standard())
	if err := empty.SetCampaignScalar(0, 0); err == nil {
		t.Fatal("a campaign edit on a file with no campaign record was accepted")
	}
}
