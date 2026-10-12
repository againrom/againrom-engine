package game

import (
	"fmt"
	"strconv"
	"strings"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapload"
)

// EventTextPath is the address campaign mission n's event number e is read at.
//
// The original composes this name from three pieces — a prefix, a format over
// the two numbers, and an extension — and the EVENT NUMBER IS TWO DIGITS while
// the mission number is not padded at all. That asymmetry is the format string's
// own and it is the whole of why this is a function rather than a concatenation
// at the call site: `m10/event01` and `m10/event1` are different entries, and
// only one of them ships.
//
// The leading identity segment is what dispatches the address to main.res rather
// than to the loose tree, and it is composed from the prefix this package spells
// once beside that archive's name.
//
// A number that is not positive names nothing on either axis and is refused
// HERE, before anything is opened, for the reason a mission number is: it is a
// caller's arithmetic reaching an archive as a lookup that merely happens to
// miss.
func EventTextPath(mission, event int) (string, bool) {
	if mission <= 0 || event <= 0 {
		return "", false
	}
	return fmt.Sprintf("%stext/battle/m%d/event%02d.txt", mainPrefix, mission, event), true
}

// ReadEventText reads the event text mission n's event number e names, and
// reports whether it is there.
//
// THERE IS NO ERROR RETURN, and that is the contract rather than a simplifying
// choice. A named event text that does not ship produces nothing at all in the
// original — no window, no fallback, no message, no fault, and no blocked state:
// the open is made purely to decide whether to proceed, its failure code is
// discarded, and the arm returns as if it had worked. Of the 242 numbers the 28
// shipped campaign scripts raise, 19 name no file, so silence is the authored
// behaviour of about one raise in thirteen and not a defect to report.
//
// An error return here would be an invitation to log, to fall back, or to fail a
// mission, and each of those shows the player something the original does not.
// Two values is what makes those unreachable from every call site at once.
func ReadEventText(src entrySource, mission, event int) ([]byte, bool) {
	if src == nil {
		return nil, false
	}
	addr, ok := EventTextPath(mission, event)
	if !ok {
		return nil, false
	}
	b, err := src.ReadFile(addr)
	if err != nil {
		return nil, false
	}
	return b, true
}

// The three literals this file looks for. THEY ARE SPELT ONCE AND IN LOWER CASE,
// because every test below is a case-folded containment and a needle carrying a
// capital would simply never match.
//
// speakerMark AND speakerTag DIFFER BY ONE CHARACTER AND THAT IS THE POINT. The
// original asks two different questions with them — whether the window has a
// portrait pane at all, and whether this part changes whose face is in it — and
// a reader who noticed only that they are nearly the same would collapse them
// into one and pass every check the shipped corpus can make.
const (
	partMark    = "part="
	speakerMark = "npc"
	speakerTag  = "npc="
)

// ReservedMessageNumber is the one message number a script may not use for
// an announcement.
//
// The mission-lost path posts the same window message carrying this value, and
// the handler compares the number against it before composing any address, so
// the two are indistinguishable at the first instruction that inspects either.
// A script raising it therefore shows the lost-mission panel and opens no file.
//
// THE SHIPPED CAMPAIGN NEVER REACHES IT: the 254 announcements the 28 campaign
// maps raise carry numbers 0..25. It is a limit on what may be authored, not a
// behaviour any shipped map has.
const ReservedMessageNumber int32 = 255

// asciiFold is b with an ASCII capital folded to lower case and every other byte
// left EXACTLY as it is.
//
// IT IS ASCII AND NOT UNICODE, and that is a correctness requirement rather than
// a shortcut. A payload is the game's own byte string in the game's own code
// page, so it is not UTF-8: running it through a Unicode case mapping decodes
// every Cyrillic byte as an error rune and REPLACES it, changing the bytes and
// the length of the very thing being searched. Nothing this file looks for is
// outside ASCII, so folding only ASCII is both sufficient and lossless.
func asciiFold(b byte) byte {
	if b >= 'A' && b <= 'Z' {
		return b + ('a' - 'A')
	}
	return b
}

