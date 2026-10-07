# Verification — lossless ALM round-trip writer

Task commits, oldest first: `7e4a185` (T1), `0952542` (T2), `73a7571` (T3), `fa5836a` (T4).
Base `73121a5` — the plan and its tasks. One untrailered commit sits in the span: `79cc13a`,
the 0022 owner-ruling record, docs-only and belonging to that story — disclosed, not a
violation. Submodule pin frozen at research `3d95f2a` through the implementation.

Environment: Go 1.26.1 (pinned in `go.mod`), Windows 11. Two kinds of number appear below: the
synthetic suite, which reads no install, and a developer corpus run over a lawful install, marked
*(corpus)*. No corpus result substitutes for a criterion the suite carries.

**A contract revision has since landed** — see *The revision* at the end. Two trees are therefore
named here: the implementation tree `fa5836a`, where the corpus run and the first gate pass were
taken, and the revision tree, where the gates below were re-run. The corpus block is the earlier
observation and is labelled as such; it was not re-run.

## Gates

Re-run in this seat at the revision tree — the contract-revision commit plus this file.

```
$ go build ./... && go vet ./... && echo ok
ok
$ gofmt -l $(git ls-files '*.go')            (no output)
$ go test -trimpath -count=1 ./...           (0 FAIL; 681 tests pass, 2242 counting subtests,
                                              25 packages ok, 3 without tests)
      pkg/formats/alm    36 tests, 136 counting subtests
                         (26 of the subtests are FuzzDocumentRoundTrip seed entries)
      cmd/almtool         3 tests,   3
$ go test -trimpath -count=1 ./internal/archtest
ok      againrom/internal/archtest
$ sh scripts/check-no-game-assets.sh
check-no-game-assets: clean (tree scan)
$ sh scripts/check-doc-budget.sh docs/0023-alm-roundtrip-writer
  provenance.md 7100 / 7168   spec.md 13065 / 13312   plan.md 10862 / 13312
  tasks.md T1..T4 all under 1400; legend+traceability 480 / 1200
  verification.md (prose) 5469 / 9216 - fenced blocks free
  plan <= 1.2 x spec ok; tasks <= 1.2 x plan ok                     (exit 0)
$ sh scripts/check-sdd-audit.sh
check-sdd-audit: ok (36 note(s)/warning(s), none enforced)
$ git log --format='%(trailers:key=SDD-Task,valueonly)' 73121a5..fa5836a | sed '/^$/d' | sort | uniq -c
      1 0023-alm-roundtrip-writer/T1  ...  1 0023-alm-roundtrip-writer/T4  (4 ids, each exactly once)
$ git log --format='%B' 73121a5..HEAD | grep -ci co-authored-by
0
```

The audit's notes are other stories' debt, reported and not enforced; no `0023` row appears.

## Witnesses

Every id and the named thing that answers for it, at the revision tree. Everything but the
*(corpus)* row runs headlessly over synthetic streams built in test code; every accept/reject
verdict is written from the spec's own lists, never captured from `Open` (the DD-3 discipline).

