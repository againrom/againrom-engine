# 0144 — verification

Evidence for `spec.md`, measured on this branch. Commands are given as run; the asset roots are the
preserved lawful installs one level above the repos and no `.sav` byte is in the repository.

**The research pin was bumped mid-story**, once, from `20dcf51` to `e2516a3`, on the orchestrator's
instruction — see `provenance.md`. Everything below the pin line was re-run against the new pin.
`spec.md` L-1 and L-3 still stand, for a narrower and now-named reason; the rows below say where.

`master` was merged at `626ed7c` (story 0143) and every number below was re-measured after it.

## The gate

```
go build ./...                 clean
go vet ./...                   clean
gofmt -l $(git ls-files '*.go')  prints nothing
go test -trimpath -count=1 ./...  all packages ok
bash scripts/check-no-game-assets.sh   clean (tree scan)
bash scripts/check-doc-budget.sh       every artifact under, chain ok
bash scripts/check-hotfix-ledger.sh    clean
bash scripts/check-sdd-audit.sh        FAIL set empty
```

Re-run at the landing on a clean tree. Only the audit's FAIL set is comparable from a worktree.

## FR-8 and AC-1 — the round trip, which is the story's central evidence

The corpus is **twenty-three files** outside both repos: the four EN install saves, the four RU
ones, and the fifteen preserved by the owner. By content it is **18 distinct files** — the EN four
are byte-identical to the RU four, and one preserved save is byte-identical to that pair as well.

> Re-measured at the landing; written at *twenty files, 16 distinct*, which was over by one and
> three files short. Pass rate unaffected — every file passes, then and now.

`savtool verify` opens each file, re-compresses its decoded body — it does not carry the blob it
read (DD-3) — and compares the whole result to the input.

```
$ go run ./cmd/savtool verify <en>/*.sav <ru>/*.sav <preserved>/*.sav
...
23/23 byte-identical
```

**23 of 23 — AC-1 holds over the whole corpus.** Sizes 3 544 to 46 489 bytes, decoded streams 6 110 to 83 802 bytes. This exercises
FR-1 (the container, whose `blobEnd`/`blobBytes` are recomputed — DD-2), FR-2 in both directions,
and everything the parse touched, because the parse ran on every one of them before the re-emit.

It is also the first time the published encoder rule has been run backwards over a whole corpus:
it reproduces the shipped bytes exactly, with no tie-breaking freedom left unstated. Recorded as
this consumer's observation in `provenance.md`, not as a claim.

## FR-3, FR-4, FR-5 — the head and the roster, and AC-2

`savtool info` over all fifteen owner saves. Every one reads five players except the between-mission
file, which reads one; missions read 10 and 20 with difficulty 2 and player-list dword 6 (2 on the
between-mission file).

**The human participant is in slot 1 in every file, and its name is not a constant.** Twelve name
`Danath`, `game9999.sav` names `Fergard`. Written as *"every one names `Danath`"*; the wider corpus
refutes it. The position is the invariant, the name is content.

**AC-2, the labelled set.** The owner said two of his saves should carry a completed-mission flag
and one was taken in the city. Read blind, the outcome latch (FR-5) is `1` on exactly three files:

| file | label | outcome | money | world half |
|---|---|---|---|---|
| `game0006.sav` | `finished` (with a leading Cyrillic byte) | **1 COMPLETE** | 100 | yes, won 1 |
| `game0009.sav` | `666` | **1 COMPLETE** | 600 | yes, won 1 |
| `game0010.sav` | `city` | **1 COMPLETE** | 600 | **none** |
| the other nine | numeric, and `Restart last mission` | 0 in progress | 100 | yes, won 0 |

The two the owner named are the two that carry it. The third is explained rather than explained
away: it is the between-mission save, it carries the same latch and the same purse as the finished
20.alm file, and that is exactly what `SAV-FLAG-027` says the latch is for — it lives in the
campaign half so that it survives the mission's end. Money reads 100 and 600, not 1.54 billion, so
the involution (FR-4) is applied.

## AC-3, FR-6, L-4 — the save with no world half

`game0010.sav`, 3 544 bytes: opens, mission number **0** with the map-name field still holding
`20.alm`, one player with the whole roster's fields, and no block records, no cell-record table and
no session block. `savtool session` and `savtool blocks` refuse it by name. This is the only witness
in existence for that shape and it is what tests FR-3's rule that the mission number and not the map
name says whether a mission is in progress.

## FR-6, DD-4 — the world half, located and validated

Located on **22 of 23 files**, refused on the one between-mission save that has no world half
(above); written at 15 of 16, re-measured at the landing. Per file the scan's forward validation
had to hold: the cell-record table's key set **equal** to the block cells carrying static bit 5.

