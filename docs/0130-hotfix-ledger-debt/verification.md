# 0130 — the hotfix ledger pays its debt: verification

Measured in `wt-0130` off master `524d059`, submodule pin `0908589`. This story
moves no code, so the evidence is document state, id sets and gate output.

## AC-1 — every hash survives, every row has an anchor

18 rows before, 18 after. Every commit hash in the `524d059` ledger is in the
finished one, and each row carries `[archive](ARCHIVE.md#<hash>)`. `872487b`'s
row gained a hash rather than losing one: `7666149`, the same hotfix as authored
in `0124`'s lane, moved out of the row's prose into its first column.

## AC-2 — the detail moved verbatim, 18 of 18

Checked independently of the executor that made the move, with a reader that
takes the pre-story cells from `git show 524d059:docs/hotfix/LEDGER.md`, splits
each row on `|`, collapses every run of whitespace in both texts to one space,
strips the archive's correction blockquotes, and asserts the `Boundary`, `What`
and `Owes` text of each row is a substring of its archive section.

```
rows: 18
verbatim ok: 18 / 18
bad: []
```

`LEDGER.md` went 45021 → 7163 bytes; `ARCHIVE.md` is 45015. Nothing was
summarised: the sum is larger than the original because the archive adds its
header note, the per-row headings and the four correction blockquotes.

## AC-3 — the FAIL set is unchanged, and it is empty

`bash scripts/check-sdd-audit.sh` on `524d059` from this worktree: `EXIT=0`,
**zero FAIL**, 36 notes. On the finished branch: `EXIT=0`, **zero FAIL**. The
note/warning counts are not compared and are not comparable — a worktree has no
`builds/`, so the per-story build warning does not fire here at all. Only the
FAIL set is enforced and only it is compared.

**Three FAILs appeared during the story and each was repaired rather than
silenced.**

1. `FAIL 8b91a75 (…/T1): trailered but touches only docs/`. Real: T1's whole
   deliverable is `docs/hotfix/ARCHIVE.md`. Repaired in the script's own
   provision for it, `trailer_may_be_docs_only()`, whose comment says *"a story
   whose deliverable IS a document belongs here"* — scoped to `T1` alone, since
   T2 and T3 both touch files outside `docs/`.
2. `FAIL 0130…: no task carries: DD4`. Real, and this story's own defect:
   `plan.md`'s fold table spelled `0038`'s `DD-4` inside backticks, which the
   audit's citation-stripper cannot read as a foreign citation, so it counted as
   an unwitnessed `DD` of `0130`. Reworded to name no id.
3. `FAIL 0130…: every task has landed and there is no verification.md` — this
   file.

The same trap as (2) was found and closed **before it fired** in `tasks.md`,
which spelled `0093`'s `FR-9` and `0119`'s `FR-22` in backticks; the dangling
check that would have flagged them only runs once `verification.md` exists.

## AC-4 — the budget

`bash scripts/check-doc-budget.sh`: `EXIT=0`. Two declared overruns were added to
`select_ceilings`, both measured before deciding and both one cause on one day:

| story | before | after | ceiling | what the bytes bought |
|---|---|---|---|---|
| `0119-chargen` | 13298 of 13312 — **14 bytes free** | 14041 | 14336 | `FR-22` (the derived preview) and `FR-15`'s carry clause |
| `0124-equip-from-the-pack` | 12792 of 13312 | 13641 | 13824 | the disclosed hole closed, the rearm's field set, the carried loadout |

No contract text was cut to make room, which is the rule that makes an overrun a
declaration rather than a habit.

## AC-5 — the script fails on a tree it should reject

`bash scripts/check-hotfix-ledger.sh` on the finished tree:

```
check-hotfix-ledger: row length ok (longest row 285 / 400 bytes)
check-hotfix-ledger: Owes column ok (every row states a disposition)
check-hotfix-ledger: file size ok (7163 / 12288 bytes)
check-hotfix-ledger: commit check ran (19 commit(s) since 9729459 examined)
check-hotfix-ledger: ok
```

Each of the four was **witnessed by mutation** — the ledger copied aside, the
working copy broken, the message captured, the copy restored:

| mutation | what it printed |
|---|---|
| a row padded to 430 bytes | `FAIL LEDGER.md row 51 is 430 bytes, over 400 - move the detail to docs/hotfix/ARCHIVE.md and leave one sentence` |
| a row's fourth cell emptied | `FAIL LEDGER.md row 51 has a blank Owes column - state FOLDED into <story> <FR-id> or NOT OWED with the reason, the header says this column is never left blank` |
| filler to 13638 bytes | `FAIL LEDGER.md is 13638 bytes, over 12288 - move more detail into docs/hotfix/ARCHIVE.md; the ceiling is what stops the file drifting back toward the ~41 KB it reached before 2026-08-10` |
| `ea870ef` removed from its row | `FAIL commit ea870ef "hotfix: the info window names the weapon he is holding" touches pkg/cmd/internal, carries no SDD-Task trailer and names no story, and has no ledger row - add one, or if it is not a hotfix say so in the exception list in this script` |

