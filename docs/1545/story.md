# The Kaarg inn taverner moves and sounds

## Intent

The second game's Kaarg inn drew its taverner as one still picture and made no taverner or room
sound. This story runs the original's five taverner series, steps the selected seat's sprite and
plays the room's entry, loop, voice and dish sounds, all as data in the town description read by
the existing scene builders.

## Authority

- Owner remarks 4 and 5 (triage W6, `DIV-2974`, `DIV-2975`).
- Research EXP-2087, knowledge k381: `R2-ENGINE-277` (the five series, the choice, the 100 ms
  step, the a5 table), `R2-ENGINE-685` (entry, leave and kept state), `R2-ENGINE-686` (the seat
  sprite's 125 ms step), `R2-ENGINE-687` (the sound requests and their clocks), `R2-ENGINE-688`,
  `R2-ASSET-073`.
- Randomness ruling: one seed, the consuming arithmetic as the original, the original `rand()`
  switchable.
- Town is timeless (owner): these are presentation clocks, outside the World hash and the SAV.

## As built

### Taverner

At rest the taverner draws r = 1 + raw*21/32768 from the `tender` source (`random.Tavern`) and
maps it to a series: 1 to a1, 2 to a2, 5 to a5, 3, 4 and 6 to a3, 7 to 21 to a4. Positions
(160+x, y): a1 and a2 (232,88), a3 (272,112), a4 (360,136), a5 (248,84). Each series runs once, one
file per step, on a paint more than 100 ms after the last step; a restarted series first draws its
cached last frame. a1 requests Kman3 at its start and a2 Kman2. State, a5 step and step clock
outlive a visit; entry restarts each series from element 0 without repeating its start sound. The
a5 table walks files 0 to 28 once, 29 steps without waits: the claimed 95-step table is not in the
public snapshot (`DIV-2974`).

### Seat and room sound

Only the selected seat steps, (cursor+1) mod frames, after more than 125 ms on the scene clock
`kaarg-seat`, which outlives a visit; entry selects seat 0 and resets every cursor. Entry requests
Kin1 (no file in either install) and starts the Kvox5 loop. Kvox6..8 by R(3) and Kdish1..4 by R(4)
each run on their own clock of 2000+R(2000) ms, redrawn after each request; an unstamped clock
counts as long past, so the first paint of the first visit requests one voice and one dish
(`DIV-4271`).

### Raw picks under the original generator

A raw pick (`amount` with `raw`) reads the source's raw value: `Scene.SetRawDraw` binds each host
source of the built scene (`Scene.HostSources`) through `bindRawDraws` to `sceneRawFrom`, the
`draw(32768)` seam or `Stream.Raw()`. Seeded play draws as before, draw for draw. Under the
original generator only rand() = 32767 changes, toward the original: the first game's tavern and
the first town's inn wait 5047 ms instead of 3000, the druid taverner 5247 instead of 3200, the
Kaarg taverner chooses a4 instead of a1, the first game's school timer `rand()/10` gives 3276
instead of 0 and the shop keeper threshold `5000+1000*(rand()%5)` 7000 instead of 5000.

### Builders

| Kind | Builder |
|---|---|
| room actor program | `town.Alternating`, shared with both games' other taverners; gains `choice`, `picks` and the `table` motion |
| kept room actor progress | one store, `town.Process.kept` (`keptProgress{Index, State, Step}` under `Scene.keepKey`), set by the one flag `ActorSpec.Keep`; shop keepers and the Kaarg taverner use it |
| scene clock | `town.SceneClock`, gains `wait` and `unstamped`; `town.StepGroup` gains `pick` |
| random pick | `town.PickSpec` with the new form `fraction`; raw values through `Scene.SetRawDraw` and `rawPick` |
| seat grid | `town.SeatSpec.clock`, stepped in `advanceSeats` |
| room sound | the scene's slots, `sounds.loop` and `enter` steps |
| town description | `pkg/game/towns/rom2-town2.json` through the existing loader |

No second builder is added.

## Proof

- Synthetic: `pkg/town/choice_test.go` (raw to value bounds, cadence, cached last frame, kept
  choice, clock and group pick, ten decode refusals), `pkg/game/secondkaarginn_test.go` (series
  choice for 12 raw values, steps, voice and dish clocks, state across entry, seat clock),
  `TestTownRawPickReadsRandUnderTheOriginalGenerator` and
  `TestPageKeepHoldsAnIndexAndAStateInOneProcessStore` (both fail on `4242d5cc`, pass after),
  `TestTownRawPickSeededDrawsAsBefore` (400 000 draws per source equal in seeded play).
- Installed, read only: `TestReleaseSecondTownKaargInnTaverner` and
  `TestReleaseSecondTownKaargInnPage` PASS on rom2-en and rom2-ru (the five series loaded at full
  length, a1 then a5 in sequence, pixel probes, the sound requests).
- Loss controls: the town, inn, tavern, square, shop and wildlife sets pass before and after; the
  review's 12-scene trace (48 000 lines) is byte-equal on `4242d5cc` and the candidate.
- Review (`pipeline/reviews/story1545-adversarial-review.md` in the seat) returned two findings,
  the raw pick under the original generator and a second kept-state store; one correction pass
  answered both.

## Open debt

- `DIV-2974`: the a5 table runs 29 steps instead of the claimed 95.
- `DIV-4271`: Kin1 plays nothing; Kvox5 as a scene loop; the first stamps of the voice and dish
  clocks; the first seat static value; the draw order.
- `DIV-4272`: a hire seat steps before it is drawn, one paint earlier than the original.
- `DIV-2731`: the square's `rand()%100` and the first game's shop `rand()%5` still differ.
- Unknown: whether Kin1 is heard, whether pause stops Kvox5, the screen on re-entry mid-series,
  the original's paint rate.
