# The original-SAV byte-identity census

## Intent

Settle, by measurement, what "we read the original SAV completely" is worth
at the document layer. The scope is exact and narrower than that sentence
sounds: this instrument measures `sav.DecodeDocumentData` and
`sav.EncodeDocumentData`, the DTO the reader produces and the container the
writer emits from it. It says nothing about whether the running game reads a
save completely, which goes through `ResumeOriginalSave` and `Snapshot` and
has its own instruments.

`pipeline/SAV-COMPLETION.md`'s milestone 2 compares thirteen structure rows
raw-file-to-live over the discovered corpus and states in as many words that a
green run "does not prove that every required field or live consumer has an
oracle". Nothing in the record performed the one test that would settle it, and
the two instruments that look like they do, do not:

- `pkg/game/savroundtrip1195_corpus_test.go` builds `want` from
  `before.Snapshot` after restoring the original file and `got` from
  `after.Snapshot` after restoring our own re-export. Both sides pass through
  our reader, so a field the reader silently drops is absent from both and its
  census reports zero mismatches. Its 94-of-102 with zero mismatches is a real
  regression bound and not a completeness claim.
- `pkg/formats/sav/save_document_corpus_test.go` (build tag `savdocumentaudit`)
  parses, serializes, re-parses and re-serializes, and asserts the SECOND
  encoding equals the first. It calls `clear(raw)` immediately after parsing, so
  the comparison that would settle completeness — our bytes against the bytes we
  read — is not merely absent, it is made impossible. That test measures
  stability.

This story adds the instrument that measures identity at that layer. It changes
no production behaviour.

## As-built behaviour

`pkg/game/savbyteidentity1197_corpus_test.go`, behind the `sessioncorpusaudit`
build tag, one test: `TestSAVByteIdentity1197OriginalCorpus`. It reuses
`milestone2Corpus` (`milestone2_acceptance_reader_test.go`) so it reads the one
discovered population the milestone-2 family already reads, in `t.Run` subtests,
with no file list of its own.

Per file: decode to `sav.DecodeDocumentData` — the intermediate whose DTO owns
no `File.Body`, `Store`, `TailRest`, offset or compressed input
(`pkg/formats/sav/save_document.go`) — then emit a whole container back out with
`sav.EncodeDocumentData` and compare the produced bytes against the input.
**No live world is involved on either side.** Restoring into a `Mission` and
re-deriving a file from game state would launder the bytes through exactly the
reader whose completeness is in question.

Every differing byte range is reported with its offset, its length, the decoded
structure covering it, and one of four classes: `intended`,
`modelled-reemitted`, `unmodelled`, `unknown`. An `unknown` range is printed in
full with its offset and fails its file's subtest. It is never explained away.

The state store is compared twice: positionally, byte against byte, and
structurally through `pkg/formats/reg` — the separate `.reg` parser, which
shares the store's `0x31415926` signature and none of `pkg/formats/sav`'s own
`parseStateStore` code. Every `modelled-reemitted` store rule requires that
independent digest to be equal on both sides, so a store value that actually
went missing breaks the digest, disqualifies those rules and lands in `unknown`.
A new loss cannot be absorbed by an existing rule.

`scripts/check-milestone2-acceptance.sh` selects the test by name and
whitelists its `SAV-BYTEID-` lines, so the census prints on a passing run.

## The result

Measured on both lawful roots through one `check-milestone2-acceptance.sh`
invocation. 103 files discovered; 1 unreadable and skipped by `milestone2Corpus`
itself (its magic is `Bsg&`); 102 attempted; 0 refused at decode or encode.

| number | value |
|---|---|
| campaign document collection equal by value | 102 of 102 |
| campaign document run equal by byte, at its own offset | 95 of 95 located (7 empty collections not locatable) |
| campaign record trailer equal by byte | 95 of 95 located |
| campaign position re-emitted identically | 102 of 102 |
| whole campaign record region re-emitted identically | 102 of 102 |
| whole file re-emitted byte-identically | 12 of 102 |
| whole file differs | 90 |
| differing ranges, unclassified | **0** |