| file | block records | cell records | session at |
|---|---:|---:|---|
| `en/game0000` | 1 890 | 186 | `0xce53` |
| `en/game9999` | 1 842 | 185 | `0xcd25` |
| `saves/game0004` | 2 181 | 174 | `0xd0ef` |
| `saves/game0009` | 4 117 | 181 | `0x12dbc` |
| range over the 15 | 1 842 – 4 117 | 174 – 197 | — |

174 to 197 independent key equalities per file, 15 files, no failure and no second candidate. DD-5:
the scan starts past the campaign head; starting at 0 finds a false candidate in the roster.

DD-7 holds by construction: the package imports nothing (P-1), so the cell test cannot consult a
map.

## FR-7, DD-6, DD-12 — the actor heads

The head scan was **falsified before it was trusted**, and the first two designs failed:

1. A three-value table of state dwords taken from the published claim found 20 heads on
   `saves/game0004` — the buildings only — and joined **1** map unit. The Humans and Units of that
   file carry a state dword no published corpus holds.
2. Accepting a zero state as well found 81 "heads", almost all coincidences, and still joined 1.

The state dword is state and not format, and its value set is **not closed**: this corpus holds
three values (`0x0612a020`, `0x0256d020`, `0x02563020`) that no published set carries. So the reader
calibrates on the file's own dwords (DD-12) instead of tabling them. With that:

```
$ missionrun -sav <en>/game9999.sav -ticks 1
resume: mission 10 map "10.alm" label "Restart last mission" outcome 0 money 100
        heads 93 (35 dead), joined 35 to map units, MOVED 0
```

**35 joined, 0 moved on the "Restart last mission" save.** That save is the mission at its start, so
every unit stands where the map places it — and the join reproduces the map's own placements for all
35 with zero disagreement. That single number checks the head location, the packed-cell reading, the
unit-id join and the fixed-point identity at once, against a source outside the save.

On the owner's mid-play saves the same drive reports `joined 30, MOVED 10` (`saves/game0004`, 10.alm)
and `joined 53, MOVED 26` (`saves/game0009`, 20.alm) — units that had walked.

The soft edge, disclosed: a file holding fewer objects than a class run yields **no** located heads,
which is what the between-mission save does (0 heads of its 7 objects). FR-12's table is what
retires the scan.

## FR-12, DD-11, DD-13, DD-14 — the class layout is the decoder

`class.go` carries one row per class: name, head, base class, members in file order, and how the
length is determined. `Player`, `Building` and `Effect` are decoded through it and by nothing else.

**The eleven programmes became rows and the reader did not move**, which is the test DD-11 was made
to pass. What changed in the reader when the layouts landed: `Kind` gained `KindIdentity` and
`KindReference`, `Class` gained `Base`, and `decode` gained the base-class concatenation. Nothing in
`Open`, `Marshal`, the codec or the round trip changed at all, and the round trip stayed green
across the change — measured before and after.

**The head is 37 bytes, not 16, and nothing broke.** The creation-order id is still the dword at +12
and the map unit id still the low half of the dword at +19, so the actor scan's offsets did not
move — the outcome a refinement predicts and a wrong model does not.

**DD-13, the three-way length.** Which class is *fixed*, *computable* or *unread* is the table in
`class.go`, not a list to keep a second copy of here. What this stage witnesses is the consequence:
only *fixed* can be stepped over without decoding, and only a *consecutive* fixed class can be
chained — which is why `Building` is the one class FR-13 walks, and `Effect` is not.

**DD-14, degradation.** `Lookup` answers an unknown name as a class with an unread extent rather
than refusing. `TestLookupDegradesOnAnUnknownName`. This is not hypothetical: `game0010.sav`
introduces **`Spell`**, a twelfth class, and this tree reports it by name and round-trips the file
(AC-8):

```
$ savtool objects <saves>/game0010.sav
classes introduced: Player Human Weapon Armor Item Effect Diary Spell
savtool: sav: this save introduces no Building
```

**DD-12 / `SAV-OBFCEN-038`.** The obfuscation and the clamp are members of `Player`'s row alone;
`TestObfuscationIsPlayersAlone` walks every class's programme and fails if either kind appears
elsewhere. It witnesses that the machinery has not spread, **not** that no other field is
transformed — the difference, and the Medium grade it follows from, are in `provenance.md`.

## FR-13, DD-15, AC-7 — the walk, and the re-encode

`Building` is walked from its class record: 77 bytes a step, the instance tag learned from the first
step and then required (DD-15), stopping on the `0x0000` null object reference. Each record is then
**re-encoded from its decoded values** and compared to the bytes it was read from.