// containsFold reports whether s contains lower, comparing with ASCII case
// folded. lower MUST already be lower case.
//
// It walks rather than allocating a folded copy, so the file test can be asked
// of a whole payload without copying it.
func containsFold(s, lower string) bool {
	if len(lower) == 0 {
		return true
	}
	for i := 0; i+len(lower) <= len(s); i++ {
		j := 0
		for ; j < len(lower); j++ {
			if asciiFold(s[i+j]) != lower[j] {
				break
			}
		}
		if j == len(lower) {
			return true
		}
	}
	return false
}

// indexFold is the first index in s at which lower occurs with ASCII case
// folded, or -1. lower MUST already be lower case.
func indexFold(s, lower string) int {
	for i := 0; i+len(lower) <= len(s); i++ {
		j := 0
		for ; j < len(lower); j++ {
			if asciiFold(s[i+j]) != lower[j] {
				break
			}
		}
		if j == len(lower) {
			return i
		}
	}
	return -1
}

// EventAudience is who the eight conditional tag arms test.
//
// THE HERO HALF IS TWO FACTS AND THE SPEAKER HALF IS A FUNCTION, because the
// two are asked at different granularities. The player's own hero is the same
// person for every tag of every part of the file; the speaker is whoever the
// tag's own `npc=` names, so two tags of one part can have two speakers.
//
// Speaker answers three things at once: the two facts, and whether the four
// speaker arms may run at all. A nil Speaker, or one reporting resolved ==
// false, is the gate failing, and a failed gate skips all four arms rather than
// rejecting the tag — a tag carrying `female` alone is then accepted by any
// hero.
//
// THE ZERO VALUE IS A LEGITIMATE AUDIENCE and not a bypass: a male fighter with
// no speaker resolvable. That is what makes one selection function safe to call
// everywhere. A second function that skipped the arms would show a part the
// player had been refused, and no call site could tell which of the two it
// wanted.
//
// tags is how the game's tag bodies are read, the campaign service's answer;
// nil reads them as the first game does.
type EventAudience struct {
	HeroFemale bool
	HeroMage   bool
	tags       eventTags
	NPCPresent func(npc int) bool

	Speaker func(npc int) (female, mage, resolved bool)
}

// HeroAudience is the audience a party makes: its SUBJECT's sex and class, with
// no speaker resolvable.
//
// THE SUBJECT IS THE STARTING HERO, the member the start marked as the player's
// own character, whatever slot a join or a load moved him to; the speaker
// records' `Me`, `MySex` and `MyClass` derive every hero speaker's sex and class
// from the same member (owner direction, DIV-3149). A party with no marked
// member answers from its first member.
//
// An empty party is a male fighter, because the four `iam*` arms are answered
// from the two bits alone and there is no third value to give them. A mission
// started with no party shows no dialogue anyway.
func HeroAudience(party []mapload.PartyMember) EventAudience {
	if len(party) == 0 {
		return EventAudience{}
	}
	subject := party[0]
	for _, m := range party {
		if primaryPlayerHero(m) {
			subject = m
			break
		}
	}
	return EventAudience{
		HeroFemale: data.FigureDir(subject.FigureDir).Female(),
		HeroMage:   subject.Mage,
	}
}

// speakerAudience gives a hero audience the speaker half the four bare arms
// need: the sex and class of the record a tag's `npc=` number names, and the
// gate that decides whether those arms run at all (SC-2, SC-3).
//
// THE GATE IS A `Start` KEY OR A `Start` FLAGS TOKEN. The key is the
// original's reading (DLG-TAGARM-027) and no shipped section has one. The token
// is the owner's: npc21 to npc24 carry it, so each speaks in its own sex derived
// from the starting hero (DIV-3149). Any other record resolves nothing, so the
// four arms are skipped and no tag is rejected on its account.
//
// THE FACTS COME FROM THE RECORD'S FLAG TOKENS, handing each axis to the
// player's hero where it says to: `MySex` and `Me` for the sex, `MyClass` and
// `Me` for the class, as at every other reader of the same list (DLG-SYNTH-042).
func speakerAudience(hero EventAudience, faces map[int32]data.NPCFace) EventAudience {
	hero.Speaker = func(npc int) (female, mage, resolved bool) {
		rec, ok := faces[int32(npc)]
		if !ok || !rec.Start && !rec.Archetype {
			return false, false, false
		}
		female = rec.Tokens.Has(data.NPCTokenFemale)
		if rec.Tokens.Has(data.NPCTokenMySex) || rec.Tokens.Has(data.NPCTokenMe) {
			female = hero.HeroFemale
		}
		if rec.Tokens.Has(data.NPCTokenNotMySex) {
			female = !hero.HeroFemale
		}
		mage = rec.Tokens.Has(data.NPCTokenMage)
		if rec.Tokens.Has(data.NPCTokenMyClass) || rec.Tokens.Has(data.NPCTokenMe) {
			mage = hero.HeroMage
		}
		if rec.Tokens.Has(data.NPCTokenNotMyClass) {
			mage = !hero.HeroMage
		}
		return female, mage, true
	}
	return hero
}