**The 12 that come back byte-identical are all files this project's own writer
produced.** Twelve paths, six distinct files by SHA-256: five appear once each
and the sixth appears at seven paths. Provenance is settled by hash, not by
directory name, and it comes from two records. The corpus carries six
`MANIFEST.md` files; EXP-0261's covers five of the twelve paths, recording each
as a "generated candidate" and every original-written file beside them as
differing. The file at seven paths is story1073's own production-route witness,
recorded with its SHA-256 in `pipeline/LOG.md`. **No save the original game
wrote re-emits byte-identically.**

## What every difference is

9226 ranges across the 90 differing files, in three groups.

**`intended`, 218 ranges.** Differences this project's code makes on purpose and
names where it makes them. The container blob extent is recomputed for the
re-emitted blob (96 ranges). The archive alignment byte is written as zero
rather than carried (48 ranges, `SAV-DECPAD-238`); the rule proves the position
really is the pad — one byte past the 400-byte trailer and the last byte of an
equal-length body — before accepting it. The label region is rebuilt from the
NUL-terminated name alone, dropping the debris the original leaves behind a
short label (74 ranges, 654 bytes, `SAV-LABELTAIL-236`).

**`modelled-reemitted`, 8794 ranges, all in the embedded state store.** The
record table is rewritten in the writer's canonical directory order (3864
ranges), directory child ranges follow it (704), pool locators follow the
rebuilt pool (19), and the value pool itself is rebuilt in that same order
(4207). Every decoded value is preserved, and that preservation is proven rather
than assumed: the independent `pkg/formats/reg` digest of both sides is equal on
all 90 files. The proof covers exactly what both readers read. It does not cover
the post-NUL tail of a node-name field, because `reg.Parse` cuts the name at the
first NUL exactly as `parseStateStore` does, so the digest is equal whether that
tail survived or not — see the next paragraph for how much of it this class
absorbs.

**`unmodelled`, 214 ranges, 766 bytes — and the larger figure the re-emission
actually drops.** One class, in one place: debris after the NUL inside the state
store's 16-byte node-name fields. The intermediate does not carry it and our
writer zero-fills all sixteen bytes before copying the name in, so none of it
survives.

The corpus holds **4379** such debris bytes in 920 maximal runs across 2656
framed records; **3848** of them are gone from our re-emission. The census
labels **766** of those `unmodelled` and the record-reorder rule absorbs the
remaining **3082** as `modelled-reemitted`, because `sav1197Inspect` tests
`idx >= 0 && !sameSlot` before the node-name rule and our canonical ordering
puts a different node at most indices. The 531-byte remainder between 4379 and
3848 is debris whose offset is now occupied by another node's own name text, not
debris that lived. Measured by an instrument independent of the census: a walk
of every `&YA1` frame's 32-byte records over all 102 readable files.

So `unmodelled` is the count at a stable slot, not the count this re-emission
drops. Both numbers are stated here because the second is the honest one.

Nothing player-visible rests on either. Both readers stop at the NUL, so no
decoded value depends on these bytes and none reaches hashed state. No claim
establishes whether the original's own store writer clears that tail.

**`unknown`, 0 ranges.** This is the number the story exists to produce.

The compressed blob is not diffed byte against byte — one changed body byte
moves every run boundary after it — but the two informative cases are separated.
On all 42 of the 90 differing files whose decompressed bodies
matched, the transport codec re-compressed bit-identically; the codec never
produced a differing blob from an unchanged body. Over the whole attempted
population of 102 the matching-body count is 54, the extra 12 being the files
that re-emit byte-identically.

## The campaign record

Owner direction put two things first: the Valuable Documents journal and the
campaign position — which mission the game is on now. Both survive completely,
and they survive by byte as well as by value.

