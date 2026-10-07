# Story 1138 — SAV trailer and four unit residual fields

## Status

Gates run on the exact candidate commit `5aeb3df8` (base `3d98dea2`, current
main after two rebases onto lanes that landed mid-story — a hotfix and its own
correction, both to an unrelated `docs/DIVERGENCES.md` row); see "Proof" for
the full list and exact counts. `gofmt -l .` clean; `go build ./...` clean;
`go test -trimpath -count=1 ./...` (whole repository, asset-free) clean, 49
packages, 0 FAIL, including `internal/gatedtests`' checked-in
`testdata/population.txt` census: 207 registered names (205 before this story,
2 added, 0 removed) for the two new release witnesses this story adds;
`scripts/check-no-game-assets.sh` clean; `pipeline/check-release-tests.sh`
against both lawful roots in one invocation, 8 packages, 207 gated tests, 207
of 207 ran and 0 lacked a subject on EN and again on RU; mission 10/20
`-trace -ticks 1` UNSUPPORTED-node count 0 on both missions, both roots,
unchanged from master's own recorded census — this story's change is reached
only through an original-save resume, a path the fresh mission drive does not
exercise. The knowledge pin moved k2 → k3 (research `a689c582`, tag
`abcbc0f0`) to reach this story's own claims (`EXP-0321`, `EXP-0312`,
`EXP-0315`, `EXP-0311`). No writer; no production mission SAVE path — both
explicitly out of scope.

## Player result

A mission loaded from an original ROM1 SAV now decodes and names the world
trailer block (`world+0x118`, 400 bytes) and four unit-record fields
(`+0x8e`, `+0x90`, `+0xa4`, `+0x18`) instead of skipping the block uninspected
or carrying the fields under a raw offset name with no cited meaning. This is
milestone 2's own last open structure row
(`pipeline/SAV-COMPLETION.md`, "Owner milestone 2," step 8).

Two of the four unit fields already reach a live consumer this engine has,
and now feed it directly from the file's own stored value, never recomputed
at load: `+0x8e`/`+0x90` (own carried weight and its paired load word) drive
`sim.ActorLoad`'s speed-penalty threshold exactly as before, and `+0xa4`
(sight) drives `sim.SourceActor.ScanRange` through its own existing high-byte
read. Both wirings were already correct before this story — `SAV-794`'s own
finding is that ROM1's load path does not recompute either field from a
record's own container or stats, so the pre-existing "plain read, no derive"
wiring already matched the file. What this story adds is the citation record
proving that match, plus the STORY-required name (`StatOwnWeight`/`StatLoad`
in place of `StatU8E`/`StatU90`) and a fixed decode/round-trip proof at scale
(see "Proof").

The other two fields have no located consumer anywhere in this engine: the
trailer's two debug-verbosity toggle dwords (`TurnTracing`, `ScriptTracing`)
and `+0x18` (`Token18`, a recipient-publication mask in ROM1 that this engine
has no fog-of-war/notification system to feed). Both are carried decoded,
typed, and round-trip byte-identical; neither is wired to invented live
behaviour (DIV-968, DIV-969). No writer exists for any of the four fields or
the trailer, and no production mission SAVE path was added — both per the
owner's standing rule, unchanged by this story.

## Authority

Claims specifically driving what this story built:

- **`SAV-646`** (High): `world+0x118` is a heap-allocated, exclusively-owned
  400-byte block with its own construct/use/teardown lifecycle, not an
  embedded array or the session object.
- **`SAV-790`** (High): one non-virtual transfer site serves both directions
  and moves the whole `0x190` (400) bytes unconditionally on load, regardless
  of the stream's marker/global arm.
- **`SAV-647`** (promoted, amended by `SAV-694`): trailer dwords 0 and 1 are
  "turn tracing"/"script tracing" debug-verbosity toggles, each
  read-branch-flip-logged against matched on/off strings, plus four further
  sites gating `Script:`-prefixed diagnostic output on the same two dwords.
- **`SAV-694`** (High): a conditional incoming debug-command route (command
  46, subcommand 3/19, gated on unsigned `Player+0x68 > 50` and two further
  bytes) can toggle both dwords, but ordinary runtime activation — what
  produces the required gate byte, or an in-game key path — is not
  established.
