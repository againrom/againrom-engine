# 1060 — campaign Victory reachability

## Intent

Prove that every shipped mission in both campaign branches can reach Victory
through this engine's compiled script. The drive may shorten ordinary player
work, but it may not write an outcome, counter, register or trigger latch.

## As built

Scenario vocabulary version 6 adds five mission-stage actions: `kill`,
`teleport`, `pick_item`, `heal`, and the faction-sized `kill_player` needed by
mission 151. Every action mutates ordinary simulation state through the
headless seam and advances one traced production tick. `kill` completes the
fall dwell through the same terminal loot and decay transition as ordinary
death, including a canonical already-fallen body, and becomes idempotent only
after the `DecayBones` loot marker. `teleport` is exact in-bounds test
relocation rather than the gameplay spell; it leaves the actor idle by ending
both pending book casts and cast recovery through the death teardown's shared
action-clear seam.

There are 28 install-backed routes, one for each mission 10 through 151. Each
route states only the ordered prerequisite actions, waits for `outcome: won`,
then asserts both `won` and zero reached unsupported script arms. Mission 10,
for example, kills the two clubmen, brings the hero to the Witch, brings both
to the village, and brings the hero to Sarindar; the mission script supplies
Victory.

## Proof

- Focused mechanism tests in `pkg/sim` and `pkg/game` cover action validation,
  terminal death/revival/placement, once-only loot from an already-fallen
  body, player ownership, pending-Heal cancellation, remote sack transfer,
  tick advancement, and a synthetic script-raised Victory.
- Preserved EN install: `pipeline/check-scenarios.sh ... 1060-campaign` — 28 of
  28 routes passed.
- Preserved RU install: the same gate — 28 of 28 routes passed.
- Final reconciled base `4994e2a58a58e1be8858209b6e77167b30314bd0`
  (research pin forward to `e60b8a125f69da1838cf86f76e7ea1b459fe9d18`):
  `go test -trimpath -count=1 ./...` passed; `check-no-game-assets.sh` reported
  `clean (tree scan)`.
- One `check-release-tests.sh` invocation selected 83 install-gated tests and
  passed 83 of 83 on EN and 83 of 83 on RU, with none lacking a subject.
- The final unfiltered scenario gate passed 43 of 43 on EN and 43 of 43 on RU:
  all 28 new campaign routes plus all 15 pre-existing scenarios.
- The required missionrun census found zero `UNSUPPORTED` nodes in mission 10
  and zero in mission 20, unchanged from the baseline. This story changes
  reachability evidence, not the compiled-script population.

## Open debt

This proves that shipped trigger chains are reachable after deterministic
semantic actions. It does not prove that every path is traversable on foot,
that every fight is balanced, or that every optional branch and message fires.
No install bytes or derived asset payloads are stored in the repository.
