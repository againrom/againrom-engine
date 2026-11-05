# One Group AI word setter

## Intent

Make the Group record's AI words (`sim.SavedGroup.AI`, the block a SAV
carries as `G3C`) one kind with one builder. Every order a writer set
becomes a row of one setter. The guard geometry and the notice base are
computed once. No player change: in both games every group moves, guards,
roams, notices and engages as before, and every tick's hashed state and
every SAV keeps its bytes.

## Authority

Owner direction for audit 20: one builder per kind, the first game as
profile data (`gameProfile`), architecture work counts 0 toward the
release, no player change. The kind, census and guard are
`review/audit-20/PLAN.md` rank R19 and the story brief. The rows keep the
claims their writers cited: AI-450 (the stance setter), AI-452 (the named
mark), AI-739, AI-741, AI-RADFREEZE-075, AI-GUARD-021 and AI-RANGE-102
(live Guard's geometry and latch), AI-ROAM-025 (the roam roll),
MOVE-GATE-035 (the rate term), AI-ACTIVITY-324 (the gate). No claim,
divergence row or knowledge pin changes.

## As-built behaviour

`pkg/sim/groupwords.go` holds `setGroupWords(w, g, members, row, values)`,
the one writer of the block. A row (`groupOrder`) names the words an order
sets and the rule of each derived word:

| row | words | rules |
|---|---|---|
| `rowCommandIssue` / `rowCommandGuard` (`commandIssueRow`) | +0x20, +0x45, +0x0a, +0x38 (+0x2d for Guard) | notice about the sub-cell mean; radius = notice |
| `rowScriptGuard` | +0x20, +0x38, +0x2d | notice about the sub-cell mean; radius = notice |
| `rowOrder` | +0x20 | script Stand Ground, Patrol's 0, a stop, Roam's stop helper, an emptied Group's 0xff |
| `rowOrderToCell` | +0x20, +0x0a | script Swarm, Move, Swarm 2; Roam |
| `rowStanceStand` | +0x20 | programme map |
| `rowStanceGuard` | +0x20, geometry, +0x00, +0x2d, +0x38, +0x39 | programme map; `fineSubCells`; radius = spread raised to the floor; notice = radius |
| `rowNativeRecord` / `rowNativeGuard` (`nativeRecordRow`) | +0x20, +0x38, +0x15, +0x0a (+0x2d for Guard) | programme map; notice given; radius = notice |
| `rowNamed` | +0x48 | |
| `rowLiveGeometry` | geometry | `wholeCells` |
| `rowLiveLatch` | +0x2d, +0x30, +0x34 | `radiusLatch` (one draw on the roll) |
| `rowRoamStep` / `rowRoamCount` | +0x0a, +0x15 / +0x15 | one byte per axis / count |
| `rowCommanded` | +0x0a | a structure use's stop |
| `rowSpeed` | +0x44 | formation rate term |
| `rowGate` | +0x45 | constructed Group, both games' activity rebuilds, a current SAV root Group |

The game dialect is the profile's group route: a row with `programme` takes
the order byte through `gameProfile.groupRoute.programme`, which holds the
second game's guard at 18. At the base no other written word or radius rule
differs by game; the rows that wrote a raw order byte (commands, script
orders) still do.

The guard geometry is one block of the setter: `groupCentroid` under the
row's centroid rule, the rule's centre cell (`centroidRule.centre`), and
`groupSpread`. The stance setter's fine mean, an inline loop before, is
`groupCentroid`'s new `fineSubCells` rule. The notice base about the
sub-cell mean is `centroidNotice`, used by the command issue, the script
Guard and the native record's freeze (`freezeGroups`). `storedCircle` is
unchanged and reads the words the setter wrote. `group.go:244`'s centroid
is the formation distribution, not a Group word, and stays.

`pkg/game/currentgrouproots.go` takes its root Group from
`sim.RootGroupRecord`, which sets the gate through `rowGate`. The SAV
seam's whole-block copies are unchanged.

`internal/archtest/groupwords.go` (`GroupWordWrites`) refuses, in every
production file of `pkg` and `cmd` but `pkg/sim/groupwords.go`, any store
through an `.AI` selector: an index store, a compound assignment or
increment, a `Put*` call, a `copy` or `clear` into a slice of the block,
the address of the block or of a word, a slice of it handed on, an `AI:`
literal field and a whole-block store. A copy source, a binary `Uint*`
read and an append source are reads. A whole-block shape is allowed only
in the named seam files: `pkg/sim/savedgroupsbinary.go`,
`pkg/formats/sav/actorgraph.go`, `pkg/game/originalgroups.go`,
`pkg/game/savgroupfields.go`, `pkg/game/citygroupcarry.go`.

## Proof

- `TestGroupWordsRecord` (`pkg/sim/groupwords_test.go`), committed alone
  before the setter: 29 synthetic cases, each in a first-game and a
  second-game World, 58 rows. Each records, before the first step and after
  every step, the World hash, the random state and every changed Group AI
  block, digested into `pkg/sim/testdata/groupwords.txt`. The cases cover
  every writer line the census names, by coverage profile. After the
  change the digests are equal and the 58 whole transcripts
  (`-group-words-dump`) are byte-equal to the base's.
- A planted change, the latch roll one cell larger, fails 27 of the 58 rows.
- The guard's self-test plants each shape and names it at its position; a
  live test runs it over the tree.
- Focused tests while editing: `./pkg/sim -run
  'Group|Guard|Roam|Swarm|Carrier|Notice|Engage|Patrol|Activity|StructureUse|Formation'`,
  `./pkg/game -run 'Group|CurrentGroup|OriginalGroup|SavGroup'`, the
  archtest guard and `TestLiveTreeClean`, and storyguard's
  `TestLiveTreeClean`, assertions unchanged.

## Open debt

- `pkg/game`'s `TestTriggerWizardOrdersAGroupToAPointAsTheShippedOrder`
  fails at the base too (map editor trigger wizard, "New trigger is not
  shown"); not this kind.
- The release tests that run Groups, guards and patrols over ticks or save
  and load Group records must run on both games at the next release.
