# Orders and order admission follow the X1 research in both games

## Result

When a defended or followed ally dies outright, the escort stands down at its own cell and engages any enemy in
reach on the same pass instead of staying bound to the corpse. A dying ally that can still be healed or finished
keeps its escort until death is final. A unit whose attack target dies outright does the same. The first game's
research reads these stand-downs; the second game's reads only the hook before them, so it follows the first.
Every other proved order behaviour matches the engine; each remaining unknown is in the ledger.

## Authority

- Owner: one order, withdrawal and AI machine per game through one builder per kind; every row built or tracked.
- `AI-652` (High): first game, an escort whose subject is at action 0x10 stands down to 0xc with `ord+0x50 = 1`.
- `AI-642` (High / Medium): first game, state 3 with a victim at action 0x10 stands down and the same call acquires.
- `R2-ENGINE-1925` (High / Medium): second game, the selector calls a hook for the same shapes and the hook stores
  `ord+0x50 = 1`; its callees are unread. The engine reads action 0x10 as `HP <= -10` (Medium).
- `R2-ENGINE-1920..1924`, `1926..1931`, `AI-842..852`, `AI-924..928`: confirm existing rows; unknowns in DIV rows.

## Rules

1. A defended or followed unit whose subject reaches -10 HP stands down to 0xc, clears escort, and acquires now.
2. A defended or followed unit whose subject is dying (0 to -9 HP) keeps its escort active.
3. A unit at engage order 3 whose victim reaches -10 HP stands down to 0xc and acquires now.
4. Terminal escort stand-down produces identical world hashes across deterministic replays and save/load.

## Code

| kind | its one builder | change |
|---|---|---|
| escort stand-down | `pkg/sim/escort.go` `standDownTerminalEscort` + `pkg/sim/targetorder.go` `stopTargetOrder` | clear terminal escort via `clearEscort` and `stopTargetOrder`, then run `savedAcquire` in the same pass |
| saved actor dispatch | `pkg/sim/savedgroups.go` `savedActorDispatch` | call `standDownTerminalEscort` before escort close/arm on states 8 and 0x11 |
| engage dispatch | `pkg/sim/savedengagement.go` `savedEngage` | stand down terminal target via `stopTargetOrder` and run `savedAcquire` in the same pass |

## Proof

| test | rules |
|---|---|
| `TestTerminalEscortSubjectStandsDownAndAcquiresInBothGames` | 1, 2 |
| `TestTerminalEngageTargetStandsDownAndAcquiresInBothGames` | 3 |
| `TestTerminalEscortStandDownReplayHashAndRoundTrip` | 1, 4 |

Release tests at the release: `TestReleaseCommandPanelComposesFromRealArtInFourStates` (EN/RU order panel witness).

## Open debt

- Remaining unknowns: DIV-558, DIV-4691, DIV-4758, DIV-4766, DIV-4799, DIV-4800, DIV-4884, DIV-4886, DIV-4895.
- Additional unknowns: DIV-4904, DIV-5160, DIV-5323, DIV-5324; explicit Retreat (`playerretreat.go`) is later.