```
FR-1  AC-1  SC-1                   pkg/formats/alm/document_test.go
      TestDocumentRoundTripMinimalMap - ten records in the shipped order, small grids, one
      type-4 record with its 8-byte extension; the roundTrip helper pins the accept half on
      Open itself per fixture (openOK), then Write == input byte-for-byte and Map() equals
      Open of the input in result and error.

FR-2  AC-2  SC-2                   TestDocumentRoundTripFreeHeaderFields
      dataSize 0xDEADBEEF, formatVersion 991, ten distinct per-map words, one of them the
      signalling NaN 0x7FBFFFFF, records in descending typeId order 9..0 - a permutation the
      shipped maps never use; every named byte re-read from a fresh Write at the spec's own
      offsets, never trusted to the fixture's construction.

FR-2  AC-3  SC-2                   TestDocumentRoundTripDroppedPayloadBytes
      non-zero bytes past the first NUL of the type-0 name, the description and a type-5 name;
      non-zero type-5 head and type-6 tail; a NaN-bit angle; arbitrary type-7/9 bodies; two
      subtests, an empty and a non-empty type-8.

FR-1  AC-4  SC-3  P-2              TestDocumentRejectsSpecRejectionList
      18 fixtures, one per clause of the spec's rejection list, each asserted BOTH ways:
      openReject AND docReject (a wrapped error and a nil document, never a partial result).

FR-1  FR-2  FR-4  AC-10  SC-3      TestDocumentOverAShorterRoster and
      TestDocumentPreservesBytesNoRecordCovers - the shapes the shipped writer never emits.
      Three records {0,1,2} and four {0,1,2,3}: accepted, RecordCount 3 and 4, RecordTypeIDs
      the ids carried, each RecordPayload the record's own, Write == input. On the
      three-record stream Map() returns a full W*H Overlay and Present(3) is FALSE - the
      manufactured plane, with Write emitting no type-3 record. The second test adds a
      typeId-10 record and a two-byte trailer: RecordCount 11, and Write drops neither.

FR-4  AC-1  SC-4                   TestDocumentNavigationAgreesWithStream - Version,
      RecordTypeIDs and each RecordPayload against streamRecords, an independent walk of the
      fixture bytes at the spec's offsets. TestDocumentRecordPayloadOutOfRangePanics - on the
      ten-record fixture, indices -1 and 10 both panic with a runtime error: misuse, not data.

FR-3  AC-5  SC-5  P-3              TestDocumentOwnershipInputAndWriteBuffers - the input buffer
      zeroed post-open, the first write buffer mutated; both writes equal the pre-mutation
      clone.

FR-4  AC-6  SC-5                   TestDocumentNavigationResultsAreFreshCopies - every
      navigation result overwritten; the document writes and navigates as before, expectations
      derived from the clone.

FR-5  AC-7  SC-6  P-1  P-2  P-3    pkg/formats/alm/fuzz_test.go
      FuzzDocumentRoundTrip - 26 seeds (the minimal map, the permuted order, the off-shape
      streams, one per rejection class) green under plain go test; on every input Open is the
      oracle: accept/reject agreement, atomicity on reject, and on accept Write == input after
      zeroing a working clone, plus a second Write after mutating the first buffer.

FR-6  AC-8  SC-7                   the full suite above; internal/archtest green unedited -
      the import check holds pkg/formats/alm to stdlib + x/text (document.go adds only
      encoding/binary and fmt, both already imported by the package).
$ git diff --name-only 73121a5..fa5836a          (the implementation span)
cmd/almtool/main.go
cmd/almtool/roundtrip_test.go
docs/0022-units-static-sprites/verification.md    (the untrailered ruling record, docs-only)
pkg/formats/alm/doc.go
pkg/formats/alm/document.go
pkg/formats/alm/document_test.go
pkg/formats/alm/fuzz_test.go
      no source change outside pkg/formats/alm and cmd/almtool.

FR-7  SC-8 (unit half)             cmd/almtool/roundtrip_test.go
      TestRoundtripVerbAccept - the exact identity line, size + MD5, over a synthetic temp-dir
      stream; TestRoundtripVerbReject - the error path with nothing on stdout; TestFirstDiff -
      equal, unequal and both prefix cases, the shorter length on length divergence.

FR-2  FR-7  AC-9  SC-8 (corpus)    the 38-map run below *(corpus)*
```

## The corpus run over a lawful install *(corpus)*

AC-9's evidence, run at `fa5836a` with `almtool` and `restool` built fresh from that tree, and
**not re-run for the revision** — the revision changed no code, and `Write` still clones. The 10
loose `.alm` at the install root were read in place — the install is read-only throughout; the 28
campaign maps live inside `scenario.res`, whose entries the shipped `restool extract` wrote to a
session temp directory **outside the repo** — extracted bytes are game assets and enter neither
the repo nor `builds/`. The landing gate's asset scan runs after all of this. Each row transcribes
one verb line, `roundtrip: identical, <bytes> bytes, md5 <md5>`, verbatim in both numbers.

