# The second game's mission losses show their own reasons

## Intent

Owner: the second game's mission losses show their own reasons, as in the
original. Every second-game mission loss takes its failure reason from the
event that caused it, and the loss notice shows that mission's failure text
for the reason. The generic notice stays only where the claims leave the
text Unknown. A won mission, a running mission, the first game and every SAV
byte are unchanged.

## Authority

Research EXP-2079, published in knowledge k372; the pin moves to k374. Claims
read with the pinned claim tool:

- R2-ENGINE-623 (High / Medium): Session +0xbe14 has 11 users per root:
  instant 5 stores its scalar, check 18 adds 1, the result tick reads it.
- R2-ENGINE-624 (High / Medium / Unknown): the result tick sends reason 2 when
  the owner has no hero actor or the hero's byte +0x13c is set; otherwise it
  sends +0xbe14 while it is above 0 and owner +0x40 is below 2.
- R2-ENGINE-625 (High / Medium): the death of a class 0x21..0x3f actor sends
  identity - 19: 2 for the hero (21), 3..11 for 22..30; hires send nothing.
- R2-ENGINE-627 (High / Medium / Unknown): check 18 adds 1 on each pass while
  its unit is in the death state; map 10 holds the one node, on unit 32.
- R2-ENGINE-629 (High / Medium / Unknown): the failure screen takes cache entry
  reason - 2 with no bound; entries run 2..1+E, slot 0x118+n for an empty
  2..4; what a reason outside them shows is Unknown.
- R2-ENGINE-630 (High / Medium): the 81 instant-5 nodes per root store 0 or
  5..9; only map 96's triggered 5 and map 40's untriggered 7 lie outside their
  cache; 214 entries per root.
- R2-ENGINE-631 (Medium / Unknown): the composed table of events, reasons and
  texts; the first send takes the report.

R2-ENGINE-626 and R2-ENGINE-628 bound the census of senders and were read
for completeness; they change no behaviour here.

## As built

Kinds and their one builder:

| Kind | Builder |
|---|---|
| Mission loss reason producer | `missionLossReason` in `pkg/game/world.go`, written by `latchOutcome` only |
| Party fall reason (the producer's fall input) | `partyFailureReason`, reading `Edition.HeroFailureReason` and `PartyIdentity.FailureReason` |
| Mission text failure sections reader | `secondGameFailureEntries`; `secondGameFailureText` returns its labels |
| Loss notice screen | `showOutcome` and the campaign's `outcomeText`, unchanged except for the reason it reads |
| Second-game census | `SecondGameCensus`, extended with four columns |

- `latchOutcome` is the one writer of a mission's outcome. Every former
  writer goes through it: the settle step (a guarded fall or the world's
  outcome), message 255 in both games, a restored world's outcome, the
  restore of a lost mission and the victory cheat. A loss takes the reason
  `missionLossReason` gives its cause and keeps it; a restore derives it
  again (DIV-4225).
- `missionLossReason`: a guarded character's fall gives the reason his party
  identity carries in the edition; any other loss, and a fall with no reason,
  gives the script's stored reason (instant 5 sets it, check 18 counts it).
  The edition gives the hero 2, identity 22 3 (the edition row that story 1528
  added, now the producer's input) and identity 23 4 (DIV-4224).
- `outcomeText` reads `lossReason` and no script counter; a reason outside
  the entries keeps the generic body (DIV-4223).
- The census gains `fall_reasons`, `failure_entries`, `slot_reasons` and
  `count_checks`, and checks fall reasons against the entries.
- `internal/archtest/lossreason.go`: in `pkg/game` only `latchOutcome` writes
  a mission's `outcome` or `lossReason`, and it takes `lossReason` from
  `missionLossReason`; only `partyFailureReason` reads the edition's failure
  reasons; `outcomeText` calls no `ScriptCounters`.
- DIV-4189 is closed: R2-ENGINE-625 publishes the producer the owner's ruling
  on Iglez matched.

The reason is presentation state. The World, its hash and the SAV do not
carry it.

## Proof

Census over the 46 maps of rom2-en and rom2-ru (both equal): 214 failure
entries per root, slot entries 3 and 4 in 10 and 20 and 4 in 30, 32 and 40,
as R2-ENGINE-630 counts. Fall reasons 2, 3 and 4 have an entry on every map.
Instant 5 values without an entry: map 40's 7 and map 96's 5. Check 18: one
node, map 10. The existing census totals do not move (`no_reason_text=2`).

Focused tests, `pkg/game/lossreason_test.go`, synthetic data: the hero's fall
(2), identity 22 (3), identity 23 (4), a hire (0, generic), instant 5 with 5
(entry), instant 5 with 7 past four entries (generic), check 18 on a dead unit
(1, generic), a fall on the step instant 5 stores 5 (the fall's 4), and a
fall after the script's loss (the script's 5 stays).

Frames, headless, `MissionOpenerWith` with the hero, identity 22 and identity
23, the notice picture after the event, base `3c0dbfc3a` and head:

| Event | Mission | Base | Head | World hash |
|---|---|---|---|---|
| hero killed | 10 | generic | failure2 | equal |
| identity 22 killed | 30 | failure3 | failure3 | equal |
| identity 23 killed | 41 | generic | failure4 | equal |
| instant 5 (5) runs | 101 | failure5 | failure5 | equal |
| check 18 unit 32 killed | 10 | generic | generic (reason 1) | equal |

The same on rom2-ru. Frames, digests and the report are in the lane's notes,
not in the repository.

Loss control: with the hero's edition reason set back to zero, the hero
case of `TestMissionLossReasonPerEvent` and the hero case of
`TestSecondGuardedLossOfIdentityTwentyTwoShowsFailureThree` fail.

## Open debt

- DIV-4223: what a reason outside the mission's entries shows.
- DIV-4224: the classes identities 22 and 23 carry, and hero byte +0x13c.
- DIV-4225: the order of the script tick and the result tick, and check 18's
  death state.
- The loss notice draws its body on one line, so a failure text longer than
  the window shows only its start (base and head alike, seen on the
  mission 41 and 101 frames). The notice builder is unchanged here.