```
$ savtool objects -n 0 <en>/game9999.sav
identity keys: 18 distinct of 18 (0 repeated)   creation order 2..19
18 Building instance(s), 18 re-encoded identically from the decoded form
$ savtool objects -n 0 <saves>/game0009.sav
identity keys: 30 distinct of 30 (0 repeated)   creation order 2..62
30 Building instance(s), 30 re-encoded identically from the decoded form
```

Over the whole corpus, re-measured at the landing: **22 of 23 files chain** — fifteen at 18
instances, seven at 30 — and **every instance of every one re-encodes identically**, 18×15 + 30×7 =
**480 records** (written at 15 of 16 and 378, before the corpus grew). The twenty-third is the
between-mission save, which has no world half and so introduces no `Building` — itself a `20.alm`
file, which is why the count tracks the **map** and not the file: of the twenty-two that chain,
every `10.alm` save reads 18 and every `20.alm` save 30, no exception, including three the decoder
had never seen. That reproduces `SAV-BLDG-037`'s own
18/18 and 30/30, its distinct identity keys, and its creation-order runs 2..19 and 2..62 —
independently, from a decoder rather than from the disassembly.

The re-encode is the check FR-8 cannot make. To learn that it is witnessed rather than decorative,
break the layout: `TestReencodeFromTheDecodedForm` decodes a `Building` through a table whose `B46`
is a `u8` where the programme says `u16`, and asserts the re-encode comes out the wrong length. A
byte-carrying round trip cannot see that error at all.

**The `Effect` question this story raised is answered** — `SAV-EFFCHAIN-046`, reading #2; which
reading and why is `provenance.md`'s. What lands here is that the table now carries `Consecutive`
as a fact separate from having a length — `Building` has both, `Effect` has only the length — and
`Chain` refuses a class without the second up front rather than discovering it at the second
instance. `TestAFixedLengthIsNotEnoughToChain`.

## FR-9, AC-4 — read, modify, write over a real save

```
$ savtool set -out $T/mod.sav -money 12345 -outcome 2 -label 0144 \
      -actor 0:31:41 -latch 3:1  <saves/2026-08-02>/game0004.sav
wrote mod.sav: 29144 bytes
$ savtool info mod.sav
  player 0  slot  1  "Danath"  HUMAN  money 12345  outcome 2 (failed)
$ savtool actors -n 1 mod.sav      # was cell (53,26)
     0  @0x0000f8  cell ( 31, 41)  fine 0x80,0x80  id 22  unit 21  living
$ savtool session mod.sav          # was 7 latches set
  8 of 1000 fire-once trigger latches set: [0 2 3 5 6 7 8 12]
$ savtool verify mod.sav
  identical (29144 bytes)   1/1 byte-identical
```

Input and output are **the same length**, and **nine bytes of the 29 144 differ** — five edits, and
nothing else moved (DD-2). The written file re-opens, re-reads every edited field, and round-trips.
DD-10: `set` requires `-out` and refuses to write in place, and `TestSetWritesOutOfPlaceAndOnlyWhereTold`
re-reads the input afterwards to prove it.

`SetLabel` writes the text and its NUL and leaves the region's debris exactly as it was, because the
original overwrites that buffer and never clears it. `TestSetLabelKeepsTheDebrisUnderIt`.

## FR-10 — the tool

`cmd/savtool`, beside the other thirteen. Every verb is exercised against real files in the sections
above; its parsing and failure arms are tested over a synthetic save built in test code.

## FR-11, DD-1, DD-8, DD-9, AC-5 — the resumed mission

`missionrun -sav FILE` resumes instead of starting. DD-1: `ResumeMission` lives in `pkg/game`, so
`cmd/missionrun`'s allow row in `internal/archtest` **did not change** — the driver reads a file and
calls the front-end tier and never names the format tier. DD-8: the transfer writes the saved cell
and fine position into the decoded map's unit record and the ordinary start path does the rest.

AC-5 has two halves and both hold.

*It places actors the map alone does not place:*

```
$ missionrun -sav <saves>/game0004.sav -census -waypoint u21:56:21:3 -waypoint p0:66:16:3
resume: mission 10 map "10.alm" label "112" outcome 0 money 100
        heads 55 (4 dead), joined 30 to map units, MOVED 10
        blocks 2181 (474 cells the map leaves open, 28 occupied), latches 7 set
        NOT CARRIED: health, mana, statistics, inventories, timers, group orders,
        script registers and trigger latches — those come from the map (spec L-3)
mission 10  scenario/10.alm  80x80  36 entities
waypoint 1  u21 -> (56,21) r3 : reached (54,24), Chebyshev 3, after 25 ticks
...
census: 7 of 33 unit(s) moved, 0 fell, over 40089 tick(s)
```

