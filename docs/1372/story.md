# ROM2 inn stage 40 to 110 entries

## Intent

Before this story the ROM2 inn offered its own entries only at stages 10 and
30 (DIV-2634). A mission-30 departure raises the stage to 40, where the town 2
inn offered no mission. This story implements every remaining EnterInn stage
body that knowledge k206 publishes, each with its kind, topic, NPC, bank gates
and current-record gates. These offers and talks are conditional on reaching
stages 40 to 110. A new ROM2 campaign still stops at mission 30: it cannot be
won without companions (DIV-2391). A kind 3 TALK admits its mission; a kind 0
TALK shows its text and changes nothing. ROM1 code paths and hashed state do
not change in the inn implementation. Companions are out of scope.

Base: `772868a6` (game 0.100.6). Knowledge pin: k205, moved to k207.
Reconciled main: `ba0b7ef3` (game 0.101.0, knowledge k207).

## Authority

| Part | Claims |
|---|---|
| stage 40 body: topics 40, 48 (kind 0), 41, 42, 43; slots 937..939 | R2-ENGINE-223 |
| stage 50 body: topic 49 (kind 0), 51 behind 533 and 947, 53 behind 949 | R2-ENGINE-224 |
| shared 60/70/80 body: nine kind 3 stores, current ID 2 or 3, exact stage, bank gates; 966 also skips 71 | R2-ENGINE-225 |
| stage 90 body: 90 for current ID 2; 91 and 92 for current ID 3 behind 987 and 988 | R2-ENGINE-226 |
| stage 100 body: 100, 102 behind 998, 103 behind 987 nonzero and 999 for current ID 2; 101 behind 997 for current ID 3 | R2-ENGINE-227 |
| stage 110 body: 110 for current ID 2 | R2-ENGINE-228 |
| kind 3 TALK admits type 1 ID topic; no bank store with initial record ID 10 | R2-ENGINE-229 |
| kind 0 topics 48 and 49 change no state; npc%dtalk%d keys | R2-ENGINE-230 (High for the state, Medium for presentation) |
| Leave30 adds 10 to slot 768 as a DWORD; Leave31 and Leave32 keep it | R2-SESSION-111 |
| named routes through stages 30 to 70, postponing mission 50 | R2-SESSION-112 (Medium) |
| slot 768 selects the body for any town; town 3 at the same stage | R2-SESSION-113 |
| scripts 35 and 36 never write slots 768 or 775 | R2-SESSION-114 |
| ten stage labels, eight bodies, stage 20 has none | R2-ENGINE-161 |
| stage 30 body; stage 30 TALK | R2-ENGINE-216, R2-ENGINE-219 |
| stage 30 at the first town 2 visit, 40 after mission 30 | R2-SESSION-107 |

## As built

- `secondInnStages` (`pkg/game/secondinn.go`) holds every stage body as an
  ordered list of `secondInnEntry`: the packed option, a current record ID
  (0 for any), an exact stage (0 for any), and the slots that must be zero
  and nonzero. Stages 60, 70 and 80 share one list. Stage 20 and unlisted
  stages have no entry. The stage 30 slot 927 gate moved from code into the
  table.
- `innOptions` emits the selected body's admitted entries in store order, then
  the continuation stores and the dynamic tail as before. It changes no
  state.
- `talkTo` is unchanged: kind 3 admits mission topic; topic 10 stores slot
  769 and NPC 22 topic 30 stores 533 and 553; every other kind 3 topic and
  every kind 0 topic store nothing.
- Departure was already R2-SESSION-111's algebra: an ID divisible by ten adds
  10 to slot 768, others keep it. A new test pins it over wrapping and signed
  inputs.
- The census producer reads the stage table: "stage N inn talk", or "stage
  60/70/80 inn talk" for the two shared-body topics no exact stage gates.
- The census reach walk now wins one mission per round and revisits every
  available town between wins, so each stage a departure reaches gets its
  own visit. Before this story the walk won every available mission in one
  round; with the stage 60 gates that skipped topics 61 and 63.

Installed mission 30's victory trigger names a companion that the fresh
engine party does not supply. A new ROM2 campaign cannot win mission 30 or
reach stage 40 through ordinary play (DIV-2391); map 30 keeps its census
`party` blocker. The stage 40 witness applies the controller's departure of
mission 30 as a disclosed constructed step, bypassing victory admission.

## Divergences

- DIV-2634 is closed and moved to `docs/DIVERGENCES-CLOSED.md`: every stage
  body is published and implemented.