```
$ go build -o <tmp>/almtool.exe ./cmd/almtool && go build -o <tmp>/restool.exe ./cmd/restool
$ <tmp>/restool.exe extract "<install>/scenario.res" <tmp>/scenario
restool: extracted 31 entries to <tmp>/scenario       (28 .alm + 3 .reg)
$ <tmp>/almtool.exe roundtrip <map>                   one line per map
map                      bytes   md5
scenario.res/10.alm      67554   2d983ccbf249c5336ebb7ccc41fc405c
scenario.res/20.alm     114958   c045f7e3ab59a4ecf8e34614c35b6607
scenario.res/30.alm      51706   84d409752a925df334aabfd136343c65
scenario.res/31.alm      51696   86c4601da74fc97d4e68ef56dee97ca8
scenario.res/40.alm     127210   1a1973bf72f2b177bbb21729d13d85db
scenario.res/41.alm      50562   d5ea3a60475b3776b55e8c867a33868f
scenario.res/50.alm     129728   e3e5c5dc1fc92fd54de21759adbcf2d0
scenario.res/51.alm      44720   3cfa74dcd87451f398e23ac9a7de5ad5
scenario.res/60.alm     192054   95735f55a58e4729382ed7ecf86150e6
scenario.res/61.alm      43526   32c9a619b0abbeb6069c78f85acc57ce
scenario.res/70.alm     131772   81aa4fbcc6bb43eac7646d644843909e
scenario.res/71.alm      57536   c6aa483f67c98dae80d27e9327dc7cf5
scenario.res/80.alm     126754   127d127c981cdda5d4009032aa85fd07
scenario.res/81.alm      82418   97f8d5357eb7514116848619c98081aa
scenario.res/90.alm     166994   8fdec5ac2272095b186df0cc251276dc
scenario.res/91.alm      65054   cf3c46d09a9cfded11923fa63585a17b
scenario.res/100.alm    159042   306579a3137acb74c03f92f75cef44a0
scenario.res/101.alm     97082   95df2195f8c52ad930c5c27e8eea1a6a
scenario.res/110.alm     77132   e10d448ed428af6293b0948bba1d955a
scenario.res/111.alm    118638   7ad2543634164c005e1642168a22be47
scenario.res/120.alm    125180   dc7e0d8a6e454ece704b6395c021c8cb
scenario.res/121.alm     46154   75ae0bf74d95d381cdb7513d91e39fac
scenario.res/130.alm    128120   422192210d1e9b2807e196b00d98270a
scenario.res/131.alm    199402   35f70c6578a6d599c2296cf2040ef064
scenario.res/140.alm    320350   4649d55a1595630ee995238d63615681
scenario.res/141.alm     73102   e58c0a46eca999a9814bc623521a1b1f
scenario.res/150.alm    163704   45de73d2169a95ea6df785a783e29e36
scenario.res/151.alm    194470   c80fbdd2dc2266eb5c78ca5a243f17d2
Beast.ALM               342544   63b43cb6142a953f53b2919d4eb59260
Cross.ALM               328298   b393fa3a6a2ac6aeabc11ddf089e58a2
Forester.alm            317784   2003d087eb178a6d6ef7e93a2e19667f
Horror.alm              409386   3a7c826cb435b3f161f9d728bb15bba3
Islands.alm             301096   50b190af1864afddc963eaea6164627b
Kids.alm                 31458   350a9d7865716b5b24fddf7bd32717bd
Kids2.ALM                34108   2906a00f20e1b8a55e16a7ab158ba0a5
LuMoir.alm              103502   1f34deebf54d32913bb153048d2c8a76
Tomb.ALM                302200   6735a98bbc2106f1dc37bfd48b4aa66c
Waters.alm              107722   dd590d0eb6a3bb549f0582f0b15a0cbe
38 maps, 38 identical, 0 mismatches, 0 rejections, every exit code 0
```

## Disclosed notes

- **`cmdRoundtrip`'s mismatch branch is unreachable in-process with an honest `Write`** — which
  is exactly why `firstDiff` is pinned by its own unit test (DD-7's stated reason): the corpus
  "identical" above must be a measurement, not an assumption, so the arithmetic it trusts is
  itself witnessed.
- **The fuzz target's live smoke runs are extra-gate observations; AC-7's floor is the seed
  corpus under plain `go test`.** The implementation executor reported 20 s / 14.2 M execs
  PASS and the orchestrator seat 15 s / 10.1 M execs PASS — their measurements, recorded as
  reported. Re-run at `fa5836a`: 15 s fuzztime, 11369272 execs, PASS, no failing input, nothing
  written under `testdata/`, tree clean after. Not re-run for the revision.
- **AC-9's corpus is one install's 38 files, and all of them carry ten records.** The shapes
  AC-10 covers therefore have a synthetic witness and no corpus one, which is stated in the
  contract rather than left to be noticed.

## The revision

A contract correction, no code — the commit immediately before this one. The reader this story delegates acceptance to
widened (`0003-alm-container/T9`, `892bbd8`), and 0023's contract went on asserting the container
it had been written against: exactly ten records, `{0..9}` once each, exact tiling with no
trailer, and four rejection clauses of which three had stopped being rejections. The corrected
text is in `spec.md`; `plan.md` is cascaded and `provenance.md` repaired.

**No `pkg/` file changed, so no trailered commit was owed.** The behaviour the corrected contract
describes is Phase A's and landed under 0003's own task — the widened frame walk, the
manufactured type-3 plane, `Map.Present`, `Document.RecordCount` and the bounds-checked span walk.
Its tests came with it, in this package's files: `TestDocumentOverAShorterRoster` and
`TestDocumentPreservesBytesNoRecordCovers` were written there as a regression test for the panic
the fixed ten-iteration span walk would have taken on the first accepted short-roster stream.
AC-10 names them; it does not commission them, and no test was added here.

What the revision therefore verifies is the correspondence, not the code: every clause the
contract now states was read off the tree before it was written, the rejection list
against `TestDocumentRejectsSpecRejectionList`'s 18 fixtures one by one, and `RecordPayload`'s
bound against the span slice rather than the constant ten. AC-1's and AC-2's ten-record fixtures
are still ten-record fixtures and were left alone: they are instances the contract permits, not
the roster it once required.

The gate above is the whole of this stage's evidence. Sizes after the revision, measured: spec
13065 of 13312 (247 free), plan 10862 of 13312, provenance 7100 of 7168 (68 free). The additions
were paid for out of duplication — the I/O example, which asserted nothing AC-2 and AC-10 do not;
FR-2's restatement of three ACs' GIVEN columns; the problem statement's re-enumeration of the
lossy fields; and the payload grammar, restated inside a document whose own constraint row says
acceptance is delegated and not restated. No requirement, criterion or disclosure was cut to fit.
