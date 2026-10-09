# ROM2 campaign support census

## Intent

The ROM2 single-player plan's full-campaign row needs, first, the list of
what each of the 46 campaign maps needs that the engine cannot yet do. This
story adds that census as an instrument and pins it by tests. It changes no
gameplay and no ROM1 behaviour.

Base: `043ee514` (game 0.94.0). Knowledge pin: k204.

## Authority

The census reports what this build supports. It asserts no ROM2 behaviour.
The published claims it reads are:

| Column | Claims |
|---|---|
| departure cases, adds, movie output | R2-ENGINE-145, R2-ENGINE-146, R2-ENGINE-148 |
| first entry producer | R2-ENGINE-073 |
| claimed opcodes (the rest run on ROM1 handlers, DIV-2355) | R2-ENGINE-043, R2-ENGINE-045..R2-ENGINE-058, R2-ENGINE-060 |
| special event IDs | R2-ENGINE-050 |
| spell table | R2-ENGINE-019 |

Unknown stays Unknown: a map with no published producer has producer
`Unknown`.

## As built

`game.SecondGameCensus` (`pkg/game/secondcensus.go`) walks every campaign map
of a ROM2 root. Per map it:

- withdraws border placements as `StartMission` does, resolves every
  placement, structure and loot item against the definition table, and counts
  placements whose record carries a current health (the engine-derived
  `+0x24` reading, DIV-2357) and those carrying zero or less;
- compiles the script twice. The census-party compile uses the one-hero
  party ordinary New Game builds. The saturated compile binds every hero-band
  value 10001..11000. Omissions in the first are attributed per node and per
  trigger to one cause: `fixture` (a hero-band subject the census party
  lacks), `dynamic` (8000..9999), `withdrawn`, `absent` or `structure`. A
  trigger with several causes takes the greatest, so a party artifact never
  hides a real gap. Opcode gaps, inert triggers, scenario bank slots, item
  and spell references come from the saturated compile, so an unsupported
  opcode behind a party omission is still found;
- checks every ordinary event and failure reason against the mission text;
- starts the map with the census party and steps it 600 ticks headless,
  recording any panic, the outcome, its tick and failure reason;
- walks the engine's own campaign controller (`secondEngineReach`,
  `completeBank`) to say whether the map can become available, whether its
  victory is continued, what the continuation adds, and whether the engine
  plays the departure movie and writes a mission SAV.

`Blockers` names the gap classes per map: `start`, `run`, `headless-loss`,
`opcode`, `reference`, `party`, `definition`, `spell`, `entry`,
`continuation`, `movie`, `save`. Missing event or failure text is reported but
blocks nothing, because the mission screen shows the generic failure notice
(DIV-2356).

One command prints the census:

    go run ./cmd/campaigncensus -assets <ROM2 root>

The EN and RU outputs of this branch are in the owner-artifact directory, not
the repository.

## Census summary

Inputs: the preserved ROM2 EN and RU roots. Instrument:
`TestReleaseSecondGameSupportCensus`, which pins every total below.

| Total | EN | RU |
|---|---|---|
| maps; started; ran 600 ticks without error | 46; 46; 46 | 46; 46; 46 |
| placements; withdrawn by the border ring | 4212; 9 | 4234; 9 |
| placements with no Units or Humans row | 0 | 0 |
| structures with no buildings row; loot or script items with no row | 0; 0 | 0; 0 |
| placements carrying a current health; carrying zero | 296; 4 | 296; 4 |
| checks; instants; triggers | 986; 1334; 715 | 986; 1334; 714 |
| opcodes this build does not run; inert triggers | 0; 0 | 0; 0 |
| nodes on ROM1 handlers with no ROM2 claim | 1146 | 1146 |
| census-party omissions: nodes/triggers/triggers losing an action | 189/192/8 on 32 maps | 189/192/8 on 32 maps |
| dynamic, withdrawn, absent and structure omissions | 0 | 0 |
| maps naming a spell with no applicable rule | 41 | 41 |
| ordinary events; events with no text; failure reasons with no text | 425; 0; 2 | 425; 0; 2 |
| maps the controller can make available; maps whose victory it continues | 3 (10, 20, 21) | 3 |
| maps with no blocker | 1 (20) | 1 (20) |

Facts behind the totals:

- The ROM2 spell table has 35 entries and entry 30 carries no parameter.
  `data.LoadSpells` refuses the whole table, so every ROM2 world has no spell
  rule. Placements and scripts name 16 spell IDs.
- Map 101 is lost at tick 32 with reason 5: its authored unit 7 carries
  health 0 under the engine-derived `+0x24` reading, starts fallen, and the
  map's alive check sees it dead at the first evaluation. Maps 30 (two) and
  50 (one) carry the other zero-health placements.
- Scripts write three departure gates: map 20 slot 772, map 50 slot 780,
  map 110 slot 779.
- Map 40 reason 7 and map 96 reason 5 have no failure section.
- The only per-map EN/RU difference beyond placement counts is map 52's
  trigger count (25 against 24).

