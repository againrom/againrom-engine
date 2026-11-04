# The second game's Group registry at a mission start

## Intent

Owner, game 0.146.18: a troll attacked while it walks home after its aggression reset walks almost
all the way home in the original before it turns again; Againrom differs. Read the whole guard
machine and run it as the original does. Owner, on story 1593's stop at the mission start: the
original's guard-bound Group registry exists in both games, so it is one mechanism built for both;
the first game is a reduced second game. Bound rulings: one builder per kind; owner-directed
behaviour need not wait for evidence, each difference a DIV row; SAV is the only save format and
every field is written from the current state.

## Authority

Knowledge k417, pinned on the base `08b8cd8a1`.

| Claim | Use |
|---|---|
| R2-ENGINE-507 (High / Medium) | The Group tick switches on programme byte +0x20; 18 runs guard B. |
| R2-ENGINE-841..843 (High / Medium) | Guard B's centroid, leash at +0x2d and radius step, built in story 1593: now from the mission's first group tick. |
| R2-ENGINE-844 (High / Medium) | Guard B's member cases 1, 2 and 4: built. Case 5 is not (DIV-4494). |
| R2-ENGINE-845 (High / Medium) | A blow writes the victim's +0x54, +0x58, +0x94, +0x98 and +0x9c. |
| R2-ENGINE-846 (High / Medium) | The stamp of cmd +0x16 with server word +0; the constructor's 65500. |
| R2-SESSION-343 (High / Medium) | The two server clocks; game0006's 2257 / 141 = 16.0; 13 Groups in each original mission-10 file. |
| R2-SESSION-339 (High) | Every original-made command record holds +0x16 65500. |
| R2-ENGINE-848 | Troll.4 is mission 10's Group 10. |
| AI-739, AI-741, AI-GRPGUARD-074, AI-RETAL-056 | The first game's guard tick; none states cases 1, 2 or 4. |

## As-built behaviour

- A second-game mission start and direct start build the Group registry with the first game's
  constructor (`constructGroups` / `constructedGroups`, `pkg/sim/carriers.go`). The profile flag
  `nativeGroups` is gone. Every Group runs `savedDecision`; the script's Group checks, hand-overs,
  Group orders and stops, the Group Players sub-carrier, the actor layer's gate, the player's Group
  commands and the activity gate take the first game's registry paths (DIV-4502). The native Group
  arm stays behind `groupDialectNative()` for the states before construction; its removal is a
  separate change.
- The second game's byte form writes the registry even when it equals the constructed one
  (`formHoldsConstructedGroups`); a form an older engine wrote without one constructs it on decode
  from its native Group records.
- A SAVE of a World without a registry writes the registry the constructor builds from the current
  state (`WithGroupRegistry`); no save is refused. The SAV writer's synthesis from
  `ConstructedGroup` is gone; one roster path writes each Group and member order (DIV-4471).
- Guard B's member cases 1, 2 and 4 and pursuit stamp run only for a second-game Group at
  programme 18 (`groupRouteProfile.runsStrikeCases`). Programme 1 (guard A), including the
  player's Guard Groups, runs only cases 3 and 6, as on the base `08b8cd8a1`. The game difference
  is `groupRouteProfile.strikeCases` data
  (`pkg/sim/guardstrike.go`, DIV-4503): `rememberAttacker` writes the blow's record on the
  victim's order (striker, fine clock, the cell 10 beyond the victim on the line from the striker).
  A member pursuing a unit beyond its reach is stamped with server word +0. A member holding a
  blow inside the stamp window (below 30 group ticks while hurt, else below 10) drops its target
  and walks to +0x9c (cases 1 and 2), and takes no foe from the join pass while the window holds
  (`strikeWindow`); a hurt member without a target within 10 fine ticks of its
  blow walks to +0x9c instead of its post (case 4). Server word +0 is World tick / 16, dword +4
  the World tick.
- SAV: every second-game save point writes the command's +0x16 (the stamp, else 65500), +0x54,
  +0x58..+0x5a and the blow record at +0x94, +0x98 and +0x9c, +0x94 as the striker's record key
  (`sav.ProfileKeeps.Commands`). A LOAD of an original's records reads them into the member's
  order (`ImportLoadedStrikes`); the engine's own LOAD keeps them through the order continuation
  (DIV-4479, DIV-4471).
- The first game runs none of the cases; its records, hashes and SAV bytes are unchanged.

## Proof

- `pkg/sim/secondmissiongroups_test.go`: a second-game mission start holds the registry (failing
  first in `096a1a457`); a SAVE of a World without one writes the constructed Groups (`cc53bcaf0`,
  a new method's test).
- `pkg/sim/secondguardstrike_test.go` (failing first in `e08ba2e5e`): the stamp window, case 4, and
  a cold load inside the window.
- `TestSecondGameStrikeWindowTakesNoJoinedFoe`: a member takes no joined foe from the first
  group decision after a blow until its window ends, with the striker standing on its blow cell
  or continuing to attack. Both reach an arrived member eligible to join its striker; the melee
  lands repeated blows inside the window. Both take the striker after expiry.
- `TestSecondGameGuardAProgrammeRunsNoStrikeCases`: programme 1 takes no pursuit stamp and
  keeps its post after a recent blow from outside sight with no target; programme 18 takes a
  stamp in pursuit and walks to +0x9c after a recent blow. The player's hero in a human Guard
  Group also stays at its post. Both new tests fail with the production files of `7f15fde0d`
  supplied by `go test -overlay`.
- `pkg/game/secondcommandstrike_test.go` (failing first in `d3cd2a777`): a records LOAD, resave and
  engine LOAD keep a planted blow record, alarm and stamp.
- A LOAD and resave of the five EN original mission-10 files keeps every Group's +0x00..+0x4b and
  the command fields of every living record; in `game0006` a dead Unit's and the hero's blow
  fields write 0 (DIV-4471).
- `TestReleaseSecondGuardStruckWalksAwayInsideTheStampWindow` (both roots, written, not run in the
  lane): Troll.4 struck once on the installed mission 10. One headless EN run of the same scenario:
  the blow written at tick 66, the stamp 6 on the group tick the troll pursued beyond reach, then
  ten group ticks walking towards +0x9c (36,6) with no victim, the hero taken again at word 16.

## Open debt

- DIV-4502: the second game's own mission-start Group constructor is unread.
- DIV-4503: the stamp's test, the clock scales and the +0x9c geometry are this build's reading.
- DIV-4494: case 5.
- DIV-4479: +0x16 under the unread stamp test.
- The native Group arm behind `groupDialectNative()` (69 production sites) remains.