// eventArm is one conditional literal: which subject it tests, which fact, the
// value that fact must have for the tag to survive, and a literal whose presence
// anywhere in the body skips the arm entirely.
type eventArm struct {
	lit     string
	speaker bool // the subject is the speaker rather than the player's hero
	sex     bool // the fact is the sex bit rather than the class bit
	want    bool
	guard   string
}

// eventArms is the eight arms in the order they are applied.
//
// EVERY TEST IS A SUBSTRING TEST, and an arm that does not reject falls
// through to the next. The four speaker arms test the body with the four `iam*`
// literals removed (DIV-3149): in the original `iamfemale` also satisfies
// `female` (DLG-TAGARM-027), which with the speaker gate open would close every
// shipped `iamfemale`/`iammale` pair spoken by a companion of the other sex.
//
// ONE GUARD EXISTS AND IT IS THE ORIGINAL'S OWN: the `male` arm is skipped for
// any body containing `female`, which every `female` body does contain as a
// substring. Without it no tag naming a female could ever survive, since
// `female` ends in `male`.
var eventArms = []eventArm{
	{lit: "iamfemale", sex: true, want: true},
	{lit: "iammale", sex: true, want: false},
	{lit: "iammage", want: true},
	{lit: "iamfighter", want: false},
	{lit: "female", speaker: true, sex: true, want: true},
	{lit: "male", speaker: true, sex: true, want: false, guard: "female"},
	{lit: "mage", speaker: true, want: true},
	{lit: "fighter", speaker: true, want: false},
}

// accepts reports whether a tag body survives the eight arms.
//
// The speaker's own facts are resolved AT MOST ONCE per tag and only if a
// speaker arm is reached, so a file using none of the four costs no lookup.
func (a EventAudience) accepts(tag string) bool { return a.tagRules().accepts(a, tag) }

// eventTags is how a game's tag bodies are read: whether a tag survives an
// audience, and the speaker number a part's tag names.
type eventTags interface {
	accepts(a EventAudience, tag string) bool
	speaker(tag string) (int, bool)
}

// firstEventTags read the eight arms with the speaker gate.
type firstEventTags struct{}

// secondEventTags read the four hero arms and the npcalive= and npcdead=
// gates.
type secondEventTags struct{}

func (a EventAudience) tagRules() eventTags {
	if a.tags == nil {
		return firstEventTags{}
	}
	return a.tags
}

func (secondEventTags) speaker(tag string) (int, bool) { return secondGameTagNumber(tag, speakerTag) }

func (firstEventTags) speaker(tag string) (int, bool) { return tagSpeaker(tag) }

func (secondEventTags) accepts(a EventAudience, tag string) bool {
	for _, arm := range eventArms[:4] {
		if !containsFold(tag, arm.lit) {
			continue
		}
		have := a.HeroFemale
		if !arm.sex {
			have = a.HeroMage
		}
		if have != arm.want {
			return false
		}
	}
	for _, gate := range []struct {
		key  string
		want bool
	}{{"npcalive=", true}, {"npcdead=", false}} {
		if npc, named := secondGameTagNumber(tag, gate.key); named {
			present := a.NPCPresent != nil && a.NPCPresent(npc)
			if present != gate.want {
				return false
			}
		}
	}
	return true
}

func (firstEventTags) accepts(a EventAudience, tag string) bool {
	var sexF, mageF, gate, asked bool
	bare := withoutHeroArms(tag)
	for _, arm := range eventArms {
		body := tag
		if arm.speaker {
			body = bare
		}
		if !containsFold(body, arm.lit) {
			continue
		}
		if arm.guard != "" && containsFold(body, arm.guard) {
			continue
		}
		have := a.HeroFemale
		if !arm.sex {
			have = a.HeroMage
		}
		if arm.speaker {
			if !asked {
				sexF, mageF, gate = a.speakerFacts(tag)
				asked = true
			}
			if !gate {
				continue
			}
			if have = sexF; !arm.sex {
				have = mageF
			}
		}
		if have != arm.want {
			return false
		}
	}
	return true
}

