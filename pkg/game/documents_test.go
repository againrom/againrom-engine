package game

import (
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/reg"
)

// documentCampaign is a scenario registry carrying the two document grant keys
// in the shape the shipped one has: one section granting three texts, one
// granting a picture beside a text, one granting neither, and one spelling the
// picture key at its FULL length.
//
// THE FULL-LENGTH SPELLING IS THE POINT OF THE LAST SECTION. Registry key
// names are truncated to fifteen characters, so the shipped file contains
// AddPictureDocum and never AddPictureDocument. A reader looking for the long
// name finds nothing, and a fixture that only ever writes the short name
// cannot tell the two readers apart.
func documentCampaign(t *testing.T) Campaign {
	t.Helper()
	raw := synth.Reg(kindRoot, []synth.RegNode{
		{Name: "General", Kind: kindDir, Children: []synth.RegNode{
			{Name: "TotalMissions", Kind: kindInt, Int: 4},
		}},
		{Name: "Mission10", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AddTextDocument", Kind: kindIntArray, Ints: []int32{1, 2, 3}},
		}},
		{Name: "Mission20", Kind: kindDir, Children: []synth.RegNode{
			{Name: "Mercenaries", Kind: kindInt, Int: 1},
		}},
		{Name: "Mission50", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AddPictureDocum", Kind: kindInt, Int: 1},
			{Name: "AddTextDocument", Kind: kindInt, Int: 4},
		}},
		{Name: "Mission60", Kind: kindDir, Children: []synth.RegNode{
			{Name: "AddPictureDocument", Kind: kindInt, Int: 9},
		}},
	})
	r, err := reg.Parse(raw)
	if err != nil {
		t.Fatalf("parse fixture: %v", err)
	}
	return ReadCampaign(r)
}

// REG-SCN-097's two keys reach Chapter, and the fifteen-character spelling is
// the one the reader answers to.
func TestTheCampaignReadsBothDocumentGrantKeys(t *testing.T) {
	c := documentCampaign(t)

	if want := []int{1, 2, 3}; !reflect.DeepEqual(c.Chapters[10].TextDocuments, want) {
		t.Errorf("[Mission10] TextDocuments = %v, want %v", c.Chapters[10].TextDocuments, want)
	}
	if got := c.Chapters[10].PictureDocuments; len(got) != 0 {
		t.Errorf("[Mission10] PictureDocuments = %v, want none", got)
	}
	if want := []int{4}; !reflect.DeepEqual(c.Chapters[50].TextDocuments, want) {
		t.Errorf("[Mission50] TextDocuments = %v, want %v", c.Chapters[50].TextDocuments, want)
	}
	if want := []int{1}; !reflect.DeepEqual(c.Chapters[50].PictureDocuments, want) {
		t.Errorf("[Mission50] PictureDocuments = %v, want %v", c.Chapters[50].PictureDocuments, want)
	}
	if got := c.Chapters[20]; len(got.TextDocuments) != 0 || len(got.PictureDocuments) != 0 {
		t.Errorf("[Mission20] grants %+v, want neither key", got)
	}
	if got := c.Chapters[60].PictureDocuments; len(got) != 0 {
		t.Errorf("[Mission60] spells the key at full length and yields %v, want none: "+
			"registry key names are truncated to fifteen characters", got)
	}
}

