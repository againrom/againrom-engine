# 1205 — the engine's comments stop being a reverse-engineering log

## Result

Six forbidden comment forms are gone from every `.go` file in the tree, and a
new gate proves the change moved no code. `internal/storyguard`'s counters:

| form | before | after |
|---|---|---|
| `specclause` (`FR-n`/`DD-n`/`P-n`/`S-n`) | 10401 | 0 |
| `storymention` ("story NNNN") | 977 | 0 |
| `calendardate` | 517 | 0 |
| `rom1address` (raw code-region) | 70 | 0 |
| `funaddr` (`FUN_xxxxxxxx`) | 142 | 0 |
| `expmention` (`EXP-NNNN`) | 54 | 0 |

Total comment bytes fell from 9,473,680 to 7,638,703: **−1,834,977 bytes,
−19.4%**. The sweep itself cut 1,845,469 bytes; 4,168 of what came back is this
story's own new instrument, and the rest is ten package docs rewritten by hand
rather than lost, plus one added baseline note. Comment groups fell from 25,062
to 22,409.

## Intent

Owner direction: the code's comments had become a history of the reverse
engineering, which is useless in the engine and costs tokens on every read.
The provenance ruling is a bare id only — `DIV-1359` or `SAV-1011` may stand
alone as a citation, the narrative around it goes. Retired spec clauses,
calendar dates, ROM1 addresses and `FUN_` names leave outright; research and
Git own those.

## As built

959 `.go` files were rewritten by a one-off tool driven by the same six
regexes `internal/storyguard/scan.go` uses. Its rules, in order:

- A toolchain directive line (`//go:`, `+build`, `//line `) is never touched.
- A parenthetical that is nothing but a citation is deleted, and a citation
  member inside a list parenthetical is deleted from it. `Foo does X (0143
  FR-1).` keeps its whole meaning; `(viewer.go, story 1026)` becomes
  `(viewer.go)`. This is what saves a doc comment's naming first sentence.
- A sentence that still carries a forbidden form after that is deleted.
- A paragraph that loses its opening sentence is deleted whole. What remains
  otherwise is pronouns with no antecedent — "Only past all eight does…" once
  the sentence that counted eight is gone.
- A comment group whose FIRST paragraph loses its opening sentence is deleted
  whole, for the same reason one step up: a doc comment that has lost the
  sentence naming the symbol is an orphan clause about an unnamed thing.
- An indented or list-shaped paragraph is atomic: it is kept verbatim or
  deleted, never re-wrapped.
- Only paragraphs that actually lost text are re-wrapped, so the diff does not
  carry reflow noise from untouched prose.

Then `gofmt -w` over the tree. Deleting a comment between two struct-literal
entries merges their alignment blocks, which changed whitespace in 255 files
and no tokens anywhere.

### No citation was lost

103 claim and divergence ids would have left the tree entirely with the prose
around them. Each is re-emitted as a bare id line where its group stood —
`pkg/ui/hud.go`'s 111-line header is now `// MENU-COMBAT-017, TOWN-091,
DIV-213`. Distinct non-spec ids cited in `.go` comments: **1,226 before, 1,226
after, 0 lost.** The rescue was iterated to a fixed point.

### Ten package docs were rewritten, not lost

Ten files had their whole package or command doc deleted, because its opening
sentence named the story that wrote the tool. A command with no doc is a
command nobody can use, so each was rewritten by hand from the original,
keeping what the tool does and its usage line and dropping the provenance:
`cmd/appearcheck`, `cmd/audioprobe`, `cmd/buttonframecheck`,
`cmd/divreconstruct`, `cmd/effectmarkcheck`, `cmd/townsquarecheck`,
`cmd/wearcheck`, `internal/notices/doc.go`, `pkg/mapload/fromalm_test.go` and
`pkg/game/savroundtrip1195_corpus_test.go` — the last of which is the corpus
instrument the SAV endgame is scored against, and its doc is the only record
of that family's conventions: corpus DISCOVERED by walking a directory, one
`t.Run` per file, and a named logged REFUSAL rather than a `t.Fatal` that
drops the rest of the population. The search was mechanical (every file whose
doc group before `package` went from more than two lines to one or none), so
it is a complete list, not a sample.

### The instrument and the gate

`internal/tokenidentity` reduces a `.go` file to two independent hashes.

- **Tokens** — the `go/scanner` stream with every `COMMENT` token dropped,
  hashing each remaining token's kind and length-delimited literal. Identical
  for two trees whose code is the same and whose comments differ; different the
  moment one token moves, including a rename, a literal, or an operator.
- **Directives** — the ordered `//go:` and `+build` lines, which the token half
  is blind to by construction because to the scanner they are comments. A
  cleanup that deletes a build constraint or an embed passes the first half and
  fails this one.