| Mission | Placements EN/RU | Born fallen | Party omissions n/t/a | Spell IDs without rule | Departure output | Producer | Blockers |
|---|---|---|---|---|---|---|---|
| 10 | 29/29 | 0 | 0/0/0 | 10, 11, 18, 24, 27 | 1 | town 1 inn talk | spell |
| 20 | 41/41 | 0 | 0/0/0 | none | none | departure of 10 | none |
| 21 | 58/58 | 0 | 0/0/0 | 1, 5, 24, 27 | none | departure of 20 | spell |
| 30 | 124/124 | 2 | 11/10/0 | 26 | 2 | Unknown | party, spell, entry, continuation, movie, save |
| 31 | 76/76 | 0 | 0/0/0 | 1, 24, 26, 27 | none | Unknown | spell, entry, continuation, save |
| 32 | 20/20 | 0 | 0/0/0 | none | none | departure of 31 | entry, continuation, save |
| 40 | 108/108 | 0 | 10/7/3 | 1, 2, 4, 10, 20, 21, 24, 26, 27 | none | Unknown | party, spell, entry, continuation, save |
| 41 | 48/48 | 0 | 1/1/0 | 1, 10, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 42 | 89/89 | 0 | 4/6/0 | 1, 2, 5, 6, 10, 16, 20, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 43 | 55/55 | 0 | 1/2/0 | 26 | none | Unknown | party, spell, entry, continuation, save |
| 50 | 74/74 | 1 | 10/12/0 | 5, 26 | none | departure of 40 | party, spell, entry, continuation, save |
| 51 | 147/147 | 0 | 2/2/0 | 26 | none | Unknown | party, spell, entry, continuation, save |
| 52 | 70/70 | 0 | 6/6/0 | 5 | none | Unknown | party, spell, entry, continuation, save |
| 53 | 90/90 | 0 | 10/10/0 | 5, 7, 26 | none | Unknown | party, spell, entry, continuation, save |
| 60 | 96/96 | 0 | 3/2/1 | 1, 6 | none | departure of 40 | party, spell, entry, continuation, save |
| 61 | 68/68 | 0 | 0/0/0 | 1 | none | Unknown | spell, entry, continuation, save |
| 62 | 39/39 | 0 | 4/4/0 | 1, 2, 5, 10, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 63 | 102/102 | 0 | 4/7/0 | 1, 2, 10, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 70 | 140/140 | 0 | 17/6/4 | 5, 6, 10, 18, 24, 26, 27 | 3 | Unknown | party, spell, entry, continuation, movie, save |
| 71 | 63/63 | 0 | 4/6/0 | 17 | none | Unknown | party, spell, entry, continuation, save |
| 72 | 165/165 | 0 | 12/8/0 | 1, 10 | none | Unknown | party, spell, entry, continuation, save |
| 73 | 97/97 | 0 | 4/4/0 | 2, 5, 6, 10, 20, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 74 | 103/103 | 0 | 2/2/0 | 1, 10 | none | Unknown | party, spell, entry, continuation, save |
| 75 | 115/115 | 0 | 6/9/0 | 5, 6, 10, 20, 24, 26, 27 | none | Unknown | party, spell, entry, continuation, save |
| 76 | 132/132 | 0 | 6/8/0 | 1, 2, 5, 10, 24, 26, 27 | none | Unknown | party, spell, entry, continuation, save |
| 77 | 131/131 | 0 | 4/6/0 | 10, 26 | none | Unknown | party, spell, entry, continuation, save |
| 80 | 127/127 | 0 | 4/4/0 | 1, 10 | 3 | departure of 60 | party, spell, entry, continuation, movie, save |
| 81 | 59/59 | 0 | 0/0/0 | 26 | none | Unknown | spell, entry, continuation, save |
| 82 | 86/86 | 0 | 0/0/0 | none | none | Unknown | entry, continuation, save |
| 83 | 92/92 | 0 | 0/0/0 | 1, 10, 26 | none | Unknown | spell, entry, continuation, save |
| 84 | 59/59 | 0 | 2/2/0 | none | none | Unknown | party, entry, continuation, save |
| 85 | 42/42 | 0 | 2/4/0 | none | none | Unknown | party, entry, continuation, save |
| 86 | 35/35 | 0 | 4/10/0 | 5, 6, 10, 20, 24, 26, 27 | none | Unknown | party, spell, entry, continuation, save |
| 87 | 35/35 | 0 | 6/10/0 | 26 | none | Unknown | party, spell, entry, continuation, save |
| 90 | 167/167 | 0 | 4/4/0 | 5, 10, 26 | none | Unknown | party, spell, entry, continuation, save |
| 91 | 160/160 | 0 | 6/6/0 | 1, 18, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 92 | 113/113 | 0 | 6/6/0 | 5, 26 | none | Unknown | party, spell, entry, continuation, save |
| 93 | 23/23 | 0 | 0/0/0 | 26 | none | Unknown | spell, entry, continuation, save |
| 94 | 22/22 | 0 | 0/0/0 | 26 | none | Unknown | spell, entry, continuation, save |
| 95 | 22/22 | 0 | 0/0/0 | 6, 26 | none | Unknown | spell, entry, continuation, save |
| 96 | 22/22 | 0 | 0/0/0 | 18, 26 | none | Unknown | spell, entry, continuation, save |
| 100 | 174/174 | 0 | 6/6/0 | 5, 11, 18, 20, 21, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 101 | 191/191 | 1 | 4/4/0 | 5, 26 | none | Unknown | headless-loss, party, spell, entry, continuation, save |
| 102 | 145/145 | 0 | 8/10/0 | 10, 11, 18, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 103 | 99/99 | 0 | 16/8/0 | 1, 10, 24, 27 | none | Unknown | party, spell, entry, continuation, save |
| 110 | 259/281 | 0 | 0/0/0 | 2, 5, 10, 11, 18, 20, 21, 24, 26, 27 | 4 | Unknown | spell, entry, continuation, movie, save |