// withoutHeroArms is a tag body with every `iam*` literal cut out, the text the
// four speaker arms test.
func withoutHeroArms(tag string) string {
	for _, arm := range eventArms[:4] {
		for {
			i := indexFold(tag, arm.lit)
			if i < 0 {
				break
			}
			tag = tag[:i] + "," + tag[i+len(arm.lit):]
		}
	}
	return tag
}

// speakerFacts is the four speaker arms' subject: the sex and class of whoever
// this tag's own `npc=` names, and whether the gate passes at all.
func (a EventAudience) speakerFacts(tag string) (female, mage, resolved bool) {
	if a.Speaker == nil {
		return false, false, false
	}
	npc, named := tagSpeaker(tag)
	if !named {
		return false, false, false
	}
	return a.Speaker(npc)
}

// eventPartTag is the tag that opens part n and the text that follows it.
//
// THE CONTENT MODEL IS A TAG SCAN, NOT A GRAMMAR, and this reproduces the scan.
// The payload is walked for `<`...`>`; each tag's body is tested with ASCII case
// folded for whether it CONTAINS `part=<n>`; the first tag in file order that
// does AND survives the audience's arms wins, and the part is the bytes from
// that tag's `>` to the next `<`, or to the end of the payload when no tag
// follows.
//
// A REJECTED TAG DOES NOT END THE SEARCH. The original's scan resumes at the
// next tag and returns nothing only at the terminator, so a part is absent
// exactly when no tag naming it survives — which is how the shipped
// `iamfemale`/`iammale` pairs are authored: two tags, one part, one audience
// each.
//
// ONE WALK ANSWERS BOTH QUESTIONS A PART IS ASKED — what it says, and who says
// it. A second scan for the speaker would be a second answer to "which tag is
// part n's", free to pick a different tag from the one the words came from on
// exactly the inputs where the containment above is doing something surprising.
func eventPartTag(payload []byte, n int, aud EventAudience) (tag, body string, ok bool) {
	tag, body, ok = eventPartTail(payload, n, aud)
	if end := strings.IndexByte(body, '<'); end >= 0 {
		body = body[:end]
	}
	return
}

func eventPartTail(payload []byte, n int, aud EventAudience) (tag, body string, ok bool) {
	if n < 0 {
		return "", "", false
	}
	want := partMark + strconv.Itoa(n)
	s := string(payload)
	for i := 0; i < len(s); {
		open := strings.IndexByte(s[i:], '<')
		if open < 0 {
			return "", "", false
		}
		open += i
		close := strings.IndexByte(s[open:], '>')
		if close < 0 {
			// An unterminated tag is the end of the scan: there is no body to
			// test and nothing after it can be a part, because a part begins at
			// a '>' this payload does not have.
			return "", "", false
		}
		close += open

		if containsFold(s[open+1:close], want) && aud.accepts(s[open+1:close]) {
			return s[open+1 : close], s[close+1:], true
		}
		i = close + 1
	}
	return "", "", false
}

// EventHasSpeaker reports whether an event text mentions a speaker ANYWHERE
// in it — which is what decides whether its window carries a portrait
// pane, once, for every part of the file.
//
// IT LOOKS FOR THREE LETTERS AND NOT FOUR, and that is the original's test rather
// than a slip. The constructor folds the case of the WHOLE payload — prose
// included — and asks whether it contains `npc`, where the per-part test asks for
// `npc=`. So a file whose prose happens to contain those three letters and whose
// tags name nobody still opens the portrait layout, showing a pane that never
// gets a face. Narrowing it to `npc=` would agree with the original on every
// shipped file and diverge on an authored one, which is the wrong direction for a
// reimplementation to diverge in.
//
// It is asked of the PAYLOAD and not of a part, so paging cannot change the shape
// of an open window.
func EventHasSpeaker(payload []byte) bool {
	return containsFold(string(payload), speakerMark)
}

