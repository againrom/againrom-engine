# Verification — `.res` resource archive container (ROM1)

## Environment

- Toolchain: Go 1.26.1 (`windows/amd64`), the version pinned in `go.mod`.
- Dependency: `golang.org/x/text v0.40.0` (resolved offline from the module cache).
- The unit + fuzz suite runs with **no game install present** (all fixtures synthetic).
- AC-8 was run on 2026-07-23 against a lawful GOG "Rage of Mages" (ROM1) install at the default
  path; only measurements are recorded here — **no game bytes are committed**.
- The revision was run 2026-07-29 against **two** lawful installs, both read-only: AC-8's English GOG
  install and a Russian release (GOG, patch 1.03).
- AC-13's census swept **three** read-only roots on 2026-07-30: the two above plus the live GOG install.

## Commands

```
go build ./...
go vet ./...
go test ./...
go test -run='^$' -fuzz='^FuzzOpenBytes$' -fuzztime=20s ./pkg/formats/res
gofmt -l $(git ls-files '*.go')
scripts/check-no-game-assets.sh            # tree scan
scripts/check-no-game-assets.sh --history  # full-history scan
scripts/check-doc-budget.sh
scripts/check-sdd-audit.sh
```

All clean: `build`, `vet`, `test`, an empty `gofmt -l`, the asset guard on both the tree and
full-history scans, the doc budget, and the SDD audit.

## Acceptance criteria