`scripts/check-comment-only-change.sh <base-ref> [--allow-added]
[--allow-moved <paths>]` extracts the base with `git archive`, builds the
instrument from the working tree, hashes both tracked `.go` sets and fails on
any moved, removed or unexpected file. Passing it is what entitles a
comment-only change to skip the release, scenario and milestone chains.

## Proof

Token identity, `7bfbf7d` against this head. Run strictly, the gate names one
mover:

```
tokens     internal/storyguard/baseline.go	a5de00b571892ccb9e297789d996352b -> fe09f1b874f24839df01ecdd16ac55c3
TOKEN IDENTITY: FAIL - 2231 files in both trees, 1 moved, 0 allowed by name, 0 removed, 3 added (allowed); 2231 before, 2234 after
```

That is correct and it is the only one. `internal/storyguard/baseline.go`
holds the ratcheted counters as integer literals, and the same commit that
drives a counter to zero must rewrite them — the guard's own rule. A cleanup of
this kind will always move that one file, so the gate gained `--allow-moved`
rather than a reason to be skipped, and it prints the two hashes of anything it
allows. Naming it:

```
allowed    internal/storyguard/baseline.go (tokens)	a5de00b571892ccb9e297789d996352b -> fe09f1b874f24839df01ecdd16ac55c3
TOKEN IDENTITY: OK - 2231 files in both trees, 0 moved, 1 allowed by name, 0 removed, 3 added (allowed); 2231 before, 2234 after
```

**Zero of the 2,231 files carried over from `7bfbf7d` moved a token or a
directive**, apart from that one named file. The three added files are this
story's own instrument and its test.

The instrument's four validating cases were reproduced on this tree before it
was trusted, and each is now a test in `internal/tokenidentity`:

1. the same tree hashed twice gives identical output, 2231 rows;
2. 167,502 bytes of line comments stripped from 40 random `pkg/` files moved
   **zero** rows, and `go build ./...` still passed on the stripped tree;
3. one deleted `//go:build` line left the token hash identical and moved the
   directive hash;
4. one `>` changed to `>=` in `pkg/game/save.go` moved the token hash.

Gates:

- `gofmt -l .` — clean.
- `go test -trimpath -count=1 ./...` — exit 0, 54 packages ok.
- `scripts/check-no-game-assets.sh` — `clean (tree scan)`.
- `go run ./internal/storyguard/cmd/measure` — six forms at 0,
  `TestIdentCount: 6072`, `TestFileCount: 272`, `CommentBytes: 7638703`, with
  `internal/storyguard/baseline.go` set to match.
- `pipeline/check-preserved-installs.sh` — `ok — 554 file(s), every root as
  recorded`.

Script census, EN install, unchanged as expected of a comment-only change:
mission 10 and mission 20 both report **0** `UNSUPPORTED` nodes, and their
script node counts equal `pipeline/milestone-baseline.txt` exactly — m10
`16 checks, 27 instants, 12 triggers`, m20 `14 checks, 15 instants, 11
triggers`.

### Comments deliberately kept

Every comment that states a constraint the code does not express was checked by
name after the run and survives: `pkg/game/save.go`'s `Snapshot` doc on why
every field is exported for `encoding/gob`; the `rawOffered`-before-
normalisation trap on `cityExportNotice`; the `*worldSaveUnsupportedError`-only
fallback; every production `MapUnitID 0` corpse-guard comment; the
`SpellbookRestored` / `validateSnapshotBooks` / `CarryRoster` chain; and
`pkg/sim/step.go`'s per-kind field meanings, which remain the only record that
`X` carries six different things and that `Spell` sometimes carries a
carried-item index. Every `//go:build` and `//go:embed` line is gated by the
directive hash, not by inspection.

What died in those places was the restatement in a test's doc comment, which
named the story that wrote it rather than what the test pins.

## Open debt

- **Per-comment line ceiling — not this pass.** The distribution is now 22,409
  groups, median 3 lines, p75 6, p90 11, p95 16, p99 31, worst case 855 at
  `pkg/sim/binary.go:82`. **1,885 groups still exceed 12 lines**, down from
  2,582. The next pass sets the ceiling on those numbers.
- Spec-clause families the six regexes do not name remain: 1,896 `AC-n` and
  510 `SC-n` citations, and 2,831 bare four-digit story numbers (`0151 T12`
  shapes) in `.go` comments.
- Only `.go` files were swept. `scripts/*.sh`, Markdown and `docs/` still carry
  dates, story numbers and spec clauses; `internal/storyguard` reads Go alone.
- `internal/tokenidentity` compares tracked `.go` files. A change to a
  non-Go file — an embedded asset, a script, a `testdata` fixture — is outside
  what it can say anything about.