- DIV-2664 (new): the engine lacks the character-choice producer for slots
  776 and 781, whose native writers are published by R2-SESSION-023 (High)
  and R2-SESSION-109 (Medium). The departure prefix in `secondcampaign.go`
  normalizes slot 536 to 1 or 2 only when it is already nonzero; zero stays
  zero. A fresh nonzero producer remains missing. The census has 12 entry
  blockers: nine family alternatives (75..77, 85..87, 94..96), topics 81
  and 83, and topic 52 with no published admission.
- DIV-2632, DIV-2392, DIV-2356 and DIV-2639 point to the new behaviour.
- Reserved DIV-2665..DIV-2671 are unused.

## Proof

Focused tests (`pkg/game`):

- `TestSecondInnStageOptions`: the full list of every body at its zero-bank
  gates for towns 2 and 3, and selected gate settings, against lists written
  from the claims.
- `TestSecondInnStageGates`: each of the 26 gated stores offered with its
  predicates held, and withheld with any one inverted: zero slots set to 1
  and to -1, nonzero slots set to 0, another current record ID (1..4), and
  neighbouring stages.
- `TestSecondInnContinuationGates`: the continuation after the body's own
  stores.
- `TestSecondInnTalkEffects`: kind 0 topics 48 and 49 change neither the bank
  nor availability.
- `TestSecondInnLaterTalksAdmitTheirTopic`: every kind 3 store of stages 40
  to 110 (14 distinct-body stores plus the shared body under three labels)
  admits its topic once and stores no bank slot.
- `TestSecondLeaveThirtiesStage`: Leave30 maps s to s+10 as a DWORD for nine
  incoming stages including both signed extremes; Leave31 and Leave32 keep s.
- `TestSecondControllerReachesAndContinuesOnlyItsMissions`: the 34 reached
  maps and the producers of 40, 41, 52, 70, 83 and 101.

Installed witness, EN and RU ROM2 roots
(`pkg/game/secondstageforty_release_test.go`):

- `TestReleaseSecondStageFortyInnOffersItsMissions` plays a new game through
  missions 10 and 20 to the stage 30 inn and its two mission talks, constructs
  the departure of mission 30 with `completeBank` (output 2, stage 40), and
  enters town 2 and its inn. This bypasses the missing mission-30 victory.
  The rows are TALK 22, 2108, 2015, 2111, 2004 and GATES. A named SAV is
  written in the inn and cold-loaded in a fresh front end; the town sample
  and rows match. On both front ends the five talks run in order: TALK 22
  admits 40, TALK 2108 (kind 0) admits nothing and leaves the bank, TALK
  2015, 2111 and 2004 admit 41, 42 and 43. Mission 40 then opens on both
  with an equal world hash, campaign and party. On EN the talks show 16, 1,
  6, 7 and 4 dialogue pages. The camera is not compared after entry: the
  source front end carries scroll state from missions 10 and 20.

## Census

`TestReleaseSecondGameSupportCensus` on both ROM2 roots (46 maps), base
`772868a6` against this branch. EN and RU agree on every per-map row.
The census supplies wins, departures and satisfied departure gates. Its
conditional controller reach and map-local support counts do not establish
playable progression from a fresh campaign past mission 30 (DIV-2391).

| Total | Base | This story |
|---|---|---|
| engine_entry (reach) | 6 | 34 |
| maps blocked on entry | 40 | 12 |
| ready (no blocker) | 5: 10, 20, 21, 31, 32 | 8: adds 61, 82, 93 |
| maps blocked on party | 32 | 32 |
| maps blocked on spell | 5 | 5 |
| maps blocked on continuation | 1 (50) | 1 |
| headless-loss | 1 (101) | 1 |

Reached past 32: 40, 41, 42, 43, 50, 51, 53, 60, 61, 62, 63, 70, 71, 72, 73,
74, 80, 82, 84, 90, 91, 92, 93, 100, 101, 102, 103, 110. Still blocked on
entry: 52 (no claim admits it), 75..77, 85..87, 94..96 and 81 (slot 776/781
alternatives), 83 (slot 536) (DIV-2664).

## Open debt

- Mission 30 cannot be won without companions, so ordinary play does not
  reach stage 40 (DIV-2391).
- The engine character-choice producer for slots 776 and 781 is missing,
  despite the published native writers. Slot 536 has departure normalization
  but no fresh nonzero producer; topic 52 has no published admission
  (DIV-2664). Complete character-choice input binding and live bank-writer
  scheduling remain Unknown.
- The composed route and visit scheduling stay Medium (DIV-2632,
  R2-SESSION-112).
