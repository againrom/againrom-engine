# 0130 — the hotfix ledger pays its debt: tasks

FR-14 binds every task: touch nothing under `pkg`, `cmd` or `internal`.

## T1 — the archive, and the ledger's one-line rows

Add `docs/hotfix/ARCHIVE.md` and rewrite `docs/hotfix/LEDGER.md`'s table.

Follow **DD-1**: a throwaway `python` script (not `python3` — the Store stub
here) splits each table row of the current `LEDGER.md` on `|` and writes the
archive. Compare the archive's cell text back against the pre-change file before
rewriting the ledger; delete the script after.

**FR-2:** one `## <the row's first commit hash>` section per hotfix, in ledger
order, carrying that row's `Boundary`, `What` and `Owes` text **verbatim** —
re-wrapped to 100 columns and not otherwise altered. Open the file saying it is
frozen, nothing needs to read it, and a later hotfix does not append to it.

**FR-1, FR-3:** every row fits one rendered line under 400 characters and the
header's *"one line of prose"* becomes true. Keep four columns and put the
archive link in the row; the row's one sentence is its own **bold lead** where it
has one, else write one from the cell.

**FR-9:** beside each of the three false sentences add a dated correction naming
`ITEM-ARMSLOT-031`, `ITEM-CORPSE-034`, `ITEM-SUIT-035` in that order. Say in the
header note that the last two are published **after this story's pin `0908589`**
and nothing built depends on them.

Leave every `Owes` cell as it stands; T2 fills them in.

## T2 — the folds

Apply `plan.md`'s **fold table**, every row of it. Nothing else.

Per row: in that story's `spec.md`, find the named id and append inside that
requirement's own paragraph block **DD-2**'s marker line then the rule, wrapped
to the file's column width (FR-5, FR-7, FR-8). The table's rule text is the
clause — use it as written, adjusting only tense and grammar so it reads as part
of the requirement. **Carry no measurement, corpus count or witness procedure
across**; those stay in the archive.

Two rows say `NEW`. One is 0093's FR-9; the other is 0119's FR-22. Mint each at
the end of that spec's requirements, then satisfy **FR-6**: one line in that
story's `plan.md` and one in its `tasks.md` naming the id and saying it landed as
hotfix `<sha>`, not as a task of that story. `scripts/check-sdd-audit.sh` fails a
spec `FR` that neither file names.

Mint no `AC`, `P`, `DD` or `SC`; remove no id (P-1, P-2).

**FR-4:** rewrite each `LEDGER.md` row's fourth column to `FOLDED into <story>
<ids>`, naming every target written for that hotfix — or, for `e09d2bb`, `NOT
OWED` with the reason its own cell already gave.

Run `bash scripts/check-sdd-audit.sh` and `bash scripts/check-doc-budget.sh`
every few stories. A budget failure is fixed by a **declared overrun** in
`select_ceilings`, measured size and reason beside it, never by cutting contract
text.

## T3 — the script

Add `scripts/check-hotfix-ledger.sh`, POSIX `sh`, in its siblings' shape: a
`bad()` that counts, a final `ok`/`FAILED` line. FR-10: non-zero on any finding,
every message names the fix.

FR-11, four checks: per-row rendered length under a ceiling stated at the top; a
non-empty fourth column on every table row; the whole file under a stated byte
ceiling; and the commit check — `git log --no-merges --format='%h %s'
--invert-grep --grep='SDD-Task:' -- pkg cmd internal`, minus subjects naming a
four-digit story, every remaining commit newer than `9729459` having its hash in
column one of some row. Carry the eight pre-`9729459` exceptions (`712a106`,
`73b6063`, `0c23241`, `cbf7815`, `78a2699`, `9d0eace`, `cd1ed31`, `1f13af4`) as a
named list with their reason.

FR-12: if `9729459` is not in the clone, say the commit check is skipped and run
the file checks anyway. Say in the header comment why the commit check is the
point (**DD-3**).

FR-13: list the script in `AGENTS.md`'s *Build & test* block beside
`check-sdd-audit.sh`, and change its hotfix paragraph — which says the ledger
*"carries the one command that finds a hotfix committed without a row"* — to say
a script measures it.

Set the ceilings from the finished file plus real headroom; say where each number
came from.
