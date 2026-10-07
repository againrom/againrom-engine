# 0091 — verification

Run from a lane worktree at `wt-0091`, Go 1.26.1, Windows 11. The two lawful
roots are named only as `AGAINROM_ASSETS`; no path is compiled in.

## The gates

```
go build ./...                                   EXIT=0
go vet ./...                                     EXIT=0
gofmt -l $(git ls-files '*.go')                  no output
go test -count=1 -trimpath ./...                 all packages ok
bash scripts/check-no-game-assets.sh             EXIT=0   clean (tree scan)
bash scripts/check-no-game-assets.sh --history   EXIT=0   clean (history scan)
bash scripts/check-doc-budget.sh                 EXIT=0
bash scripts/check-sdd-audit.sh                  EXIT=0
```

Every one of those was re-run in the foreground on the MERGED tree, after master
was merged in, and each is judged by its own exit code. The audit reports 119
notes and zero FAIL; the count is meaningless from a lane once its own `builds/`
directory exists and only the FAIL set is comparable.

`-trimpath` on the test run because Windows Defender quarantines one test binary
without it.

The submodule was bumped to research master at the story boundary and frozen
there for the rest of the story:

```
$ git submodule status research
 12a4092ea47c38ddeefce1681564ff03894c6c0e research (heads/master)
```

## Acceptance criteria

| Criterion | Evidence |
|---|---|
| AC-1 | `TestAUnitMarchesAtItsOwnRange` — three ranges including 0, and the entity built naming none |
| AC-2 | same, and `TestAWiderRangeContainsANarrowerOne` |
| AC-3 | `TestAGroupsStampIsOneMarchPerMemberAtItsOwnRange` |
| AC-4 | `TestTheNoticeRadiusFollowsThePairAndNotEitherTerm`, on three members whose furthest, widest and largest-pair are three different members |
| AC-5, AC-6 | `TestAPlacementsRangeIsItsOwnBandsColumn`, four arms |
| AC-7 | `TestSightIsDerivedFromMindAndReactionTogether` (`pkg/data`), `TestAPartyMembersRangeIsHisOwnDerivation` (`pkg/mapload`) |
| AC-8 | `TestAColumnOutsideAByteIsTruncated`, four columns including 300 and −2 |
| AC-9, AC-10, AC-11 | `TestTheRangeIsCanonicalState`, which round trips all 256 byte values |
| AC-12 | `TestMarshalledBytesArePinned`, `TestHashIsPinned`, `TestThePinIsThePreStoryPinPlusTheSightRange` |
| AC-13 | `TestTheStartColumnsAreTheDecodedViewportsOwnSpan` |
| AC-14 | `TestStartViewSpansTheAuthoredColumnsAndCentresOnTheCell` |
| P-1 | `TestAWiderRangeContainsANarrowerOne`, ranges 0…12 |
| P-2 | `TestAGroupsStampIsOneMarchPerMemberAtItsOwnRange` |
| P-3 | `TestEveryPlacedUnitHasARangeFromOneSource`, with and without a table |
| P-4, P-5 | `TestTheRangeIsCanonicalState`; `TestThePredicatesOwnTablesAndStampAreNotState` |
| SC-1 | every row above; the mutation battery below is the discrimination measurement |
| SC-2 | `go test ./...` green with no install present, above |
| SC-3 | the battery below, survivors first |
| SC-4 | the milestone section below |
| SC-5 | the census below, through the shipped tool on both roots |

## The merge, and the record it produced

`0089` landed on master while this branch was open, taking **version 17** and
appending SEVEN bytes to each entity record — the decay stage, the dwell owed and
the dying time — at `+92`…`+98`. This story appends ONE at `+99`. Master was
merged into this branch (not rebased), and the merged record is **100 bytes at
version 18**. Neither lane's offsets moved: both put their field at the tail, so
17 owns `+92`…`+98` and 18 owns `+99`, and no documented offset was restated.

Sixteen files collided and every transcription in them was re-derived rather than
hand-patched. The merge's own sharpest check is the peel below.

## The pinned forms, derived rather than recorded

Both pinned transcriptions gained the byte by hand on the MERGED form. The two
digests were computed from those hand-written byte tables by an implementation of
FNV-1a outside this tree, checked against the published vectors first — and they
agree with the merged encoder to the bit. The same program then strips the byte
back off every record, puts the version byte back to 17, and reaches **`0089`'s
own pinned digests**, which were fixed before this branch merged and which no
code here can move:

```
fnv vectors ok
pinBytes  len=4327  want 4327
rtfBytes  len=4331  want 4331
pinDigest = 0x7a94e926fd68cc91
rtfDigest = 0xdb02db658ef7abe7
pin peeled = 0xb942b5d55f697fac  (want 0089's 0xb942b5d55f697fac)
rtf peeled = 0x5e8ef7f50db81ea7  (want 0089's 0x5e8ef7f50db81ea7)
```

