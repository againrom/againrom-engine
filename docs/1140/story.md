# Story 1140 — milestone-2 acceptance instrument

## Status

Base: engine main merge `61e4f6f` (story 1139, Form85, `formatVersion` 84→85,
hash pin `0x8a439cba352d8ae0`). This document covers this story's own
commits plus one correction pass answering
`pipeline/reviews/story1140-pass1.md` (RETURN: F-1 and F-2 returned the
story; F-6, F-7 and F-8 are debt findings corrected in the same pass; F-3
through F-5 stay open debt, recorded below with their own measurement), and a
reconciliation with engine main `53bad05` (stories 1141 and 1142, plus seat
hotfixes `f1053f5` and `06a6e423`). `f1053f5` fixed the F-1 refusal this
story's correction pass had named and excluded by mechanism
(`2027-09-07/game0031.sav`); the four `TestMilestone2*` instruments were
re-run on this reconciled candidate.

**The corpus itself grew again during this reconciliation.** The owner added
five more ROM1-written, round-trip-verified files under
`gameversions/saves/2027-09-07/` while this pass was running. Every count in
this document is re-measured against that current corpus, not the one this
story or its correction pass first ran against — because the corpus is
DISCOVERED, not pinned (F-1), that is the only honest way to report it. The
current corpus carries three `ResumeOriginalSave` refusals, from an importer
family `f1053f5` did not touch (a dead-actor identity bind and a late-dead
simulation stage neither `pkg/game/originaldead.go` nor
`pkg/sim/originaldead.go` currently admits). That defect is out of this
story's scope — see "Open debt" — and this document does not claim zero
refusals anywhere below.

`gofmt -l .` clean; `go build ./...` clean; `go test -trimpath -count=1 ./...`
(whole repository, asset-free) clean, 49 packages, 0 FAIL;
`scripts/check-no-game-assets.sh` clean; `internal/gatedtests`
`TestScanMatchesTheCheckedInPopulationList` clean, 209 registered names (208
on `origin/main` after story 1139, 1 added by this story's new release
witness, 0 removed); `pipeline/check-release-tests.sh` against both lawful
roots in one invocation: 8 packages, 209 gated tests, 209 of 209 ran and 0
lacked a subject, on EN and again on RU. Hashed-state instrument
(`TestHashIsPinned`, `pkg/sim`): pin unchanged at `0x8a439cba352d8ae0`
(`formatVersion` 85) — this story and its correction pass add tests, one
release witness and one new gate script; no importer or `pkg/sim` logic
changed, so the pin moves in no direction. Mission 10/20 `-trace -ticks 1`
UNSUPPORTED-node count: 0 on both missions, both roots, unchanged from
master's own recorded census — every change this story makes is reached only
through an original-save resume, a path the fresh mission drive does not
exercise. Knowledge pin unchanged at k3 (`abcbc0f0`) — no new claim was
needed; every field this story compares is covered by claims already active
in the pinned snapshot, including the four the correction pass re-cites
(`SAV-CELLREC-017`, `SAV-CELLLOAD-111`, `SESS-LOAD-009`, `SESS-LOSE-012`),
re-read here rather than newly published.

## Player result

Four of milestone 2's thirteen structure rows (`pipeline/SAV-COMPLETION.md`,
"Owner milestone 2") now carry the acceptance evidence the milestone itself
requires: an independent comparison of the SAV file's own bytes against
Againrom's restored live state, not a comparison of two decodes of the same
bytes. This closes review debt from story 1138
(`pipeline/reviews/story1138-pass1.md`, finding F-3): the prior corpus audit
(`originalunitresidual1138_corpus_test.go`) put `sav.File`'s own decoded
accessors on the file side of its comparison, so a wrong byte offset shared by
the decoder and the importer would have agreed on both sides and reported
zero mismatches. This story's four new instruments never call a decoded
accessor on the file side — every file-side value is read from `sav.File.Body`
by this story's own byte arithmetic, using only claim-cited offsets and the
handful of exported *location* fields (`WorldHalf.BlocksDataOff`,
`.CellRecDataOff`, `.SessionOff`) that name where a structure starts without
decoding what it contains.

