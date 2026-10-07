# 0121 — mission flow: provenance

## Claims this story builds on

| Claim | Confidence | What it settles here |
|---|---|---|
| `MISSION-END-013` | High | The outcome reporter's own order. Below the re-placement arm the tests are, in order: **no hero or `hero+0x13c != 0`** → `player+0x3c = 2`, announcement `0xb4`; else `session+0xb3b4 == 1` (lose); else `session+0xb3ac == 1` (win). The no-hero arm is **first**, ahead of both counters, and it is a loss. Read here through `tools/claim` at the pin; the ordering clause was verified against the claim text rather than taken from the census. |
| `TRIG-END-009` | High for the counters, the reporter's tests and the two announcement ids (active, amended) | The same arm from the other side: *"the earlier arm of the same routine sets `player+0x3c = 2` with announcement `0xb4` when the player has no living hero, which is the other way to lose."* Two independent statements of one rule. Its *"told once"* clause is **refuted** and its *"no other routine"* clause **corrected**; neither clause is used here. |
| `PARTY-GATE-013` | High | The entry-side twin, in the engine's own words: *"Client %s tries to enter mission without Hero. Rejected."* A player with no hero does not get into a mission at all. This is why a mission that never had a hero is not the case the reporter's first test is about. |
| `MISSION-WIN-003` | High | Instant 5 (`lose`) is authored on **14 of 28** campaign maps and check 18 on 8 of 28, while instant 4 (`win`) is on 28 of 28. More than half the campaign can only be lost by a rule the engine owns — which is why the missing arm was never noticed. |
| `MISSION-ROOT-017` | Medium (amended) | The campaign is **28** maps in `scenario.res`. `pkg/game/maplist.go:102` says the same of a stock install, independently. The pipeline ledger's 29 is corrected in this story's `analysis.md`. |
| `MISSION-START-001` | — (already cited by 0088, `pkg/game/frontend.go:751`) | The hero is placed at the drop cell exactly, radius 0, and everyone else is crowded around him — which is why `Start.IDs[0]` is the hero and not merely the first party member. |

Nothing new was decoded for this story and no research item was opened.

## Ours by choice

- **The mission rows themselves.** That the map list carries one row per campaign mission, that
  those rows sit above the map rows in mission-number order, and the words on them. The original
  has no such screen; this list is the project's own placeholder front end.
- **The picker's title.** `SELECT A MISSION OR MAP` is ours, as `SELECT A MAP` was.
- **Where the no-hero test lives.** `MISSION-END-013` puts it in the engine's per-player reporter,
  which is `pkg/sim`'s `scriptReport`. It is implemented one tier up, in the front-end driver's
  `settleNotices`, in the reporter's own order. The reason is stated in `analysis.md` and its
  cost is in *Open* below.
- **A downed hero is not a living hero.** The claim's field test is `hero+0x13c != 0` and nothing
  published says what that field is. This tree has three states where the original has two, and
  `pkg/sim`'s own `scriptDead` already resolves that split for the VIP rule the same way — a unit
  that has stopped participating counts as lost. `Entity.Alive()` is that rule.
- **A mission started with no hero is never lost by this rule.** The original refuses such a
  mission at the door (`PARTY-GATE-013`) and so has no state to test; this tree lets one start, so
  the rule is guarded rather than made to fire instantly.
- Both outcome banner strings and the map-list sentence remain 0066's authored ones — the
  original's panels come from string-table entries 140 and 141, which this project does not read.

## Open

- **`sim.World.Outcome()` does not carry the hero loss.** A headless `-check -mission N` run,
  `almtool`, and anything reading the world's byte form still see `OutcomeUndecided` when the hero
  is dead. The rule is the front end's until a story that owns a byte-form version can move it
  into `scriptReport` beside the two counters, where the claim puts it.
- **The outcome still latches permanently** (`pkg/sim/script.go:1043`). `MISSION-END-013` shows the
  two guards testing different values of one field, so the original re-reports win→lose and, past
  a lose count of 1, lose→win. Unchanged here and still disclosed at that line.
- **`hero+0x13c` is undecoded.** The claim's own confidence cell says the routine does not
  discriminate what that field means, and this story does not need it: no hero at all is the arm
  being built.
- **The team-mates half of the owner's testimony is not built.** `MissionParty` returns a party of
  exactly one, so there is no team-mate to lose. When a roster exists, the question of whether a
  dead team-mate also ends the mission is open — nothing published states it, and the rule built
  here is the hero's alone.
- **The original refuses a mission entry with no hero; this tree still starts one.** Not changed
  here; it belongs to the story that owns character generation.