### Valuable Documents, by value and by byte

The two results are reported separately because they answer different questions,
and **they agree**.

| line | numbers |
|---|---|
| `SAV-BYTEID-1197-DOCUMENT-VALUES` | attempted 102, identical 102, differing 0 |
| `SAV-BYTEID-1197-DOCUMENT-BYTES` | located 95, byte-identical 95, differing 0, unlocatable-empty-collection 7 |

The value result compares both sides' decoded `Documents` element by element:
`Value`, `Kind`, count and order. A pre-existing seat hotfix (`1e1df55`) added
the same comparison to the round-trip census, where both sides pass through this
project's reader. Here the `want` side is the original file's own bytes, so the
result is not laundered.

The byte result is the stronger one. It answers whether the re-emission puts the
same records at the same offsets in the same order with the same padding, rather
than reconstructing an equal-valued list somewhere. The census builds the record
run the reader decoded (`u32(count)` followed by that many 8-byte `Value`/`Kind`
pairs), requires it to
occur exactly once in the source campaign record, and then compares our own
emission at that same offset. Same offset, same order, same padding, on every
file where the run can be located.

The 7 unlocated files all carry a zero-document collection. A count of zero
encodes as four zero bytes, which are not unique inside the record, so no byte
claim is made for them. Their whole campaign record is byte-identical anyway.

### Campaign position

`SAV-BYTEID-1197-POSITION`: attempted 102, identical 102, differing 0. The
compared fields are `Main.Mission`, `Main.Announced`, `SelectedMission`,
`AutoGetMission`, `LastMission`, `MissionTime`, `FirstMapPoint` and every child
mission.

### The 32-byte trailer

`pkg/game/save.go` records the eight dwords between the document run and EOF as
unattributed. They are not unclassified here.

`SAV-BYTEID-1197-CAMPAIGN-TRAILER`: located 95, byte-identical 95, differing 0,
first-dword-is-`SelectedMission` 95 of 95. The census locates the span as the 32
bytes immediately after the document run, requiring it to end exactly at the end
of the campaign record, so the position is proven per file rather than assumed.
Its first dword equalling the decoded `SelectedMission` on all 95 is not
independent confirmation of anything about ROM1, and this census does not offer
it as one. The span is located at the position our own parse read the document
run, and `parseCampaignProjection` reads `SelectedMission` at that same cursor,
so on every file where the run locates uniquely the equality is forced. What the
number actually gates is a field mapping in our own reader: a projection that
maps the wrong file scalar into `SelectedMission` drops it to 0 of 95 and
reddens the gate while no byte moves. That is worth gating, and it is what this
line means.

The ROM1 statement belongs to research and already exists. `MISSION-DOC-021`
records that the 32-byte trailer's first dword is 10, 20 or 30 across the
corpus, the campaign's mission number — High for the record's field set and
8-byte stride, Medium for the save location, with the trailer called
unattributed. `pkg/game/save.go` cites it, with `REG-SCN-097`, for the
document-availability guard. This census adds nothing to that.

This build's own `DocumentMission` guard has no counterpart in the original's
record, so it is deliberately not compared by value. It produced no byte
difference here: the trailer is byte-identical on all 95 located files.

### The whole region

`SAV-BYTEID-1197-CAMPAIGN-BYTES`: the campaign record region (`File.TailRest`)
re-emits byte-identically on all 102 attempted files, so every record inside it
does, including the two located spans above and the 7 empty document
collections.

## The completed-mission set

Measured because the brief asked for it, and reported without a divergence row
because owner direction ruled it unimportant and said not to open one on it
alone.

It is lost on read and on write, for one reason: `sav.CampaignProjection` models
no completed-mission set, so neither side of the game has anywhere to put it.

- Read. Every between-mission save in the corpus restores with `Snapshot.Won`
  empty, including files whose campaign main mission is 30 or 40 and whose
  earlier main missions are therefore complete.
  `newTownFromCampaignProgress` (`pkg/game/town.go`) populates `open`, the
  mercenary arrays, `available`, the documents and `docMission`, and never
  `t.won`.