That is the strongest evidence in this document. Reaching a literal the OTHER
lane pinned proves two things at once: every byte `0089` wrote is exactly where
`0089` put it, and this story added one byte per record and nothing else.

The two relaxation digests were reassembled from the form's documented layout by
a second implementation of the encoder, also outside this tree, and the same
assembly reproduces `0089`'s values when the sight byte is stripped:

```
rlx len 4150 want 4150      rlxTick1Digest = 0x18eabdf2701d7017
hyb len 4307 want 4307      hybTick1Digest = 0x40db93b44ac9c083
rlx pre-sight = 0xb87c5641df8f2724  (want 0089's 0xb87c5641df8f2724)
hyb pre-sight = 0xede68478ea249156  (want 0089's 0xede68478ea249156)
```

The loader's two digests (`pkg/mapload`) were moved by the same backwards-only
method that file already uses: every peel below the new one reaches a literal
that predates this story, and only the head of the chain is new.

**No test states the form version as a literal any more.** Three did — `0090`'s
AC-11, this story's own canonical-state test, and the budget story's
version-is-unmoved test — and all three were edited by `0089` or by this story or
by both, which is what put them among the sixteen collisions. A literal version
number cannot witness "my story added a field": it holds when a story adds one and
forgets to bump, and it fails when another lane legitimately takes the next
number. What witnesses it is the record WIDTH, the round trip, the digest, and
the peel above.

## The corpus, through the shipped tool

`classdump -databin` now prints the sight range per placement, read off the built
world. Mission 10's own map, extracted outside the repository, at difficulty 2:

```
  index           key     band  charge  relax  toHit  defence  absorb  dmgBase  dmgSpread  always  sight  entry
      0 0x000a/0x0000   person       7      4      3        6       0        2          1   false      5  "M10_Brigands"
      2 0x0001/0x0033        -       8      4      0        0       0        0          0   false      5  none - the npc arm reached no entry
      3 0x0045/0x0001 creature       8      4     40       50       2        1          7   false      6  "Ghost"
```

The whole report, per band and value, over 35 placements — and **the two roots
produce byte-identical output**, re-checked at the landing by running both roots
against the same input path and comparing the bytes. A digest of this report is
**not** the witness and the one this section first carried is withdrawn: the tool
echoes the path it was given, so the MD5 moves with where the map was extracted
to and reproduces on no other machine. What reproduces is the equality:

```
      2 -        sight=5
     14 creature sight=6
      5 creature sight=8
      4 person   sight=5
     10 person   sight=6
```

So **29** of the 35 placements carry a range the constant would have got wrong,
and **33** take their range from a row rather than from the constructor. Six sit
at 5: four persons whose own row says 5, and the two the npc arm resolved to no
entry, which reach it by the constructor. (This paragraph said *33 … the constant
was wrong about* and *the two that carry 5* until the landing re-ran the report;
the census above always said six.)

The whole-table census behind the spec's 4-to-12 figure was taken with a
throwaway program over `pkg/data`'s own decoders, outside the tree, on both
roots, which agreed exactly. It is our decoder's reading and not a shipped
tool's, and is recorded as that:

```
Units:  4:2  5:7  6:13  7:18  8:9  9:2  10:2  11:2  12:1     (56 rows)
Humans: 4:8  5:20  6:150  7:32                                (210 rows)
```

## One incident, recorded because a gate caught it

The mutation harness used a single shared backup file and restored the wrong
one: `pkg/game/world.go`'s entire contents were written into `pkg/sim/sight.go`,
and the merge commit was made before it was noticed. `go build` caught it on the
next gate run — **and only because the two files declare different packages,
which is luck and not a check**. Had the harness clobbered a file within the same
package the tree would have compiled.

Three things were then done rather than one. `pkg/sim/sight.go` was restored
bit-exact from this branch's tip, master having never touched that file, so there
is no judgement in the restoration. A whole-tree audit compared every tracked
file against BOTH merge parents and required every file differing from both to be
one deliberately resolved here — the list came back as exactly the twenty-one
resolutions and nothing else. And `git diff --diff-filter=D` was run against each
parent separately: the merge deletes nothing from either.

The harness now restores with `git checkout -- <file>`, which cannot put one
path's content into another. The general shape is worth more than the incident: a
tool that writes files during verification is inside the blast radius of what it
verifies, and a shared scratch path is how it gets there.

## The mutation battery — survivors first

Seventeen mutations were run before the merge and **one survived the first pass**.
It is recorded before the score because the score is worth less:

- **M17 — the notice radius narrows AFTER the maximum instead of per member.**
  Survived. The law's group record holds the maximum distance, the maximum sight
  and their combined maximum in three consecutive bytes, so every term is
  narrowed on the way in; narrowing the maximum instead is indistinguishable on
  every world this tree can load, because a map is at most 136 cells on a side
  and no sum reaches 256 there. The two part company past that, and then they
  disagree about **which member wins** rather than by a multiple of 256. Killed
  by `TestTheNoticeRadiusNarrowsPerMemberAndNotAfterTheMaximum`, on a world built
  by hand at bounds no loader produces — which is the only way the difference is
  observable at all.

The other sixteen were killed on the first pass. **The whole battery was then
re-run on the MERGED tree**, where sixteen files had moved under it, plus two
mutations only the merge makes possible — and the M17 fixture was re-checked to
confirm it still separates the two readings (204 against 48, unchanged). All
twenty died:

| # | Mutation | Killed by |
|---|---|---|
| M1 | the march is seeded with a constant 5 | `TestAUnitMarchesAtItsOwnRange` |
| M2 | every member marches at the first member's range | `pkg/sim` |
| M3 | radius = max(distance) + max(range) | `pkg/sim` |
| M4 | the radius uses a constant 5 | `pkg/sim` |
| M5 | the loader clamps an oversized column | `TestAColumnOutsideAByteIsTruncated` |
| M6 | the person arm does not fill the range | `pkg/mapload` |
| M7 | the unresolved arm does not fill the range | `pkg/mapload` |
| M8 | the hero derive divides by 20 | `pkg/data` |
| M9 | the hero derive adds no base | `pkg/data` |
| M10 | the hero derive skips the caps | `pkg/data` |
| M11 | the encoder writes a zero range | `pkg/sim` |
| M12 | the decoder reads the facing byte instead | `pkg/sim` |
| M13 | the opening view spans 20 columns again | `pkg/ui` |
| M14 | the creature arm takes the constructor's range | `pkg/mapload` |
| M15 | a party member takes 5 instead of his derivation | `pkg/mapload` |
| M16 | the hero derive reads Mind alone | `pkg/data` |
| M18 | the decoder refuses a range of zero | `pkg/sim` |
| M19 | *(merge)* the encoder writes the range over the decay stage at +92 | `pkg/sim` |
| M20 | *(merge)* the decoder reads the decay stage as the range | `pkg/sim` |

## The milestone — a negative result, stated as one

`TestTheTenthMissionIsDrivenToAWin`, with an asset root, on both roots:

```
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (43,46), Chebyshev 25, after 272 ticks
outcome lost at tick 272
```

**The mission still loses, at the same tick it lost at before this story.**
Nothing was tuned to move it and the test was not edited to expect the loss.

One thing did move: the ordered unit ends at **(43,46)** where `0090` recorded
**(44,46)** — one cell short of where it stopped before. So the per-entity range
is live on the drive's own path and changes the simulation; it is simply not what
was losing the mission.

**The merge does not move it either.** These are the lines after master was merged
in, and they are the lines this branch produced before the merge, byte for byte —
so `0089`'s decay ladder does not touch this drive any more than the sight range
does. Both roots agree, and the shipped report over mission 10 is still
byte-identical across them after the rebuild — re-checked at the landing, and by
comparison rather than by the withdrawn digest above.

A probe says what the drive's output cannot, and it removes a hypothesis rather
than merely failing to confirm one. The same mission was stepped with **no
commands at all** — nobody ordered anywhere — at the shipped ranges and at five
forced ones:

```
shipped    outcome=undecided tick=4000  party alive=1  fallen=0  ranges={5:6, 6:25, 8:5}
forced 1   outcome=undecided tick=4000  party alive=1  fallen=0
forced 3   outcome=undecided tick=4000  party alive=1  fallen=0
forced 5   outcome=undecided tick=4000  party alive=1  fallen=0
forced 12  outcome=LOST      tick=272   party alive=1  fallen=5
forced 40  outcome=LOST      tick=288   party alive=1  fallen=9
```

Three things follow, and none of them is visible in the drive's own two lines.

**The sight range IS on the path to this loss.** Forcing it moves the outcome
from undecided to lost and moves the tick; the shipped values sit below the
threshold when nothing is ordered and above it once the drive issues its walk.

**The party is not what dies.** It survives every one of these runs, including
both losses. The working reading — that the party is killed en route and the
mission's lose condition is its death — is wrong. What differs between the
undecided runs and the lost ones is that units *the map placed* fall: five by
tick 272, nine by 288, with the party untouched.

**So the loss is driven by acquisition among the map's own units**, which is the
same conclusion `0090` reached from the other side when it measured that emptying
the group stamp makes the drive pass. This story sharpens it: it is not that the
groups see too much or too little, it is that what they do on seeing each other
kills units the mission counts. The relation the map authors and the notice clip
are still on that decision, and the outcome counters are now the thing to read.
