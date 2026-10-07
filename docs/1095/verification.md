# City conversion proof

## Candidate scope

Assigned base: `d7606aa1dde6fc593161bb748aa66c23d573ea52`.
Reconciled master: `70403c7d71feb2cf3f4ea2945fcda523033624f9`.
Unchanged research pin: `ba21c9aa9a949023b3d678b22ca29b3a3b0cd95f`.
The story adds one command, `saveconvert`; the reconciled tree has 43 commands.
The seat owns publication into `builds/current/` and the single fresh-context
review. The reviewed candidate was `9b1017fa98c70bf17f631d2366bbf209f482bf6c`;
the three reproduced findings received one correction pass below.

## Observable result

Before this slice, native Snapshot dropped the original-city provenance and a
fresh AGS process could not author SAV. The built `saveconvert` now ran twice
against the lawful EN root, producing `review/story1095/city.ags` (13030 bytes)
and then `review/story1095/city.sav` (3215 bytes) outside the installs. Input:
`gameversions/saves/2026-08-15/game0010.sav`, SHA-256
`89cfca4c14e2b0bafd1fe28911badf246e213d83398874699947008739e8c5d4`.
Output SHA-256:
`af8f9a5eab8481a9dd24ff4e352685e3f78a9aff13b025ae3955ee726cf5db24`.
The source label `city` is retained.

The registered release test
`cmd/saveconvert.TestReleaseImportedCityConversionAcrossFreshProcesses` passed
on EN. It runs the actual command entry point in two child processes, removes
only its copied source before the second process, verifies 25 objects and two
characters, compares the whole semantic model after independent identity-key
normalization, loads the AGS through production SaveSeams and saves SAV, and
requires changed-Human export to fail with no output. This is not an original
ROM1 process witness. The seat must run the registered test on EN and RU through
its paired release invocation; this lane does not claim the RU result.

The previous ordinary-SAVE candidate SHA-256
`bbee204a94f45e6463ee107f2eb23aaef7059b835c45d6e5338a8090330f781f`
also has 3215 bytes. Direct comparison finds exactly 15 changed bytes at
offsets 1932 through 1946, entirely inside the fixed label region. Its label is
`town - gold 683`; the new label is `city`. Every header, compressed document,
state-store and campaign byte outside that label is identical. Relative to the
original source, identity keys are reminted, the encoder-added document pad is
zero instead of `d8`, and physical codec/label/store framing is rebuilt. The
normalized semantic model has no remaining differences.

## Independent reader

Pinned `savdoc -mode postload` exits zero and reports `document_ok=true` for
source, prior candidate and converted output, each with a 5592-byte decoded
document. Receipt: `review/story1095-savdoc-postload.tsv`.

Pinned `savfull` writes exact-reader rows for all three city inputs: 25 archive
objects, 22 state records, zero physical gaps and zero physical overlaps.
Receipt: `review/story1095-reader-complete/path-results.tsv` and its interval
coverage. Its later mutation self-check exits one with
`world-nonzero-discriminator did not produce its preregistered discriminator`,
including when given an exact world control. This is not a complete savfull
PASS. The older `savauthor` comparison mode refuses the existing source Weapon
at reference74; it contributes no comparison claim. Neither tool was patched.

The seat subsequently ran unmodified research master
`f03d58475878c9bafa514bd6fd788c63ae6e46b7` with the world control sorted first.
[Its receipt](../../../review/story1095/reader-controls.md) records four of four
exact documents (source/prior/converted city and world control) and 20/20
adversarial discriminators passing. Mutations apply only to the first exact
document, the world control, not the cities. Each city retains the same six
unresolved nonzero pointer-map references; exact coverage does not mean every
pointer resolves. This supplements, and does not rewrite, the earlier failed
invocations. The implementation research pin stays unchanged.

## Initial candidate gates

Focused DTO, SAVE, restore, conversion, malformed-model, alias and allocation
tests pass. They cover foreign/repeated bindings, missing class bodies, invalid
member lengths, absent-container fields, count/depth/byte limits, native fallback
on Human/item/roster/session mutation, old AGS, new-session clearing, label
overrides and no replacement. The current native fixture digest changed only
for the additive gob type descriptor; older decode fixtures and the canonical
World digest remain unchanged. The first full run caught that expected stale
digest; after updating the encode-side pin, the complete
`go test -trimpath -count=1 ./...` chain passed. `gofmt` and `git diff --check`
are clean. Full log: `review/story1095-final-go-corrected.txt`.
`scripts/check-no-game-assets.sh` prints `clean (tree scan)`.