// B1: the collection grows once per mission, forward only, and a section
// carrying neither key still moves the guard.
func TestCollectDocumentsGrowsOnceAndOnlyForward(t *testing.T) {
	town := NewTown(documentCampaign(t))

	if got := town.Documents(); got != nil {
		t.Fatalf("a new town already holds %v", got)
	}
	if got := town.DocumentMission(); got != 0 {
		t.Fatalf("a new town's guard is %d, want 0", got)
	}

	town.CollectDocuments(10)
	first := []Document{{1, DocumentText}, {2, DocumentText}, {3, DocumentText}}
	if got := town.Documents(); !reflect.DeepEqual(got, first) {
		t.Fatalf("after mission 10 the collection is %v, want %v", got, first)
	}

	// Replaying the same mission grants nothing a second time.
	town.CollectDocuments(10)
	if got := town.Documents(); !reflect.DeepEqual(got, first) {
		t.Errorf("replaying mission 10 made the collection %v, want %v", got, first)
	}

	// A section with neither key moves the guard and adds nothing.
	town.CollectDocuments(20)
	if got := town.DocumentMission(); got != 20 {
		t.Errorf("after mission 20 the guard is %d, want 20", got)
	}
	if got := town.Documents(); !reflect.DeepEqual(got, first) {
		t.Errorf("mission 20 grants neither key and made the collection %v", got)
	}

	// Text before picture within one section (spec.md B1).
	town.CollectDocuments(50)
	full := append(append([]Document{}, first...),
		Document{4, DocumentText}, Document{1, DocumentPicture})
	if got := town.Documents(); !reflect.DeepEqual(got, full) {
		t.Fatalf("after mission 50 the collection is %v, want %v", got, full)
	}

	// A mission BELOW the guard is refused outright, so re-entering an earlier
	// mission from a save grants nothing and does not wind the guard back.
	// The guard is read on the next statement rather than after another
	// forward call: a later CollectDocuments(50) would restore the guard's
	// value whether or not the guard exists, and the assertion would pass
	// against a town that has no guard at all.
	town.CollectDocuments(10)
	if got := town.DocumentMission(); got != 50 {
		t.Errorf("going back moved the guard to %d, want 50", got)
	}
	if got := town.Documents(); !reflect.DeepEqual(got, full) {
		t.Errorf("going back made the collection %v, want %v", got, full)
	}
}

// The append is deduplicated on the PAIR, so one value in both kinds is two
// elements and the same value twice in one kind is one.
func TestTheCollectionDeduplicatesOnThePairAndNotOnTheValue(t *testing.T) {
	town := NewTown(Campaign{})
	town.addDocument(Document{1, DocumentText})
	town.addDocument(Document{1, DocumentPicture})
	town.addDocument(Document{1, DocumentText})
	want := []Document{{1, DocumentText}, {1, DocumentPicture}}
	if got := town.Documents(); !reflect.DeepEqual(got, want) {
		t.Errorf("collection = %v, want %v", got, want)
	}
}

// Documents hands out a copy, so a caller cannot write into campaign state
// through a reader.
func TestDocumentsHandsOutACopy(t *testing.T) {
	town := NewTown(documentCampaign(t))
	town.CollectDocuments(10)
	got := town.Documents()
	got[0] = Document{99, DocumentPicture}
	if again := town.Documents(); again[0] != (Document{1, DocumentText}) {
		t.Errorf("writing into the returned slice changed the collection to %v", again)
	}
}

// B1's save half: the collection rides in grant order and comes back through
// the same append.
func TestTheSaveCarriesTheCollectionInGrantOrder(t *testing.T) {
	camp := documentCampaign(t)
	town := NewTown(camp)
	town.CollectDocuments(10)
	town.CollectDocuments(50)

	var s Snapshot
	snapshotTown(town, &s)
	if got, want := s.DocumentMission, 50; got != want {
		t.Errorf("Snapshot.DocumentMission = %d, want %d", got, want)
	}
	wantPairs := []SnapshotDocument{{1, DocumentText}, {2, DocumentText}, {3, DocumentText},
		{4, DocumentText}, {1, DocumentPicture}}
	if !reflect.DeepEqual(s.Documents, wantPairs) {
		t.Fatalf("Snapshot.Documents = %v, want %v", s.Documents, wantPairs)
	}

	back := restoreTown(camp, s)
	if !reflect.DeepEqual(back.Documents(), town.Documents()) {
		t.Errorf("restored collection = %v, want %v", back.Documents(), town.Documents())
	}
	if got := back.DocumentMission(); got != 50 {
		t.Errorf("restored guard = %d, want 50", got)
	}

	// A payload carrying the same pair twice restores as one element.
	s.Documents = append(s.Documents, SnapshotDocument{1, DocumentText})
	if got := restoreTown(camp, s).Documents(); !reflect.DeepEqual(got, town.Documents()) {
		t.Errorf("a duplicated pair restored as %v, want %v", got, town.Documents())
	}

	// An old payload declares neither field and restores an empty collection.
	if got := restoreTown(camp, Snapshot{}); got.Documents() != nil || got.DocumentMission() != 0 {
		t.Errorf("an empty payload restored %v at guard %d, want no collection",
			got.Documents(), got.DocumentMission())
	}
}