**And it caught two real gaps on its first run against the unmodified tree**,
which is the evidence that matters more than the four mutations. `7666149` was
in a row's prose and not its first column; `d36f866` — `Story 0107 —
verification: …` — slipped the subject filter the ledger had carried since
2026-08-07, because that pattern is anchored on the digits immediately after the
hash. The first was fixed in the ledger, the second in the filter, which now
also admits the `Story NNNN` form; `LEDGER.md`'s copy of the command moved with
it and the script's comment says the two must not drift.

Also run under `dash` as well as `bash`: byte-identical output, so it is POSIX
`sh` and not bash tolerated.

## AC-6, FR-14 — no code moved

```
$ git diff --stat 524d059..HEAD -- pkg cmd internal
(empty)
```

`go build ./...`, `go vet ./...` clean; `gofmt -l $(git ls-files '*.go')` prints
nothing; `go test -count=1 -trimpath ./...` green. No byte-form version moves and
no test moves, because nothing they read is different.

## P-1, P-2 — the id sets

Every artifact this story touched, id sets diffed against `524d059` with
`FR|AC|P|DD|SC` extracted from each file before and after:

```
docs/0020-sim-render-snapshot/spec.md  LOST: []  GAINED: ['DD2','DD3']
docs/0038-dim-map-border/spec.md       LOST: []  GAINED: ['DD4']
docs/0093-script-trace/spec.md         LOST: []  GAINED: ['FR9']
docs/0093-script-trace/plan.md         LOST: []  GAINED: ['FR9']
docs/0093-script-trace/tasks.md        LOST: []  GAINED: ['FR9']
docs/0111-sacks/spec.md                LOST: []  GAINED: ['DD2']
docs/0119-chargen/spec.md              LOST: []  GAINED: ['FR22']
docs/0119-chargen/plan.md              LOST: []  GAINED: ['FR22']
docs/0119-chargen/tasks.md             LOST: []  GAINED: ['FR22']
```

**Nothing was lost, anywhere.** The only ids minted are `0093`'s `FR-9` and
`0119`'s `FR-22`, each in all three of that story's artifacts as FR-6 requires —
which is what makes the audit green, and the demand was measured rather than
assumed (below). No `AC` and no `P` was minted; the acceptance criterion 0038
owed and the property 0124 owed were amended in place, keeping their ids. The
four spec `DD` gains are **mentions**, not mints: a folded clause naming the plan decision it voids. The
audit takes `DD` from `plan.md` only, so a `DD` named in a `spec.md` is selected
by nothing.

## The premise this story was handed, and what it actually was

The brief said `FR` ids carry no witness demand, having read
`check-sdd-audit.sh:447-478`. That is right about `verification.md` and **wrong
about the gate**, and the difference decides the whole shape of the work. Probed
on `0126-sound` before a single clause was written — one `FR-99` appended to a
landed spec:

```
FAIL 0126-sound: plan.md accounts for no: FR99
FAIL 0126-sound: no task carries: FR99
```

`report yes` is `bad`, and `enforced` is on for every story at or past
`WITNESS_FROM=0014`, so both are FAILs. Adding one line naming `FR-99` to that
story's `plan.md` and one to its `tasks.md` cleared them:
`check-sdd-audit: ok (0 note(s)/warning(s), none enforced)`. The probe was
reverted before work began. Hence FR-6, and hence a fold that amends an existing
`FR` wherever one can honestly carry the rule: 21 of 23 do.

## What was folded

23 fold-table rows, 24 clause sites, 18 landed specs. Two rows of the table were
written short by this story's own planning and were added afterwards — the
`dbd5c89` row owed `0119` and `0124` clauses as well as the `0066` and `0125`
ones the table named, and the ledger row now discharges all four.

`e09d2bb` is the one row that folds nowhere: its own `Owes` cell already said no
`spec.md` owes it, so it reads `NOT OWED` with that reason.

## Not done, and disclosed

- **The pin gap.** `ITEM-CORPSE-034` and `ITEM-SUIT-035` are on research
  `master` and not in this story's pin `0908589`. They appear only as dated
  corrections inside the frozen archive; no clause in any `spec.md` depends on
  either. See `provenance.md`.
- **`ITEM-CORPSE-034` says worn armour reaches the sack**, and this tree leaves
  the ten humanoid armour fields on the corpse. That is now a known divergence
  and it owes a story. This one does not fix it — `spec.md`, *Out of scope*.
- The open questions the archived rows named — the boots, the death gold, the
  pre-contact attack cycle, whether a press may target through fog, the town —
  are recorded in the archive and scheduled nowhere.