Both allocation sweeps report `missing answers: 0`. `DIV-642` uses the assigned
range. `check-div-claims.sh` exits zero; its 69 retraction-bearing existing rows
do not include DIV-642. The three newly cited claims were read at the pin.
`check-preserved-installs.sh` reports 181 files unchanged across both roots.

Built missionrun with the EN root, `-trace -ticks 1`: mission 10 reports zero
UNSUPPORTED nodes and mission 20 reports zero. The corresponding master
`pipeline/milestone-baseline.txt` entries contain no unsupported nodes (16/27/12
and 14/15/11 check/instant/trigger counts). Both unsupported counts remain zero;
this slice changes conversion output, not that script population.

No GUI input, original process, install writes, research pin changes, review
commission, or broad native/world SAV export was performed.

## One correction pass

- Install fence: both `.ags` and `.sav` use the configured asset root plus the
  archive-ancestor and physical alias checks. CLI preflight runs before input
  reads or asset loading; writer preflight runs before directory creation;
  resolved-directory validation repeats before publication. Tests use fresh
  synthetic archive markers, empty/missing configured roots, root/target aliases,
  an empty asset argument and a directory becoming an install before publication.
  No install-target or temporary output survives refusal.
- Baseline trust: native LOAD rebuilds characters from the semantic graph,
  applies the shared ordinary party importer with the selected table/body
  context, and compares every baseline member before publishing the candidate.
  Session baselines are checked against source/import rules too. Encode/decode
  validate structure without install context; they do not grant export
  eligibility. Body, name, skill, XP, inventory/item, spell membership, identity,
  profile and saved-pool forgeries refuse LOAD/conversion transactionally.
  Current-party-only edits still load as native progress and refuse SAV export.
  Source loadout alias/stack expansion is bounded before materialization.
- Determinism: city DTO version 2 emits strictly path-sorted ordinary records.
  Version-1 maps remain decode-compatible and normalize on encode/decode; no
  opaque custom gob payload bypasses allocation preflight. Thirty-two iterations
  per legacy/current representation compare same-Snapshot AGS bytes, decoded
  AGS re-encoding and rebuilt SAV bytes. Duplicate/unordered records and record
  allocation counts are rejected. The additive descriptor changes only the
  current encode fixture SHA to `f6c644294e1cf01d2779fca9969bad2428eacefc72fd9c12c6bad90f0c507795`;
  older fixtures and canonical simulation form remain unchanged.

Focused correction checks pass; logs are
`review/story1095-correction-focused-final.txt` and
`review/story1095-correction-focused-latest.txt`. The first focused fixture
check caught the old additive descriptor hash, retained in
`review/story1095-correction-focused.txt`.
The expanded EN cross-process test passes:
`review/story1095-correction-en.txt`. It produces 13342-byte AGS and the same
3215-byte SAV SHA `af8f9a5eab8481a9dd24ff4e352685e3f78a9aff13b025ae3955ee726cf5db24`,
checks 32 identical real-city AGS encodes, production LOAD/SAVE, both-format CLI
install refusal, unchanged source and forged-baseline refusal with no output.
This EN result is not the seat's pending paired EN/RU gate.

Final correction code commit `7f7175a72859ca7db1d0cf8441b60f29b3cda67b` passes
`go test -trimpath -count=1 ./...` once, recorded in
`review/story1095-correction-full-go.txt`. `gofmt -l` emits no paths and
`git diff --check` is clean. The no-assets guard prints `clean (tree scan)`;
the preservation guard reports 181 files across both roots as recorded.
Receipts: `review/story1095-correction-noassets.txt` and
`review/story1095-correction-preserved.txt`. Only this verification status text
was updated after that code gate. No new command was added by the correction:
command count remains 43. No second adversarial review
was commissioned. Full generated/native/world conversion and changed Human/item
authoring remain out of scope. The later saved-spellbook story must reconcile
its import policy with this shared baseline reconstruction; no unpublished
spellbook fields are assumed here.