- Write. `campaignProgress.projection()` (`pkg/game/campaignprogress.go`) emits
  no such set, and no production path in `pkg/game` writes `Town.won` into a
  SAV; the only `Won` a SAV carries is the world session's script win counter,
  which is a different quantity.

Whether ROM1's own campaign record holds an explicit completed set is Unknown
here. `cityCampaign`'s decoded `dwords` and `scalars` still hold elements with no
established meaning, and this is a clean room: that is a research question, not
something to infer from our own code.

## Bounds

- This census measures the **re-emit** path: file to intermediate to file. It
  does not enumerate the losses seen on the original-game round trip recorded in
  `pipeline/SAV-COMPLETION.md` (export, play in ROM1, resave, reopen). That path
  runs through live game state and a different set of producers.
- Byte attribution inside the archive body takes the greatest located landmark
  at or before an offset, so a byte inside a nested object's fields names that
  object rather than its owner.
- The baseline this test commits gates six numbers: the byte-identical count as
  a floor, and refusals, campaign-record, campaign-document, campaign-position
  and unclassified-range counts as ceilings of zero. Per-rule range and byte
  counts are logged and not gated, because a larger corpus legitimately raises
  them. A genuinely new original save carrying a difference no rule explains
  reddens this gate, which is the intended direction of the error.

## Open debt

- `DIV-1329` records the whole-file divergence: a file this engine writes back is
  not the file the original wrote, even when nothing in the game changed.
- Story1196 owns the completed-mission set. The finding above is the evidence it
  needs so it does not have to guess: the field is missing from
  `sav.CampaignProjection`, not dropped by a writer.
- Whether the original's `&YA1` writer clears its node-name field past the NUL,
  and whether its record order is stable, are the two facts that would let a
  byte-preserving store writer be built or ruled out.
- The baseline floors the byte-identical count but not `attempted`, so a change
  that made original saves unreadable would shrink the population without
  reddening this gate on its own. `TestSAVRoundTrip1195` floors its own count in
  the same invocation, so the family is covered; this instrument is not, by
  itself.
- The census measures the `DecodeDocumentData`/`EncodeDocumentData` DTO layer.
  It is not a statement about whether the running game reads an original save
  completely, which goes through `ResumeOriginalSave` and `Snapshot`.

## Gates

Run on this branch's tip.

- `gofmt -l` is clean. An earlier revision of this section recorded
  `pkg/game/installtext.go` as a pre-existing unformatted file; hotfix
  `37690e3` fixed it on `main` and nothing is unformatted on this tree.
- `go test -trimpath -count=1 ./...` untagged: exit 0, no FAIL.
- `go build -tags sessioncorpusaudit ./...`: exit 0.
- `scripts/check-no-game-assets.sh`: `clean (tree scan)`, exit 0.
- `scripts/check-milestone2-acceptance.sh` on both absolute lawful roots:
  exit 0, and every `SAV-BYTEID-1197-` line above appears in its output on
  both. `AGAINROM_AGS_CORPUS` must name the owner's `.ags` directory, since a
  worktree has no `saves/` of its own.
- `pipeline/check-div-claims.sh` against this checkout: exit 0.
- `pipeline/check-preserved-installs.sh`: 554 files, every root as recorded.

The milestone script-gap census is unchanged, as a test-only change must leave
it: `missionrun -mission 10 -trace -ticks 1` and the same for mission 20 both
report `UNSUPPORTED` 0 times on the EN root.

## Read-only discipline

`gameversions/` was read only. No file in it was written, renamed or deleted;
`check-preserved-installs.sh` verifies that after the run. No game asset or
install byte is in this commit, and no owner save name appears in any tracked
file: every committed string in the instrument is a region, structure or rule
name, and the corpus-relative path reaches `t.Logf` only.
