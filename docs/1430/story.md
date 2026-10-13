# One speaker resolver

## Intent

The mission line, the tavern talk and the town voice pick the speaker by one
rule, companion first, as the mission line does. A speaker order or sex
correction made once reaches all of them. Started at public main `1e444902`
(pin k287); merged with main `80fd0d53` (pin k289).

## Authority

Owner decision 1 of the architecture audit. Claims `DLG-SPEAKER-023`,
`DLG-TAGARM-027`, `DLG-SYNTH-042`, `DLG-SOUND-028`, `DLG-VOICE-074`. Choice the
claims leave open: the candidate order (the original's is not decoded); the
companion who joined as the line's npc record is tried first (DIV-3150), now in
the tavern talk and the voice too. Row: DIV-3238. DIV-3239 to DIV-3245 are
returned unused.

## As built

- `playerCharacter` is the hero predicate, once (`speakeractors.go`);
  `guardedEntities`, the lying-hero filter in `frontend.go` and the party actor
  use it.
- `partyActor` builds a party member's candidate; `playerDirection` is the
  player's figure direction (the marked starting hero).
- `newSpeakerCast(party, world)` is the one cast constructor. With a
  `speakerWorld` (placements, party actors, liveness, worn set) it is the mission
  cast; without one the candidates are the party in order, all alive.
  Callers: `mapWorld.SpeakerFace` (mission), `tavernTalkCast` (tavern talk, with
  the stock units appended; `tavernSpeakerCast` slices them out), the voice
  leaf in `townSpeechPath`. The town resolver outside the tavern keeps the zero
  cast and takes only `playerDirection`.
- `partyAudience(party, faces)` is the one audience function; nil faces give the
  hero audience alone. Callers: `world.go`, `scenario.go` (with faces),
  `towndialogue.go`, `townscreen.go`, `secondcampaign.go` (without).
  `secondGameAudience` decorates the mission audience with the second game's tag
  rule and presence test; it builds no speaker half and remains.
- `resolveTavernTalk` and `townSpeechPath` call `resolveNPC`.

## Second game

The second game's install holds no npc face records (0 on `rom2-en` and
`rom2-ru`), so it has no speaker cast, tavern talk or hero voice leaf; the
census resolves 0 lines there. Its dialogue pages read `partyAudience` with the
same result as `HeroAudience`. Its own audience rules (tags, presence) stay.

## Proof

`TestReleaseSpeakerCastCensusOneResolver` resolves every installed mission line,
tavern talk part and voice leaf for four starting heroes and five join stages
through the production cast and through reference copies of the former builders.
EN: 2050 mission lines, none changed; 3168 tavern talk lines, 62 changed; 552
voice leaves, 62 changed. RU: 2070, none; 3960, 124; 984, 124. Every change is
npc23 (female hero) or npc24 (male mage), from Brian (party slot 2) to the
companion of that npc. The test fails on any other change, so it fails against
the former builders for these lines. `TestPlayerCharacterPredicateIsStatedOnce`
fails on a new copy of the predicate.

Gates on merge `d376e8e7` (main `638142e9`, pin k289): `gofmt -l` clean; `go test -trimpath -count=1 ./...` ok in 2m58s; one `check-release-tests.sh` invocation on EN and RU ok, 1061 of 1163 tests ran per root, 102 lacked a subject, no failure, 33m51s; `check-no-game-assets.sh` clean; `check-div-claims.sh` exit 0. The census re-run on `rom2-en` and `rom2-ru` finds 0 mission lines, 0 tavern talk lines, 0 voice leaves and 0 differences; the test skips there.

## Open debt

The original's candidate order is not decoded (DIV-3150, DIV-3238). The player
hero loop over `primaryPlayerHero` (13 sites) is a different predicate and stays.
