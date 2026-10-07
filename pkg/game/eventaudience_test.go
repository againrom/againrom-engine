package game

import (
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// The eight conditional tag arms and the reserved message number (AC-3,
// AC-4).
//
// Every payload here is written in this file. No install is opened and no test
// below reads one (golden rule 2).

// heroAud is one of the four heroes the four `iam*` arms can distinguish.
func heroAud(female, mage bool) EventAudience {
	return EventAudience{HeroFemale: female, HeroMage: mage}
}

// speakerAud is heroAud with a speaker table attached, so the four bare arms
// have a subject and a gate to pass.
func speakerAud(female, mage bool, faces map[int32]data.NPCFace) EventAudience {
	return speakerAudience(heroAud(female, mage), faces)
}

// TestTheHeroArmsSelectExactlyOnePartOfAPair is AC-3.
//
// The payload is the shape the shipped files author: two tags for one part,
// one per sex, and no other difference. Exactly one survives for each of the
// four heroes, and the class does not move the answer — which is the second
// half of the claim, since a sex arm that read the class bit would still pass
// a two-hero test.
func TestTheHeroArmsSelectExactlyOnePartOfAPair(t *testing.T) {
	t.Parallel()

	const payload = "<part=1,iamfemale>she<part=1,iammale>he"
	for _, tc := range []struct {
		female, mage bool
		want         string
	}{
		{false, false, "he"},
		{false, true, "he"},
		{true, false, "she"},
		{true, true, "she"},
	} {
		got, ok := EventPart([]byte(payload), 1, heroAud(tc.female, tc.mage))
		if !ok || got != tc.want {
			t.Errorf("hero female=%v mage=%v got %q, %v; want %q",
				tc.female, tc.mage, got, ok, tc.want)
		}
	}
}

// TestTheClassArmsSelectOnClassAlone is AC-3's class half, on the same shape.
func TestTheClassArmsSelectOnClassAlone(t *testing.T) {
	t.Parallel()

	const payload = "<part=1,iammage>spell<part=1,iamfighter>sword"
	for _, tc := range []struct {
		female, mage bool
		want         string
	}{
		{false, false, "sword"},
		{true, false, "sword"},
		{false, true, "spell"},
		{true, true, "spell"},
	} {
		got, ok := EventPart([]byte(payload), 1, heroAud(tc.female, tc.mage))
		if !ok || got != tc.want {
			t.Errorf("hero female=%v mage=%v got %q, %v; want %q",
				tc.female, tc.mage, got, ok, tc.want)
		}
	}
}

func TestARejectedTagDoesNotEndTheSearch(t *testing.T) {
	t.Parallel()

	// The refused tag is FIRST for a male hero.
	if got, ok := EventPart([]byte("<part=1,iamfemale>she<part=1,iammale>he"), 1,
		heroAud(false, false)); !ok || got != "he" {
		t.Errorf("got %q, %v; want the second tag's body", got, ok)
	}
	// And a later part still resolves past a refused tag of an earlier one.
	payload := []byte("<part=1,iamfemale>she<part=1,iammale>he<part=2>both")
	if got, ok := EventPart(payload, 2, heroAud(false, false)); !ok || got != "both" {
		t.Errorf("part 2 = %q, %v; want %q", got, ok, "both")
	}
}

// TestAPartWithNoSurvivingTagIsAbsent is AC-4's first half.
func TestAPartWithNoSurvivingTagIsAbsent(t *testing.T) {
	t.Parallel()

	// Both tags for part 1 demand a female; the hero is not one.
	payload := []byte("<part=1,iamfemale>she<part=1,iamfemale>her<part=2>after")
	if got, ok := EventPart(payload, 1, heroAud(false, false)); ok {
		t.Errorf("part 1 = %q, %v; want absent — no tag naming it survives", got, ok)
	}
	// The file is not otherwise damaged: part 2 is still there.
	if got, ok := EventPart(payload, 2, heroAud(false, false)); !ok || got != "after" {
		t.Errorf("part 2 = %q, %v; want %q", got, ok, "after")
	}
}

func TestTheFourIamArmsSatisfyTheFourBareOnes(t *testing.T) {
	t.Parallel()

	faces := map[int32]data.NPCFace{
		1: {Start: true, Tokens: tokens(data.NPCTokenFemale)},
		2: {Start: true},
	}
	// A female hero, an `iamfemale` tag: accepted by the female speaker.
	if got, ok := EventPart([]byte("<part=1,npc=1,iamfemale>hers"), 1,
		speakerAud(true, false, faces)); !ok || got != "hers" {
		t.Errorf("female speaker: got %q, %v; want %q", got, ok, "hers")
	}
	// The same tag read by the same hero, with a MALE speaker, is refused: the
	// body contains `female`, so the speaker's own sex arm runs too.
	if got, ok := EventPart([]byte("<part=1,npc=2,iamfemale>hers"), 1,
		speakerAud(true, false, faces)); ok {
		t.Errorf("male speaker: got %q, %v; want the tag refused", got, ok)
	}
}

func TestTheMaleArmIsGuardedByFemale(t *testing.T) {
	t.Parallel()

	faces := map[int32]data.NPCFace{
		1: {Start: true, Tokens: tokens(data.NPCTokenFemale)},
	}
	if got, ok := EventPart([]byte("<part=1,npc=1,female>hers"), 1,
		speakerAud(false, false, faces)); !ok || got != "hers" {
		t.Errorf("got %q, %v; want %q — the male arm must be skipped for a body containing female",
			got, ok, "hers")
	}
	// And a bare `male` body, with no `female` in it, still tests the speaker.
	if _, ok := EventPart([]byte("<part=1,npc=1,male>his"), 1,
		speakerAud(false, false, faces)); ok {
		t.Error("a male-tagged part was shown for a female speaker")
	}
}

func TestTheSpeakerArmsAreGated(t *testing.T) {
	t.Parallel()

	const payload = "<part=1,npc=1,female>hers"
	for _, tc := range []struct {
		name  string
		faces map[int32]data.NPCFace
	}{
		{"no speaker table at all", nil},
		{"the record is absent", map[int32]data.NPCFace{9: {Start: true}}},
		{"the record carries no Start key", map[int32]data.NPCFace{1: {Tokens: tokens(data.NPCTokenFemale)}}},
	} {
		if got, ok := EventPart([]byte(payload), 1,
			speakerAud(false, false, tc.faces)); !ok || got != "hers" {
			t.Errorf("%s: got %q, %v; want the tag accepted", tc.name, got, ok)
		}
	}
	// A nil Speaker function is the gate failing too, which is what the zero
	// audience is: every call site that states no speaker gets this.
	if got, ok := EventPart([]byte(payload), 1, EventAudience{}); !ok || got != "hers" {
		t.Errorf("zero audience: got %q, %v; want the tag accepted", got, ok)
	}
}

// TestTheSpeakerTakesTheHeroWhereItsRecordSaysTo is SC-3, the authored half.
//
// `MySex` hands the sex axis to the player's own hero and `MyClass` the class
// axis; `Me` hands over both. Nothing shipped reaches this, because the gate
// above does not open on shipped data.
func TestTheSpeakerTakesTheHeroWhereItsRecordSaysTo(t *testing.T) {
	t.Parallel()

	faces := map[int32]data.NPCFace{
		1: {Start: true, Tokens: tokens(data.NPCTokenMySex)},
		2: {Start: true, Tokens: tokens(data.NPCTokenMe)},
	}
	// A female hero makes the MySex speaker female, so a `female` tag survives.
	if _, ok := EventPart([]byte("<part=1,npc=1,female>hers"), 1,
		speakerAud(true, false, faces)); !ok {
		t.Error("MySex speaker: the female tag was refused for a female hero")
	}
	if _, ok := EventPart([]byte("<part=1,npc=1,female>hers"), 1,
		speakerAud(false, false, faces)); ok {
		t.Error("MySex speaker: the female tag was shown for a male hero")
	}
	// `Me` carries the class axis as well.
	if _, ok := EventPart([]byte("<part=1,npc=2,mage>spell"), 1,
		speakerAud(false, true, faces)); !ok {
		t.Error("Me speaker: the mage tag was refused for a mage hero")
	}
	if _, ok := EventPart([]byte("<part=1,npc=2,mage>spell"), 1,
		speakerAud(false, false, faces)); ok {
		t.Error("Me speaker: the mage tag was shown for a fighter hero")
	}
}

// TestTheSpeakerTakesTheOppositeOfTheHeroWhereItsRecordNegates covers the
// negated tokens: a `!MySex` speaker is the opposite sex of the hero and a
// `!MyClass` speaker the other class.
func TestTheSpeakerTakesTheOppositeOfTheHeroWhereItsRecordNegates(t *testing.T) {
	t.Parallel()

	faces := map[int32]data.NPCFace{
		1: {Start: true, Tokens: tokens(data.NPCTokenNotMySex)},
		2: {Start: true, Tokens: tokens(data.NPCTokenNotMyClass)},
	}
	const sex = "<part=1,npc=1,female>she<part=1,npc=1,male>he"
	for _, tc := range []struct {
		heroFemale bool
		want       string
	}{{false, "she"}, {true, "he"}} {
		got, ok := EventPart([]byte(sex), 1, speakerAud(tc.heroFemale, false, faces))
		if !ok || got != tc.want {
			t.Errorf("!MySex speaker, hero female=%v: got %q, %v; want %q", tc.heroFemale, got, ok, tc.want)
		}
	}
	const class = "<part=1,npc=2,mage>spell<part=1,npc=2,fighter>sword"
	for _, tc := range []struct {
		heroMage bool
		want     string
	}{{false, "spell"}, {true, "sword"}} {
		got, ok := EventPart([]byte(class), 1, speakerAud(false, tc.heroMage, faces))
		if !ok || got != tc.want {
			t.Errorf("!MyClass speaker, hero mage=%v: got %q, %v; want %q", tc.heroMage, got, ok, tc.want)
		}
	}
}

func TestPartSelectionIsAFunctionOfItsArgumentsAlone(t *testing.T) {
	t.Parallel()

	payload := []byte("<part=1,iamfemale>she<part=1,iammale>he<part=2>both")
	before := string(payload)
	aud := []EventAudience{heroAud(false, false), heroAud(true, false), heroAud(true, true)}
	want := []string{"he", "she", "she"}
	for pass := 0; pass < 3; pass++ {
		// Ask in a different order each pass; the answers may not move.
		for i := range aud {
			j := (i + pass) % len(aud)
			if got, ok := EventPart(payload, 1, aud[j]); !ok || got != want[j] {
				t.Fatalf("pass %d, audience %d = %q, %v; want %q", pass, j, got, ok, want[j])
			}
		}
	}
	if string(payload) != before {
		t.Fatalf("the payload was written to: %q, was %q", payload, before)
	}
}

// tokens is a record's flag set built from the exported bits, so a fixture
// names the tokens it means rather than a registry line that has to be parsed.
func tokens(list ...data.NPCToken) data.NPCTokens {
	var set data.NPCTokens
	for _, tok := range list {
		set |= data.NPCTokens(tok)
	}
	return set
}

// ------------------------------------------------------- the driver's own arms

// missionAudienceDriver is missionDialogue with a party and a raised number of
// this test's choosing, so the arms and the reserved number are driven through
// the production path rather than through EventPart alone.
func missionAudienceDriver(t *testing.T, event int32, payload string,
	party []mapload.PartyMember) (*mapWorld, *ui.Viewer) {
	t.Helper()
	w := missionWorld(t,
		[]sim.ScriptInstant{{Op: sim.ScriptInstantMessage, Args: [10]int32{event}}},
		[]sim.ScriptTrigger{missionTrigger(3, 1, true, 0)})
	m := worldFixtureMap()
	v := worldFixtureViewer(t, m)
	v.SetFont(missionFont())
	src := missionSource{}
	if payload != "" {
		src[missionEvent(t, 7, int(event))] = []byte(payload)
	}
	mw := openMission(&Mission{Number: 7, Map: m, World: w, Party: party,
		Raises: []mapload.ScriptRaise{{Latch: 3, Event: event}}}, nil, nil, v, src, nil, nil)
	missionSteps(mw, 2)
	return mw, v
}

// womanMage and manFighter are the two party subjects the tests below use.
func womanMage() []mapload.PartyMember {
	return []mapload.PartyMember{{FigureDir: string(data.FigureDirWomanMage), Mage: true}}
}

func manFighter() []mapload.PartyMember {
	return []mapload.PartyMember{{FigureDir: string(data.FigureDirManFighter)}}
}

func TestTheMissionPanelShowsThePartItsOwnHeroReceives(t *testing.T) {
	const payload = "<part=1,iamfemale>\r\nshe\r\n<part=1,iammale>\r\nhe"
	for _, tc := range []struct {
		name  string
		party []mapload.PartyMember
		want  string
	}{
		{"a woman mage", womanMage(), "she"},
		{"a man fighter", manFighter(), "he"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, v := missionAudienceDriver(t, 4, payload, tc.party)
			got, kind, open := v.NoticeState()
			if !open || kind != ui.NoticeDialogue || got != tc.want {
				t.Fatalf("notice = %q/%v/%v, want %q as a dialogue", got, kind, open, tc.want)
			}
		})
	}
}

