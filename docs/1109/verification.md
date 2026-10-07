# Item weight verification

## Status

This candidate merges landed master (story 1107's saved non-party current
profile and 70 further commits under it) into the composed weight work. The
research gitlink is 682184eb, forward of the prior 1172d41a pin. No
unpromoted experiment, original process or install write was used. This is
the candidate the seat's one adversarial review pass attacks; it is not yet
reviewed.

## Contract witnesses

FR-1 and DD-1: Piece reads signed F4A; shared originalItemInstance imports it.
The natural game0002 SAV has Witch mapID21, Potion code0e06, count3, weight1.
The source SHA is b1ce079cc2b3f1bc101862c1dcf2afd8237421458b3e2474c4447efe5df2e761.
Ground fixtures independently write signed F4A -9 and explicit zero. The world
and city consumers use instance values, with unchanged absent-value fallback.
ItemInstance.Weight is int16 (ITEM-STACK-003's item+0x4a is s16); WeightPresent
distinguishes explicit zero from code-table fallback.

FR-2 and DD-3: unit tests exercise split transfer, equip, drop, pickup, death,
reserved-scroll native reload/cancellation, shop conversions and city load.
Zero, negative and equal-code distinct weights survive. Signed multiplication
wraps at int32; signed halving truncates toward zero. DIV-754 records the
state-retention difference from ROM1 stack equality.

FR-3 and DD-2: pkg/sim/itemweightbinary.go implements the form74 suffix: a
4-byte trailing length, then zero or more 6-byte records (uint32 ordinal,
int16 weight) in strictly increasing ordinal order, covering sacks, carried
stacks, equipment slots and reserved scrolls in that fixed traversal order.
Malformed spans, out-of-range/duplicate ordinals, an ordinal naming an empty
item, and unconsumed records all refuse atomically (decodeInstanceWeights).
Native party input refuses absent-weight residue via itemWeightsFault.
Canonical stack validation (foldContainer) happens after weight assignment.

The genuine form73 fixture was generated before edits by exact d71426b6 using
review/story1109/form73-producer.go outside Git. Its 5390 bytes hash to
bcdcd9a6d962d8e0233e48100ffb63d4a4577a155e3a984de740633d9713fc0b.
It contains nonempty sack, carried and worn instances, a book, dead provenance
and source-current profile. Its genuine AGS is 12711 bytes, SHA
3d0f9b5e89cfd486b71cae099023490e7ef275ed2666cf790fbe4ea2a9736d1b.

The seat separately froze a real source-city-v3 AGS from exact d71426b6:
16191 bytes, SHA 60082e1537dad3ca76c4e84531c5380db71d4323ed3803dbb10414cf83819b76,
at review/story1109/seat-city-v3/en/trained.ags, with trained.sav (3223 bytes,
SHA 2de0eeac89790d14ba0fea9685c7ececcf220fd1b8bd9667aed5a14c1d25baf7). Neither
file was regenerated on this candidate. TestInstanceWeightFrozenCityV3 reads
both through AGAINROM_WEIGHT_OLD_CITY; production App LOAD retains its party
and source baseline exactly, SAV export matches the original 3223-byte output,
and a new source LOAD imports weights. This envelope has no world, so it is
not an embedded form73-world witness. Without this env var pointing at that
directory, the subject is skipped and is not claimed as witnessed; the paired
release gate below supplied it and the subject ran on both roots.

## Observable result

TestReleaseOriginalItemWeightTransferAndNativeReload (production App LOAD,
not a GUI observation): the natural game0002 SAV's Witch carries one Potion
stack at F4A weight1, count3. A fresh App LOAD imports it; the receiving
member's load reads 53 before transfer. Moving 2 of the 3 units raises it to
54, matching the independent source sum 39+(29+2)/2. Ordinary menu SAVE then
a fresh, separate-process-style App LOAD retain both the transfer and load54;
the reloaded world hash matches the pre-save hash exactly. Measured fresh on
this candidate: both EN and RU roots produce the identical world hash
e28fda5b462e9450 after that SAVE/LOAD round trip. The saved Witch's own load
field is 0 while its item-derived load is 1; this story does not claim exact
saved current-load restoration (DIV-755).

The world hash differs from earlier pre-merge checkpoints of this same test,
because landed master (structure health, cell-trigger overlays, non-party
current profiles and pools, spellbooks, late corpses) now restores more state
into the same fixture's App LOAD than it did when those checkpoints were
taken. That is master's own restored state reaching this fixture, not a
change this story makes to it; DIV-754/DIV-755 are the only rows this story
opens, and neither concerns those fields.

## Known debt

DIV-754 (item identity / distinct stored weights, DEVIATION, OPEN) and DIV-755
(saved container and actor load state, FIDELITY-DEBT, OPEN) are the two rows
this story spends; DIV-756 through DIV-761 were reserved and are unused.
Stored container load, presence, order and insertion index remain required
follow-up (DIV-755). Recomputed item-weight sums do not preserve stale saved
actor load: the seat's bounded source census found 12 such differences among
346 living Humans. Full Token, derived item fields and source object
lifecycle remain required follow-up (DIV-746, pre-existing). This story does
not invent their producers or default values, and no original process runs.

## Gate record

Measured on the merge commit composing landed master 6c47ec6a into this
branch; artifacts are under the seat's untracked review/story1109 directory.
No game asset, lawful SAV or generated executable enters the repository.

- `gofmt -l .`: clean.
- `go test -trimpath -count=1 ./...`: PASS, 49 of 49 tested packages, 0
  failures (25 further packages have no test files). GOFLAGS=-buildvcs=false
  was used only because this worktree's `.git` is owned by another Windows
  account, which fails Go's VCS stamp; no other failure was masked by it.
  Recorded in reconciled-go-all.txt.
- `scripts/check-no-game-assets.sh`: clean tree scan.
- `scripts/check-claim-citations.sh`: ok, 1552 distinct citations resolve
  against 1890 claims and 298 experiments under 1008 prefixes. Every claim
  story 1109 itself cites (ITEM-STACK-003, ITEM-LOAD-005, ITEM-SAVE-014, and
  DIV-746's SAV-TOKEN-034/ITEM-EFFSAVE-077/ITEM-DEATH-012/ITEM-CORPSE-034) was
  read individually at the new 682184eb pin: all are active or active
  (amended); none of their retracted clauses is one this story's own text
  relies on. No citation needed a wording change.
- `check-div-claims.sh`: 335 live rows of 335, citing 485 distinct claim ids,
  exit0. This is master's 333 rows plus this story's DIV-754/DIV-755; the id
  count is unchanged because both new rows cite ids master's own rows already
  cite. Recorded in reconciled-div-claims.txt.
- `check-release-tests.sh` against both EN and RU roots in one invocation:
  ok, 159 of 159 gated tests ran on each root, 0 lacking a subject. The
  reference count at master alone is 157; the two extra are this story's own
  TestReleaseOriginalItemWeightTransferAndNativeReload and
  TestInstanceWeightFrozenCityV3. AGAINROM_WEIGHT_OLD_CITY was supplied so the
  latter ran rather than skipped. Recorded in reconciled-release-en-ru.txt.
- `check-scenarios.sh` against EN: ok, 50 of 50, unchanged from master (this
  story adds no scenario file). Recorded in reconciled-scenarios-en.txt.
- `check-preserved-installs.sh`: ok, 181 files, both roots as recorded.
  Recorded in reconciled-preserved-installs.txt.
- Mission 10 and 20, EN, trace/ticks1: measured directly on this candidate's
  code, built with go build -o mr ./cmd/missionrun: 0 and 0 UNSUPPORTED
  script nodes. pipeline/milestone-baseline.txt records 0 "cannot run"
  lines for every one of the 28 shipped campaign maps on both roots
  already, so this story's own two named missions are unchanged against
  that baseline, not merely assumed unchanged. This story does not touch
  script opcodes; the seat's own milestone gate over all 28 maps and both
  roots remains the authority for the full census, not this file.
- Changed Go files are formatted; `git diff --check` passes. No commit
  trailer or model/provider attribution was added.

## Native save bytes and simulation hashes

This story's own change moves pkg/sim's formatVersion from 72 to 74 (73 for
CurrentProfileBasis, landed independently by story 1107; 74 for this story's
own sparse weight suffix) and grows the Snapshot gob envelope's type
descriptor, because sim.ItemInstance gains Weight/WeightPresent and
PartyMember.WornItems/CarriedItems reach that type — every native SAVE now
carries more bytes than before this story, by design (FR-1..FR-3). Old forms
below 74 read back with WeightPresent false and Weight 0 (DD-1's absent-value
fallback); TestReleasedEnvelopeAtTheCurrentSimulationFormFixture pins the
exact current-descriptor envelope hash and every older decode-only form
alongside it. Merging origin/master moved that pin's two "current descriptor"
SHA256 values (current72 and currentReleasedSaveFixtureSHA256): both changed
because the outer Snapshot gob type graph grew, not because any lower, already
-shipped form's bytes changed. Every one of the test's other assertions —
the old72/old71/form69 envelope hashes, the decode-equality checks, the
migration from form69, the v66 fnv probe, and every fnv over a peeled World
at forms 69/70/71/72 — is byte-for-byte unchanged from master alone, confirmed
by an isolated comparison run against origin/master in a detached worktree. The
final w.Hash() pin is not in that set and did move, from 0xb0f10eac24ea90a9 to
0x8c8b3139c1bebab0. That is required, not a regression: Hash() is FNV-1a over
MarshalBinary(), so the form byte is inside the hashed input by construction,
and story1107's landing moved the same pin the same way one form earlier. That is direct evidence the lower-form
World encoding this story's peeling helpers reconstruct is unaffected; only
the envelope-level descriptor for the current form moved, and it moved for a
documented, mechanical reason (gob's type graph, not a behavior change).