## Candidate slices

Ordered smallest playable result first. "Unlocks" names the maps that have no
blocker once the slice and every earlier slice land, read from the census
rows above.

1. **ROM2 spell table.** Load the 29-spell table past its empty trailing
   entries and give each ROM2 spell its arm. Unlocks 10 and 21; removes the
   spell blocker from 41 maps. Authority: R2-ENGINE-017..R2-ENGINE-032
   (published, High/Medium). Unknown: the arm of each spell new against ROM1
   where the claims do not name it.
2. **Second-chapter route without companions.** Town 2 TALK entries, ordinary
   departure for every mission ID (case bodies, the join, the common stores
   and the stage switch), movie outputs 2..4, and mission SAV on every
   ordinary mission. Unlocks 31 and 32; removes the continuation blocker from
   43 maps and the movie blocker from 4. Authority: R2-ENGINE-145,
   R2-ENGINE-146, R2-ENGINE-148, R2-SESSION-047..R2-SESSION-049 and
   R2-ENGINE-075 (published, High); the SAV contract and DIV-2410. Unknown,
   needing research: which catalog records the stage-30 EnterInn and town 2
   TALK entries add, under which bank gates (R2-ENGINE-161 leaves their types
   and IDs open).
3. **Campaign party producer.** Bind the companions and roles the scripts name
   as 10002, 10003 and later hero-band values. Unlocks 30; removes the party
   blocker from 32 maps. Authority: R2-ENGINE-076, R2-ENGINE-105..R2-ENGINE-108
   and R2-ENGINE-124 (Medium); complete membership is Unknown
   (R2-ENGINE-127, R2-ENGINE-141, R2-ENGINE-158) and needs research. DIV-2391
   holds the current bridge.
4. **Later chapter entries.** The EnterInn entries for stages 40..110, town 3
   (departure of 50) and the bank775 restored list. Unlocks 40..100, 102,
   103 and 110 (every remaining map except 101); 50, 60 and 80 also follow
   from published departures once 40 and 60 are entered. Authority:
   R2-ENGINE-146 for departure adds; Unknown, needing research, for the inn
   entries (R2-ENGINE-161) and the restored records (R2-ENGINE-149).
5. **Placement current health.** Decide what the ROM2 record's `+0x24` word
   means and how a zero is treated. Unlocks 101; changes the start of 30 and
   50. Authority: Unknown (DIV-2357 names no claim for the record layout);
   needs research. The question can go to research before 101 is reachable.
6. **Shared-arm contracts.** 1146 compiled nodes run on ROM1 handlers with no
   ROM2 claim. Unlocks no map; it is fidelity debt under DIV-2355 and needs
   research per opcode.

Owner direction can replace the research in slices 2 and 4 with a portable
route that offers each chapter's maps, recorded as a divergence; the census
then reports those maps as available.

## Proof

- `TestSecondRefCauseSeparatesPartyArtifactsFromRealGaps`,
  `TestSecondOmissionsAttributeEachTriggerToItsGreatestCause`,
  `TestSecondGapsCountTheTriggersEachOpcodeHolds`,
  `TestSecondItemRowSelectsTheClassCollection`,
  `TestSecondControllerReachesAndContinuesOnlyItsMissions`,
  `TestSecondBlockersNameEachClass` and
  `TestWriteSecondCensusPrintsRowsAndTotals` classify synthetic inputs.
- `TestReleaseSecondGameSupportCensus` runs on each ROM2 root and pins the
  totals above, the blocker-free map 20, map 101's loss and the two failure
  reasons without text. `AGAINROM_ROM2_CENSUS_OUT=<dir>` also writes the
  census there.
- `cmd/campaigncensus` `TestRunNamesTheMissingRoot`.

## Open debt

- The census does not cover presentation: unit sprites, sounds, the town and
  inn screens, and the mission screen layout.
- It does not compare a map's simulation with the original. A map with no
  blocker is one the census cannot tell from a supported map, not a map
  proven to play as the original.
- The headless run issues no player command, so a loss that needs the player
  to act is reported as `headless-loss` until a scenario proves otherwise.
- Reserved DIV-2608..DIV-2615 are unused: the census records no new
  difference.