// TestThePanelDoesNotOpenWhenNoTagSurvives is AC-4's second half.
//
// The file ships and is read; every tag naming part 1 refuses this hero. The
// original's scan produces nothing at that point, so nothing opens — the same
// silence an absent file gives, arrived at for a different reason.
func TestThePanelDoesNotOpenWhenNoTagSurvives(t *testing.T) {
	mw, v := missionAudienceDriver(t, 4, "<part=1,iamfemale>\r\nshe", manFighter())
	if _, _, open := v.NoticeState(); open {
		t.Fatal("a notice opened over a part this hero was refused")
	}
	if mw.mission.open {
		t.Fatal("the driver considers a window open")
	}
}

func TestPagingReAppliesTheSameAudience(t *testing.T) {
	mw, v := missionAudienceDriver(t, 4, "<part=1>\r\nfirst\r\n<part=2,iamfemale>\r\nsecond", manFighter())
	if got, _, open := v.NoticeState(); !open || got != "first" {
		t.Fatalf("notice = %q, open=%v; want the first part", got, open)
	}
	mw.advanceNotice()
	if got, _, open := v.NoticeState(); open {
		t.Fatalf("the window paged to %q; want it closed — part 2 refuses this hero", got)
	}
}

func TestTheReservedMessageNumberOpensTheLostNotice(t *testing.T) {
	if ReservedMessageNumber != 255 {
		t.Fatalf("ReservedMessageNumber = %d, want 255", ReservedMessageNumber)
	}
	mw, v := missionAudienceDriver(t, ReservedMessageNumber, "", nil)
	got, kind, open := v.NoticeState()
	if !open || kind != ui.NoticeFailure {
		t.Fatalf("notice = %q/%v/%v, want the outcome notice", got, kind, open)
	}
	if !mw.mission.announced {
		t.Fatal("the driver did not record the outcome as announced")
	}
	if mw.mission.outcome != sim.OutcomeLost {
		t.Fatalf("outcome = %v, want lost", mw.mission.outcome)
	}
}

func TestTheReservedNumberChangesNoSimulationState(t *testing.T) {
	mw, _ := missionAudienceDriver(t, ReservedMessageNumber, "", nil)
	if got := mw.world.Outcome(); got != sim.OutcomeUndecided {
		t.Fatalf("the simulation's outcome is %v, want undecided", got)
	}
}