The mission runs, the escort target is ordered and arrives, and the drive ends undecided at the
ceiling. It ends differently from a fresh start — the party hero does not survive it — and that is
the resumed *position* meeting a hero at the map's placed health, which is exactly L-3.

*And a resume from the start-of-mission save is a no-op:*

```
$ missionrun -mission 10 -census -waypoint u21:56:21:3 -waypoint p0:66:16:3   > fresh
$ missionrun -sav <en>/game9999.sav -census -waypoint ... (same)             > resumed
$ diff fresh <(tail -n +6 resumed)   # the report is the only extra
IDENTICAL DRIVE
```

Byte-identical drive output: 224 ticks, 4 of 36 units moved, 1 fell. A transfer that had invented
anything would show here.

DD-9: the block deltas and the session latches are reported and not applied. The report's
`474 cells the map leaves open` is the count of delta records naming a cell this tree's own ingest
derives as open — the encoding-independent form of that question, and the measurement
`SAV-BLOCK-012` asks for.

## AC-6 — every refusal is named, and nothing panics

Thirty-one refusal arms are covered by `go test`, each with its own message and none a panic: the
five container checks, the four codec ones, the roster count, the world-half key-set disagreement,
every out-of-range index on a player, actor, block, latch, diplomacy entry or session field, an
outcome outside 0..2, a cell outside the plane, a class with no programme, an unknown class, and the
four ways a chain can fail.

## P-1 … P-4

- **P-1** `pkg/formats/sav` is in `internal/archtest`'s allow map with an **empty** import set; the
  import-graph test enforces it. Every fixture is built in test code and no test reads an install.
- **P-2** This branch touches no file under `pkg/sim`: `git diff --stat origin/master..HEAD --
  pkg/sim` is empty after the 0143 merge, and `pkg/sim`'s tests are unchanged and green.
- **P-3** `check-no-game-assets.sh` is clean on the tree; every command above writes to a temp
  directory and every test uses `t.TempDir()`.
- **P-4** Decode is total or an error: `Open` has no arm returning a partial value and a nil error
  — the world half and the actor list are absences with their own meaning, not partial decodes, and
  both are reported as such.

## The script-gap census — unchanged, and that is the claim

`pipeline/milestone-baseline.txt` carries, for master before this branch, mission 10 unable to run
1 + 2 + 13 + 1 = **17** of its own script nodes and mission 20 1 + 11 + 1 = **13**.

```
$ go build -o /tmp/mr ./cmd/missionrun
$ for m in 10 20; do AGAINROM_ASSETS=<en> /tmp/mr -mission $m -trace -ticks 1 | grep -c UNSUPPORTED; done
17
13
```

**17 and 13, unchanged.** This story decodes a file format and adds a resume; it touches no script
opcode, so the census must not move and it does not. The result this story is pointed at is the two
runnable paths above — `savtool`, which reads and edits the owner's own saves, and
`missionrun -sav`, which resumes a real mission from one. A worktree has no `builds/`; the binaries
reach `builds/current/` when this branch lands.

## A retracted fact this tree had built on

`SAV-HUMAN-043` retracts `SAV-MEMBER-036`'s "a `Human` is a `Unit` plus 24 bytes", which this table
carried as `Human`'s row. It is now `Base: "Humanoid"`, `Extent: 0`, with `Humanoid` a row of its
own. `TestHumanIsHumanoidAndHumanoidIsWhereTheBytesAre` refuses **both halves by name** — verified
at the landing by putting each back alone: the wrong base fails at `class_test.go:174`, the value 24
at `:177`. **Why the round trip could not have caught this, and did not, is in `provenance.md`** —
it is a fact about the evidence, not a measurement.

## THE STORY IS NOT WHOLE

Two of the three things asked of `0144` are **not built**, and nothing about the format blocks them
any more — `spec.md` L-1 names what they need. This is a lane's unfinished work:

- **Full-fidelity read.** Only `Player`'s straight run, `Token`, `Building` and `Effect` decode from
  a decoded form; the rest are carried opaque.
- **A save authored from nothing.** Blocked only on the above. No key has ever been minted here, so
  the silent-null trap stays the first test that writer owes.
- **The resume's divergence list did not shrink**: health, mana, statistics, inventories and timers
  are inside `Unit`, and `Unit` is in the set above.

`research/tools/savreplay` walks the stream by programme while keeping that counter, and is the
instrument the next round should be checked against file by file.
- That the original game **loads** a file this tree wrote. Nothing in this repository can answer
  that; the round trip is the strongest available substitute and it is what FR-8 is for.
- The embedded `&YA1` state store in the tail. Carried verbatim, never opened.
- Whether a load re-runs the terrain ingest before applying the block records. The resume's report
  measures the disagreement, which is the number that question wants, but it does not settle it.