The corpus this instrument reads is DISCOVERED — whatever
`AGAINROM_SAVE_CORPUS` currently contains — not pinned to a fixed file count
(`pipeline/reviews/story1140-pass1.md` F-1). Measured on this candidate: 82
preserved files total, 58 world-half, 23 between-mission, 1 unreadable
(`2026-08-27/EXP-0261-owner-runs/game9000.sav`, `sav.Open` magic mismatch,
skipped and logged, never counted as a comparison). Of the 58 world-half
files, 55 resume and are live-checked; 3 are refused by `ResumeOriginalSave`
and named by the harness rather than silently dropped — see the excluded-file
list below. One of those files, `gameversions/saves/2027-09-07/game0031.sav`,
used to be a fourth refusal: a pre-existing importer defect
(`pkg/game/savitemobjects1115.go:246`, "saved SAV Item 4 has ambiguous live
ownership" on this file, `pipeline/reviews/story1140-pass1.md` F-1). Seat
hotfix `f1053f5` narrowed the ambiguity guard to the five ownership slots
`importSavedItemObjects` actually follows, so the actor's transient item-cast
alias of its own held weapon (`actor+0x68`, `ITEM-CASTSTATE-056`) no longer
counts as a rival owner; `game0031.sav` now resumes and agrees with all four
instruments, both roots.

**The three files currently refused are not that defect and not this
story's to fix.** All three are ROM1-written saves under
`gameversions/saves/2027-09-07/`, added to the corpus after the reconciliation
above, and all three fail a different importer family — a dead-actor identity
bind (`pkg/game/originaldead.go`) or a late-dead simulation stage
(`pkg/sim/originaldead.go`) the importer does not currently admit, neither one
touched by `f1053f5`. Every `TestMilestone2*` instrument excludes each by
name and reason in its own log rather than hiding it inside a smaller count
or failing the whole instrument over files it does not own
(`milestone2ResumeRefusal`, `pkg/game/milestone2_acceptance_reader_test.go`)
— this is exactly the F-1 mechanism doing its job on a population it was not
built to know about in advance. Fixing this family is a separate, seat-owned
defect, out of this story's scope; this story neither touches
`pkg/game/originaldead.go`/`pkg/sim/originaldead.go` nor adds a named
exclusion to make any of the three files pass.

Over the 55 resumable world-half files, EN and RU, all four completed
structures show zero unexplained differences: map reopen, the terrain
block-plane delta (141,830 delta records), the 54-byte cell records' 9,428
key/baseline/layer-count fields, and the full 4,374-byte session block. One
of reserved `DIV-977` through `DIV-984` is now used — `DIV-977`, for the
mid-mission map-reopen mechanism ROM1's own claims describe against the
numeric-only mechanism this engine implements (see "Divergence rows" and
"Open debt"); `DIV-978` through `DIV-984` remain reserved and unused.

Nine structure rows remain open debt, in table order, starting from Unit /
Human records — see "Open debt." Per the brief's own escape valve, this story
lands the harness and the structures it completed rather than attempt the
remainder without matching rigor.

## Authority

Claims driving the four completed structures' independent field reads (all
already cited by the stories that built each importer; re-read here to derive
this instrument's own offsets, not reused as a citation of convenience):

- **`SAV-HEAD-025`** (map reopen): `Mission` is a mode switch read at load —
  nonzero reopens `"Scenario\" + MapName`, the file's own decoded name
  string. `Mission` itself is parsed from the map file name at save time by
  `R0512`, read whole by **`SESS-LOAD-009`** — not
  `PARTY-SESSION-008`, which is itself CONTESTED and explicitly defers this
  exact clause to `SESS-LOAD-009` ("this row does not restate it")
  (`pipeline/reviews/story1140-pass1.md` F-7). `SESS-LOAD-009` also states
  that once the mission number is already nonzero, the same routine does not
  parse a name at all: it "regenerates the name from `server+0x80`", the
  numeric-first direction this engine's own importer takes. Neither claim
  says whether `R0512` is the routine `SAV-HEAD-025`'s own load arm
  (`L08281`) belongs to, so this narrows — without settling — the "two
  mechanisms, outcome equivalence not mechanism identity" framing in "Open
  debt"; recorded as `DIV-977`.
- **`SAV-BLOCK-011`**, **`SAV-BLOCK-012`** (terrain block plane): the delta's
  packed `u32` layout (`cell<<16 | dyn<<8 | static`), the `[0x807,
  0x807+0xe5e7)` cell window, `dyn > 0x0f`, and strictly increasing cells.
- **`SAV-CELLREC-017`**, **`SAV-CELLLOAD-111`** (54-byte cell records): the
  record's two-part shape — a 2-byte key (`SAV-CELLREC-017`) plus a 52-byte
  payload — and the compared fields' own offsets within it: payload `+0x00`
  cost baseline, `+0x01` static baseline, `+0x02` layer count
  (`SAV-CELLLOAD-111`). **`SAV-CELLREC-032`** (partially RETRACTED, below)
  gives the 54-byte record stride (`2 + 54 x count + 4`), a measurement its
  retraction leaves explicitly untouched. **`SAV-CELLLOAD-109`** gives the
  overlay-not-replace mechanism this row's own live/absent distinction
  relies on ("a saved key overwrites its constructed payload … a
  construction-only node survives"). **`SAV-CELLLOAD-110`** names the ten
  identity dwords this row does not compare (story 1131's own scope) —
  cited for that boundary, not as a source of the four fields this
  instrument reads.
- **`SAV-SESS-031`** (session block): the offsets and widths of six of the
  seven compared spans — registers, latches, RawHead, RawMid, diplomacy, and
  the Won counter (`+0xb3ac`, "the win counter") — and, read together with
  **`DIV-927`**, the three unpromoted gaps (8, 6 and 4 bytes) this
  instrument excludes rather than compares. `SAV-SESS-031` locates the
  seventh span, `+0xb3b4`, but does not name it: its own text calls the
  neighbouring `+0xb3b0` "not named" and gives `+0xb3b4` no name at all.
  **`SESS-LOSE-012`** is the row that names it ("opcode 5 does
  `session+0xb3b4++`"), the "Lose" counter `DIV-927` already calls by that
  name.
- **`SAV-DOC-053`**: `sav.File.Body` is the whole decoded stream, the
  authority this instrument reads directly instead of any `Fields`/`Object`
  decode built on top of it. Partially RETRACTED: the retraction withdraws
  its 7-of-18 success endpoint, its eleven claimed Unit-subtree failures and
  its terminal-padding reading of the trailer's trailing 400 bytes ("loaded
  state, not padding", corrected by `SAV-UNITCORP-157`); the clause this
  instrument relies on, the static top-level document order, is explicitly
  left untouched by that retraction ("the static top-level order remains
  right").

`SAV-SESS-031` marks `session+0x08`'s 48 bytes (RawHead) and `session+0xa828`'s
400 bytes (RawMid) **Unknown** — no claim promotes a meaning for either span.
Both are still compared here, byte for byte, as round-trip fidelity on an
opaque carried live value (`sim.World.RawSessionHead`/`RawSessionMid`), the
same treatment story 1138 gives the trailer's own opaque dwords. A comparison
of a field a claim marks Unknown is not the same as excluding a field for
being Unknown, and nothing was excluded on that ground for these four
structures (contrast the session block's DIV-927 gaps, excluded because no
claim promotes them to a field at all).

## As-built behaviour

**`pkg/game/milestone2_acceptance_reader_test.go`** (build tag
`sessioncorpusaudit`): the shared harness. `milestone2Corpus` walks
`AGAINROM_SAVE_CORPUS`, opens each `.sav` with `sav.Open` (the only
container-level trust this instrument extends — a custom run/literal codec,
not a claimed game-data format), and calls a per-structure test function once
per file with the raw body, the decoded `*sav.File` (for *location* fields
only) and a `*FrontEnd`. `rawU16`/`rawU32` are this file's own little-endian
readers, used by every structure test in place of `sav.Fields`/`Object`
decode. Correction pass: `milestone2ResumeRefusal` names one file
`ResumeOriginalSave` refuses, by relative path and the importer's own error;
`milestone2LogRefusals` prints one line per refusal under each instrument's
own label. A refusal is now excluded from that instrument's live comparison
and logged by name, never a `t.Fatalf` that failed the whole population over
one file (`pipeline/reviews/story1140-pass1.md` F-1).

**`pkg/game/milestone2_acceptance_mapreopen_test.go`**
(`TestMilestone2MapReopen`): reads the head's `MapName` and `Mission` at
their claim-given body offsets, checks the file-internal stem-vs-Mission
relationship `SAV-HEAD-025`/`SESS-LOAD-009` predicts, then resumes the save
and checks live `Mission.Number`/`Mission.Address` against the same file
bytes. Result: 81 files with a readable head of 82 total (1 unreadable; 23
between-mission, correctly producing no live comparison; 55 resumed and
live-checked, 3 refused and excluded by name — see "Player result"), 0
mismatches on all three checks, identical on EN and RU.

**`pkg/game/milestone2_acceptance_terrain_test.go`**
(`TestMilestone2TerrainBlockPlane`): reads each delta record at
`WorldHalf.BlocksDataOff+4*i`, unpacks Cell/Dyn/Static with this file's own
shifts, checks `SAV-BLOCK-011`'s window/gate/ordering invariants against its
own transcription, and compares against `sim.World.SavedCellPlanes()`.
Result: 141,830 delta records over the 55 live-checked world-half files (3
refused and excluded by name), 0
Static/Dynamic mismatches, 0 invariant violations, map widths `{80: 34, 144:
21}` — the "256-wide maps Unknown" gap this row's own
`pipeline/SAV-COMPLETION.md` entry names does not arise in this corpus,
which the test logs rather than assumes.

**`pkg/game/milestone2_acceptance_cellrecords_test.go`**
(`TestMilestone2CellRecords`): reads each 54-byte record at
`WorldHalf.CellRecDataOff+54*i` for Key/BaselineCost/BaselineStatic/
LayerCount only (this row's own stated scope), compares against
`sim.World.SavedStructures()`'s cell baselines and `SavedCellRecords()`'s
layer counts. A file key absent from a live carrier is counted, not treated
as a mismatch in the three per-field counters — both live carriers are
populated only where the importer actually materializes a record, a
different and already-documented admission rule from "the file disagrees
with what was imported." Correction pass: the absence counters
(`baselineNotLive`, `layerNotLive`) now gate the result exactly like a
mismatch would — previously they were printed but never checked, so
disabling a whole live importer reported "0 mismatches" (F-2). Result: 9,428
records over the 55 live-checked world-half files (3 refused and excluded by
name), 0 duplicate keys, 0 BaselineCost/BaselineStatic/LayerCount mismatches,
0 records absent from either live carrier.

**`pkg/game/milestone2_acceptance_session_test.go`**
(`TestMilestone2SessionBlock`): its own sub-offset constants, independently
transcribed from `SAV-SESS-031` (matching, not reusing,
`pkg/formats/sav/world.go`'s unexported constants of the same values —
agreement between two independent transcriptions is itself a check), the
Lose offset's own name corrected to `SESS-LOSE-012` (see "Authority").
Compares the clock pair, all 100 trigger registers, all 1,000 latches,
RawHead and RawMid byte-for-byte, the full 50x50 diplomacy matrix, and
Won/Lose, against `sim.World`'s `SessionClock`/`ScriptRegisters`/
`ScriptLatched`/`RawSessionHead`/`RawSessionMid`/`Relations`/`ScriptCounters`.
DIV-927's three gaps are excluded by construction (this file's offset table
has no entry for them). Roster slot 0's live diplomacy exclusion
(`sim.relationIndex`) is logged, not silently assumed zero: 0 files had a
nonzero file byte there anyway. Result: 55 live-checked world-half files (3
refused and excluded by name), 0 mismatches across all eight checks,
identical on EN and RU.

**`pkg/game/milestone2_mapreopen_release_test.go`** (new,
`TestReleaseMilestone2MapReopen1140`, no build tag): the one structure among
the four completed here with no pre-existing release witness. Terrain, cell
records and the session block already had one each
(`originalcellplanes_release_test.go`, `originalcellrecords1131_release_test.go`,
`originalsession1130_release_test.go`, all confirmed loading a real preserved
save via `groundCorpusFile`). This new witness loads `game0021.sav`, decodes
its `Mission`/`MapName` normally (release witnesses are not held to this
story's independence rule — that rule is scoped to the opt-in acceptance
family), resumes it with `ResumeOriginalSave`, and asserts live
`Mission.Number`/`Address` reproduce the file's own values. Registered in
`internal/gatedtests/testdata/population.txt`.

**`scripts/check-milestone2-acceptance.sh`** (new, correction pass,
`pipeline/reviews/story1140-pass1.md` F-6): runs the four `TestMilestone2*`
instruments against one or more asset roots and fails on the first FAIL.
Nothing in any existing gate built or ran this family (behind
`sessioncorpusaudit`, the same tag every 1130-1139 corpus audit already
used, and `pipeline/check-release-tests.sh` runs `go test` untagged) — which
is why the acceptance declaration went silently red between this story's own
measurement and its push (F-1) and no gate caught it. This script is not yet
wired into any seat gate; see "Open debt."

## Divergence rows

**`DIV-977`** (reserved for this story, `pipeline/ALLOCATIONS.md`): the
mid-mission map-reopen mechanism. `SAV-HEAD-025` describes the original's
load arm reopening from the file's own stored `MapName` string when Mission
is nonzero; this engine's importer resolves the map from `Mission`'s numeric
value alone through its own table. `SESS-LOAD-009` narrows the gap —
`R0512` also derives the name from the number once the number is
already nonzero — without settling whether that routine is the same one
`SAV-HEAD-025` describes (see "Authority" and "Open debt"). Zero live
Address mismatches on the 55 live-checked world-half saves, both roots, is
behavioural equivalence over this corpus, not proof of mechanism identity.

All four completed structures otherwise showed zero unexplained differences
across the 55 resumable world-half files, both lawful roots; `DIV-978`
through `DIV-984` (reserved for this story, `pipeline/ALLOCATIONS.md`) are
unused and remain available for a later story completing one of the
remaining nine rows.

## Proof

- `gofmt -l .`: clean.
- `go build ./...`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean,
  209 registered names (`population.txt` is 213 lines, 4 comments) — 208 on
  `origin/main` after story 1139, 1 added by this story, 0 removed.
- `scripts/check-no-game-assets.sh`: clean.
- **Re-measured after reconciling with engine main `53bad05`**, which
  includes seat hotfix `f1053f5` (ledger row `docs/HOTFIXES.md`, "held weapon
  U68 alias, ambiguous ownership") landed after the correction pass above.
  That hotfix is the fix the "Player result" and "Open debt" sections
  describe; it removes the one `ResumeOriginalSave` refusal these four
  instruments used to report (`2027-09-07/game0031.sav`).
- **The corpus grew again, by five files, while this reconciliation was in
  progress**, and the counts below are the final re-measurement against that
  current corpus, not an intermediate one. Three of the five new files are
  refused — a different importer family from the one `f1053f5` fixed (see
  "Player result" and "Open debt") — and every count and quoted log line
  below includes them as named exclusions, not as zero.
- `TestMilestone2MapReopen` (`-tags sessioncorpusaudit`, fresh run, both
  roots, corpus as of this candidate — 82 preserved files, discovered not
  pinned): `map reopen: 81 file(s) with a readable head (23 between-mission),
  55 resumed and live-checked, 3 refused`; three `map reopen: excluded
  (ResumeOriginalSave refused)` lines — `2027-09-07/game0011.sav: original
  dead actors: unsupported late-dead state: stage 2 HP -14 timer 0 runtime 90
  cell 0x3332 fine 128,128`; `2027-09-07/game0032.sav: original dead actor
  0x123fbb48: MapUnitID 0 is missing, ambiguous or party-bound`;
  `2027-09-07/game0033.sav: original dead actor 0x123fbb48: MapUnitID 0 is
  missing, ambiguous or party-bound`; `map reopen mismatches: 0
  name/Mission stem, 0 live Mission.Number, 0 live Address` — identical on EN
  and RU.
- `TestMilestone2TerrainBlockPlane` (same, both roots): `terrain block plane:
  55 world-half file(s), 141830 delta record(s) checked; map widths
  map[80:34 144:21]; 3 refused`; the same three excluded-file lines as above;
  `terrain block plane mismatches: 0 Static, 0
  Dynamic; SAV-BLOCK-011 invariant violations: 0 window, 0 gate, 0 order` —
  identical on EN and RU.
- `TestMilestone2CellRecords` (same, both roots): `cell records: 55
  world-half file(s), 9428 record(s) checked (0 duplicate key(s),
  last-write-wins), 3 refused`; the same three excluded-file lines as above;
  `cell records not carried live: 0 baseline,
  0 layer count` (both counters now gate the result — see the mutation check
  below); `cell records mismatches: 0 BaselineCost, 0 BaselineStatic, 0
  LayerCount` — identical on EN and RU.
- `TestMilestone2SessionBlock` (same, both roots): `session block: 55
  world-half file(s) checked, 3 refused`; the same three excluded-file lines
  as above; `session block mismatches: 0
  clock, 0 register, 0 latch, 0 RawHead, 0 RawMid, 0 diplomacy, 0 Won, 0
  Lose`; `diplomacy roster-slot-0 rows/columns with a nonzero file byte: 0`
  — identical on EN and RU.
- **F-2 mutation check, post-fix, re-run on this candidate.** With
  `applyOriginalCellRecords` disabled (a single `return nil` inserted before
  its own body, M7 in `pipeline/reviews/story1140-pass1.md`'s table),
  `TestMilestone2CellRecords` now reports `cell records not carried live: 0
  baseline, 9428 layer count` and **FAILS**: `0 BaselineCost, 0
  BaselineStatic, 0 LayerCount mismatch(es); 0 baseline, 9428 layer count not
  carried live`. With `applyOriginalStructures` disabled the same way (M8),
  it reports `9428 baseline, 0 layer count` and **FAILS** the same way. Both
  mutations were applied to a scratch copy of the two importer files, run
  once each on EN, and reverted (verified against a saved copy); neither
  mutation is in this candidate.
- `TestReleaseMilestone2MapReopen1140` (fresh run, both roots): `game0021.sav`
  decodes to `Mission=10, MapName="10.alm"`; live `Address="scenario/10.alm"`
  — matches, on both roots.
- `scripts/check-milestone2-acceptance.sh gameversions/en gameversions/ru`
  (new, this pass, F-6): both roots print `ok` for all four `TestMilestone2*`
  tests; exit 0.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 209 gated
  tests, 2 roots; **209 of 209 ran and 0 lacked a subject, on EN and again on
  RU.**
- Hashed-state instrument: `TestHashIsPinned`,
  `TestThePinnedDigestIsFNV1aOfThePinnedBytes` and
  `TestHashIsFNV1aOverExactlyTheByteForm` (`pkg/sim/hash_test.go`) all pass
  against the same literal pinned digest story 1139 landed,
  `0x8a439cba352d8ae0` (`formatVersion` 85, `pkg/sim/binary.go`) — this story
  moves it in no direction and updates no pin.
- `go build ./cmd/missionrun` plus a `-trace -ticks 1` UNSUPPORTED-node count
  on mission 10 and mission 20: **0 on both missions, both roots** (EN: m10 0,
  m20 0; RU: m10 0, m20 0), unchanged from master's own recorded census
  (`pipeline/milestone-baseline.txt`) — this story's changes are reached only
  through an original-save resume, a path the fresh mission drive does not
  exercise.

## Open debt

**Nine of the milestone's thirteen structure rows still need the same
independent acceptance treatment**, in `pipeline/SAV-COMPLETION.md`'s own
table order, starting from the next one: Unit / Human records, then Building,
Sack and items, Effect/SpellEffect, Projectile store, Group, Player, dead and
object graph, and the trailer. Per the brief's own escape valve, this story
lands the harness and the four structures it completed rather than attempt
the remainder at lower rigor; `DIV-978` through `DIV-984` stay reserved for
whichever of these a following story explains an unexplained difference with
(`DIV-977` is now used — see "Divergence rows").

**The corpus this instrument reads is DISCOVERED, not pinned to a fixed
count** (`pipeline/reviews/story1140-pass1.md` F-1).
`pipeline/SAV-COMPLETION.md`'s own acceptance paragraph names "the 26
preserved mission saves"; this candidate's instruments find 82 preserved
files total (58 world-half, 23 between-mission, 1 unreadable) under the
corpus's current contents, up from 72 (51 world-half) when this story first
measured — ten files were added under `gameversions/saves/2027-09-07/` across
this story's push, its correction pass and this reconciliation, most recently
five more added by the owner while this reconciliation itself was in
progress. Every `TestMilestone2*` instrument reports whatever the corpus
currently holds, plus any file `ResumeOriginalSave` refuses, by name and
reason, rather than either a fixed count or a silent shrink — this is not a
historical property of an earlier pass, it is what the corpus holds as of
this candidate's own measurement, and it currently includes 3 refused files
(see below). Updating the "26" figure in `pipeline/SAV-COMPLETION.md` is a
seat-repo edit outside this story's own repository; this document's own
counts are the current, reproducible ones.

**Closed: the item-ownership refusal.** One preserved ROM1-written save the
LOAD GAME window listed, `gameversions/saves/2027-09-07/game0031.sav`, used
to be refused by a pre-existing importer defect, not this story's own code:
`importSavedItemObjects` (`pkg/game/savitemobjects1115.go:246`) refused any
item object whose incoming reference count was not exactly 1, and object 4 in
this file had 2 — a Weapon referenced from the same actor twice, once as
`HeldWeapon` and once as `Unit`'s transient item-cast alias `U68`
(`ITEM-CASTSTATE-056`). Seat hotfix `f1053f5` (ledger row
`docs/HOTFIXES.md`) narrowed the ambiguity guard to the five ownership slots
`importSavedItemObjects` actually follows (Contents, Inventory, HeldWeapon,
HeldShield, Worn), so the alias no longer counts as a rival owner; a genuine
multi-owner conflict across those five slots is still refused. Re-measured
after reconciling with engine main `53bad05` (which includes the hotfix):
`game0031.sav` now resumes and agrees with all four instruments on both
lawful roots. Still Unknown: whether `game0031.sav`'s original
doubly-referenced Item 4 is a shape ROM1 itself would also have rejected;
only our reader has read this file.

**Open, and not this story's: a dead-actor refusal family, currently
excluding three corpus files, being handled separately.** After the closure
above, the owner added five more ROM1-written, round-trip-verified files
under `gameversions/saves/2027-09-07/`. Two resume cleanly (`game0005.sav`,
a world-half save, and `game0034.sav`, a between-mission save) and agree with
every instrument. Three are refused by `ResumeOriginalSave`, from a family
`f1053f5` does not touch: `game0011.sav` fails
`pkg/sim/originaldead.go`'s `deadStateFault` guard, "unsupported late-dead
state" (the guard admits stages 3 through 5 only; this file's dead actor is
at stage 2); `game0032.sav` and `game0033.sav` both fail
`pkg/game/originaldead.go`'s identity bind on the same dead-actor address,
"`MapUnitID 0` is missing, ambiguous or party-bound". Every
`TestMilestone2*` instrument excludes all three by name and reason in its own
log (see "Proof") rather than going red or silently shrinking its count —
exactly the F-1 mechanism working as designed, on a refusal family the
mechanism was not built with in mind. This story does not fix this family:
it does not touch `pkg/game/originaldead.go` or `pkg/sim/originaldead.go`,
and it does not add a named exclusion to force any of the three files green.
The defect is the seat's, tracked separately from this story.

**Closed by story 1143.** Both guards this paragraph names were this
project's own and narrower than published law: `SAV-DEADLOAD-125` (promoted)
is that ROM1's own load never joins a dead actor to an authored map unit at
all, and `SAV-DEADLOAD-128` names the decay ladder's Stage 2 an actor-proven
route. `deadStateFault` now admits Stage 2..5; `applyOriginalDead` now
carries a `MapUnitID 0` record (the observed producer, `0x123fbb48`, is a
dead hired mercenary, Class Human, never an ALM placement) as a virtual dead
actor with no live entity. All three files now resume and agree with every
`TestMilestone2*` instrument on both lawful roots; the corpus this paragraph
measured at 3 refused now measures 0. The exclusion mechanism this paragraph
describes is unchanged and still active for any future refusal.

**Map reopen's two mechanisms are checked for outcome equivalence, narrowed
but not settled as mechanism identity** (`DIV-977`). `SAV-HEAD-025` describes
the original's load arm reopening from the file's own stored `MapName`
string when Mission is nonzero; this engine's importer instead resolves the
map from `Mission`'s numeric value alone through its own table
(`OriginalSaveResume.MapName` is a display label only). `SESS-LOAD-009`
narrows this: once the mission number is already nonzero, the same original
routine (`R0512`) does not read a stored name at all — it
"regenerates the name from `server+0x80`", the numeric-first direction this
engine also takes. Neither claim states whether `R0512` is the same
routine as `SAV-HEAD-025`'s own cited load arm at `L08281`, so this narrows
without settling the question outright. The two reach the same map on every
live-checked corpus member (0 Address mismatches, both roots), which is what
this story's acceptance check tests; that remains evidence of behavioural
equivalence over this corpus, not proof for a save the corpus does not
contain (a mission number whose map name's stem disagrees with it — which
`SAV-HEAD-025` itself treats as malformed, not a case either mechanism is
specified to handle).

**The compared population is supplied by the decoder under test, for three
of the four structures** (`pipeline/reviews/story1140-pass1.md` F-3). The
terrain and cell-record instruments take their own record counts
(`len(w.Blocks)`, `w.CellRecCount`) from `pkg/formats/sav`'s own decode
rather than reading the count word at `WorldHalf.BlocksOff`/`.CellRecOff`
themselves, the way `milestone2_acceptance_mapreopen_test.go` already walks
the head unaided. Measured by the review: halving either decoded count
halves the instrument's own compared population and still reports zero
mismatches (M3, M4 in the review's mutation table). Not fixed in this pass —
recorded as debt for whichever story next extends these two instruments.

**What the four instruments actually compare is narrower than their file
counts suggest, for several spans** (`pipeline/reviews/story1140-pass1.md`
F-4). Measured by the review, corpus-wide: `RawMid` (400 bytes) is zero in 54
of 54 world-half files and `Lose` is zero in 54 of 54 — both spans carry no
discriminating information over this corpus, the latter reproducing
`SAV-SESS-031`'s own "0 in 11/11" at roughly five times the population.
Session registers are file-nonzero in 637 of 5,400 slots (11.8%), and a
six-file subset (`2026-08-15`) found 21 of 94 nonzero slots re-derived by
`presetRegisters` after import, so those particular comparisons prove the
compiled map rather than the reader. The terrain row's 137,829 delta-record
comparisons include only 9,070 (6.6%) where the file's static byte differs
from the plane the ALM alone would produce; the other 128,759 already agree
before any overlay is applied. None of this makes a reported zero wrong; it
makes the headline record counts a poor description of how much each span is
actually exercised. Not fixed in this pass.

**This candidate's new gate script, `scripts/check-milestone2-acceptance.sh`,
is not yet wired into any seat gate** (`pipeline/reviews/story1140-pass1.md`
F-6). It runs clean on both lawful roots (see "Proof") but nothing calls it
automatically. The seat needs to register a call to it, both roots, after a
landing that touches `pkg/game/original*.go` or the milestone-2 acceptance
instruments themselves — the same role `pipeline/check-release-tests.sh`
plays for the untagged gated-test population. Registering it is a
`pipeline/`-repo edit outside this repository's own boundary.

## Touched surfaces

- `pkg/game/milestone2_acceptance_reader_test.go`: shared corpus-walk harness
  (`sessioncorpusaudit`); correction pass adds `milestone2ResumeRefusal` and
  `milestone2LogRefusals` (F-1).
- `pkg/game/milestone2_acceptance_mapreopen_test.go`: `TestMilestone2MapReopen`
  (`sessioncorpusaudit`); correction pass fixes the resume-refusal handling
  (F-1) and the `PARTY-SESSION-008` → `SESS-LOAD-009` citation (F-7).
- `pkg/game/milestone2_acceptance_terrain_test.go`:
  `TestMilestone2TerrainBlockPlane` (`sessioncorpusaudit`); correction pass
  fixes the resume-refusal handling (F-1).
- `pkg/game/milestone2_acceptance_cellrecords_test.go`:
  `TestMilestone2CellRecords` (`sessioncorpusaudit`); correction pass fixes
  the resume-refusal handling (F-1), gates `baselineNotLive`/`layerNotLive`
  (F-2) and the `SAV-CELLREC-032`/`SAV-CELLLOAD-109/110` → `SAV-CELLREC-017`/
  `SAV-CELLLOAD-111` citation (F-7).
- `pkg/game/milestone2_acceptance_session_test.go`:
  `TestMilestone2SessionBlock` (`sessioncorpusaudit`); correction pass fixes
  the resume-refusal handling (F-1) and the Lose-counter citation to
  `SESS-LOSE-012` (F-7).
- `pkg/game/milestone2_mapreopen_release_test.go`:
  `TestReleaseMilestone2MapReopen1140`. Unchanged by the correction pass.
- `scripts/check-milestone2-acceptance.sh`: new, correction pass (F-6).
- `docs/DIVERGENCES.md`: new row `DIV-977` (correction pass, F-5).
- `internal/gatedtests/testdata/population.txt`: one new release-witness
  entry. Unchanged by the correction pass.
- `docs/1140/story.md`: this document.