// EventPartSpeaker is the speaker part n names, and whether it names one at
// all.
//
// THE TEST IS THE PART'S OWN TAG AND IT IS THE FOUR-CHARACTER ONE. Where
// EventHasSpeaker decides the window's shape from the whole file, this decides
// whose face stands in the pane from one part, and the original asks the two
// questions with two different needles. A part whose tag carries no `npc=` names
// nobody — which is not an omission but the instruction to LEAVE THE STANDING
// FACE ALONE, and the caller is what holds it.
//
// THE NUMBER AND THE NAMING ARE REPORTED SEPARATELY, because the original's
// flag is set by the containment alone. A tag reading `npc=` with no digits
// after it therefore still names a speaker — the pane is refreshed — for
// a speaker whose number is absent, which resolves to no picture.
//
// The number is the run of decimal digits immediately following, and it stops at
// the first byte that is not one: `npc=21,` is 21. A run too long for the type is
// no number, not a wrapped one.
func EventPartSpeaker(payload []byte, n int, aud EventAudience) (speaker int, named bool) {
	tag, _, ok := eventPartTag(payload, n, aud)
	if !ok {
		return 0, false
	}
	return aud.tagRules().speaker(tag)
}

// ROM2's header scan parses a five-byte substring, rather than the complete
// numeric run. The native atoi-style prefix admits whitespace and a sign.
func secondGameTagNumber(tag, key string) (int, bool) {
	i := indexFold(tag, key)
	if i < 0 {
		return 0, false
	}
	s := tag[i+len(key):]
	if len(s) > 5 {
		s = s[:5]
	}
	s = strings.TrimLeft(s, " \t\r\n\v\f")
	n := 0
	if len(s) > 0 && (s[0] == '+' || s[0] == '-') {
		n++
	}
	digits := n
	for n < len(s) && s[n] >= '0' && s[n] <= '9' {
		n++
	}
	if n == digits {
		return 0, true
	}
	v, _ := strconv.Atoi(s[:n])
	return v, true
}

// tagSpeaker is the `npc=` number one tag body names, split out because the four
// speaker arms need it of a tag the scan has not accepted yet — the gate is part
// of deciding whether to accept it.
func tagSpeaker(tag string) (speaker int, named bool) {
	i := indexFold(tag, speakerTag)
	if i < 0 {
		return 0, false
	}
	digits := tag[i+len(speakerTag):]
	end := 0
	for end < len(digits) && digits[end] >= '0' && digits[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, true
	}
	v, err := strconv.Atoi(digits[:end])
	if err != nil {
		return 0, true
	}
	return v, true
}

// EventPart is the content of part n of an event text, and whether the payload
// holds one.
//
// THE SUBSTRING TEST IS REPRODUCED RATHER THAN CORRECTED. Because the match is a
// containment rather than an equality, a tag reading `part=10` also satisfies a
// search for part 1 — and since the first match wins, a file whose part 10 came
// first would answer a request for part 1 with the part-10 body. The shipped
// corpus never trips it: no part number on either root is a prefix of an earlier
// part number in the same file. Correcting it would diverge from the original on
// authored data where the original does not, which is the wrong direction for a
// reimplementation to diverge in.
//
// THE EIGHT CONDITIONAL ARMS ARE NOW READ — see EventAudience and eventArms —
// and so is the speaker, off the same tag this returns the text of. Town
// dialogue speech consumes sound= separately (speech.go, DLG-SOUND-028).
// Other control tags still terminate the text body, as any tag does.
//
// EventPart is the generic raw tag-body reader. Resource dialogue delivery
// separately selects its LF/CR/NUL pointer interval before wrapping.
// The bytes come back as a Go string and are NOT decoded: they are the game's
// own byte string in the game's own arrangement, which is precisely what the
// draw path's byte-to-glyph rule expects. Transcoding here would corrupt every
// Cyrillic byte on the way to a renderer that is built to receive it raw.
func EventPart(payload []byte, n int, aud EventAudience) (string, bool) {
	_, body, ok := eventPartTag(payload, n, aud)
	return body, ok
}

// MissionTipPath is the archive path of tip n of the mission, from the
// edition's tip table (TRIG-TIPS-087); false when the table has no mission
// tips.
func MissionTipPath(e base.Edition, mission, n int) (string, bool) {
	return TipTable(e).MissionTip(mission, n)
}