// DIV-306, SHOP-SELL-010's second test: the sell walk takes an element only
// when the customer owns it AND it carries a price. A place the shop will not
// pay for stays on the table.
//
// THE BASKET HOLDS BOTH KINDS, because the guard above the loop tests the
// basket's TOTAL: one priced place beside an unpriced one was enough to hand
// the unpriced one over for nothing.
func TestSellKeepsAPlaceTheShopWillNotPayFor(t *testing.T) {
	s := NewShop(1000)
	priced := ShopItem{Code: data.ComposeItemCode(0, 1, 0, 3), Price: 5, Count: 3}
	free := ShopItem{Code: data.QuestDocumentCode, Price: 0, Count: 1}
	s.PutOnTable(priced)
	s.PutOnTable(free)

	paid, ok := s.Sell()
	if !ok || paid != 8 {
		t.Fatalf("Sell = (%d, %v), want the priced stack's 8", paid, ok)
	}

	table := s.Table()
	if len(table) != 1 || table[0].Code != free.Code || table[0].Count != 1 || !table[0].Mine {
		t.Fatalf("after the sell the table holds %+v, want the unpriced place alone", table)
	}
	if got := s.Shelf(ShelfMagic); len(got) != 0 {
		t.Errorf("the Magic Items shelf holds %+v, want the unpriced place never to reach it", got)
	}
	if got := s.Shelf(ShelfWeapons); len(got) != 1 || got[0].Count != 3 {
		t.Errorf("the weapon shelf holds %+v, want the priced stack of three", got)
	}

	// A basket of nothing but unpriced places commits nothing at all.
	if _, ok := s.Sell(); ok {
		t.Error("a basket the shop will not pay for still committed")
	}
	if len(s.Table()) != 1 {
		t.Errorf("the refused sell changed the table to %+v", s.Table())
	}
}

func TestWonMissionGrantsItsDocumentsWithoutMissionEntry(t *testing.T) {
	town := NewTown(documentCampaign(t))
	town.Won(10)
	first := []Document{{1, DocumentText}, {2, DocumentText}, {3, DocumentText}}
	if got := town.Documents(); !reflect.DeepEqual(got, first) {
		t.Fatalf("after Won(10) the collection is %v, want %v", got, first)
	}
	if got := town.DocumentMission(); got != 10 {
		t.Errorf("after Won(10) the guard is %d, want 10", got)
	}
	s := Snapshot{}
	snapshotTown(town, &s)
	if len(s.Documents) != 3 {
		t.Errorf("snapshot carries %d documents, want 3", len(s.Documents))
	}
}

func TestWonMissionDocumentsAreIdempotentAndNotGuardBound(t *testing.T) {
	town := NewTown(documentCampaign(t))
	town.CollectDocuments(10)
	town.Won(10)
	first := []Document{{1, DocumentText}, {2, DocumentText}, {3, DocumentText}}
	if got := town.Documents(); !reflect.DeepEqual(got, first) {
		t.Fatalf("enter then win made the collection %v, want %v", got, first)
	}

	late := NewTown(documentCampaign(t))
	late.CollectDocuments(20)
	late.Won(10)
	if got := late.Documents(); !reflect.DeepEqual(got, first) {
		t.Fatalf("a win below the guard made the collection %v, want %v", got, first)
	}
	if got := late.DocumentMission(); got != 20 {
		t.Errorf("a win below the guard moved the guard to %d, want 20", got)
	}
}
