# The second game's guard machine

## Intent

Owner, game 0.146.18: a troll walking home after its aggression reset re-aggroes at once in
Againrom, while in the original it walks almost all the way home first. Read the whole guard
machine (EXP-2118, knowledge k417) and run it as the original does, one builder for both games.

## Authority

Knowledge k417 is pinned by `4e0088d90`, before the tests and the change.

| Claim | Use |
|---|---|
| R2-ENGINE-841 (High / Medium) | Both guards first write the members' truncated mean cell to +0x24/+0x28: `guardGeometry`. |
| R2-ENGINE-842 (High / Medium) | The leash drops each listed enemy farther than +0x2d from +0x28; no enemy left ends the pursuit: `storedCircle`, `clipCircle`. |
| R2-ENGINE-843 (High / Medium) | +0x2d = +0x38 + 4 + U{-1,0,1} on the first-member tick with +0x30 = 0, then the counter: `guardLatch`. |
| R2-ENGINE-844 (High / Medium) | Guard B's six member cases; cases 3 and 6 run, 1, 2, 4 and 5 do not (DIV-4494). |
| R2-ENGINE-845, R2-ENGINE-846, R2-SESSION-343 | The blow's record, the stamp and the clocks the cases 1, 2 and 4 need; not held by the World (DIV-4494). |
| R2-ENGINE-847 (Medium) | No stored walk-back state: a walking creature retakes a target listed within +0x2d of the centroid. |
| R2-ENGINE-848 | Mission 10's 29 creatures are all in Groups; Troll.4 is group 10. |
| R2-ENGINE-507 (amended), R2-ENGINE-503, R2-ENGINE-505, R2-ENGINE-808, R2-ENGINE-828 | Dispatch, assignment, planner and the EN arm order. |
| AI-739, AI-741, AI-GRPGUARD-074, AI-RADFREEZE-075, AI-RETAL-056, AI-REISSUE-077 | The first game's same guard tick, already built. |

Owner direction DIV-1523: the whole group still joins the fight but drops the chase when the
attacker has left the circle, as in the original; it binds both games.

## As-built behaviour

- A second-game World that holds a Group registry (a LOAD of an original save) runs programme
  byte 18 through `savedDecision` with `guardGeometry`, `storedCircle` and `guardLatch`, the
  first game's guard tick. The pursuit ends on the group tick no listed enemy lies within +0x2d
  of +0x28; the member then walks to its post on each tick (`walkHome`) and holds cmd +8 = 0xb
  there.
- An engaged foe counts only inside the clipped list in both games (`withEngagedFoes`,
  `joinEngagedGroup`). The second game's profile rules `guardNeedsMembers`,
  `engagedFoesUnclipped` and `heldCircle` are removed.
- A second-game mission start and direct start still hold no registry (`nativeGroups`) and run
  the native `decide` with the held circle: sub-cell centre, radius +0x38 + 4, no jitter, latch
  or counter, now with the clipped engaged foes (DIV-4285).
- Not built: the registry at the second game's mission start. `nativeGroups` dates from story
  1395 ("a second-game World keeps its present native paths until second-game evidence describes
  its carriers"); story 1567 made it a profile flag. Constructing the registry there switches 69
  `groupDialectNative()` sites, among them the script's Group checks, hand-overs, Group orders
  and stops, the Group Players sub-carrier (auto-healing, dropped gold), the actor layer's gate,
  the player's Group commands and the state byte form; with the flag cleared, 12 second-game
  tests of `pkg/sim` fail (animate's raised Group, script report parity, activity gates, the
  state form). The brief stops there.

## Proof

`pkg/sim/secondguardleash_test.go`, committed failing in `ca7ece5ae` and `289a30af4`:
the leash ends a pursuit and the member walks home to 0xb; the radius rolls once and survives a
cold load with the next tick's decision; the centroid follows two members; a striker outside
the leash is not joined after a LOAD nor at a mission start. The first game's Group tests pass
with their assertions unchanged.

## Open debt

- DIV-4285: the second game's mission start runs the native decision, not the guard tick.
- DIV-4494: guard B's cases 1, 2, 4 and 5; cmd +0x94..+0x9c and the two server clocks.
- The SAV write and read of the second-game command's +0x54 and +0x58, the five-original
  field counts, the Troll.4 release test and the headless probe were not done.