| AC | Method | Evidence / outcome |
|---|---|---|
| AC-1 | Unit `TestOpenIndexesNestedTree` | A synthetic tree (root dir, nested subdir, files, and a top-level file root) indexes every file under its full normalized path with correct offset/size; `Entries()` is in node order. **PASS** |
| AC-2 | Unit `TestReadFileNormalizesPath` | Mixed-case and `\`-separated spellings all read the same bytes; normalized-equal paths resolve to one entry. **PASS** |
| AC-3 | Unit `TestReadFileMissingIsNotExist` | An absent path returns an error with `errors.Is(err, fs.ErrNotExist)`, no panic. **PASS** |
| AC-4 | Unit `TestOpenRejectsHeader` | A bad signature, and separately a `regOffset` past EOF, each return an error and a nil archive. **PASS** |
| FR-3 · SC-4a (geometry) | Unit `TestOpenRejectsGeometry` | A `regOffset` below `0x18` rejects with a nil archive. **PASS.** Two further rejections this row used to record — mod-32, and a count disagreeing with `(EOF−regOffset)/32` — were wrong and are gone (AC-11, AC-12) |
| AC-5 | Unit `TestOpenRejectsRanges` | A directory child range past `nodeCount`, and separately a file payload past `[0x18, regOffset)`, each reject. **PASS** |
| AC-6 | Unit `TestOpenRejectsTypeAndCycle` | An unknown node type, and separately a cyclic directory link, each reject with no panic. **PASS** |
| FR-3 · SC-6a (non-tree) | Unit `TestOpenRejectsNonTree` | A node referenced as a child by two directories (an acyclic shared subtree) rejects with no panic. **PASS** |
| AC-7 | Unit `TestNameDecodeCP866` | A name `{'A','B',0x80}` + NUL + `0xCD` padding decodes (CP866) to `"AB" + U+0410` — asserted via `EqualFold` because the normalized path is lower-cased; the expected value is written as a `\u` escape. **PASS** |
| AC-9 | Unit `TestOpenEmptyArchive` | An empty archive (`nodeCount == 0`, `regOffset == EOF`) opens to a non-nil, length-0 `Entries()` with no error. **PASS** |
| AC-10 | Unit `TestOpenKeepsHighByteFoldASCIIOnly` | Names differing only in a byte `≥ 0x80` (hex 0x80 / 0xA0) stay distinct entries; an ASCII name still resolves case-folded. **PASS** |
| AC-11 · SC-14 | Unit `TestOpenIgnoresRegistryResidue` | 23 stale bytes after the registry; a `nodeCount` one record short; 1…40 appended bytes. **PASS** |
| AC-12 · SC-13 | Unit `TestOpenRejectsShortRegistry` | A `nodeCount` claiming records the archive does not hold rejects with a nil archive — one over, five over, `0xFFFFFFFF`. **PASS** |
| AC-8 | Manual / developer-run (`restool`) | See "AC-8 evidence" below. **PASS** |
| AC-13 | Unit `TestOpenIndexesANamelessRecord`, + a three-root census | All three shapes — nameless root file, nameless file in a directory, nameless *directory* — **PASS**; no shipped container holds any. See "The nameless record" |

## Properties

- **P-1 / P-2 (no out-of-range read, no panic on malformed input).** `FuzzOpenBytes` ran 20 s / ~17.8 M
  executions across 20 workers, then 25 s / ~21.0 M again over the revised geometry step with a
  residue-bearing seed — **zero** crashes, panics, hangs or OOMs; every result is either an error with a
  nil archive or a valid archive whose `Entries()`/`ReadFile` are also panic-free. No failing input was
  produced, so no `testdata/fuzz` corpus exists.
- **P-3 (lookup is a pure function of the normalized path).** Covered by `TestReadFileNormalizesPath`
  and confirmed live in AC-8 (a mixed-case, backslash path resolved the same entry as its normalized
  form), and again on the Russian corpus below.
- **P-4 (the archive's length changes nothing).** `TestOpenIgnoresRegistryResidue` appends 1…40 bytes to
  an archive that opens and asserts the entries against **hand-written literals** each time, never
  against a re-parse of the same fixture; both directions of a header/length disagreement are pinned
  (AC-11, AC-12). Bounded, not universal: 1…40 covers the sub-record, whole-record and multi-record
  cases, but no test asserts it for an arbitrary suffix.

## AC-8 evidence (lawful ROM1 install; measurements only)

`restool list` opened every container. Because the reader validates all node ranges eagerly at open, a
successful open means **every entry's payload range lies within the data region** — the AC-8 assertion.

```
archive              entries   open
graphics.res            2732   OK
main.res                 452   OK
world.res                  3   OK
scenario.res              31   OK
sfx.res                  290   OK
speech.res               355   OK
movies.res               116   OK
patch.res                  1   OK
KIDS.LM                    0   OK  (empty archive, AC-9 on real data)
Allods/MUSIC.RES          21   OK
Allods/VIDEO4.RES         36   OK
Allods/VIDEO8.RES         30   OK
```

Zero rejections, and graphics.res alone indexes 2732 entries ("thousands", AC-8).
`restool cat graphics.res version.txt`
returned exactly the 12 bytes listed; `restool cat` with a mixed-case, backslashed
path (`UNITS\MONSTERS\TURTLE\PALETTE_.PAL`) returned the 17462-byte entry listed as
`units/monsters/turtle/palette_.pal`; a
missing path returned a not-exist error with a non-zero exit. `restool extract world.res` into a
scratch directory outside the repository reproduced all three entries with their subdirectories and
exact sizes, then the scratch directory was deleted. **No game bytes were copied into the repository**
(the asset guard's tree scan is clean).

## The acceptance revision — SC-13, SC-14, SC-15 (two lawful installs; measurements only)

The defect, reproduced with the pre-revision reader before anything changed, then the same command after:

```
restool list <ru>/MAIN.RES     before: res: registry length 16151 not a multiple of 32
restool list <ru>/MAIN.RES     after:  462 entries
```

Header geometry of every container in both installs — `regOffset` (`0x10`), `nodeCount` (`0x14`), the node
array's `nodeCount×32` bytes, and the residue the reader now ignores; read from the eight header bytes at
`0x10`, with no payload read:

```
count = registry records, entries = file entries, so count leads where an archive has directories
                     size      regOffset   count  nodeBytes  residue   entries
