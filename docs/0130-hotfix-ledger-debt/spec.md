# 0130 — the hotfix ledger pays its debt: specification

**Eighteen hotfix rows carry eighteen undischarged debts. Every owed clause
reaches the story's `spec.md`, the detail behind it moves whole into a frozen
archive, the ledger becomes one line per row, and a script measures all of it
from here on.** This contract is self-contained. It moves no code.

## Functional requirements

### The ledger

**FR-1 — a row is one line, and the header's own rule becomes true.**
`docs/hotfix/LEDGER.md` keeps its header, its bounds and its check section, and
its table becomes exactly one row per hotfix with no row's rendered line longer
than the ceiling `scripts/check-hotfix-ledger.sh` enforces. The row states what
the hotfix did in one sentence, names its boundary in a phrase, names where its
detail went, and states the debt's disposition. No row is deleted and no commit
hash is dropped: the header already forbids that and this story does not make an
exception of itself.

**FR-2 — the detail moves whole, and is frozen.** A new `docs/hotfix/ARCHIVE.md`
carries, per hotfix, the **verbatim** text of that row's `Boundary`, `What` and
`Owes` cells as they stood before this story, under a heading anchored on the
hotfix's own commit hash. Nothing is summarised, paraphrased, shortened or
dropped in the move. The file opens with a note saying it is frozen and that
nothing needs to read it — `pipeline/archive/`'s own precedent — and it is not
appended to by a later hotfix, which writes its detail in its own commit message
and its one line in the ledger.

**FR-3 — the ledger points at the archive.** Every ledger row names the archive
anchor its detail moved to. A reader who wants the measurements, the declines and
the weighed alternatives reaches them in one hop and never has to reconstruct
them from a diff.

**FR-4 — every row's `Owes` cell is discharged, and says how.** Each row's fourth
column becomes either `FOLDED into <story> <FR-id>[, …]`, naming every clause
target this story wrote, or `NOT OWED` with the reason, for a hotfix whose own
`Owes` cell already said that no `spec.md` owes it. No row is left with an
undischarged debt and none is left blank; the ledger's own sentence — that the
column is *"never left blank"* — becomes true of the discharge as well as of the
debt.

### The folds

**FR-5 — a clause reaches the contract as prose on an `FR`, never as a new `AC`
or `P`.** A fold either appends to an existing requirement or mints a new `FR`.
It mints no `AC`, no `P`, no `DD` and no `SC` in any landed story, and it removes
no id of any kind. This is not a style preference: `scripts/check-sdd-audit.sh`
requires every `AC` and `P` in a `spec.md` and every `SC` in a `plan.md` to be
witnessed in that story's `verification.md`, and it refuses an id named
downstream that has gone upstream, so a minted `AC` or a deleted `FR` turns the
gate red in a story that is green today and that this story does not reopen.

**FR-6 — a new `FR` is accounted for where the audit demands it.** A minted `FR`
is named in that story's `plan.md` and in its `tasks.md`, because the same audit
requires both of an `FR` in a `spec.md`; the `tasks.md` mention says plainly that
the clause landed as a hotfix and not as a task of that story, so the note does
not read as work a task did.

**FR-7 — every fold is marked as one, and names its source.** A folded clause is
introduced in the spec by a marker naming the hotfix's commit hash and its
archive anchor, so a reader of a landed contract can tell a clause the story
wrote from a clause a hotfix taught it, and can reach the evidence.

**FR-8 — the spec gets the rule, never the measurement.** Corpus counts,
per-mission figures, witness procedures, named declines and weighed alternatives
stay in the archive. What reaches a `spec.md` is the normative sentence alone.

### What the ledger says that is no longer true

**FR-9 — three statements are corrected where they stand, and the correction is
visible.** Three claims in the archived text are false as of research `master`,
all downstream of `ITEM-DEATH-012`'s amendment: that the destination field of a
helm is undecoded; that worn armour does not reach the sack; and that
`sutableFor` is a name without a decode. The archive carries the original
sentence and a dated correction beside it, so the frozen text stays a faithful
record of what was believed while remaining readable without misleading. **The
correcting claims are disclosed as being published after this story's submodule
pin**, and no clause folded into any `spec.md` depends on them.

### The script

**FR-10 — a script measures the ledger.** `scripts/check-hotfix-ledger.sh`
exists, runs under `bash` from the repository root, exits non-zero on a finding
and zero otherwise, and prints for every finding a message naming the fix rather
than only the fault — the sibling scripts' own convention.

**FR-11 — it measures four things.** Per-row rendered length against a stated
ceiling; a non-empty fourth column on every row; the file's total size against a
stated ceiling; and — the check the ledger's header carries today as a command
for a human to run — that **every commit touching `pkg`, `cmd` or `internal`
newer than `9729459`, carrying no `SDD-Task:` trailer and naming no story in its
subject, has a row whose first column names it**. The eight pre-`9729459`
commits the header already names are the only exceptions and they are named in
the script, each beside the reason it is not a hotfix.

**FR-12 — it degrades honestly.** Run where the trailer base commit is not in
the clone, the commit check says so and is skipped rather than passing silently;
the file checks still run. It reads the working tree for the file checks and
`git log` for the commit check, and it takes no argument it needs.

**FR-13 — it is in the gate chain and in `AGENTS.md`.** The script is listed in
`AGENTS.md`'s *Build & test* block beside `check-doc-budget.sh` and
`check-sdd-audit.sh`, and `AGENTS.md`'s hotfix paragraph says a script now
measures the file where it previously said the ledger carries the command.

### What does not move

**FR-14 — no code changes.** Nothing under `pkg/`, `cmd/` or `internal/` is
added, removed or edited. No test moves, no byte-form version moves, no
simulation state is touched, and the milestone drive is unaffected because
nothing it runs is different.

## Acceptance criteria

**AC-1** Every commit hash in the ledger before this story is present in the
ledger after it, and every one has an archive anchor.

**AC-2** For every hotfix, the archive's `What` text is byte-identical to the
pre-story ledger cell, allowing only the line re-wrapping the move requires and
the dated corrections FR-9 names.

**AC-3** `bash scripts/check-sdd-audit.sh` reports the same **FAIL set** as it
did on `524d059` — empty — with every fold in place.

**AC-4** `bash scripts/check-doc-budget.sh` exits zero with every fold in place.

**AC-5** `bash scripts/check-hotfix-ledger.sh` exits zero on the finished tree,
and each of its four checks is shown to fail on a tree it should reject.

**AC-6** `git diff --stat 524d059..HEAD -- pkg cmd internal` is empty.

## Properties

**P-1** No landed `spec.md` loses an id of any kind, and no landed `spec.md`
gains an `AC` or a `P`.

**P-2** The set of `FR` ids added to landed specs is exactly the set named in
`plan.md`'s fold table, and every one of them is named in its story's `plan.md`
and `tasks.md`.

## Out of scope

Fixing anything a folded clause reveals the code does not do — such a finding is
reported, and the clause keeps describing what the hotfix did. Bumping the
`research/` submodule pin. Any owner-review artifact. Retiring the ledger itself:
it stays the record, and this story only makes it one that can be read.