- **`SAV-791`** (Medium, complete for one instruction form): a full
  `[base+0x118]` displacement census crossed against the world singleton's
  own reference census finds a located consumer for only dwords 0 and 1;
  dwords 2..99 have none.
- **`SAV-792`** (promoted): `+0x8e` is an incrementally maintained
  accumulator with a second, far-better-connected writer (13 call sites) than
  the published derive (0 direct callers, reached only through a vtable
  slot); that second writer also recomputes `+0x90` from `+0x8e` and the
  container's own running weight, instruction-for-instruction identical to
  the derive.
- **`SAV-793`** (High for width/signedness/boundary): the carried-load
  arithmetic is a 16-bit add with a signed 64000-threshold compare; a stored
  value at or above 0x8000 reads back negative through `MOVSX` and cancels
  the speed penalty rather than deepening it. No preserved record reaches the
  wrap (largest corpus `+0x90` is 457).
- **`SAV-794`** (High for "the load does not recompute"): the load arm reads
  `+0x8e`, `+0x90` and `+0xa4` through the same archive helper and keeps the
  stored value, even on a record where no recompute could have produced it
  (own weight 178, stored load 181, empty container, reproduced across five
  streams).
- **`SAV-795`** (promoted): `+0xa4`'s corpus distribution is six values on
  `Unit` (4–9 whole cells) and fourteen on `Human`, wider than an earlier
  enumeration; the low byte is zero on all 826 sampled `Unit` records, and
  the high byte is the whole-cell scan range this engine's own
  `ScanRange = basis.Sight >> 8` already reads.
- **`SAV-796`** (Medium): the "sight `+0xa4 = 0` occurs in no record" reading
  survives on the 1,274-record original-authored partition; the three
  exceptions are project-authored candidates and the original's own resave
  of one of them, not the constructor's default (which is 1280, not 0).
- **`SAV-636`** (High, amended): Unit `+0x18` is `Token+0x18`, the same byte
  Building already exposes as `Token18`, since `Unit::Serialize` begins at
  Token's own unadjusted `this`.
- **`SAV-678`** (High for the mechanism, Medium for breadth): `Token+0x18` is
  a 16-bit recipient-publication mask three located helpers set/clear/test
  against `Player+0x2c`, revoked-then-republished rather than monotonic; no
  universal fog-of-war meaning is established.
- **`SAV-797`** (Medium, corpus census): the field takes exactly two values
  corpus-wide, 2 on 1,404 of 1,420 records and 0 on the other 16; the
  terminal lifecycle stage is ruled out as the sole cause.

Claims read for completeness, confirming boundaries this story does not
cross:

- **`SAV-648`** (High): the marker-adjacent global `[L07886]` — a
  plain int with exactly three whole-image touch points, written directly
  after the stream's `0xbadface1` marker — is a separate field from the
  trailer's own 100 dwords; this project's narrow reader still skips it
  uninspected (`document.go`'s own `"trailer global"` skip, unchanged).
- **`SAV-649`** (Medium): the 400-byte tail is zero in all 45 parsed research
  streams; `[L07886]` itself (not part of the trailer) is 0 in 44 and 520
  in exactly one, ruling out a monotonic counter reading for that separate
  field.
- **`SAV-657`** (High for the identification): independently confirms
  `world+0x118` is genuinely the world/"server singleton" object
  (`[L00285]`), through different code from `SAV-646`, and corrects an
  unrelated citation error about a distinct `this+0x14` object with no
  connection to the trailer.
- **`SAV-TRAIL-026`** (High for the structure): the stream's tail is three
  writes — the `0xbadface1` marker, `[L07886]`, then `world+0x118`'s own
  transfer, in that order, with the trailer last.
- **`SAV-635`** (High): four of `SAV-UNITFLD-049`'s five listed Unknowns —
  `+0x8e`, `+0x90`, `+0xa2`, `+0xa4` — were already named at High confidence
  by pre-existing claims (`ITEM-LOAD-005`, `HERO-SIGHT-007`, `HERO-REGEN-021`)
  that never cross-referenced it; `+0xa2` is the health-regeneration
  remainder byte, a different field from this story's four, already closed
  elsewhere. The fifth, `+0x18`, is `SAV-636`.
- **`SAV-UNITFLD-049`** (active, amended; two of its own enumeration
  sentences separately retracted and narrowed by `SAV-795`/`SAV-796`): the
  original ledger row naming all fourteen `Unit` stat words and grading
  `+0x8e`, `+0x90`, `+0xa2`, `+0xa4`, `+0x18` Unknown — the antecedent this
  story's own `EXP-0321` claims (`SAV-792`–`SAV-797`) deepen past
  `SAV-635`'s cross-reference-only closure.

## As-built behaviour

**`pkg/formats/sav`**: `TrailerBody{TurnTracing, ScriptTracing uint32;
Residual [98]uint32}` (`archive_document.go`) replaces the flat `[100]uint32`
in `archiveDocument` and is decoded/serialized by `readTrailerBody`/the
existing writer loop in both directions of the wide graph path
(`parseArchiveDocument`/`serializeArchiveDocument`). `DocumentData.Trailer`
keeps its original `[100]uint32` gob wire shape unchanged — gob distinguishes
an array from a same-width struct, and a frozen native envelope already
depends on that exact shape — with `trailerBodyToArray`/`trailerBodyFromArray`
converting at the two boundary sites (`document_data.go`). The narrow reader
(`document.go`) gains its own `trailer TrailerBody` field via the same
`readTrailerBody`, in place of the prior unconditional `w.skip(...400)`, and
`container.go`'s `File` exposes it as `File.Trailer`. `program.go` renames
`StatU8E`/`StatU90` to `StatOwnWeight`/`StatLoad` (wire keys `"U8E"`/`"U90"`
unchanged) with a doc comment citing `SAV-792`/`SAV-794`. `actorbasis.go`
gains `ActorBasis.Token18 uint16` (`r.value("T18")`, the same generic
Token-head member `class.go` already declares and Building already reads)
and a citation comment above the pre-existing `Sight` read. `holdings.go`
gains a citation comment on `ActorLoadState` (no logic change: it already
read `+0x8e`/`+0x90` directly with no recompute).

**`pkg/sim`**: `actorload.go` gains citation comments on `CurrentLoad()`
(`SAV-793`'s width/signedness/boundary) and `finishLoadMutation()`
(`SAV-792`'s second-writer finding); no logic change — the existing 16-bit,
signed-threshold arithmetic already matched.

**`pkg/game`**: `originalholdings.go` gains a citation comment on
`originalActorLoad()` (`SAV-794`'s own example); no logic change — it already
set `Load` directly from the file. `originalactorregistry.go` gains a
citation comment above `ScanRange = uint8(basis.Sight >> 8)`; no logic
change. `nativecity.go`'s Weight/Load comment is rewritten to record that
`SAV-792` sharpens, without resolving, whether a native (non-imported) city
party's own weight/load should stay zero (DIV-880). `originalhuman.go` and
`cmd/savtool/party.go` get the mechanical `StatU8E`/`StatU90` rename.

**Tests**: `sav_test.go` adds `fixture.trailerTurnTracing`/
`trailerScriptTracing`, a byte-identity case with named nonzero dwords, and
two new tests (`TestTrailerDwordsAreNamedAndResidualCarried`,
`TestTrailerResidualIsByteIdenticalOnReExport`).
`actorbasis1138_test.go` pins the sight decode's whole-cell high byte and
`Token18`'s carry. `program_test.go` adds a `sight` field to its `wantChar`
fixture (previously a hardcoded filler word) so both existing character
fixtures keep their exact prior byte output. `actorload1138_test.go` pins
that `originalActorLoad` does not recompute from the container and returns
nil for an absent source. `originalunitresidual1138_corpus_test.go`
(`sessioncorpusaudit`, on `TestGroundContainerCorpusAudit1136`'s own shape)
cross-checks OwnWeight/Load/sight against live state for every living,
map-placed actor in the full preserved corpus, logs Token18's and the
trailer's own value distributions (no live join for either, per DIV-968/969),
and round-trips the whole 400-byte trailer through `File.Marshal` on every
openable file. Two new release witnesses
(`originaltrailer1138_release_test.go`,
`originalunitresidual1138_release_test.go`) prove the same decode against one
real, install-derived save through the ordinary App LOAD path, not only
synthetic fixtures — see "Proof" for their exact assertions.

## Divergence rows

- **DIV-968** (new, OPEN): world trailer block, dwords 0/1 named, 2..99
  carried opaque. Naming the two debug toggles answers "is this free padding"
  (no) but not "should this engine reproduce a developer debug-toggle
  console," which no claim traces to a player-visible effect; the other 98
  dwords have no located consumer at all (`SAV-791`).
- **DIV-969** (new, OPEN): Unit/Token `+0x18` recipient mask decoded, not
  wired. `SAV-678` names a mechanism this engine has no equivalent system
  for, and `SAV-797` does not establish which of the two corpus values a
  given record's own history requires; wiring a live mask with no located
  consumer or owner-specified visibility model would invent behaviour no
  claim states.
- **DIV-880** (updated, remains OPEN): the row recording that a native
  (non-imported) city party's own Weight/Load stay zero unconditionally is
  sharpened, not resolved. `SAV-792` now finds `+0x8e` is normally an
  incrementally maintained accumulator, not merely a zero default, which
  makes the row's "0 is the correct value" justification look increasingly
  questionable — but verifying or fixing the native-city writer is outside
  this story's "no writer" scope, so the row's own revisit condition is not
  met and it stays OPEN.

DIV-970 through DIV-973 (reserved for this story) are unused; nothing this
story found needed a fourth or fifth row.

## Proof

- `gofmt -l .`: clean.
- `go build ./...`: clean.
- `go test -trimpath -count=1 ./...` (whole repository, asset-free): clean,
  49 packages, 0 FAIL.
- `go vet ./...` and `go vet -tags sessioncorpusaudit ./...`: clean (the same
  pre-existing `BookSpell` unkeyed-field warnings story 1137 recorded, in
  files this story does not touch, are unrelated and unchanged).
- `internal/gatedtests` `TestScanMatchesTheCheckedInPopulationList`: clean,
  207 registered names (`population.txt` is 211 lines, 4 comments) — 205
  before this story, 2 added by the two new release witnesses, 0 removed.
- `scripts/check-no-game-assets.sh`: clean.
- `TestUnitResidualCorpusAudit1138` (`-tags sessioncorpusaudit`, fresh run,
  both lawful roots): `audited 72 file(s): 51 world-half, 20 between-mission,
  1 unreadable`; `1977 actor(s) cross-checked: 0 OwnWeight/Load mismatches, 0
  sight mismatches`; `Token18 corpus distribution: map[2:2044]`; `Trailer
  TurnTracing distribution: map[0:71]; ScriptTracing distribution:
  map[0:71]`; 0 trailer Marshal round-trip mismatches — identical on EN and
  RU.
- `TestReleaseOriginalTrailerRoundTripsOnLoad1138` (fresh run, both roots):
  `game0021.sav`'s trailer decodes to `TurnTracing=0 ScriptTracing=0`,
  round-trips byte-identical through `File.Marshal`, and the file still
  reaches the map through the ordinary App LOAD path.
- `TestReleaseOriginalUnitResidualFieldsRestoreOnLoad1138` (fresh run, both
  roots): four map units on `game0021.sav` (MapUnitID 21/44/48 `Human`, 25
  `Unit`) — file-side OwnWeight/Load 0/0, 3/3, 106/106, 0/0, sight 5/6/6/8
  whole cells, Token18 2 on all four — match live `sim.Entity`
  `ActorLoad.OwnWeight`/`.Load`/`.ScanRange` after LOAD, on both roots.
- `pipeline/check-release-tests.sh` (seat script, `AGAINROM_IMPL` pointed at
  this worktree, both lawful roots in one invocation): 8 packages, 207 gated
  tests, 2 roots; **207 of 207 ran and 0 lacked a subject, on EN and again on
  RU.**
- Hashed-state instrument: `TrailerBody` lives only in `pkg/formats/sav`
  (both decode paths) and `Token18` lives only in
  `pkg/formats/sav.ActorBasis`, deliberately not threaded into
  `pkg/sim.SourceActor` — neither enters `pkg/sim.World.encode()`.
  `TestHashIsPinned`, `TestThePinnedDigestIsFNV1aOfThePinnedBytes` and
  `TestHashIsFNV1aOverExactlyTheByteForm` (`pkg/sim/hash_test.go`) all pass
  unchanged against the same literal pinned digest, `0xf8eeb7a2b2bd91df`
  ("Form84") — this story moves it in no direction and updates no pin.
- `pipeline/check-milestone.sh`: not run, on story 1136/1137's own precedent
  for the same population — this story's change is reached only through an
  original-save resume. `go build ./cmd/missionrun` plus a `-trace -ticks 1`
  UNSUPPORTED-node count on mission 10 and mission 20: **0 on both missions,
  both roots** (EN: m10 0, m20 0; RU: m10 0, m20 0) — unchanged from master's
  own recorded census.

## Open debt

**The trailer's two named dwords have no live console or toggle to attach
to** (DIV-968). This engine has no debug-command route or turn/script-tracing
output; "correct" live behaviour for either dword is not established even
though both are now named.

**No preserved save — research's or this project's own local corpus —
carries a nonzero trailer.** `SAV-649` finds the 400-byte tail zero in all 45
parsed research streams; this story's own fresh corpus audit finds the same
across all 72 locally preserved saves. The nonzero case (`TestTrailerDwordsAreNamedAndResidualCarried`,
`TestTrailerResidualIsByteIdenticalOnReExport`) is proven only by a synthetic
fixture, never by an authentic sample.

**Token18 has a real ROM1 mechanism this engine has no equivalent for**
(DIV-969). `SAV-678` names a per-recipient publication bit; building a live
visibility or notification system to receive it would be new gameplay
behaviour no claim specifies, well outside a decode story's own scope.

**DIV-880 is sharpened, not resolved.** `SAV-792` makes "0 is the correct
value for a native city party's own weight/load" a live open question rather
than a settled default, but resolving it means auditing and possibly changing
the native-city writer, which this story's "no writer" scope excludes.

**Native mission SAVE still has no interactive path.** All four fields and
the trailer are exercised only by this story's own corpus audit and release
witnesses; the owner's own standing rule against inventing a production
mission SAVE entry point still applies.

## Touched surfaces

- `pkg/formats/sav/archive_document.go`: `TrailerBody` type, conversion
  helpers, `readTrailerBody`, both decode-path call sites.
- `pkg/formats/sav/archive_document_test.go`: repointed at
  `trailer.Residual[97]`.
- `pkg/formats/sav/document_data.go`: `Trailer` field comment; both
  conversion call sites.
- `pkg/formats/sav/document.go`, `pkg/formats/sav/container.go`: narrow
  reader's own `trailer`/`File.Trailer`.
- `pkg/formats/sav/program.go`: `StatOwnWeight`/`StatLoad` rename and
  citation comment.
- `pkg/formats/sav/actorbasis.go`: `Token18` field; `Sight` citation comment.
- `pkg/formats/sav/holdings.go`: `ActorLoadState` citation comment.
- `pkg/formats/sav/sav_test.go`, `pkg/formats/sav/program_test.go`,
  `pkg/formats/sav/actorbasis1138_test.go`: new/updated fixtures and tests.
- `pkg/sim/actorload.go`: citation comments only.
- `pkg/game/originalholdings.go`, `pkg/game/originalactorregistry.go`:
  citation comments only.
- `pkg/game/nativecity.go`: sharpened DIV-880 comment.
- `pkg/game/originalhuman.go`, `cmd/savtool/party.go`: mechanical rename.
- `pkg/game/actorload1138_test.go`,
  `pkg/game/originalunitresidual1138_corpus_test.go`,
  `pkg/game/originaltrailer1138_release_test.go`,
  `pkg/game/originalunitresidual1138_release_test.go`: new tests.
- `internal/gatedtests/testdata/population.txt`: two new release-witness
  entries.
- `docs/DIVERGENCES.md`: DIV-968, DIV-969 (both new, both OPEN); DIV-880
  (updated, remains OPEN).
- `knowledge` (gitlink): k2 → k3.