EN graphics.res   61394716    61294268    3139    100448        0       2732
EN main.res        5938096     5922288     494     15808        0        452
EN world.res         89655       89527       4       128        0          3
EN scenario.res    3236832     3235840      31       992        0         31
EN sfx.res        10917975    10907671     322     10304        0        290
EN speech.res    100243020   100231436     362     11584        0        355
EN movies.res      9117552     9113616     123      3936        0        116
EN patch.res          1474        1442       1        32        0          1
EN KIDS.LM              24          24       0         0        0          0
EN MUSIC.RES     110632356   110631684      21       672        0         21
EN VIDEO4.RES    124827552   124825920      51      1632        0         36
EN VIDEO8.RES    181692324   181690916      44      1408        0         30
RU GRAPHICS.RES   59584832    59484608    3132    100224        0       2724
RU MAIN.RES        4922540     4906389     504     16128       23        462
RU MOVIES.RES      9117552     9113616     123      3936        0        116
RU SCENARIO.RES    3234872     3233880      31       992        0         31
RU SFX.RES        10192111    10181807     322     10304        0        290
RU SPEECH.RES    119983188   119971796     356     11392        0        349
RU WORLD.RES         89655       89527       4       128        0          3
RU patch.res          1536        1504       1        32        0          1
RU MUSIC.RES     110632356   110631684      21       672        0         21
RU VIDEO4.RES    123556000   123554368      51      1632        0         36
RU VIDEO8.RES    192608084   192606676      44      1408        0         30
```

**SC-15.** All 23 containers open — 12 English (11 `.res`/`.RES` plus `KIDS.LM`) and 11 Russian, which
ships no `.LM`. Every English count is **identical** to AC-8's of 2026-07-23, container for container:
no regression.

**The residue.** One container carries any — `RU MAIN.RES`, `16151 = 504×32 + 23`. The
other 22 measure `regOffset + nodeCount×32 == EOF` exactly, which is why the derived-count reader passed
every English container. `RU MAIN.RES`'s 462 entries sum to **4906365** payload bytes against `regOffset`
4906389: payloads tile `[24, regOffset)` exactly, so no part of the residue lies inside an entry.

**Lookup, live (P-3).** `cat <ru>/MAIN.RES 'TEXT\CREDITS.TXT'` returned 2192 bytes, the size listed for
`text/credits.txt`; `'GRAPHICS\ChrGen\LeftUp.BMP'` the 114296 listed for `graphics/chrgen/leftup.bmp`; an
absent path a not-exist error and exit 1. `extract <ru>/WORLD.RES` into a scratch directory outside the
repository reproduced its 3 entries and sizes (156 / 88327 / 1020), then deleted. All 462 names are ASCII
(no byte outside `0x20`–`0x7E`), so this corpus discriminates no code page either.

## The nameless record — AC-13 (three lawful roots; measurements only)

Whether shipped data holds such a record at all.

```
sweep: every file in each root tested for the magic at offset 0, not only *.res/*.LM;
       headers and node arrays read, no payload read
empty: nodes whose name field is NUL at byte 0
dup:   file nodes whose normalized paths collide, keyed as pkg/formats/res keys them

                 nodes  empty  dup            nodes  empty  dup
EN graphics.res   3139      0    0   RU GRAPHICS.RES  3132     0    0
EN main.res        494      0    0   RU MAIN.RES       504     0    0
EN speech.res      362      0    0   RU SPEECH.RES     356     0    0
EN sfx.res         322      0    0   RU SFX.RES        322     0    0
EN movies.res      123      0    0   RU MOVIES.RES     123     0    0
EN VIDEO4.RES       51      0    0   RU VIDEO4.RES      51     0    0
EN VIDEO8.RES       44      0    0   RU VIDEO8.RES      44     0    0
EN scenario.res     31      0    0   RU SCENARIO.RES    31     0    0
EN MUSIC.RES        21      0    0   RU MUSIC.RES       21     0    0
EN world.res         4      0    0   RU WORLD.RES        4     0    0
EN patch.res         1      0    0   RU patch.res        1     0    0
EN KIDS.LM           0      -    -   (RU ships no .LM)
EN total          4592      0    0   RU total         4589     0    0

GOG install = third root, reproduces the EN column exactly (sizes, header words, 4592 nodes)
sweep total: 33 containers, 13773 nodes, 0 empty names, 0 duplicate paths
every name field holds a NUL (none runs into its 0xCD padding); shortest name 2 bytes

agreements, not findings:
  EN 4592 nodes = the count research reports for this release, reproduced independently
  0 duplicates  extends the published within-archive collision negative from the 8
                archives a normal launch registers to all 33 -- MUSIC, VIDEO4, VIDEO8
                and both patch.res included, which that registration walk never opens
```

**Falsifier.** A container outside these three roots — another release, a patched install, a
user-packed archive — could still carry one. The census claims what these roots hold and nothing
wider, which is why AC-13 is a synthetic unit test rather than a corpus assertion.

**Measured, and declined.**

```
restool extract <fixture> <scratch dir outside the repo>
  -> <dir>/x.bin written, then: restool: open <dir>: is a directory   exit 1
cause:  filepath.Join(root, "") == root, so the entry targets the output dir itself
escape: filepath.Rel gives ".", neither ".." nor ".."-prefixed -> guard passes, correctly
```

No escape, and no shipped archive reaches it. Changing the tool in the same breath as pinning the
reader would be two decisions in one, so it stands — recorded so the next reader need not re-derive
that it is safe.

**Mutation evidence**, both mutants of `res.go` reverted after measuring.

```
mutant                                    pkg/formats/res         pkg/vfs
skip nameless records in the index loop   FAIL (1 entry, want 2)  FAIL (fixture assertion)
index binds the LAST record, not first    FAIL (both collisions)  ok
```

Nothing else catches the second — the rest of the `res` suite is green under it too — so first-wins
was an unpinned property two stories already depend on. The first does redden `pkg/vfs`, but at its
fixture assertion, which reads as a broken VFS fixture rather than as a reader that stopped indexing
a record.

## Dependency hygiene

`go.mod` declares exactly one `require` (`golang.org/x/text v0.40.0`, no indirect entries);
`THIRD_PARTY_NOTICES.md` lists that one module as `BSD-3-Clause`; `LICENSES/BSD-3-Clause.txt` holds
the license text. `internal/notices` (go.mod↔notices sync) and `internal/archtest` (the DAG, including
the `pkg/formats/res → golang.org/x/text` edge) both pass.

## Limitations and residual risks

- **The three opaque header/node words (R-1)** — header `0x04`, header `0x0C`, node `0x00` — carry no
  interpreted meaning and are read and ignored; they are Unknown in the research too. Non-blocking: a
  successful decode needs none of them, as the 2732-entry graphics.res open demonstrates. Explaining
  them is a research-team item, not a reader gap.
- **Behavioural determinism** of the wider engine is out of this story's scope; this leaf reader is
  pure (no floats, no global state) but that property is exercised structurally, not asserted here.
- The AC-8 environment was one lawful install; the revision adds the release that exposed the mod-32
  assert. Two releases are still not "every packer" — what the reader now refuses to infer from a corpus
  is the point, not the corpus's size.
- **The reader is deliberately stricter than the original, whose behaviour outside its corpus is
  Unknown** (RES-ACCEPT-031): it accepts on the magic alone, reads a short registry silently, and never
  bounds-checks a directory range. Our rejections there are engineering under P-1/P-2 and `spec.md`
  declares them as ours — if one ever refuses a lawful archive, the hardening is what to revisit.

```
DOCUMENT BUDGET -- declared here rather than in the ceiling table, and why.
Measured 2026-07-30, after every measurement in this file was moved into a fence.

  verification.md prose   10595 / 9216    over by 1379 (14%)
  spec.md                 15420 / 14336   over by 1084 (7%)
  plan.md                 20471 / 20480   ok
  chain: plan > 1.2 x spec                broken, and predates this story's revisions

Spent on, in this file: why the census claims three roots and nothing wider; why
cmd/restool was measured and left alone; what the second mutant proves by reddening
nothing else in the repository; and AC-13's own row. Each is reasoning, which no
fenced block can carry. spec.md's overshoot is its Unknown clause and its consumer
warning. Nothing measured was cut to reach these numbers.

NO select_ceilings row is declared for 0001, and that is a decision, not an oversight:
check-doc-budget.sh sets FIRST_STORY=0012, so it never prints a 0001 line at all. A row
declared here could not bind, and an exception that never binds is one a later reader
has to disprove before trusting the ceilings beside it.

The figure worth more than a row would have been: this file stood at 9208 of 9216
BEFORE this story reopened it -- eight bytes of headroom -- so ANY new acceptance
criterion's reasoning breaches this ceiling, whatever its subject. The ceiling is
already exhausted. Do not read a quiet gate here as room.

If FIRST_STORY ever moves below 0012, the four numbers above are the ones to declare;
they are measured, and the work does not need redoing.
```

## Conclusion

Every applicable acceptance criterion and derived property has passing evidence: AC-1…AC-7, AC-9…AC-13 by
synthetic unit tests, P-2 by a 17.8M-execution fuzz run, AC-8 by a developer-run pass over a lawful
corpus. The revision adds a second release: 23 containers open across two installs — the Russian
`MAIN.RES`, refused outright before it, at 462 entries — with every English count unchanged. The reader
meets its contract; no game data entered the repository.
